package dash0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTeam_201(t *testing.T) {
	team := TeamDefinition{
		Metadata: TeamMetadata{
			Name: "test-team",
		},
		Spec: TeamSpec{
			Display: TeamDisplay{Name: "Test Team"},
			Members: []string{"user@example.com"},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(team)
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	got, err := client.CreateTeam(context.Background(), &team)
	if err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}

	if got.Metadata.Name != "test-team" {
		t.Errorf("expected name %q, got %q", "test-team", got.Metadata.Name)
	}
	if got.Spec.Display.Name != "Test Team" {
		t.Errorf("expected display name %q, got %q", "Test Team", got.Spec.Display.Name)
	}
}
