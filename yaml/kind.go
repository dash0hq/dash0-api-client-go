package yaml

import (
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
func DetectKind(data []byte) string {
	var probe kindProbe
	if err := sigsyaml.Unmarshal(data, &probe); err != nil {
		return ""
	}
	if probe.Kind != "" {
		return probe.Kind
	}
	if probe.Name != "" && probe.Expression != "" {
		return "CheckRule"
	}
	return ""
}
