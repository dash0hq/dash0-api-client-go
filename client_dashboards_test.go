package dash0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStripDashboardServerFields(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	deletedAt := time.Now()
	version := int64(3)
	dataset := "my-dataset"

	d := &DashboardDefinition{
		Metadata: DashboardMetadata{
			CreatedAt:   &createdAt,
			UpdatedAt:   &updatedAt,
			Version:     &version,
			Annotations: &DashboardAnnotations{Dash0ComdeletedAt: &deletedAt},
			Dash0Extensions: &DashboardMetadataExtensions{
				Dataset: &dataset,
				Id:      Ptr("keep-this"),
			},
		},
	}

	StripDashboardServerFields(d)

	if d.Metadata.CreatedAt != nil {
		t.Error("CreatedAt should be nil")
	}
	if d.Metadata.UpdatedAt != nil {
		t.Error("UpdatedAt should be nil")
	}
	if d.Metadata.Version != nil {
		t.Error("Version should be nil")
	}
	if d.Metadata.Annotations.Dash0ComdeletedAt != nil {
		t.Error("Dash0ComdeletedAt should be nil")
	}
	if d.Metadata.Dash0Extensions.Dataset != nil {
		t.Error("Dataset should be nil")
	}
	if d.Metadata.Dash0Extensions.Id == nil || *d.Metadata.Dash0Extensions.Id != "keep-this" {
		t.Error("Id should be preserved")
	}
}

func TestClearDashboardID(t *testing.T) {
	id := "my-id"
	d := &DashboardDefinition{
		Metadata: DashboardMetadata{
			Dash0Extensions: &DashboardMetadataExtensions{Id: &id},
		},
	}

	ClearDashboardID(d)

	if d.Metadata.Dash0Extensions.Id != nil {
		t.Error("Id should be nil")
	}
}

func TestClearDashboardID_NilExtensions(t *testing.T) {
	d := &DashboardDefinition{}
	ClearDashboardID(d) // should not panic
}

