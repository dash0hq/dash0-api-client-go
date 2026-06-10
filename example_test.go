package dash0_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	dash0 "github.com/dash0hq/dash0-api-client-go"
	"github.com/dash0hq/dash0-api-client-go/profiles"
)

// Transport

func ExampleNewTransport() {
	t := dash0.NewTransport(
		dash0.WithTransportMaxRetries(3),
		dash0.WithTransportTimeout(10*time.Second),
	)
	fmt.Println(t.HTTPClient() != nil)
	// Output: true
}

func ExampleTransport_HTTPClient() {
	t := dash0.NewTransport()
	httpClient := t.HTTPClient()
	fmt.Println(httpClient != nil)
	// Output: true
}

func ExampleWithTransport() {
	t := dash0.NewTransport(
		dash0.WithTransportMaxRetries(3),
		dash0.WithTransportMaxConcurrentRequests(5),
	)

	// Use the same transport for both a raw HTTP client and a typed client.
	httpClient := t.HTTPClient()
	_ = httpClient

	client, err := dash0.NewClient(
		dash0.WithApiUrl("https://api.eu-west-1.aws.dash0.com"),
		dash0.WithAuthToken("auth_yourtoken"),
		dash0.WithTransport(t),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close(context.Background()) }()
	fmt.Println(client != nil)
	// Output: true
}

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

// Recording rule helpers

func ExampleGetRecordingRuleName() {
	rule := &dash0.RecordingRule{
		Metadata: dash0.PrometheusRuleMetadata{Name: "cpu-usage-rules"},
	}
	fmt.Println(dash0.GetRecordingRuleName(rule))
	// Output: cpu-usage-rules
}

func ExampleGetRecordingRuleID() {
	labels := map[string]string{"dash0.com/id": "rr-42"}
	rule := &dash0.RecordingRule{
		Metadata: dash0.PrometheusRuleMetadata{Labels: &labels},
	}
	fmt.Println(dash0.GetRecordingRuleID(rule))
	// Output: rr-42
}

func ExampleGetRecordingRuleDataset() {
	labels := map[string]string{"dash0.com/dataset": "production"}
	rule := &dash0.RecordingRule{
		Metadata: dash0.PrometheusRuleMetadata{Labels: &labels},
	}
	fmt.Println(dash0.GetRecordingRuleDataset(rule))
	// Output: production
}

func ExampleSetRecordingRuleDataset() {
	rule := &dash0.RecordingRule{
		Metadata: dash0.PrometheusRuleMetadata{Name: "cpu-usage-rules"},
	}
	dash0.SetRecordingRuleDataset(rule, "production")
	fmt.Println((*rule.Metadata.Labels)["dash0.com/dataset"])
	// Output: production
}

func ExampleSetRecordingRuleID() {
	rule := &dash0.RecordingRule{
		Metadata: dash0.PrometheusRuleMetadata{Name: "cpu-usage-rules"},
	}
	dash0.SetRecordingRuleID(rule, "rr-42")
	fmt.Println((*rule.Metadata.Labels)["dash0.com/id"])
	// Output: rr-42
}

func ExampleSetRecordingRuleIDIfAbsent() {
	labels := map[string]string{"dash0.com/id": "existing"}
	rule := &dash0.RecordingRule{
		Metadata: dash0.PrometheusRuleMetadata{Labels: &labels},
	}
	dash0.SetRecordingRuleIDIfAbsent(rule, "new-id")
	fmt.Println((*rule.Metadata.Labels)["dash0.com/id"])
	// Output: existing
}

func ExampleStripRecordingRuleServerFields() {
	labels := map[string]string{
		"dash0.com/id":     "rr-42",
		"dash0.com/origin": "terraform",
	}
	rule := &dash0.RecordingRule{
		Metadata: dash0.PrometheusRuleMetadata{Labels: &labels},
	}
	dash0.StripRecordingRuleServerFields(rule)
	_, hasID := (*rule.Metadata.Labels)["dash0.com/id"]
	fmt.Println(hasID)
	// Output: false
}

func ExampleClearRecordingRuleID() {
	labels := map[string]string{"dash0.com/id": "rr-42"}
	rule := &dash0.RecordingRule{
		Metadata: dash0.PrometheusRuleMetadata{Labels: &labels},
	}
	dash0.ClearRecordingRuleID(rule)
	fmt.Println(dash0.GetRecordingRuleID(rule) == "")
	// Output: true
}

// OAuthClient

