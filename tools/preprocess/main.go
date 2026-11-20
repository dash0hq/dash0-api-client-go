// Command preprocess fetches the OpenAPI spec and removes every operation's
// operationId before oapi-codegen runs. oapi-codegen names client methods after
// the operationId when one is present, so keeping them would rename every
// method (GetApiDatasets becomes GetOrganizationDatasets).
// It also keeps one released value field from turning into a pointer
// (see keepKeyMatcherValue).
//
// TODO(v2): remove this tool and generate from the published spec directly.
//
// Usage: go run ./tools/preprocess <spec URL or path> <output path>
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: preprocess <spec URL or path> <output path>")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintf(os.Stderr, "preprocess: %v\n", err)
		os.Exit(1)
	}
}

func run(source, output string) error {
	raw, err := read(source)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse spec: %w", err)
	}
	removed := removeOperationIDs(&doc)
	if err := keepKeyMatcherValue(&doc); err != nil {
		return err
	}
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("encode spec: %w", err)
	}
	fmt.Printf("preprocess: removed %d operationIds\n", removed)
	return os.WriteFile(output, out, 0o644)
}

func read(source string) ([]byte, error) {
	if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
		return os.ReadFile(source)
	}
	resp, err := http.Get(source)
	if err != nil {
		return nil, fmt.Errorf("fetch spec: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch spec: %s returned %s", source, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// removeOperationIDs deletes operationId from each operation under paths. It
// walks paths -> path item -> operation only, so a schema property that happens
// to be named operationId is left alone.
func removeOperationIDs(doc *yaml.Node) int {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return 0
	}
	paths := mappingValue(doc.Content[0], "paths")
	if paths == nil {
		return 0
	}
	removed := 0
	for i := 1; i < len(paths.Content); i += 2 {
		pathItem := paths.Content[i]
		for j := 1; j < len(pathItem.Content); j += 2 {
			if deleteKey(pathItem.Content[j], "operationId") {
				removed++
			}
		}
	}
	return removed
}

// keepKeyMatcherValue keeps TimeSeriesAggregationAttributeModificationSpec.KeyMatcher
// the Matcher value it was released as. The spec made keyMatcher optional for the
// keep_entity and drop_entity kinds, and oapi-codegen turns optional properties
// into pointers. x-go-type-skip-optional-pointer keeps the value type, and
// x-omitzero leaves the zero Matcher out of the JSON. It fails when the property
// is missing, because the field would otherwise turn into a pointer unnoticed.
func keepKeyMatcherValue(doc *yaml.Node) error {
	var root *yaml.Node
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		root = doc.Content[0]
	}
	prop := root
	for _, key := range []string{"components", "schemas", "TimeSeriesAggregationAttributeModificationSpec", "properties", "keyMatcher"} {
		prop = mappingValue(prop, key)
	}
	if prop == nil || prop.Kind != yaml.MappingNode {
		return fmt.Errorf("TimeSeriesAggregationAttributeModificationSpec.keyMatcher not found in the spec")
	}
	for _, hint := range []string{"x-go-type-skip-optional-pointer", "x-omitzero"} {
		deleteKey(prop, hint)
		prop.Content = append(prop.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: hint},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
	}
	return nil
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func deleteKey(node *yaml.Node, key string) bool {
	if node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content = append(node.Content[:i], node.Content[i+2:]...)
			return true
		}
	}
	return false
}
