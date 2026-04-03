package yaml_test

import (
	"fmt"
	"log"

	dash0yaml "github.com/dash0hq/dash0-api-client-go/yaml"
)

func ExampleDetectKind() {
	kind, err := dash0yaml.DetectKind([]byte("kind: PrometheusRule\nspec: {}"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(kind)
	// Output: PrometheusRule
}

func ExampleDetectKind_inferredCheckRule() {
	// Documents without an explicit "kind" but with "name" and "expression"
	// are inferred as check rules.
	kind, err := dash0yaml.DetectKind([]byte("name: HighErrors\nexpression: up == 0"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(kind)
	// Output: CheckRule
}

func ExampleParseAsDashboard() {
	data := []byte(`{
		"kind": "Dashboard",
		"metadata": {"name": "My Dashboard"},
		"spec": {"display": {"name": "My Dashboard"}}
	}`)
	dashboard, err := dash0yaml.ParseAsDashboard(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dashboard.Metadata.Name)
	// Output: My Dashboard
}

func ExampleParseAsPrometheusAlertRules() {
	data := []byte(`
name: HighErrorRate
expression: "sum(rate(errors[5m])) > 0.1"
`)
	rules, err := dash0yaml.ParseAsPrometheusAlertRules(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(rules), rules[0].Name)
	// Output: 1 HighErrorRate
}

func ExampleMarshalPrometheusRule() {
	rule := &dash0yaml.PrometheusRules{
		Metadata: dash0yaml.PrometheusRulesMetadata{Name: "my-rules"},
	}
	fmt.Println(dash0yaml.GetPrometheusRuleName(rule))
	// Output: my-rules
}

func ExampleConvertPersesDashboardToDashboard() {
	perses := &dash0yaml.PersesDashboard{
		APIVersion: "perses.dev/v1alpha1",
		Kind:       "PersesDashboard",
		Metadata:   dash0yaml.PersesDashboardMetadata{Name: "my-perses-dashboard"},
		Spec: map[string]interface{}{
			"display": map[string]interface{}{"name": "Perses Dashboard"},
		},
	}
	dashboard := dash0yaml.ConvertPersesDashboardToDashboard(perses)
	fmt.Println(dashboard.Metadata.Name)
	// Output: Perses Dashboard
}

func ExampleFormatDuration() {
	fmt.Println(dash0yaml.FormatDuration(5 * 60e9))  // 5 minutes
	fmt.Println(dash0yaml.FormatDuration(90e9))       // 1m30s
	fmt.Println(dash0yaml.FormatDuration(2 * 3600e9)) // 2 hours
	// Output:
	// 5m
	// 1m30s
	// 2h
}
