package yaml

import (
	"bytes"
	"errors"
	"io"

	yamlv3 "gopkg.in/yaml.v3"
	sigsyaml "sigs.k8s.io/yaml"
)

// Unmarshal decodes YAML or JSON into v, honoring v's JSON struct tags.
// Unlike sigs.k8s.io/yaml.Unmarshal, plain y, n, yes, no, on, and off stay
// strings, as YAML 1.2 defines them. Use it instead of sigs.k8s.io/yaml to
// decode any user-authored asset document.
func Unmarshal(data []byte, v any) error {
	return sigsyaml.Unmarshal(quoteYAML11Bools(data), v)
}

// ToJSON converts YAML or JSON to JSON with the same rules as Unmarshal.
func ToJSON(data []byte) ([]byte, error) {
	return sigsyaml.YAMLToJSON(quoteYAML11Bools(data))
}

// yaml11Bools are the plain scalars that YAML 1.2 reads as strings but
// YAML 1.1, and so sigs.k8s.io/yaml, reads as booleans. Unquoted, a
// dashboard grid item's `y: 0` would otherwise decode as `"true": 0`.
var yaml11Bools = map[string]bool{
	"y": true, "Y": true, "yes": true, "Yes": true, "YES": true,
	"n": true, "N": true, "no": true, "No": true, "NO": true,
	"on": true, "On": true, "ON": true,
	"off": true, "Off": true, "OFF": true,
}

// quoteYAML11Bools double-quotes every plain scalar in data that YAML 1.2
// reads as a string but YAML 1.1 reads as a boolean. Input that needs no
// quoting, or that does not parse, is returned unchanged, so JSON input and
// parse errors behave as before.
func quoteYAML11Bools(data []byte) []byte {
	var docs []*yamlv3.Node
	changed := false
	decoder := yamlv3.NewDecoder(bytes.NewReader(data))
	for {
		var node yamlv3.Node
		err := decoder.Decode(&node)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return data
		}
		if quoteNode(&node) {
			changed = true
		}
		docs = append(docs, &node)
	}
	if !changed {
		return data
	}

	var buf bytes.Buffer
	encoder := yamlv3.NewEncoder(&buf)
	encoder.SetIndent(2)
	for _, doc := range docs {
		if err := encoder.Encode(doc); err != nil {
			return data
		}
	}
	if err := encoder.Close(); err != nil {
		return data
	}
	return buf.Bytes()
}

func quoteNode(node *yamlv3.Node) bool {
	changed := false
	if node.Kind == yamlv3.ScalarNode && node.Style == 0 && node.Tag == "!!str" && yaml11Bools[node.Value] {
		node.Style = yamlv3.DoubleQuotedStyle
		changed = true
	}
	for _, child := range node.Content {
		if quoteNode(child) {
			changed = true
		}
	}
	return changed
}
