package dash0

import (
	"reflect"
	"testing"
)

func TestNewListOptions(t *testing.T) {
	tests := []struct {
		name       string
		opts       []ListOption
		wantPrefix *string
	}{
		{"no options", nil, nil},
		{"origin prefix", []ListOption{WithOriginPrefix("tf_")}, Ptr("tf_")},
		{"empty prefix leaves filter unset", []ListOption{WithOriginPrefix("")}, nil},
		{"later option wins", []ListOption{WithOriginPrefix("a_"), WithOriginPrefix("b_")}, Ptr("b_")},
		{"empty prefix clears an earlier one", []ListOption{WithOriginPrefix("a_"), WithOriginPrefix("")}, nil},
		{"nil option is skipped", []ListOption{nil, WithOriginPrefix("tf_")}, Ptr("tf_")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewListOptions(tt.opts...).OriginPrefix
			if tt.wantPrefix == nil {
				if got != nil {
					t.Errorf("OriginPrefix = %q, want nil", *got)
				}
				return
			}
			assertPtrEqual(t, "OriginPrefix", got, *tt.wantPrefix)
		})
	}
}

func TestNewListOptions_SignalToMetricsFilters(t *testing.T) {
	t.Run("no options leave every filter unset", func(t *testing.T) {
		got := NewListOptions()
		if got.SignalToMetricsName != nil || got.Signal != nil || got.Enabled != nil {
			t.Errorf("NewListOptions() = %+v, want every filter nil", got)
		}
	})

	t.Run("each option sets its filter", func(t *testing.T) {
		got := NewListOptions(WithSignalToMetricsName("checkout"), WithSignal(SignalToMetricsSignalTypeLogs), WithEnabled(false))
		assertPtrEqual(t, "SignalToMetricsName", got.SignalToMetricsName, "checkout")
		assertPtrEqual(t, "Signal", got.Signal, SignalToMetricsSignalTypeLogs)
		assertPtrEqual(t, "Enabled", got.Enabled, false)
	})

	t.Run("empty values clear an earlier name and signal", func(t *testing.T) {
		got := NewListOptions(WithSignalToMetricsName("checkout"), WithSignal(SignalToMetricsSignalTypeLogs), WithSignalToMetricsName(""), WithSignal(""))
		if got.SignalToMetricsName != nil {
			t.Errorf("SignalToMetricsName = %q, want nil", *got.SignalToMetricsName)
		}
		if got.Signal != nil {
			t.Errorf("Signal = %q, want nil", *got.Signal)
		}
	})

	t.Run("later enabled option wins", func(t *testing.T) {
		got := NewListOptions(WithEnabled(true), WithEnabled(false))
		assertPtrEqual(t, "Enabled", got.Enabled, false)
	})
}

func TestListOptions_requireOriginPrefixOnly(t *testing.T) {
	const wantErr = "dash0: list check rules supports only the origin prefix filter"
	tests := []struct {
		name    string
		opts    []ListOption
		wantErr string
	}{
		{"no options", nil, ""},
		{"origin prefix", []ListOption{WithOriginPrefix("tf_")}, ""},
		{"empty origin prefix", []ListOption{WithOriginPrefix("")}, ""},
		{"empty name and signal leave the filters unset", []ListOption{WithSignalToMetricsName(""), WithSignal("")}, ""},
		{"name cleared by a later empty name", []ListOption{WithSignalToMetricsName("checkout"), WithSignalToMetricsName("")}, ""},
		{"name", []ListOption{WithSignalToMetricsName("checkout")}, wantErr},
		{"signal", []ListOption{WithSignal(SignalToMetricsSignalTypeSpans)}, wantErr},
		{"enabled", []ListOption{WithEnabled(true)}, wantErr},
		{"enabled false", []ListOption{WithEnabled(false)}, wantErr},
		{"origin prefix with name", []ListOption{WithOriginPrefix("tf_"), WithSignalToMetricsName("checkout")}, wantErr},
		{"origin prefix with signal", []ListOption{WithOriginPrefix("tf_"), WithSignal(SignalToMetricsSignalTypeLogs)}, wantErr},
		{"origin prefix with enabled", []ListOption{WithOriginPrefix("tf_"), WithEnabled(true)}, wantErr},
		{"every filter", []ListOption{WithOriginPrefix("tf_"), WithSignalToMetricsName("checkout"), WithSignal(SignalToMetricsSignalTypeSpans), WithEnabled(true)}, wantErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewListOptions(tt.opts...).requireOriginPrefixOnly("list check rules")
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("requireOriginPrefixOnly() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("requireOriginPrefixOnly() = nil, want %q", tt.wantErr)
			}
			assertEqual(t, "error", err.Error(), tt.wantErr)
		})
	}
}

func TestListOptions_requireOriginPrefixOnly_namesTheOperation(t *testing.T) {
	err := NewListOptions(WithSignalToMetricsName("checkout")).requireOriginPrefixOnly("list SLOs")
	if err == nil {
		t.Fatal("requireOriginPrefixOnly() = nil, want an error")
	}
	assertEqual(t, "error", err.Error(), "dash0: list SLOs supports only the origin prefix filter")
}

// TestListOptions_requireOriginPrefixOnly_rejectsEveryOtherField sets each ListOptions field other than OriginPrefix in turn, with and without an origin prefix, so a filter added later is rejected without a new test case.
func TestListOptions_requireOriginPrefixOnly_rejectsEveryOtherField(t *testing.T) {
	typ := reflect.TypeOf(ListOptions{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "OriginPrefix" {
			continue
		}
		for _, prefix := range []*string{nil, Ptr("tf_")} {
			name := field.Name
			if prefix != nil {
				name += " with origin prefix"
			}
			t.Run(name, func(t *testing.T) {
				o := ListOptions{OriginPrefix: prefix}
				value := reflect.ValueOf(&o).Elem().Field(i)
				value.Set(reflect.New(value.Type().Elem()))
				if err := o.requireOriginPrefixOnly("list check rules"); err == nil {
					t.Errorf("requireOriginPrefixOnly() with %s set = nil, want an error", name)
				}
			})
		}
	}
}
