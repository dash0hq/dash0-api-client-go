package dash0

import "fmt"

// ListOption configures optional filters on List* methods.
// List methods whose endpoint does not support a given filter do not accept options.
type ListOption func(*ListOptions)

// ListOptions holds the filters resolved from a set of [ListOption] values.
type ListOptions struct {
	// OriginPrefix restricts results to assets whose dash0.com/origin starts with the given string.
	// Nil means no filter.
	OriginPrefix *string

	// Name restricts results to assets whose name contains the given string, ignoring case, or whose ID equals it.
	// Nil means no filter.
	Name *string

	// Signal restricts results to signal-to-metrics rules with the given source signal.
	// Nil means no filter.
	Signal *SignalToMetricsSignalType

	// Enabled restricts results to assets with the given enabled state.
	// Nil means no filter.
	Enabled *bool
}

// WithOriginPrefix restricts a list call to assets whose dash0.com/origin starts with prefix.
// An empty prefix leaves the filter unset.
// Supported by [Client.ListCheckRules], [Client.ListRecordingRules], [Client.ListSLOs], [Client.ListSignalToMetrics], [Client.CountSignalToMetrics], and their Iter variants.
func WithOriginPrefix(prefix string) ListOption {
	return func(o *ListOptions) {
		if prefix == "" {
			o.OriginPrefix = nil
			return
		}
		o.OriginPrefix = &prefix
	}
}

// WithName restricts a list call to assets whose name contains name, ignoring case, or whose ID equals name.
// An empty name leaves the filter unset.
// Supported by [Client.ListSignalToMetrics], [Client.ListSignalToMetricsIter], and [Client.CountSignalToMetrics].
// The other list methods return an error when given this option.
func WithName(name string) ListOption {
	return func(o *ListOptions) {
		if name == "" {
			o.Name = nil
			return
		}
		o.Name = &name
	}
}

// WithSignal restricts a list call to signal-to-metrics rules whose source signal is signal.
// An empty signal leaves the filter unset.
// Supported by [Client.ListSignalToMetrics], [Client.ListSignalToMetricsIter], and [Client.CountSignalToMetrics].
// The other list methods return an error when given this option.
func WithSignal(signal SignalToMetricsSignalType) ListOption {
	return func(o *ListOptions) {
		if signal == "" {
			o.Signal = nil
			return
		}
		o.Signal = &signal
	}
}

// WithEnabled restricts a list call to assets whose enabled state is enabled.
// Supported by [Client.ListSignalToMetrics], [Client.ListSignalToMetricsIter], and [Client.CountSignalToMetrics].
// The other list methods return an error when given this option.
func WithEnabled(enabled bool) ListOption {
	return func(o *ListOptions) {
		o.Enabled = &enabled
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

// requireOriginPrefixOnly returns an error when o sets a filter other than OriginPrefix.
// List methods whose endpoint filters by origin prefix alone call it, so an unsupported filter fails the call instead of being ignored.
// The operation names the call in the error, for example "list check rules".
func (o ListOptions) requireOriginPrefixOnly(operation string) error {
	switch {
	case o.Name != nil:
		return fmt.Errorf("dash0: %s does not support the name filter", operation)
	case o.Signal != nil:
		return fmt.Errorf("dash0: %s does not support the signal filter", operation)
	case o.Enabled != nil:
		return fmt.Errorf("dash0: %s does not support the enabled filter", operation)
	}
	return nil
}
