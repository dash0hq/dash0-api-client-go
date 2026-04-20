package dash0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStripCheckRuleServerFields(t *testing.T) {
	dataset := "ds"
	labels := map[string]string{
		"severity":         "critical",
		"dash0.com/origin": "terraform",
	}
	r := &PrometheusAlertRule{
		Name:    "test",
		Dataset: &dataset,
		Labels:  &labels,
	}

	StripCheckRuleServerFields(r)

	if r.Dataset != nil {
		t.Error("Dataset should be nil")
	}
	if _, ok := (*r.Labels)["dash0.com/origin"]; ok {
		t.Error("dash0.com/origin label should be removed")
	}
	if (*r.Labels)["severity"] != "critical" {
		t.Error("other labels should be preserved")
	}
}

func TestStripCheckRuleServerFields_NilLabels(t *testing.T) {
	r := &PrometheusAlertRule{Name: "test"}
	StripCheckRuleServerFields(r) // should not panic
}

func TestClearCheckRuleID(t *testing.T) {
	r := &PrometheusAlertRule{Id: Ptr("abc")}
	ClearCheckRuleID(r)
	if r.Id != nil {
		t.Error("Id should be nil")
	}
}

func TestClearCheckRuleID_AlreadyNil(t *testing.T) {
	r := &PrometheusAlertRule{}
	ClearCheckRuleID(r) // should not panic
}

func TestGetCheckRuleDataset(t *testing.T) {
	tests := []struct {
		name string
		rule *PrometheusAlertRule
		want string
	}{
		{"nil rule", nil, ""},
		{"nil dataset", &PrometheusAlertRule{}, ""},
		{"with dataset", &PrometheusAlertRule{Dataset: Ptr("production")}, "production"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCheckRuleDataset(tt.rule); got != tt.want {
				t.Errorf("GetCheckRuleDataset() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetCheckRuleDataset(t *testing.T) {
	rule := &PrometheusAlertRule{Name: "test"}
	SetCheckRuleDataset(rule, "production")
	if rule.Dataset == nil || *rule.Dataset != "production" {
		t.Error("expected dataset to be set to production")
	}
}

func TestSetCheckRuleDataset_Nil(t *testing.T) {
	SetCheckRuleDataset(nil, "production") // should not panic
}

func TestGetCheckRuleID(t *testing.T) {
	tests := []struct {
		name string
		rule *PrometheusAlertRule
		want string
	}{
		{"with ID", &PrometheusAlertRule{Id: Ptr("abc")}, "abc"},
		{"nil ID", &PrometheusAlertRule{}, ""},
		{"nil rule", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCheckRuleID(tt.rule); got != tt.want {
				t.Errorf("GetCheckRuleID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetCheckRuleID(t *testing.T) {
	r := &PrometheusAlertRule{Name: "test"}
	SetCheckRuleID(r, "new-id")
	if r.Id == nil || *r.Id != "new-id" {
		t.Error("expected ID to be set to new-id")
	}
}

func TestSetCheckRuleID_Overwrites(t *testing.T) {
	r := &PrometheusAlertRule{Id: Ptr("existing-id")}
	SetCheckRuleID(r, "new-id")
	if *r.Id != "new-id" {
		t.Errorf("ID = %q, want %q", *r.Id, "new-id")
	}
}

func TestSetCheckRuleIDIfAbsent(t *testing.T) {
	r := &PrometheusAlertRule{Name: "test"}
	SetCheckRuleIDIfAbsent(r, "new-id")
	if r.Id == nil || *r.Id != "new-id" {
		t.Error("expected ID to be set to new-id")
	}
}

func TestSetCheckRuleIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	r := &PrometheusAlertRule{Id: Ptr("existing-id")}
	SetCheckRuleIDIfAbsent(r, "new-id")
	if *r.Id != "existing-id" {
		t.Errorf("ID = %q, want %q (should not overwrite)", *r.Id, "existing-id")
	}
}

func TestGetCheckRuleName(t *testing.T) {
	if got := GetCheckRuleName(&PrometheusAlertRule{Name: "my-rule"}); got != "my-rule" {
		t.Errorf("got %q, want %q", got, "my-rule")
	}
	if got := GetCheckRuleName(&PrometheusAlertRule{}); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := GetCheckRuleName(nil); got != "" {
		t.Errorf("got %q, want empty for nil", got)
	}
}

func TestCreateCheckRule_201(t *testing.T) {
	rule := PrometheusAlertRule{
		Name: "test-rule",
		Id:   Ptr("cr-123"),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(rule)
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	got, err := client.CreateCheckRule(context.Background(), &rule, nil)
	if err != nil {
		t.Fatalf("CreateCheckRule failed: %v", err)
	}

	if got.Name != "test-rule" {
		t.Errorf("expected name %q, got %q", "test-rule", got.Name)
	}
	if GetCheckRuleID(got) != "cr-123" {
		t.Errorf("expected ID %q, got %q", "cr-123", GetCheckRuleID(got))
	}
}
