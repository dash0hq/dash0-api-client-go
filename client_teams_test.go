package dash0

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Fixture-derived team envelopes. The wire shapes match the shared fixtures
// at dash0-iac-maintainer/fixtures/teams-crd-api/{create,update}.yaml and the
// corresponding response fixtures. Fields not surfaced by the generated
// types (e.g. spec.display.description, or server-managed timestamp
// annotations that the current spec does not model yet) are omitted here
// because the generated struct does not carry them; the round-trip
// semantics being exercised are the ones the client depends on.

const (
	testTeamID    = "00000000-0000-0000-0000-000000000001"
	testMemberID1 = "00000000-0000-0000-0000-0000000000A1"
	testMemberID2 = "00000000-0000-0000-0000-0000000000A2"
	testMemberID3 = "00000000-0000-0000-0000-0000000000A3"
)

func newCreateFixtureTeam() *TeamDefinitionV1Alpha1 {
	return &TeamDefinitionV1Alpha1{
		Kind: Dash0Team,
		Metadata: TeamMetadata{
			Name: "backend-team",
		},
		Spec: TeamSpec{
			Display: TeamDisplay{
				Name:  "Backend Team",
				Color: Gradient{From: "#6366F1", To: "#8B5CF6"},
			},
			Members: []string{"alice@example.com", "bob@example.com"},
		},
	}
}

func newCreateResponseFixtureTeam() *TeamDefinitionV1Alpha1 {
	source := CrdSource("api")
	id := testTeamID
	origin := testTeamID
	return &TeamDefinitionV1Alpha1{
		Kind: Dash0Team,
		Metadata: TeamMetadata{
			Name: "backend-team",
			Labels: &TeamLabels{
				Dash0Comid:     &id,
				Dash0Comorigin: &origin,
				Dash0Comsource: &source,
			},
		},
		Spec: TeamSpec{
			Display: TeamDisplay{
				Name:  "Backend Team",
				Color: Gradient{From: "#6366F1", To: "#8B5CF6"},
			},
			Members: []string{testMemberID1, testMemberID2},
		},
	}
}


// Helpers

