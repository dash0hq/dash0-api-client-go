package yaml

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"encoding/json"

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

func TestUnmarshalStrict_Kinds(t *testing.T) {
	tests := []struct {
		name string
		// get is the type the API returns, decode the type callers decode into.
		get, decode reflect.Type
		// parent is where the test adds an unknown field.
		parent string
	}{
		{"dashboard", typeOf[dash0.DashboardDefinition](), typeOf[dash0.DashboardDefinition](), "metadata.dash0Extensions"},
		{"PersesDashboard", typeOf[dash0.PersesDashboard](), typeOf[dash0.PersesDashboard](), "metadata"},
		{"check rule", typeOf[dash0.PrometheusAlertRule](), typeOf[dash0.PrometheusAlertRule](), "thresholds"},
		{"PrometheusRule", typeOf[dash0.RecordingRule](), typeOf[dash0.RecordingRule](), "spec.groups[0].rules[0]"},
		{"synthetic check", typeOf[dash0.SyntheticCheckDefinition](), typeOf[dash0.SyntheticCheckDefinition](), "spec.plugin.spec.request"},
		{"SLO", typeOf[dash0.SloDefinition](), typeOf[dash0.SloDefinition](), "spec"},
		{"view", typeOf[dash0.ViewDefinition](), typeOf[dash0.ViewDefinition](), "spec"},
		{"recording rule", typeOf[dash0.RecordingRule](), typeOf[dash0.RecordingRule](), "spec.groups[0]"},
		{"notification channel", typeOf[dash0.NotificationChannelDefinition](), typeOf[dash0.NotificationChannelDefinition](), "spec.config"},
		{"spam filter v1alpha1", typeOf[dash0.SpamFilter](), typeOf[dash0.SpamFilter](), "spec"},
		{"spam filter v1alpha2", typeOf[dash0.SpamFilterV1Alpha2](), typeOf[dash0.SpamFilterV1Alpha2](), "spec"},
		{"team", typeOf[dash0.TeamDefinitionV1Alpha1](), typeOf[dash0.TeamDefinitionV1Alpha1](), "spec.display"},
		{"time series aggregation", typeOf[dash0.TimeSeriesAggregationDefinition](), typeOf[dash0.TimeSeriesAggregationDefinition](), "spec"},
		{"dataset", typeOf[dash0.DatasetResponse](), typeOf[dash0.DatasetDefinition](), "spec"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := fullDocument(tt.get)
			data, err := json.Marshal(doc)
			require.NoError(t, err)
			require.NoError(t, UnmarshalStrict(data, reflect.New(tt.decode).Interface()), "export must decode")

			parent := doc
			for _, key := range strings.Split(strings.ReplaceAll(tt.parent, "[0]", ".0"), ".") {
				if key == "0" {
					parent = parent.([]any)[0]
				} else {
					parent = parent.(map[string]any)[key]
				}
			}
			parent.(map[string]any)["typo"] = "x"
			data, err = json.Marshal(doc)
			require.NoError(t, err)
			err = UnmarshalStrict(data, reflect.New(tt.decode).Interface())
			assert.EqualError(t, err, `unknown field "`+tt.parent+`.typo"`)
		})
	}
}

// An auth token export carries the server-generated spec.token, which the
// create and update type has no place for.
func TestUnmarshalStrict_AuthTokenExport(t *testing.T) {
	data, err := json.Marshal(fullDocument(typeOf[dash0.AuthTokenResponseDefinition]()))
	require.NoError(t, err)
	assert.NoError(t, UnmarshalStrict(data, &dash0.AuthTokenResponseDefinition{}))
	assert.EqualError(t, UnmarshalStrict(data, &dash0.AuthTokenDefinition{}), `unknown field "spec.token"`)
}