func ExampleNewOAuthClient() {
	client, err := dash0.NewOAuthClient(
		dash0.WithApiUrl("https://api.eu-west-1.aws.dash0.com"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close(context.Background()) }()
	fmt.Println(client != nil)
	// Output: true
}

func ExampleOAuthClient_AuthorizeURL() {
	client, err := dash0.NewOAuthClient(
		dash0.WithApiUrl("https://api.eu-west-1.aws.dash0.com"),
	)
	if err != nil {
		log.Fatal(err)
	}

	authorizeURL, err := client.AuthorizeURL(&dash0.AuthorizeURLParams{
		ResponseType:        "code",
		ClientID:            "my-client",
		RedirectURI:         "http://localhost:8080/callback",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(authorizeURL != "")
	// Output: true
}

func ExampleOAuthClient_RegisterClient() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"client_id":                 "generated-id",
			"client_name":               "My CLI",
			"registration_access_token": "reg_token",
		})
	}))
	defer server.Close()

	client, err := dash0.NewOAuthClient(dash0.WithApiUrl(server.URL))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	resp, err := client.RegisterClient(context.Background(), &dash0.OAuthClientRegistrationRequest{
		ClientName:   "My CLI",
		RedirectUris: []string{"http://localhost:8080/callback"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.ClientId)
	// Output: generated-id
}

func ExampleOAuthClient_ExchangeToken() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "dash0_at_example",
			"token_type":   "Bearer",
			"expires_in":   900,
		})
	}))
	defer server.Close()

	client, err := dash0.NewOAuthClient(dash0.WithApiUrl(server.URL))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	resp, err := client.ExchangeToken(context.Background(), &dash0.OAuthTokenRequest{
		GrantType:    dash0.OAuthGrantTypeAuthorizationCode,
		Code:         dash0.Ptr("auth-code"),
		RedirectUri:  dash0.Ptr("http://localhost:8080/callback"),
		CodeVerifier: dash0.Ptr("verifier"),
		ClientId:     dash0.Ptr("my-client"),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.TokenType)
	// Output: Bearer
}

func ExampleOAuthClient_GetAuthorizationServerMetadata() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                   "https://auth.example.com",
			"authorization_endpoint":   "https://auth.example.com/oauth/authorize",
			"token_endpoint":           "https://auth.example.com/oauth/token",
			"response_types_supported": []string{"code"},
		})
	}))
	defer server.Close()

	client, err := dash0.NewOAuthClient(dash0.WithApiUrl(server.URL))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	metadata, err := client.GetAuthorizationServerMetadata(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(metadata.Issuer)
	// Output: https://auth.example.com
}

func ExampleOAuthClient_GetProtectedResourceMetadata() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"resource":              "https://api.example.com",
			"authorization_servers": []string{"https://auth.example.com"},
		})
	}))
	defer server.Close()

	client, err := dash0.NewOAuthClient(dash0.WithApiUrl(server.URL))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	metadata, err := client.GetProtectedResourceMetadata(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(metadata.Resource)
	// Output: https://api.example.com
}

func ExampleOAuthClient_RevokeToken() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := dash0.NewOAuthClient(dash0.WithApiUrl(server.URL))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	hint := dash0.OAuthTokenTypeRefreshToken
	err = client.RevokeToken(context.Background(), &dash0.OAuthRevocationRequest{
		Token:         "dash0_rt_to-revoke",
		TokenTypeHint: &hint,
	})
	fmt.Println(err == nil)
	// Output: true
}

func ExampleIsOAuthTokenError() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "authorization code has expired",
		})
	}))
	defer server.Close()

	client, err := dash0.NewOAuthClient(dash0.WithApiUrl(server.URL))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	_, err = client.ExchangeToken(context.Background(), &dash0.OAuthTokenRequest{
		GrantType: dash0.OAuthGrantTypeAuthorizationCode,
		Code:      dash0.Ptr("expired"),
		ClientId:  dash0.Ptr("my-client"),
	})
	fmt.Println(dash0.IsOAuthTokenError(err))
	// Output: true
}

// FormatDuration

func ExampleFormatDuration() {
	fmt.Println(dash0.FormatDuration(5 * 60e9))   // 5 minutes
	fmt.Println(dash0.FormatDuration(90e9))       // 1m30s
	fmt.Println(dash0.FormatDuration(2 * 3600e9)) // 2 hours
	// Output:
	// 5m
	// 1m30s
	// 2h
}

// Spam filter helpers

func ExampleGetSpamFilterName() {
	filter := &dash0.SpamFilter{
		Metadata: dash0.SpamFilterMetadata{Name: "Drop noisy health checks"},
	}
	fmt.Println(dash0.GetSpamFilterName(filter))
	// Output: Drop noisy health checks
}

func ExampleGetSpamFilterID() {
	filter := &dash0.SpamFilter{
		Metadata: dash0.SpamFilterMetadata{
			Labels: &dash0.SpamFilterLabels{Dash0Comid: dash0.Ptr("sf-123")},
		},
	}
	fmt.Println(dash0.GetSpamFilterID(filter))
	// Output: sf-123
}

