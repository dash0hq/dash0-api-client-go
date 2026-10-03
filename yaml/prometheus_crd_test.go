package yaml

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dash0 "github.com/dash0hq/dash0-api-client-go"
)

func assertEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

func assertPtrEqual[T comparable](t *testing.T, field string, got *T, want T) {
	t.Helper()
	if got == nil {
		t.Errorf("%s is nil, want %v", field, want)
		return
	}
	if *got != want {
		t.Errorf("%s = %v, want %v", field, *got, want)
	}
}

func TestParseAsPrometheusAlertRules_NativeCheckRule(t *testing.T) {
	data := []byte(`name: My Check Rule
expression: sum(rate(errors[5m])) > 0.1
`)
	rules, err := ParseAsPrometheusAlertRules(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}
	assertEqual(t, "Name", rules[0].Name, "My Check Rule")
	assertEqual(t, "Expression", rules[0].Expression, "sum(rate(errors[5m])) > 0.1")
}

func TestParseAsPrometheusAlertRules_PrometheusRule(t *testing.T) {
	data := []byte(`apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: test-rules
  labels:
    dash0.com/id: test-id
spec:
  groups:
    - name: test-group
      interval: 1m
      rules:
        - alert: HighErrorRate
          expr: sum(rate(errors[5m])) > 0.1
          for: 5m
        - alert: LowAvailability
          expr: up == 0
`)
	rules, err := ParseAsPrometheusAlertRules(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(rules))
	}
	assertEqual(t, "rules[0].Name", rules[0].Name, "HighErrorRate")
	assertEqual(t, "rules[1].Name", rules[1].Name, "LowAvailability")
	assertPtrEqual(t, "rules[0].Id", rules[0].Id, "test-id")
	assertPtrEqual(t, "rules[1].Id", rules[1].Id, "test-id")
}

func TestParseAsPrometheusAlertRules_SkipsRecordingRules(t *testing.T) {
	data := []byte(`apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: mixed-rules
spec:
  groups:
    - name: group
      rules:
        - record: my_recording_rule
          expr: sum(rate(http_requests[5m]))
        - alert: MyAlert
          expr: up == 0
`)
	rules, err := ParseAsPrometheusAlertRules(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}
	assertEqual(t, "Name", rules[0].Name, "MyAlert")
}

func TestParseAsPrometheusAlertRules_NoAlerts(t *testing.T) {
	data := []byte(`apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: recording-only
spec:
  groups:
    - name: group
      rules:
        - record: my_recording_rule
          expr: sum(rate(http_requests[5m]))
`)
	_, err := ParseAsPrometheusAlertRules(data)
	if err == nil {
		t.Error("expected error for no alerting rules")
	}
}

