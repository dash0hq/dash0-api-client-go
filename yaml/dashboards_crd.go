package yaml

import (
	"fmt"
	"strings"

	dash0 "github.com/dash0hq/dash0-api-client-go"
	sigsyaml "sigs.k8s.io/yaml"
)

// ParseAsDashboard detects whether data is a Dashboard or PersesDashboard
// CRD, unmarshals it, and returns a normalized DashboardDefinition ready for
// the API.
// PersesDashboard CRDs are converted via [dash0.ConvertPersesDashboardToDashboard].
func ParseAsDashboard(data []byte) (*dash0.DashboardDefinition, error) {
	detectedKind, err := DetectKind(data)
	if err != nil {
		return nil, err
	}
	kind := strings.ToLower(detectedKind)
	if kind == "persesdashboard" {
		var perses dash0.PersesDashboard
		if err := sigsyaml.Unmarshal(data, &perses); err != nil {
			return nil, fmt.Errorf("failed to parse PersesDashboard definition: %w", err)
		}
		return dash0.ConvertPersesDashboardToDashboard(&perses), nil
	}

	var dashboard dash0.DashboardDefinition
	if err := sigsyaml.Unmarshal(data, &dashboard); err != nil {
		return nil, fmt.Errorf("failed to parse dashboard definition: %w", err)
	}
	return &dashboard, nil
}
