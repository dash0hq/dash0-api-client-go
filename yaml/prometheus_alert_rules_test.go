package yaml

import (
	"testing"
	"time"

	dash0 "github.com/dash0hq/dash0-api-client-go"
)

func TestGetPrometheusRuleID(t *testing.T) {
	tests := []struct {
		name string
		rule *PrometheusRules
		want string
	}{
		{"nil labels", &PrometheusRules{}, ""},
		{"empty labels", &PrometheusRules{Metadata: PrometheusRulesMetadata{Labels: map[string]string{}}}, ""},
		{"id present", &PrometheusRules{Metadata: PrometheusRulesMetadata{Labels: map[string]string{"dash0.com/id": "abc"}}}, "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPrometheusRuleID(tt.rule); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClearPrometheusRuleID(t *testing.T) {
	rule := &PrometheusRules{Metadata: PrometheusRulesMetadata{Labels: map[string]string{"dash0.com/id": "abc"}}}
	ClearPrometheusRuleID(rule)
	if _, ok := rule.Metadata.Labels["dash0.com/id"]; ok {
		t.Error("dash0.com/id should be removed")
	}
}

func TestClearPrometheusRuleID_NilLabels(t *testing.T) {
	rule := &PrometheusRules{}
	ClearPrometheusRuleID(rule) // should not panic
}

func TestSetPrometheusRuleID(t *testing.T) {
	rule := &PrometheusRules{}
	SetPrometheusRuleID(rule, "new-id")
	if rule.Metadata.Labels == nil {
		t.Fatal("expected labels to be initialized")
	}
	if rule.Metadata.Labels["dash0.com/id"] != "new-id" {
		t.Errorf("ID = %q, want %q", rule.Metadata.Labels["dash0.com/id"], "new-id")
	}
}

func TestSetPrometheusRuleID_Overwrites(t *testing.T) {
	rule := &PrometheusRules{
		Metadata: PrometheusRulesMetadata{
			Labels: map[string]string{"dash0.com/id": "existing-id"},
		},
	}
	SetPrometheusRuleID(rule, "new-id")
	if rule.Metadata.Labels["dash0.com/id"] != "new-id" {
		t.Errorf("ID = %q, want %q", rule.Metadata.Labels["dash0.com/id"], "new-id")
	}
}

func TestSetPrometheusRuleIDIfAbsent(t *testing.T) {
	rule := &PrometheusRules{}
	SetPrometheusRuleIDIfAbsent(rule, "new-id")
	if rule.Metadata.Labels == nil {
		t.Fatal("expected labels to be initialized")
	}
	if rule.Metadata.Labels["dash0.com/id"] != "new-id" {
		t.Errorf("ID = %q, want %q", rule.Metadata.Labels["dash0.com/id"], "new-id")
	}
}

func TestSetPrometheusRuleIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	rule := &PrometheusRules{
		Metadata: PrometheusRulesMetadata{
			Labels: map[string]string{"dash0.com/id": "existing-id"},
		},
	}
	SetPrometheusRuleIDIfAbsent(rule, "new-id")
	if rule.Metadata.Labels["dash0.com/id"] != "existing-id" {
		t.Errorf("ID = %q, want %q (should not overwrite)", rule.Metadata.Labels["dash0.com/id"], "existing-id")
	}
}

func TestGetPrometheusRuleName(t *testing.T) {
	rule := &PrometheusRules{Metadata: PrometheusRulesMetadata{Name: "my-rule"}}
	if got := GetPrometheusRuleName(rule); got != "my-rule" {
		t.Errorf("got %q, want %q", got, "my-rule")
	}
	if got := GetPrometheusRuleName(&PrometheusRules{}); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestConvertPrometheusRuleToPrometheusAlertRule(t *testing.T) {
	rule := &PrometheusRule{
		Alert: "HighErrorRate",
		Expr:  "sum(rate(errors[5m])) > 0.1",
		For:   PromDuration(5 * time.Minute),
		Labels: map[string]string{
			"severity": "critical",
		},
		Annotations: map[string]string{
			"summary":     "High error rate detected",
			"description": "Error rate exceeds threshold",
		},
	}

	checkRule, err := ConvertPrometheusRuleToPrometheusAlertRule(rule, PromDuration(1*time.Minute), "test-id")
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, "Name", checkRule.Name, "HighErrorRate")
	assertEqual(t, "Expression", checkRule.Expression, "sum(rate(errors[5m])) > 0.1")
	assertPtrEqual(t, "For", checkRule.For, "5m")
	assertPtrEqual(t, "Interval", checkRule.Interval, "1m")
	assertPtrEqual(t, "Id", checkRule.Id, "test-id")
	if checkRule.Annotations == nil {
		t.Fatal("Annotations is nil")
	}
	assertPtrEqual(t, "Summary", checkRule.Annotations.Summary, "High error rate detected")
	assertPtrEqual(t, "Description", checkRule.Annotations.Description, "Error rate exceeds threshold")

	if checkRule.Labels == nil {
		t.Fatal("Labels is nil")
	}
	if (*checkRule.Labels)["severity"] != "critical" {
		t.Errorf("Labels[severity] = %q, want critical", (*checkRule.Labels)["severity"])
	}
}

func TestConvertPrometheusRuleToPrometheusAlertRule_Minimal(t *testing.T) {
	rule := &PrometheusRule{Alert: "SimpleAlert", Expr: "up == 0"}
	checkRule, err := ConvertPrometheusRuleToPrometheusAlertRule(rule, 0, "")
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, "Name", checkRule.Name, "SimpleAlert")
	assertEqual(t, "Expression", checkRule.Expression, "up == 0")
	if checkRule.For != nil {
		t.Error("For should be nil")
	}
	if checkRule.Interval != nil {
		t.Error("Interval should be nil")
	}
	if checkRule.Id != nil {
		t.Error("Id should be nil")
	}
	if checkRule.Enabled == nil || !*checkRule.Enabled {
		t.Error("Enabled should default to true")
	}
}