func ExampleGetSpamFilterDataset() {
	filter := &dash0.SpamFilter{
		Metadata: dash0.SpamFilterMetadata{
			Labels: &dash0.SpamFilterLabels{Dash0Comdataset: dash0.Ptr("production")},
		},
	}
	fmt.Println(dash0.GetSpamFilterDataset(filter))
	// Output: production
}

func ExampleSetSpamFilterDataset() {
	filter := &dash0.SpamFilter{}
	dash0.SetSpamFilterDataset(filter, "production")
	fmt.Println(*filter.Metadata.Labels.Dash0Comdataset)
	// Output: production
}

func ExampleSetSpamFilterID() {
	filter := &dash0.SpamFilter{}
	dash0.SetSpamFilterID(filter, "sf-456")
	fmt.Println(*filter.Metadata.Labels.Dash0Comid)
	// Output: sf-456
}

func ExampleSetSpamFilterIDIfAbsent() {
	filter := &dash0.SpamFilter{
		Metadata: dash0.SpamFilterMetadata{
			Labels: &dash0.SpamFilterLabels{Dash0Comid: dash0.Ptr("existing")},
		},
	}
	// Does not overwrite an existing ID.
	dash0.SetSpamFilterIDIfAbsent(filter, "new-id")
	fmt.Println(*filter.Metadata.Labels.Dash0Comid)
	// Output: existing
}

func ExampleStripSpamFilterServerFields() {
	filter := &dash0.SpamFilter{
		Metadata: dash0.SpamFilterMetadata{Name: "Drop noisy health checks"},
	}
	dash0.SetSpamFilterID(filter, "sf-123")
	dash0.StripSpamFilterServerFields(filter)
	fmt.Println(filter.Metadata.Labels.Dash0Comid == nil)
	// Output: true
}

func ExampleClearSpamFilterID() {
	filter := &dash0.SpamFilter{
		Metadata: dash0.SpamFilterMetadata{
			Labels: &dash0.SpamFilterLabels{Dash0Comid: dash0.Ptr("sf-123")},
		},
	}
	dash0.ClearSpamFilterID(filter)
	fmt.Println(filter.Metadata.Labels.Dash0Comid == nil)
	// Output: true
}

// Notification channel helpers

func ExampleGetNotificationChannelName() {
	channel := &dash0.NotificationChannelDefinition{
		Metadata: dash0.NotificationChannelMetadata{Name: "Slack Alerts"},
	}
	fmt.Println(dash0.GetNotificationChannelName(channel))
	// Output: Slack Alerts
}

func ExampleGetNotificationChannelOrigin() {
	channel := &dash0.NotificationChannelDefinition{
		Metadata: dash0.NotificationChannelMetadata{
			Labels: &dash0.NotificationChannelLabels{Dash0Comorigin: dash0.Ptr("terraform")},
		},
	}
	fmt.Println(dash0.GetNotificationChannelOrigin(channel))
	// Output: terraform
}

func ExampleSetNotificationChannelOrigin() {
	channel := &dash0.NotificationChannelDefinition{
		Metadata: dash0.NotificationChannelMetadata{Name: "Slack Alerts"},
	}
	dash0.SetNotificationChannelOrigin(channel, "terraform")
	fmt.Println(*channel.Metadata.Labels.Dash0Comorigin)
	// Output: terraform
}

func ExampleStripNotificationChannelServerFields() {
	channel := &dash0.NotificationChannelDefinition{
		Metadata: dash0.NotificationChannelMetadata{Name: "Slack Alerts"},
	}
	dash0.SetNotificationChannelID(channel, "nc-123")
	dash0.StripNotificationChannelServerFields(channel)
	fmt.Println(channel.Metadata.Labels.Dash0Comid == nil)
	// Output: true
}

func ExampleClearNotificationChannelID() {
	channel := &dash0.NotificationChannelDefinition{
		Metadata: dash0.NotificationChannelMetadata{Name: "Slack Alerts"},
	}
	dash0.SetNotificationChannelID(channel, "nc-123")
	dash0.ClearNotificationChannelID(channel)
	fmt.Println(channel.Metadata.Labels.Dash0Comid == nil)
	// Output: true
}

// Deep links

func ExampleAppBaseURL() {
	fmt.Println(dash0.AppBaseURL("https://api.eu-west-1.aws.dash0.com"))
	// Output: https://app.dash0.com
}

func ExampleDeeplinkURL() {
	fmt.Println(dash0.DeeplinkURL("https://api.us-west-2.aws.dash0.com", dash0.DeeplinkAssetTypeDashboard, "abc-123", dash0.Ptr("production")))
	// Output: https://app.dash0.com/goto/dashboards?dashboard_id=abc-123&dataset=production
}

func ExampleViewDeeplinkURL() {
	fmt.Println(dash0.ViewDeeplinkURL("https://api.us-west-2.aws.dash0.com", dash0.Spans, "view-7", dash0.Ptr("production")))
	// Output: https://app.dash0.com/goto/traces/explorer?dataset=production&view_id=view-7
}
