package main

import (
	"bytes"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

// postprocessSource runs the rename passes and returns the result with runs of whitespace collapsed to one space.
func postprocessSource(t *testing.T, src string) string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "generated.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	renameSymbols(file)
	renameTypedConsts(file)
	renameConsts(file)
	var out bytes.Buffer
	if err := printer.Fprint(&out, fset, file); err != nil {
		t.Fatal(err)
	}
	return strings.Join(strings.Fields(out.String()), " ")
}

func TestRenameTypedConsts_BareNamesGetTheirOwnTypePrefix(t *testing.T) {
	const src = `package dash0

type LinearWebhookAction string
type AwsLogGroupPolicyDisposition string

const (
	Create LinearWebhookAction = "create"
	Update LinearWebhookAction = "update"
)

const (
	Keep   AwsLogGroupPolicyDisposition = "keep"
	Remove AwsLogGroupPolicyDisposition = "remove"
)

func (e LinearWebhookAction) Valid() bool {
	switch e {
	case Create, Update:
		return true
	}
	return false
}

func (e AwsLogGroupPolicyDisposition) Valid() bool {
	switch e {
	case Keep, Remove:
		return true
	}
	return false
}
`
	out := postprocessSource(t, src)
	for _, want := range []string{
		"LinearWebhookActionCreate LinearWebhookAction =",
		"LinearWebhookActionUpdate LinearWebhookAction =",
		"AwsLogGroupPolicyDispositionKeep AwsLogGroupPolicyDisposition =",
		"AwsLogGroupPolicyDispositionRemove AwsLogGroupPolicyDisposition =",
		"case LinearWebhookActionCreate, LinearWebhookActionUpdate:",
		"case AwsLogGroupPolicyDispositionKeep, AwsLogGroupPolicyDispositionRemove:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q: %s", want, out)
		}
	}
	if strings.Contains(out, "LinearWebhookActionRemove") {
		t.Errorf("a disposition constant was renamed into LinearWebhookAction: %s", out)
	}
}

func TestRenameTypedConsts_KeepsPrefixedNames(t *testing.T) {
	const src = `package dash0

type LinearWebhookAction string

const (
	LinearWebhookActionRemove LinearWebhookAction = "remove"
)

func (e LinearWebhookAction) Valid() bool {
	switch e {
	case LinearWebhookActionRemove:
		return true
	}
	return false
}
`
	out := postprocessSource(t, src)
	if got := strings.Count(out, "LinearWebhookActionRemove"); got != 2 {
		t.Errorf("prefixed names must stay unchanged, found %d occurrences: %s", got, out)
	}
}
