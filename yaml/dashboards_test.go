package yaml

import (
	"testing"

	dash0 "github.com/dash0hq/dash0-api-client-go"
)

func TestParseAsDashboard_Native(t *testing.T) {
	data := []byte(`kind: Dashboard
metadata:
  name: My Dashboard
  dash0Extensions:
    id: dash-123
spec:
  display:
    name: My Dashboard
`)
	dashboard, err := ParseAsDashboard(data)
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.Metadata.Name != "My Dashboard" {
		t.Errorf("Name = %q, want %q", dashboard.Metadata.Name, "My Dashboard")
	}
	if dashboard.Metadata.Dash0Extensions == nil || dashboard.Metadata.Dash0Extensions.Id == nil || *dashboard.Metadata.Dash0Extensions.Id != "dash-123" {
		t.Error("expected dash0Extensions.id = dash-123")
	}
}

func TestParseAsDashboard_PersesDashboard(t *testing.T) {
	data := []byte(`apiVersion: perses.dev/v1alpha1
kind: PersesDashboard
metadata:
  name: my-perses-dashboard
  labels:
    dash0.com/id: perses-123
spec:
  display:
    name: My Perses Dashboard
  duration: 5m
  panels: {}
`)
	dashboard, err := ParseAsDashboard(data)
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.Metadata.Name != "My Perses Dashboard" {
		t.Errorf("Name = %q, want %q", dashboard.Metadata.Name, "My Perses Dashboard")
	}
	if dashboard.Metadata.Dash0Extensions == nil || dashboard.Metadata.Dash0Extensions.Id == nil || *dashboard.Metadata.Dash0Extensions.Id != "perses-123" {
		t.Error("expected dash0Extensions.id = perses-123")
	}
}

