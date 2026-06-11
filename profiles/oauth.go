package profiles

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	dash0 "github.com/dash0hq/dash0-api-client-go"
)

// OAuthRefreshThreshold is how far before expiry a token refresh is attempted.
const OAuthRefreshThreshold = 5 * time.Minute

// OAuthRefreshMaxExpiresIn caps the trusted lifetime of an OAuth access token.
// A token endpoint that returns expires_in beyond this ceiling is treated as
// untrustworthy and the response is rejected before the caller can adopt it.
const OAuthRefreshMaxExpiresIn = 24 * time.Hour

// OAuthRefreshTokenEndpointRetries is the number of additional attempts
// refreshOAuthToken makes when the token endpoint returns a transient failure
// (5xx response or network error).
// A single transient blip should not force the user to re-authenticate.
const OAuthRefreshTokenEndpointRetries = 2

// OAuthRefreshTokenEndpointRetryDelay is the fixed delay between retries of
// the token endpoint when the previous attempt failed transiently.
const OAuthRefreshTokenEndpointRetryDelay = 250 * time.Millisecond

// ErrReauthenticationRequired is returned by refresh-bearing operations when
// the OAuth refresh token is no longer accepted by the authorization server
// (RFC 6749 invalid_grant) and the caller must initiate a fresh interactive
// login.
// Callers should treat this as a terminal state for the affected profile: the
// stored refresh token has been cleared from disk before this error is
// returned, so a subsequent call no longer retries the dead credential.
var ErrReauthenticationRequired = errors.New("OAuth refresh token rejected; re-authentication required")

// ErrRevocationFailed is wrapped into the result of [Store.RemoveProfile]
// (and equivalents) when the local profile was successfully removed but the
// best-effort revocation of the OAuth refresh token failed.
// The refresh token may still be live on the authorization server until its
// natural expiry.
// Callers should detect this via [errors.Is] and surface a "revoke manually"
// hint to the user.
var ErrRevocationFailed = errors.New("OAuth refresh token revocation failed; the token may still be valid on the authorization server")

