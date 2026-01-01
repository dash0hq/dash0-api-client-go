package dash0

import (
	"net/http"
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
	DefaultUserAgent = "dash0-api-client-go/1.0.0"

	// DefaultRetryWaitMin is the default minimum wait time between retries.
	DefaultRetryWaitMin = 500 * time.Millisecond

	// DefaultRetryWaitMax is the default maximum wait time between retries.
	DefaultRetryWaitMax = 30 * time.Second

	// MaxRetries is the maximum allowed number of retries.
	MaxRetries = 5
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
// This is required and must be a valid Dash0 API endpoint URL.
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
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *clientConfig) {
		c.httpClient = client
	}
}

// WithMaxConcurrentRequests sets the maximum number of concurrent API calls.
// The value must be between 1 and 10 (inclusive).
// Values outside this range will be clamped.
// Default is 3.
func WithMaxConcurrentRequests(n int64) ClientOption {
	return func(c *clientConfig) {
		c.maxConcurrent = n
	}
}

// WithTimeout sets the HTTP request timeout.
// Default is 30 seconds.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.timeout = d
	}
}

// WithUserAgent sets a custom User-Agent header.
// Default is "dash0-api-client-go/1.0.0".
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
	}
}

// WithRetryWaitMin sets the minimum wait time between retries.
// Default is 500ms. The actual wait time uses exponential backoff
// starting from this value.
func WithRetryWaitMin(d time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.retryWaitMin = d
	}
}

// WithRetryWaitMax sets the maximum wait time between retries.
// Default is 30s. The backoff will not exceed this value.
func WithRetryWaitMax(d time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.retryWaitMax = d
	}
}
