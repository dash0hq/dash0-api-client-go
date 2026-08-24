package dash0

import (
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/sync/semaphore"
)

const (
	// conflictRetryWaitMin is the starting point for the exponential backoff
	// between 409 conflict retries.
	// It is shorter than [DefaultRetryWaitMin] because a dataset version race
	// resolves almost immediately: the wait is on another writer that already
	// won, not on a rate limiter or an overloaded server.
	conflictRetryWaitMin = 100 * time.Millisecond

	// conflictRetryWaitMax caps the backoff between 409 conflict retries.
	conflictRetryWaitMax = 2 * time.Second
)

// transportConfig holds configuration for building a [Transport].
type transportConfig struct {
	base            http.RoundTripper
	maxConcurrent   int64
	maxRetries      int
	retryWaitMin    time.Duration
	retryWaitMax    time.Duration
	retryOnConflict bool
	timeout         time.Duration
}

func defaultTransportConfig() *transportConfig {
	return &transportConfig{
		maxConcurrent: DefaultMaxConcurrentRequests,
		maxRetries:    1,
		retryWaitMin:  DefaultRetryWaitMin,
		retryWaitMax:  DefaultRetryWaitMax,
		timeout:       DefaultTimeout,
	}
}

// TransportOption configures a [Transport].
type TransportOption func(*transportConfig)

// WithBaseTransport sets the underlying [http.RoundTripper] that the transport
// stack wraps.
// If not set, [http.DefaultTransport] is used.
func WithBaseTransport(rt http.RoundTripper) TransportOption {
	return func(c *transportConfig) {
		c.base = rt
	}
}

// WithTransportMaxRetries sets the maximum number of retries for failed requests.
// Only idempotent requests (GET, PUT, DELETE) and requests marked with
// withIdempotent context are retried.
// Default is 1. Maximum is 5. Set to 0 to disable retries.
func WithTransportMaxRetries(n int) TransportOption {
	return func(c *transportConfig) {
		c.maxRetries = n
	}
}

// WithTransportRetryWaitMin sets the minimum wait time between retries.
// Default is 500ms.
// The actual wait time uses exponential backoff starting from this value.
func WithTransportRetryWaitMin(d time.Duration) TransportOption {
	return func(c *transportConfig) {
		c.retryWaitMin = d
	}
}

// WithTransportRetryWaitMax sets the maximum wait time between retries.
// Default is 30s.
// The backoff will not exceed this value.
func WithTransportRetryWaitMax(d time.Duration) TransportOption {
	return func(c *transportConfig) {
		c.retryWaitMax = d
	}
}

// WithTransportRetryOnConflict enables retrying requests that fail with
// 409 Conflict.
// Retrying a conflict is off unless this option is passed, so the option takes
// no argument.
//
// See [WithRetryOnConflict] for what the 409 means and why replaying the
// request is safe.
func WithTransportRetryOnConflict() TransportOption {
	return func(c *transportConfig) {
		c.retryOnConflict = true
	}
}

// WithTransportMaxConcurrentRequests sets the maximum number of concurrent
// API calls.
// The value must be between 1 and 10 (inclusive).
// Values outside this range will be clamped.
// Default is 3.
func WithTransportMaxConcurrentRequests(n int64) TransportOption {
	return func(c *transportConfig) {
		c.maxConcurrent = n
	}
}

// WithTransportTimeout sets the HTTP request timeout applied to clients
// returned by [Transport.HTTPClient].
// Default is 30 seconds.
func WithTransportTimeout(d time.Duration) TransportOption {
	return func(c *transportConfig) {
		c.timeout = d
	}
}

// Transport is a reusable HTTP transport stack that provides rate limiting
// and retry with exponential backoff.
// Build one with [NewTransport] and share it between raw [http.Client] usage
// and a typed [Client] via [WithTransport].
//
// A Transport is safe for concurrent use.
type Transport struct {
	roundTripper http.RoundTripper
	timeout      time.Duration
}

