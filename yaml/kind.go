package yaml

import (
	sigsyaml "sigs.k8s.io/yaml"
)

// DetectKind extracts the "kind" field from raw YAML/JSON bytes. When the
// document has no explicit "kind" (e.g. a check rule exported via
// `check-rules get -o yaml`), the kind is inferred from the document
// structure: the "expression" field is required for check rules and absent
// in all other asset types.
func DetectKind(data []byte) string {
	var doc map[string]any
	if err := sigsyaml.Unmarshal(data, &doc); err != nil {
		return ""
	}
	if kind, ok := doc["kind"].(string); ok && kind != "" {
		return kind
	}
	_, hasName := doc["name"]
	_, hasExpr := doc["expression"]
	if hasName && hasExpr {
		return "CheckRule"
	}
	return ""
}
