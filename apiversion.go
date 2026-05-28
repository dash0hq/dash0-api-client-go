package dash0

import "strings"

// NormalizeDash0ApiVersion strips a Dash0-owned apiGroup prefix from an
// apiVersion string and returns the bare version. It accepts:
//
//   - Bare versions like "v1alpha1" or "v1alpha2" (no slash, returned as-is).
//   - Group-prefixed forms whose group is "dash0.com" or a subdomain of
//     "dash0.com" (e.g. "operator.dash0.com/v1alpha1"). The group prefix is
//     removed and only the version after the slash is returned.
//
// ok is false (and version is "") when the apiVersion carries a group prefix
// that is neither "dash0.com" nor a "*.dash0.com" subdomain — including the
// pathological "/v1alpha1" (empty prefix) and group names that merely embed
// "dash0.com" as a substring (e.g. "evildash0.com").
//
// Callers are responsible for matching the returned version against their
// asset's supported set. This helper only enforces the apiGroup contract:
// the version itself can be anything once the prefix check passes.
//
// Motivation: the Dash0 Kubernetes operator marshals its custom resources
// directly into Dash0 API requests, which leaks "metav1.TypeMeta" — so an
// asset written by the operator carries an apiGroup-prefixed apiVersion
// rather than the bare version the API contract specifies. Tolerating that
// prefix on the client side lets old API client builds still read those
// records.
func NormalizeDash0ApiVersion(apiVersion string) (version string, ok bool) {
	idx := strings.LastIndex(apiVersion, "/")
	if idx < 0 {
		return apiVersion, true
	}
	prefix := apiVersion[:idx]
	if prefix != "dash0.com" && !strings.HasSuffix(prefix, ".dash0.com") {
		return "", false
	}
	return apiVersion[idx+1:], true
}