// NewTransport creates a configured [Transport] with rate limiting and retry.
// Values outside valid ranges are clamped silently.
//
// Example:
//
//	t := dash0.NewTransport(
//	    dash0.WithTransportMaxRetries(3),
//	    dash0.WithTransportTimeout(10 * time.Second),
//	)
//	httpClient := t.HTTPClient()
func NewTransport(opts ...TransportOption) *Transport {
	cfg := defaultTransportConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.maxConcurrent < 1 {
		cfg.maxConcurrent = 1
	}
	if cfg.maxConcurrent > MaxConcurrentRequests {
		cfg.maxConcurrent = MaxConcurrentRequests
	}
	if cfg.maxRetries < 0 {
		cfg.maxRetries = 0
	}
	if cfg.maxRetries > MaxRetries {
		cfg.maxRetries = MaxRetries
	}

	base := cfg.base
	if base == nil {
		base = http.DefaultTransport
	}

	rl := newRateLimitedTransport(base, cfg.maxConcurrent)
	rt := newRetryTransport(rl, cfg.maxRetries, cfg.retryWaitMin, cfg.retryWaitMax, cfg.retryOnConflict)

	return &Transport{
		roundTripper: rt,
		timeout:      cfg.timeout,
	}
}

// HTTPClient returns a new [http.Client] that uses this transport's rate
// limiting and retry stack.
// Each call returns a new [http.Client], but all returned clients share the
// same underlying transport so rate-limit budgets and concurrency limits
// are shared.
func (t *Transport) HTTPClient() *http.Client {
	return &http.Client{
		Transport: t.roundTripper,
		Timeout:   t.timeout,
	}
}

// RoundTripper returns the underlying [http.RoundTripper] with the full
// transport stack (rate limiting and retry) applied.
// Use this to plug the transport into an existing [http.Client] or other
// HTTP infrastructure.
func (t *Transport) RoundTripper() http.RoundTripper {
	return t.roundTripper
}

// rateLimitedTransport wraps an http.RoundTripper and limits concurrent requests
// using a semaphore.
type rateLimitedTransport struct {
	base      http.RoundTripper
	semaphore *semaphore.Weighted
}

// newRateLimitedTransport creates a transport that limits concurrent HTTP calls.
func newRateLimitedTransport(base http.RoundTripper, maxConcurrent int64) *rateLimitedTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &rateLimitedTransport{
		base:      base,
		semaphore: semaphore.NewWeighted(maxConcurrent),
	}
}

// RoundTrip implements http.RoundTripper with concurrency limiting.
func (t *rateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	if err := t.semaphore.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer t.semaphore.Release(1)

	return t.base.RoundTrip(req)
}

// retryTransport wraps an http.RoundTripper and retries failed requests
// with exponential backoff. Only idempotent requests are retried.
type retryTransport struct {
	base            http.RoundTripper
	maxRetries      int
	waitMin         time.Duration
	waitMax         time.Duration
	retryOnConflict bool
}

// newRetryTransport creates a transport that retries failed requests.
func newRetryTransport(base http.RoundTripper, maxRetries int, waitMin, waitMax time.Duration, retryOnConflict bool) *retryTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	if maxRetries < 0 {
		maxRetries = 0
	}
	if maxRetries > 5 {
		maxRetries = 5
	}
	if waitMin <= 0 {
		waitMin = 500 * time.Millisecond
	}
	if waitMax <= 0 {
		waitMax = 30 * time.Second
	}
	return &retryTransport{
		base:            base,
		maxRetries:      maxRetries,
		waitMin:         waitMin,
		waitMax:         waitMax,
		retryOnConflict: retryOnConflict,
	}
}

// RoundTrip implements http.RoundTripper with retry logic.
func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Only retry if enabled
	if t.maxRetries == 0 {
		return t.base.RoundTrip(req)
	}

	// Idempotent requests are replayable for any retryable status. A
	// non-idempotent request enters the loop only to retry a 409, which the
	// server rejected without applying it; see shouldRetry.
	idempotent := isIdempotentRequest(req)
	if !idempotent && !t.retryOnConflict {
		return t.base.RoundTrip(req)
	}

	// A body that cannot be rewound can only be sent once.
	if req.Body != nil && req.GetBody == nil {
		return t.base.RoundTrip(req)
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		// For retries, we need to clone the request body if present
		if attempt > 0 && req.Body != nil && req.GetBody != nil {
			body, bodyErr := req.GetBody()
			if bodyErr != nil {
				return nil, bodyErr
			}
			req.Body = body
		}

		resp, err = t.base.RoundTrip(req)

		// Don't retry if successful or non-retryable
		if err == nil && !t.shouldRetry(resp, idempotent) {
			return resp, nil
		}

		// A transport error leaves it unknown whether the server applied the
		// request, so only an idempotent one may be replayed.
		if err != nil && !idempotent {
			return resp, err
		}

		// Don't retry on last attempt
		if attempt >= t.maxRetries {
			break
		}

		// Close response body before retry to avoid leaking
		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		// Calculate backoff
		wait, ok := t.backoff(attempt, resp)
		if !ok {
			break
		}

		// Wait with context cancellation support
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(wait):
		}
	}

	return resp, err
}

