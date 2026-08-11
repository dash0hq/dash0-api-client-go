package dash0

import (
	"strings"
	"testing"
	"time"
)

func TestGetPrometheusRuleDataset(t *testing.T) {
	tests := []struct {
		name string
		rule *PrometheusRules
		want string
	}{
		{"nil rule", nil, ""},
		{"nil labels", &PrometheusRules{}, ""},
		{"empty labels", &PrometheusRules{Metadata: PrometheusRulesMetadata{Labels: map[string]string{}}}, ""},
		{"dataset present", &PrometheusRules{Metadata: PrometheusRulesMetadata{Labels: map[string]string{"dash0.com/dataset": "production"}}}, "production"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPrometheusRuleDataset(tt.rule); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetPrometheusRuleDataset(t *testing.T) {
	rule := &PrometheusRules{}
	SetPrometheusRuleDataset(rule, "production")
	if rule.Metadata.Labels["dash0.com/dataset"] != "production" {
		t.Errorf("Dataset = %q, want %q", rule.Metadata.Labels["dash0.com/dataset"], "production")
	}
}

func TestSetPrometheusRuleDataset_Nil(t *testing.T) {
	SetPrometheusRuleDataset(nil, "production") // should not panic
}

func TestGetPrometheusRuleID(t *testing.T) {
	tests := []struct {
		name string
		rule *PrometheusRules
		want string
	}{
		{"nil rule", nil, ""},
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
	if got := GetPrometheusRuleName(nil); got != "" {
		t.Errorf("got %q, want empty for nil", got)
	}
}

func TestConvertPrometheusRuleToPrometheusAlertRule(t *testing.T) {
	rule := &PrometheusRule{
		Alert: "HighErrorRate",
		Expr:  "sum(rate(errors[5m])) > 0.1",
		For:   5 * time.Minute,
		Labels: map[string]string{
			"severity": "critical",
		},
		Annotations: map[string]string{
			"summary":     "High error rate detected",
			"description": "Error rate exceeds threshold",
		},
	}

	checkRule, err := ConvertPrometheusRuleToPrometheusAlertRule(rule, 1*time.Minute, "test-id")
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
		KeepFiringFor: 10 * time.Minute,
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

// --- Threshold-annotation presence validation ---
//
// Adapted from
// fixtures/check-rule-annotation-parity/threshold-missing-annotation.yaml and
// its multi-rule-with-top-level-annotations.yaml sibling.

func TestConvertPrometheusRuleToPrometheusAlertRule_ThresholdTokenWithoutAnnotation(t *testing.T) {
	rule := &PrometheusRule{
		Alert: "AdserviceErrorBudgetBurn",
		Expr:  "sum(rate(http_requests_errors[5m])) / sum(rate(http_requests_total[5m])) * 100 > $__threshold",
		Annotations: map[string]string{
			"summary":     "Adservice error budget burn is high",
			"description": "Adservice's 5xx error rate has crossed the (undeclared) threshold.",
		},
	}

	_, err := ConvertPrometheusRuleToPrometheusAlertRule(rule, 0, "")
	if err == nil {
		t.Fatal("expected error for $__threshold token with no threshold annotation")
	}
	if !strings.Contains(err.Error(), "$__threshold") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConvertPrometheusRuleToPrometheusAlertRule_ThresholdTokenWithAnnotation_Succeeds(t *testing.T) {
	rule := &PrometheusRule{
		Alert: "AdserviceErrorBudgetBurn",
		Expr:  "sum(rate(http_requests_errors[5m])) / sum(rate(http_requests_total[5m])) * 100 > $__threshold",
		Annotations: map[string]string{
			"dash0-threshold-critical": "5",
		},
	}

	result, err := ConvertPrometheusRuleToPrometheusAlertRule(rule, 0, "")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result.Thresholds == nil || result.Thresholds.Failed == nil || *result.Thresholds.Failed != 5 {
		t.Errorf("Thresholds.Failed = %v, want 5", result.Thresholds)
	}
}

func TestConvertPrometheusRuleToPrometheusAlertRule_ThresholdTokenWithLegacyAnnotation_Succeeds(t *testing.T) {
	rule := &PrometheusRule{
		Alert: "AdserviceErrorBudgetBurn",
		Expr:  "sum(rate(http_requests_errors[5m])) > $__threshold",
		Annotations: map[string]string{
			"threshold-degraded": "0.5",
		},
	}

	if _, err := ConvertPrometheusRuleToPrometheusAlertRule(rule, 0, ""); err != nil {
		t.Fatalf("expected success with legacy threshold-degraded annotation, got error: %v", err)
	}
}

func TestConvertPrometheusRuleToPrometheusAlertRule_NoThresholdTokenNoAnnotation_Succeeds(t *testing.T) {
	rule := &PrometheusRule{
		Alert: "SimpleAlert",
		Expr:  "up == 0",
	}

	if _, err := ConvertPrometheusRuleToPrometheusAlertRule(rule, 0, ""); err != nil {
		t.Fatalf("expected success for rule without $__threshold token, got error: %v", err)
	}
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
