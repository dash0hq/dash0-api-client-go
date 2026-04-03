package dash0_test

import (
	"context"
	"fmt"
	"log"

	dash0 "github.com/dash0hq/dash0-api-client-go"
)

// Client construction

func ExampleNewClient_apiOnly() {
	client, err := dash0.NewClient(
		dash0.WithApiUrl("https://api.eu-west-1.aws.dash0.com"),
		dash0.WithAuthToken("auth_yourtoken"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	dashboards, err := client.ListDashboards(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d dashboards\n", len(dashboards))
}

func ExampleNewClient_apiAndOtlp() {
	client, err := dash0.NewClient(
		dash0.WithApiUrl("https://api.eu-west-1.aws.dash0.com"),
		dash0.WithOtlpEndpoint(dash0.OtlpEncodingJson, "https://ingress.eu-west-1.aws.dash0.com"),
		dash0.WithAuthToken("auth_yourtoken"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close(context.Background()) }()
	_ = client
}

// DatasetPtr

func ExampleDatasetPtr() {
	// Returns nil for "default" — the API treats absent dataset as "default".
	p := dash0.DatasetPtr("default")
	fmt.Println(p) // Output: <nil>
}

func ExampleDatasetPtr_nonDefault() {
	p := dash0.DatasetPtr("production")
	fmt.Println(*p) // Output: production
}

// Import

func ExampleClient_ImportDashboard() {
	client, err := dash0.NewClient(
		dash0.WithApiUrl("https://api.eu-west-1.aws.dash0.com"),
		dash0.WithAuthToken("auth_yourtoken"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	dashboard := &dash0.DashboardDefinition{
		Metadata: dash0.DashboardMetadata{Name: "My Dashboard"},
		Spec: map[string]any{
			"display": map[string]any{"name": "My Dashboard"},
		},
	}
	imported, err := client.ImportDashboard(context.Background(), dashboard, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dash0.GetDashboardID(imported))
}

func ExampleClient_ImportCheckRule() {
	client, err := dash0.NewClient(
		dash0.WithApiUrl("https://api.eu-west-1.aws.dash0.com"),
		dash0.WithAuthToken("auth_yourtoken"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	rule := &dash0.PrometheusAlertRule{
		Name:       "HighErrorRate",
		Expression: "sum(rate(http_errors_total[5m])) > 0.1",
	}
	imported, err := client.ImportCheckRule(context.Background(), rule, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dash0.GetCheckRuleID(imported))
}

// Dashboard helpers

func ExampleGetDashboardName() {
	dashboard := &dash0.DashboardDefinition{
		Spec: map[string]any{
			"display": map[string]any{"name": "API Latency"},
		},
	}
	fmt.Println(dash0.GetDashboardName(dashboard))
	// Output: API Latency
}

func ExampleGetDashboardID() {
	dashboard := &dash0.DashboardDefinition{
		Metadata: dash0.DashboardMetadata{
			Dash0Extensions: &dash0.DashboardMetadataExtensions{
				Id: dash0.Ptr("d-123"),
			},
		},
	}
	fmt.Println(dash0.GetDashboardID(dashboard))
	// Output: d-123
}

func ExampleSetDashboardID() {
	dashboard := &dash0.DashboardDefinition{}
	dash0.SetDashboardID(dashboard, "d-456")
	fmt.Println(*dashboard.Metadata.Dash0Extensions.Id)
	// Output: d-456
}

func ExampleSetDashboardIDIfAbsent() {
	dashboard := &dash0.DashboardDefinition{
		Metadata: dash0.DashboardMetadata{
			Dash0Extensions: &dash0.DashboardMetadataExtensions{
				Id: dash0.Ptr("existing"),
			},
		},
	}
	// Does not overwrite an existing ID.
	dash0.SetDashboardIDIfAbsent(dashboard, "new-id")
	fmt.Println(*dashboard.Metadata.Dash0Extensions.Id)
	// Output: existing
}

func ExampleStripDashboardServerFields() {
	dashboard := &dash0.DashboardDefinition{
		Metadata: dash0.DashboardMetadata{
			Version: dash0.Ptr(int64(3)),
		},
	}
	dash0.StripDashboardServerFields(dashboard)
	fmt.Println(dashboard.Metadata.Version == nil)
	// Output: true
}

func ExampleClearDashboardID() {
	dashboard := &dash0.DashboardDefinition{
		Metadata: dash0.DashboardMetadata{
			Dash0Extensions: &dash0.DashboardMetadataExtensions{
				Id: dash0.Ptr("d-123"),
			},
		},
	}
	dash0.ClearDashboardID(dashboard)
	fmt.Println(dashboard.Metadata.Dash0Extensions.Id == nil)
	// Output: true
}

// Check rule helpers

func ExampleGetCheckRuleName() {
	rule := &dash0.PrometheusAlertRule{Name: "HighErrorRate"}
	fmt.Println(dash0.GetCheckRuleName(rule))
	// Output: HighErrorRate
}

func ExampleGetCheckRuleID() {
	rule := &dash0.PrometheusAlertRule{Id: dash0.Ptr("cr-42")}
	fmt.Println(dash0.GetCheckRuleID(rule))
	// Output: cr-42
}

func ExampleSetCheckRuleID() {
	rule := &dash0.PrometheusAlertRule{Name: "HighErrorRate"}
	dash0.SetCheckRuleID(rule, "cr-42")
	fmt.Println(*rule.Id)
	// Output: cr-42
}

func ExampleSetCheckRuleIDIfAbsent() {
	rule := &dash0.PrometheusAlertRule{Id: dash0.Ptr("existing")}
	dash0.SetCheckRuleIDIfAbsent(rule, "new-id")
	fmt.Println(*rule.Id)
	// Output: existing
}

func ExampleStripCheckRuleServerFields() {
	rule := &dash0.PrometheusAlertRule{
		Name:    "HighErrorRate",
		Dataset: dash0.Ptr("prod"),
	}
	dash0.StripCheckRuleServerFields(rule)
	fmt.Println(rule.Dataset == nil)
	// Output: true
}

func ExampleClearCheckRuleID() {
	rule := &dash0.PrometheusAlertRule{Id: dash0.Ptr("cr-42")}
	dash0.ClearCheckRuleID(rule)
	fmt.Println(rule.Id == nil)
	// Output: true
}

// View helpers

func ExampleGetViewName() {
	view := &dash0.ViewDefinition{
		Spec: dash0.ViewSpec{Display: dash0.ViewDisplay{Name: "Error Logs"}},
	}
	fmt.Println(dash0.GetViewName(view))
	// Output: Error Logs
}

func ExampleGetViewName_fallback() {
	// Falls back to metadata.name when no display name is set.
	view := &dash0.ViewDefinition{
		Metadata: dash0.ViewMetadata{Name: "error-logs"},
	}
	fmt.Println(dash0.GetViewName(view))
	// Output: error-logs
}

func ExampleGetViewID() {
	view := &dash0.ViewDefinition{
		Metadata: dash0.ViewMetadata{
			Labels: &dash0.ViewLabels{Dash0Comid: dash0.Ptr("v-99")},
		},
	}
	fmt.Println(dash0.GetViewID(view))
	// Output: v-99
}

func ExampleSetViewID() {
	view := &dash0.ViewDefinition{}
	dash0.SetViewID(view, "v-99")
	fmt.Println(*view.Metadata.Labels.Dash0Comid)
	// Output: v-99
}

// Synthetic check helpers

func ExampleGetSyntheticCheckName() {
	check := &dash0.SyntheticCheckDefinition{
		Spec: dash0.SyntheticCheckSpec{
			Display: &dash0.SyntheticCheckDisplay{Name: "Homepage Ping"},
		},
	}
	fmt.Println(dash0.GetSyntheticCheckName(check))
	// Output: Homepage Ping
}

func ExampleGetSyntheticCheckID() {
	check := &dash0.SyntheticCheckDefinition{
		Metadata: dash0.SyntheticCheckMetadata{
			Labels: &dash0.SyntheticCheckLabels{Dash0Comid: dash0.Ptr("sc-7")},
		},
	}
	fmt.Println(dash0.GetSyntheticCheckID(check))
	// Output: sc-7
}

func ExampleSetSyntheticCheckID() {
	check := &dash0.SyntheticCheckDefinition{}
	dash0.SetSyntheticCheckID(check, "sc-7")
	fmt.Println(*check.Metadata.Labels.Dash0Comid)
	// Output: sc-7
}
