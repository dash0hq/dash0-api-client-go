package dash0

// ConvertPersesDashboardToDashboard converts a PersesDashboard CRD into a Dash0
// DashboardDefinition.
// It normalizes v1alpha1/v1alpha2 differences (the v1alpha2 CRD wraps the spec
// in a "config" key) and ensures a display name is set, falling back to
// metadata.name.
func ConvertPersesDashboardToDashboard(perses *PersesDashboard) *DashboardDefinition {
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

	dashboard := &DashboardDefinition{
		Kind: Dashboard,
		Metadata: DashboardMetadata{
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

	// Copy dash0.com/id and dash0.com/dataset from labels into dash0Extensions.
	if perses.Metadata.Labels != nil {
		id := perses.Metadata.Labels["dash0.com/id"]
		ds := perses.Metadata.Labels["dash0.com/dataset"]
		if id != "" || ds != "" {
			ext := &DashboardMetadataExtensions{}
			if id != "" {
				ext.Id = &id
			}
			if ds != "" {
				dataset := Dataset(ds)
				ext.Dataset = &dataset
			}
			dashboard.Metadata.Dash0Extensions = ext
		}
	}

	return dashboard
}

// convertPersesDashboardAnnotations converts the untyped annotation map from a
// PersesDashboard CRD into the typed DashboardAnnotations struct, copying only
// user-settable annotations (folder-path, sharing, source).
func convertPersesDashboardAnnotations(annotations map[string]string) *DashboardAnnotations {
	if len(annotations) == 0 {
		return nil
	}
	var result DashboardAnnotations
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
		source := DashboardSource(v)
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
	if perses == nil {
		return ""
	}
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
	if perses == nil || perses.Metadata.Labels == nil {
		return ""
	}
	return perses.Metadata.Labels["dash0.com/id"]
}

// ClearPersesDashboardID removes the dash0.com/id label from a PersesDashboard CRD.
func ClearPersesDashboardID(perses *PersesDashboard) {
	if perses == nil {
		return
	}
	if perses.Metadata.Labels != nil {
		delete(perses.Metadata.Labels, "dash0.com/id")
	}
}

// SetPersesDashboardID sets the dash0.com/id label on a PersesDashboard CRD,
// initializing the labels map if needed.
func SetPersesDashboardID(perses *PersesDashboard, id string) {
	if perses == nil {
		return
	}
	if perses.Metadata.Labels == nil {
		perses.Metadata.Labels = map[string]string{}
	}
	perses.Metadata.Labels["dash0.com/id"] = id
}

// SetPersesDashboardIDIfAbsent sets the dash0.com/id label on a
// PersesDashboard CRD only if it is not already set, initializing the labels
// map if needed.
func SetPersesDashboardIDIfAbsent(perses *PersesDashboard, id string) {
	if perses == nil {
		return
	}
	if perses.Metadata.Labels == nil {
		perses.Metadata.Labels = map[string]string{}
	}
	if _, ok := perses.Metadata.Labels["dash0.com/id"]; !ok {
		perses.Metadata.Labels["dash0.com/id"] = id
	}
}
