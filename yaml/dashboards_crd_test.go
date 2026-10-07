package yaml

import (
	"testing"
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

func TestParseAsDashboard_NativeFolderPath(t *testing.T) {
	data := []byte(`kind: Dashboard
metadata:
  name: My Dashboard
  annotations:
    dash0.com/folder-path: /team/sre
spec:
  display:
    name: My Dashboard
`)
	dashboard, err := ParseAsDashboard(data)
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.Metadata.Annotations == nil || dashboard.Metadata.Annotations.Dash0ComfolderPath == nil {
		t.Fatal("expected folder-path annotation")
	}
	if *dashboard.Metadata.Annotations.Dash0ComfolderPath != "/team/sre" {
		t.Errorf("folder-path = %q, want %q", *dashboard.Metadata.Annotations.Dash0ComfolderPath, "/team/sre")
	}
}

func TestParseAsDashboard_PersesFolderPath(t *testing.T) {
	data := []byte(`apiVersion: perses.dev/v1alpha1
kind: PersesDashboard
metadata:
  name: my-perses-dashboard
  annotations:
    dash0.com/folder-path: /team/sre
spec:
  display:
    name: My Perses Dashboard
`)
	dashboard, err := ParseAsDashboard(data)
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.Metadata.Annotations == nil || dashboard.Metadata.Annotations.Dash0ComfolderPath == nil {
		t.Fatal("expected folder-path annotation")
	}
	if *dashboard.Metadata.Annotations.Dash0ComfolderPath != "/team/sre" {
		t.Errorf("folder-path = %q, want %q", *dashboard.Metadata.Annotations.Dash0ComfolderPath, "/team/sre")
	}
}

func TestParseAsDashboard_InvalidYAML(t *testing.T) {
	_, err := ParseAsDashboard([]byte("{{invalid"))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseAsDashboard_UnquotedYGridKey(t *testing.T) {
	dashboard, err := ParseAsDashboard([]byte(`kind: Dashboard
metadata:
  name: d
spec:
  layouts:
    - kind: Grid
      spec:
        items:
          - x: 0
            y: 3
`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	items := dashboard.Spec["layouts"].([]any)[0].(map[string]any)["spec"].(map[string]any)["items"].([]any)
	if got := items[0].(map[string]any); got["y"] != float64(3) {
		t.Errorf("grid item = %v, want key y = 3", got)
	}
}
