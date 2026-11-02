package dash0

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func newTestCustomSkill() SkillDefinition {
	return SkillDefinition{
		Kind: Dash0Skill,
		Metadata: SkillMetadata{
			Name: Ptr("incident-response"),
			Labels: &SkillLabels{
				Dash0Comid:     Ptr("skill_01"),
				Dash0Comorigin: Ptr("tf_incident-response"),
			},
		},
		Spec: SkillSpec{
			Files: []SkillFile{{
				RelativePath: "SKILL.md",
				Content:      "---\nname: incident-response\ndescription: Triage an incident.\n---\n\nSteps.\n",
			}},
		},
	}
}

func TestCustomSkills_Integration(t *testing.T) {
	t.Run("ListCustomSkills returns skills", func(t *testing.T) {
		skill := newTestCustomSkill()
		var gotURL *url.URL
		var gotMethod string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(SkillListResponse{Skills: []SkillDefinition{skill}})
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).ListCustomSkills(context.Background())
		if err != nil {
			t.Fatalf("ListCustomSkills failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodGet)
		assertEqual(t, "path", gotURL.Path, "/api/agent0/custom-skills")
		if len(gotURL.Query()) != 0 {
			t.Errorf("query = %q, want none", gotURL.RawQuery)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 skill, got %d", len(got))
		}
		assertEqual(t, "skills[0] name", GetCustomSkillName(got[0]), "incident-response")
	})

	t.Run("ListCustomSkills sends the origin prefix and enabled filters", func(t *testing.T) {
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(SkillListResponse{Skills: []SkillDefinition{}})
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListCustomSkills(context.Background(), WithOriginPrefix("tf_"), WithEnabled(true))
		if err != nil {
			t.Fatalf("ListCustomSkills failed: %v", err)
		}

		assertEqual(t, "originPrefix", gotURL.Query().Get("originPrefix"), "tf_")
		assertEqual(t, "enabled", gotURL.Query().Get("enabled"), "true")
	})

	t.Run("ListCustomSkills rejects an unsupported filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("the request should not be sent")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListCustomSkills(context.Background(), WithSignal(SignalToMetricsSignalTypeSpans))
		if err == nil {
			t.Fatal("expected an error for an unsupported filter")
		}
	})

	t.Run("ListCustomSkillsIter iterates over skills", func(t *testing.T) {
		skill := newTestCustomSkill()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(SkillListResponse{Skills: []SkillDefinition{skill, skill}})
		}))
		defer server.Close()

		iter := newTestClient(t, server.URL).ListCustomSkillsIter(context.Background())
		count := 0
		for iter.Next() {
			count++
		}
		if iter.Err() != nil {
			t.Fatalf("iterator failed: %v", iter.Err())
		}
		if count != 2 {
			t.Errorf("iterated %d skills, want 2", count)
		}
	})

	t.Run("GetCustomSkill reads by origin", func(t *testing.T) {
		skill := newTestCustomSkill()
		var gotPath, gotMethod string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotMethod = r.Method
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(skill)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).GetCustomSkill(context.Background(), "tf_incident-response")
		if err != nil {
			t.Fatalf("GetCustomSkill failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodGet)
		assertEqual(t, "path", gotPath, "/api/agent0/custom-skills/tf_incident-response")
		assertEqual(t, "id", GetCustomSkillID(got), "skill_01")
	})

	t.Run("CreateCustomSkill posts the skill", func(t *testing.T) {
		skill := newTestCustomSkill()
		var gotMethod, gotPath string
		var gotBody SkillDefinition
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(skill)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateCustomSkill(context.Background(), &skill)
		if err != nil {
			t.Fatalf("CreateCustomSkill failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodPost)
		assertEqual(t, "path", gotPath, "/api/agent0/custom-skills")
		assertEqual(t, "body origin", GetCustomSkillOrigin(&gotBody), "tf_incident-response")
		if len(gotBody.Spec.Files) != 1 || gotBody.Spec.Files[0].RelativePath != "SKILL.md" {
			t.Errorf("body files = %+v, want one SKILL.md", gotBody.Spec.Files)
		}
	})

	t.Run("CreateCustomSkill accepts 201", func(t *testing.T) {
		skill := newTestCustomSkill()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(skill)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).CreateCustomSkill(context.Background(), &skill)
		if err != nil {
			t.Fatalf("CreateCustomSkill failed: %v", err)
		}
		assertEqual(t, "id", GetCustomSkillID(got), "skill_01")
	})

	t.Run("CreateCustomSkill rejects a nil skill", func(t *testing.T) {
		if _, err := newTestClient(t, "http://127.0.0.1:1").CreateCustomSkill(context.Background(), nil); err == nil {
			t.Fatal("expected an error for a nil skill")
		}
	})

	t.Run("UpdateCustomSkill puts the skill by origin", func(t *testing.T) {
		skill := newTestCustomSkill()
		var gotMethod, gotPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(skill)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).UpdateCustomSkill(context.Background(), "tf_incident-response", &skill)
		if err != nil {
			t.Fatalf("UpdateCustomSkill failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodPut)
		assertEqual(t, "path", gotPath, "/api/agent0/custom-skills/tf_incident-response")
	})

	t.Run("UpdateCustomSkill rejects a nil skill", func(t *testing.T) {
		if _, err := newTestClient(t, "http://127.0.0.1:1").UpdateCustomSkill(context.Background(), "x", nil); err == nil {
			t.Fatal("expected an error for a nil skill")
		}
	})

	t.Run("DeleteCustomSkill deletes by origin", func(t *testing.T) {
		for _, status := range []int{http.StatusOK, http.StatusNoContent} {
			var gotMethod, gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				w.WriteHeader(status)
			}))

			err := newTestClient(t, server.URL).DeleteCustomSkill(context.Background(), "tf_incident-response")
			server.Close()
			if err != nil {
				t.Fatalf("DeleteCustomSkill with status %d failed: %v", status, err)
			}
			assertEqual(t, "method", gotMethod, http.MethodDelete)
			assertEqual(t, "path", gotPath, "/api/agent0/custom-skills/tf_incident-response")
		}
	})

	t.Run("errors carry the API status", func(t *testing.T) {
		skill := newTestCustomSkill()
		tests := []struct {
			name   string
			status int
			check  func(error) bool
			call   func(c Client) error
		}{
			{"get 404", http.StatusNotFound, IsNotFound, func(c Client) error {
				_, err := c.GetCustomSkill(context.Background(), "missing")
				return err
			}},
			{"create 400", http.StatusBadRequest, IsBadRequest, func(c Client) error {
				_, err := c.CreateCustomSkill(context.Background(), &skill)
				return err
			}},
			{"update 403", http.StatusForbidden, IsForbidden, func(c Client) error {
				_, err := c.UpdateCustomSkill(context.Background(), "tf_incident-response", &skill)
				return err
			}},
			{"delete 404", http.StatusNotFound, IsNotFound, func(c Client) error {
				return c.DeleteCustomSkill(context.Background(), "missing")
			}},
			{"list 403", http.StatusForbidden, IsForbidden, func(c Client) error {
				_, err := c.ListCustomSkills(context.Background())
				return err
			}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					writeAPIError(w, tt.status, "nope")
				}))
				defer server.Close()

				err := tt.call(newTestClient(t, server.URL))
				if !tt.check(err) {
					t.Errorf("error = %v, want status %d", err, tt.status)
				}
			})
		}
	})
}

