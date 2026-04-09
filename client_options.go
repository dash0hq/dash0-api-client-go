package dash0

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultMaxConcurrentRequests is the default maximum number of concurrent API requests.
	DefaultMaxConcurrentRequests = 3

	// MaxConcurrentRequests is the maximum allowed value for concurrent requests.
	MaxConcurrentRequests = 10

	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 30 * time.Second

	// DefaultUserAgent is the default User-Agent header value.
	DefaultUserAgent = "dash0-api-client-go/" + Version

	// DefaultRetryWaitMin is the default minimum wait time between retries.
	DefaultRetryWaitMin = 500 * time.Millisecond

	// DefaultRetryWaitMax is the default maximum wait time between retries.
	DefaultRetryWaitMax = 30 * time.Second

	// MaxRetries is the maximum allowed number of retries.
	MaxRetries = 5
)

// OtlpEncoding specifies the wire format for OTLP data.
type OtlpEncoding string

const (
	// OtlpEncodingJson is the OTLP/JSON encoding over HTTP.
	OtlpEncodingJson OtlpEncoding = "otlp/json"
)

// ClientOption configures a Dash0 client.
type ClientOption func(*clientConfig)

type clientConfig struct {
	httpClient    *http.Client
	apiUrl        string
	authToken     string
	maxConcurrent int64
	timeout       time.Duration
	userAgent     string
	maxRetries    int
	retryWaitMin  time.Duration
	retryWaitMax  time.Duration
	otlpEncoding  OtlpEncoding
	otlpEndpoint  string
	transport     *Transport

	// Flags that track whether transport-level options were explicitly set.
	// Used to detect conflicts with WithTransport.
	httpClientSet    bool
	maxConcurrentSet bool
	maxRetriesSet    bool
	retryWaitMinSet  bool
	retryWaitMaxSet  bool
	timeoutSet       bool
}

func defaultConfig() *clientConfig {
	return &clientConfig{
		maxConcurrent: DefaultMaxConcurrentRequests,
		timeout:       DefaultTimeout,
		userAgent:     DefaultUserAgent,
		maxRetries:    1, // Retry once by default
		retryWaitMin:  DefaultRetryWaitMin,
		retryWaitMax:  DefaultRetryWaitMax,
	}
}

// WithApiUrl sets the Dash0 API URL.
// This is required for REST API operations (dashboards, check rules, etc.).
// Examples:
//   - https://api.eu-west-1.aws.dash0.com
//   - https://api.eu-central-1.aws.dash0.com
//   - https://api.us-west-2.aws.dash0.com
//   - https://api.europe-west4.gcp.dash0.com
func WithApiUrl(url string) ClientOption {
	return func(c *clientConfig) {
		c.apiUrl = url
	}
}

// WithAuthToken sets the auth token for authentication.
// This is required for all API requests.
func WithAuthToken(authToken string) ClientOption {
	return func(c *clientConfig) {
		c.authToken = authToken
	}
}

// WithHTTPClient sets a custom HTTP client.
// The client's transport will be wrapped with rate limiting middleware.
// Other settings like CheckRedirect and Jar will be preserved.
//
// This option conflicts with [WithTransport].
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *clientConfig) {
		c.httpClient = client
		c.httpClientSet = true
	}
}

// WithMaxConcurrentRequests sets the maximum number of concurrent API calls.
// The value must be between 1 and 10 (inclusive).
// Values outside this range will be clamped.
// Default is 3.
//
// This option conflicts with [WithTransport].
func WithMaxConcurrentRequests(n int64) ClientOption {
	return func(c *clientConfig) {
		c.maxConcurrent = n
		c.maxConcurrentSet = true
	}
}

// WithTimeout sets the HTTP request timeout.
// Default is 30 seconds.
//
// When used with [WithTransport], this overrides the timeout configured on the
// [Transport].
func WithTimeout(d time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.timeout = d
		c.timeoutSet = true
	}
}

// WithUserAgent sets a custom User-Agent header.
// Default is "dash0-api-client-go/" followed by the current library version.
func WithUserAgent(ua string) ClientOption {
	return func(c *clientConfig) {
		c.userAgent = ua
	}
}

// WithMaxRetries sets the maximum number of retries for failed requests.
// Only idempotent requests (GET, PUT, DELETE) and requests marked with
// withIdempotent context are retried.
// Default is 1. Maximum is 5. Set to 0 to disable retries.
//
// This option conflicts with [WithTransport].
//
// Example:
//
//	client, _ := dash0.NewClient(
//	    dash0.WithApiUrl("https://api.eu-west-1.aws.dash0.com"),
//	    dash0.WithAuthToken("your-auth-token"),
//	    dash0.WithMaxRetries(3),
//	)
func WithMaxRetries(n int) ClientOption {
	return func(c *clientConfig) {
		c.maxRetries = n
		c.maxRetriesSet = true
	}
}

