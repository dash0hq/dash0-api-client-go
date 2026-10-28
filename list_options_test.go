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
		if got.Name != nil || got.Signal != nil || got.Enabled != nil {
			t.Errorf("NewListOptions() = %+v, want every filter nil", got)
		}
	})

	t.Run("each option sets its filter", func(t *testing.T) {
		got := NewListOptions(WithName("checkout"), WithSignal(SignalToMetricsSignalTypeLogs), WithEnabled(false))
		assertPtrEqual(t, "Name", got.Name, "checkout")
		assertPtrEqual(t, "Signal", got.Signal, SignalToMetricsSignalTypeLogs)
		assertPtrEqual(t, "Enabled", got.Enabled, false)
	})

	t.Run("empty values clear an earlier name and signal", func(t *testing.T) {
		got := NewListOptions(WithName("checkout"), WithSignal(SignalToMetricsSignalTypeLogs), WithName(""), WithSignal(""))
		if got.Name != nil {
			t.Errorf("Name = %q, want nil", *got.Name)
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
	tests := []struct {
		name    string
		opts    []ListOption
		wantErr string
	}{
		{"no options", nil, ""},
		{"origin prefix", []ListOption{WithOriginPrefix("tf_")}, ""},
		{"empty name", []ListOption{WithName("")}, ""},
		{"name", []ListOption{WithName("checkout")}, "dash0: list check rules does not support the name filter"},
		{"signal", []ListOption{WithSignal(SignalToMetricsSignalTypeSpans)}, "dash0: list check rules does not support the signal filter"},
		{"enabled", []ListOption{WithEnabled(true)}, "dash0: list check rules does not support the enabled filter"},
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

// TestListOptions_requireOriginPrefixOnly_rejectsEveryOtherField sets each ListOptions field other than OriginPrefix in turn, so a filter added later is rejected without a new test case.
func TestListOptions_requireOriginPrefixOnly_rejectsEveryOtherField(t *testing.T) {
	typ := reflect.TypeOf(ListOptions{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "OriginPrefix" {
			continue
		}
		t.Run(field.Name, func(t *testing.T) {
			var o ListOptions
			value := reflect.ValueOf(&o).Elem().Field(i)
			value.Set(reflect.New(value.Type().Elem()))
			if err := o.requireOriginPrefixOnly("list check rules"); err == nil {
				t.Errorf("requireOriginPrefixOnly() with %s set = nil, want an error", field.Name)
			}
		})
	}
}
