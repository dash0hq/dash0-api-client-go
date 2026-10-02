package dash0

import "testing"

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