func TestParseAsPrometheusAlertRules_InvalidYAML(t *testing.T) {
	_, err := ParseAsPrometheusAlertRules([]byte("{{invalid"))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestUnmarshalPrometheusRule(t *testing.T) {
	yamlDoc := `apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  labels:
    dash0.com/id: rule-99
    dash0.com/dataset: production
spec:
  groups:
    - name: my-group
      interval: 1m
      rules:
        - alert: HighErrors
          expr: sum(rate(errors[5m])) > 0.1
          for: 5m
          annotations:
            summary: High error rate
`
	rule, err := UnmarshalPrometheusRule([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "Name", rule.Name, "my-group - HighErrors")
	assertEqual(t, "Expression", rule.Expression, "sum(rate(errors[5m])) > 0.1")
	assertPtrEqual(t, "Id", rule.Id, "rule-99")
	assertPtrEqual(t, "Dataset", rule.Dataset, "production")
	assertPtrEqual(t, "For", rule.For, "5m")
	assertPtrEqual(t, "Interval", rule.Interval, "1m")
	if rule.Annotations == nil {
		t.Fatal("Annotations is nil")
	}
	assertPtrEqual(t, "Summary", rule.Annotations.Summary, "High error rate")

	if rule.Enabled == nil || !*rule.Enabled {
		t.Error("Enabled should default to true")
	}
}

func TestUnmarshalPrometheusRule_MultipleGroups(t *testing.T) {
	yamlDoc := `apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
spec:
  groups:
    - name: group1
      rules:
        - alert: A
          expr: up == 0
    - name: group2
      rules:
        - alert: B
          expr: up == 0
`
	_, err := UnmarshalPrometheusRule([]byte(yamlDoc))
	if err == nil {
		t.Error("expected error for multiple groups")
	}
}

func TestMarshalPrometheusRule(t *testing.T) {
	forDur := dash0.Duration("5m")
	rule := &dash0.PrometheusAlertRule{
		Name:       "my-group - HighErrors",
		Expression: "sum(rate(errors[5m])) > 0.1",
		For:        &forDur,
	}

	yamlStr, err := MarshalPrometheusRule(rule)
	if err != nil {
		t.Fatal(err)
	}

	// Verify it round-trips back
	parsed, err := UnmarshalPrometheusRule(yamlStr)
	if err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}
	assertEqual(t, "Name", parsed.Name, "my-group - HighErrors")
	assertEqual(t, "Expression", parsed.Expression, "sum(rate(errors[5m])) > 0.1")
}

func TestUnmarshalPrometheusRule_PreservesDoubleQuotesInExpression(t *testing.T) {
	yamlDoc := `apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
spec:
  groups:
    - name: quoting
      rules:
        - alert: DoubleQuoted
          expr: 'sum(rate(http_requests{service="my-api", status="500"}[5m]))'
`
	rule, err := UnmarshalPrometheusRule([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "Expression", rule.Expression,
		`sum(rate(http_requests{service="my-api", status="500"}[5m]))`)
}

func TestUnmarshalPrometheusRule_PreservesSingleQuotesInAnnotations(t *testing.T) {
	yamlDoc := `apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
spec:
  groups:
    - name: quoting
      rules:
        - alert: QuotedAnnotation
          expr: up == 0
          annotations:
            summary: "Service 'my-api' is down"
            description: "Check the 'status' label for details"
`
	rule, err := UnmarshalPrometheusRule([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	assertPtrEqual(t, "Summary", rule.Annotations.Summary, "Service 'my-api' is down")
	assertPtrEqual(t, "Description", rule.Annotations.Description, "Check the 'status' label for details")
}

func TestUnmarshalPrometheusRule_PreservesMixedQuotesInExpression(t *testing.T) {
	yamlDoc := `apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
spec:
  groups:
    - name: quoting
      rules:
        - alert: MixedQuotes
          expr: "sum(rate(http_requests{service='my-api', path=\"/health\"}[5m]))"
`
	rule, err := UnmarshalPrometheusRule([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "Expression", rule.Expression,
		`sum(rate(http_requests{service='my-api', path="/health"}[5m]))`)
}

func TestRoundTripPreservesDoubleQuotesInExpression(t *testing.T) {
	expr := `sum(rate(http_requests{service="my-api", status="500"}[5m]))`
	forDur := dash0.Duration("5m")
	rule := &dash0.PrometheusAlertRule{
		Name:       "quoting - DoubleQuoted",
		Expression: expr,
		For:        &forDur,
	}

	yamlBytes, err := MarshalPrometheusRule(rule)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := UnmarshalPrometheusRule(yamlBytes)
	if err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}
	assertEqual(t, "Expression", parsed.Expression, expr)
}

func TestRoundTripPreservesSingleQuotesInAnnotations(t *testing.T) {
	summary := "Service 'my-api' is down"
	description := "Check the 'status' label for details"
	rule := &dash0.PrometheusAlertRule{
		Name:       "quoting - SingleQuotedAnnotations",
		Expression: "up == 0",
		Annotations: &dash0.PrometheusAlertRule_Annotations{
			Summary:     &summary,
			Description: &description,
		},
	}

	yamlBytes, err := MarshalPrometheusRule(rule)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := UnmarshalPrometheusRule(yamlBytes)
	if err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}
	assertPtrEqual(t, "Summary", parsed.Annotations.Summary, summary)
	assertPtrEqual(t, "Description", parsed.Annotations.Description, description)
}

func TestRoundTripPreservesMixedQuotesInExpression(t *testing.T) {
	expr := `sum(rate(http_requests{service='my-api', path="/health"}[5m]))`
	rule := &dash0.PrometheusAlertRule{
		Name:       "quoting - MixedQuotes",
		Expression: expr,
	}

	yamlBytes, err := MarshalPrometheusRule(rule)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := UnmarshalPrometheusRule(yamlBytes)
	if err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}
	assertEqual(t, "Expression", parsed.Expression, expr)
}

func TestRoundTripPreservesQuotesInLabels(t *testing.T) {
	labels := map[string]string{
		"team":     `platform "core"`,
		"severity": "it's critical",
	}
	rule := &dash0.PrometheusAlertRule{
		Name:       "quoting - LabelQuotes",
		Expression: "up == 0",
		Labels:     &labels,
	}

	yamlBytes, err := MarshalPrometheusRule(rule)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := UnmarshalPrometheusRule(yamlBytes)
	if err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}
	if parsed.Labels == nil {
		t.Fatal("Labels is nil")
	}
	if (*parsed.Labels)["team"] != `platform "core"` {
		t.Errorf("team label = %q, want %q", (*parsed.Labels)["team"], `platform "core"`)
	}
	if (*parsed.Labels)["severity"] != "it's critical" {
		t.Errorf("severity label = %q, want %q", (*parsed.Labels)["severity"], "it's critical")
	}
}

func TestParseAsPrometheusAlertRules_PreservesQuotesInExpressions(t *testing.T) {
	data := []byte(`apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: quoted-rules
spec:
  groups:
    - name: quoting
      rules:
        - alert: DoubleQuoted
          expr: 'sum(rate(http_requests{service="my-api"}[5m]))'
        - alert: SingleQuoted
          expr: "sum(rate(http_requests{service='my-api'}[5m]))"
`)
	rules, err := ParseAsPrometheusAlertRules(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(rules))
	}
	assertEqual(t, "rules[0].Expression", rules[0].Expression,
		`sum(rate(http_requests{service="my-api"}[5m]))`)
	assertEqual(t, "rules[1].Expression", rules[1].Expression,
		`sum(rate(http_requests{service='my-api'}[5m]))`)
}