func TestGetTeamID(t *testing.T) {
	tests := []struct {
		name string
		team *TeamDefinitionV1Alpha1
		want string
	}{
		{"nil team", nil, ""},
		{"nil labels", &TeamDefinitionV1Alpha1{}, ""},
		{"nil ID", &TeamDefinitionV1Alpha1{Metadata: TeamMetadata{Labels: &TeamLabels{}}}, ""},
		{"with ID", &TeamDefinitionV1Alpha1{Metadata: TeamMetadata{Labels: &TeamLabels{Dash0Comid: Ptr(testTeamID)}}}, testTeamID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTeamID(tt.team); got != tt.want {
				t.Errorf("GetTeamID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetTeamName(t *testing.T) {
	tests := []struct {
		name string
		team *TeamDefinitionV1Alpha1
		want string
	}{
		{"nil team", nil, ""},
		{"empty name", &TeamDefinitionV1Alpha1{}, ""},
		{"with name", &TeamDefinitionV1Alpha1{Metadata: TeamMetadata{Name: "backend-team"}}, "backend-team"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTeamName(tt.team); got != tt.want {
				t.Errorf("GetTeamName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetTeamDisplayName(t *testing.T) {
	tests := []struct {
		name string
		team *TeamDefinitionV1Alpha1
		want string
	}{
		{"nil team", nil, ""},
		{"empty display", &TeamDefinitionV1Alpha1{}, ""},
		{"with display name", &TeamDefinitionV1Alpha1{Spec: TeamSpec{Display: TeamDisplay{Name: "Backend Team"}}}, "Backend Team"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTeamDisplayName(tt.team); got != tt.want {
				t.Errorf("GetTeamDisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetTeamOrigin(t *testing.T) {
	tests := []struct {
		name string
		team *TeamDefinitionV1Alpha1
		want string
	}{
		{"nil team", nil, ""},
		{"nil labels", &TeamDefinitionV1Alpha1{}, ""},
		{"nil origin", &TeamDefinitionV1Alpha1{Metadata: TeamMetadata{Labels: &TeamLabels{}}}, ""},
		{"with origin", &TeamDefinitionV1Alpha1{Metadata: TeamMetadata{Labels: &TeamLabels{Dash0Comorigin: Ptr("tf_backend")}}}, "tf_backend"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTeamOrigin(tt.team); got != tt.want {
				t.Errorf("GetTeamOrigin() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetTeamID(t *testing.T) {
	t.Run("sets ID when unset", func(t *testing.T) {
		team := &TeamDefinitionV1Alpha1{}
		SetTeamID(team, "team-1")
		if team.Metadata.Labels == nil || team.Metadata.Labels.Dash0Comid == nil {
			t.Fatal("expected ID to be set")
		}
		if *team.Metadata.Labels.Dash0Comid != "team-1" {
			t.Errorf("ID = %q, want %q", *team.Metadata.Labels.Dash0Comid, "team-1")
		}
	})
	t.Run("overwrites existing ID", func(t *testing.T) {
		team := &TeamDefinitionV1Alpha1{Metadata: TeamMetadata{Labels: &TeamLabels{Dash0Comid: Ptr("old")}}}
		SetTeamID(team, "new")
		if *team.Metadata.Labels.Dash0Comid != "new" {
			t.Errorf("ID = %q, want %q", *team.Metadata.Labels.Dash0Comid, "new")
		}
	})
	t.Run("nil-safe", func(t *testing.T) {
		SetTeamID(nil, "team-1") // should not panic
	})
}

func TestSetTeamIDIfAbsent(t *testing.T) {
	t.Run("sets when absent", func(t *testing.T) {
		team := &TeamDefinitionV1Alpha1{}
		SetTeamIDIfAbsent(team, "team-1")
		if team.Metadata.Labels == nil || team.Metadata.Labels.Dash0Comid == nil {
			t.Fatal("expected ID to be set")
		}
		if *team.Metadata.Labels.Dash0Comid != "team-1" {
			t.Errorf("ID = %q, want %q", *team.Metadata.Labels.Dash0Comid, "team-1")
		}
	})
	t.Run("no-op when already set", func(t *testing.T) {
		team := &TeamDefinitionV1Alpha1{Metadata: TeamMetadata{Labels: &TeamLabels{Dash0Comid: Ptr("existing")}}}
		SetTeamIDIfAbsent(team, "new")
		if *team.Metadata.Labels.Dash0Comid != "existing" {
			t.Errorf("ID = %q, want %q (should not overwrite)", *team.Metadata.Labels.Dash0Comid, "existing")
		}
	})
	t.Run("nil-safe", func(t *testing.T) {
		SetTeamIDIfAbsent(nil, "team-1") // should not panic
	})
}

func TestClearTeamID(t *testing.T) {
	t.Run("clears set ID", func(t *testing.T) {
		team := &TeamDefinitionV1Alpha1{Metadata: TeamMetadata{Labels: &TeamLabels{Dash0Comid: Ptr("team-1")}}}
		ClearTeamID(team)
		if team.Metadata.Labels.Dash0Comid != nil {
			t.Error("expected ID to be nil")
		}
	})
	t.Run("no-op when labels nil", func(t *testing.T) {
		team := &TeamDefinitionV1Alpha1{}
		ClearTeamID(team) // should not panic
		if team.Metadata.Labels != nil {
			t.Error("Labels should remain nil")
		}
	})
	t.Run("nil-safe", func(t *testing.T) {
		ClearTeamID(nil) // should not panic
	})
}

func TestStripTeamServerFields(t *testing.T) {
	t.Run("clears server-managed labels, preserves user data", func(t *testing.T) {
		source := CrdSource("api")
		team := &TeamDefinitionV1Alpha1{
			Metadata: TeamMetadata{
				Name: "backend-team",
				Labels: &TeamLabels{
					Dash0Comid:     Ptr("id-1"),
					Dash0Comorigin: Ptr("origin-1"),
					Dash0Comsource: &source,
				},
			},
		}
		StripTeamServerFields(team)

		if team.Metadata.Labels.Dash0Comid != nil {
			t.Error("dash0.com/id should be nil")
		}
		if team.Metadata.Labels.Dash0Comorigin == nil || *team.Metadata.Labels.Dash0Comorigin != "origin-1" {
			t.Error("dash0.com/origin should be preserved (client-settable on create, immutable after)")
		}
		if team.Metadata.Labels.Dash0Comsource != nil {
			t.Error("dash0.com/source should be nil")
		}
		if team.Metadata.Name != "backend-team" {
			t.Errorf("metadata.name = %q, want %q (should be preserved)", team.Metadata.Name, "backend-team")
		}
	})
	t.Run("nil labels", func(t *testing.T) {
		team := &TeamDefinitionV1Alpha1{}
		StripTeamServerFields(team) // should not panic
		if team.Metadata.Labels != nil {
			t.Error("Labels should remain nil")
		}
	})
	t.Run("nil-safe", func(t *testing.T) {
		StripTeamServerFields(nil) // should not panic
	})
}

func TestDecodeRoundTrip_Labels(t *testing.T) {
	// Encode the response fixture, decode it back, and verify that
	// server-managed labels survive the round trip.
	src := newCreateResponseFixtureTeam()

	body, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var got TeamDefinitionV1Alpha1
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if got.Metadata.Name != "backend-team" {
		t.Errorf("metadata.name = %q, want %q", got.Metadata.Name, "backend-team")
	}
	if GetTeamID(&got) != testTeamID {
		t.Errorf("dash0.com/id did not round-trip: got %q", GetTeamID(&got))
	}
	if GetTeamOrigin(&got) != testTeamID {
		t.Errorf("dash0.com/origin did not round-trip: got %q", GetTeamOrigin(&got))
	}
	if got.Metadata.Labels.Dash0Comsource == nil || string(*got.Metadata.Labels.Dash0Comsource) != "api" {
		t.Errorf("dash0.com/source did not round-trip")
	}
	if len(got.Spec.Members) != 2 || got.Spec.Members[0] != testMemberID1 {
		t.Errorf("spec.members did not round-trip: got %v", got.Spec.Members)
	}
}

// Integration tests (httptest.Server against the real generated client).

func TestTeams_Integration(t *testing.T) {
	t.Run("ListTeams returns teams", func(t *testing.T) {
		items := []TeamsListItem{{Id: testTeamID, Name: "Backend Team"}}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/teams" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method: %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(items)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		got, err := client.ListTeams(context.Background())
		if err != nil {
			t.Fatalf("ListTeams failed: %v", err)
		}
		if len(got) != 1 || got[0].Id != testTeamID {
			t.Errorf("unexpected list: %+v", got)
		}
	})

	t.Run("CreateTeam succeeds and returns CRD envelope", func(t *testing.T) {
		response := newCreateResponseFixtureTeam()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/teams" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("unexpected method: %s", r.Method)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body failed: %v", err)
			}
			var req TeamDefinitionV1Alpha1
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("unmarshal request failed: %v", err)
			}
			if req.Metadata.Name != "backend-team" {
				t.Errorf("unexpected metadata.name: %q", req.Metadata.Name)
			}
			if len(req.Spec.Members) != 2 || req.Spec.Members[0] != "alice@example.com" {
				t.Errorf("unexpected spec.members: %v", req.Spec.Members)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		team := newCreateFixtureTeam()
		got, err := client.CreateTeam(context.Background(), team)
		if err != nil {
			t.Fatalf("CreateTeam failed: %v", err)
		}
		if GetTeamID(got) != testTeamID {
			t.Errorf("expected id %q, got %q", testTeamID, GetTeamID(got))
		}
		if GetTeamName(got) != "backend-team" {
			t.Errorf("expected metadata.name %q, got %q", "backend-team", GetTeamName(got))
		}
	})

	t.Run("CreateTeam accepts 201 with body parsed from raw response", func(t *testing.T) {
		response := newCreateResponseFixtureTeam()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		got, err := client.CreateTeam(context.Background(), newCreateFixtureTeam())
		if err != nil {
			t.Fatalf("CreateTeam failed: %v", err)
		}
		if GetTeamID(got) != testTeamID {
			t.Errorf("expected id %q, got %q", testTeamID, GetTeamID(got))
		}
	})

	t.Run("CreateTeam rejects nil team", func(t *testing.T) {
		client, err := NewClient(WithApiUrl("http://example.invalid"), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		if _, err := client.CreateTeam(context.Background(), nil); err == nil {
			t.Fatal("expected error for nil team")
		}
	})

	t.Run("UpsertTeam PUTs to /api/teams/{originOrId} and returns the envelope", func(t *testing.T) {
		response := newCreateResponseFixtureTeam()
		var gotPath, gotMethod string
		var gotBody TeamDefinitionV1Alpha1
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotMethod = r.Method
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		team := newCreateFixtureTeam()
		got, err := client.UpsertTeam(context.Background(), "tf_backend", team)
		if err != nil {
			t.Fatalf("UpsertTeam failed: %v", err)
		}
		if gotPath != "/api/teams/tf_backend" {
			t.Errorf("expected PUT to /api/teams/tf_backend, got %s", gotPath)
		}
		if gotMethod != http.MethodPut {
			t.Errorf("expected method PUT, got %s", gotMethod)
		}
		if gotBody.Spec.Display.Name != "Backend Team" {
			t.Errorf("expected request body to carry display.name=Backend Team, got %q", gotBody.Spec.Display.Name)
		}
		if len(gotBody.Spec.Members) != 2 {
			t.Errorf("expected 2 members in the request body, got %d", len(gotBody.Spec.Members))
		}
		if GetTeamID(got) != testTeamID {
			t.Errorf("expected id %q, got %q", testTeamID, GetTeamID(got))
		}
	})

	t.Run("UpsertTeam falls back to raw response body when JSON200 is nil", func(t *testing.T) {
		response := newCreateResponseFixtureTeam()
		body, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("marshal fixture: %v", err)
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Omit Content-Type so the codegen wrapper's JSON200 field stays nil
			// and the wrapper falls through to the raw json.Unmarshal path.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		team := newCreateFixtureTeam()
		got, err := client.UpsertTeam(context.Background(), testTeamID, team)
		if err != nil {
			t.Fatalf("UpsertTeam failed: %v", err)
		}
		if GetTeamID(got) != testTeamID {
			t.Errorf("expected id %q, got %q", testTeamID, GetTeamID(got))
		}
	})

	t.Run("UpsertTeam surfaces server 400", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "spec.members contains unresolvable email address(es): ghost@example.com",
			})
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		team := newCreateFixtureTeam()
		team.Spec.Members = []string{"ghost@example.com"}
		if _, err := client.UpsertTeam(context.Background(), "tf_bad", team); err == nil {
			t.Fatal("expected 400 to surface as an error")
		} else if !strings.Contains(err.Error(), "unresolvable email") {
			t.Errorf("expected error to relay server body, got %q", err.Error())
		}
	})

	t.Run("UpsertTeam rejects nil team", func(t *testing.T) {
		client, err := NewClient(WithApiUrl("http://example.invalid"), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		if _, err := client.UpsertTeam(context.Background(), "tf_x", nil); err == nil {
			t.Fatal("expected error for nil team")
		}
	})

	t.Run("GetTeam extracts the CRD envelope from the enriched response", func(t *testing.T) {
		response := &GetTeamResponse{
			Team:            *newCreateResponseFixtureTeam(),
			CheckRules:      []AccessibleAsset{},
			Dashboards:      []AccessibleAsset{},
			Datasets:        []AccessibleAsset{},
			Members:         []MemberDefinition{},
			SyntheticChecks: []AccessibleAsset{},
			Views:           []AccessibleAsset{},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expected := "/api/teams/" + testTeamID
			if r.URL.Path != expected {
				t.Errorf("unexpected path: %s, want %s", r.URL.Path, expected)
			}
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method: %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		got, err := client.GetTeam(context.Background(), testTeamID)
		if err != nil {
			t.Fatalf("GetTeam failed: %v", err)
		}
		if GetTeamID(got) != testTeamID {
			t.Errorf("expected id %q, got %q", testTeamID, GetTeamID(got))
		}
		if GetTeamDisplayName(got) != "Backend Team" {
			t.Errorf("expected display name %q, got %q", "Backend Team", GetTeamDisplayName(got))
		}
	})

	t.Run("GetTeamWithAssets returns the enriched shape", func(t *testing.T) {
		response := &GetTeamResponse{
			Team: *newCreateResponseFixtureTeam(),
			Dashboards: []AccessibleAsset{
				{Name: "Production Overview"},
			},
			CheckRules:      []AccessibleAsset{},
			Datasets:        []AccessibleAsset{},
			Members:         []MemberDefinition{},
			SyntheticChecks: []AccessibleAsset{},
			Views:           []AccessibleAsset{},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		got, err := client.GetTeamWithAssets(context.Background(), testTeamID)
		if err != nil {
			t.Fatalf("GetTeamWithAssets failed: %v", err)
		}
		if GetTeamID(&got.Team) != testTeamID {
			t.Errorf("expected team id %q, got %q", testTeamID, GetTeamID(&got.Team))
		}
		if len(got.Dashboards) != 1 || got.Dashboards[0].Name != "Production Overview" {
			t.Errorf("expected 1 dashboard, got %v", got.Dashboards)
		}
	})

	t.Run("GetTeam handles 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "team not found"})
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		_, err = client.GetTeam(context.Background(), "missing")
		if err == nil {
			t.Fatal("expected error")
		}
		if !IsNotFound(err) {
			t.Errorf("expected IsNotFound to return true")
		}
	})

	t.Run("DeleteTeam succeeds with 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("unexpected method: %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		if err := client.DeleteTeam(context.Background(), testTeamID); err != nil {
			t.Fatalf("DeleteTeam failed: %v", err)
		}
	})

	t.Run("DeleteTeam succeeds with 204", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		if err := client.DeleteTeam(context.Background(), testTeamID); err != nil {
			t.Fatalf("DeleteTeam failed: %v", err)
		}
	})

	t.Run("UpdateTeamDisplay hits the legacy endpoint", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expected := "/api/teams/" + testTeamID + "/display"
			if r.URL.Path != expected {
				t.Errorf("unexpected path: %s, want %s", r.URL.Path, expected)
			}
			if r.Method != http.MethodPut {
				t.Errorf("unexpected method: %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		display := &TeamDisplay{Name: "Backend Platform", Color: Gradient{From: "#000000", To: "#FFFFFF"}}
		if err := client.UpdateTeamDisplay(context.Background(), testTeamID, display); err != nil {
			t.Fatalf("UpdateTeamDisplay failed: %v", err)
		}
	})

	t.Run("AddTeamMembers hits the legacy endpoint", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expected := "/api/teams/" + testTeamID + "/members"
			if r.URL.Path != expected {
				t.Errorf("unexpected path: %s, want %s", r.URL.Path, expected)
			}
			if r.Method != http.MethodPost {
				t.Errorf("unexpected method: %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		if err := client.AddTeamMembers(context.Background(), testTeamID, &AddTeamMembersRequest{MemberIds: []string{testMemberID1}}); err != nil {
			t.Fatalf("AddTeamMembers failed: %v", err)
		}
	})

	t.Run("RemoveTeamMember hits the legacy endpoint", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expected := "/api/teams/" + testTeamID + "/members/" + testMemberID1
			if r.URL.Path != expected {
				t.Errorf("unexpected path: %s, want %s", r.URL.Path, expected)
			}
			if r.Method != http.MethodDelete {
				t.Errorf("unexpected method: %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		if err := client.RemoveTeamMember(context.Background(), testTeamID, testMemberID1); err != nil {
			t.Fatalf("RemoveTeamMember failed: %v", err)
		}
	})

	t.Run("ListTeamsIter iterates all items", func(t *testing.T) {
		items := []TeamsListItem{
			{Id: testTeamID, Name: "Backend Team"},
			{Id: "team-2", Name: "Frontend Team"},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(items)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		iter := client.ListTeamsIter(context.Background())
		var names []string
		for iter.Next() {
			names = append(names, iter.Current().Name)
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("iterator error: %v", err)
		}
		if len(names) != 2 || names[0] != "Backend Team" || names[1] != "Frontend Team" {
			t.Errorf("unexpected iteration: %v", names)
		}
	})
}

// ResolveMemberIDsToEmails

func TestResolveMemberIDsToEmails(t *testing.T) {
	newMembersServer := func(members []MemberDefinition) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/members" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(members)
		}))
	}

	fixtureMembers := []MemberDefinition{
		{
			Kind: MemberDefinitionKind("Dash0Member"),
			Metadata: MemberMetadata{
				Name:   testMemberID1,
				Labels: &MemberLabels{Dash0Comid: Ptr(testMemberID1)},
			},
			Spec: MemberSpec{Display: MemberDisplay{Email: Ptr("alice@example.com")}},
		},
		{
			Kind: MemberDefinitionKind("Dash0Member"),
			Metadata: MemberMetadata{
				Name:   testMemberID2,
				Labels: &MemberLabels{Dash0Comid: Ptr(testMemberID2)},
			},
			Spec: MemberSpec{Display: MemberDisplay{Email: Ptr("bob@example.com")}},
		},
	}

	t.Run("happy path resolves IDs to emails", func(t *testing.T) {
		server := newMembersServer(fixtureMembers)
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		got, err := client.ResolveMemberIDsToEmails(context.Background(), []string{testMemberID2, testMemberID1})
		if err != nil {
			t.Fatalf("ResolveMemberIDsToEmails failed: %v", err)
		}
		want := []string{"bob@example.com", "alice@example.com"}
		if len(got) != len(want) {
			t.Fatalf("expected %d results, got %d", len(want), len(got))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("index %d = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("empty input passes through without calling API", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("no HTTP call expected for empty input, got %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		got, err := client.ResolveMemberIDsToEmails(context.Background(), nil)
		if err != nil {
			t.Fatalf("ResolveMemberIDsToEmails failed: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty result, got %v", got)
		}
		got, err = client.ResolveMemberIDsToEmails(context.Background(), []string{})
		if err != nil {
			t.Fatalf("ResolveMemberIDsToEmails failed on empty slice: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty result, got %v", got)
		}
	})

	t.Run("unresolved ID passes through as-is", func(t *testing.T) {
		server := newMembersServer(fixtureMembers)
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		got, err := client.ResolveMemberIDsToEmails(context.Background(), []string{testMemberID1, "user_orphaned"})
		if err != nil {
			t.Fatalf("ResolveMemberIDsToEmails failed: %v", err)
		}
		want := []string{"alice@example.com", "user_orphaned"}
		if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("expected %v, got %v", want, got)
		}
	})

	t.Run("propagates ListMembers error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "boom"})
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		_, err = client.ResolveMemberIDsToEmails(context.Background(), []string{testMemberID1})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "resolve member IDs to emails") {
			t.Errorf("expected wrapped error, got %v", err)
		}
	})

	t.Run("skips members without an email display", func(t *testing.T) {
		members := []MemberDefinition{
			{
				Metadata: MemberMetadata{Name: testMemberID1, Labels: &MemberLabels{Dash0Comid: Ptr(testMemberID1)}},
				Spec:     MemberSpec{Display: MemberDisplay{}},
			},
		}
		server := newMembersServer(members)
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		got, err := client.ResolveMemberIDsToEmails(context.Background(), []string{testMemberID1})
		if err != nil {
			t.Fatalf("ResolveMemberIDsToEmails failed: %v", err)
		}
		if len(got) != 1 || got[0] != testMemberID1 {
			t.Errorf("expected passthrough of raw ID, got %v", got)
		}
	})

	t.Run("resolves via metadata.name when the dash0.com/id label is absent", func(t *testing.T) {
		// The /api/members response on the public API populates metadata.name
		// with the canonical member id but leaves labels["dash0.com/id"] unset
		// (the field is `omitempty` on the wire). The resolver must key on
		// metadata.name as the primary lookup and only use the label as a
		// fallback — otherwise every id passes through as-is because the map
		// keyed on the missing label is empty.
		members := []MemberDefinition{
			{
				Metadata: MemberMetadata{Name: testMemberID1}, // no Labels
				Spec:     MemberSpec{Display: MemberDisplay{Email: Ptr("alice@example.com")}},
			},
			{
				Metadata: MemberMetadata{Name: testMemberID2}, // no Labels
				Spec:     MemberSpec{Display: MemberDisplay{Email: Ptr("bob@example.com")}},
			},
		}
		server := newMembersServer(members)
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		got, err := client.ResolveMemberIDsToEmails(context.Background(), []string{testMemberID1, testMemberID2})
		if err != nil {
			t.Fatalf("ResolveMemberIDsToEmails failed: %v", err)
		}
		want := []string{"alice@example.com", "bob@example.com"}
		if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("expected %v, got %v", want, got)
		}
	})
}

// Compile-time proof that TeamDefinition is a compat alias for
// TeamDefinitionV1Alpha1. If the alias is broken, this fails to compile.
var _ = func() TeamDefinition { return TeamDefinitionV1Alpha1{} }

func TestTeamDefinition_AliasIsCompatible(t *testing.T) {
	// Round-trip a value through both types to prove the alias is a true
	// type alias (identical underlying representation).
	tv1 := TeamDefinitionV1Alpha1{Metadata: TeamMetadata{Name: "alias"}}
	td := TeamDefinition(tv1)
	if td.Metadata.Name != "alias" {
		t.Errorf("alias round-trip failed: got %q", td.Metadata.Name)
	}
}

// Guard against silent regressions of the fixture-derived member list. Not
// a network test — verifies only local Go semantics.
func TestCreateFixture_MembersAsEmails(t *testing.T) {
	team := newCreateFixtureTeam()
	if len(team.Spec.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(team.Spec.Members))
	}
	for _, m := range team.Spec.Members {
		if !strings.Contains(m, "@") {
			t.Errorf("expected member %q to be an email", m)
		}
	}
}

// Ensures the response-fixture-derived envelope emits internal IDs, not
// emails. Guards against fixture drift.
func TestCreateResponseFixture_MembersAsIDs(t *testing.T) {
	team := newCreateResponseFixtureTeam()
	for _, m := range team.Spec.Members {
		if strings.Contains(m, "@") {
			t.Errorf("expected server response member %q to be an internal ID", m)
		}
	}
}

func TestResolveTeamMembersToEmails(t *testing.T) {
	newMembersServer := func(members []MemberDefinition) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/members" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(members)
		}))
	}

	fixtureMembers := []MemberDefinition{
		{
			Kind: MemberDefinitionKind("Dash0Member"),
			Metadata: MemberMetadata{
				Name:   testMemberID1,
				Labels: &MemberLabels{Dash0Comid: Ptr(testMemberID1)},
			},
			Spec: MemberSpec{Display: MemberDisplay{Email: Ptr("alice@example.com")}},
		},
		{
			Kind: MemberDefinitionKind("Dash0Member"),
			Metadata: MemberMetadata{
				Name:   testMemberID2,
				Labels: &MemberLabels{Dash0Comid: Ptr(testMemberID2)},
			},
			Spec: MemberSpec{Display: MemberDisplay{Email: Ptr("bob@example.com")}},
		},
	}

	t.Run("nil team is a no-op", func(t *testing.T) {
		// No client call must happen — pass an unreachable client and confirm
		// no error is returned.
		client, err := NewClient(WithApiUrl("http://example.invalid"), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		if err := ResolveTeamMembersToEmails(context.Background(), client, nil); err != nil {
			t.Errorf("expected nil error for nil team, got %v", err)
		}
	})

	t.Run("empty spec.members is a no-op", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("no HTTP call expected for empty members, got %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		team := &TeamDefinitionV1Alpha1{Spec: TeamSpec{Members: []string{}}}
		if err := ResolveTeamMembersToEmails(context.Background(), client, team); err != nil {
			t.Errorf("expected nil error for empty members, got %v", err)
		}
		if len(team.Spec.Members) != 0 {
			t.Errorf("expected members untouched, got %v", team.Spec.Members)
		}
	})

	t.Run("rewrites IDs in place to emails", func(t *testing.T) {
		server := newMembersServer(fixtureMembers)
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		team := &TeamDefinitionV1Alpha1{Spec: TeamSpec{Members: []string{testMemberID1, testMemberID2}}}
		if err := ResolveTeamMembersToEmails(context.Background(), client, team); err != nil {
			t.Fatalf("ResolveTeamMembersToEmails failed: %v", err)
		}
		want := []string{"alice@example.com", "bob@example.com"}
		if len(team.Spec.Members) != len(want) {
			t.Fatalf("expected %d members, got %d", len(want), len(team.Spec.Members))
		}
		for i := range team.Spec.Members {
			if team.Spec.Members[i] != want[i] {
				t.Errorf("index %d = %q, want %q", i, team.Spec.Members[i], want[i])
			}
		}
	})

	t.Run("passes through IDs that do not resolve", func(t *testing.T) {
		// Only member1 is known to the server; member2 must survive as an ID.
		server := newMembersServer(fixtureMembers[:1])
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		team := &TeamDefinitionV1Alpha1{Spec: TeamSpec{Members: []string{testMemberID1, "user_orphan"}}}
		if err := ResolveTeamMembersToEmails(context.Background(), client, team); err != nil {
			t.Fatalf("ResolveTeamMembersToEmails failed: %v", err)
		}
		want := []string{"alice@example.com", "user_orphan"}
		if len(team.Spec.Members) != len(want) || team.Spec.Members[0] != want[0] || team.Spec.Members[1] != want[1] {
			t.Errorf("expected %v, got %v", want, team.Spec.Members)
		}
	})

	t.Run("propagates errors from the members lookup", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "boom"})
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		original := []string{testMemberID1, testMemberID2}
		team := &TeamDefinitionV1Alpha1{Spec: TeamSpec{Members: append([]string(nil), original...)}}
		if err := ResolveTeamMembersToEmails(context.Background(), client, team); err == nil {
			t.Fatal("expected error to be propagated")
		}
		// Members must remain untouched when the lookup fails.
		if len(team.Spec.Members) != len(original) || team.Spec.Members[0] != original[0] || team.Spec.Members[1] != original[1] {
			t.Errorf("expected members untouched on error, got %v", team.Spec.Members)
		}
	})
}