func TestParseAsDashboard_InvalidYAML(t *testing.T) {
	_, err := ParseAsDashboard([]byte("{{invalid"))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestConvertPersesDashboardToDashboard_WithDisplayName(t *testing.T) {
	perses := &PersesDashboard{
		Metadata: PersesDashboardMetadata{Name: "meta-name"},
		Spec: map[string]interface{}{
			"display": map[string]interface{}{"name": "Display Name"},
		},
	}
	d := ConvertPersesDashboardToDashboard(perses)
	if d.Metadata.Name != "Display Name" {
		t.Errorf("Name = %q, want %q", d.Metadata.Name, "Display Name")
	}
}

func TestConvertPersesDashboardToDashboard_FallsBackToMetadataName(t *testing.T) {
	perses := &PersesDashboard{
		Metadata: PersesDashboardMetadata{Name: "meta-name"},
		Spec:     map[string]interface{}{},
	}
	d := ConvertPersesDashboardToDashboard(perses)
	if d.Metadata.Name != "meta-name" {
		t.Errorf("Name = %q, want %q", d.Metadata.Name, "meta-name")
	}
}

func TestConvertPersesDashboardToDashboard_WithDisplaySectionButNoName(t *testing.T) {
	perses := &PersesDashboard{
		Metadata: PersesDashboardMetadata{Name: "metadata-name"},
		Spec: map[string]interface{}{
			"display": map[string]interface{}{
				"description": "A dashboard without a display name",
			},
		},
	}
	d := ConvertPersesDashboardToDashboard(perses)
	if d.Metadata.Name != "metadata-name" {
		t.Errorf("Name = %q, want %q", d.Metadata.Name, "metadata-name")
	}
	if dash0.GetDashboardName(d) != "metadata-name" {
		t.Errorf("DisplayName = %q, want %q", dash0.GetDashboardName(d), "metadata-name")
	}
}

func TestConvertPersesDashboardToDashboard_V1Alpha2(t *testing.T) {
	perses := &PersesDashboard{
		APIVersion: "perses.dev/v1alpha2",
		Metadata:   PersesDashboardMetadata{Name: "meta-name"},
		Spec: map[string]interface{}{
			"config": map[string]interface{}{
				"display": map[string]interface{}{"name": "V1Alpha2 Name"},
			},
		},
	}
	d := ConvertPersesDashboardToDashboard(perses)
	if d.Metadata.Name != "V1Alpha2 Name" {
		t.Errorf("Name = %q, want %q", d.Metadata.Name, "V1Alpha2 Name")
	}
}

func TestConvertPersesDashboardToDashboard_WithID(t *testing.T) {
	perses := &PersesDashboard{
		Metadata: PersesDashboardMetadata{
			Name:   "test",
			Labels: map[string]string{"dash0.com/id": "my-id"},
		},
		Spec: map[string]interface{}{},
	}
	d := ConvertPersesDashboardToDashboard(perses)
	if d.Metadata.Dash0Extensions == nil || d.Metadata.Dash0Extensions.Id == nil || *d.Metadata.Dash0Extensions.Id != "my-id" {
		t.Error("expected dash0Extensions.id = my-id")
	}
}

func TestConvertPersesDashboardToDashboard_NilSpec(t *testing.T) {
	perses := &PersesDashboard{
		Metadata: PersesDashboardMetadata{Name: "test"},
	}
	d := ConvertPersesDashboardToDashboard(perses)
	if d.Metadata.Name != "test" {
		t.Errorf("Name = %q, want %q", d.Metadata.Name, "test")
	}
}

func TestConvertPersesDashboardToDashboard_WithAnnotations(t *testing.T) {
	perses := &PersesDashboard{
		Metadata: PersesDashboardMetadata{
			Name: "test",
			Annotations: map[string]string{
				"dash0.com/folder-path": "/my/folder",
				"dash0.com/sharing":     "public",
				"unrelated":             "ignored",
			},
		},
		Spec: map[string]interface{}{},
	}
	d := ConvertPersesDashboardToDashboard(perses)
	if d.Metadata.Annotations == nil {
		t.Fatal("expected annotations")
	}
	if d.Metadata.Annotations.Dash0ComfolderPath == nil || *d.Metadata.Annotations.Dash0ComfolderPath != "/my/folder" {
		t.Error("expected folder-path annotation")
	}
	if d.Metadata.Annotations.Dash0Comsharing == nil || *d.Metadata.Annotations.Dash0Comsharing != "public" {
		t.Error("expected sharing annotation")
	}
}

func TestConvertPersesDashboardToDashboard_WithAnnotationsIncludingSource(t *testing.T) {
	perses := &PersesDashboard{
		Metadata: PersesDashboardMetadata{
			Name: "test",
			Annotations: map[string]string{
				"dash0.com/folder-path": "/test/foo/bar",
				"dash0.com/sharing":     "role:basic_member",
				"dash0.com/source":      "terraform",
			},
		},
		Spec: map[string]interface{}{},
	}
	d := ConvertPersesDashboardToDashboard(perses)
	if d.Metadata.Annotations == nil {
		t.Fatal("expected annotations")
	}
	if d.Metadata.Annotations.Dash0ComfolderPath == nil || *d.Metadata.Annotations.Dash0ComfolderPath != "/test/foo/bar" {
		t.Error("expected folder-path annotation")
	}
	if d.Metadata.Annotations.Dash0Comsharing == nil || *d.Metadata.Annotations.Dash0Comsharing != "role:basic_member" {
		t.Error("expected sharing annotation")
	}
	if d.Metadata.Annotations.Dash0Comsource == nil || string(*d.Metadata.Annotations.Dash0Comsource) != "terraform" {
		t.Error("expected source annotation")
	}
}

func TestConvertPersesDashboardToDashboard_WithFolderPathOnly(t *testing.T) {
	perses := &PersesDashboard{
		Metadata: PersesDashboardMetadata{
			Name: "test",
			Annotations: map[string]string{
				"dash0.com/folder-path": "/my/folder",
			},
		},
		Spec: map[string]interface{}{},
	}
	d := ConvertPersesDashboardToDashboard(perses)
	if d.Metadata.Annotations == nil {
		t.Fatal("expected annotations")
	}
	if d.Metadata.Annotations.Dash0ComfolderPath == nil || *d.Metadata.Annotations.Dash0ComfolderPath != "/my/folder" {
		t.Error("expected folder-path annotation")
	}
	if d.Metadata.Annotations.Dash0Comsharing != nil {
		t.Error("sharing should be nil")
	}
	if d.Metadata.Annotations.Dash0Comsource != nil {
		t.Error("source should be nil")
	}
}

func TestConvertPersesDashboardToDashboard_WithNoRelevantAnnotations(t *testing.T) {
	perses := &PersesDashboard{
		Metadata: PersesDashboardMetadata{
			Name: "test",
			Annotations: map[string]string{
				"some-other-annotation": "value",
			},
		},
		Spec: map[string]interface{}{},
	}
	d := ConvertPersesDashboardToDashboard(perses)
	if d.Metadata.Annotations != nil {
		t.Error("expected nil annotations when no relevant annotations present")
	}
}

func TestGetPersesDashboardName(t *testing.T) {
	tests := []struct {
		name   string
		perses *PersesDashboard
		want   string
	}{
		{
			"from display name",
			&PersesDashboard{
				Metadata: PersesDashboardMetadata{Name: "meta"},
				Spec:     map[string]interface{}{"display": map[string]interface{}{"name": "Display"}},
			},
			"Display",
		},
		{
			"falls back to metadata name",
			&PersesDashboard{
				Metadata: PersesDashboardMetadata{Name: "meta"},
				Spec:     map[string]interface{}{},
			},
			"meta",
		},
		{
			"v1alpha2 with config wrapper",
			&PersesDashboard{
				Metadata: PersesDashboardMetadata{Name: "meta"},
				Spec: map[string]interface{}{
					"config": map[string]interface{}{
						"display": map[string]interface{}{"name": "V1Alpha2 Name"},
					},
				},
			},
			"V1Alpha2 Name",
		},
		{"nil perses", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPersesDashboardName(tt.perses); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClearPersesDashboardID(t *testing.T) {
	perses := &PersesDashboard{Metadata: PersesDashboardMetadata{Labels: map[string]string{"dash0.com/id": "abc"}}}
	ClearPersesDashboardID(perses)
	if _, ok := perses.Metadata.Labels["dash0.com/id"]; ok {
		t.Error("dash0.com/id should be removed")
	}
}

func TestClearPersesDashboardID_NilLabels(t *testing.T) {
	perses := &PersesDashboard{}
	ClearPersesDashboardID(perses) // should not panic
}

func TestSetPersesDashboardID(t *testing.T) {
	perses := &PersesDashboard{}
	SetPersesDashboardID(perses, "new-id")
	if perses.Metadata.Labels == nil {
		t.Fatal("expected labels to be initialized")
	}
	if perses.Metadata.Labels["dash0.com/id"] != "new-id" {
		t.Errorf("ID = %q, want %q", perses.Metadata.Labels["dash0.com/id"], "new-id")
	}
}

func TestSetPersesDashboardID_Overwrites(t *testing.T) {
	perses := &PersesDashboard{
		Metadata: PersesDashboardMetadata{
			Labels: map[string]string{"dash0.com/id": "existing-id"},
		},
	}
	SetPersesDashboardID(perses, "new-id")
	if perses.Metadata.Labels["dash0.com/id"] != "new-id" {
		t.Errorf("ID = %q, want %q", perses.Metadata.Labels["dash0.com/id"], "new-id")
	}
}

func TestSetPersesDashboardIDIfAbsent(t *testing.T) {
	perses := &PersesDashboard{}
	SetPersesDashboardIDIfAbsent(perses, "new-id")
	if perses.Metadata.Labels == nil {
		t.Fatal("expected labels to be initialized")
	}
	if perses.Metadata.Labels["dash0.com/id"] != "new-id" {
		t.Errorf("ID = %q, want %q", perses.Metadata.Labels["dash0.com/id"], "new-id")
	}
}

func TestSetPersesDashboardIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	perses := &PersesDashboard{
		Metadata: PersesDashboardMetadata{
			Labels: map[string]string{"dash0.com/id": "existing-id"},
		},
	}
	SetPersesDashboardIDIfAbsent(perses, "new-id")
	if perses.Metadata.Labels["dash0.com/id"] != "existing-id" {
		t.Errorf("ID = %q, want %q (should not overwrite)", perses.Metadata.Labels["dash0.com/id"], "existing-id")
	}
}

func TestGetPersesDashboardID(t *testing.T) {
	perses := &PersesDashboard{Metadata: PersesDashboardMetadata{Labels: map[string]string{"dash0.com/id": "abc"}}}
	if got := GetPersesDashboardID(perses); got != "abc" {
		t.Errorf("got %q, want %q", got, "abc")
	}
	if got := GetPersesDashboardID(&PersesDashboard{}); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := GetPersesDashboardID(nil); got != "" {
		t.Errorf("got %q, want empty for nil", got)
	}
}
