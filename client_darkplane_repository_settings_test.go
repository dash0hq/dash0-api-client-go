package dash0

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testRepositoryID = "0b7f1c2e-4a5d-4e8f-9c61-2d3a4b5c6d7e"

const repositorySettingsJSON = `{"apiVersion":"dash0.com/v1alpha1","kind":"DarkplaneRepositorySettings","metadata":{"annotations":{"dash0.com/created-at":"2026-01-05T09:00:00Z","dash0.com/updated-at":"2026-01-12T10:30:00Z"},"labels":{"dash0.com/id":"0b7f1c2e-4a5d-4e8f-9c61-2d3a4b5c6d7e","dash0.com/version":"2"},"name":"acme/api"},"spec":{"botMode":"auto_approve","triggerMode":"auto","waitForChecks":true,"softThreshold":75,"softRiskCriteria":[{"multiplier":0.34,"name":"Blast radius","prompt":"Score how contained the change is","tools":["read_file","list_files","grep","git_diff_base","git_log","pr_files"]},{"multiplier":0.33,"name":"Evidence quality","prompt":"Score the evidence that it works","tools":["pr_reviews_and_comments","pr_checks","pr_metadata","git_log"]},{"multiplier":0.33,"name":"Change footprint","prompt":"Score how routine the change is","tools":["pr_files","git_diff_base"]}],"protectedPaths":[],"fastTrackReverts":false,"pathRules":[],"reviewer":{"automergeAcceptedVerdicts":["approve"],"commentRequests":true,"enabled":false,"requireForAutomerge":false,"settleDelaySeconds":120,"skipAfterHumanReview":true,"triggerMode":"ready"}},"usesDefaultCriteria":true}`

func repositoryJSON(id, account, name string) string {
	return `{"kind":"DarkplaneRepository","metadata":{"name":"` + account + `/` + name + `","labels":{"dash0.com/id":"` + id + `"},"annotations":{}},"spec":{"provider":"github","account":"` + account + `","name":"` + name + `","botMode":"auto_approve","openPullRequests":0,"timeToMergeP50Seconds":null,"autoApprovalRate":null}}`
}

func TestListDarkplaneRepositories_FollowsCursorsAndPassesSearch(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assertEqual(t, "path", r.URL.Path, "/api/darkplane/repositories")
		assertEqual(t, "search", r.URL.Query().Get("search"), "load")
		assertEqual(t, "limit", r.URL.Query().Get("limit"), "200")
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("cursor") {
		case "":
			_, _ = io.WriteString(w, `{"items":[`+repositoryJSON("id-1", "acme", "api")+`],"cursors":{"after":"page-2"}}`)
		case "page-2":
			_, _ = io.WriteString(w, `{"items":[`+repositoryJSON("id-2", "acme", "api-gateway")+`],"cursors":{}}`)
		default:
			t.Errorf("unexpected cursor %q", r.URL.Query().Get("cursor"))
		}
	}))
	t.Cleanup(server.Close)

	got, err := newTestClient(t, server.URL).ListDarkplaneRepositories(context.Background(), "load")
	if err != nil {
		t.Fatalf("ListDarkplaneRepositories failed: %v", err)
	}
	if calls != 2 || len(got) != 2 {
		t.Fatalf("calls = %d, repositories = %d, want 2 and 2", calls, len(got))
	}
	assertEqual(t, "second slug", GetDarkplaneRepositorySlug(&got[1]), "acme/api-gateway")
	assertEqual(t, "second id", GetDarkplaneRepositoryID(&got[1]), "id-2")
}

