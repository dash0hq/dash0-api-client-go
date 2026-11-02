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
