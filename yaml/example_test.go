package yaml_test

import (
	"fmt"
	"log"

	dash0 "github.com/dash0hq/dash0-api-client-go"
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
	data := []byte(`kind: Dashboard
metadata:
  name: My Dashboard
  dash0Extensions:
    id: dash-123
    dataset: production
spec:
  display:
    name: My Dashboard
`)
	dashboard, err := dash0yaml.ParseAsDashboard(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dashboard.Metadata.Name)
	fmt.Println(dash0.GetDashboardID(dashboard))
	fmt.Println(string(*dashboard.Metadata.Dash0Extensions.Dataset))
	// Output:
	// My Dashboard
	// dash-123
	// production
}

func ExampleParseAsDashboard_persesDashboard() {
	// PersesDashboard CRDs are automatically detected and converted to
	// the Dash0 DashboardDefinition format.
	// The dash0.com/id and dash0.com/dataset labels are extracted into
	// dash0Extensions.
	data := []byte(`apiVersion: perses.dev/v1alpha1
kind: PersesDashboard
metadata:
  name: my-perses-dashboard
  labels:
    dash0.com/id: perses-123
    dash0.com/dataset: production
  annotations:
    dash0.com/folder-path: /team/sre
spec:
  display:
    name: SRE Overview
  panels: {}
`)
	dashboard, err := dash0yaml.ParseAsDashboard(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dashboard.Metadata.Name)
	fmt.Println(dash0.GetDashboardID(dashboard))
	fmt.Println(string(*dashboard.Metadata.Dash0Extensions.Dataset))
	fmt.Println(*dashboard.Metadata.Annotations.Dash0ComfolderPath)
	// Output:
	// SRE Overview
	// perses-123
	// production
	// /team/sre
}

func ExampleParseAsPrometheusAlertRules_nativeCheckRule() {
	// A native check rule (no "kind" field, detected by the presence of
	// "name" and "expression") returns a single-element slice.
	// The "dataset" field is preserved from the input.
	data := []byte(`
name: HighErrorRate
expression: "sum(rate(errors[5m])) > 0.1"
dataset: production
`)
	rules, err := dash0yaml.ParseAsPrometheusAlertRules(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(rules))
	fmt.Println(rules[0].Name)
	fmt.Println(rules[0].Expression)
	fmt.Println(dash0.StringValue(rules[0].Dataset))
	// Output:
	// 1
	// HighErrorRate
	// sum(rate(errors[5m])) > 0.1
	// production
}

func ExampleParseAsPrometheusAlertRules() {
	// The dash0.com/id and dash0.com/dataset labels from CRD metadata are
	// propagated to every returned rule.
	data := []byte(`apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: my-rules
  labels:
    dash0.com/id: rule-42
    dash0.com/dataset: production
spec:
  groups:
    - name: availability
      rules:
        - alert: HighErrorRate
          expr: "sum(rate(errors[5m])) > 0.1"
          for: 5m
        - alert: ServiceDown
          expr: up == 0
`)
	rules, err := dash0yaml.ParseAsPrometheusAlertRules(data)
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range rules {
		fmt.Printf("%s (id=%s, dataset=%s)\n", r.Name, dash0.StringValue(r.Id), dash0.StringValue(r.Dataset))
	}
	// Output:
	// HighErrorRate (id=rule-42, dataset=production)
	// ServiceDown (id=rule-42, dataset=production)
}

func ExampleUnmarshalPrometheusRule() {
	// The dash0.com/id and dash0.com/dataset labels from CRD metadata are
	// extracted and set on the returned PrometheusAlertRule.
	data := []byte(`apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  labels:
    dash0.com/id: rule-42
    dash0.com/dataset: production
spec:
  groups:
    - name: my-group
      interval: 1m
      rules:
        - alert: HighErrors
          expr: "sum(rate(errors[5m])) > 0.1"
          for: 5m
          annotations:
            summary: High error rate detected
`)
	rule, err := dash0yaml.UnmarshalPrometheusRule(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(rule.Name)
	fmt.Println(rule.Expression)
	fmt.Println(dash0.StringValue(rule.Id))
	fmt.Println(dash0.StringValue(rule.Dataset))
	fmt.Println(*rule.For)
	fmt.Println(*rule.Interval)
	fmt.Println(*rule.Annotations.Summary)
	// Output:
	// my-group - HighErrors
	// sum(rate(errors[5m])) > 0.1
	// rule-42
	// production
	// 5m
	// 1m
	// High error rate detected
}

func ExampleMarshalPrometheusRule() {
	forDur := dash0.Duration("5m")
	summary := "High error rate detected"
	rule := &dash0.PrometheusAlertRule{
		Name:       "my-group - HighErrors",
		Expression: "sum(rate(errors[5m])) > 0.1",
		For:        &forDur,
		Annotations: &dash0.PrometheusAlertRule_Annotations{
			Summary: &summary,
		},
	}

	// Marshal to YAML and unmarshal back to verify the round-trip.
	data, err := dash0yaml.MarshalPrometheusRule(rule)
	if err != nil {
		log.Fatal(err)
	}
	roundTripped, err := dash0yaml.UnmarshalPrometheusRule(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(roundTripped.Name)
	fmt.Println(roundTripped.Expression)
	fmt.Println(*roundTripped.For)
	fmt.Println(*roundTripped.Annotations.Summary)
	// Output:
	// my-group - HighErrors
	// sum(rate(errors[5m])) > 0.1
	// 5m
	// High error rate detected
}

func ExampleWithFlatDocument() {
	// dash0-cli's native CheckRule kind has no "metadata" nesting, and its
	// top-level "annotations" carries genuine user content (summary here)
	// rather than server-managed provenance -- WithFlatDocument() tells
	// Equivalent both facts at once.
	reference := []byte("id: rule-1\nname: test-rule\nannotations:\n  summary: High error rate\n")
	apiResponse := []byte("id: rule-1\nname: test-rule\nannotations:\n  summary: Error rate too high\n")

	equivalent, err := dash0yaml.Equivalent(reference, apiResponse, nil, nil, dash0yaml.WithFlatDocument())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(equivalent)
	// Output: false
}

func ExampleConditionallyIgnoredFields() {
	fields := dash0yaml.ConditionallyIgnoredFields()
	fmt.Println(fields)
	// Output: [metadata.name spec.permissions]
}