func TestCustomSkills_APINotConfigured(t *testing.T) {
	c, err := NewClient(
		WithOtlpEndpoint(OtlpEncodingJson, "https://ingress.eu-west-1.aws.dash0.com"),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	skill := newTestCustomSkill()
	ctx := context.Background()

	if _, err := c.ListCustomSkills(ctx); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("ListCustomSkills error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.GetCustomSkill(ctx, "x"); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("GetCustomSkill error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.CreateCustomSkill(ctx, &skill); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("CreateCustomSkill error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.UpdateCustomSkill(ctx, "x", &skill); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("UpdateCustomSkill error = %v, want ErrAPINotConfigured", err)
	}
	if err := c.DeleteCustomSkill(ctx, "x"); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("DeleteCustomSkill error = %v, want ErrAPINotConfigured", err)
	}
	if iter := c.ListCustomSkillsIter(ctx); !errors.Is(iter.Err(), ErrAPINotConfigured) {
		t.Errorf("ListCustomSkillsIter error = %v, want ErrAPINotConfigured", iter.Err())
	}
}

func TestStripCustomSkillServerFields(t *testing.T) {
	now := time.Now()
	source := CrdSource("terraform")
	skill := newTestCustomSkill()
	skill.Metadata.Labels.Dash0Comversion = Ptr("3")
	skill.Metadata.Labels.Dash0Comsource = &source
	skill.Metadata.Annotations = &SkillAnnotations{
		Dash0ComcreatedAt: &now,
		Dash0ComupdatedAt: &now,
		Dash0ComcreatedBy: Ptr("user-1"),
		Dash0Comenabled:   Ptr("true"),
	}
	skill.Spec.PermittedActions = &[]CustomSkillAction{CustomSkillRead}
	skill.Spec.Source = &SkillSource{Provider: SkillSourceProviderGithub, RepositoryUrl: "https://github.com/acme/skills", Path: "x"}
	skill.Spec.Permissions = &[]SkillPermission{
		{Role: Ptr("admin"), Actions: []CustomSkillAction{CustomSkillRead, CustomSkillDelete}, DefaultEnabled: Ptr(true)},
		{Role: Ptr("basic_member"), Actions: []CustomSkillAction{CustomSkillRead}},
	}

	StripCustomSkillServerFields(&skill)

	if skill.Metadata.Name != nil {
		t.Error("Metadata.Name should be nil")
	}
	if skill.Metadata.Labels.Dash0Comversion != nil {
		t.Error("Dash0Comversion should be nil")
	}
	if skill.Metadata.Labels.Dash0Comsource != nil {
		t.Error("Dash0Comsource should be nil")
	}
	if skill.Metadata.Annotations != nil {
		t.Error("Annotations should be nil")
	}
	if skill.Spec.PermittedActions != nil {
		t.Error("PermittedActions should be nil")
	}
	if skill.Spec.Source != nil {
		t.Error("Source should be nil")
	}
	assertEqual(t, "id", GetCustomSkillID(&skill), "skill_01")
	assertEqual(t, "origin", GetCustomSkillOrigin(&skill), "tf_incident-response")
	if len(skill.Spec.Files) != 1 {
		t.Errorf("Files = %d entries, want 1", len(skill.Spec.Files))
	}
	if skill.Spec.Permissions == nil || len(*skill.Spec.Permissions) != 1 {
		t.Fatalf("Permissions = %+v, want only the basic_member grant", skill.Spec.Permissions)
	}
	assertPtrEqual(t, "remaining grant role", (*skill.Spec.Permissions)[0].Role, "basic_member")
}

func TestStripCustomSkillServerFields_KeepsAdminGrantThatIsOffByDefault(t *testing.T) {
	skill := newTestCustomSkill()
	skill.Spec.Permissions = &[]SkillPermission{
		{Role: Ptr("admin"), Actions: []CustomSkillAction{CustomSkillRead}, DefaultEnabled: Ptr(false)},
	}

	StripCustomSkillServerFields(&skill)

	if skill.Spec.Permissions == nil || len(*skill.Spec.Permissions) != 1 {
		t.Fatalf("Permissions = %+v, want the admin grant kept", skill.Spec.Permissions)
	}
}

func TestStripCustomSkillServerFields_ZeroValue(t *testing.T) {
	skill := &SkillDefinition{}
	StripCustomSkillServerFields(skill)
	if skill.Metadata.Labels != nil {
		t.Error("Labels should remain nil")
	}
	if skill.Spec.Permissions != nil {
		t.Error("Permissions should remain nil, so an update leaves the stored grants unchanged")
	}
}

func TestStripCustomSkillServerFields_Nil(t *testing.T) {
	StripCustomSkillServerFields(nil)
}

func TestGetCustomSkillID(t *testing.T) {
	skill := newTestCustomSkill()
	tests := []struct {
		name  string
		skill *SkillDefinition
		want  string
	}{
		{"nil skill", nil, ""},
		{"nil labels", &SkillDefinition{}, ""},
		{"with ID", &skill, "skill_01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, "GetCustomSkillID", GetCustomSkillID(tt.skill), tt.want)
		})
	}
}

