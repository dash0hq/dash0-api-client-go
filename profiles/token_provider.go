package profiles

import (
	"context"
	"fmt"
	"sync"
	"time"

	dash0 "github.com/dash0hq/dash0-api-client-go"
)

// AuthTokenProvider returns a [dash0.AuthTokenProvider] that keeps this
// configuration's OAuth access token fresh.
//
// Pass the result to [dash0.WithAuthTokenProvider] instead of handing
// [dash0.WithAuthToken] a token resolved once at startup.
// A Dash0 OAuth access token is valid for 15 minutes, so a client built around
// a single snapshot of it starts failing with 401 partway through any operation
// that runs longer than the token's remaining validity — a large telemetry
// export, a bulk apply, or a Terraform run over many resources.
// This provider is consulted before every request and refreshes the token once
// it comes within [OAuthRefreshThreshold] of expiry, which keeps such an
// operation running on a single interactive login.
//
// For a configuration with no OAuth state the returned provider serves
// cfg.AuthToken unchanged, so callers do not need to branch on the profile's
// auth kind.
//
// The returned provider is safe for concurrent use and holds its own copy of
// cfg, so refreshes are visible to every request the client issues but do not
// mutate the caller's [Configuration].
func (cfg *Configuration) AuthTokenProvider(opts ...StoreOption) dash0.AuthTokenProvider {
	if cfg.OAuth == nil {
		return dash0.StaticAuthTokenProvider(cfg.AuthToken)
	}

	provider := &authTokenProvider{cfg: *cfg}
	// Copy the OAuth state as well. The shallow struct copy above shares the
	// pointer, and refreshOAuthToken writes through it.
	oauthState := *cfg.OAuth
	provider.cfg.OAuth = &oauthState

	// Default to the directory this configuration was resolved from, so a caller
	// that built its Store with WithConfigDir does not silently refresh against
	// a different profiles.json. Caller-supplied options come after, and win.
	if cfg.ConfigDir != "" {
		opts = append([]StoreOption{WithConfigDir(cfg.ConfigDir)}, opts...)
	}

	store, err := NewStore(opts...)
	if err != nil {
		// Surface this when a token is actually requested rather than returning
		// an error here: a nil provider would be far easier for callers to
		// mishandle than one that reports the reason it cannot serve a token.
		provider.storeErr = fmt.Errorf("failed to open the profile store for OAuth token refresh: %w", err)
		return provider
	}
	provider.store = store
	return provider
}

// Satisfying the interface is checked here rather than left to a runtime type
// assertion. Interface methods are matched by exact name, so renaming
// ForceRefreshAuthToken would otherwise still compile and only surface as a
// failed assertion once the API client hit a 401.
var _ dash0.RefreshingAuthTokenProvider = (*authTokenProvider)(nil)

// authTokenProvider serves a profile's OAuth access token, refreshing it as it
// approaches expiry.
//
// It implements [dash0.RefreshingAuthTokenProvider], so the API client can also
// ask it for a replacement token after the authorization server rejects one the
// provider still considered valid.
type authTokenProvider struct {
	// mu guards cfg. refreshOAuthToken writes cfg.AuthToken and the OAuth
	// state, and the API client calls AuthToken from up to
	// [dash0.MaxConcurrentRequests] goroutines at once. The cross-process file
	// lock inside refreshOAuthToken serializes the refresh itself; mu is what
	// makes the in-memory struct safe.
	//
	// It is an RWMutex because the overwhelmingly common case is a request that
	// finds a token with plenty of validity left and only needs to read it.
	// Holding an exclusive lock for those would serialize every request in a
	// large batch behind a lock none of them needs to write to.
	mu  sync.RWMutex
	cfg Configuration

	store    *Store
	storeErr error
}

