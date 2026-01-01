// Package dash0 provides a high-level client for the Dash0 API.
package dash0

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Client defines the Dash0 API client interface.
// Use NewClient to create a concrete implementation.
type Client interface {
	// Dashboards
	ListDashboards(ctx context.Context, dataset *string) ([]*DashboardApiListItem, error)
	GetDashboard(ctx context.Context, originOrID string, dataset *string) (*DashboardDefinition, error)
	CreateDashboard(ctx context.Context, dashboard *DashboardDefinition, dataset *string) (*DashboardDefinition, error)
	UpdateDashboard(ctx context.Context, originOrID string, dashboard *DashboardDefinition, dataset *string) (*DashboardDefinition, error)
	DeleteDashboard(ctx context.Context, originOrID string, dataset *string) error
	ListDashboardsIter(ctx context.Context, dataset *string) *Iter[DashboardApiListItem]

	// Check Rules
	ListCheckRules(ctx context.Context, dataset *string) ([]*PrometheusAlertRuleApiListItem, error)
	GetCheckRule(ctx context.Context, originOrID string, dataset *string) (*PrometheusAlertRule, error)
	CreateCheckRule(ctx context.Context, rule *PrometheusAlertRule, dataset *string) (*PrometheusAlertRule, error)
	UpdateCheckRule(ctx context.Context, originOrID string, rule *PrometheusAlertRule, dataset *string) (*PrometheusAlertRule, error)
	DeleteCheckRule(ctx context.Context, originOrID string, dataset *string) error
	ListCheckRulesIter(ctx context.Context, dataset *string) *Iter[PrometheusAlertRuleApiListItem]

	// Synthetic Checks
	ListSyntheticChecks(ctx context.Context, dataset *string) ([]*SyntheticChecksApiListItem, error)
	GetSyntheticCheck(ctx context.Context, originOrID string, dataset *string) (*SyntheticCheckDefinition, error)
	CreateSyntheticCheck(ctx context.Context, check *SyntheticCheckDefinition, dataset *string) (*SyntheticCheckDefinition, error)
	UpdateSyntheticCheck(ctx context.Context, originOrID string, check *SyntheticCheckDefinition, dataset *string) (*SyntheticCheckDefinition, error)
	DeleteSyntheticCheck(ctx context.Context, originOrID string, dataset *string) error
	ListSyntheticChecksIter(ctx context.Context, dataset *string) *Iter[SyntheticChecksApiListItem]

	// Views
	ListViews(ctx context.Context, dataset *string) ([]*ViewApiListItem, error)
	GetView(ctx context.Context, originOrID string, dataset *string) (*ViewDefinition, error)
	CreateView(ctx context.Context, view *ViewDefinition, dataset *string) (*ViewDefinition, error)
	UpdateView(ctx context.Context, originOrID string, view *ViewDefinition, dataset *string) (*ViewDefinition, error)
	DeleteView(ctx context.Context, originOrID string, dataset *string) error
	ListViewsIter(ctx context.Context, dataset *string) *Iter[ViewApiListItem]

	// Spans
	GetSpans(ctx context.Context, request *GetSpansRequest) (*GetSpansResponse, error)
	GetSpansIter(ctx context.Context, request *GetSpansRequest) *Iter[ResourceSpans]

	// Logs
	GetLogRecords(ctx context.Context, request *GetLogRecordsRequest) (*GetLogRecordsResponse, error)
	GetLogRecordsIter(ctx context.Context, request *GetLogRecordsRequest) *Iter[ResourceLogs]

	// Import
	ImportCheckRule(ctx context.Context, rule *PostApiImportCheckRuleJSONRequestBody, dataset *string) (*PrometheusAlertRule, error)
	ImportDashboard(ctx context.Context, dashboard *PostApiImportDashboardJSONRequestBody, dataset *string) (*DashboardDefinition, error)
	ImportSyntheticCheck(ctx context.Context, check *PostApiImportSyntheticCheckJSONRequestBody, dataset *string) (*SyntheticCheckDefinition, error)
	ImportView(ctx context.Context, view *PostApiImportViewJSONRequestBody, dataset *string) (*ViewDefinition, error)

	// Inner returns the underlying generated client for advanced use cases.
	Inner() *ClientWithResponses
}

// client is the concrete implementation of the Client interface.
type client struct {
	inner  *ClientWithResponses
	config *clientConfig
}

// NewClient creates a new Dash0 API client.
//
// Required options:
//   - WithApiUrl: The Dash0 API endpoint URL
//   - WithAuthToken: The auth token for authentication
//
// Example:
//
//	client, err := dash0.NewClient(
//	    dash0.WithApiUrl("https://api.eu-west-1.aws.dash0.com"),
//	    dash0.WithAuthToken("your-auth-token"),
//	)
func NewClient(opts ...ClientOption) (Client, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate required configuration
	if cfg.apiUrl == "" {
		return nil, fmt.Errorf("dash0: API URL is required (use WithApiUrl)")
	}
	if cfg.authToken == "" {
		return nil, fmt.Errorf("dash0: auth token is required (use WithAuthToken)")
	}
	if !strings.HasPrefix(cfg.authToken, "auth_") {
		return nil, fmt.Errorf("dash0: auth token must start with 'auth_'")
	}

	// Clamp maxConcurrent to valid range
	if cfg.maxConcurrent < 1 {
		cfg.maxConcurrent = 1
	}
	if cfg.maxConcurrent > MaxConcurrentRequests {
		cfg.maxConcurrent = MaxConcurrentRequests
	}

	// Clamp maxRetries to valid range
	if cfg.maxRetries < 0 {
		cfg.maxRetries = 0
	}
	if cfg.maxRetries > MaxRetries {
		cfg.maxRetries = MaxRetries
	}

	// Get base transport from custom client or use default
	var transport http.RoundTripper
	if cfg.httpClient != nil {
		transport = cfg.httpClient.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}
	} else {
		transport = http.DefaultTransport
	}

	// Stack transports: base -> rate limit -> retry
	// Rate limiting is applied first, then retry wraps it
	rateLimitedTransport := newRateLimitedTransport(transport, cfg.maxConcurrent)
	retryingTransport := newRetryTransport(rateLimitedTransport, cfg.maxRetries, cfg.retryWaitMin, cfg.retryWaitMax)

	// Build HTTP client
	httpClient := &http.Client{
		Transport: retryingTransport,
		Timeout:   cfg.timeout,
	}

	// Preserve other settings from custom client if provided
	if cfg.httpClient != nil {
		httpClient.CheckRedirect = cfg.httpClient.CheckRedirect
		httpClient.Jar = cfg.httpClient.Jar
	}

	// Create auth token request editor
	authEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+cfg.authToken)
		req.Header.Set("User-Agent", cfg.userAgent)
		return nil
	}

	// Create generated client with responses
	inner, err := NewClientWithResponses(
		cfg.apiUrl,
		withGeneratedHTTPClient(httpClient),
		WithRequestEditorFn(authEditor),
	)
	if err != nil {
		return nil, fmt.Errorf("dash0: failed to create client: %w", err)
	}

	return &client{
		inner:  inner,
		config: cfg,
	}, nil
}

// Inner returns the underlying generated client for advanced use cases.
// Use this when you need access to endpoints not wrapped by the high-level client.
func (c *client) Inner() *ClientWithResponses {
	return c.inner
}
