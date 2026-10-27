package dash0

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testIntegrationID = "491cb653-954b-4d0b-a84c-4b426b562938"

// Captured from the dev API: GET after a PUT by Terraform.
const awsMetricScrapeConfigJSON = `{"apiVersion":"dash0.com/v1alpha1","kind":"Dash0AwsMetricScrapeConfig","metadata":{"name":"prod-account","labels":{"dash0.com/id":"491cb653-954b-4d0b-a84c-4b426b562938","dash0.com/dataset":"default","dash0.com/origin":"tf_clientcheck","dash0.com/source":"terraform"},"annotations":{"dash0.com/created-at":"2026-10-27T10:00:00Z","dash0.com/updated-at":"2026-10-27T10:00:00Z"}},"spec":{"namespaces":[{"namespace":"AWS/Lambda","metrics":["Errors","Invocations"]},{"namespace":"AWS/SQS","metrics":["ApproximateAgeOfOldestMessage","NumberOfMessagesSent"]}]}}`

// Captured from the dev API: GET after DELETE.
const awsMetricScrapeConfigClearedJSON = `{"apiVersion":"dash0.com/v1alpha1","kind":"Dash0AwsMetricScrapeConfig","metadata":{"name":"prod-account","labels":{"dash0.com/id":"491cb653-954b-4d0b-a84c-4b426b562938","dash0.com/dataset":"default","dash0.com/source":"ui"}},"spec":{"namespaces":[]}}`

func awsMetricScrapeConfigServer(t *testing.T, wantMethod string, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != wantMethod {
			t.Errorf("method = %s, want %s", r.Method, wantMethod)
		}
		assertEqual(t, "path", r.URL.Path, "/api/aws-metric-scrape-configs/"+testIntegrationID)
		assertEqual(t, "query", r.URL.RawQuery, "")
		if status >= 400 {
			writeAPIError(w, status, "boom")
			return
		}
		if body != "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestGetAwsMetricScrapeConfig(t *testing.T) {
	server := awsMetricScrapeConfigServer(t, http.MethodGet, http.StatusOK, awsMetricScrapeConfigJSON)

	got, err := newTestClient(t, server.URL).GetAwsMetricScrapeConfig(context.Background(), testIntegrationID)
	if err != nil {
		t.Fatalf("GetAwsMetricScrapeConfig failed: %v", err)
	}
	assertPtrEqual(t, "name", got.Metadata.Name, "prod-account")
	assertEqual(t, "origin", GetAwsMetricScrapeConfigOrigin(got), "tf_clientcheck")
	assertPtrEqual(t, "source", got.Metadata.Labels.Dash0Comsource, CrdSource("terraform"))
	assertPtrEqual(t, "created-at", got.Metadata.Annotations.Dash0ComcreatedAt, time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC))
	if len(got.Spec.Namespaces) != 2 || got.Spec.Namespaces[1].Namespace != "AWS/SQS" || len(got.Spec.Namespaces[1].Metrics) != 2 {
		t.Errorf("unexpected spec: %+v", got.Spec)
	}
}

func TestGetAwsMetricScrapeConfig_Cleared(t *testing.T) {
	server := awsMetricScrapeConfigServer(t, http.MethodGet, http.StatusOK, awsMetricScrapeConfigClearedJSON)

	got, err := newTestClient(t, server.URL).GetAwsMetricScrapeConfig(context.Background(), testIntegrationID)
	if err != nil {
		t.Fatalf("GetAwsMetricScrapeConfig failed: %v", err)
	}
	if len(got.Spec.Namespaces) != 0 || got.Metadata.Annotations != nil {
		t.Errorf("expected empty config, got %+v", got)
	}
	assertEqual(t, "origin", GetAwsMetricScrapeConfigOrigin(got), "")
}

func TestGetAwsMetricScrapeConfig_NotFound(t *testing.T) {
	server := awsMetricScrapeConfigServer(t, http.MethodGet, http.StatusNotFound, "")

	_, err := newTestClient(t, server.URL).GetAwsMetricScrapeConfig(context.Background(), testIntegrationID)
	if !IsNotFound(err) {
		t.Fatalf("expected not found APIError, got %v", err)
	}
}

func TestUpdateAwsMetricScrapeConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		assertEqual(t, "path", r.URL.Path, "/api/aws-metric-scrape-configs/"+testIntegrationID)
		var body AwsMetricScrapeConfigDefinition
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		assertEqual(t, "apiVersion", string(body.ApiVersion), "dash0.com/v1alpha1")
		assertEqual(t, "kind", string(body.Kind), "Dash0AwsMetricScrapeConfig")
		assertEqual(t, "origin", GetAwsMetricScrapeConfigOrigin(&body), "tf_clientcheck")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, awsMetricScrapeConfigJSON)
	}))
	defer server.Close()

	cfg := &AwsMetricScrapeConfigDefinition{
		ApiVersion: AwsMetricScrapeConfigApiVersionDash0Comv1alpha1,
		Kind:       Dash0AwsMetricScrapeConfig,
		Spec: AwsMetricScrapeConfigSpec{Namespaces: []AwsMetricScrapeConfigNamespace{
			{Namespace: "AWS/SQS", Metrics: []string{"NumberOfMessagesSent", "ApproximateAgeOfOldestMessage"}},
			{Namespace: "AWS/Lambda", Metrics: []string{"Invocations", "Errors"}},
		}},
	}
	SetAwsMetricScrapeConfigOrigin(cfg, "tf_clientcheck")

	got, err := newTestClient(t, server.URL).UpdateAwsMetricScrapeConfig(context.Background(), testIntegrationID, cfg)
	if err != nil {
		t.Fatalf("UpdateAwsMetricScrapeConfig failed: %v", err)
	}
	assertEqual(t, "first namespace", got.Spec.Namespaces[0].Namespace, "AWS/Lambda")
}

