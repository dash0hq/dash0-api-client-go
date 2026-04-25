package dash0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestStripRecordingRuleServerFields(t *testing.T) {
	labels := map[string]string{
		"dash0.com/id":      "rr-1",
		"dash0.com/origin":  "terraform",
		"dash0.com/dataset": "production",
		"dash0.com/version": "3",
		"dash0.com/source":  "api",
		"team":              "sre",
	}
	annotations := map[string]string{
		"dash0.com/created-at":          "2025-01-01T00:00:00Z",
		"dash0.com/updated-at":          "2025-01-02T00:00:00Z",
		"dash0.com/deleted-at":          "2025-01-03T00:00:00Z",
		"dash0.com/first-evaluation-at": "2025-01-01T00:01:00Z",
		"dash0.com/enabled":             "true",
	}
	rule := &RecordingRule{
		Metadata: PrometheusRuleMetadata{
			Name:        "test-rule",
			Labels:      &labels,
			Annotations: &annotations,
		},
	}

	StripRecordingRuleServerFields(rule)

	if _, ok := (*rule.Metadata.Labels)["dash0.com/id"]; ok {
		t.Error("dash0.com/id label should be removed")
	}
	if _, ok := (*rule.Metadata.Labels)["dash0.com/origin"]; ok {
		t.Error("dash0.com/origin label should be removed")
	}
	if _, ok := (*rule.Metadata.Labels)["dash0.com/dataset"]; ok {
		t.Error("dash0.com/dataset label should be removed")
	}
	if _, ok := (*rule.Metadata.Labels)["dash0.com/version"]; ok {
		t.Error("dash0.com/version label should be removed")
	}
	if _, ok := (*rule.Metadata.Labels)["dash0.com/source"]; ok {
		t.Error("dash0.com/source label should be removed")
	}
	if (*rule.Metadata.Labels)["team"] != "sre" {
		t.Error("other labels should be preserved")
	}
	if _, ok := (*rule.Metadata.Annotations)["dash0.com/created-at"]; ok {
		t.Error("dash0.com/created-at annotation should be removed")
	}
	if _, ok := (*rule.Metadata.Annotations)["dash0.com/updated-at"]; ok {
		t.Error("dash0.com/updated-at annotation should be removed")
	}
	if _, ok := (*rule.Metadata.Annotations)["dash0.com/deleted-at"]; ok {
		t.Error("dash0.com/deleted-at annotation should be removed")
	}
	if _, ok := (*rule.Metadata.Annotations)["dash0.com/first-evaluation-at"]; ok {
		t.Error("dash0.com/first-evaluation-at annotation should be removed")
	}
	if (*rule.Metadata.Annotations)["dash0.com/enabled"] != "true" {
		t.Error("other annotations should be preserved")
	}
}

func TestStripRecordingRuleServerFields_NilLabels(t *testing.T) {
	rule := &RecordingRule{Metadata: PrometheusRuleMetadata{Name: "test"}}
	StripRecordingRuleServerFields(rule) // should not panic
}

func TestStripRecordingRuleServerFields_Nil(t *testing.T) {
	StripRecordingRuleServerFields(nil) // should not panic
}