func TestGetDashboardFolderPath(t *testing.T) {
	tests := []struct {
		name      string
		dashboard *DashboardDefinition
		want      string
	}{
		{"nil dashboard", nil, ""},
		{"nil annotations", &DashboardDefinition{}, ""},
		{"nil folder-path", &DashboardDefinition{Metadata: DashboardMetadata{Annotations: &DashboardAnnotations{}}}, ""},
		{"with folder-path", &DashboardDefinition{Metadata: DashboardMetadata{Annotations: &DashboardAnnotations{Dash0ComfolderPath: Ptr("/team/sre")}}}, "/team/sre"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDashboardFolderPath(tt.dashboard); got != tt.want {
				t.Errorf("GetDashboardFolderPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetDashboardFolderPath(t *testing.T) {
	dashboard := &DashboardDefinition{}
	SetDashboardFolderPath(dashboard, "/team/sre")
	if dashboard.Metadata.Annotations == nil || dashboard.Metadata.Annotations.Dash0ComfolderPath == nil {
		t.Fatal("expected folder-path to be set")
	}
	if *dashboard.Metadata.Annotations.Dash0ComfolderPath != "/team/sre" {
		t.Errorf("FolderPath = %q, want %q", *dashboard.Metadata.Annotations.Dash0ComfolderPath, "/team/sre")
	}
}

func TestSetDashboardFolderPath_Nil(t *testing.T) {
	SetDashboardFolderPath(nil, "/team/sre") // should not panic
}

func TestGetDashboardDataset(t *testing.T) {
	tests := []struct {
		name      string
		dashboard *DashboardDefinition
		want      string
	}{
		{"nil dashboard", nil, ""},
		{"nil extensions", &DashboardDefinition{}, ""},
		{"nil dataset", &DashboardDefinition{Metadata: DashboardMetadata{Dash0Extensions: &DashboardMetadataExtensions{}}}, ""},
		{"with dataset", &DashboardDefinition{Metadata: DashboardMetadata{Dash0Extensions: &DashboardMetadataExtensions{Dataset: Ptr("production")}}}, "production"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDashboardDataset(tt.dashboard); got != tt.want {
				t.Errorf("GetDashboardDataset() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetDashboardDataset(t *testing.T) {
	dashboard := &DashboardDefinition{}
	SetDashboardDataset(dashboard, "production")
	if dashboard.Metadata.Dash0Extensions == nil || dashboard.Metadata.Dash0Extensions.Dataset == nil {
		t.Fatal("expected dataset to be set")
	}
	if *dashboard.Metadata.Dash0Extensions.Dataset != "production" {
		t.Errorf("Dataset = %q, want %q", *dashboard.Metadata.Dash0Extensions.Dataset, "production")
	}
}

func TestSetDashboardDataset_Nil(t *testing.T) {
	SetDashboardDataset(nil, "production") // should not panic
}

func TestGetDashboardID(t *testing.T) {
	tests := []struct {
		name      string
		dashboard *DashboardDefinition
		want      string
	}{
		{
			"with ID",
			&DashboardDefinition{Metadata: DashboardMetadata{
				Dash0Extensions: &DashboardMetadataExtensions{Id: Ptr("abc-123")},
			}},
			"abc-123",
		},
		{"nil dashboard", nil, ""},
		{"nil extensions", &DashboardDefinition{}, ""},
		{
			"nil ID",
			&DashboardDefinition{Metadata: DashboardMetadata{
				Dash0Extensions: &DashboardMetadataExtensions{},
			}},
			"",
		},
		{
			"empty ID",
			&DashboardDefinition{Metadata: DashboardMetadata{
				Dash0Extensions: &DashboardMetadataExtensions{Id: Ptr("")},
			}},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDashboardID(tt.dashboard); got != tt.want {
				t.Errorf("GetDashboardID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetDashboardID(t *testing.T) {
	d := &DashboardDefinition{}
	SetDashboardID(d, "new-id")
	if d.Metadata.Dash0Extensions == nil || d.Metadata.Dash0Extensions.Id == nil {
		t.Fatal("expected ID to be set")
	}
	if *d.Metadata.Dash0Extensions.Id != "new-id" {
		t.Errorf("ID = %q, want %q", *d.Metadata.Dash0Extensions.Id, "new-id")
	}
}

func TestSetDashboardID_Overwrites(t *testing.T) {
	d := &DashboardDefinition{
		Metadata: DashboardMetadata{
			Dash0Extensions: &DashboardMetadataExtensions{Id: Ptr("existing-id")},
		},
	}
	SetDashboardID(d, "new-id")
	if *d.Metadata.Dash0Extensions.Id != "new-id" {
		t.Errorf("ID = %q, want %q", *d.Metadata.Dash0Extensions.Id, "new-id")
	}
}

func TestSetDashboardIDIfAbsent(t *testing.T) {
	d := &DashboardDefinition{}
	SetDashboardIDIfAbsent(d, "new-id")
	if d.Metadata.Dash0Extensions == nil || d.Metadata.Dash0Extensions.Id == nil {
		t.Fatal("expected ID to be set")
	}
	if *d.Metadata.Dash0Extensions.Id != "new-id" {
		t.Errorf("ID = %q, want %q", *d.Metadata.Dash0Extensions.Id, "new-id")
	}
}

func TestSetDashboardIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	d := &DashboardDefinition{
		Metadata: DashboardMetadata{
			Dash0Extensions: &DashboardMetadataExtensions{Id: Ptr("existing-id")},
		},
	}
	SetDashboardIDIfAbsent(d, "new-id")
	if *d.Metadata.Dash0Extensions.Id != "existing-id" {
		t.Errorf("ID = %q, want %q (should not overwrite)", *d.Metadata.Dash0Extensions.Id, "existing-id")
	}
}

func TestGetDashboardName(t *testing.T) {
	tests := []struct {
		name      string
		dashboard *DashboardDefinition
		want      string
	}{
		{
			"from spec display name",
			&DashboardDefinition{Spec: map[string]any{
				"display": map[string]any{"name": "My Dashboard"},
			}},
			"My Dashboard",
		},
		{"nil dashboard", nil, ""},
		{"nil spec", &DashboardDefinition{}, ""},
		{"no display section", &DashboardDefinition{Spec: map[string]any{}}, ""},
		{
			"display without name",
			&DashboardDefinition{Spec: map[string]any{"display": map[string]any{}}},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDashboardName(tt.dashboard); got != tt.want {
				t.Errorf("GetDashboardName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateDashboard_201(t *testing.T) {
	dashboard := DashboardDefinition{
		Metadata: DashboardMetadata{
			Dash0Extensions: &DashboardMetadataExtensions{
				Id: Ptr("db-123"),
			},
		},
		Spec: map[string]any{
			"display": map[string]any{"name": "Test Dashboard"},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(dashboard)
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	got, err := client.CreateDashboard(context.Background(), &dashboard, nil)
	if err != nil {
		t.Fatalf("CreateDashboard failed: %v", err)
	}

	if GetDashboardName(got) != "Test Dashboard" {
		t.Errorf("expected name %q, got %q", "Test Dashboard", GetDashboardName(got))
	}
	if GetDashboardID(got) != "db-123" {
		t.Errorf("expected ID %q, got %q", "db-123", GetDashboardID(got))
	}
}
