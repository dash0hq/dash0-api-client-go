package dash0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestStripIntegrationServerFields(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	version := int64(3)
	annotCreatedAt := "2024-01-01T00:00:00Z"
	annotUpdatedAt := "2024-01-02T00:00:00Z"

	def := &IntegrationDefinition{
		Kind: "Dash0Integration",
		Metadata: IntegrationMetadata{
			Name:      "test-integration",
			CreatedAt: &createdAt,
			UpdatedAt: &updatedAt,
			Version:   &version,
			Labels:    map[string]string{"dash0.com/id": "keep-this"},
			Annotations: &IntegrationAnnotations{
				CreatedAt:     &annotCreatedAt,
				LastUpdatedAt: &annotUpdatedAt,
			},
		},
	}

	StripIntegrationServerFields(def)

	if def.Metadata.CreatedAt != nil {
		t.Error("CreatedAt should be nil")
	}
	if def.Metadata.UpdatedAt != nil {
		t.Error("UpdatedAt should be nil")
	}
	if def.Metadata.Version != nil {
		t.Error("Version should be nil")
	}
	if def.Metadata.Annotations.CreatedAt != nil {
		t.Error("Annotations.CreatedAt should be nil")
	}
	if def.Metadata.Annotations.LastUpdatedAt != nil {
		t.Error("Annotations.LastUpdatedAt should be nil")
	}
	if def.Metadata.Labels["dash0.com/id"] != "keep-this" {
		t.Error("Labels should be preserved")
	}
}

func TestStripIntegrationServerFields_Nil(t *testing.T) {
	StripIntegrationServerFields(nil) // should not panic
}

func TestStripIntegrationServerFields_NilAnnotations(t *testing.T) {
	def := &IntegrationDefinition{}
	StripIntegrationServerFields(def) // should not panic
}

