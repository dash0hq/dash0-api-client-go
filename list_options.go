package dash0

// ListOption configures optional filters on List* methods.
// List methods whose endpoint does not support a given filter do not accept options.
type ListOption func(*ListOptions)

// ListOptions holds the filters resolved from a set of [ListOption] values.
type ListOptions struct {
	// OriginPrefix restricts results to assets whose dash0.com/origin starts with the given string.
	// Nil means no filter.
	OriginPrefix *string
}

// WithOriginPrefix restricts a list call to assets whose dash0.com/origin starts with prefix.
// An empty prefix leaves the filter unset.
// Supported by [Client.ListCheckRules], [Client.ListRecordingRules], [Client.ListSLOs], [Client.ListSignalToMetrics], and their Iter variants.
func WithOriginPrefix(prefix string) ListOption {
	return func(o *ListOptions) {
		if prefix == "" {
			o.OriginPrefix = nil
			return
		}
		o.OriginPrefix = &prefix
	}
}

// NewListOptions applies opts in order and returns the resolved filters.
// Nil options are skipped.
// It is exported so that mock implementations of [Client] can inspect the options a caller passed.
func NewListOptions(opts ...ListOption) ListOptions {
	var o ListOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}
