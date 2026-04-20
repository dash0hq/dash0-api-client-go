package dash0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateSamplingRule_201(t *testing.T) {
	rule := SamplingDefinition{
		Metadata: SamplingMetadata{
			Name: "test-sampling-rule",
		},
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

	got, err := client.CreateSamplingRule(context.Background(), &rule, nil)
	if err != nil {
		t.Fatalf("CreateSamplingRule failed: %v", err)
	}

	if got.Metadata.Name != "test-sampling-rule" {
		t.Errorf("expected name %q, got %q", "test-sampling-rule", got.Metadata.Name)
	}
}
