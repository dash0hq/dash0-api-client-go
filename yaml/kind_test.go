package yaml

import (
	"testing"
)

func TestDetectKind(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    string
		wantErr bool
	}{
		{"explicit kind", "kind: PrometheusRule\nspec: {}", "PrometheusRule", false},
		{"inferred check rule", "name: MyRule\nexpression: up == 0", "CheckRule", false},
		{"no kind no expression", "name: MyDashboard\nspec: {}", "", false},
		{"empty", "", "", false},
		{"invalid YAML", "{{invalid", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectKind([]byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("DetectKind() = %q, want %q", got, tt.want)
			}
		})
	}
}
