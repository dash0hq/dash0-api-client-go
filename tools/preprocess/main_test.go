package main

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestRemoveOperationIDsKeepsSchemaProperties(t *testing.T) {
	spec := `
paths:
  /api/datasets:
    get:
      operationId: getOrganizationDatasets
    post:
      operationId: postOrganizationDataset
components:
  schemas:
    Job:
      properties:
        operationId:
          type: string
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(spec), &doc); err != nil {
		t.Fatal(err)
	}
	if got := removeOperationIDs(&doc); got != 2 {
		t.Fatalf("removed %d operationIds, want 2", got)
	}
	out, err := yaml.Marshal(&doc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(out), "operationId") != 1 {
		t.Fatalf("want only the schema property left, got:\n%s", out)
	}
}

func TestKeepKeyMatcherValue(t *testing.T) {
	spec := `
components:
  schemas:
    TimeSeriesAggregationAttributeModificationSpec:
      properties:
        keyMatcher:
          $ref: "#/components/schemas/Matcher"
          x-omitzero: false
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(spec), &doc); err != nil {
		t.Fatal(err)
	}
	if err := keepKeyMatcherValue(&doc); err != nil {
		t.Fatal(err)
	}
	out, err := yaml.Marshal(&doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"x-go-type-skip-optional-pointer: true", "x-omitzero: true"} {
		if strings.Count(string(out), want) != 1 {
			t.Errorf("want %q once, got:\n%s", want, out)
		}
	}

	var empty yaml.Node
	if err := yaml.Unmarshal([]byte("components: {}"), &empty); err != nil {
		t.Fatal(err)
	}
	if err := keepKeyMatcherValue(&empty); err == nil {
		t.Error("want an error when keyMatcher is missing")
	}
}