// AuthToken implements [dash0.AuthTokenProvider].
// It refreshes the access token when it is within [OAuthRefreshThreshold] of
// expiry, and otherwise returns the current one without any network call.
//
// The no-refresh-needed case is served under a read lock and performs one time
// comparison, no I/O, and no allocation, so driving a large batch of requests
// through this provider costs no meaningful contention.
func (p *authTokenProvider) AuthToken(ctx context.Context) (string, error) {
	p.mu.RLock()
	authToken, servable := p.cfg.AuthToken, p.servableWithoutRefresh()
	p.mu.RUnlock()
	if servable {
		return authToken, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Re-check under the write lock: another goroutine may have refreshed while
	// this one waited for it.
	if p.servableWithoutRefresh() {
		return p.cfg.AuthToken, nil
	}
	if err := p.refresh(ctx, refreshIfExpiring); err != nil {
		return "", err
	}
	return p.cfg.AuthToken, nil
}

// servableWithoutRefresh reports whether the token currently held can be handed
// out as-is.
// Callers must hold at least a read lock on p.mu.
//
// A configuration with no OAuth state has nothing to refresh, so its token is
// always servable as long as it is non-empty. An empty one falls through to the
// slow path, which reports why it cannot be served.
func (p *authTokenProvider) servableWithoutRefresh() bool {
	if p.storeErr != nil {
		return false
	}
	if p.cfg.OAuth == nil {
		return p.cfg.AuthToken != ""
	}
	return time.Until(p.cfg.OAuth.ExpiresAt) > OAuthRefreshThreshold
}

// ForceRefreshAuthToken implements [dash0.RefreshingAuthTokenProvider].
// It replaces staleAuthToken regardless of how much validity it had left,
// unless something already replaced it.
//
// The staleAuthToken comparison is what keeps one revocation from rotating the
// refresh token once per in-flight request. Every caller serializes on p.mu, so
// the first one through mints a replacement and the rest find a token that no
// longer matches the one they were rejected with and return it as-is.
// [refreshOAuthToken] handles the same race across processes, by re-reading
// the profile after taking the file lock.
//
// The name is exported because a cross-package interface can only be satisfied
// by exported methods. It is not part of this package's public surface: the
// authTokenProvider type is unexported, so the only way to reach this method is
// through the interface.
func (p *authTokenProvider) ForceRefreshAuthToken(ctx context.Context, staleAuthToken string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cfg.AuthToken != "" && p.cfg.AuthToken != staleAuthToken {
		return p.cfg.AuthToken, nil
	}
	if err := p.refresh(ctx, forceRefresh); err != nil {
		return "", err
	}
	return p.cfg.AuthToken, nil
}

// refresh refreshes the provider's own configuration copy.
// Callers must hold p.mu.
func (p *authTokenProvider) refresh(ctx context.Context, mode refreshMode) error {
	if p.storeErr != nil {
		return p.storeErr
	}
	if p.cfg.ProfileName == "" {
		// An empty ConfigDir is fine: the store then resolves the directory from
		// DASH0_CONFIG_DIR or the home directory. An empty profile name is not,
		// because there is nowhere to write the rotated tokens.
		return ErrNoAssociatedProfile
	}
	// Detach from the caller's context. A refresh cancelled between the
	// authorization server rotating the refresh token and the response being
	// read burns the stored credential: the rotated token is never persisted,
	// and the next refresh gets invalid_grant. Since the provider is consulted
	// on the API request's own deadline, an ordinary request timeout would
	// otherwise be enough to force an interactive re-login. This is the same
	// hazard exchangeRefreshToken cites as the reason it never retries.
	//
	// Cancellation is dropped but the values are kept, and the refresh gets its
	// own bound so a hung authorization server or a sibling process holding
	// .profile-lock cannot pin the caller indefinitely.
	refreshCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), OAuthRefreshTimeout)
	defer cancel()

	if err := refreshOAuthToken(refreshCtx, p.store, p.cfg.ProfileName, &p.cfg, mode); err != nil {
		return err
	}
	if p.cfg.OAuth == nil && p.cfg.AuthToken == "" {
		// The refresh cleared the profile's OAuth state -- the refresh token was
		// rejected, or it was cleared concurrently -- leaving no credential to
		// serve. Report the terminal condition rather than handing back an empty
		// token for the client to reject as malformed.
		return ErrReauthenticationRequired
	}
	return nil
}
