package yaml

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	sigsyaml "sigs.k8s.io/yaml"
)

// ConditionallyIgnoredFields are fields ignored during comparison only when
// absent from the reference document (typically the user's local
// definition). These are fields the API enriches on retrieval but that
// users may optionally manage. When present in the user's document, drift
// detection is preserved. Paths are relative to the document root
// regardless of AnnotationsRoot.
var ConditionallyIgnoredFields = []string{
	"metadata.name",    // server-generated when absent, but user-declared intent when present
	"spec.permissions", // API-managed: stored separately, enriched on retrieval
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
func Normalize(data []byte, additionalIgnoredFields []string, preservedAnnotationKeys []string, opts ...Option) ([]byte, error) {
	o := resolveOptions(opts)

	var parsed map[string]any
	if err := sigsyaml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("error parsing document: %w", err)
	}

	cleanupMap(parsed, prefixedIgnoredFields(o.AnnotationsRoot, additionalIgnoredFields))
	stripAnnotations(parsed, o.AnnotationsRoot, preservedAnnotationKeys)

	encoded, err := sigsyaml.Marshal(parsed)
	if err != nil {
		return nil, fmt.Errorf("error encoding document: %w", err)
	}

	return []byte(strings.TrimSuffix(string(encoded), "\n")), nil
}

// mapAtRoot returns the map at the given dot-separated root path within
// data, or data itself when root is "". Only used for the single-level
// "metadata" root Dash0 assets use today; deeper roots are not needed.
func mapAtRoot(data map[string]any, root string) (map[string]any, bool) {
	if root == "" {
		return data, true
	}
	m, ok := data[root].(map[string]any)
	return m, ok
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
		delete(data, root)
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
		if (key == "dash0-threshold-critical" || key == "dash0-threshold-degraded") && strVal == "0" {
			delete(annotations, key)
		}
		if key == "dash0-enabled" && strVal == "true" {
			delete(annotations, key)
		}
	}
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
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					cleanupMap(m, nil)
				}
			}
			if len(v) == 0 {
				delete(data, key)
			}
		case string:
			if v == "" {
				delete(data, key)
			} else if key == "for" || key == "keep_firing_for" {
				// for and keep_firing_for use Duration with omitempty, so
				// marshalling drops them when the value is zero. Remove them
				// here so "for: 0s" in a user document matches the
				// round-tripped form that omits the field. If parsing fails,
				// the value is not a duration, so keep it as-is.
				if d, err := time.ParseDuration(v); err == nil && d == 0 {
					delete(data, key)
				}
			}
		}
	}
}

// isEmpty checks if a map is empty or contains only empty/nil values.
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
	default:
		return fmt.Sprint(v)
	}
}

// normalizeNumericTypes recursively converts all integer and float types to
// float64 in a parsed YAML/JSON structure, so the same numeric value
// appearing as different types (e.g. int from YAML vs. float64 from JSON)
// compares equal.
func normalizeNumericTypes(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, v := range val {
			val[k] = normalizeNumericTypes(v)
		}
		return val
	case []any:
		for i, v := range val {
			val[i] = normalizeNumericTypes(v)
		}
		return val
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case float32:
		return float64(val)
	default:
		return v
	}
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
func Equivalent(a, b []byte, additionalIgnoredFields []string, preservedAnnotationKeys []string, opts ...Option) (bool, error) {
	normalizedA, err := Normalize(a, additionalIgnoredFields, preservedAnnotationKeys, opts...)
	if err != nil {
		return false, fmt.Errorf("error normalizing first document: %w", err)
	}
	normalizedB, err := Normalize(b, additionalIgnoredFields, preservedAnnotationKeys, opts...)
	if err != nil {
		return false, fmt.Errorf("error normalizing second document: %w", err)
	}

	var parsedA, parsedB any
	if err := sigsyaml.Unmarshal(normalizedA, &parsedA); err != nil {
		return false, fmt.Errorf("error parsing first normalized document: %w", err)
	}
	if err := sigsyaml.Unmarshal(normalizedB, &parsedB); err != nil {
		return false, fmt.Errorf("error parsing second normalized document: %w", err)
	}

	parsedA = normalizeNumericTypes(parsedA)
	parsedB = normalizeNumericTypes(parsedB)

	// Strip zero-value fields from B that are absent in A. This prevents
	// API-enriched defaults (e.g. "enabled": false, "retries": null) from
	// being treated as drift when the reference doesn't set those fields.
	// If the reference explicitly set a zero value, it will be present in A
	// and preserved.
	if mapA, ok := parsedA.(map[string]any); ok {
		if mapB, ok := parsedB.(map[string]any); ok {
			stripAbsentZeroValues(mapA, mapB)
		}
	}

	cmpOptions := []cmp.Option{
		// Ignore slice order deeper in the structure. Uses canonicalString,
		// which recursively sorts nested structures, so the sort key stays
		// stable regardless of inner element ordering.
		cmpopts.SortSlices(func(x, y any) bool {
			return canonicalString(x) < canonicalString(y)
		}),
		// Duration-aware string comparison: treats "2m" and "2m0s" as
		// equivalent when both strings are valid Go duration strings.
		cmp.FilterValues(
			func(x, y string) bool {
				_, errX := time.ParseDuration(x)
				_, errY := time.ParseDuration(y)
				return errX == nil && errY == nil
			},
			cmp.Comparer(func(x, y string) bool {
				dx, _ := time.ParseDuration(x)
				dy, _ := time.ParseDuration(y)
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
		if refMap, ok := refVal.(map[string]any); ok {
			if targetMap, ok := targetVal.(map[string]any); ok {
				stripAbsentZeroValues(refMap, targetMap)
				if len(targetMap) == 0 {
					delete(target, key)
				}
			}
		}
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
		return isEmpty(val)
	case []any:
		return len(val) == 0
	default:
		return false
	}
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

// hasFieldPath checks if a dot-separated path exists in a nested map.
func hasFieldPath(data map[string]any, path string) bool {
	parts := strings.SplitN(path, ".", 2)
	val, exists := data[parts[0]]
	if !exists || val == nil {
		return false
	}
	if len(parts) == 1 {
		return true
	}
	nested, ok := val.(map[string]any)
	if !ok {
		return false
	}
	return hasFieldPath(nested, parts[1])
}