func TestFindDarkplaneRepository(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"items":[`+repositoryJSON("id-1", "acme", "api-gateway")+`,`+repositoryJSON("id-2", "acme", "api")+`],"cursors":{}}`)
	}))
	t.Cleanup(server.Close)
	c := newTestClient(t, server.URL)

	t.Run("matches the whole slug, case-insensitively", func(t *testing.T) {
		got, err := c.FindDarkplaneRepository(context.Background(), "Acme/API")
		if err != nil {
			t.Fatalf("FindDarkplaneRepository failed: %v", err)
		}
		assertEqual(t, "id", GetDarkplaneRepositoryID(got), "id-2")
	})

	t.Run("a substring match alone is not found", func(t *testing.T) {
		_, err := c.FindDarkplaneRepository(context.Background(), "acme/ap")
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			t.Fatalf("err = %v, want a 404 APIError", err)
		}
	})
}

func TestGetDarkplaneRepositorySettings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		assertEqual(t, "path", r.URL.Path, "/api/darkplane/repositories/"+testRepositoryID+"/settings")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, repositorySettingsJSON)
	}))
	t.Cleanup(server.Close)

	got, err := newTestClient(t, server.URL).GetDarkplaneRepositorySettings(context.Background(), testRepositoryID)
	if err != nil {
		t.Fatalf("GetDarkplaneRepositorySettings failed: %v", err)
	}
	assertEqual(t, "apiVersion", string(got.ApiVersion), string(RepositorySettingsApiVersionDash0Comv1alpha1))
	assertEqual(t, "id", GetDarkplaneRepositorySettingsID(got), testRepositoryID)
	assertEqual(t, "version", GetDarkplaneRepositorySettingsVersion(got), "2")
	assertEqual(t, "botMode", string(got.Spec.BotMode), "auto_approve")
	if got.Spec.SoftThreshold != 75 || len(got.Spec.SoftRiskCriteria) != 3 {
		t.Errorf("unexpected spec: threshold %v, %d criteria", got.Spec.SoftThreshold, len(got.Spec.SoftRiskCriteria))
	}
}

func TestUpdateDarkplaneRepositorySettings(t *testing.T) {
	var settings RepositorySettingsDefinition
	if err := json.Unmarshal([]byte(repositorySettingsJSON), &settings); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	tests := []struct {
		name           string
		edit           func(*RepositorySettingsDefinition)
		wantVersion    string
		wantAPIVersion string
	}{
		{
			name:           "sends the version and apiVersion it read",
			edit:           func(*RepositorySettingsDefinition) {},
			wantVersion:    "2",
			wantAPIVersion: "dash0.com/v1alpha1",
		},
		{
			name:           "sends no version when the document has none",
			edit:           func(s *RepositorySettingsDefinition) { s.Metadata.Labels.Dash0Comversion = nil },
			wantVersion:    "",
			wantAPIVersion: "dash0.com/v1alpha1",
		},
		{
			name:           "sends no apiVersion when the document has none",
			edit:           func(s *RepositorySettingsDefinition) { s.ApiVersion = "" },
			wantVersion:    "2",
			wantAPIVersion: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body map[string]json.RawMessage
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Errorf("method = %s, want PUT", r.Method)
				}
				assertEqual(t, "path", r.URL.Path, "/api/darkplane/repositories/"+testRepositoryID+"/settings")
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode request: %v", err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, repositorySettingsJSON)
			}))
			t.Cleanup(server.Close)

			edited := settings
			tt.edit(&edited)
			edited.Spec.ProtectedPaths = nil

			if _, err := newTestClient(t, server.URL).UpdateDarkplaneRepositorySettings(context.Background(), testRepositoryID, &edited); err != nil {
				t.Fatalf("UpdateDarkplaneRepositorySettings failed: %v", err)
			}
			assertEqual(t, "kind", string(body["kind"]), `"DarkplaneRepositorySettings"`)
			if _, ok := body["usesDefaultCriteria"]; ok {
				t.Error("request carries usesDefaultCriteria, which the PUT does not read")
			}
			var request RepositorySettingsRequest
			raw, _ := json.Marshal(body)
			if err := json.Unmarshal(raw, &request); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if request.Spec.ProtectedPaths == nil {
				t.Error("protectedPaths was sent as null, want []")
			}
			gotVersion := ""
			if request.Metadata != nil && request.Metadata.Labels != nil && request.Metadata.Labels.Dash0Comversion != nil {
				gotVersion = *request.Metadata.Labels.Dash0Comversion
			}
			assertEqual(t, "version", gotVersion, tt.wantVersion)
			gotAPIVersion := ""
			if request.ApiVersion != nil {
				gotAPIVersion = string(*request.ApiVersion)
			}
			assertEqual(t, "apiVersion", gotAPIVersion, tt.wantAPIVersion)
		})
	}
}

func TestUpdateDarkplaneRepositorySettings_ReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusConflict, "superseded")
	}))
	t.Cleanup(server.Close)
	var settings RepositorySettingsDefinition
	if err := json.Unmarshal([]byte(repositorySettingsJSON), &settings); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	_, err := newTestClient(t, server.URL).UpdateDarkplaneRepositorySettings(context.Background(), testRepositoryID, &settings)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("err = %v, want a 409 APIError", err)
	}
}

func TestListDarkplaneRepositorySettingsVersions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "path", r.URL.Path, "/api/darkplane/repositories/"+testRepositoryID+"/settings/versions")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"versions":[`+repositorySettingsJSON+`]}`)
	}))
	t.Cleanup(server.Close)

	got, err := newTestClient(t, server.URL).ListDarkplaneRepositorySettingsVersions(context.Background(), testRepositoryID)
	if err != nil {
		t.Fatalf("ListDarkplaneRepositorySettingsVersions failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("versions = %d, want 1", len(got))
	}
	assertEqual(t, "version", GetDarkplaneRepositorySettingsVersion(&got[0]), "2")
}

