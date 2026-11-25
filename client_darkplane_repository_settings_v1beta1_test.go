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

const testSettingsID = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"

const repositorySettingsV1Beta1JSON = `{"apiVersion":"dash0.com/v1beta1","kind":"DarkplaneRepositorySettings","metadata":{"name":"acme/api","labels":{"dash0.com/id":"7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f","dash0.com/origin":"tf-acme-api","dash0.com/version":"3"},"annotations":{"dash0.com/created-at":"2026-01-05T09:00:00Z","dash0.com/updated-at":"2026-01-12T10:30:00Z","dash0.com/uses-default-criteria":"true"}},"spec":{"repository":"acme/api","bot":{"mode":"dry_run","trigger":"auto","checks":"wait","ignoredChecks":[],"reverts":"evaluate"},"comments":{"mode":"post","approvalMessage":""},"knowledge":"","scoring":{"threshold":75,"criteria":[{"name":"Blast radius","prompt":"Score how contained the change is","weight":1}]},"paths":{"rules":[{"name":"Docs","match":["docs/**"],"level":"auto_merge","authors":{"mode":"only","ids":["1234567"],"otherwise":"dry_run"}}]},"reviewer":{"trigger":"push","settleDelaySeconds":120,"skipAfterHumanReview":true,"commentRequests":"author_and_writers","automerge":{"review":"optional","acceptedVerdicts":["approve"]},"authors":{"mode":"only","ids":["1234567"]}}}}`

func TestGetDarkplaneRepositorySettingsV1Beta1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		assertEqual(t, "path", r.URL.Path, "/api/darkplane/repository-settings/tf-acme-api")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, repositorySettingsV1Beta1JSON)
	}))
	t.Cleanup(server.Close)

	got, err := newTestClient(t, server.URL).GetDarkplaneRepositorySettingsV1Beta1(context.Background(), "tf-acme-api")
	if err != nil {
		t.Fatalf("GetDarkplaneRepositorySettingsV1Beta1 failed: %v", err)
	}
	assertEqual(t, "id", GetDarkplaneRepositorySettingsV1Beta1ID(got), testSettingsID)
	assertEqual(t, "origin", GetDarkplaneRepositorySettingsV1Beta1Origin(got), "tf-acme-api")
	assertEqual(t, "version", GetDarkplaneRepositorySettingsV1Beta1Version(got), "3")
	assertEqual(t, "repository", got.Spec.Repository, "acme/api")
	assertEqual(t, "reviewer trigger", string(*got.Spec.Reviewer.Trigger), string(RepositorySettingsV1beta1ReviewerTriggerPush))
	assertEqual(t, "path rule otherwise", string(*(*got.Spec.Paths.Rules)[0].Authors.Otherwise), "dry_run")
}

func TestUpdateDarkplaneRepositorySettingsV1Beta1_SendsOnlyWhatThePutReads(t *testing.T) {
	var body map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		assertEqual(t, "path", r.URL.Path, "/api/darkplane/repository-settings/tf-acme-api")
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, repositorySettingsV1Beta1JSON)
	}))
	t.Cleanup(server.Close)

	var settings RepositorySettingsV1beta1Definition
	if err := json.Unmarshal([]byte(repositorySettingsV1Beta1JSON), &settings); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	got, err := newTestClient(t, server.URL).UpdateDarkplaneRepositorySettingsV1Beta1(context.Background(), "tf-acme-api", &settings)
	if err != nil {
		t.Fatalf("UpdateDarkplaneRepositorySettingsV1Beta1 failed: %v", err)
	}
	assertEqual(t, "returned id", GetDarkplaneRepositorySettingsV1Beta1ID(got), testSettingsID)

	assertEqual(t, "apiVersion", string(body["apiVersion"]), `"dash0.com/v1beta1"`)
	assertEqual(t, "kind", string(body["kind"]), `"DarkplaneRepositorySettings"`)
	assertEqual(t, "metadata", string(body["metadata"]), `{"labels":{"dash0.com/origin":"tf-acme-api","dash0.com/version":"3"}}`)
	var spec map[string]json.RawMessage
	if err := json.Unmarshal(body["spec"], &spec); err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	assertEqual(t, "spec.repository", string(spec["repository"]), `"acme/api"`)
}

func TestUpdateDarkplaneRepositorySettingsV1Beta1_OmitsMetadataWithoutLabels(t *testing.T) {
	var body map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, repositorySettingsV1Beta1JSON)
	}))
	t.Cleanup(server.Close)

	settings := &RepositorySettingsV1beta1Definition{Spec: RepositorySettingsV1beta1Spec{Repository: "acme/api"}}
	if _, err := newTestClient(t, server.URL).UpdateDarkplaneRepositorySettingsV1Beta1(context.Background(), testSettingsID, settings); err != nil {
		t.Fatalf("UpdateDarkplaneRepositorySettingsV1Beta1 failed: %v", err)
	}
	if _, ok := body["metadata"]; ok {
		t.Errorf("metadata = %s, want it omitted", body["metadata"])
	}
}

func TestDarkplaneRepositorySettingsV1Beta1_RequireOriginOrID(t *testing.T) {
	c := newTestClient(t, "http://127.0.0.1:1")
	if _, err := c.GetDarkplaneRepositorySettingsV1Beta1(context.Background(), ""); err == nil {
		t.Error("Get with an empty origin or ID succeeded, want an error")
	}
	if _, err := c.UpdateDarkplaneRepositorySettingsV1Beta1(context.Background(), "", &RepositorySettingsV1beta1Definition{}); err == nil {
		t.Error("Update with an empty origin or ID succeeded, want an error")
	}
	if _, err := c.UpdateDarkplaneRepositorySettingsV1Beta1(context.Background(), "tf-acme-api", nil); err == nil {
		t.Error("Update with nil settings succeeded, want an error")
	}
}

func TestDarkplaneRepositorySettingsV1Beta1_ReturnAPIErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"code":400,"message":"Invalid repository settings: spec.repository is 'acme/web', but these settings belong to 'acme/api'."}}`)
	}))
	t.Cleanup(server.Close)

	settings := &RepositorySettingsV1beta1Definition{Spec: RepositorySettingsV1beta1Spec{Repository: "acme/web"}}
	_, err := newTestClient(t, server.URL).UpdateDarkplaneRepositorySettingsV1Beta1(context.Background(), "tf-acme-api", settings)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("err = %v, want a 400 APIError", err)
	}
}

func TestDarkplaneRepositorySettingsV1Beta1Accessors_AcceptNil(t *testing.T) {
	assertEqual(t, "id", GetDarkplaneRepositorySettingsV1Beta1ID(nil), "")
	assertEqual(t, "origin", GetDarkplaneRepositorySettingsV1Beta1Origin(nil), "")
	assertEqual(t, "version", GetDarkplaneRepositorySettingsV1Beta1Version(nil), "")
}
