package dash0

// Ptr returns a pointer to the given value. This is useful for creating
// pointers to literals when calling API methods that accept optional parameters.
//
// Example:
//
//	client.ListDashboards(ctx, dash0.Ptr("default"))
func Ptr[T any](v T) *T {
	return &v
}

// String returns a pointer to the given string value.
// This is a convenience wrapper around Ptr for the common case of string pointers.
//
// Example:
//
//	client.ListDashboards(ctx, dash0.String("default"))
func String(v string) *string {
	return &v
}

// Int64 returns a pointer to the given int64 value.
func Int64(v int64) *int64 {
	return &v
}

// Bool returns a pointer to the given bool value.
func Bool(v bool) *bool {
	return &v
}

// Float64 returns a pointer to the given float64 value.
func Float64(v float64) *float64 {
	return &v
}

// StringValue returns the value of a string pointer, or empty string if nil.
func StringValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// Int64Value returns the value of an int64 pointer, or 0 if nil.
func Int64Value(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// BoolValue returns the value of a bool pointer, or false if nil.
func BoolValue(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

// Float64Value returns the value of a float64 pointer, or 0 if nil.
func Float64Value(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// toPointerSlice converts a slice of values to a slice of pointers.
func toPointerSlice[T any](slice []T) []*T {
	result := make([]*T, len(slice))
	for i := range slice {
		result[i] = &slice[i]
	}
	return result
}
