package yaml

import (
	"fmt"
	"strings"

	dash0 "github.com/dash0hq/dash0-api-client-go"
	sigsyaml "sigs.k8s.io/yaml"
)

// ConvertPersesDashboardToDashboard converts a PersesDashboard CRD into a Dash0
// DashboardDefinition. It normalizes v1alpha1/v1alpha2 differences (the
// v1alpha2 CRD wraps the spec in a "config" key) and ensures a display name
// is set, falling back to metadata.name.
func ConvertPersesDashboardToDashboard(perses *PersesDashboard) *dash0.DashboardDefinition {
	spec := perses.Spec
	if spec == nil {
		spec = make(map[string]interface{})
	}

	// Normalize v1alpha1/v1alpha2: if spec.config exists, unwrap it.
	// v1alpha2 wraps the dashboard content in spec.config; v1alpha1 puts it
	// directly under spec. After normalization both look the same.
	if configRaw, ok := spec["config"]; ok {
		if config, ok := configRaw.(map[string]interface{}); ok {
			spec = config
		}
	}

	// Ensure display section exists
	displayRaw, hasDisplay := spec["display"]
	if !hasDisplay {
		spec["display"] = map[string]interface{}{
			"name": perses.Metadata.Name,
		}
	} else if display, ok := displayRaw.(map[string]interface{}); ok {
		// Set display.name to metadata.name if missing
		if _, hasName := display["name"]; !hasName {
			display["name"] = perses.Metadata.Name
		}
	}

	displayName := extractDisplayName(spec)
	if displayName == "" {
		displayName = perses.Metadata.Name
	}

	dashboard := &dash0.DashboardDefinition{
		Kind: dash0.Dashboard,
		Metadata: dash0.DashboardMetadata{
			Name: displayName,
		},
		Spec: spec,
	}

	// Copy user-settable annotations (folder-path, sharing, source) from
	// Perses CRD metadata into the typed Dashboard annotations struct.
	annotations := convertPersesDashboardAnnotations(perses.Metadata.Annotations)
	if annotations != nil {
		dashboard.Metadata.Annotations = annotations
	}

	// Copy dash0.com/id from labels into dash0Extensions.id
	if perses.Metadata.Labels != nil {
		if id := perses.Metadata.Labels["dash0.com/id"]; id != "" {
			dashboard.Metadata.Dash0Extensions = &dash0.DashboardMetadataExtensions{
				Id: &id,
			}
		}
	}

	return dashboard
}

// convertPersesDashboardAnnotations converts the untyped annotation map from a
// PersesDashboard CRD into the typed DashboardAnnotations struct, copying only
// user-settable annotations (folder-path, sharing, source).
func convertPersesDashboardAnnotations(annotations map[string]string) *dash0.DashboardAnnotations {
	if len(annotations) == 0 {
		return nil
	}
	var result dash0.DashboardAnnotations
	hasAny := false
	if v, ok := annotations["dash0.com/folder-path"]; ok {
		result.Dash0ComfolderPath = &v
		hasAny = true
	}
	if v, ok := annotations["dash0.com/sharing"]; ok {
		result.Dash0Comsharing = &v
		hasAny = true
	}
	if v, ok := annotations["dash0.com/source"]; ok {
		source := dash0.DashboardSource(v)
		result.Dash0Comsource = &source
		hasAny = true
	}
	if !hasAny {
		return nil
	}
	return &result
}

// extractDisplayName reads spec.display.name from a dashboard spec map.
func extractDisplayName(spec map[string]interface{}) string {
	display, ok := spec["display"].(map[string]interface{})
	if !ok {
		return ""
	}
	name, ok := display["name"].(string)
	if !ok {
		return ""
	}
	return name
}

// GetPersesDashboardName returns the display name from the Perses spec,
// falling back to metadata.name.
func GetPersesDashboardName(perses *PersesDashboard) string {
	if perses.Spec != nil {
		// Check after normalization: handle both v1alpha1 and v1alpha2
		spec := perses.Spec
		if configRaw, ok := spec["config"]; ok {
			if config, ok := configRaw.(map[string]interface{}); ok {
				spec = config
			}
		}
		if name := extractDisplayName(spec); name != "" {
			return name
		}
	}
	return perses.Metadata.Name
}

// GetPersesDashboardID returns the dash0.com/id label value if present.
func GetPersesDashboardID(perses *PersesDashboard) string {
	if perses.Metadata.Labels != nil {
		return perses.Metadata.Labels["dash0.com/id"]
	}
	return ""
}

// ClearPersesDashboardID removes the dash0.com/id label from a PersesDashboard CRD.
func ClearPersesDashboardID(perses *PersesDashboard) {
	if perses.Metadata.Labels != nil {
		delete(perses.Metadata.Labels, "dash0.com/id")
	}
}

// SetPersesDashboardID sets the dash0.com/id label on a PersesDashboard CRD,
// initializing the labels map if needed. It is a no-op if the ID is already set.
func SetPersesDashboardID(perses *PersesDashboard, id string) {
	if perses.Metadata.Labels == nil {
		perses.Metadata.Labels = map[string]string{}
	}
	if _, ok := perses.Metadata.Labels["dash0.com/id"]; !ok {
		perses.Metadata.Labels["dash0.com/id"] = id
	}
}

// ParseAsDashboard detects whether data is a Dashboard or PersesDashboard
// CRD, unmarshals it, and returns a normalized DashboardDefinition ready for
// the API. PersesDashboard CRDs are converted via ConvertPersesDashboardToDashboard.
func ParseAsDashboard(data []byte) (*dash0.DashboardDefinition, error) {
	kind := strings.ToLower(DetectKind(data))
	if kind == "persesdashboard" {
		var perses PersesDashboard
		if err := sigsyaml.Unmarshal(data, &perses); err != nil {
			return nil, fmt.Errorf("failed to parse PersesDashboard definition: %w", err)
		}
		return ConvertPersesDashboardToDashboard(&perses), nil
	}

	var dashboard dash0.DashboardDefinition
	if err := sigsyaml.Unmarshal(data, &dashboard); err != nil {
		return nil, fmt.Errorf("failed to parse dashboard definition: %w", err)
	}
	return &dashboard, nil
}