// isIdempotent returns true if the request is safe to retry.
// isIdempotentRequest reports whether req is safe to send more than once.
//
// GET, PUT, DELETE, HEAD, and OPTIONS always are. POST requests marked with
// withIdempotent context are treated as safe too.
//
// Both replay mechanisms consult this: the retry transport for transient
// failures, and authTransport for the single replay after a 401. Keeping the
// rule in one place is what stops the two from disagreeing about which requests
// may be duplicated.
// Read-only POST endpoints opt in via withIdempotent.
func isIdempotentRequest(req *http.Request) bool {
	switch req.Method {
	case http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodHead, http.MethodOptions:
		return true
	default:
		return isIdempotent(req.Context())
	}
}

// shouldRetry returns true if the response indicates a retryable error.
// idempotent reports whether the request itself is safe to send more than once,
// as [isIdempotentRequest] decides.
//
// A 409 is retryable for any request, idempotent or not, when the caller opted
// in via [WithRetryOnConflict]: the Dash0 API returns it when a write loses the
// dataset's optimistic-concurrency check, which means the server rejected that
// write without applying it. Sending it again is therefore not a duplicate
// write. Every other retryable status leaves the outcome unknown, so it is
// retried only for an idempotent request.
func (t *retryTransport) shouldRetry(resp *http.Response, idempotent bool) bool {
	if resp == nil {
		return true
	}
	if t.retryOnConflict && resp.StatusCode == http.StatusConflict {
		return true
	}
	if !idempotent {
		return false
	}
	// Retry on 429 (rate limited) and 5xx (server errors)
	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
}

// backoff calculates the wait time before the next retry. It returns ok == false when retrying should be abandoned, for
// example when a Retry-After header requests a longer wait than t.waitMax allows.
func (t *retryTransport) backoff(attempt int, resp *http.Response) (time.Duration, bool) {
	// Check Retry-After header
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
				wait := time.Duration(secs) * time.Second
				if wait > t.waitMax {
					return 0, false
				}
				return wait, true
			}
		}
	}

	// A conflict gets its own, much shorter window, and full jitter rather
	// than the 25 percent jitter below: racing writers that all back off by
	// nearly the same amount just collide again on the next attempt.
	if resp != nil && resp.StatusCode == http.StatusConflict {
		return conflictBackoff(attempt), true
	}

	// Exponential backoff: waitMin * 2^attempt
	wait := t.waitMin * time.Duration(1<<attempt)
	if wait > t.waitMax {
		wait = t.waitMax
	}

	// Add jitter (0-25% of wait time)
	if wait > 0 {
		jitter := time.Duration(rand.Int63n(int64(wait / 4)))
		wait += jitter
	}

	return wait, true
}

// conflictBackoff returns the wait before the given zero-based conflict retry
// attempt: full jitter, meaning a random duration in [0, ceiling), over an
// exponential ceiling that doubles from conflictRetryWaitMin and saturates at
// conflictRetryWaitMax.
// The ceiling saturates at conflictRetryWaitMax, which also keeps an attempt
// number high enough to overflow the shift or the multiplication from
// producing a nonsensical wait.
func conflictBackoff(attempt int) time.Duration {
	ceiling := conflictRetryWaitMax
	if attempt >= 0 && attempt < 63 {
		if scaled := conflictRetryWaitMin * time.Duration(1<<attempt); scaled > 0 && scaled < ceiling {
			ceiling = scaled
		}
	}
	return time.Duration(rand.Int63n(int64(ceiling)))
}