func TestUpdateAwsMetricScrapeConfig_Errors(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusPreconditionFailed} {
		server := awsMetricScrapeConfigServer(t, http.MethodPut, status, "")
		cfg := &AwsMetricScrapeConfigDefinition{ApiVersion: AwsMetricScrapeConfigApiVersionDash0Comv1alpha1, Kind: Dash0AwsMetricScrapeConfig}

		_, err := newTestClient(t, server.URL).UpdateAwsMetricScrapeConfig(context.Background(), testIntegrationID, cfg)
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != status {
			t.Errorf("status %d: expected APIError, got %v", status, err)
		}
	}
}

func TestUpdateAwsMetricScrapeConfig_NilConfig(t *testing.T) {
	c := newTestClient(t, "http://127.0.0.1:1")
	if _, err := c.UpdateAwsMetricScrapeConfig(context.Background(), testIntegrationID, nil); err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestDeleteAwsMetricScrapeConfig(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		server := awsMetricScrapeConfigServer(t, http.MethodDelete, status, "")
		if err := newTestClient(t, server.URL).DeleteAwsMetricScrapeConfig(context.Background(), testIntegrationID); err != nil {
			t.Errorf("status %d: DeleteAwsMetricScrapeConfig failed: %v", status, err)
		}
	}
}

func TestDeleteAwsMetricScrapeConfig_Forbidden(t *testing.T) {
	server := awsMetricScrapeConfigServer(t, http.MethodDelete, http.StatusForbidden, "")
	err := newTestClient(t, server.URL).DeleteAwsMetricScrapeConfig(context.Background(), testIntegrationID)
	if !IsForbidden(err) {
		t.Fatalf("expected forbidden APIError, got %v", err)
	}
}

func TestAwsMetricScrapeConfig_InvalidIntegrationID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()
	c := newTestClient(t, server.URL)
	ctx := context.Background()
	cfg := &AwsMetricScrapeConfigDefinition{}

	if _, err := c.GetAwsMetricScrapeConfig(ctx, "not-a-uuid"); err == nil {
		t.Error("Get: expected error")
	}
	if _, err := c.UpdateAwsMetricScrapeConfig(ctx, "not-a-uuid", cfg); err == nil {
		t.Error("Update: expected error")
	}
	if err := c.DeleteAwsMetricScrapeConfig(ctx, "not-a-uuid"); err == nil {
		t.Error("Delete: expected error")
	}
}

func TestStripAwsMetricScrapeConfigServerFields(t *testing.T) {
	var cfg AwsMetricScrapeConfigDefinition
	if err := json.Unmarshal([]byte(awsMetricScrapeConfigJSON), &cfg); err != nil {
		t.Fatal(err)
	}

	StripAwsMetricScrapeConfigServerFields(&cfg)

	if cfg.Metadata.Annotations != nil {
		t.Error("Annotations should be nil")
	}
	labels := cfg.Metadata.Labels
	if labels.Dash0Comid != nil || labels.Dash0Comdataset != nil || labels.Dash0Comsource != nil {
		t.Errorf("server labels should be nil: %+v", labels)
	}
	assertEqual(t, "origin", GetAwsMetricScrapeConfigOrigin(&cfg), "tf_clientcheck")
}

func TestStripAwsMetricScrapeConfigServerFields_DropsEmptyLabels(t *testing.T) {
	var cfg AwsMetricScrapeConfigDefinition
	if err := json.Unmarshal([]byte(awsMetricScrapeConfigClearedJSON), &cfg); err != nil {
		t.Fatal(err)
	}

	StripAwsMetricScrapeConfigServerFields(&cfg)

	if cfg.Metadata.Labels != nil {
		t.Errorf("Labels should be nil, got %+v", cfg.Metadata.Labels)
	}
}

func TestSetAwsMetricScrapeConfigOrigin(t *testing.T) {
	cfg := &AwsMetricScrapeConfigDefinition{}
	SetAwsMetricScrapeConfigOrigin(cfg, "tf_a")
	assertEqual(t, "origin", GetAwsMetricScrapeConfigOrigin(cfg), "tf_a")
	SetAwsMetricScrapeConfigOrigin(cfg, "tf_b")
	assertEqual(t, "origin", GetAwsMetricScrapeConfigOrigin(cfg), "tf_b")
}

func TestAwsMetricScrapeConfigHelpers_Nil(t *testing.T) {
	StripAwsMetricScrapeConfigServerFields(nil)
	SetAwsMetricScrapeConfigOrigin(nil, "tf_a")
	assertEqual(t, "origin", GetAwsMetricScrapeConfigOrigin(nil), "")

	empty := &AwsMetricScrapeConfigDefinition{}
	StripAwsMetricScrapeConfigServerFields(empty)
	assertEqual(t, "origin", GetAwsMetricScrapeConfigOrigin(empty), "")
}
