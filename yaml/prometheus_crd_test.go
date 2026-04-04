package yaml

import (
	"strings"
	"testing"

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