func TestGetCustomSkillName(t *testing.T) {
	skill := newTestCustomSkill()
	tests := []struct {
		name  string
		skill *SkillDefinition
		want  string
	}{
		{"nil skill", nil, ""},
		{"no name", &SkillDefinition{}, ""},
		{"with name", &skill, "incident-response"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, "GetCustomSkillName", GetCustomSkillName(tt.skill), tt.want)
		})
	}
}

func TestSetCustomSkillID(t *testing.T) {
	skill := &SkillDefinition{}
	SetCustomSkillID(skill, "skill_01")
	assertEqual(t, "id", GetCustomSkillID(skill), "skill_01")
	SetCustomSkillID(skill, "skill_02")
	assertEqual(t, "id after overwrite", GetCustomSkillID(skill), "skill_02")
}

func TestSetCustomSkillID_Nil(t *testing.T) {
	SetCustomSkillID(nil, "skill_01")
}

func TestSetCustomSkillIDIfAbsent(t *testing.T) {
	skill := &SkillDefinition{}
	SetCustomSkillIDIfAbsent(skill, "skill_01")
	assertEqual(t, "id", GetCustomSkillID(skill), "skill_01")
	SetCustomSkillIDIfAbsent(skill, "skill_02")
	assertEqual(t, "id after second call", GetCustomSkillID(skill), "skill_01")
}

func TestSetCustomSkillIDIfAbsent_Nil(t *testing.T) {
	SetCustomSkillIDIfAbsent(nil, "skill_01")
}

func TestClearCustomSkillID(t *testing.T) {
	skill := newTestCustomSkill()
	ClearCustomSkillID(&skill)
	assertEqual(t, "id", GetCustomSkillID(&skill), "")
}

func TestClearCustomSkillID_Nil(t *testing.T) {
	ClearCustomSkillID(nil)
	ClearCustomSkillID(&SkillDefinition{})
}

func TestGetAndSetCustomSkillOrigin(t *testing.T) {
	skill := &SkillDefinition{}
	assertEqual(t, "origin before set", GetCustomSkillOrigin(skill), "")
	SetCustomSkillOrigin(skill, "tf_incident-response")
	assertEqual(t, "origin", GetCustomSkillOrigin(skill), "tf_incident-response")
}

func TestCustomSkillOrigin_Nil(t *testing.T) {
	assertEqual(t, "origin of nil", GetCustomSkillOrigin(nil), "")
	SetCustomSkillOrigin(nil, "tf_incident-response")
}
