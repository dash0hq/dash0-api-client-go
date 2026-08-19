package yaml

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	sigsyaml "sigs.k8s.io/yaml"
)

// conditionallyIgnoredFields are fields ignored during comparison only when
// absent from the reference document (typically the user's local
// definition). These are fields the API enriches on retrieval but that
// users may optionally manage. When present in the user's document, drift
// detection is preserved. Paths are relative to the document root
// regardless of AnnotationsRoot -- both entries are metadata-nested, so a
// flat-document caller (see WithFlatDocument) has nothing matching either
// path to ignore.
var conditionallyIgnoredFields = []string{
	"metadata.name",    // server-generated when absent, but user-declared intent when present
	"spec.permissions", // API-managed: stored separately, enriched on retrieval
}

// ConditionallyIgnoredFields returns a copy of the fields ignored during
// comparison only when absent from the reference document. See
// conditionallyIgnoredFields for the full description. Returns a fresh
// slice on every call so a caller mutating the result cannot shift drift
// semantics for every other caller in the process.
func ConditionallyIgnoredFields() []string {
	return append([]string(nil), conditionallyIgnoredFields...)
}

// defaultIgnoredFields are always removed when comparing documents, relative
// to the configured annotations root (see Option/WithAnnotationsRoot).
var defaultIgnoredFields = []string{
	"labels",
	"createdAt",
	"updatedAt",
	"version",
	"dash0Extensions",
}

// rootLevelIgnoredFields are always removed regardless of AnnotationsRoot.
var rootLevelIgnoredFields = []string{
	"apiVersion",
	"kind",
}

// Options configure Equivalent and Normalize.
type Options struct {
	// AnnotationsRoot is the dot-separated path under which "annotations"
	// and "labels" (and the rest of defaultIgnoredFields) live. Defaults to
	// "metadata", matching every Kubernetes-CRD-shaped Dash0 asset
	// (Dashboard, View, SyntheticCheck, PrometheusRule, Dash0SpamFilter,
	// Dash0NotificationChannel, Dash0Team). Set to "" via WithAnnotationsRoot
	// for a flat document that carries "annotations"/"labels" at the root
	// instead of nested under "metadata" -- dash0-cli's native (non-CRD)
	// CheckRule kind is the one Dash0 asset shaped this way.
	AnnotationsRoot string
	// AnnotationsUnfiltered disables preservedAnnotationKeys filtering: every
	// key in the annotations map at AnnotationsRoot participates in
	// comparison (still subject to the unconditional stringify/default-value
	// cleanup cleanupMap always does), instead of being stripped unless
	// explicitly preserved. See WithAnnotationsUnfiltered.
	AnnotationsUnfiltered bool
}

// Option configures Options.
type Option func(*Options)

// WithAnnotationsRoot overrides the default "metadata" root under which
// "annotations" and "labels" are expected to live. Pass "" for a flat
// document where they live at the document root.
func WithAnnotationsRoot(root string) Option {
	return func(o *Options) {
		o.AnnotationsRoot = root
	}
}

// WithAnnotationsUnfiltered disables preservedAnnotationKeys filtering
// entirely, so every key already present in the annotations map at
// AnnotationsRoot takes part in comparison (the unconditional
// stringify/default-value cleanup still applies). Without this option, an
// empty preservedAnnotationKeys means "strip every annotation" -- correct
// for a metadata.annotations convention that is provenance-only unless a
// key is explicitly opted back in (Dashboard, View, ...). Use this option
// for a kind whose annotations map holds genuine user content by
// convention instead -- dash0-cli's CheckRule kind is the one Dash0 asset
// shaped this way: its flat top-level "annotations" carries
// summary/description/sharing directly (the same PrometheusAlertRule type
// backs both a native CheckRule document and a PrometheusRule CRD's
// per-alert annotations, which TerraformProvider-dash0 already compares in
// full, only auto-removing the three known default values).
func WithAnnotationsUnfiltered() Option {
	return func(o *Options) {
		o.AnnotationsUnfiltered = true
	}
}

