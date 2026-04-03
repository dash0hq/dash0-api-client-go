package dash0

import (
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
