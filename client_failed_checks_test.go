package dash0

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const failedChecksFixtureSuccess = `{
    "cursors": {
        "before": "before-13208543997169216472"
    },
    "executionTime": "2026-06-19T17:38:09.330356322Z",
    "issues": [
        {
            "affectedResourceSummaries": [{}],
            "annotations": [],
            "checkRule": {
                "descriptionTemplate": "Deliberately block deployments",
                "id": "482a551a-ca38-4f22-b8c0-2a69276a7169",
                "modes": [],
                "name": "BLOCK DEPLOYMENTS",
                "summaryTemplate": "BLOCK DEPLOYMENTS due to XYZ",
                "version": 34
            },
            "description": "Deliberately block deployments",
            "earliestEvaluatedTime": "2026-06-19T13:10:54.68453466Z",
            "end": "1970-01-01T00:00:00Z",
            "id": "03bd9db7-5ce1-4d7d-a9ad-5fc22bc27601",
            "instanceStatus": "critical",
            "issueIdentifier": "13208543997169216472",
            "labels": [
                {"key": "priority", "value": {"stringValue": "p1"}}
            ],
            "start": "2026-06-19T13:10:54.68453466Z",
            "summary": "BLOCK DEPLOYMENTS due to XYZ"
        }
    ],
    "timeRange": {
        "from": "2026-06-19T17:23:09.330356322Z",
        "to": "2026-06-19T17:38:09.330356322Z"
    }
}`

func TestGetFailedChecks_Success(t *testing.T) {
	var receivedBody []byte
	var receivedPath string
	var receivedMethod string
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedHeaders = r.Header.Clone()
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, failedChecksFixtureSuccess)
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	dataset := "production"
	resp, err := client.GetFailedChecks(context.Background(), &GetFailedChecksRequest{
		TimeRange:  TimeReferenceRange{From: "now-1h", To: "now"},
		Dataset:    &dataset,
		Pagination: &CursorPagination{Limit: Int64(50)},
	})
	if err != nil {
		t.Fatalf("GetFailedChecks failed: %v", err)
	}

	if receivedMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", receivedMethod)
	}
	if receivedPath != failedChecksAPIPath {
		t.Errorf("path = %q, want %q", receivedPath, failedChecksAPIPath)
	}
	if got := receivedHeaders.Get("Authorization"); got != "Bearer auth_test123" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer auth_test123")
	}
	if got := receivedHeaders.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := receivedHeaders.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q, want application/json", got)
	}

	var sent GetFailedChecksRequest
	if err := json.Unmarshal(receivedBody, &sent); err != nil {
		t.Fatalf("failed to unmarshal sent body: %v", err)
	}
	if sent.Dataset == nil || *sent.Dataset != "production" {
		t.Errorf("Dataset = %v, want production", sent.Dataset)
	}

	if len(resp.Issues) != 1 {
		t.Fatalf("Issues = %d, want 1", len(resp.Issues))
	}
	issue := resp.Issues[0]
	if issue.Id != "03bd9db7-5ce1-4d7d-a9ad-5fc22bc27601" {
		t.Errorf("Id = %q", issue.Id)
	}
	if issue.InstanceStatus != IssueInstanceStatusCritical {
		t.Errorf("InstanceStatus = %q, want critical", issue.InstanceStatus)
	}
	if issue.CheckRule.Name != "BLOCK DEPLOYMENTS" {
		t.Errorf("CheckRule.Name = %q", issue.CheckRule.Name)
	}
	if len(issue.Labels) != 1 || issue.Labels[0].Key != "priority" {
		t.Errorf("Labels = %+v", issue.Labels)
	}
	if issue.Labels[0].Value.StringValue == nil || *issue.Labels[0].Value.StringValue != "p1" {
		t.Errorf("Labels[0].Value.StringValue = %v", issue.Labels[0].Value.StringValue)
	}
	if resp.Cursors == nil || resp.Cursors.Before == nil || *resp.Cursors.Before != "before-13208543997169216472" {
		t.Errorf("Cursors.Before = %+v, want before-13208543997169216472", resp.Cursors)
	}
}

func TestGetFailedChecks_NotConfigured(t *testing.T) {
	c := &client{}
	_, err := c.GetFailedChecks(context.Background(), &GetFailedChecksRequest{})
	if err != ErrAPINotConfigured {
		t.Errorf("err = %v, want ErrAPINotConfigured", err)
	}
}

func TestGetFailedChecks_NilRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("server should not be called")
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if _, err := client.GetFailedChecks(context.Background(), nil); err == nil {
		t.Error("expected error for nil request, got nil")
	}
}

func TestGetFailedChecks_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":{"code":401,"message":"unauthorized"}}`)
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.GetFailedChecks(context.Background(), &GetFailedChecksRequest{
		TimeRange: TimeReferenceRange{From: "now-1h", To: "now"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsUnauthorized(err) {
		t.Errorf("expected IsUnauthorized, got %v", err)
	}
}

func TestGetFailedChecksIter_Pagination(t *testing.T) {
	page1 := `{
        "cursors": {"after": "cursor-page-2"},
        "issues": [
            {"id": "i1", "checkRule": {"id": "r1", "name": "rule-1", "version": 1}, "instanceStatus": "critical", "summary": "s1", "description": "d1", "start": "2026-06-19T13:10:54Z", "end": "1970-01-01T00:00:00Z", "labels": []}
        ]
    }`
	page2 := `{
        "cursors": {},
        "issues": [
            {"id": "i2", "checkRule": {"id": "r2", "name": "rule-2", "version": 1}, "instanceStatus": "degraded", "summary": "s2", "description": "d2", "start": "2026-06-19T13:11:00Z", "end": "1970-01-01T00:00:00Z", "labels": []}
        ]
    }`

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req GetFailedChecksRequest
		_ = json.Unmarshal(body, &req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		calls++
		if req.Pagination != nil && req.Pagination.Cursor != nil {
			if *req.Pagination.Cursor != "cursor-page-2" {
				t.Errorf("cursor = %q, want %q", *req.Pagination.Cursor, "cursor-page-2")
			}
			_, _ = io.WriteString(w, page2)
		} else {
			_, _ = io.WriteString(w, page1)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	iter := client.GetFailedChecksIter(context.Background(), &GetFailedChecksRequest{
		TimeRange: TimeReferenceRange{From: "now-1h", To: "now"},
	})

	var ids []string
	for iter.Next() {
		ids = append(ids, iter.Current().Id)
	}
	if err := iter.Err(); err != nil {
		t.Fatalf("iter error: %v", err)
	}
	if len(ids) != 2 || ids[0] != "i1" || ids[1] != "i2" {
		t.Errorf("ids = %v, want [i1 i2]", ids)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

// TestGetFailedChecksUsesTheProviderToken guards against a hand-built request
// setting its own Authorization header.
//
// authTransport owns the header; this endpoint builds its request by hand, and
// for a client created with WithAuthTokenProvider the config's static token is
// empty, so any local Header.Set here would put "Bearer " on the wire if the
// signing layer ever stopped overwriting it.
func TestGetFailedChecksUsesTheProviderToken(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"failedChecks":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthTokenProvider(StaticAuthTokenProvider("dash0_at_from-provider")),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	if _, err := client.GetFailedChecks(context.Background(), &GetFailedChecksRequest{
		TimeRange: TimeReferenceRange{From: "now-1h", To: "now"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Authorization", gotAuth, "Bearer dash0_at_from-provider")
}
