package profiles

import (
	"context"
	"testing"
	"time"
)

// newBenchProvider builds a provider holding a token with an hour of validity
// left, which is the state nearly every request in a large batch finds.
func newBenchProvider(tb testing.TB) *authTokenProvider {
	tb.Helper()
	store, _ := newTestStore(tb)
	provider := &authTokenProvider{
		cfg: *newResolved(Configuration{
			ApiUrl: "https://api.example.com",
			OAuth: &OAuthState{
				ClientID:     "cid",
				AccessToken:  "dash0_at_current",
				RefreshToken: "rt",
				ExpiresAt:    time.Now().Add(1 * time.Hour),
			},
		}, "test", ""),
		store: store,
	}
	return provider
}

// BenchmarkAuthTokenNoRefreshNeededParallel is the one that matters for the
// batch case: the API client issues up to MaxConcurrentRequests in parallel, so
// the fast path must not serialize them behind an exclusive lock.
func BenchmarkAuthTokenNoRefreshNeededParallel(b *testing.B) {
	provider := newBenchProvider(b)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			if _, err := provider.AuthToken(ctx); err != nil {
				b.Fatal(err)
			}
		}
	})
}
