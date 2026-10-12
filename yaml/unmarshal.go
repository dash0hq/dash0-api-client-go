package yaml

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"unicode/utf8"

	yamlv3 "go.yaml.in/yaml/v3"
	sigsyaml "sigs.k8s.io/yaml"
)

func Unmarshal(data []byte, v any) error {
	return sigsyaml.Unmarshal(quoteYAML11BoolKeys(data), v)
}

func ToJSON(data []byte) ([]byte, error) {
	var v any
	if err := yamlv3.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("error marshaling to JSON: %w", err)
	}
	return out, nil
}

var yaml11Bools = map[string]bool{
	"y": true, "Y": true, "yes": true, "Yes": true, "YES": true,
	"n": true, "N": true, "no": true, "No": true, "NO": true,
	"on": true, "On": true, "ON": true,
	"off": true, "Off": true, "OFF": true,
}

var yaml11BoolToken = regexp.MustCompile(`\b(?:[yYnN]|yes|Yes|YES|no|No|NO|on|On|ON|off|Off|OFF)\b`)

func quoteYAML11BoolKeys(data []byte) []byte {
	if !yaml11BoolToken.Match(data) || json.Valid(data) {
		return data
	}

	var doc yamlv3.Node
	if err := yamlv3.NewDecoder(bytes.NewReader(data)).Decode(&doc); err != nil {
		return data
	}

	var keys []*yamlv3.Node
	collectYAML11BoolKeys(&doc, &keys)

	starts := make([]int, 0, len(keys))
	ends := map[int]int{}
	for _, key := range keys {
		start, ok := byteOffset(data, key.Line, key.Column)
		if !ok || !bytes.HasPrefix(data[start:], []byte(key.Value)) {
			return data
		}
		starts = append(starts, start)
		ends[start] = start + len(key.Value)
	}
	if len(starts) == 0 {
		return data
	}
	sort.Sort(sort.Reverse(sort.IntSlice(starts)))

	out := bytes.Clone(data)
	for _, start := range starts {
		out = insertQuote(out, ends[start])
		out = insertQuote(out, start)
	}
	return out
}

func collectYAML11BoolKeys(node *yamlv3.Node, keys *[]*yamlv3.Node) {
	if node.Kind == yamlv3.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind == yamlv3.ScalarNode && key.Style == 0 && key.Tag == "!!str" && yaml11Bools[key.Value] {
				*keys = append(*keys, key)
			}
		}
	}
	for _, child := range node.Content {
		collectYAML11BoolKeys(child, keys)
	}
}

func byteOffset(data []byte, line, column int) (int, bool) {
	offset := 0
	for l := 1; l < line; l++ {
		i := bytes.IndexByte(data[offset:], '\n')
		if i < 0 {
			return 0, false
		}
		offset += i + 1
	}
	for c := 1; c < column; c++ {
		if offset >= len(data) || data[offset] == '\n' {
			return 0, false
		}
		_, size := utf8.DecodeRune(data[offset:])
		offset += size
	}
	return offset, true
}

func insertQuote(data []byte, at int) []byte {
	return append(data[:at], append([]byte{'"'}, data[at:]...)...)
}
