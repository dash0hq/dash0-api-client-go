package dash0_test

import (
	"context"
	"fmt"
	"log"

	dash0 "github.com/dash0hq/dash0-api-client-go"
	"github.com/dash0hq/dash0-api-client-go/profiles"
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

func ExampleNewClient_fromProfile() {
	cfg, err := profiles.ResolveConfiguration("", "")
	if err != nil {
		log.Fatal(err)
	}

	opts := append(cfg.ClientOptions(), dash0.WithUserAgent("my-tool/1.0"))
	client, err := dash0.NewClient(opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	dashboards, err := client.ListDashboards(context.Background(), cfg.DatasetPtr())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d dashboards\n", len(dashboards))
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

func ExampleGetDashboardFolderPath() {
	dashboard := &dash0.DashboardDefinition{
		Metadata: dash0.DashboardMetadata{
			Annotations: &dash0.DashboardAnnotations{
				Dash0ComfolderPath: dash0.Ptr("/team/sre"),
			},
		},
	}
	fmt.Println(dash0.GetDashboardFolderPath(dashboard))
	// Output: /team/sre
}

func ExampleSetDashboardFolderPath() {
	dashboard := &dash0.DashboardDefinition{}
	dash0.SetDashboardFolderPath(dashboard, "/team/sre")
	fmt.Println(*dashboard.Metadata.Annotations.Dash0ComfolderPath)
	// Output: /team/sre
}

func ExampleGetDashboardDataset() {
	dashboard := &dash0.DashboardDefinition{
		Metadata: dash0.DashboardMetadata{
			Dash0Extensions: &dash0.DashboardMetadataExtensions{
				Dataset: dash0.Ptr("production"),
			},
		},
	}
	fmt.Println(dash0.GetDashboardDataset(dashboard))
	// Output: production
}

func ExampleSetDashboardDataset() {
	dashboard := &dash0.DashboardDefinition{}
	dash0.SetDashboardDataset(dashboard, "production")
	fmt.Println(string(*dashboard.Metadata.Dash0Extensions.Dataset))
	// Output: production
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

// Integration helpers

func ExampleGetIntegrationName() {
	def := &dash0.IntegrationDefinition{
		Spec: dash0.IntegrationSpec{
			Display: dash0.IntegrationDisplay{Name: "AWS 123456789012"},
		},
	}
	fmt.Println(dash0.GetIntegrationName(def))
	// Output: AWS 123456789012
}

func ExampleGetIntegrationID() {
	def := &dash0.IntegrationDefinition{
		Metadata: dash0.IntegrationMetadata{
			Labels: map[string]string{"dash0.com/id": "int-123"},
		},
	}
	fmt.Println(dash0.GetIntegrationID(def))
	// Output: int-123
}

func ExampleGetIntegrationOrigin() {
	def := &dash0.IntegrationDefinition{
		Metadata: dash0.IntegrationMetadata{
			Labels: map[string]string{"dash0.com/origin": "terraform:aws:123"},
		},
	}
	fmt.Println(dash0.GetIntegrationOrigin(def))
	// Output: terraform:aws:123
}

func ExampleSetIntegrationID() {
	def := &dash0.IntegrationDefinition{}
	dash0.SetIntegrationID(def, "int-456")
	fmt.Println(def.Metadata.Labels["dash0.com/id"])
	// Output: int-456
}

func ExampleSetIntegrationIDIfAbsent() {
	def := &dash0.IntegrationDefinition{
		Metadata: dash0.IntegrationMetadata{
			Labels: map[string]string{"dash0.com/id": "existing"},
		},
	}
	// Does not overwrite an existing ID.
	dash0.SetIntegrationIDIfAbsent(def, "new-id")
	fmt.Println(def.Metadata.Labels["dash0.com/id"])
	// Output: existing
}

func ExampleStripIntegrationServerFields() {
	def := &dash0.IntegrationDefinition{
		Metadata: dash0.IntegrationMetadata{
			Version: dash0.Ptr(int64(3)),
		},
	}
	dash0.StripIntegrationServerFields(def)
	fmt.Println(def.Metadata.Version == nil)
	// Output: true
}

func ExampleClearIntegrationID() {
	def := &dash0.IntegrationDefinition{
		Metadata: dash0.IntegrationMetadata{
			Labels: map[string]string{"dash0.com/id": "int-123"},
		},
	}
	dash0.ClearIntegrationID(def)
	_, exists := def.Metadata.Labels["dash0.com/id"]
	fmt.Println(exists)
	// Output: false
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

func ExampleGetCheckRuleDataset() {
	rule := &dash0.PrometheusAlertRule{Dataset: dash0.Ptr("production")}
	fmt.Println(dash0.GetCheckRuleDataset(rule))
	// Output: production
}

func ExampleSetCheckRuleDataset() {
	rule := &dash0.PrometheusAlertRule{Name: "HighErrorRate"}
	dash0.SetCheckRuleDataset(rule, "production")
	fmt.Println(dash0.StringValue(rule.Dataset))
	// Output: production
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

func ExampleGetViewDataset() {
	view := &dash0.ViewDefinition{
		Metadata: dash0.ViewMetadata{
			Labels: &dash0.ViewLabels{Dash0Comdataset: dash0.Ptr("production")},
		},
	}
	fmt.Println(dash0.GetViewDataset(view))
	// Output: production
}

func ExampleSetViewDataset() {
	view := &dash0.ViewDefinition{}
	dash0.SetViewDataset(view, "production")
	fmt.Println(*view.Metadata.Labels.Dash0Comdataset)
	// Output: production
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

func ExampleGetSyntheticCheckDataset() {
	check := &dash0.SyntheticCheckDefinition{
		Metadata: dash0.SyntheticCheckMetadata{
			Labels: &dash0.SyntheticCheckLabels{Dash0Comdataset: dash0.Ptr("production")},
		},
	}
	fmt.Println(dash0.GetSyntheticCheckDataset(check))
	// Output: production
}

func ExampleSetSyntheticCheckDataset() {
	check := &dash0.SyntheticCheckDefinition{}
	dash0.SetSyntheticCheckDataset(check, "production")
	fmt.Println(*check.Metadata.Labels.Dash0Comdataset)
	// Output: production
}

func ExampleSetSyntheticCheckID() {
	check := &dash0.SyntheticCheckDefinition{}
	dash0.SetSyntheticCheckID(check, "sc-7")
	fmt.Println(*check.Metadata.Labels.Dash0Comid)
	// Output: sc-7
}

// Prometheus rule helpers

func ExampleGetPrometheusRuleName() {
	rule := &dash0.PrometheusRules{
		Metadata: dash0.PrometheusRulesMetadata{Name: "my-rules"},
	}
	fmt.Println(dash0.GetPrometheusRuleName(rule))
	// Output: my-rules
}

func ExampleGetPrometheusRuleDataset() {
	rule := &dash0.PrometheusRules{
		Metadata: dash0.PrometheusRulesMetadata{
			Labels: map[string]string{"dash0.com/dataset": "production"},
		},
	}
	fmt.Println(dash0.GetPrometheusRuleDataset(rule))
	// Output: production
}

func ExampleSetPrometheusRuleDataset() {
	rule := &dash0.PrometheusRules{}
	dash0.SetPrometheusRuleDataset(rule, "production")
	fmt.Println(rule.Metadata.Labels["dash0.com/dataset"])
	// Output: production
}

// Perses dashboard helpers

func ExampleConvertPersesDashboardToDashboard() {
	perses := &dash0.PersesDashboard{
		APIVersion: "perses.dev/v1alpha1",
		Kind:       "PersesDashboard",
		Metadata:   dash0.PersesDashboardMetadata{Name: "my-perses-dashboard"},
		Spec: map[string]any{
			"display": map[string]any{"name": "Perses Dashboard"},
		},
	}
	dashboard := dash0.ConvertPersesDashboardToDashboard(perses)
	fmt.Println(dashboard.Metadata.Name)
	// Output: Perses Dashboard
}

func ExampleGetPersesDashboardFolderPath() {
	perses := &dash0.PersesDashboard{
		Metadata: dash0.PersesDashboardMetadata{
			Annotations: map[string]string{dash0.AnnotationFolderPath: "/team/sre"},
		},
	}
	fmt.Println(dash0.GetPersesDashboardFolderPath(perses))
	// Output: /team/sre
}

func ExampleSetPersesDashboardFolderPath() {
	perses := &dash0.PersesDashboard{}
	dash0.SetPersesDashboardFolderPath(perses, "/team/sre")
	fmt.Println(perses.Metadata.Annotations[dash0.AnnotationFolderPath])
	// Output: /team/sre
}

func ExampleGetPersesDashboardDataset() {
	perses := &dash0.PersesDashboard{
		Metadata: dash0.PersesDashboardMetadata{
			Labels: map[string]string{"dash0.com/dataset": "production"},
		},
	}
	fmt.Println(dash0.GetPersesDashboardDataset(perses))
	// Output: production
}

func ExampleSetPersesDashboardDataset() {
	perses := &dash0.PersesDashboard{}
	dash0.SetPersesDashboardDataset(perses, "production")
	fmt.Println(perses.Metadata.Labels["dash0.com/dataset"])
	// Output: production
}

// FormatDuration

func ExampleFormatDuration() {
	fmt.Println(dash0.FormatDuration(5 * 60e9))  // 5 minutes
	fmt.Println(dash0.FormatDuration(90e9))       // 1m30s
	fmt.Println(dash0.FormatDuration(2 * 3600e9)) // 2 hours
	// Output:
	// 5m
	// 1m30s
	// 2h
}
