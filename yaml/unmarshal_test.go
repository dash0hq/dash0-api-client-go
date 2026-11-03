package yaml

import (
	"encoding/json"
	"testing"

	dash0 "github.com/dash0hq/dash0-api-client-go"
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

func TestUnmarshal_TelemetryTransformationRuleRoundTrip(t *testing.T) {
	const input = `kind: Dash0TelemetryTransformationRule
metadata:
  name: redact-checkout
  labels:
    dash0.com/id: rule-71
    dash0.com/origin: terraform-checkout-19
    dash0.com/dataset: production
    dash0.com/source: terraform
  annotations:
    dash0.com/created-at: "2026-10-25T10:00:00Z"
    dash0.com/updated-at: "2026-10-27T14:03:08.123456789Z"
    dash0.com/deleted-at: "2026-10-26T12:00:00Z"
spec:
  enabled: true
  signal: logs
  context: log
  priority: 100
  display:
    name: Redact checkout secrets
  conditions:
    - 'severity_number > 9'
  transformations:
    - name: redact
      statements:
        - 'delete_key(attributes, "password")'
    - name: tag
      statements:
        - 'set(attributes["team"], "payments")'
  examples:
    - name: checkout warning
      otlp:
        resourceLogs:
          - scopeLogs:
              - logRecords:
                  - timeUnixNano: "1793109788123456789"
                    body: {stringValue: Payment accepted}
                    attributes:
                      - key: order.id
                        value: {intValue: "9007199254740993"}
`
	const want = `{
  "kind":"Dash0TelemetryTransformationRule",
  "metadata":{
    "name":"redact-checkout",
    "labels":{"dash0.com/id":"rule-71","dash0.com/origin":"terraform-checkout-19","dash0.com/dataset":"production","dash0.com/source":"terraform"},
    "annotations":{"dash0.com/created-at":"2026-10-25T10:00:00Z","dash0.com/updated-at":"2026-10-27T14:03:08.123456789Z","dash0.com/deleted-at":"2026-10-26T12:00:00Z"}
  },
  "spec":{
    "enabled":true,"signal":"logs","context":"log","priority":100,"display":{"name":"Redact checkout secrets"},
    "conditions":["severity_number > 9"],
    "transformations":[{"name":"redact","statements":["delete_key(attributes, \"password\")"]},{"name":"tag","statements":["set(attributes[\"team\"], \"payments\")"]}],
    "examples":[{"name":"checkout warning","otlp":{"resourceLogs":[{"scopeLogs":[{"logRecords":[{"timeUnixNano":"1793109788123456789","body":{"stringValue":"Payment accepted"},"attributes":[{"key":"order.id","value":{"intValue":"9007199254740993"}}]}]}]}]}}]
  }
}`
	kind, err := DetectKind([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, "Dash0TelemetryTransformationRule", kind)
	var rule dash0.TelemetryTransformationRuleDefinition
	require.NoError(t, Unmarshal([]byte(input), &rule))
	got, err := json.Marshal(rule)
	require.NoError(t, err)
	assert.JSONEq(t, want, string(got))
	encoded, err := sigsyaml.Marshal(rule)
	require.NoError(t, err)
	var roundTrip dash0.TelemetryTransformationRuleDefinition
	require.NoError(t, Unmarshal(encoded, &roundTrip))
	got, err = json.Marshal(roundTrip)
	require.NoError(t, err)
	assert.JSONEq(t, want, string(got))
}