func TestUnmarshalStrict(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		v       any
		wantErr string
	}{
		{
			name:    "typo in auth token scope",
			in:      "kind: Dash0AuthToken\nmetadata: {name: t}\nspec:\n  scope:\n    datasets:\n      restrction: restricted\n",
			v:       &dash0.AuthTokenDefinition{},
			wantErr: `unknown field "spec.scope.datasets.restrction"`,
		},
		{
			name:    "JSON input, every unknown field",
			in:      `{"kind": "Dashboard", "metadata": {"name": "d", "nmae": "d"}, "sepc": {}}`,
			v:       &dash0.DashboardDefinition{},
			wantErr: `unknown fields "metadata.nmae", "sepc"`,
		},
		{
			name:    "list item",
			in:      "spec:\n  groups:\n    - name: g\n      rules:\n        - expr: up\n          alrt: A\n",
			v:       &dash0.RecordingRule{},
			wantErr: `unknown field "spec.groups[0].rules[0].alrt"`,
		},
		{
			name: "dashboard spec, labels, and annotations are free-form",
			in:   "kind: Dashboard\nmetadata:\n  name: d\n  labels: {app: web}\n  annotations: {owner: sre, dash0.com/folder-path: /a}\nspec:\n  anything: {y: 1}\n",
			v:    &dash0.DashboardDefinition{},
		},
		{
			name: "AdditionalProperties is free-form",
			in:   "name: r\nexpression: up\nannotations: {summary: s, runbook: url}\nlabels: {team: a}\n",
			v:    &dash0.PrometheusAlertRule{},
		},
		{
			name: "free-form map values keep their type",
			in:   "metadata:\n  annotations:\n    dash0.com/folder-path: 1\n",
			v:    &dash0.DashboardDefinition{},
		},
		{
			name:    "union matches no variant",
			in:      "spec:\n  type: slack\n  config: {url: https://example.com, chanel: x}\n",
			v:       &dash0.NotificationChannelDefinition{},
			wantErr: `unknown field "spec.config.chanel"`,
		},
		{
			name: "key differs only in case, as encoding/json accepts",
			in:   "Kind: Dashboard\nMetadata: {Name: d}\n",
			v:    &dash0.DashboardDefinition{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnmarshalStrict([]byte(tt.in), tt.v)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestUnmarshalStrict_DecodesLikeUnmarshal(t *testing.T) {
	in := []byte("kind: Dashboard\nmetadata: {name: d}\nspec:\n  layouts: [{x: 0, y: 3}]\n")
	var want, got dash0.DashboardDefinition
	require.NoError(t, Unmarshal(in, &want))
	require.NoError(t, UnmarshalStrict(in, &got))
	assert.Equal(t, want, got)
}

func TestUnmarshalStrict_InvalidYAML(t *testing.T) {
	assert.Error(t, UnmarshalStrict([]byte("not: [valid"), &dash0.DashboardDefinition{}))
}

func typeOf[T any]() reflect.Type { return reflect.TypeFor[T]() }

// fullDocument returns a document of type t with every field set, as a full
// API response would have it.
// A union gets its first variant.
func fullDocument(t reflect.Type) any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch {
	case t == reflect.TypeFor[time.Time]():
		return "2026-01-02T03:04:05Z"
	case t.Kind() == reflect.Struct && t.NumField() == 1 && t.Field(0).Name == "union":
		for i := range t.NumMethod() {
			if m := t.Method(i); strings.HasPrefix(m.Name, "As") {
				return fullDocument(m.Type.Out(0))
			}
		}
	case t.Kind() == reflect.Struct:
		doc := map[string]any{}
		for i := range t.NumField() {
			f := t.Field(i)
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if f.IsExported() && name != "-" {
				doc[name] = fullDocument(f.Type)
			}
		}
		return doc
	case t.Kind() == reflect.Map:
		return map[string]any{"key": fullDocument(t.Elem())}
	case t.Kind() == reflect.Array: // openapi_types.UUID
		return "00000000-0000-0000-0000-000000000000"
	case t.Kind() == reflect.Slice:
		return []any{fullDocument(t.Elem())}
	case t.Kind() == reflect.Bool:
		return true
	case t.Kind() == reflect.String, t.Kind() == reflect.Interface:
		return "x"
	}
	return 1
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