// WithFlatDocument configures a flat document whose annotations carry
// genuine user content by convention, the dash0-cli CheckRule shape
// WithAnnotationsUnfiltered's doc comment describes. Equivalent to
// WithAnnotationsRoot("") combined with WithAnnotationsUnfiltered(); prefer
// this over combining the two by hand for that shape. WithAnnotationsRoot("")
// alone still means what it always has -- a flat document whose
// non-preserved annotations should still be filtered out, a legitimate and
// separately supported combination -- so this option does not change or
// replace it, only names the specific pairing CheckRule-shaped callers need.
func WithFlatDocument() Option {
	return func(o *Options) {
		o.AnnotationsRoot = ""
		o.AnnotationsUnfiltered = true
	}
}

func resolveOptions(opts []Option) Options {
	o := Options{AnnotationsRoot: "metadata"}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// prefixedIgnoredFields returns defaultIgnoredFields joined onto root (dot-
// separated), plus rootLevelIgnoredFields, plus any additionalIgnoredFields
// passed by the caller.
//
// "labels" is skipped entirely at the flat root (root == ""): on every
// metadata-nested Dash0 asset, metadata.labels carries only provenance/id
// keys (dash0.com/id, dash0.com/origin, ...) with no user-customizable
// content, so wholesale-stripping it from comparison is safe. dash0-cli's
// flat CheckRule kind is different -- its top-level labels map can carry
// genuine user-set custom labels (e.g. "team: platform") alongside
// dash0.com/origin, and the caller's own server-field stripping (done
// before this engine ever sees the document) has already removed the
// provenance-only keys -- so at the flat root there is nothing left in
// "labels" that should be blanket-ignored.
func prefixedIgnoredFields(root string, additionalIgnoredFields []string) []string {
	all := make([]string, 0, len(defaultIgnoredFields)+len(rootLevelIgnoredFields)+len(additionalIgnoredFields))
	all = append(all, rootLevelIgnoredFields...)
	for _, f := range defaultIgnoredFields {
		if root == "" {
			if f == "labels" {
				continue
			}
			all = append(all, f)
		} else {
			all = append(all, root+"."+f)
		}
	}
	all = append(all, additionalIgnoredFields...)
	return all
}

// Normalize normalizes a YAML/JSON document by removing fields that don't
// participate in drift detection: server-managed metadata, empty
// containers, default values, and non-preserved annotations.
// additionalIgnoredFields are extra dot-separated field paths to strip
// (e.g. from ConditionallyIgnoredFields, or a kind-specific API-managed
// field). preservedAnnotationKeys lists annotation keys that should survive
// normalization (e.g. "dash0.com/sharing"); every other annotation is
// stripped. If empty, all annotations are stripped. See WithAnnotationsRoot
// for documents that don't nest annotations/labels under "metadata".
// WithAnnotationsUnfiltered (and WithFlatDocument, which implies it) disables
// this filtering entirely, making preservedAnnotationKeys inert -- every
// annotation key participates in normalization instead.
func Normalize(data []byte, additionalIgnoredFields []string, preservedAnnotationKeys []string, opts ...Option) ([]byte, error) {
	parsed, err := normalizeToMap(data, additionalIgnoredFields, preservedAnnotationKeys, opts...)
	if err != nil {
		return nil, err
	}

	encoded, err := sigsyaml.Marshal(parsed)
	if err != nil {
		return nil, fmt.Errorf("error encoding document: %w", err)
	}

	return []byte(strings.TrimSuffix(string(encoded), "\n")), nil
}

// normalizeToMap does Normalize's parsing and field-stripping work but
// returns the result as a map instead of marshaling it back to bytes.
// Equivalent uses this directly instead of calling Normalize and
// immediately re-parsing its marshaled output -- profiling on an 80-panel
// document showed that round-trip was over half of Equivalent's cost.
func normalizeToMap(data []byte, additionalIgnoredFields []string, preservedAnnotationKeys []string, opts ...Option) (map[string]any, error) {
	o := resolveOptions(opts)

	var parsed map[string]any
	if err := sigsyaml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("error parsing document: %w", err)
	}
	if parsed == nil {
		// Empty input (or a document that is only "null") unmarshals to a
		// nil map, which would otherwise marshal back out as the literal
		// "null" instead of "{}" -- the same document a fully-stripped
		// non-empty input normalizes to. Two documents that both end up
		// with nothing left must produce the same normalized form.
		parsed = map[string]any{}
	}

	cleanupMap(parsed, prefixedIgnoredFields(o.AnnotationsRoot, additionalIgnoredFields))
	if !o.AnnotationsUnfiltered {
		stripAnnotations(parsed, o.AnnotationsRoot, preservedAnnotationKeys)
	}

	return parsed, nil
}

