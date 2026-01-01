package dash0

import (
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/sync/semaphore"
)

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
	base       http.RoundTripper
	maxRetries int
	waitMin    time.Duration
	waitMax    time.Duration
}

// newRetryTransport creates a transport that retries failed requests.
func newRetryTransport(base http.RoundTripper, maxRetries int, waitMin, waitMax time.Duration) *retryTransport {
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
		base:       base,
		maxRetries: maxRetries,
		waitMin:    waitMin,
		waitMax:    waitMax,
	}
}

// RoundTrip implements http.RoundTripper with retry logic.
func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Only retry if enabled
	if t.maxRetries == 0 {
		return t.base.RoundTrip(req)
	}

	// Only retry idempotent requests
	if !t.isIdempotent(req) {
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
		if err == nil && !t.shouldRetry(resp) {
			return resp, nil
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
		wait := t.backoff(attempt, resp)

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
// GET, PUT, DELETE are always idempotent. POST requests marked with
// withIdempotent context are also retried.
func (t *retryTransport) isIdempotent(req *http.Request) bool {
	switch req.Method {
	case http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodHead, http.MethodOptions:
		return true
	default:
		// Check if context marks this as idempotent
		return isIdempotent(req.Context())
	}
}

// shouldRetry returns true if the response indicates a retryable error.
func (t *retryTransport) shouldRetry(resp *http.Response) bool {
	if resp == nil {
		return true
	}
	// Retry on 429 (rate limited) and 5xx (server errors)
	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
}

// backoff calculates the wait time before the next retry.
func (t *retryTransport) backoff(attempt int, resp *http.Response) time.Duration {
	// Check Retry-After header
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
				wait := time.Duration(secs) * time.Second
				if wait > t.waitMax {
					wait = t.waitMax
				}
				return wait
			}
		}
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

	return wait
}
