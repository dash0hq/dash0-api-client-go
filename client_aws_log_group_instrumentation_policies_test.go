package dash0

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const awsLogGroupPolicyPath = "/api/aws-log-group-instrumentation-policies/" + testIntegrationID

const awsLogGroupPolicyJSON = `{"apiVersion":"dash0.com/v1alpha1","kind":"Dash0AwsLogGroupInstrumentationPolicy","metadata":{"name":"491cb653-954b-4d0b-a84c-4b426b562938","labels":{"dash0.com/id":"491cb653-954b-4d0b-a84c-4b426b562938","dash0.com/dataset":"default","dash0.com/version":"7","dash0.com/origin":"tf_policy","dash0.com/source":"terraform"},"annotations":{"dash0.com/created-at":"2026-11-17T08:00:00Z","dash0.com/updated-at":"2026-11-17T09:00:00Z"}},"spec":{"onNoLongerMatching":"keep","rules":[{"conditions":[{"key":"dash0.cloud.tags.team","operator":"is","value":"platform"},{"key":"dash0.cloud.tags.env","operator":"is_one_of","values":["prod","staging"]}]}]}}`

const awsLogGroupPolicyDeletedJSON = `{"apiVersion":"dash0.com/v1alpha1","kind":"Dash0AwsLogGroupInstrumentationPolicy","metadata":{"name":"491cb653-954b-4d0b-a84c-4b426b562938","labels":{"dash0.com/id":"491cb653-954b-4d0b-a84c-4b426b562938","dash0.com/dataset":"default","dash0.com/version":"8","dash0.com/source":"ui"},"annotations":{"dash0.com/created-at":"2026-11-17T08:00:00Z","dash0.com/updated-at":"2026-11-17T10:00:00Z","dash0.com/deleted-at":"2026-11-17T10:00:00Z","dash0.com/deletion-disposition":"remove"}},"spec":{"onNoLongerMatching":"keep","rules":[{"conditions":[{"key":"dash0.cloud.tags.team","operator":"is","value":"platform"}]}]}}`

func writeAPIErrorWithType(w http.ResponseWriter, status int, errorType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{"code": status, "message": "rejected", "type": errorType},
	})
}

func TestGetAwsLogGroupInstrumentationPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "method", r.Method, http.MethodGet)
		assertEqual(t, "path", r.URL.Path, awsLogGroupPolicyPath)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, awsLogGroupPolicyJSON)
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL).GetAwsLogGroupInstrumentationPolicy(context.Background(), testIntegrationID)
	if err != nil {
		t.Fatalf("GetAwsLogGroupInstrumentationPolicy failed: %v", err)
	}
	assertEqual(t, "origin", GetAwsLogGroupInstrumentationPolicyOrigin(got), "tf_policy")
	assertEqual(t, "id", GetAwsLogGroupInstrumentationPolicyID(got), testIntegrationID)
	revision, ok := GetAwsLogGroupInstrumentationPolicyRevision(got)
	if !ok || revision != 7 {
		t.Errorf("revision = %d, %v, want 7, true", revision, ok)
	}
	if IsAwsLogGroupInstrumentationPolicyDeleted(got) {
		t.Error("active policy reported as deleted")
	}
	conditions := got.Spec.Rules[0].Conditions
	if len(conditions) != 2 || conditions[1].Operator != AwsLogGroupPolicyOperatorIsOneOf || conditions[1].Values == nil || len(*conditions[1].Values) != 2 {
		t.Errorf("unexpected conditions: %+v", conditions)
	}
}

func TestGetAwsLogGroupInstrumentationPolicy_Deleted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, awsLogGroupPolicyDeletedJSON)
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL).GetAwsLogGroupInstrumentationPolicy(context.Background(), testIntegrationID)
	if err != nil {
		t.Fatalf("GetAwsLogGroupInstrumentationPolicy failed: %v", err)
	}
	if !IsAwsLogGroupInstrumentationPolicyDeleted(got) {
		t.Error("deleted policy not reported as deleted")
	}
	assertPtrEqual(t, "deletion disposition", got.Metadata.Annotations.Dash0ComdeletionDisposition, AwsLogGroupPolicyDispositionRemove)
	assertEqual(t, "origin", GetAwsLogGroupInstrumentationPolicyOrigin(got), "")
}

