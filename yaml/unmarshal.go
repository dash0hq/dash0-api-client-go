package yaml

import (
	"bytes"
	"errors"
	"io"

	yamlv3 "gopkg.in/yaml.v3"
	sigsyaml "sigs.k8s.io/yaml"
)

func Unmarshal(data []byte, v any) error {
	return sigsyaml.Unmarshal(quoteYAML11Bools(data), v)
}

func ToJSON(data []byte) ([]byte, error) {
	return sigsyaml.YAMLToJSON(quoteYAML11Bools(data))
}

var yaml11Bools = map[string]bool{
	"y": true, "Y": true, "yes": true, "Yes": true, "YES": true,
	"n": true, "N": true, "no": true, "No": true, "NO": true,
	"on": true, "On": true, "ON": true,
	"off": true, "Off": true, "OFF": true,
}

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