func TestConvertPrometheusRuleToPrometheusAlertRule_AnnotationsPreserved(t *testing.T) {
	rule := &PrometheusRule{
		Alert: "Test",
		Expr:  "up == 0",
		Annotations: map[string]string{
			"summary":     "Sum",
			"description": "Desc",
			"runbook_url": "https://example.com",
		},
	}

	result, err := ConvertPrometheusRuleToPrometheusAlertRule(rule, 0, "")
	if err != nil {
		t.Fatal(err)
	}

	if result.Annotations == nil {
		t.Fatal("Annotations is nil")
	}
	if result.Annotations.AdditionalProperties["runbook_url"] != "https://example.com" {
		t.Errorf("runbook_url = %q, want %q", result.Annotations.AdditionalProperties["runbook_url"], "https://example.com")
	}
	assertPtrEqual(t, "Summary", result.Annotations.Summary, "Sum")
}

func TestConvertPrometheusRuleToPrometheusAlertRule_DoesNotMutateInput(t *testing.T) {
	rule := &PrometheusRule{
		Alert: "Test",
		Expr:  "up == 0",
		Annotations: map[string]string{
			"summary": "Sum",
			"custom":  "value",
		},
		Labels: map[string]string{
			"severity": "critical",
		},
	}

	_, err := ConvertPrometheusRuleToPrometheusAlertRule(rule, 0, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(rule.Annotations) != 2 {
		t.Errorf("original annotations mutated: got %d entries, want 2", len(rule.Annotations))
	}
	if len(rule.Labels) != 1 {
		t.Errorf("original labels mutated: got %d entries, want 1", len(rule.Labels))
	}
}

func TestConvertPrometheusRuleToPrometheusAlertRule_KeepFiringFor(t *testing.T) {
	rule := &PrometheusRule{
		Alert:         "Test",
		Expr:          "up == 0",
		KeepFiringFor: PromDuration(10 * time.Minute),
	}

	result, err := ConvertPrometheusRuleToPrometheusAlertRule(rule, 0, "")
	if err != nil {
		t.Fatal(err)
	}

	assertPtrEqual(t, "KeepFiringFor", result.KeepFiringFor, "10m")
}

func TestConvertPrometheusRuleToPrometheusAlertRule_Thresholds(t *testing.T) {
	rule := &PrometheusRule{
		Alert: "Test",
		Expr:  "up == 0",
		Annotations: map[string]string{
			"dash0-threshold-critical": "0.95",
			"dash0-threshold-degraded": "0.8",
			"summary":                  "Sum",
		},
	}

	result, err := ConvertPrometheusRuleToPrometheusAlertRule(rule, 0, "")
	if err != nil {
		t.Fatal(err)
	}

	if result.Thresholds == nil {
		t.Fatal("Thresholds is nil")
	}
	if result.Thresholds.Failed == nil || *result.Thresholds.Failed != 0.95 {
		t.Errorf("Thresholds.Failed = %v, want 0.95", result.Thresholds.Failed)
	}
	if result.Thresholds.Degraded == nil || *result.Thresholds.Degraded != 0.8 {
		t.Errorf("Thresholds.Degraded = %v, want 0.8", result.Thresholds.Degraded)
	}

	// Threshold annotations should be removed from the annotations map.
	if result.Annotations == nil {
		t.Fatal("Annotations is nil")
	}
	if _, ok := result.Annotations.AdditionalProperties["dash0-threshold-critical"]; ok {
		t.Error("dash0-threshold-critical should be removed from annotations")
	}
	if _, ok := result.Annotations.AdditionalProperties["dash0-threshold-degraded"]; ok {
		t.Error("dash0-threshold-degraded should be removed from annotations")
	}
	assertPtrEqual(t, "Summary", result.Annotations.Summary, "Sum")
}

func TestConvertPrometheusRuleToPrometheusAlertRule_EnabledFalse(t *testing.T) {
	rule := &PrometheusRule{
		Alert: "Test",
		Expr:  "up == 0",
		Annotations: map[string]string{
			"dash0-enabled": "false",
		},
	}

	result, err := ConvertPrometheusRuleToPrometheusAlertRule(rule, 0, "")
	if err != nil {
		t.Fatal(err)
	}

	if result.Enabled == nil || *result.Enabled {
		t.Error("Enabled should be false")
	}

	// dash0-enabled should be removed from annotations.
	if result.Annotations != nil {
		if _, ok := result.Annotations.AdditionalProperties["dash0-enabled"]; ok {
			t.Error("dash0-enabled should be removed from annotations")
		}
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
	rule, err := UnmarshalPrometheusRule([]byte(yamlDoc), "my-dataset")
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "Name", rule.Name, "my-group - HighErrors")
	assertEqual(t, "Expression", rule.Expression, "sum(rate(errors[5m])) > 0.1")
	assertPtrEqual(t, "Dataset", rule.Dataset, "my-dataset")
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
	_, err := UnmarshalPrometheusRule([]byte(yamlDoc), "ds")
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
	parsed, err := UnmarshalPrometheusRule(yamlStr, "ds")
	if err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}
	assertEqual(t, "Name", parsed.Name, "my-group - HighErrors")
	assertEqual(t, "Expression", parsed.Expression, "sum(rate(errors[5m])) > 0.1")
}
