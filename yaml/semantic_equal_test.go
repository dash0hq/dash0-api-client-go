package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeYAML(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		additionalIgnored []string
		expected          string
		wantErr           bool
	}{
		{
			name: "removes metadata fields",
			input: `
apiVersion: v1
kind: Dash0SyntheticCheck
metadata:
  name: examplecom
  createdAt: "2024-01-01T00:00:00Z"
  updatedAt: "2024-01-02T00:00:00Z"
  version: 1
  dash0Extensions:
    something: value
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://www.example.com
`,
			expected: `metadata:
  name: examplecom
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://www.example.com`,
			wantErr: false,
		},
		{
			name: "handles missing metadata fields",
			input: `
kind: Dash0SyntheticCheck
metadata:
  name: test
spec:
  enabled: false
`,
			expected: `metadata:
  name: test
spec:
  enabled: false`,
			wantErr: false,
		},
		{
			name: "handles complex structure",
			input: `
apiVersion: v1
kind: Dash0SyntheticCheck
metadata:
  name: complex
  createdAt: "2024-01-01T00:00:00Z"
  updatedAt: "2024-01-02T00:00:00Z"
  version: 2
spec:
  enabled: true
  notifications:
    channels:
      - id: channel1
      - id: channel2
  plugin:
    display:
      name: example.com
    kind: http
    spec:
      assertions:
        criticalAssertions:
          - kind: status_code
            spec:
              value: "200"
              operator: is
      request:
        method: get
        url: https://www.example.com
        headers:
          - key: User-Agent
            value: Mozilla/5.0
  retries:
    kind: fixed
    spec:
      attempts: 3
      delay: 1s
  schedule:
    interval: 1m
    locations:
      - gcp-europe-west3
`,
			expected: `metadata:
  name: complex
spec:
  enabled: true
  notifications:
    channels:
    - id: channel1
    - id: channel2
  plugin:
    display:
      name: example.com
    kind: http
    spec:
      assertions:
        criticalAssertions:
        - kind: status_code
          spec:
            operator: is
            value: "200"
      request:
        headers:
        - key: User-Agent
          value: Mozilla/5.0
        method: get
        url: https://www.example.com
  retries:
    kind: fixed
    spec:
      attempts: 3
      delay: 1s
  schedule:
    interval: 1m
    locations:
    - gcp-europe-west3`,
			wantErr: false,
		},
		{
			name: "removes empty arrays and empty maps",
			input: `
kind: Dash0View
metadata:
  name: test
  annotations: {}
spec:
  display:
    name: Test View
    folder: []
  type: spans
`,
			expected: `metadata:
  name: test
spec:
  display:
    name: Test View
  type: spans`,
			wantErr: false,
		},
		{
			name:  "removes null values from JSON API responses",
			input: `{"kind":"Dash0SyntheticCheck","metadata":{"name":"test"},"spec":{"enabled":true,"notifications":{"channels":null},"plugin":{"kind":"http","spec":{"request":{"url":"https://example.com"}}}}}`,
			expected: `metadata:
  name: test
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://example.com`,
			wantErr: false,
		},
		{
			name: "removes spec.permissions when passed as additional ignored field",
			input: `
kind: Dash0SyntheticCheck
metadata:
  name: test
spec:
  enabled: true
  permissions:
    - actions:
        - "synthetic_check:read"
        - "synthetic_check:delete"
      role: admin
    - actions:
        - "synthetic_check:read"
      role: basic_member
  plugin:
    kind: http
    spec:
      request:
        url: https://www.example.com
`,
			additionalIgnored: []string{"spec.permissions"},
			expected: `metadata:
  name: test
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://www.example.com`,
			wantErr: false,
		},
		{
			name: "preserves spec.permissions when not in additional ignored fields",
			input: `
kind: Dash0SyntheticCheck
metadata:
  name: test
spec:
  enabled: true
  permissions:
    - actions:
        - "synthetic_check:read"
      role: admin
  plugin:
    kind: http
    spec:
      request:
        url: https://www.example.com
`,
			expected: `metadata:
  name: test
spec:
  enabled: true
  permissions:
  - actions:
    - synthetic_check:read
    role: admin
  plugin:
    kind: http
    spec:
      request:
        url: https://www.example.com`,
			wantErr: false,
		},
		{
			name:     "handles invalid YAML",
			input:    "invalid: : : yaml",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Normalize([]byte(tt.input), tt.additionalIgnored, nil)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, string(result))
			}
		})
	}
}

