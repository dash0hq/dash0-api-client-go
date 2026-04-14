// Package dash0 provides a high-level client for the Dash0 API.
package dash0

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// ErrAPINotConfigured is returned when a REST API method is called on a client
// created without WithApiUrl.
var ErrAPINotConfigured = errors.New("dash0: API endpoint not configured (use WithApiUrl)")

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

	// Sampling Rules
	ListSamplingRules(ctx context.Context, dataset *string) ([]*SamplingDefinition, error)
	GetSamplingRule(ctx context.Context, originOrID string, dataset *string) (*SamplingDefinition, error)
	CreateSamplingRule(ctx context.Context, rule *SamplingDefinition, dataset *string) (*SamplingDefinition, error)
	UpdateSamplingRule(ctx context.Context, originOrID string, rule *SamplingDefinition, dataset *string) (*SamplingDefinition, error)
	DeleteSamplingRule(ctx context.Context, originOrID string, dataset *string) error
	ListSamplingRulesIter(ctx context.Context, dataset *string) *Iter[SamplingDefinition]

	// Members
	ListMembers(ctx context.Context) ([]*MemberDefinition, error)
	InviteMember(ctx context.Context, request *InviteMemberRequest) error
	DeleteMember(ctx context.Context, memberID string) error
	ListMembersIter(ctx context.Context) *Iter[MemberDefinition]

	// Teams
	ListTeams(ctx context.Context) ([]*TeamsListItem, error)
	CreateTeam(ctx context.Context, team *TeamDefinition) (*TeamDefinition, error)
	GetTeam(ctx context.Context, originOrID string) (*GetTeamResponse, error)
	DeleteTeam(ctx context.Context, originOrID string) error
	UpdateTeamDisplay(ctx context.Context, originOrID string, display *TeamDisplay) error
	AddTeamMembers(ctx context.Context, originOrID string, request *AddTeamMembersRequest) error
	RemoveTeamMember(ctx context.Context, originOrID string, memberID string) error
	ListTeamsIter(ctx context.Context) *Iter[TeamsListItem]

	// Spans
	GetSpans(ctx context.Context, request *GetSpansRequest) (*GetSpansResponse, error)
	GetSpansIter(ctx context.Context, request *GetSpansRequest) *Iter[ResourceSpans]

	// Logs
	GetLogRecords(ctx context.Context, request *GetLogRecordsRequest) (*GetLogRecordsResponse, error)
	GetLogRecordsIter(ctx context.Context, request *GetLogRecordsRequest) *Iter[ResourceLogs]

	// Import
	ImportCheckRule(ctx context.Context, rule *PrometheusAlertRule, dataset *string) (*PrometheusAlertRule, error)
	ImportDashboard(ctx context.Context, dashboard *DashboardDefinition, dataset *string) (*DashboardDefinition, error)
	ImportSyntheticCheck(ctx context.Context, check *SyntheticCheckDefinition, dataset *string) (*SyntheticCheckDefinition, error)
	ImportView(ctx context.Context, view *ViewDefinition, dataset *string) (*ViewDefinition, error)

	// OTLP
	SendLogs(ctx context.Context, logs plog.Logs, dataset *string) error
	SendMetrics(ctx context.Context, metrics pmetric.Metrics, dataset *string) error
	SendTraces(ctx context.Context, traces ptrace.Traces, dataset *string) error

	// Close releases resources associated with the client. Callers should
	// call Close when the client is no longer needed.
	Close(ctx context.Context) error

	// Inner returns the underlying generated client for advanced use cases.
	Inner() *ClientWithResponses
}

// client is the concrete implementation of the Client interface.
type client struct {
	inner      *ClientWithResponses
	config     *clientConfig
	httpClient *http.Client
}

// NewClient creates a new Dash0 API client.
//
// Required options:
//   - WithAuthToken: The auth token for authentication
//   - At least one of WithApiUrl or WithOtlpEndpoint
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
	if cfg.apiUrl == "" && cfg.otlpEndpoint == "" {
		return nil, fmt.Errorf("dash0: at least one of API URL or OTLP endpoint is required (use WithApiUrl and/or WithOtlpEndpoint)")
	}
	if cfg.authToken == "" {
		return nil, fmt.Errorf("dash0: auth token is required (use WithAuthToken)")
	}
	if !strings.HasPrefix(cfg.authToken, "auth_") {
		return nil, fmt.Errorf("dash0: auth token must start with 'auth_'")
	}

	// Validate that WithTransport does not conflict with transport-level options.
	if cfg.transport != nil {
		if err := validateTransportConflicts(cfg); err != nil {
			return nil, err
		}
	}

	// Build HTTP client, either from a pre-built Transport or by assembling
	// the transport stack from individual options.
	var httpClient *http.Client
	if cfg.transport != nil {
		httpClient = cfg.transport.HTTPClient()
		// Allow WithTimeout to override the transport's timeout.
		if cfg.timeoutSet {
			httpClient.Timeout = cfg.timeout
		}
	} else {
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
		rateLimited := newRateLimitedTransport(transport, cfg.maxConcurrent)
		retrying := newRetryTransport(rateLimited, cfg.maxRetries, cfg.retryWaitMin, cfg.retryWaitMax)

		// Build HTTP client
		httpClient = &http.Client{
			Transport: retrying,
			Timeout:   cfg.timeout,
		}

		// Preserve other settings from custom client if provided
		if cfg.httpClient != nil {
			httpClient.CheckRedirect = cfg.httpClient.CheckRedirect
			httpClient.Jar = cfg.httpClient.Jar
		}
	}

	// Create generated REST API client only when API URL is configured
	var inner *ClientWithResponses
	if cfg.apiUrl != "" {
		authEditor := func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+cfg.authToken)
			req.Header.Set("User-Agent", cfg.userAgent)
			return nil
		}
		var err error
		inner, err = NewClientWithResponses(
			cfg.apiUrl,
			withGeneratedHTTPClient(httpClient),
			WithRequestEditorFn(authEditor),
		)
		if err != nil {
			return nil, fmt.Errorf("dash0: failed to create client: %w", err)
		}
	}

	// Validate OTLP config if configured
	if cfg.otlpEndpoint != "" {
		if err := validateOtlpConfig(cfg.otlpEncoding, cfg.otlpEndpoint); err != nil {
			return nil, err
		}
	}

	return &client{
		inner:      inner,
		config:     cfg,
		httpClient: httpClient,
	}, nil
}

// requireAPI returns an error if the REST API client is not configured.
func (c *client) requireAPI() error {
	if c.inner == nil {
		return ErrAPINotConfigured
	}
	return nil
}

// Inner returns the underlying generated client for advanced use cases.
// Use this when you need access to endpoints not wrapped by the high-level client.
func (c *client) Inner() *ClientWithResponses {
	return c.inner
}