func TestGetIntegrationID(t *testing.T) {
	tests := []struct {
		name string
		def  *IntegrationDefinition
		want string
	}{
		{
			"with ID",
			&IntegrationDefinition{Metadata: IntegrationMetadata{
				Labels: map[string]string{"dash0.com/id": "int-123"},
			}},
			"int-123",
		},
		{"nil definition", nil, ""},
		{"nil labels", &IntegrationDefinition{}, ""},
		{
			"no ID label",
			&IntegrationDefinition{Metadata: IntegrationMetadata{
				Labels: map[string]string{"other": "value"},
			}},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetIntegrationID(tt.def); got != tt.want {
				t.Errorf("GetIntegrationID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetIntegrationOrigin(t *testing.T) {
	tests := []struct {
		name string
		def  *IntegrationDefinition
		want string
	}{
		{
			"with origin",
			&IntegrationDefinition{Metadata: IntegrationMetadata{
				Labels: map[string]string{"dash0.com/origin": "terraform:aws:123"},
			}},
			"terraform:aws:123",
		},
		{"nil definition", nil, ""},
		{"nil labels", &IntegrationDefinition{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetIntegrationOrigin(tt.def); got != tt.want {
				t.Errorf("GetIntegrationOrigin() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetIntegrationName(t *testing.T) {
	tests := []struct {
		name string
		def  *IntegrationDefinition
		want string
	}{
		{
			"from display name",
			&IntegrationDefinition{Spec: IntegrationSpec{
				Display: IntegrationDisplay{Name: "AWS 123456789012"},
			}},
			"AWS 123456789012",
		},
		{"nil definition", nil, ""},
		{"empty display name", &IntegrationDefinition{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetIntegrationName(tt.def); got != tt.want {
				t.Errorf("GetIntegrationName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetIntegrationID(t *testing.T) {
	def := &IntegrationDefinition{}
	SetIntegrationID(def, "int-456")
	if def.Metadata.Labels == nil || def.Metadata.Labels["dash0.com/id"] != "int-456" {
		t.Error("expected ID to be set")
	}
}

func TestSetIntegrationID_Overwrites(t *testing.T) {
	def := &IntegrationDefinition{
		Metadata: IntegrationMetadata{
			Labels: map[string]string{"dash0.com/id": "old-id"},
		},
	}
	SetIntegrationID(def, "new-id")
	if def.Metadata.Labels["dash0.com/id"] != "new-id" {
		t.Errorf("ID = %q, want %q", def.Metadata.Labels["dash0.com/id"], "new-id")
	}
}

func TestSetIntegrationID_Nil(t *testing.T) {
	SetIntegrationID(nil, "int-456") // should not panic
}

func TestSetIntegrationIDIfAbsent(t *testing.T) {
	def := &IntegrationDefinition{}
	SetIntegrationIDIfAbsent(def, "new-id")
	if def.Metadata.Labels == nil || def.Metadata.Labels["dash0.com/id"] != "new-id" {
		t.Error("expected ID to be set")
	}
}

func TestSetIntegrationIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	def := &IntegrationDefinition{
		Metadata: IntegrationMetadata{
			Labels: map[string]string{"dash0.com/id": "existing-id"},
		},
	}
	SetIntegrationIDIfAbsent(def, "new-id")
	if def.Metadata.Labels["dash0.com/id"] != "existing-id" {
		t.Errorf("ID = %q, want %q (should not overwrite)", def.Metadata.Labels["dash0.com/id"], "existing-id")
	}
}

func TestSetIntegrationIDIfAbsent_Nil(t *testing.T) {
	SetIntegrationIDIfAbsent(nil, "id") // should not panic
}

func TestClearIntegrationID(t *testing.T) {
	def := &IntegrationDefinition{
		Metadata: IntegrationMetadata{
			Labels: map[string]string{"dash0.com/id": "int-123"},
		},
	}
	ClearIntegrationID(def)
	if _, ok := def.Metadata.Labels["dash0.com/id"]; ok {
		t.Error("ID label should be removed")
	}
}

func TestClearIntegrationID_Nil(t *testing.T) {
	ClearIntegrationID(nil) // should not panic
}

func TestClearIntegrationID_NilLabels(t *testing.T) {
	def := &IntegrationDefinition{}
	ClearIntegrationID(def) // should not panic
}

// HTTP-level CRUD tests

func newTestIntegration() *IntegrationDefinition {
	return &IntegrationDefinition{
		Kind: "Dash0Integration",
		Metadata: IntegrationMetadata{
			Name:   "AWS 123456789012 (terraform)",
			Labels: map[string]string{LabelOrigin: "terraform:aws:123456789012"},
		},
		Spec: IntegrationSpec{
			Enabled: true,
			Display: IntegrationDisplay{Name: "AWS 123456789012 (terraform)"},
			AI:      IntegrationAI{Access: "none"},
			Integration: IntegrationInner{
				Kind: "aws",
				Spec: map[string]any{"accountId": "123456789012"},
			},
		},
	}
}

func newIntegrationTestClient(t *testing.T, server *httptest.Server) Client {
	t.Helper()
	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

func TestGetIntegration_Success(t *testing.T) {
	expected := newTestIntegration()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/integrations/terraform:aws:123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer auth_test123" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := newIntegrationTestClient(t, server)
	got, err := client.GetIntegration(context.Background(), "terraform:aws:123", nil)
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if got.Kind != expected.Kind {
		t.Errorf("Kind = %q, want %q", got.Kind, expected.Kind)
	}
	if got.Metadata.Name != expected.Metadata.Name {
		t.Errorf("Name = %q, want %q", got.Metadata.Name, expected.Metadata.Name)
	}
}

func TestGetIntegration_WithDataset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ds := r.URL.Query().Get("dataset")
		if ds != "production" {
			t.Errorf("expected dataset=production, got %q", ds)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(newTestIntegration())
	}))
	defer server.Close()

	client := newIntegrationTestClient(t, server)
	ds := "production"
	_, err := client.GetIntegration(context.Background(), "some-id", &ds)
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
}

func TestGetIntegration_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
	}))
	defer server.Close()

	client := newIntegrationTestClient(t, server)
	_, err := client.GetIntegration(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound, got: %v", err)
	}
}

func TestCreateIntegration_Success(t *testing.T) {
	integration := newTestIntegration()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		expectedPath := "/api/integrations/" + url.PathEscape("terraform:aws:123456789012")
		if r.URL.Path != expectedPath {
			t.Errorf("path = %q, want %q", r.URL.Path, expectedPath)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected Content-Type: %s", r.Header.Get("Content-Type"))
		}
		var body IntegrationDefinition
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Kind != "Dash0Integration" {
			t.Errorf("request body Kind = %q", body.Kind)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(integration)
	}))
	defer server.Close()

	client := newIntegrationTestClient(t, server)
	got, err := client.CreateIntegration(context.Background(), integration, nil)
	if err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}
	if got.Metadata.Name != integration.Metadata.Name {
		t.Errorf("Name = %q, want %q", got.Metadata.Name, integration.Metadata.Name)
	}
}

func TestCreateIntegration_NilIntegration(t *testing.T) {
	client, err := NewClient(
		WithApiUrl("https://example.com"),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	_, err = client.CreateIntegration(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for nil integration")
	}
}

func TestCreateIntegration_EmptyOrigin(t *testing.T) {
	client, err := NewClient(
		WithApiUrl("https://example.com"),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	def := &IntegrationDefinition{Kind: "Dash0Integration"}
	_, err = client.CreateIntegration(context.Background(), def, nil)
	if err == nil {
		t.Fatal("expected error for empty origin")
	}
}

func TestUpdateIntegration_Success(t *testing.T) {
	integration := newTestIntegration()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(integration)
	}))
	defer server.Close()

	client := newIntegrationTestClient(t, server)
	got, err := client.UpdateIntegration(context.Background(), "terraform:aws:123456789012", integration, nil)
	if err != nil {
		t.Fatalf("UpdateIntegration failed: %v", err)
	}
	if got.Kind != "Dash0Integration" {
		t.Errorf("Kind = %q, want %q", got.Kind, "Dash0Integration")
	}
}

func TestDeleteIntegration_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/integrations/int-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newIntegrationTestClient(t, server)
	err := client.DeleteIntegration(context.Background(), "int-123", nil)
	if err != nil {
		t.Fatalf("DeleteIntegration failed: %v", err)
	}
}

func TestDeleteIntegration_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
	}))
	defer server.Close()

	client := newIntegrationTestClient(t, server)
	err := client.DeleteIntegration(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound, got: %v", err)
	}
}

