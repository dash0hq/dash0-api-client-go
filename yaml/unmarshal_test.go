package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal_KeepsYAML11BoolsAsStrings(t *testing.T) {
	in := []byte("items:\n  - x: 0\n    y: 0\n    flag: true\n    mode: off\n")

	var got map[string]any
	require.NoError(t, Unmarshal(in, &got))
	item := got["items"].([]any)[0].(map[string]any)

	assert.Equal(t, float64(0), item["y"])
	assert.Equal(t, "off", item["mode"])
	assert.Equal(t, true, item["flag"])
}

func TestToJSON_KeepsYAML11BoolsAsStrings(t *testing.T) {
	got, err := ToJSON([]byte("x: 0\ny: 0\n"))
	require.NoError(t, err)
	assert.JSONEq(t, `{"x":0,"y":0}`, string(got))
}

func TestQuoteYAML11Bools_UnchangedInputIsReturnedAsIs(t *testing.T) {
	for _, in := range []string{
		`{"y": 0}`,
		"x: 0\n",
		"not: [valid",
	} {
		assert.Equal(t, in, string(quoteYAML11Bools([]byte(in))))
	}
}

func TestParseAsDashboard_UnquotedYGridKey(t *testing.T) {
	dashboard, err := ParseAsDashboard([]byte(`kind: Dashboard
metadata:
  name: d
spec:
  layouts:
    - kind: Grid
      spec:
        items:
          - x: 0
            y: 3
`))
	require.NoError(t, err)
	items := dashboard.Spec["layouts"].([]any)[0].(map[string]any)["spec"].(map[string]any)["items"].([]any)
	assert.Equal(t, float64(3), items[0].(map[string]any)["y"])
}

func TestUnmarshalPrometheusRule_UnquotedYLabelKey(t *testing.T) {
	rule, err := UnmarshalPrometheusRule([]byte(`apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: r
spec:
  groups:
    - name: g
      rules:
        - alert: A
          expr: up == 0
          labels:
            y: a
`))
	require.NoError(t, err)
	require.NotNil(t, rule.Labels)
	assert.Equal(t, map[string]string{"y": "a"}, *rule.Labels)
}