func TestGetAwsLogGroupInstrumentationPolicy_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusNotFound, "log-group policy not found")
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).GetAwsLogGroupInstrumentationPolicy(context.Background(), testIntegrationID)
	if !IsNotFound(err) {
		t.Fatalf("expected not found APIError, got %v", err)
	}
}

func TestUpdateAwsLogGroupInstrumentationPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "method", r.Method, http.MethodPut)
		assertEqual(t, "path", r.URL.Path, awsLogGroupPolicyPath)
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
			return
		}
		var body struct {
			Metadata struct {
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
			Spec struct {
				Rules []struct {
					Conditions json.RawMessage `json:"conditions"`
				} `json:"rules"`
			} `json:"spec"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		assertEqual(t, "version label", body.Metadata.Labels["dash0.com/version"], "7")
		assertEqual(t, "origin label", body.Metadata.Labels["dash0.com/origin"], "tf_policy")
		if len(body.Spec.Rules) != 1 {
			t.Errorf("rules = %d, want 1", len(body.Spec.Rules))
			return
		}
		assertEqual(t, "nil conditions sent as", string(body.Spec.Rules[0].Conditions), "[]")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, awsLogGroupPolicyJSON)
	}))
	defer server.Close()

	policy := &AwsLogGroupInstrumentationPolicy{
		ApiVersion: AwsLogGroupPolicyApiVersionDash0Comv1alpha1,
		Kind:       Dash0AwsLogGroupInstrumentationPolicy,
		Spec:       AwsLogGroupPolicySpec{Rules: []AwsLogGroupPolicyRule{{}}},
	}
	SetAwsLogGroupInstrumentationPolicyOrigin(policy, "tf_policy")
	SetAwsLogGroupInstrumentationPolicyRevision(policy, 7)

	if _, err := newTestClient(t, server.URL).UpdateAwsLogGroupInstrumentationPolicy(context.Background(), testIntegrationID, policy); err != nil {
		t.Fatalf("UpdateAwsLogGroupInstrumentationPolicy failed: %v", err)
	}
	if policy.Spec.Rules[0].Conditions != nil {
		t.Error("UpdateAwsLogGroupInstrumentationPolicy mutated the caller's policy")
	}
}

func TestUpdateAwsLogGroupInstrumentationPolicy_Rejected(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		errorType      string
		wantSuperseded bool
		wantConflict   bool
	}{
		{name: "stale revision", status: http.StatusBadRequest, errorType: "superseded_version", wantSuperseded: true},
		{name: "invalid rule", status: http.StatusBadRequest, errorType: ""},
		{name: "origin owned by another client", status: http.StatusConflict, errorType: "origin_conflict", wantConflict: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeAPIErrorWithType(w, tt.status, tt.errorType)
			}))
			defer server.Close()

			policy := &AwsLogGroupInstrumentationPolicy{Spec: AwsLogGroupPolicySpec{Rules: []AwsLogGroupPolicyRule{}}}
			_, err := newTestClient(t, server.URL).UpdateAwsLogGroupInstrumentationPolicy(context.Background(), testIntegrationID, policy)
			if got := IsSupersededVersion(err); got != tt.wantSuperseded {
				t.Errorf("IsSupersededVersion = %v, want %v (err: %v)", got, tt.wantSuperseded, err)
			}
			if got := IsConflict(err); got != tt.wantConflict {
				t.Errorf("IsConflict = %v, want %v (err: %v)", got, tt.wantConflict, err)
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected APIError, got %v", err)
			}
			assertEqual(t, "type", apiErr.Type, tt.errorType)
		})
	}
}

func TestDeleteAwsLogGroupInstrumentationPolicy(t *testing.T) {
	revision := int64(7)
	tests := []struct {
		name             string
		disposition      AwsLogGroupPolicyDisposition
		expectedRevision *int64
		wantQuery        string
	}{
		{name: "keep without revision", disposition: AwsLogGroupPolicyDispositionKeep, wantQuery: "disposition=keep"},
		{name: "remove fenced by revision", disposition: AwsLogGroupPolicyDispositionRemove, expectedRevision: &revision, wantQuery: "disposition=remove&expectedRevision=7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assertEqual(t, "method", r.Method, http.MethodDelete)
				assertEqual(t, "path", r.URL.Path, awsLogGroupPolicyPath)
				assertEqual(t, "query", r.URL.RawQuery, tt.wantQuery)
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, awsLogGroupPolicyDeletedJSON)
			}))
			defer server.Close()

			got, err := newTestClient(t, server.URL).DeleteAwsLogGroupInstrumentationPolicy(context.Background(), testIntegrationID, tt.disposition, tt.expectedRevision)
			if err != nil {
				t.Fatalf("DeleteAwsLogGroupInstrumentationPolicy failed: %v", err)
			}
			if !IsAwsLogGroupInstrumentationPolicyDeleted(got) {
				t.Error("returned policy is not marked deleted")
			}
		})
	}
}

func TestDeleteAwsLogGroupInstrumentationPolicy_RequiresDisposition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL)
	}))
	defer server.Close()

	for _, disposition := range []AwsLogGroupPolicyDisposition{"", "delete"} {
		if _, err := newTestClient(t, server.URL).DeleteAwsLogGroupInstrumentationPolicy(context.Background(), testIntegrationID, disposition, nil); err == nil {
			t.Errorf("disposition %q: expected an error", disposition)
		}
	}
}

func TestDeleteAwsLogGroupInstrumentationPolicy_DispatchFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIErrorWithType(w, http.StatusServiceUnavailable, "reconciliation_dispatch_failed")
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).DeleteAwsLogGroupInstrumentationPolicy(context.Background(), testIntegrationID, AwsLogGroupPolicyDispositionRemove, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	assertEqual(t, "type", apiErr.Type, "reconciliation_dispatch_failed")
}

func TestStripAwsLogGroupInstrumentationPolicyServerFields(t *testing.T) {
	var policy AwsLogGroupInstrumentationPolicy
	if err := json.Unmarshal([]byte(awsLogGroupPolicyJSON), &policy); err != nil {
		t.Fatal(err)
	}
	StripAwsLogGroupInstrumentationPolicyServerFields(&policy)

	if policy.Metadata.Name != nil || policy.Metadata.Annotations != nil {
		t.Errorf("name or annotations kept: %+v", policy.Metadata)
	}
	labels := policy.Metadata.Labels
	if labels.Dash0Comid != nil || labels.Dash0Comdataset != nil || labels.Dash0Comsource != nil || labels.Dash0Comversion != nil {
		t.Errorf("server labels kept: %+v", labels)
	}
	assertEqual(t, "origin", GetAwsLogGroupInstrumentationPolicyOrigin(&policy), "tf_policy")
	if _, ok := GetAwsLogGroupInstrumentationPolicyRevision(&policy); ok {
		t.Error("revision kept, so an Update would still be fenced")
	}
	if len(policy.Spec.Rules) != 1 {
		t.Error("spec changed")
	}
}

func TestAwsLogGroupInstrumentationPolicyHelpers_NilAndMalformed(t *testing.T) {
	StripAwsLogGroupInstrumentationPolicyServerFields(nil)
	SetAwsLogGroupInstrumentationPolicyOrigin(nil, "x")
	SetAwsLogGroupInstrumentationPolicyRevision(nil, 1)
	if IsAwsLogGroupInstrumentationPolicyDeleted(nil) || GetAwsLogGroupInstrumentationPolicyID(nil) != "" {
		t.Error("nil policy reported state")
	}
	bad := "abc"
	policy := &AwsLogGroupInstrumentationPolicy{Metadata: AwsLogGroupPolicyMetadata{Labels: &AwsLogGroupPolicyLabels{Dash0Comversion: &bad}}}
	if _, ok := GetAwsLogGroupInstrumentationPolicyRevision(policy); ok {
		t.Error("non-numeric revision parsed")
	}
	SetAwsLogGroupInstrumentationPolicyRevision(policy, 0)
	if revision, ok := GetAwsLogGroupInstrumentationPolicyRevision(policy); !ok || revision != 0 {
		t.Errorf("revision = %d, %v, want 0, true", revision, ok)
	}
}
