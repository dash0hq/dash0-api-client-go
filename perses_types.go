package dash0

// PersesDashboard represents the Perses Operator PersesDashboard CRD
// (perses.dev/v1alpha1 and perses.dev/v1alpha2).
type PersesDashboard struct {
	APIVersion string                  `json:"apiVersion"`
	Kind       string                  `json:"kind"`
	Metadata   PersesDashboardMetadata `json:"metadata"`
	Spec       map[string]interface{}  `json:"spec"`
}

// PersesDashboardMetadata contains metadata for a PersesDashboard.
type PersesDashboardMetadata struct {
	Name        string            `json:"name,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