func TestMarshalPrometheusRule_InvalidForDuration(t *testing.T) {
	badDur := dash0.Duration("not-a-duration")
	rule := &dash0.PrometheusAlertRule{
		Name:       "g - r",
		Expression: "up == 0",
		For:        &badDur,
	}
	_, err := MarshalPrometheusRule(rule)
	if err == nil {
		t.Fatal("expected error for invalid \"for\" duration")
	}
	if !strings.Contains(err.Error(), "invalid \"for\" duration") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMarshalPrometheusRule_InvalidKeepFiringForDuration(t *testing.T) {
	badDur := dash0.Duration("bad")
	rule := &dash0.PrometheusAlertRule{
		Name:          "g - r",
		Expression:    "up == 0",
		KeepFiringFor: &badDur,
	}
	_, err := MarshalPrometheusRule(rule)
	if err == nil {
		t.Fatal("expected error for invalid \"keep_firing_for\" duration")
	}
	if !strings.Contains(err.Error(), "invalid \"keep_firing_for\" duration") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMarshalPrometheusRule_InvalidIntervalDuration(t *testing.T) {
	badDur := dash0.Duration("bad")
	rule := &dash0.PrometheusAlertRule{
		Name:       "g - r",
		Expression: "up == 0",
		Interval:   &badDur,
	}
	_, err := MarshalPrometheusRule(rule)
	if err == nil {
		t.Fatal("expected error for invalid \"interval\" duration")
	}
	if !strings.Contains(err.Error(), "invalid \"interval\" duration") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- MergeAnnotations ---
//
// These cases derive from
// fixtures/check-rule-annotation-parity/multi-rule-with-top-level-annotations.yaml
// and its expected-check-rules.yaml counterpart in dash0-iac-maintainer-skills.

func TestMergeAnnotations_RuleWinsOnConflict(t *testing.T) {
	metadataAnnotations := map[string]string{
		"dash0.com/notification-channel-ids": "3fa42d0c-6b8e-4c1a-9f2d-111111111111,3fa42d0c-6b8e-4c1a-9f2d-222222222222",
	}
	ruleAnnotations := map[string]string{
		"summary":                            "Checkout error rate is elevated",
		"dash0.com/notification-channel-ids": "3fa42d0c-6b8e-4c1a-9f2d-333333333333",
		"runbook_url":                        "https://runbooks.example.com/checkout-error-rate",
	}

	merged := MergeAnnotations(metadataAnnotations, ruleAnnotations)

	assertEqual(t, "notification-channel-ids", merged["dash0.com/notification-channel-ids"], "3fa42d0c-6b8e-4c1a-9f2d-333333333333")
	assertEqual(t, "summary", merged["summary"], "Checkout error rate is elevated")
	assertEqual(t, "runbook_url", merged["runbook_url"], "https://runbooks.example.com/checkout-error-rate")
	if len(merged) != 3 {
		t.Errorf("got %d merged entries, want 3", len(merged))
	}
}

func TestMergeAnnotations_TopLevelOnlyPreserved(t *testing.T) {
	metadataAnnotations := map[string]string{
		"dash0.com/notification-channel-ids": "3fa42d0c-6b8e-4c1a-9f2d-111111111111,3fa42d0c-6b8e-4c1a-9f2d-222222222222",
	}

	merged := MergeAnnotations(metadataAnnotations, nil)

	assertEqual(t, "notification-channel-ids", merged["dash0.com/notification-channel-ids"], "3fa42d0c-6b8e-4c1a-9f2d-111111111111,3fa42d0c-6b8e-4c1a-9f2d-222222222222")
	if len(merged) != 1 {
		t.Errorf("got %d merged entries, want 1", len(merged))
	}
}

func TestMergeAnnotations_NoTopLevelAnnotations_NoOp(t *testing.T) {
	ruleAnnotations := map[string]string{"summary": "Sum"}

	merged := MergeAnnotations(nil, ruleAnnotations)

	if len(merged) != 1 || merged["summary"] != "Sum" {
		t.Errorf("expected merge with no top-level annotations to be a no-op, got %v", merged)
	}
	if len(merged) != len(ruleAnnotations) {
		t.Errorf("got %d entries, want %d (no-op)", len(merged), len(ruleAnnotations))
	}
}

func TestMergeAnnotations_NilSafe(t *testing.T) {
	if merged := MergeAnnotations(nil, nil); len(merged) != 0 {
		t.Errorf("MergeAnnotations(nil, nil) = %v, want empty", merged)
	}
	if merged := MergeAnnotations(map[string]string{}, nil); len(merged) != 0 {
		t.Errorf("MergeAnnotations(empty, nil) = %v, want empty", merged)
	}
	if merged := MergeAnnotations(nil, map[string]string{}); len(merged) != 0 {
		t.Errorf("MergeAnnotations(nil, empty) = %v, want empty", merged)
	}
}

// --- ParseAsPrometheusAlertRules: top-level annotation merge ---
//
// Adapted from
// fixtures/check-rule-annotation-parity/multi-rule-with-top-level-annotations.yaml.

func TestParseAsPrometheusAlertRules_MergesTopLevelAnnotations(t *testing.T) {
	data := []byte(`apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: checkout-check-rules
  namespace: monitoring
  labels:
    prometheus: example
    dash0.com/dataset: default
  annotations:
    dash0.com/notification-channel-ids: "3fa42d0c-6b8e-4c1a-9f2d-111111111111,3fa42d0c-6b8e-4c1a-9f2d-222222222222"
spec:
  groups:
    - name: Alerting
      interval: 1m
      rules:
        - alert: CheckoutHighLatency
          expr: histogram_quantile(0.99, sum(rate(http_request_duration[5m]))) > 1.5
          labels:
            team: checkout
            severity: high
        - alert: CheckoutHighErrorRate
          expr: sum(rate(http_requests_errors[5m])) / sum(rate(http_requests_total[5m])) * 100 > 5
          for: 5m
          annotations:
            summary: "Checkout error rate is elevated"
            description: "More than 5 percent of checkout requests are failing with 5xx responses."
            dash0.com/notification-channel-ids: "3fa42d0c-6b8e-4c1a-9f2d-333333333333"
            runbook_url: "https://runbooks.example.com/checkout-error-rate"
          labels:
            team: checkout
            severity: critical
`)

	rules, err := ParseAsPrometheusAlertRules(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(rules))
	}

	// Rule 1 had no annotations of its own: it must inherit the top-level
	// notification-channel-ids value in full.
	rule1 := rules[0]
	if rule1.Annotations == nil {
		t.Fatal("rule 1 Annotations is nil")
	}
	assertEqual(t, "rule1 notification-channel-ids",
		rule1.Annotations.AdditionalProperties["dash0.com/notification-channel-ids"],
		"3fa42d0c-6b8e-4c1a-9f2d-111111111111,3fa42d0c-6b8e-4c1a-9f2d-222222222222")

	// Rule 2 overrides notification-channel-ids and keeps its own
	// runbook_url; the top-level value must not leak in alongside the override.
	rule2 := rules[1]
	if rule2.Annotations == nil {
		t.Fatal("rule 2 Annotations is nil")
	}
	assertEqual(t, "rule2 notification-channel-ids",
		rule2.Annotations.AdditionalProperties["dash0.com/notification-channel-ids"],
		"3fa42d0c-6b8e-4c1a-9f2d-333333333333")
	assertEqual(t, "rule2 runbook_url",
		rule2.Annotations.AdditionalProperties["runbook_url"],
		"https://runbooks.example.com/checkout-error-rate")
}

func TestUnmarshalPrometheusRule_MergesTopLevelAnnotations(t *testing.T) {
	yamlDoc := `apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: single-rule
  annotations:
    dash0.com/notification-channel-ids: "top-level-channel"
spec:
  groups:
    - name: my-group
      rules:
        - alert: HighErrors
          expr: up == 0
          annotations:
            summary: "High error rate"
`
	rule, err := UnmarshalPrometheusRule([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	if rule.Annotations == nil {
		t.Fatal("Annotations is nil")
	}
	assertEqual(t, "notification-channel-ids",
		rule.Annotations.AdditionalProperties["dash0.com/notification-channel-ids"], "top-level-channel")
	assertPtrEqual(t, "Summary", rule.Annotations.Summary, "High error rate")
}

func TestDetectorExportRoundTrip(t *testing.T) {
	paths, err := filepath.Glob("testdata/detector-export/*.yaml")
	require.NoError(t, err)
	require.Len(t, paths, 7)
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			input, err := os.ReadFile(path)
			require.NoError(t, err)
			rule, err := UnmarshalPrometheusRule(input)
			require.NoError(t, err)
			original := rule.Expression
			output, err := MarshalPrometheusRule(rule)
			require.NoError(t, err)
			equivalent, err := Equivalent(input, output, []string{"metadata.name", "metadata.labels"}, nil)
			require.NoError(t, err)
			assert.True(t, equivalent, string(output))
			reread, err := UnmarshalPrometheusRule(output)
			require.NoError(t, err)
			assert.Equal(t, original, reread.Expression)
			assert.NotContains(t, string(output), "dash0.com/volume-floor")
			assert.Nil(t, reread.Dataset)
			assert.False(t, *reread.Enabled)
		})
	}
}

func TestMarshalDetectorAnnotations(t *testing.T) {
	for _, direction := range []dash0.AnomalyDirection{dash0.AnomalyDirectionAbove, dash0.AnomalyDirectionBelow, dash0.AnomalyDirectionBoth} {
		rule := &dash0.PrometheusAlertRule{Name: "rule", Expression: "observed", Thresholds: &dash0.CheckThresholds{Failed: dash0.Ptr(3.0), Degraded: dash0.Ptr(2.0), Baseline: &dash0.CheckThresholdBaseline{Direction: direction, SpreadFloor: dash0.Ptr(0.1)}}, Annotations: &dash0.PrometheusAlertRule_Annotations{AdditionalProperties: map[string]string{"dash0.com/change-gate-value": "invalid", "dash0.com/volume-floor": "999"}}}
		data, err := MarshalPrometheusRule(rule)
		require.NoError(t, err)
		parsed, err := UnmarshalPrometheusRule(data)
		require.NoError(t, err)
		require.NotNil(t, parsed.Thresholds.Baseline)
		assert.Equal(t, direction, parsed.Thresholds.Baseline.Direction)
		assert.Nil(t, parsed.Annotations)
		rule.Thresholds.Baseline.VolumeFloor = dash0.Ptr(1.0)
		data, err = MarshalPrometheusRule(rule)
		require.NoError(t, err)
		parsed, err = UnmarshalPrometheusRule(data)
		require.NoError(t, err)
		assert.InDelta(t, 1, *parsed.Thresholds.Baseline.VolumeFloor, 1e-12)
	}
}

func TestMarshalChangeGateZeroDegradedThreshold(t *testing.T) {
	rule := &dash0.PrometheusAlertRule{Name: "rule", Expression: "observed > $__threshold", Thresholds: &dash0.CheckThresholds{Failed: dash0.Ptr(1.0), Degraded: dash0.Ptr(0.0), ChangeGate: &dash0.CheckThresholdChangeGate{Comparison: dash0.RelativeFactor, Value: 2, BaselineWindow: "1h"}}}
	data, err := MarshalPrometheusRule(rule)
	require.NoError(t, err)
	read, err := UnmarshalPrometheusRule(data)
	require.NoError(t, err)
	require.NotNil(t, read.Thresholds.Degraded)
	assert.Zero(t, *read.Thresholds.Degraded)
}

func TestMarshalAPIReadDetectorFloors(t *testing.T) {
	for _, thresholds := range []string{
		`{"failed":0,"degraded":1,"baseline":{"direction":"below","spreadFloor":0.17,"volumeFloor":12.5}}`,
		`{"failed":0,"degraded":1,"changeGate":{"comparison":"absolute_delta","value":0.037,"baselineWindow":"90m","volumeFloor":12.5}}`,
	} {
		t.Run(thresholds, func(t *testing.T) {
			// Deserialize the public API shape rather than reconstructing fields from annotations.
			data := []byte(`{"name":"Alerting - Native floor","expression":"observed > $__threshold","dataset":"production","thresholds":` + thresholds + `,"annotations":{"summary":"Native floor"}}`)
			var apiRule dash0.PrometheusAlertRule
			require.NoError(t, json.Unmarshal(data, &apiRule))
			before, err := json.Marshal(apiRule)
			require.NoError(t, err)
			exported, err := MarshalPrometheusRule(&apiRule)
			require.NoError(t, err)
			require.Contains(t, string(exported), "dash0.com/volume-floor")
			require.Contains(t, string(exported), "dash0-threshold-critical")
			imported, err := UnmarshalPrometheusRule(exported)
			require.NoError(t, err)
			assert.Equal(t, apiRule.Thresholds, imported.Thresholds)
			assert.Equal(t, apiRule.Expression, imported.Expression)
			assert.Nil(t, imported.Dataset, "marshal must keep the existing metadata-label behavior")
			reexported, err := MarshalPrometheusRule(imported)
			require.NoError(t, err)
			assert.Equal(t, string(exported), string(reexported))
			after, err := json.Marshal(apiRule)
			require.NoError(t, err)
			assert.JSONEq(t, string(before), string(after))
			rules, err := ParseAsPrometheusAlertRules(exported)
			require.NoError(t, err)
			require.Len(t, rules, 1)
			assert.Equal(t, apiRule.Thresholds, rules[0].Thresholds)
		})
	}
}

func TestMarshalRejectsInvalidDetectorConfiguration(t *testing.T) {
	for _, thresholds := range []*dash0.CheckThresholds{
		{Baseline: &dash0.CheckThresholdBaseline{}},
		{Baseline: &dash0.CheckThresholdBaseline{Direction: "sideways"}},
		{Baseline: &dash0.CheckThresholdBaseline{Direction: dash0.AnomalyDirectionAbove}, ChangeGate: &dash0.CheckThresholdChangeGate{}},
	} {
		_, err := MarshalPrometheusRule(&dash0.PrometheusAlertRule{Name: "rule", Expression: "observed", Thresholds: thresholds})
		require.Error(t, err)
	}
}
