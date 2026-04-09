package dash0

import (
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
