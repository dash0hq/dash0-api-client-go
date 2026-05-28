package dash0

import "testing"

func TestNormalizeDash0ApiVersion(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantVersion string
		wantOK      bool
	}{
		{"empty", "", "", true},
		{"bare version", "v1alpha1", "v1alpha1", true},
		{"another bare version", "v1beta3", "v1beta3", true},
		{"operator group v1alpha1", "operator.dash0.com/v1alpha1", "v1alpha1", true},
		{"operator group v1alpha2", "operator.dash0.com/v1alpha2", "v1alpha2", true},
		{"bare dash0.com group", "dash0.com/v1alpha1", "v1alpha1", true},
		{"deeper Dash0 subgroup", "api.platform.dash0.com/v1", "v1", true},
		{"unknown version under dash0.com still normalizes",
			"operator.dash0.com/v1beta1", "v1beta1", true},
		{"foreign group is rejected", "monitoring.coreos.com/v1", "", false},
		{"prefix masquerading as dash0.com is rejected", "evildash0.com/v1alpha1", "", false},
		{"empty prefix is rejected", "/v1alpha1", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NormalizeDash0ApiVersion(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("input=%q: ok=%v, want %v", tt.input, ok, tt.wantOK)
			}
			if got != tt.wantVersion {
				t.Errorf("input=%q: version=%q, want %q", tt.input, got, tt.wantVersion)
			}
		})
	}
}