// WithRetryWaitMin sets the minimum wait time between retries.
// Default is 500ms. The actual wait time uses exponential backoff
// starting from this value.
//
// This option conflicts with [WithTransport].
func WithRetryWaitMin(d time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.retryWaitMin = d
		c.retryWaitMinSet = true
	}
}

// WithRetryWaitMax sets the maximum wait time between retries.
// Default is 30s. The backoff will not exceed this value.
//
// This option conflicts with [WithTransport].
func WithRetryWaitMax(d time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.retryWaitMax = d
		c.retryWaitMaxSet = true
	}
}

// WithTransport configures the client to use a pre-built [Transport] for rate
// limiting and retry.
// When set, transport-level [ClientOption] functions ([WithMaxRetries],
// [WithRetryWaitMin], [WithRetryWaitMax], [WithMaxConcurrentRequests], and
// [WithHTTPClient]) must not be used.
// [NewClient] returns an error if both [WithTransport] and any of these
// options are provided.
//
// [WithTimeout] remains compatible: when set alongside [WithTransport] it
// overrides the timeout configured on the [Transport].
//
// Example:
//
//	t := dash0.NewTransport(
//	    dash0.WithTransportMaxRetries(3),
//	)
//	client, err := dash0.NewClient(
//	    dash0.WithApiUrl("https://api.eu-west-1.aws.dash0.com"),
//	    dash0.WithAuthToken("auth_yourtoken"),
//	    dash0.WithTransport(t),
//	)
func WithTransport(t *Transport) ClientOption {
	return func(c *clientConfig) {
		c.transport = t
	}
}

// otlpPathSuffixes are OTLP signal-specific path suffixes that should not
// appear at the end of the base endpoint URL.
var otlpPathSuffixes = []string{
	"/v1/traces",
	"/v1/metrics",
	"/v1/logs",
	"/v1/profiles",
	"/v1development/profiles",
}

// WithOtlpEndpoint configures the client to push telemetry data via OTLP/HTTP.
// The encoding parameter specifies the wire format; currently only
// OtlpEncodingJson is supported.
// The url is the base OTLP endpoint (e.g., "https://otlp.example.com:4318");
// signal-specific paths like /v1/traces are appended automatically.
//
// This option is optional. If not set, SendTraces, SendMetrics, and SendLogs
// will return an error.
func WithOtlpEndpoint(encoding OtlpEncoding, url string) ClientOption {
	return func(c *clientConfig) {
		c.otlpEncoding = encoding
		c.otlpEndpoint = url
	}
}

// validateTransportConflicts returns an error if any transport-level
// ClientOption was set alongside WithTransport.
func validateTransportConflicts(cfg *clientConfig) error {
	var conflicts []string
	if cfg.httpClientSet {
		conflicts = append(conflicts, "WithHTTPClient")
	}
	if cfg.maxConcurrentSet {
		conflicts = append(conflicts, "WithMaxConcurrentRequests")
	}
	if cfg.maxRetriesSet {
		conflicts = append(conflicts, "WithMaxRetries")
	}
	if cfg.retryWaitMinSet {
		conflicts = append(conflicts, "WithRetryWaitMin")
	}
	if cfg.retryWaitMaxSet {
		conflicts = append(conflicts, "WithRetryWaitMax")
	}
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf(
		"dash0: WithTransport conflicts with %s; configure these on the Transport instead",
		strings.Join(conflicts, ", "),
	)
}

// validateOtlpConfig validates the OTLP encoding and endpoint together.
// The encoding determines what URL schemes are valid for the endpoint.
func validateOtlpConfig(encoding OtlpEncoding, endpoint string) error {
	switch encoding {
	case OtlpEncodingJson:
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			return fmt.Errorf(
				"dash0: OTLP endpoint for encoding %q must start with http:// or https://, got %q",
				encoding, endpoint)
		}
	default:
		return fmt.Errorf("dash0: unsupported OTLP encoding %q (supported: %s)", encoding, OtlpEncodingJson)
	}

	for _, suffix := range otlpPathSuffixes {
		if strings.HasSuffix(endpoint, suffix) {
			return fmt.Errorf(
				"dash0: OTLP endpoint URL must not end with %q; "+
					"signal-specific paths are appended automatically", suffix)
		}
	}
	return nil
}