// refreshOAuthToken checks whether the OAuth access token in cfg needs
// refreshing and, if so, performs a token refresh using the Dash0 OAuth API.
// On success it persists the new tokens via [Store.UpdateProfile] *before*
// updating cfg, so a persistence failure leaves the caller's in-memory state
// unchanged rather than diverging from disk.
//
// The function is a no-op when cfg.OAuth is nil or the token is not close to
// expiry.
//
// Concurrent calls are serialized at two levels:
//   - Within a single process, [Store.refreshMu] guards the refresh + persist
//     sequence so goroutines do not race against each other.
//   - Across processes sharing the same config directory, an OS-level
//     advisory lock on .profile-lock (acquired via [acquireProfileLock],
//     backed by the [github.com/gofrs/flock] cross-platform library)
//     prevents two CLI invocations from concurrently rotating the same
//     refresh-token family.
//
// After acquiring the locks the profile is re-read from disk so a process
// that lost the race picks up the freshly persisted tokens instead of
// refreshing again with a stale refresh token.
//
// On a server response of invalid_grant the stored OAuth state is cleared
// from disk and [ErrReauthenticationRequired] is returned, so a subsequent
// call does not retry the dead credential.
func refreshOAuthToken(ctx context.Context, store *Store, profileName string, cfg *Configuration) error {
	if cfg.OAuth == nil {
		return nil
	}

	if time.Until(cfg.OAuth.ExpiresAt) > OAuthRefreshThreshold {
		return nil
	}

	if cfg.ApiUrl == "" {
		return fmt.Errorf("OAuth token refresh requires an API URL")
	}

	store.refreshMu.Lock()
	defer store.refreshMu.Unlock()

	// Acquire the cross-process lock as well so a sibling CLI invocation
	// sharing this config directory cannot refresh concurrently and
	// invalidate our rotated refresh token. The lock is held until this
	// function returns.
	releaseFileLock, err := acquireProfileLock(ctx, store.configDir)
	if err != nil {
		return fmt.Errorf("failed to acquire cross-process refresh lock: %w", err)
	}
	defer releaseFileLock()

	// Re-read the profile from disk; another goroutine or process may have
	// refreshed the token while we were waiting for the lock.
	profiles, err := store.GetProfiles()
	if err != nil {
		return fmt.Errorf("failed to re-read profiles for OAuth refresh: %w", err)
	}
	var profileFound bool
	var persistedOAuth *OAuthState
	var persistedAuthToken string
	for _, p := range profiles {
		if p.Name == profileName {
			profileFound = true
			persistedAuthToken = p.Configuration.AuthToken
			persistedOAuth = p.Configuration.OAuth
			break
		}
	}
	if !profileFound {
		// The profile was removed from disk concurrently. Surface a
		// dedicated error rather than silently zeroing cfg.AuthToken.
		return fmt.Errorf("OAuth refresh aborted: profile %q no longer exists", profileName)
	}
	if persistedOAuth == nil {
		// The profile still exists but its OAuth state was cleared
		// concurrently (re-authentication elsewhere, profile rewritten
		// without OAuth, etc). Reflect that in cfg and bail out.
		cfg.AuthToken = persistedAuthToken
		cfg.OAuth = nil
		return nil
	}

	// Adopt the latest persisted state, then re-check expiry.
	cfg.AuthToken = persistedAuthToken
	*cfg.OAuth = *persistedOAuth
	if time.Until(cfg.OAuth.ExpiresAt) > OAuthRefreshThreshold {
		return nil
	}

	oauthClient, err := dash0.NewOAuthClient(dash0.WithApiUrl(cfg.ApiUrl))
	if err != nil {
		return fmt.Errorf("failed to create OAuth client for token refresh: %w", err)
	}
	defer func() { _ = oauthClient.Close(ctx) }()

	resp, err := exchangeRefreshTokenWithRetry(ctx, oauthClient, cfg.OAuth.ClientID, cfg.OAuth.RefreshToken)
	if err != nil {
		if dash0.IsOAuthInvalidGrant(err) {
			// The refresh token has been revoked, expired, or otherwise
			// rejected. Clear the dead state from disk so the next
			// invocation does not retry the same credential, and signal
			// the caller that interactive re-auth is required.
			// updateProfileLocked is used here because we already hold
			// both the in-process mutex and the .profile-lock file lock.
			clearErr := store.updateProfileLocked(profileName, func(persisted *Configuration) {
				persisted.OAuth = nil
				persisted.AuthToken = ""
			})
			cfg.OAuth = nil
			cfg.AuthToken = ""
			if clearErr != nil {
				return fmt.Errorf("%w (and failed to clear stored state: %v)", ErrReauthenticationRequired, clearErr)
			}
			return ErrReauthenticationRequired
		}
		return fmt.Errorf("OAuth token refresh failed: %w", err)
	}

	if err := validateRefreshResponse(resp); err != nil {
		return fmt.Errorf("OAuth token refresh: %w", err)
	}

	newExpiresAt := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	newRefreshToken := cfg.OAuth.RefreshToken
	if resp.RefreshToken != nil && *resp.RefreshToken != "" {
		newRefreshToken = *resp.RefreshToken
	}
	newAccessToken := resp.AccessToken

	// Persist first; only mutate cfg if the write succeeded. A persistence
	// failure after a successful exchange means the server-side refresh
	// token has rotated but the local file still has the old one; better to
	// surface the error and leave cfg untouched than to leave the in-memory
	// process holding tokens disk has never seen.
	if err := store.updateProfileLocked(profileName, func(persisted *Configuration) {
		persisted.AuthToken = newAccessToken
		if persisted.OAuth != nil {
			persisted.OAuth.ExpiresAt = newExpiresAt
			persisted.OAuth.RefreshToken = newRefreshToken
		}
	}); err != nil {
		return fmt.Errorf("OAuth token refresh succeeded but persisting tokens failed: %w", err)
	}

	cfg.AuthToken = newAccessToken
	cfg.OAuth.ExpiresAt = newExpiresAt
	cfg.OAuth.RefreshToken = newRefreshToken

	return nil
}