// mapAtRoot returns the map at the given dot-separated root path within
// data, or data itself when root is "". Walks one segment of nesting per
// dot, matching how prefixedIgnoredFields and cleanupMap treat the same
// AnnotationsRoot option -- a dotted root is one path, not one literal key.
func mapAtRoot(data map[string]any, root string) (map[string]any, bool) {
	if root == "" {
		return data, true
	}
	current := data
	for segment := range strings.SplitSeq(root, ".") {
		next, ok := current[segment].(map[string]any)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

// stripAnnotations removes annotation keys not in preservedKeys from the
// annotations map found at root (or the document root when root == "").
// If preservedKeys is empty, all annotations are removed. The annotations
// map is deleted from its parent when it becomes empty.
func stripAnnotations(data map[string]any, root string, preservedKeys []string) {
	parent, ok := mapAtRoot(data, root)
	if !ok {
		return
	}
	annotations, ok := parent["annotations"].(map[string]any)
	if !ok {
		return
	}

	if len(preservedKeys) == 0 {
		delete(parent, "annotations")
	} else {
		preserved := make(map[string]bool, len(preservedKeys))
		for _, k := range preservedKeys {
			preserved[k] = true
		}
		for key := range annotations {
			if !preserved[key] {
				delete(annotations, key)
			}
		}
		if len(annotations) == 0 {
			delete(parent, "annotations")
		}
	}

	if root != "" && isEmpty(parent) {
		deleteAtRoot(data, root)
	}
}

// deleteAtRoot removes the final segment of the dot-separated root path from
// its immediate parent map within data. Mirrors mapAtRoot's own path
// walking, so a dotted root is deleted from the map that actually holds it
// rather than as a literal top-level key.
// Cascades upward: if removing the final segment leaves its parent empty,
// that parent is itself removed from its own parent, and so on, so a
// multi-level root doesn't leave a chain of empty shell maps behind (which
// would then look like added content next to a reference document that
// never had this root at all).
func deleteAtRoot(data map[string]any, root string) {
	idx := strings.LastIndex(root, ".")
	if idx == -1 {
		delete(data, root)
		return
	}
	parentPath := root[:idx]
	parent, ok := mapAtRoot(data, parentPath)
	if !ok {
		return
	}
	delete(parent, root[idx+1:])
	if isEmpty(parent) {
		deleteAtRoot(data, parentPath)
	}
}

// stringifyMapValues converts all non-string values in a map to their string
// representation. Used for annotation/label maps, which are semantically
// map[string]string, but untyped YAML parsing may produce non-string types
// (e.g., an unquoted 5000 becomes int, true becomes bool).
func stringifyMapValues(m map[string]any) {
	for key, value := range m {
		if _, ok := value.(string); !ok {
			m[key] = fmt.Sprintf("%v", value)
		}
	}
}

// removeDefaultAnnotationValues removes annotations whose values match the
// defaults used by the check rule round-trip conversion, so explicitly
// setting a default value is treated as semantically equivalent to omitting
// the annotation.
//   - dash0-threshold-critical: "0" and dash0-threshold-degraded: "0" are
//     removed because zero-value thresholds are omitted during the Dash0
//     JSON -> Prometheus YAML conversion.
//   - dash0-enabled: "true" is removed because true is the default and is
//     omitted during the same conversion.
func removeDefaultAnnotationValues(annotations map[string]any) {
	for key, value := range annotations {
		strVal, ok := value.(string)
		if !ok {
			continue
		}
		if (key == thresholdCriticalAnnotationKey || key == thresholdDegradedAnnotationKey) && strVal == "0" {
			delete(annotations, key)
		}
		if key == enabledAnnotationKey && strVal == "true" {
			delete(annotations, key)
		}
	}
}

// zeroOmittedDurationFields are map keys backed by a Duration field with
// omitempty, so the server drops them from a round-trip when the value is
// zero. A zero value in these fields is therefore treated as absent, not as
// explicit drift. All three sit on the same PrometheusAlertRule struct
// (generated.go): For, Interval, and KeepFiringFor are each *Duration
// `json:"...,omitempty"`.
var zeroOmittedDurationFields = map[string]bool{
	"for":             true,
	"interval":        true,
	"keep_firing_for": true,
}

// cleanupMap removes specified fields by path and empty values from a map in
// place. fieldsToRemove contains dot-separated paths (e.g., "metadata.createdAt").
// Empty arrays, maps, and strings are also removed to ensure consistent
// comparison.
func cleanupMap(data map[string]any, fieldsToRemove []string) {
	removeHere := make(map[string]bool)
	nestedRemovals := make(map[string][]string)
	for _, path := range fieldsToRemove {
		if idx := strings.Index(path, "."); idx == -1 {
			removeHere[path] = true
		} else {
			key := path[:idx]
			nestedRemovals[key] = append(nestedRemovals[key], path[idx+1:])
		}
	}

	for key, value := range data {
		if removeHere[key] {
			delete(data, key)
			continue
		}

		// JSON null becomes Go nil after yaml.Unmarshal; treat as absent.
		if value == nil {
			delete(data, key)
			continue
		}

		switch v := value.(type) {
		case map[string]any:
			cleanupMap(v, nestedRemovals[key])
			if key == "annotations" || key == "labels" {
				// Annotations and labels are semantically map[string]string, but
				// untyped YAML parsing may produce non-string types.
				stringifyMapValues(v)
			}
			if key == "annotations" {
				// Must run after stringifyMapValues, which it expects string values from.
				removeDefaultAnnotationValues(v)
			}
			if isEmpty(v) {
				delete(data, key)
			}
		case []any:
			cleanupNestedSlice(v, nestedRemovals[key])
			if len(v) == 0 {
				delete(data, key)
			}
		case string:
			if v == "" {
				delete(data, key)
			} else if zeroOmittedDurationFields[key] {
				// for and keep_firing_for use Duration with omitempty, so
				// marshalling drops them when the value is zero. Remove them
				// here so "for: 0s" in a user document matches the
				// round-tripped form that omits the field. If parsing fails,
				// the value is not a duration, so keep it as-is.
				if d, err := parseDuration(v); err == nil && d == 0 {
					delete(data, key)
				}
			}
		case float64:
			// Hand-written rule YAML often leaves a zero duration unquoted
			// (for: 0 rather than for: "0s"), which parses as a number, not
			// a string -- the same omitempty round-trip the string case
			// handles still applies.
			if zeroOmittedDurationFields[key] && v == 0 {
				delete(data, key)
			}
		}
	}
}

// cleanupNestedSlice applies cleanupMap's emptiness cleanup to every
// map-shaped element of items, and recurses into slice-shaped elements
// (e.g. a matrix: [[{a: ""}]]) so nesting depth doesn't hide an empty value
// from cleanup. Scalar elements are left as-is. fieldsToRemove is the
// portion of the caller's dotted removal paths rooted one level past this
// array field (e.g. cleanupMap forwards nestedRemovals[key] here), so an
// additionalIgnoredFields path whose intermediate segment names an array
// field reaches the array's own elements instead of being silently dropped.
func cleanupNestedSlice(items []any, fieldsToRemove []string) {
	for _, item := range items {
		switch v := item.(type) {
		case map[string]any:
			cleanupMap(v, fieldsToRemove)
		case []any:
			cleanupNestedSlice(v, fieldsToRemove)
		}
	}
}

// isEmpty checks if a map is structurally vacuous: empty, or containing
// only nil/empty-container/empty-string values. Used by cleanupMap to decide
// whether a container carries no information worth keeping. Deliberately
// narrower than isZeroValue's notion of "zero" -- a bool false or number 0
// is a real, meaningful value a user may have explicitly set (e.g.
// spec: {enabled: false}), not vacuous content to delete during
// normalization, even though isZeroValue does treat it as a zero value for
// the different question stripAbsentZeroValues asks (see isAllZeroValues).
func isEmpty(m map[string]any) bool {
	if len(m) == 0 {
		return true
	}
	for _, value := range m {
		if value == nil {
			continue
		}
		switch v := value.(type) {
		case map[string]any:
			if !isEmpty(v) {
				return false
			}
		case []any:
			if len(v) > 0 {
				return false
			}
		case string:
			if v != "" {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// canonicalString produces a deterministic string representation of a
// value, recursively sorting maps by key and slices by their canonical
// representation. This ensures the SortSlices comparator produces stable
// sort keys even when nested structures (like action lists within
// permissions) differ in order.
func canonicalString(v any) string {
	switch val := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, k+":"+canonicalString(val[k]))
		}
		return "{" + strings.Join(parts, ",") + "}"
	case []any:
		strs := make([]string, len(val))
		for i, item := range val {
			strs[i] = canonicalString(item)
		}
		sort.Strings(strs)
		return "[" + strings.Join(strs, ",") + "]"
	case string:
		// %q quotes and escapes the value, so a colon or comma inside it
		// can't be mistaken for a structural separator (e.g. {"a": "b:c"}
		// and {"a:b": "c"} would otherwise both render as "{a:b:c}", making
		// a real difference between them look like a reorder). It also
		// distinguishes a string leaf from a same-looking non-string one
		// (e.g. "1" from 1).
		return fmt.Sprintf("%q", val)
	default:
		return fmt.Sprint(v)
	}
}

// promDurationUnit matches one <number><unit> component of a Prometheus-
// style duration string (e.g. "1d12h30m"). Alternation order matters: "ms"
// must be tried before "m", or Go's leftmost-first regexp semantics would
// match the "m" alternative first and leave a dangling "s". The grammar is
// case-sensitive: "M" (month) and "Q" (quarter) are distinct units from "m"
// (minute).
var promDurationUnit = regexp.MustCompile(`^(\d+)(y|w|d|h|ms|m|s|M|Q)`)

// promDurationUnitLength approximates a calendar month as 30 days and a
// quarter as 3 such months (90 days) -- the OpenAPI spec's Duration schema
// (pattern `(\d+(ms|s|m|h|d|w|M|Q|y))+`) documents M and Q as valid units
// but, like every calendar-based duration unit, doesn't give them a fixed
// length; this matches the common approximation other monitoring tools use
// for the same units (e.g. Grafana's relative time ranges).
var promDurationUnitLength = map[string]time.Duration{
	"y":  365 * 24 * time.Hour,
	"Q":  90 * 24 * time.Hour,
	"M":  30 * 24 * time.Hour,
	"w":  7 * 24 * time.Hour,
	"d":  24 * time.Hour,
	"h":  time.Hour,
	"m":  time.Minute,
	"s":  time.Second,
	"ms": time.Millisecond,
}

// parseDuration parses a duration string, trying Go's time.ParseDuration
// first and falling back to Prometheus's own duration syntax, which
// additionally supports y/w/d/M/Q suffixes (1y = 365d, 1Q = 90d, 1M = 30d,
// 1w = 7d, 1d = 24h) that time.ParseDuration doesn't understand. for,
// keep_firing_for, and interval
// are written in Prometheus's format, not Go's, and day/week units are
// common in hand-written alerting rules ("1d", "2w").
func parseDuration(s string) (time.Duration, error) {
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	if s == "" {
		// The loop below would otherwise treat an empty string as a
		// trivially valid zero duration by never executing at all.
		return 0, fmt.Errorf("invalid duration %q", s)
	}

	remaining := s
	var total time.Duration
	for remaining != "" {
		m := promDurationUnit.FindStringSubmatch(remaining)
		if m == nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q: %w", s, err)
		}
		total += time.Duration(n) * promDurationUnitLength[m[2]]
		remaining = remaining[len(m[0]):]
	}
	return total, nil
}

// orderInsensitiveKeys are map keys whose slice values represent sets rather
// than sequences, so reordering them is not drift: permission grants, the
// actions within a grant, notification channels, and a SyntheticCheck's
// execution regions (which run independently -- "all_locations" is a
// SyntheticCheckSchedulingStrategy, not a sequence).
var orderInsensitiveKeys = map[string]bool{
	"permissions": true,
	"actions":     true,
	"channels":    true,
	"locations":   true,
}

// durationComparisonFields are map keys whose string values are Go duration
// strings, so equal values in different textual forms (e.g. "2m" and "2m0s")
// must compare equal. Scoped by field name rather than by "does this string
// happen to parse as a duration," which would otherwise treat unrelated
// duration-shaped strings (a name, a timestamp) as equal too.
//
// Covers every Duration-typed field on the asset kinds this package
// documents as supported (SyntheticCheck retries/backoff, PrometheusRule,
// SLO). Deliberately excludes "window", "timeout", and "value" -- real
// Duration fields too, but on AgenticWorkflow and SslCertificateAssertion,
// which aren't yet part of this package's supported-kinds list, and generic
// enough names to risk colliding with an unrelated field elsewhere in the
// tree (e.g. an assertion's status-code "value").
var durationComparisonFields = map[string]bool{
	"for":             true,
	"keep_firing_for": true,
	"interval":        true,
	"delay":           true,
	"maximumDelay":    true,
	"alertAfter":      true,
	"lookbackWindow":  true,
	"timeSliceWindow": true,
	"staleAfter":      true,
}

// pathEndsAtKey reports whether the map key immediately owning the value at
// the end of p is in keys. Searches backward, skipping the TypeAssertion
// steps cmp inserts when unwrapping an any-typed map value into its concrete
// dynamic type, and stops at the first cmp.MapIndex (a real match or
// mismatch) or cmp.SliceIndex (the value is a slice element, not owned by a
// map key at all -- never a match).
func pathEndsAtKey(p cmp.Path, keys map[string]bool) bool {
	for i := len(p) - 1; i >= 0; i-- {
		switch step := p[i].(type) {
		case cmp.MapIndex:
			key, ok := step.Key().Interface().(string)
			return ok && keys[key]
		case cmp.SliceIndex:
			return false
		}
	}
	return false
}

// Equivalent reports whether two documents are semantically equivalent for
// drift-detection purposes, ignoring fields that don't matter for that
// decision: server-managed metadata, empty containers, default values,
// non-preserved annotations, slice element order, and duration-string
// formatting differences (e.g. "2m" == "2m0s").
//
// a is the reference document (typically the user's local definition) and b
// is the value to compare against (typically the current API state).
// additionalIgnoredFields are extra field paths to strip beyond the
// defaults (e.g. ConditionallyIgnoredFields filtered by AbsentFields, or a
// kind-specific API-managed field like "spec.routing.assets").
// preservedAnnotationKeys lists annotation keys that participate in drift
// detection; every other annotation is stripped before comparison.
// WithAnnotationsUnfiltered (and WithFlatDocument, which implies it) disables
// this filtering entirely, making preservedAnnotationKeys inert -- every
// annotation key participates in the comparison instead.
func Equivalent(a, b []byte, additionalIgnoredFields []string, preservedAnnotationKeys []string, opts ...Option) (bool, error) {
	// sigsyaml.Unmarshal (inside normalizeToMap) decodes through
	// encoding/json, which always produces float64 for a JSON/YAML number
	// regardless of its original notation (3, 3.0, 3e0) -- both documents
	// already agree on numeric type once parsed, so no further cross-type
	// numeric normalization is needed here. That decode step is also where
	// precision is lost for an integer beyond 2^53 (float64's exact-integer
	// limit): two distinct values that large can come out equal. No field
	// compared by this package needs an exact integer beyond that range
	// today (Dash0 asset IDs are strings, and the numeric fields here are
	// thresholds, small counts, and durations) -- if one ever does, this
	// decode path would need to switch to a precision-preserving number
	// type end-to-end.
	parsedA, err := normalizeToMap(a, additionalIgnoredFields, preservedAnnotationKeys, opts...)
	if err != nil {
		return false, fmt.Errorf("error normalizing first document: %w", err)
	}
	parsedB, err := normalizeToMap(b, additionalIgnoredFields, preservedAnnotationKeys, opts...)
	if err != nil {
		return false, fmt.Errorf("error normalizing second document: %w", err)
	}

	// Strip zero-value fields from B that are absent in A. This prevents
	// API-enriched defaults (e.g. "enabled": false, "retries": null) from
	// being treated as drift when the reference doesn't set those fields.
	// If the reference explicitly set a zero value, it will be present in A
	// and preserved.
	stripAbsentZeroValues(parsedA, parsedB)

	cmpOptions := []cmp.Option{
		// Ignore slice order only for fields that are semantically sets
		// rather than sequences (see orderInsensitiveKeys). Every other
		// slice -- notably spec.groups[].rules, which Prometheus evaluates
		// in declaration order and where a recording rule can depend on one
		// declared above it -- stays order-sensitive by default, so a
		// meaningful reorder surfaces as drift instead of being silently
		// discarded. Uses canonicalString, which recursively sorts nested
		// structures, so the sort key stays stable regardless of inner
		// element ordering.
		cmp.FilterPath(
			func(p cmp.Path) bool { return pathEndsAtKey(p, orderInsensitiveKeys) },
			cmpopts.SortSlices(func(x, y any) bool {
				return canonicalString(x) < canonicalString(y)
			}),
		),
		// Duration-aware string comparison for durationComparisonFields:
		// treats "2m" and "2m0s" as equivalent. Scoped by field name (see
		// durationComparisonFields) rather than by "do both strings happen
		// to parse as a duration" -- the latter would also match unrelated
		// fields, e.g. a name of "1h" against a name of "60m".
		cmp.FilterPath(
			func(p cmp.Path) bool { return pathEndsAtKey(p, durationComparisonFields) },
			cmp.Comparer(func(x, y string) bool {
				dx, errX := parseDuration(x)
				dy, errY := parseDuration(y)
				if errX != nil || errY != nil {
					return x == y
				}
				return dx == dy
			}),
		),
	}
	return cmp.Equal(parsedA, parsedB, cmpOptions...), nil
}

// stripAbsentZeroValues removes keys from target that don't exist in
// reference and have zero values (false, 0, empty string, empty map, empty
// slice, nil). Applied recursively to nested maps.
func stripAbsentZeroValues(reference, target map[string]any) {
	for key, targetVal := range target {
		refVal, existsInRef := reference[key]
		if !existsInRef {
			if isZeroValue(targetVal) {
				delete(target, key)
			}
			continue
		}
		switch refTyped := refVal.(type) {
		case map[string]any:
			if targetMap, ok := targetVal.(map[string]any); ok {
				stripAbsentZeroValues(refTyped, targetMap)
				if len(targetMap) == 0 {
					delete(target, key)
				}
			}
		case []any:
			if targetSlice, ok := targetVal.([]any); ok {
				if orderInsensitiveKeys[key] {
					// Raw index-pairing assumes matching position, which a
					// reordered order-insensitive set (e.g. permissions)
					// violates -- but pairing by sorted canonical form
					// doesn't work either, because the very zero-value
					// field being stripped is also part of what the sort
					// key is computed from, so it can shift an element's
					// sorted position differently on each side. Match by
					// shared-field identity instead.
					stripAbsentZeroValuesByBestMatch(refTyped, targetSlice)
				} else {
					stripAbsentZeroValuesInSlices(refTyped, targetSlice)
				}
			}
		}
	}
}

// stripAbsentZeroValuesByBestMatch pairs each target map element with the
// reference map element it shares the most identical (key, value) pairs
// with, for order-insensitive slices where position doesn't identify an
// element. Greedily claims the best-scoring available reference element for
// each target element in turn; a target element with no field in common
// with any remaining reference element is left unpaired (and unstripped) --
// a wrong strip is worse than a missed one.
func stripAbsentZeroValuesByBestMatch(reference, target []any) {
	refMaps := make([]map[string]any, 0, len(reference))
	for _, r := range reference {
		if m, ok := r.(map[string]any); ok {
			refMaps = append(refMaps, m)
		}
	}
	claimed := make([]bool, len(refMaps))
	for _, t := range target {
		targetMap, ok := t.(map[string]any)
		if !ok {
			continue
		}
		best, bestScore := -1, 0
		for i, refMap := range refMaps {
			if claimed[i] {
				continue
			}
			if score := matchingFieldCount(refMap, targetMap); score > bestScore {
				best, bestScore = i, score
			}
		}
		if best == -1 {
			continue
		}
		claimed[best] = true
		stripAbsentZeroValues(refMaps[best], targetMap)
	}
}

// matchingFieldCount counts keys present in both a and b whose values have
// the same canonical form.
func matchingFieldCount(a, b map[string]any) int {
	count := 0
	for k, v := range a {
		if bv, ok := b[k]; ok && canonicalString(v) == canonicalString(bv) {
			count++
		}
	}
	return count
}

// stripAbsentZeroValuesInSlices pairs slice elements by index and recurses
// into map-shaped pairs, mirroring stripAbsentZeroValues for list elements
// (e.g. spec.groups[].rules[].dash0Enabled, which the API adds per-rule).
// Elements beyond the shorter slice's length are left untouched -- a genuine
// length mismatch is real drift, not something to paper over here.
func stripAbsentZeroValuesInSlices(reference, target []any) {
	n := min(len(reference), len(target))
	for i := range n {
		refMap, ok := reference[i].(map[string]any)
		if !ok {
			continue
		}
		targetMap, ok := target[i].(map[string]any)
		if !ok {
			continue
		}
		stripAbsentZeroValues(refMap, targetMap)
	}
}

// isZeroValue returns true if v is a JSON zero value (false, 0, "", nil,
// empty map, or empty slice).
func isZeroValue(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case bool:
		return !val
	case float64:
		return val == 0
	case int:
		return val == 0
	case string:
		return val == ""
	case map[string]any:
		return isAllZeroValues(val)
	case []any:
		return len(val) == 0
	default:
		return false
	}
}

// isAllZeroValues reports whether every value in m is itself a zero value
// per isZeroValue -- e.g. {"enabled": false} -- so a whole API-added default
// substructure absent from the reference is recognized as equivalent to
// absence, the same as a single zero-valued field already is. Distinct from
// isEmpty, which cleanupMap uses for a narrower, unrelated question (is this
// container structurally vacuous) where a bool false or number 0 is real,
// meaningful content, not something to delete during normalization.
func isAllZeroValues(m map[string]any) bool {
	for _, value := range m {
		if !isZeroValue(value) {
			return false
		}
	}
	return true
}

// AbsentFields returns which of the given dot-separated field paths are
// absent from the parsed document.
// Used to conditionally ignore API-managed fields the reference document
// didn't include (e.g. ConditionallyIgnoredFields).
func AbsentFields(data []byte, fields []string) []string {
	var parsed map[string]any
	if err := sigsyaml.Unmarshal(data, &parsed); err != nil {
		return nil // on error, don't ignore anything extra (safe default)
	}

	var absent []string
	for _, field := range fields {
		if !hasFieldPath(parsed, field) {
			absent = append(absent, field)
		}
	}
	return absent
}

// hasFieldPath checks if a dot-separated path exists in a nested map. A
// segment that resolves to a slice (e.g. "spec.groups.rules.dash0Enabled",
// where "groups" and "rules" are both arrays) is present if the remaining
// path exists on at least one element -- matching how a human would
// describe "does this document specify the field."
func hasFieldPath(data map[string]any, path string) bool {
	parts := strings.SplitN(path, ".", 2)
	val, exists := data[parts[0]]
	if !exists || val == nil {
		return false
	}
	if len(parts) == 1 {
		return true
	}
	switch v := val.(type) {
	case map[string]any:
		return hasFieldPath(v, parts[1])
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok && hasFieldPath(m, parts[1]) {
				return true
			}
		}
		return false
	default:
		return false
	}
}
