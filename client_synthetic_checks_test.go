package dash0

import (
	"testing"
	"time"
)

func TestStripSyntheticCheckServerFields(t *testing.T) {
	deletedAt := time.Now()
	version := "2"
	dataset := "ds"
	origin := "my-origin"
	custom := map[string]string{"key": "val"}

	c := &SyntheticCheckDefinition{
		Metadata: SyntheticCheckMetadata{
			Annotations: &SyntheticCheckAnnotations{Dash0ComdeletedAt: &deletedAt},
			Labels: &SyntheticCheckLabels{
				Dash0Comversion: &version,
				Custom:          &custom,
				Dash0Comdataset: &dataset,
				Dash0Comorigin:  &origin,
				Dash0Comid:      Ptr("keep-this"),
			},
		},
	}

	StripSyntheticCheckServerFields(c)

	if c.Metadata.Annotations.Dash0ComdeletedAt != nil {
		t.Error("Dash0ComdeletedAt should be nil")
	}
	if c.Metadata.Labels.Dash0Comversion != nil {
		t.Error("Dash0Comversion should be nil")
	}
	if c.Metadata.Labels.Custom != nil {
		t.Error("Custom should be nil")
	}
	if c.Metadata.Labels.Dash0Comdataset != nil {
		t.Error("Dash0Comdataset should be nil")
	}
	if c.Metadata.Labels.Dash0Comorigin != nil {
		t.Error("Dash0Comorigin should be nil")
	}
	if c.Metadata.Labels.Dash0Comid == nil || *c.Metadata.Labels.Dash0Comid != "keep-this" {
		t.Error("Dash0Comid should be preserved")
	}
}

func TestStripSyntheticCheckServerFields_NilLabels(t *testing.T) {
	c := &SyntheticCheckDefinition{}
	StripSyntheticCheckServerFields(c) // should not panic
	if c.Metadata.Labels != nil {
		t.Error("Labels should remain nil")
	}
}

func TestClearSyntheticCheckID(t *testing.T) {
	c := &SyntheticCheckDefinition{Metadata: SyntheticCheckMetadata{Labels: &SyntheticCheckLabels{Dash0Comid: Ptr("sc-1")}}}
	ClearSyntheticCheckID(c)
	if c.Metadata.Labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
}

func TestClearSyntheticCheckID_NilLabels(t *testing.T) {
	c := &SyntheticCheckDefinition{}
	ClearSyntheticCheckID(c) // should not panic
}

func TestGetSyntheticCheckID(t *testing.T) {
	tests := []struct {
		name  string
		check *SyntheticCheckDefinition
		want  string
	}{
		{
			"with ID",
			&SyntheticCheckDefinition{Metadata: SyntheticCheckMetadata{Labels: &SyntheticCheckLabels{Dash0Comid: Ptr("sc-123")}}},
			"sc-123",
		},
		{"nil labels", &SyntheticCheckDefinition{}, ""},
		{"nil ID", &SyntheticCheckDefinition{Metadata: SyntheticCheckMetadata{Labels: &SyntheticCheckLabels{}}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSyntheticCheckID(tt.check); got != tt.want {
				t.Errorf("GetSyntheticCheckID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetSyntheticCheckID(t *testing.T) {
	c := &SyntheticCheckDefinition{}
	SetSyntheticCheckID(c, "new-id")
	if c.Metadata.Labels == nil || c.Metadata.Labels.Dash0Comid == nil {
		t.Fatal("expected ID to be set")
	}
	if *c.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *c.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetSyntheticCheckID_Overwrites(t *testing.T) {
	c := &SyntheticCheckDefinition{
		Metadata: SyntheticCheckMetadata{Labels: &SyntheticCheckLabels{Dash0Comid: Ptr("existing-id")}},
	}
	SetSyntheticCheckID(c, "new-id")
	if *c.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *c.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetSyntheticCheckIDIfAbsent(t *testing.T) {
	c := &SyntheticCheckDefinition{}
	SetSyntheticCheckIDIfAbsent(c, "new-id")
	if c.Metadata.Labels == nil || c.Metadata.Labels.Dash0Comid == nil {
		t.Fatal("expected ID to be set")
	}
	if *c.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *c.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetSyntheticCheckIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	c := &SyntheticCheckDefinition{
		Metadata: SyntheticCheckMetadata{Labels: &SyntheticCheckLabels{Dash0Comid: Ptr("existing-id")}},
	}
	SetSyntheticCheckIDIfAbsent(c, "new-id")
	if *c.Metadata.Labels.Dash0Comid != "existing-id" {
		t.Errorf("ID = %q, want %q (should not overwrite)", *c.Metadata.Labels.Dash0Comid, "existing-id")
	}
}

func TestGetSyntheticCheckName(t *testing.T) {
	tests := []struct {
		name  string
		check *SyntheticCheckDefinition
		want  string
	}{
		{
			"from display name",
			&SyntheticCheckDefinition{Spec: SyntheticCheckSpec{Display: &SyntheticCheckDisplay{Name: "Display Name"}}},
			"Display Name",
		},
		{
			"falls back to metadata name",
			&SyntheticCheckDefinition{Metadata: SyntheticCheckMetadata{Name: "meta-name"}},
			"meta-name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSyntheticCheckName(tt.check); got != tt.want {
				t.Errorf("GetSyntheticCheckName() = %q, want %q", got, tt.want)
			}
		})
	}
}