// exchangeRefreshTokenWithRetry calls ExchangeToken with a refresh_token
// grant, retrying once for transient failures (5xx or network errors).
// A non-retriable error — most notably an OAuth invalid_grant on a 400 — is
// returned immediately so callers can react without burning extra round-trips.
func exchangeRefreshTokenWithRetry(ctx context.Context, oauthClient dash0.OAuthClient, clientID, refreshToken string) (*dash0.OAuthTokenResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= OAuthRefreshTokenEndpointRetries; attempt++ {
		resp, err := oauthClient.ExchangeToken(ctx, &dash0.OAuthTokenRequest{
			GrantType:    dash0.OAuthGrantTypeRefreshToken,
			RefreshToken: dash0.Ptr(refreshToken),
			ClientId:     dash0.Ptr(clientID),
		})
		if err == nil {
			return resp, nil
		}
		lastErr = err
		// 4xx and structured OAuth errors are not retried — the IdP has
		// made a deliberate decision. Only transient errors (5xx, network)
		// get another attempt.
		if !isTransientRefreshError(err) {
			return nil, err
		}
		if attempt < OAuthRefreshTokenEndpointRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(OAuthRefreshTokenEndpointRetryDelay):
			}
		}
	}
	return nil, lastErr
}

// isTransientRefreshError reports whether err is the kind of failure that is
// worth retrying once at the token endpoint: a 5xx response, or a transport
// error that does not carry a 4xx classification.
func isTransientRefreshError(err error) bool {
	if dash0.IsServerError(err) {
		return true
	}
	// IsOAuthTokenError covers 400 structured responses. Any non-OAuth,
	// non-APIError-classified failure (network reset, connection refused,
	// TLS handshake error, etc.) is a transport failure worth retrying.
	if dash0.IsOAuthTokenError(err) {
		return false
	}
	var apiErr *dash0.APIError
	if errors.As(err, &apiErr) {
		// Any classifiable HTTP response other than 5xx: do not retry.
		return false
	}
	// Default: treat unclassified errors as transient (network-level).
	return true
}

// validateRefreshResponse rejects responses that violate the basic invariants
// of an OAuth token-endpoint reply.
// The IdP can return anything, including hostile or buggy values; trusting
// expires_in <= 0, missing access_token, or unexpected token_type leads to a
// runaway refresh loop or silent authentication failures.
func validateRefreshResponse(resp *dash0.OAuthTokenResponse) error {
	if resp == nil {
		return fmt.Errorf("nil response from token endpoint")
	}
	if resp.AccessToken == "" {
		return fmt.Errorf("token endpoint returned empty access_token")
	}
	if resp.TokenType != "" && !strings.EqualFold(resp.TokenType, "Bearer") {
		return fmt.Errorf("token endpoint returned unsupported token_type %q (only Bearer is accepted)", resp.TokenType)
	}
	if resp.ExpiresIn <= 0 {
		return fmt.Errorf("token endpoint returned non-positive expires_in (%d); refusing to enter a refresh loop", resp.ExpiresIn)
	}
	if maxSeconds := int64(OAuthRefreshMaxExpiresIn / time.Second); int64(resp.ExpiresIn) > maxSeconds || int64(resp.ExpiresIn) > math.MaxInt32 {
		return fmt.Errorf("token endpoint returned implausibly large expires_in (%d); refusing", resp.ExpiresIn)
	}
	return nil
}

// revokeOAuthTokens revokes the OAuth refresh token associated with cfg.
// Revoking the refresh token transitively revokes all descendant access tokens
// on the server side.
//
// The function is a no-op when cfg.OAuth is nil or cfg.ApiUrl is empty.
// Errors are returned so callers can decide whether to propagate or ignore them.
func revokeOAuthTokens(ctx context.Context, cfg *Configuration) error {
	if cfg.OAuth == nil {
		return nil
	}
	if cfg.ApiUrl == "" {
		return nil
	}

	oauthClient, err := dash0.NewOAuthClient(dash0.WithApiUrl(cfg.ApiUrl))
	if err != nil {
		return fmt.Errorf("failed to create OAuth client for token revocation: %w", err)
	}
	defer func() { _ = oauthClient.Close(ctx) }()

	hint := dash0.OAuthTokenTypeRefreshToken
	if err := oauthClient.RevokeToken(ctx, &dash0.OAuthRevocationRequest{
		Token:         cfg.OAuth.RefreshToken,
		TokenTypeHint: &hint,
	}); err != nil {
		return fmt.Errorf("OAuth token revocation failed: %w", err)
	}

	return nil
}