func TestDeleteIntegration_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newIntegrationTestClient(t, server)
	err := client.DeleteIntegration(context.Background(), "int-123", nil)
	if err != nil {
		t.Fatalf("DeleteIntegration with 200 failed: %v", err)
	}
}

func TestIntegrationURL(t *testing.T) {
	c := &client{
		config: &clientConfig{apiUrl: "https://api.example.com"},
	}

	t.Run("without dataset", func(t *testing.T) {
		u := c.integrationURL("terraform:aws:123", nil)
		parsed, err := url.Parse(u)
		if err != nil {
			t.Fatalf("failed to parse URL: %v", err)
		}
		if parsed.Scheme != "https" {
			t.Errorf("scheme = %q, want %q", parsed.Scheme, "https")
		}
		if parsed.Host != "api.example.com" {
			t.Errorf("host = %q, want %q", parsed.Host, "api.example.com")
		}
		if parsed.Path != "/api/integrations/terraform:aws:123" {
			t.Errorf("path = %q, want %q", parsed.Path, "/api/integrations/terraform:aws:123")
		}
		if parsed.RawQuery != "" {
			t.Errorf("query = %q, want empty", parsed.RawQuery)
		}
	})

	t.Run("with dataset", func(t *testing.T) {
		ds := "production"
		u := c.integrationURL("id-1", &ds)
		parsed, err := url.Parse(u)
		if err != nil {
			t.Fatalf("failed to parse URL: %v", err)
		}
		if parsed.Path != "/api/integrations/id-1" {
			t.Errorf("path = %q, want %q", parsed.Path, "/api/integrations/id-1")
		}
		if parsed.Query().Get("dataset") != "production" {
			t.Errorf("dataset = %q, want %q", parsed.Query().Get("dataset"), "production")
		}
	})

	t.Run("with empty dataset pointer", func(t *testing.T) {
		empty := ""
		u := c.integrationURL("id-1", &empty)
		parsed, err := url.Parse(u)
		if err != nil {
			t.Fatalf("failed to parse URL: %v", err)
		}
		if parsed.RawQuery != "" {
			t.Errorf("query = %q, want empty for empty dataset", parsed.RawQuery)
		}
	})
}
