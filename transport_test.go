package dash0

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockTransport is a configurable http.RoundTripper for testing.
type mockTransport struct {
	handler func(*http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.handler(req)
}

func TestNewRateLimitedTransport(t *testing.T) {
	t.Run("with base transport", func(t *testing.T) {
		base := &mockTransport{}
		rt := newRateLimitedTransport(base, 5)

		if rt.base != base {
			t.Error("expected base transport to be set")
		}
		if rt.semaphore == nil {
			t.Error("expected semaphore to be initialized")
		}
	})

	t.Run("nil base defaults to DefaultTransport", func(t *testing.T) {
		rt := newRateLimitedTransport(nil, 5)

		if rt.base != http.DefaultTransport {
			t.Error("expected nil base to default to http.DefaultTransport")
		}
	})
}

func TestRateLimitedTransport_RoundTrip(t *testing.T) {
	t.Run("passes request through", func(t *testing.T) {
		expectedResp := &http.Response{StatusCode: http.StatusOK}
		base := &mockTransport{
			handler: func(req *http.Request) (*http.Response, error) {
				return expectedResp, nil
			},
		}

		rt := newRateLimitedTransport(base, 1)
		req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", nil)

		resp, err := rt.RoundTrip(req)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp != expectedResp {
			t.Error("expected response to pass through")
		}
	})

	t.Run("limits concurrent requests", func(t *testing.T) {
		const maxConcurrent = 2
		const totalRequests = 10

		var (
			currentConcurrent atomic.Int32
			maxObserved       atomic.Int32
			wg                sync.WaitGroup
		)

		base := &mockTransport{
			handler: func(req *http.Request) (*http.Response, error) {
				// Track concurrent requests
				current := currentConcurrent.Add(1)

				// Update max observed
				for {
					max := maxObserved.Load()
					if current <= max || maxObserved.CompareAndSwap(max, current) {
						break
					}
				}

				// Simulate some work
				time.Sleep(10 * time.Millisecond)

				currentConcurrent.Add(-1)
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		rt := newRateLimitedTransport(base, maxConcurrent)

		// Launch many concurrent requests
		wg.Add(totalRequests)
		for i := 0; i < totalRequests; i++ {
			go func() {
				defer wg.Done()
				req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", nil)
				_, _ = rt.RoundTrip(req)
			}()
		}

		wg.Wait()

		observed := maxObserved.Load()
		if observed > maxConcurrent {
			t.Errorf("max concurrent requests exceeded: got %d, want <= %d", observed, maxConcurrent)
		}
		if observed < maxConcurrent {
			t.Logf("note: only observed %d concurrent requests (may be timing-dependent)", observed)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		// Create transport with only 1 slot
		blockCh := make(chan struct{})
		base := &mockTransport{
			handler: func(req *http.Request) (*http.Response, error) {
				<-blockCh // Block until signaled
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		rt := newRateLimitedTransport(base, 1)

		// Start a request that will hold the semaphore
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", nil)
			_, _ = rt.RoundTrip(req)
		}()

		// Give first request time to acquire semaphore
		time.Sleep(10 * time.Millisecond)

		// Try a second request with a context that will be cancelled
		ctx, cancel := context.WithCancel(context.Background())
		req, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)

		// Cancel context while waiting for semaphore
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		_, err := rt.RoundTrip(req)

		if err == nil {
			t.Error("expected error when context is cancelled")
		}
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got: %v", err)
		}

		// Cleanup: unblock the first request
		close(blockCh)
		wg.Wait()
	})

	t.Run("releases semaphore after request completes", func(t *testing.T) {
		callCount := 0
		base := &mockTransport{
			handler: func(req *http.Request) (*http.Response, error) {
				callCount++
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		}

		rt := newRateLimitedTransport(base, 1)

		// Make multiple sequential requests - should all succeed if semaphore is released
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", nil)
			_, err := rt.RoundTrip(req)
			if err != nil {
				t.Fatalf("request %d failed: %v", i, err)
			}
		}

		if callCount != 5 {
			t.Errorf("expected 5 calls, got %d", callCount)
		}
	})
}
