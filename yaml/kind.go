package yaml

import (
	"fmt"

	sigsyaml "sigs.k8s.io/yaml"
)

// kindProbe is a minimal struct that captures only the top-level fields needed
// to detect the document kind, avoiding a full unmarshal into map[string]any.
type kindProbe struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

// DetectKind extracts the "kind" field from raw YAML/JSON bytes.
// When the document has no explicit "kind" (e.g., a check rule exported via
// `check-rules get -o yaml`), the kind is inferred from the document
// structure: the "expression" field is required for check rules and absent
// in all other asset types.
// An empty kind is returned when the input is valid YAML but has no
// recognizable kind.
// An error is returned when the input cannot be parsed as YAML.
func DetectKind(data []byte) (string, error) {
	var probe kindProbe
	if err := sigsyaml.Unmarshal(data, &probe); err != nil {
		return "", fmt.Errorf("failed to detect document kind: %w", err)
	}
	if probe.Kind != "" {
		return probe.Kind, nil
	}
	if probe.Name != "" && probe.Expression != "" {
		return "CheckRule", nil
	}
	return "", nil
}
