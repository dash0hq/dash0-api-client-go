package dash0

import (
	"fmt"
	"strings"
)

// ListOption configures optional filters on List* methods.
// Each option names the list methods that support it.
// The other list methods return an error when given it, rather than ignoring the filter.
type ListOption func(*ListOptions)

// ListOptions holds the filters resolved from a set of [ListOption] values.
type ListOptions struct {
	// OriginPrefix restricts results to assets whose dash0.com/origin starts with the given string.
	// Nil means no filter.
	OriginPrefix *string

	// SignalToMetricsName restricts results to signal-to-metrics rules that match the given search string.
	// See [WithSignalToMetricsName] for the fields it matches.
	// Nil means no filter.
	SignalToMetricsName *string

	// Signal restricts results to signal-to-metrics rules with the given source signal.
	// Nil means no filter.
	Signal *SignalToMetricsSignalType

	// Enabled restricts results to assets with the given enabled state.
	// Nil means no filter.
	Enabled *bool
}

// WithOriginPrefix restricts a list call to assets whose dash0.com/origin starts with prefix.
// An empty prefix leaves the filter unset.
// Supported by [Client.ListCheckRules], [Client.ListRecordingRules], [Client.ListSLOs], [Client.ListSignalToMetrics], [Client.CountSignalToMetrics], [Client.ListCustomSkills], and their Iter variants.
func WithOriginPrefix(prefix string) ListOption {
	return func(o *ListOptions) {
		if prefix == "" {
			o.OriginPrefix = nil
			return
		}
		o.OriginPrefix = &prefix
	}
}

// WithSignalToMetricsName restricts a list call to signal-to-metrics rules whose display name (spec.display.name) or output metric name (spec.output.name) contains name, ignoring case, or whose ID equals name, ignoring case.
// It does not search metadata.name.
// It sets the name query parameter of the signal-to-metrics list endpoint.
// An empty name leaves the filter unset.
// Supported by [Client.ListSignalToMetrics], [Client.ListSignalToMetricsIter], and [Client.CountSignalToMetrics].
// The other list methods return an error when given this option.
func WithSignalToMetricsName(name string) ListOption {
	return func(o *ListOptions) {
		if name == "" {
			o.SignalToMetricsName = nil
			return
		}
		o.SignalToMetricsName = &name
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
// Supported by [Client.ListSignalToMetrics], [Client.ListSignalToMetricsIter], [Client.CountSignalToMetrics], [Client.ListCustomSkills], and [Client.ListCustomSkillsIter].
// A custom skill is on or off per user, so for an API token without a user every readable skill counts as enabled.
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

// listFilter names a [ListOptions] field that a list method supports.
type listFilter int

const (
	filterOriginPrefix listFilter = iota
	filterEnabled
)

func (f listFilter) String() string {
	switch f {
	case filterOriginPrefix:
		return "origin prefix"
	case filterEnabled:
		return "enabled"
	default:
		return fmt.Sprintf("listFilter(%d)", int(f))
	}
}

// requireOnly returns an error when o sets a filter that is not in allowed.
// List methods call it with the filters their endpoint supports, so an unsupported filter fails the call instead of being ignored.
// It compares o against a value that holds only the allowed fields, so a filter added to ListOptions later is rejected without a change here.
// The operation names the call in the error, for example "list check rules".
func (o ListOptions) requireOnly(operation string, allowed ...listFilter) error {
	var kept ListOptions
	names := make([]string, 0, len(allowed))
	for _, filter := range allowed {
		switch filter {
		case filterOriginPrefix:
			kept.OriginPrefix = o.OriginPrefix
		case filterEnabled:
			kept.Enabled = o.Enabled
		}
		names = append(names, filter.String())
	}
	if o == kept {
		return nil
	}
	noun := "filter"
	if len(names) > 1 {
		noun = "filters"
	}
	return fmt.Errorf("dash0: %s supports only the %s %s", operation, strings.Join(names, " and "), noun)
}
