package yaml

import (
	"testing"
)

func TestDetectKind(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{"explicit kind", "kind: PrometheusRule\nspec: {}", "PrometheusRule"},
		{"inferred check rule", "name: MyRule\nexpression: up == 0", "CheckRule"},
		{"no kind no expression", "name: MyDashboard\nspec: {}", ""},
		{"invalid YAML", "{{invalid", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectKind([]byte(tt.data))
			if got != tt.want {
				t.Errorf("DetectKind() = %q, want %q", got, tt.want)
			}
		})
	}
}