func TestEquivalent(t *testing.T) {
	tests := []struct {
		name              string
		yaml1             string
		yaml2             string
		additionalIgnored []string
		equivalent        bool
		wantErr           bool
	}{
		{
			name: "identical checks",
			yaml1: `
kind: Dash0SyntheticCheck
metadata:
  name: test
spec:
  enabled: true
`,
			yaml2: `
kind: Dash0SyntheticCheck
metadata:
  name: test
spec:
  enabled: true
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "equivalent checks with different metadata",
			yaml1: `
apiVersion: v1
kind: Dash0SyntheticCheck
metadata:
  name: test
  createdAt: "2024-01-01T00:00:00Z"
  updatedAt: "2024-01-01T00:00:00Z"
  version: 1
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://www.example.com
`,
			yaml2: `
apiVersion: v2
kind: SomeOtherKind
metadata:
  name: test
  createdAt: "2024-02-02T00:00:00Z"
  updatedAt: "2024-02-02T00:00:00Z"
  version: 2
  dash0Extensions:
    extra: field
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://www.example.com
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "different checks",
			yaml1: `
metadata:
  name: check1
spec:
  enabled: true
`,
			yaml2: `
metadata:
  name: check2
spec:
  enabled: false
`,
			equivalent: false,
			wantErr:    false,
		},
		{
			name: "different spec content",
			yaml1: `
metadata:
  name: test
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://www.example.com
`,
			yaml2: `
metadata:
  name: test
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://www.different.com
`,
			equivalent: false,
			wantErr:    false,
		},
		{
			name: "different only in metadata.name",
			yaml1: `
metadata:
  name: old-name
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://www.example.com
`,
			yaml2: `
metadata:
  name: new-name
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://www.example.com
`,
			equivalent: false,
			wantErr:    false,
		},
		{
			name: "equivalent with different order",
			yaml1: `
metadata:
  name: test
spec:
  schedule:
    interval: 1m
    locations:
      - gcp-us-west1
      - gcp-europe-west3
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://www.example.com
        method: get
`,
			yaml2: `
metadata:
  name: test
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        method: get
        url: https://www.example.com
  schedule:
    locations:
      - gcp-us-west1
      - gcp-europe-west3
    interval: 1m
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name:       "invalid YAML in first",
			yaml1:      "invalid: : : yaml",
			yaml2:      "metadata:\n  name: test",
			equivalent: false,
			wantErr:    true,
		},
		{
			name:       "invalid YAML in second",
			yaml1:      "metadata:\n  name: test",
			yaml2:      "invalid: : : yaml",
			equivalent: false,
			wantErr:    true,
		},
		{
			name: "ignore different order in slices",
			yaml1: `
kind: Dash0SyntheticCheck
spec:
  notifications:
    channels:
      - id: channel-a
        type: email
      - id: channel-b
        type: slack
`,
			yaml2: `
kind: Dash0SyntheticCheck
spec:
  notifications:
    channels:
      - id: channel-b
        type: slack
      - id: channel-a
        type: email
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "equivalent with different annotation ordering and quoting styles",
			yaml1: `
spec:
  groups:
    - interval: 1m0s
      name: test-group
      rules:
        - alert: test-alert
          annotations:
            summary: "{{ $labels.reason }} event detected"
            description: "Events exceeded threshold"
            dash0-threshold-critical: "0"
            dash0-threshold-degraded: "0"
          labels:
            severity: critical
            team: "{{ $labels.team_name }}"
`,
			yaml2: `
spec:
  groups:
    - interval: 1m0s
      name: test-group
      rules:
        - alert: test-alert
          annotations:
            dash0-threshold-critical: "0"
            dash0-threshold-degraded: "0"
            description: Events exceeded threshold
            summary: '{{ $labels.reason }} event detected'
          labels:
            severity: "critical"
            team: '{{ $labels.team_name }}'
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "equivalent when JSON has null values and YAML omits them",
			yaml1: `
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://example.com
`,
			yaml2:      `{"spec":{"enabled":true,"notifications":{"channels":null},"plugin":{"kind":"http","spec":{"request":{"url":"https://example.com"}}}}}`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "equivalent when one has empty arrays and other omits them",
			yaml1: `
kind: Dash0View
metadata:
  name: test
  annotations: {}
spec:
  display:
    name: Test View
    folder: []
  type: spans
`,
			yaml2: `
kind: Dash0View
metadata:
  name: test
spec:
  display:
    name: Test View
  type: spans
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "equivalent when one has zero threshold annotations and other omits them",
			yaml1: `
spec:
  groups:
    - rules:
        - annotations:
            summary: Test
            dash0-threshold-critical: "0"
            dash0-threshold-degraded: "0"
`,
			yaml2: `
spec:
  groups:
    - rules:
        - annotations:
            summary: Test
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "NOT equivalent when threshold values differ",
			yaml1: `
spec:
  groups:
    - rules:
        - annotations:
            dash0-threshold-critical: "50"
`,
			yaml2: `
spec:
  groups:
    - rules:
        - annotations:
            dash0-threshold-critical: "0"
`,
			equivalent: false,
			wantErr:    false,
		},
		{
			name: "equivalent when both have same non-zero threshold annotations",
			yaml1: `
spec:
  groups:
    - rules:
        - annotations:
            summary: Test
            dash0-threshold-critical: "50"
            dash0-threshold-degraded: "30"
`,
			yaml2: `
spec:
  groups:
    - rules:
        - annotations:
            summary: Test
            dash0-threshold-critical: "50"
            dash0-threshold-degraded: "30"
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "NOT equivalent when one has non-zero threshold and other omits it",
			yaml1: `
spec:
  groups:
    - rules:
        - annotations:
            summary: Test
            dash0-threshold-critical: "50"
`,
			yaml2: `
spec:
  groups:
    - rules:
        - annotations:
            summary: Test
`,
			equivalent: false,
			wantErr:    false,
		},
		{
			name: "equivalent when durations use different formats (2m vs 2m0s)",
			yaml1: `
spec:
  groups:
    - interval: 2m
      name: test-group
      rules:
        - alert: test-alert
          for: 5m
          keep_firing_for: 10s
          expr: test > 0
`,
			yaml2: `
spec:
  groups:
    - interval: 2m0s
      name: test-group
      rules:
        - alert: test-alert
          for: 5m0s
          keep_firing_for: 10s
          expr: test > 0
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "equivalent with complex duration formats (1h30m vs 1h30m0s)",
			yaml1: `
spec:
  groups:
    - interval: 1h30m
      name: test-group
      rules:
        - alert: test-alert
          for: 1h
          expr: test > 0
`,
			yaml2: `
spec:
  groups:
    - interval: 1h30m0s
      name: test-group
      rules:
        - alert: test-alert
          for: 1h0m0s
          expr: test > 0
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "NOT equivalent when durations actually differ",
			yaml1: `
spec:
  groups:
    - rules:
        - for: 2m
`,
			yaml2: `
spec:
  groups:
    - rules:
        - for: 3m
`,
			equivalent: false,
			wantErr:    false,
		},
		{
			name: "equivalent when integers and floats represent the same value",
			yaml1: `
spec:
  retries:
    spec:
      attempts: 3
  schedule:
    interval: 60
`,
			yaml2: `
spec:
  retries:
    spec:
      attempts: 3.0
  schedule:
    interval: 60.0
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "NOT equivalent when numeric values actually differ",
			yaml1: `
spec:
  retries:
    spec:
      attempts: 3
`,
			yaml2: `
spec:
  retries:
    spec:
      attempts: 4
`,
			equivalent: false,
			wantErr:    false,
		},
		{
			name: "equivalent when one has dash0-enabled true and other omits it",
			yaml1: `
spec:
  groups:
    - rules:
        - annotations:
            summary: Test
            dash0-enabled: "true"
`,
			yaml2: `
spec:
  groups:
    - rules:
        - annotations:
            summary: Test
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "equivalent when annotation has unquoted number vs quoted string",
			yaml1: `
spec:
  groups:
    - rules:
        - annotations:
            summary: Test
            dash0-threshold-critical: 5000
            dash0-threshold-degraded: "1000"
`,
			yaml2: `
spec:
  groups:
    - rules:
        - annotations:
            summary: Test
            dash0-threshold-critical: "5000"
            dash0-threshold-degraded: "1000"
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "NOT equivalent when dash0-enabled is false vs absent",
			yaml1: `
spec:
  groups:
    - rules:
        - annotations:
            summary: Test
            dash0-enabled: "false"
`,
			yaml2: `
spec:
  groups:
    - rules:
        - annotations:
            summary: Test
`,
			equivalent: false,
			wantErr:    false,
		},
		{
			name: "equivalent when label has unquoted number vs quoted string",
			yaml1: `
spec:
  groups:
    - rules:
        - labels:
            severity: critical
            port: 8080
`,
			yaml2: `
spec:
  groups:
    - rules:
        - labels:
            severity: critical
            port: "8080"
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "equivalent when label has unquoted boolean vs quoted string",
			yaml1: `
spec:
  groups:
    - rules:
        - labels:
            severity: critical
            enabled: true
`,
			yaml2: `
spec:
  groups:
    - rules:
        - labels:
            severity: critical
            enabled: "true"
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "equivalent when for is 0s vs absent",
			yaml1: `
spec:
  groups:
    - rules:
        - alert: test
          for: 0s
          expr: test > 0
`,
			yaml2: `
spec:
  groups:
    - rules:
        - alert: test
          expr: test > 0
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "NOT equivalent when for is non-zero vs absent",
			yaml1: `
spec:
  groups:
    - rules:
        - alert: test
          for: 30s
          expr: test > 0
`,
			yaml2: `
spec:
  groups:
    - rules:
        - alert: test
          expr: test > 0
`,
			equivalent: false,
			wantErr:    false,
		},
		{
			name: "equivalent when keep_firing_for is 0s vs absent",
			yaml1: `
spec:
  groups:
    - rules:
        - alert: test
          for: 5m
          keep_firing_for: 0s
          expr: test > 0
`,
			yaml2: `
spec:
  groups:
    - rules:
        - alert: test
          for: 5m
          expr: test > 0
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "NOT equivalent when keep_firing_for is non-zero vs absent",
			yaml1: `
spec:
  groups:
    - rules:
        - alert: test
          for: 5m
          keep_firing_for: 30s
          expr: test > 0
`,
			yaml2: `
spec:
  groups:
    - rules:
        - alert: test
          for: 5m
          expr: test > 0
`,
			equivalent: false,
			wantErr:    false,
		},
		{
			name: "equivalent when one has spec.permissions and other omits it (conditionally ignored)",
			yaml1: `
spec:
  enabled: true
  permissions:
    - actions:
        - "synthetic_check:read"
        - "synthetic_check:delete"
      role: admin
    - actions:
        - "synthetic_check:read"
      role: basic_member
`,
			yaml2: `
spec:
  enabled: true
`,
			additionalIgnored: []string{"spec.permissions"},
			equivalent:        true,
			wantErr:           false,
		},
		{
			name: "equivalent when API adds spec.permissions and metadata (conditionally ignored)",
			yaml1: `
kind: Dash0SyntheticCheck
metadata:
  name: test-check
  description: test check
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://example.com
  schedule:
    interval: 5m
    locations:
      - de-frankfurt
    strategy: all_locations
`,
			yaml2: `
kind: Dash0SyntheticCheck
metadata:
  name: test-check
  description: test check
  annotations: {}
  labels:
    dash0.com/dataset: test-dataset
    dash0.com/id: some-uuid
    dash0.com/origin: tf_some-origin
    dash0.com/version: "1"
spec:
  enabled: true
  permissions:
    - actions:
        - "synthetic_check:read"
        - "synthetic_check:delete"
      role: admin
    - actions:
        - "synthetic_check:read"
      role: basic_member
  plugin:
    kind: http
    spec:
      request:
        url: https://example.com
  schedule:
    interval: 5m
    locations:
      - de-frankfurt
    strategy: all_locations
`,
			additionalIgnored: []string{"spec.permissions"},
			equivalent:        true,
			wantErr:           false,
		},
		{
			name: "NOT equivalent when both have different spec.permissions (drift detection)",
			yaml1: `
spec:
  enabled: true
  permissions:
    - actions:
        - "synthetic_check:read"
      role: admin
`,
			yaml2: `
spec:
  enabled: true
  permissions:
    - actions:
        - "synthetic_check:read"
        - "synthetic_check:delete"
      role: admin
`,
			equivalent: false,
			wantErr:    false,
		},
		{
			name: "equivalent when both have same spec.permissions (no drift)",
			yaml1: `
spec:
  enabled: true
  permissions:
    - actions:
        - "synthetic_check:read"
        - "synthetic_check:delete"
      role: admin
`,
			yaml2: `
spec:
  enabled: true
  permissions:
    - actions:
        - "synthetic_check:read"
        - "synthetic_check:delete"
      role: admin
`,
			equivalent: true,
			wantErr:    false,
		},
		{
			name: "NOT equivalent when one has spec.permissions and other omits it (without conditional ignore)",
			yaml1: `
spec:
  enabled: true
  permissions:
    - actions:
        - "synthetic_check:read"
      role: admin
`,
			yaml2: `
spec:
  enabled: true
`,
			equivalent: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Equivalent([]byte(tt.yaml1), []byte(tt.yaml2), tt.additionalIgnored, nil)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.equivalent, result)
			}
		})
	}
}

func TestRemoveYAMLField(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		path     string
		expected map[string]any
	}{
		{
			name: "remove top-level field",
			input: map[string]any{
				"apiVersion": "v1",
				"kind":       "Dash0SyntheticCheck",
				"metadata":   map[string]any{"name": "test"},
			},
			path: "apiVersion",
			expected: map[string]any{
				"kind":     "Dash0SyntheticCheck",
				"metadata": map[string]any{"name": "test"},
			},
		},
		{
			name: "remove nested field",
			input: map[string]any{
				"metadata": map[string]any{
					"name":      "test",
					"createdAt": "2024-01-01",
					"updatedAt": "2024-01-02",
				},
			},
			path: "metadata.createdAt",
			expected: map[string]any{
				"metadata": map[string]any{
					"name":      "test",
					"updatedAt": "2024-01-02",
				},
			},
		},
		{
			name: "path doesn't exist",
			input: map[string]any{
				"metadata": map[string]any{
					"name": "test",
				},
			},
			path: "metadata.nonexistent",
			expected: map[string]any{
				"metadata": map[string]any{
					"name": "test",
				},
			},
		},
		{
			name: "intermediate path doesn't exist",
			input: map[string]any{
				"spec": map[string]any{
					"enabled": true,
				},
			},
			path: "metadata.createdAt",
			expected: map[string]any{
				"spec": map[string]any{
					"enabled": true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupMap(tt.input, []string{tt.path})
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestCanonicalString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "string value",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "int value",
			input:    42,
			expected: "42",
		},
		{
			name:     "bool value",
			input:    true,
			expected: "true",
		},
		{
			name:     "map with sorted keys",
			input:    map[string]any{"b": "2", "a": "1"},
			expected: "{a:1,b:2}",
		},
		{
			name:     "slice elements are sorted",
			input:    []any{"cherry", "apple", "banana"},
			expected: "[apple,banana,cherry]",
		},
		{
			name: "nested map with unsorted inner list produces same canonical form regardless of list order",
			input: map[string]any{
				"role":    "admin",
				"actions": []any{"views:read", "views:delete"},
			},
			expected: "{actions:[views:delete,views:read],role:admin}",
		},
		{
			name: "same nested map with different inner list order produces identical canonical form",
			input: map[string]any{
				"role":    "admin",
				"actions": []any{"views:delete", "views:read"},
			},
			expected: "{actions:[views:delete,views:read],role:admin}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canonicalString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEquivalent_PermissionsWithReorderedActions(t *testing.T) {
	// This test verifies that permissions with actions in different order are
	// treated as equivalent, and that the outer permissions sort is stable even
	// when inner action lists differ in order.
	yaml1 := `
spec:
  permissions:
    - actions:
        - "views:read"
        - "views:delete"
      role: admin
    - actions:
        - "views:read"
      role: basic_member
`
	yaml2 := `
spec:
  permissions:
    - actions:
        - "views:read"
      role: basic_member
    - actions:
        - "views:delete"
        - "views:read"
      role: admin
`
	result, err := Equivalent([]byte(yaml1), []byte(yaml2), nil, nil)
	require.NoError(t, err)
	assert.True(t, result, "permissions with reordered actions and reordered entries should be equivalent")
}

func TestNormalizeNumericTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name:     "converts int to float64",
			input:    int(100),
			expected: float64(100),
		},
		{
			name:     "converts int32 to float64",
			input:    int32(100),
			expected: float64(100),
		},
		{
			name:     "converts int64 to float64",
			input:    int64(100),
			expected: float64(100),
		},
		{
			name:     "converts float32 to float64",
			input:    float32(100.5),
			expected: float64(100.5),
		},
		{
			name:     "keeps float64 as is",
			input:    float64(100.5),
			expected: float64(100.5),
		},
		{
			name:     "keeps string as is",
			input:    "hello",
			expected: "hello",
		},
		{
			name: "converts various numeric types in nested map",
			input: map[string]any{
				"count":   int(3),
				"count32": int32(4),
				"count64": int64(5),
				"name":    "test",
				"nested": map[string]any{
					"value":   int(42),
					"float32": float32(1.5),
					"float64": float64(2.5),
				},
			},
			expected: map[string]any{
				"count":   float64(3),
				"count32": float64(4),
				"count64": float64(5),
				"name":    "test",
				"nested": map[string]any{
					"value":   float64(42),
					"float32": float64(1.5),
					"float64": float64(2.5),
				},
			},
		},
		{
			name:     "converts various numeric types in slices",
			input:    []any{int(1), int32(2), int64(3), float32(4.5), "five"},
			expected: []any{float64(1), float64(2), float64(3), float64(4.5), "five"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeNumericTypes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasFieldPath(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		path     string
		expected bool
	}{
		{
			name:     "top-level field exists",
			data:     map[string]any{"enabled": true},
			path:     "enabled",
			expected: true,
		},
		{
			name:     "top-level field missing",
			data:     map[string]any{"enabled": true},
			path:     "permissions",
			expected: false,
		},
		{
			name: "nested field exists",
			data: map[string]any{
				"spec": map[string]any{
					"permissions": []any{"read"},
				},
			},
			path:     "spec.permissions",
			expected: true,
		},
		{
			name: "nested field missing",
			data: map[string]any{
				"spec": map[string]any{
					"enabled": true,
				},
			},
			path:     "spec.permissions",
			expected: false,
		},
		{
			name: "intermediate path missing",
			data: map[string]any{
				"metadata": map[string]any{"name": "test"},
			},
			path:     "spec.permissions",
			expected: false,
		},
		{
			name: "nil value treated as absent",
			data: map[string]any{
				"spec": map[string]any{
					"permissions": nil,
				},
			},
			path:     "spec.permissions",
			expected: false,
		},
		{
			name:     "empty map",
			data:     map[string]any{},
			path:     "spec.permissions",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasFieldPath(tt.data, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripAbsentZeroValues(t *testing.T) {
	tests := []struct {
		name       string
		userYAML   string
		apiJSON    string
		equivalent bool
	}{
		{
			name: "API has enabled false that user did not set - should be equivalent",
			userYAML: `
spec:
  plugin:
    kind: http
    spec:
      request:
        url: https://example.com
`,
			apiJSON:    `{"spec":{"enabled":false,"plugin":{"kind":"http","spec":{"request":{"url":"https://example.com"}}}}}`,
			equivalent: true,
		},
		{
			name: "API has retries null that user did not set - should be equivalent",
			userYAML: `
spec:
  plugin:
    kind: http
    spec:
      request:
        url: https://example.com
`,
			apiJSON:    `{"spec":{"plugin":{"kind":"http","spec":{"request":{"url":"https://example.com"}}},"retries":null}}`,
			equivalent: true,
		},
		{
			name: "API has nested zero-value struct that user did not set - should be equivalent",
			userYAML: `
spec:
  plugin:
    kind: http
    spec:
      request:
        url: https://example.com
`,
			apiJSON:    `{"spec":{"plugin":{"kind":"http","spec":{"request":{"url":"https://example.com"}}},"notifications":{"channels":null}}}`,
			equivalent: true,
		},
		{
			name: "user explicitly set enabled false - should preserve (both have it)",
			userYAML: `
spec:
  enabled: false
  plugin:
    kind: http
    spec:
      request:
        url: https://example.com
`,
			apiJSON:    `{"spec":{"enabled":false,"plugin":{"kind":"http","spec":{"request":{"url":"https://example.com"}}}}}`,
			equivalent: true,
		},
		{
			name: "user set enabled true but API returns enabled false - real diff",
			userYAML: `
spec:
  enabled: true
  plugin:
    kind: http
    spec:
      request:
        url: https://example.com
`,
			apiJSON:    `{"spec":{"enabled":false,"plugin":{"kind":"http","spec":{"request":{"url":"https://example.com"}}}}}`,
			equivalent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Equivalent([]byte(tt.userYAML), []byte(tt.apiJSON), nil, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.equivalent, result)
		})
	}
}

func TestAbsentFields(t *testing.T) {
	tests := []struct {
		name     string
		yamlStr  string
		fields   []string
		expected []string
	}{
		{
			name: "field present - not returned",
			yamlStr: `
spec:
  permissions:
    - role: admin
`,
			fields:   []string{"spec.permissions"},
			expected: nil,
		},
		{
			name: "field absent - returned",
			yamlStr: `
spec:
  enabled: true
`,
			fields:   []string{"spec.permissions"},
			expected: []string{"spec.permissions"},
		},
		{
			name: "mix of present and absent fields",
			yamlStr: `
spec:
  enabled: true
  permissions:
    - role: admin
`,
			fields:   []string{"spec.permissions", "spec.notifications"},
			expected: []string{"spec.notifications"},
		},
		{
			name:     "invalid YAML returns nil (safe default)",
			yamlStr:  "invalid: : : yaml",
			fields:   []string{"spec.permissions"},
			expected: nil,
		},
		{
			name: "all fields present - returns nil",
			yamlStr: `
spec:
  permissions:
    - role: admin
  notifications:
    channels: []
`,
			fields:   []string{"spec.permissions", "spec.notifications"},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AbsentFields([]byte(tt.yamlStr), tt.fields)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// The following tests cover WithAnnotationsRoot, added so this engine also
// correctly handles a flat document shape -- one that carries
// "annotations"/"labels" at the document root instead of nested under
// "metadata" -- which dash0-cli's native (non-CRD) CheckRule kind uses.
// Every test above this point exercises the default root ("metadata"),
// matching terraform-provider-dash0's shape exactly; these exercise root ""
// (flat) explicitly, so a regression in either root can't hide behind the
// other's coverage.

func TestNormalize_FlatAnnotationsRoot(t *testing.T) {
	input := `
id: rule-1
name: test-rule
expression: up == 0
labels:
  dash0.com/origin: dash0-cli
  team: platform
annotations:
  summary: Test summary
  sharing: team:team_01abc
  dash0-threshold-critical: "0"
`
	result, err := Normalize([]byte(input), nil, []string{"dash0.com/sharing", "sharing"}, WithAnnotationsRoot(""))
	require.NoError(t, err)
	assert.Equal(t, `annotations:
  sharing: team:team_01abc
expression: up == 0
id: rule-1
labels:
  dash0.com/origin: dash0-cli
  team: platform
name: test-rule`, string(result))
}

func TestNormalize_FlatAnnotationsRoot_DefaultRootLeavesFlatDocumentUntouched(t *testing.T) {
	// Regression guard for the gap this option exists to close: without
	// WithAnnotationsRoot(""), a flat document's top-level "annotations" is
	// invisible to the metadata-nested annotation-preservation step (there
	// is no "metadata" key to find), so *all* annotations survive
	// regardless of preservedAnnotationKeys -- correct for the generic
	// per-key default/empty stripping cleanupMap already does, but it means
	// preservedAnnotationKeys silently does nothing without the root option.
	input := `
id: rule-1
name: test-rule
annotations:
  summary: Test summary
  not-preserved: should-normally-be-stripped
`
	result, err := Normalize([]byte(input), nil, []string{"dash0.com/sharing"} /* no WithAnnotationsRoot */)
	require.NoError(t, err)
	assert.Contains(t, string(result), "not-preserved: should-normally-be-stripped",
		"documents this option's necessity: default root leaves a flat document's annotations unfiltered")
}

func TestEquivalent_FlatAnnotationsRoot_PreservedKeySurvivesComparison(t *testing.T) {
	a := `
id: rule-1
name: test-rule
expression: up == 0
annotations:
  sharing: team:team_01abc
`
	b := `
id: rule-1
name: test-rule
expression: up == 0
annotations:
  sharing: team:team_02xyz
`
	equivalent, err := Equivalent([]byte(a), []byte(b), nil, []string{"sharing"}, WithAnnotationsRoot(""))
	require.NoError(t, err)
	assert.False(t, equivalent, "a preserved annotation key that actually changed must be reported as a difference")
}

func TestEquivalent_FlatAnnotationsRoot_NonPreservedAnnotationIgnored(t *testing.T) {
	a := `
id: rule-1
name: test-rule
expression: up == 0
annotations:
  server-added: value-a
`
	b := `
id: rule-1
name: test-rule
expression: up == 0
annotations:
  server-added: value-b
`
	equivalent, err := Equivalent([]byte(a), []byte(b), nil, []string{"sharing"}, WithAnnotationsRoot(""))
	require.NoError(t, err)
	assert.True(t, equivalent, "a non-preserved annotation must be stripped before comparison, even at the flat root")
}

func TestEquivalent_FlatAnnotationsRoot_CustomLabelStillCompared(t *testing.T) {
	// Custom, user-set labels (e.g. "team: platform" alongside the
	// server-managed "dash0.com/origin") must remain part of the
	// comparison -- WithAnnotationsRoot("") must not cause defaultIgnoredFields
	// ("labels" at the flat root) to swallow the whole labels map.
	a := `
id: rule-1
name: test-rule
expression: up == 0
labels:
  team: platform
`
	b := `
id: rule-1
name: test-rule
expression: up == 0
labels:
  team: backend
`
	equivalent, err := Equivalent([]byte(a), []byte(b), nil, nil, WithAnnotationsRoot(""))
	require.NoError(t, err)
	assert.False(t, equivalent, "a real label change must not be silently ignored under the flat root")
}
