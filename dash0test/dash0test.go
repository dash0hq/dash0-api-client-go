// Package dash0test provides testing utilities for the Dash0 API client.
package dash0test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	// TestAuthToken is a valid test auth token for use in tests.
	TestAuthToken = "auth_test_token"

	// TestDataset is a test dataset name for use in tests.
	TestDataset = "test-dataset"
)

// MockServer creates a test server that responds with the given data.
// It validates that requests match the expected method and path.
//
// Example:
//
//	server := dash0test.MockServer(t, "GET", "/api/dashboards", http.StatusOK, []dash0.Dashboard{})
//	defer server.Close()
//
//	client, _ := dash0.NewClient(dash0.WithApiUrl(server.URL), dash0.WithAuthToken(dash0test.TestAuthToken))
func MockServer(t *testing.T, method, path string, statusCode int, response interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			t.Errorf("expected method %s, got %s", method, r.Method)
		}
		if r.URL.Path != path {
			t.Errorf("expected path %s, got %s", path, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if response != nil {
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
}

// MockServerFunc creates a test server with a custom handler function.
// Use this when you need more control over the response behavior.
//
// Example:
//
//	server := dash0test.MockServerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    w.Header().Set("x-trace-id", "test-trace-id")
//	    w.WriteHeader(http.StatusOK)
//	    json.NewEncoder(w).Encode(myResponse)
//	})
//	defer server.Close()
func MockServerFunc(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}
