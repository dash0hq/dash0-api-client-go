package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sigsyaml "sigs.k8s.io/yaml"
)

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want any
	}{
		{"block key", "x: 0\ny: 0\n", map[string]any{"x": float64(0), "y": float64(0)}},
		{"every YAML 1.1 bool word as key", "n: 1\nyes: 2\nOFF: 3\n", map[string]any{"n": float64(1), "yes": float64(2), "OFF": float64(3)}},
		{"nested and flow keys", "a:\n  - {ä: 1, y: 2}\n", map[string]any{"a": []any{map[string]any{"ä": float64(1), "y": float64(2)}}}},
		{"values keep YAML 1.1 meaning", "enabled: yes\nmode: off\n", map[string]any{"enabled": true, "mode": false}},
		{"quoted key untouched", "'y': 1\n", map[string]any{"y": float64(1)}},
		{"broken later document", "a:\n  y: 1\n---\nb: [unclosed\n", map[string]any{"a": map[string]any{"y": float64(1)}}},
		{"empty first document", "---\n---\ny: 1\n", nil},
		{"merge key", "base: &b {k: 1}\nx:\n  <<: *b\n  y: 2\n", map[string]any{"base": map[string]any{"k": float64(1)}, "x": map[string]any{"k": float64(1), "y": float64(2)}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got any
			require.NoError(t, Unmarshal([]byte(tt.in), &got))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnmarshal_MatchesSigsWhenNoKeyIsQuoted(t *testing.T) {
	for _, in := range []string{
		`{"y": 0, "enabled": true}`,
		"enabled: yes\nmode: [on, off]\n",
		"not: [valid",
	} {
		var want, got any
		wantErr := sigsyaml.Unmarshal([]byte(in), &want)
		gotErr := Unmarshal([]byte(in), &got)
		assert.Equal(t, wantErr, gotErr, in)
		assert.Equal(t, want, got, in)
	}
}

func TestToJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"bool-word keys stay strings", "y: 0\nn: 1\nOFF: 2\n", `{"OFF":2,"n":1,"y":0}`},
		{"bool-word values stay strings", "name: off\nregion: NO\nenabled: yes\n", `{"enabled":"yes","name":"off","region":"NO"}`},
		{"bool-word list items stay strings", "countries: [GB, NO]\n", `{"countries":["GB","NO"]}`},
		{"YAML 1.2 booleans stay booleans", "a: true\nb: False\n", `{"a":true,"b":false}`},
		{"only the first document", "a: 1\n---\nb: 2\n", `{"a":1}`},
		{"JSON input", `{"y": 0, "enabled": "yes"}`, `{"enabled":"yes","y":0}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToJSON([]byte(tt.in))
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))
		})
	}
}

func TestToJSON_Errors(t *testing.T) {
	for _, in := range []string{"not: [valid", "true: a\n", "inf: .inf\n"} {
		_, err := ToJSON([]byte(in))
		assert.Error(t, err, in)
	}
}