func TestGetRecordingRuleID(t *testing.T) {
	tests := []struct {
		name string
		rule *RecordingRule
		want string
	}{
		{"nil rule", nil, ""},
		{"nil labels", &RecordingRule{Metadata: PrometheusRuleMetadata{Name: "test"}}, ""},
		{"with ID", func() *RecordingRule {
			labels := map[string]string{"dash0.com/id": "rr-42"}
			return &RecordingRule{Metadata: PrometheusRuleMetadata{Labels: &labels}}
		}(), "rr-42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRecordingRuleID(tt.rule); got != tt.want {
				t.Errorf("GetRecordingRuleID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetRecordingRuleID(t *testing.T) {
	rule := &RecordingRule{Metadata: PrometheusRuleMetadata{Name: "test"}}
	SetRecordingRuleID(rule, "rr-42")
	if GetRecordingRuleID(rule) != "rr-42" {
		t.Error("expected ID to be set to rr-42")
	}
}

func TestSetRecordingRuleID_Overwrites(t *testing.T) {
	labels := map[string]string{"dash0.com/id": "existing"}
	rule := &RecordingRule{Metadata: PrometheusRuleMetadata{Labels: &labels}}
	SetRecordingRuleID(rule, "new-id")
	if GetRecordingRuleID(rule) != "new-id" {
		t.Errorf("ID = %q, want %q", GetRecordingRuleID(rule), "new-id")
	}
}

func TestSetRecordingRuleID_Nil(t *testing.T) {
	SetRecordingRuleID(nil, "rr-42") // should not panic
}

func TestSetRecordingRuleIDIfAbsent(t *testing.T) {
	rule := &RecordingRule{Metadata: PrometheusRuleMetadata{Name: "test"}}
	SetRecordingRuleIDIfAbsent(rule, "rr-42")
	if GetRecordingRuleID(rule) != "rr-42" {
		t.Error("expected ID to be set to rr-42")
	}
}

func TestSetRecordingRuleIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	labels := map[string]string{"dash0.com/id": "existing"}
	rule := &RecordingRule{Metadata: PrometheusRuleMetadata{Labels: &labels}}
	SetRecordingRuleIDIfAbsent(rule, "new-id")
	if GetRecordingRuleID(rule) != "existing" {
		t.Errorf("ID = %q, want %q (should not overwrite)", GetRecordingRuleID(rule), "existing")
	}
}

func TestSetRecordingRuleIDIfAbsent_Nil(t *testing.T) {
	SetRecordingRuleIDIfAbsent(nil, "rr-42") // should not panic
}

func TestClearRecordingRuleID(t *testing.T) {
	labels := map[string]string{"dash0.com/id": "rr-42"}
	rule := &RecordingRule{Metadata: PrometheusRuleMetadata{Labels: &labels}}
	ClearRecordingRuleID(rule)
	if GetRecordingRuleID(rule) != "" {
		t.Error("ID should be cleared")
	}
}

func TestClearRecordingRuleID_AlreadyNil(t *testing.T) {
	rule := &RecordingRule{Metadata: PrometheusRuleMetadata{Name: "test"}}
	ClearRecordingRuleID(rule) // should not panic
}

func TestClearRecordingRuleID_Nil(t *testing.T) {
	ClearRecordingRuleID(nil) // should not panic
}

func TestGetRecordingRuleName(t *testing.T) {
	if got := GetRecordingRuleName(&RecordingRule{Metadata: PrometheusRuleMetadata{Name: "my-rule"}}); got != "my-rule" {
		t.Errorf("got %q, want %q", got, "my-rule")
	}
	if got := GetRecordingRuleName(&RecordingRule{}); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := GetRecordingRuleName(nil); got != "" {
		t.Errorf("got %q, want empty for nil", got)
	}
}

func TestGetRecordingRuleDataset(t *testing.T) {
	tests := []struct {
		name string
		rule *RecordingRule
		want string
	}{
		{"nil rule", nil, ""},
		{"nil labels", &RecordingRule{Metadata: PrometheusRuleMetadata{Name: "test"}}, ""},
		{"with dataset", func() *RecordingRule {
			labels := map[string]string{"dash0.com/dataset": "production"}
			return &RecordingRule{Metadata: PrometheusRuleMetadata{Labels: &labels}}
		}(), "production"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRecordingRuleDataset(tt.rule); got != tt.want {
				t.Errorf("GetRecordingRuleDataset() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetRecordingRuleDataset(t *testing.T) {
	rule := &RecordingRule{Metadata: PrometheusRuleMetadata{Name: "test"}}
	SetRecordingRuleDataset(rule, "production")
	if GetRecordingRuleDataset(rule) != "production" {
		t.Error("expected dataset to be set to production")
	}
}

func TestSetRecordingRuleDataset_Nil(t *testing.T) {
	SetRecordingRuleDataset(nil, "production") // should not panic
}

func TestCreateRecordingRule_201(t *testing.T) {
	rule := RecordingRule{
		ApiVersion: "monitoring.coreos.com/v1",
		Kind:       "PrometheusRule",
		Metadata: PrometheusRuleMetadata{
			Name: "test-recording-rule",
		},
		Spec: PrometheusRuleSpec{
			Groups: []PrometheusRuleGroup{
				{
					Name: "test-group",
				},
			},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		labels := map[string]string{"dash0.com/id": "rr-123"}
		response := rule
		response.Metadata.Labels = &labels
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	got, err := client.CreateRecordingRule(context.Background(), &rule, nil)
	if err != nil {
		t.Fatalf("CreateRecordingRule failed: %v", err)
	}

	if got.Metadata.Name != "test-recording-rule" {
		t.Errorf("expected name %q, got %q", "test-recording-rule", got.Metadata.Name)
	}
	if GetRecordingRuleID(got) != "rr-123" {
		t.Errorf("expected ID %q, got %q", "rr-123", GetRecordingRuleID(got))
	}
}

func TestListRecordingRules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/recording-rules" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]RecordingRule{
			{
				ApiVersion: "monitoring.coreos.com/v1",
				Kind:       "PrometheusRule",
				Metadata:   PrometheusRuleMetadata{Name: "rule-1"},
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	rules, err := client.ListRecordingRules(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListRecordingRules failed: %v", err)
	}

	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Metadata.Name != "rule-1" {
		t.Errorf("expected name %q, got %q", "rule-1", rules[0].Metadata.Name)
	}
}

func TestGetRecordingRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(RecordingRule{
			ApiVersion: "monitoring.coreos.com/v1",
			Kind:       "PrometheusRule",
			Metadata:   PrometheusRuleMetadata{Name: "rule-1"},
		})
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	rule, err := client.GetRecordingRule(context.Background(), "rr-123", nil)
	if err != nil {
		t.Fatalf("GetRecordingRule failed: %v", err)
	}

	if rule.Metadata.Name != "rule-1" {
		t.Errorf("expected name %q, got %q", "rule-1", rule.Metadata.Name)
	}
}

func TestUpdateRecordingRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(RecordingRule{
			ApiVersion: "monitoring.coreos.com/v1",
			Kind:       "PrometheusRule",
			Metadata:   PrometheusRuleMetadata{Name: "updated-rule"},
		})
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	rule := &RecordingRule{
		ApiVersion: "monitoring.coreos.com/v1",
		Kind:       "PrometheusRule",
		Metadata:   PrometheusRuleMetadata{Name: "updated-rule"},
	}
	got, err := client.UpdateRecordingRule(context.Background(), "rr-123", rule, nil)
	if err != nil {
		t.Fatalf("UpdateRecordingRule failed: %v", err)
	}

	if got.Metadata.Name != "updated-rule" {
		t.Errorf("expected name %q, got %q", "updated-rule", got.Metadata.Name)
	}
}

func TestUpdateRecordingRule_201(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(RecordingRule{
			ApiVersion: "monitoring.coreos.com/v1",
			Kind:       "PrometheusRule",
			Metadata:   PrometheusRuleMetadata{Name: "updated-rule"},
		})
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	rule := &RecordingRule{
		ApiVersion: "monitoring.coreos.com/v1",
		Kind:       "PrometheusRule",
		Metadata:   PrometheusRuleMetadata{Name: "updated-rule"},
	}
	got, err := client.UpdateRecordingRule(context.Background(), "rr-123", rule, nil)
	if err != nil {
		t.Fatalf("UpdateRecordingRule failed: %v", err)
	}

	if got.Metadata.Name != "updated-rule" {
		t.Errorf("expected name %q, got %q", "updated-rule", got.Metadata.Name)
	}
}

func TestCreateRecordingRule_Dataset(t *testing.T) {
	rule := RecordingRule{
		ApiVersion: "monitoring.coreos.com/v1",
		Kind:       "PrometheusRule",
		Metadata:   PrometheusRuleMetadata{Name: "test-recording-rule"},
		Spec:       PrometheusRuleSpec{Groups: []PrometheusRuleGroup{{Name: "test-group"}}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parsedURL, err := url.Parse(r.RequestURI)
		if err != nil {
			t.Fatalf("failed to parse request URI: %v", err)
		}
		if got := parsedURL.Query().Get("dataset"); got != "iac-tests" {
			t.Errorf("expected dataset query param %q, got %q", "iac-tests", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(rule)
	}))
	defer server.Close()

	client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ds := "iac-tests"
	_, err = client.CreateRecordingRule(context.Background(), &rule, &ds)
	if err != nil {
		t.Fatalf("CreateRecordingRule failed: %v", err)
	}
}

func TestUpdateRecordingRule_Dataset(t *testing.T) {
	rule := RecordingRule{
		ApiVersion: "monitoring.coreos.com/v1",
		Kind:       "PrometheusRule",
		Metadata:   PrometheusRuleMetadata{Name: "updated-rule"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parsedURL, err := url.Parse(r.RequestURI)
		if err != nil {
			t.Fatalf("failed to parse request URI: %v", err)
		}
		if got := parsedURL.Query().Get("dataset"); got != "iac-tests" {
			t.Errorf("expected dataset query param %q, got %q", "iac-tests", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(rule)
	}))
	defer server.Close()

	client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ds := "iac-tests"
	_, err = client.UpdateRecordingRule(context.Background(), "rr-123", &rule, &ds)
	if err != nil {
		t.Fatalf("UpdateRecordingRule failed: %v", err)
	}
}

func TestListRecordingRules_Dataset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parsedURL, err := url.Parse(r.RequestURI)
		if err != nil {
			t.Fatalf("failed to parse request URI: %v", err)
		}
		if got := parsedURL.Query().Get("dataset"); got != "iac-tests" {
			t.Errorf("expected dataset query param %q, got %q", "iac-tests", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]RecordingRule{})
	}))
	defer server.Close()

	client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ds := "iac-tests"
	_, err = client.ListRecordingRules(context.Background(), &ds)
	if err != nil {
		t.Fatalf("ListRecordingRules failed: %v", err)
	}
}

func TestGetRecordingRule_Dataset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parsedURL, err := url.Parse(r.RequestURI)
		if err != nil {
			t.Fatalf("failed to parse request URI: %v", err)
		}
		if got := parsedURL.Query().Get("dataset"); got != "iac-tests" {
			t.Errorf("expected dataset query param %q, got %q", "iac-tests", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(RecordingRule{
			ApiVersion: "monitoring.coreos.com/v1",
			Kind:       "PrometheusRule",
			Metadata:   PrometheusRuleMetadata{Name: "rule-1"},
		})
	}))
	defer server.Close()

	client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ds := "iac-tests"
	_, err = client.GetRecordingRule(context.Background(), "rr-123", &ds)
	if err != nil {
		t.Fatalf("GetRecordingRule failed: %v", err)
	}
}

func TestDeleteRecordingRule_Dataset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parsedURL, err := url.Parse(r.RequestURI)
		if err != nil {
			t.Fatalf("failed to parse request URI: %v", err)
		}
		if got := parsedURL.Query().Get("dataset"); got != "iac-tests" {
			t.Errorf("expected dataset query param %q, got %q", "iac-tests", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ds := "iac-tests"
	err = client.DeleteRecordingRule(context.Background(), "rr-123", &ds)
	if err != nil {
		t.Fatalf("DeleteRecordingRule failed: %v", err)
	}
}

func TestDeleteRecordingRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.DeleteRecordingRule(context.Background(), "rr-123", nil)
	if err != nil {
		t.Fatalf("DeleteRecordingRule failed: %v", err)
	}
}

func TestDeleteRecordingRule_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.DeleteRecordingRule(context.Background(), "rr-123", nil)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true, got false")
	}
}

func TestGetRecordingRule_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.GetRecordingRule(context.Background(), "rr-123", nil)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true, got false")
	}
}

func TestCreateRecordingRule_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid"})
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	rule := &RecordingRule{
		ApiVersion: "monitoring.coreos.com/v1",
		Kind:       "PrometheusRule",
		Metadata:   PrometheusRuleMetadata{Name: "test"},
	}
	_, err = client.CreateRecordingRule(context.Background(), rule, nil)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestListRecordingRules_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "forbidden"})
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.ListRecordingRules(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}