func TestDarkplaneRepositorySettings_RequireRepositoryID(t *testing.T) {
	c := newTestClient(t, "http://127.0.0.1:1")
	if _, err := c.GetDarkplaneRepositorySettings(context.Background(), ""); err == nil {
		t.Error("GetDarkplaneRepositorySettings accepted an empty repository ID")
	}
	if _, err := c.UpdateDarkplaneRepositorySettings(context.Background(), "", &RepositorySettingsDefinition{}); err == nil {
		t.Error("UpdateDarkplaneRepositorySettings accepted an empty repository ID")
	}
}

func TestListDarkplaneRepositories_RejectsCursorsThatDoNotAdvance(t *testing.T) {
	tests := []struct {
		name string
		page func(cursor string) string
	}{
		{
			name: "an empty page with a cursor",
			page: func(string) string { return `{"items":[],"cursors":{"after":"page-2"}}` },
		},
		{
			name: "the same cursor twice",
			page: func(string) string {
				return `{"items":[` + repositoryJSON("id-1", "acme", "api") + `],"cursors":{"after":"page-2"}}`
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tt.page(r.URL.Query().Get("cursor")))
			}))
			t.Cleanup(server.Close)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, err := newTestClient(t, server.URL).ListDarkplaneRepositories(ctx, ""); err == nil {
				t.Fatal("ListDarkplaneRepositories succeeded, want an error")
			}
			if calls > 2 {
				t.Errorf("calls = %d, want at most 2", calls)
			}
		})
	}
}

func TestDarkplaneRepositorySettings_ReturnAPIErrors(t *testing.T) {
	var settings RepositorySettingsDefinition
	if err := json.Unmarshal([]byte(repositorySettingsJSON), &settings); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	calls := map[string]func(Client) error{
		"list repositories": func(c Client) error {
			_, err := c.ListDarkplaneRepositories(context.Background(), "")
			return err
		},
		"get settings": func(c Client) error {
			_, err := c.GetDarkplaneRepositorySettings(context.Background(), testRepositoryID)
			return err
		},
		"update settings": func(c Client) error {
			_, err := c.UpdateDarkplaneRepositorySettings(context.Background(), testRepositoryID, &settings)
			return err
		},
		"list settings versions": func(c Client) error {
			_, err := c.ListDarkplaneRepositorySettingsVersions(context.Background(), testRepositoryID)
			return err
		},
	}
	statuses := []struct {
		status int
		is     func(error) bool
	}{
		{http.StatusBadRequest, IsBadRequest},
		{http.StatusForbidden, IsForbidden},
		{http.StatusNotFound, IsNotFound},
	}
	for name, call := range calls {
		for _, st := range statuses {
			t.Run(fmt.Sprintf("%s %d", name, st.status), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					writeAPIError(w, st.status, "denied")
				}))
				t.Cleanup(server.Close)

				err := call(newTestClient(t, server.URL))
				var apiErr *APIError
				if !errors.As(err, &apiErr) || apiErr.StatusCode != st.status || !st.is(err) {
					t.Fatalf("err = %v, want a %d APIError", err, st.status)
				}
				assertEqual(t, "message", apiErr.Message, "denied")
			})
		}
	}
}

func TestDarkplaneRepositoryAccessors_AcceptNil(t *testing.T) {
	assertEqual(t, "slug", GetDarkplaneRepositorySlug(nil), "")
	assertEqual(t, "repository id", GetDarkplaneRepositoryID(nil), "")
	assertEqual(t, "settings id", GetDarkplaneRepositorySettingsID(nil), "")
	assertEqual(t, "settings version", GetDarkplaneRepositorySettingsVersion(nil), "")
	assertEqual(t, "zero repository id", GetDarkplaneRepositoryID(&RepositoryDefinition{}), "")
	assertEqual(t, "zero settings version", GetDarkplaneRepositorySettingsVersion(&RepositorySettingsDefinition{}), "")
}
