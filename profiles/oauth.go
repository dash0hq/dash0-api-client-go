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

	resp, err := exchangeRefreshToken(ctx, oauthClient, cfg.OAuth.ClientID, cfg.OAuth.RefreshToken)
	if err != nil {
		if dash0.IsOAuthTerminalError(err) {
			// The refresh token has been revoked, expired, or otherwise
			// rejected — RFC 6749 §5.2 invalid_grant / invalid_client /
			// unauthorized_client are all terminal for the stored
			// credential. Clear the dead state from disk so the next
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

	trustedExpiresIn, err := validateRefreshResponse(resp)
	if err != nil {
		// The IdP returned 200 OK and (per RFC 6749 §6) may have already
		// rotated the refresh token before we rejected its response. The
		// on-disk credential is therefore stale: the next refresh will
		// receive invalid_grant. Treat this case as terminal — clear stored
		// state and surface ErrReauthenticationRequired — so the caller
		// learns about the forced re-auth immediately instead of after a
		// confusing extra round-trip.
		clearErr := store.updateProfileLocked(profileName, func(persisted *Configuration) {
			persisted.OAuth = nil
			persisted.AuthToken = ""
		})
		cfg.OAuth = nil
		cfg.AuthToken = ""
		if clearErr != nil {
			return fmt.Errorf("%w (response validation failed: %v; and failed to clear stored state: %v)",
				ErrReauthenticationRequired, err, clearErr)
		}
		return fmt.Errorf("%w (response validation failed: %v)", ErrReauthenticationRequired, err)
	}

	newExpiresAt := time.Now().Add(time.Duration(trustedExpiresIn) * time.Second)
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
	clientID := ResolveOAuthClientID(cfg.ApiUrl, cfg.OAuth.ClientID, WithConfigDir(store.configDir))
	if err := store.updateProfileLocked(profileName, func(persisted *Configuration) {
		persisted.AuthToken = newAccessToken
		if persisted.OAuth != nil {
			persisted.OAuth.ExpiresAt = newExpiresAt
			persisted.OAuth.RefreshToken = newRefreshToken
			if persisted.OAuth.ClientID == "" && clientID != "" {
				persisted.OAuth.ClientID = clientID
			}
		}
	}); err != nil {
		return fmt.Errorf("OAuth token refresh succeeded but persisting tokens failed: %w", err)
	}

	cfg.AuthToken = newAccessToken
	cfg.OAuth.ExpiresAt = newExpiresAt
	cfg.OAuth.RefreshToken = newRefreshToken
	if cfg.OAuth.ClientID == "" && clientID != "" {
		cfg.OAuth.ClientID = clientID
	}

	return nil
}

// exchangeRefreshToken calls the token endpoint with a refresh_token grant.
//
// The library does not retry refresh-token grants on transient (5xx, network)
// errors: the authorization server may have already rotated the refresh token
// before failing to deliver its response, so a retry would re-send a now-burned
// credential and the server would reply invalid_grant, forcing
// re-authentication.
// Any failure — including transient ones — is propagated unchanged so the
// caller can surface a "try again later" error rather than wedge the stored
// credential.
func exchangeRefreshToken(ctx context.Context, oauthClient dash0.OAuthClient, clientID, refreshToken string) (*dash0.OAuthTokenResponse, error) {
	return oauthClient.ExchangeToken(ctx, &dash0.OAuthTokenRequest{
		GrantType:    dash0.OAuthGrantTypeRefreshToken,
		RefreshToken: dash0.Ptr(refreshToken),
		ClientId:     dash0.Ptr(clientID),
	})
}

// OAuthRefreshExpiresInDefault is the lifetime assumed when the IdP omits
// expires_in.
// RFC 6749 §5.1 makes expires_in RECOMMENDED but not required; rather than
// reject a compliant response that omits it, the library falls back to this
// conservative default, which is comfortably above [OAuthRefreshMinExpiresIn]
// so an immediate refresh storm does not follow.
const OAuthRefreshExpiresInDefault = 1 * time.Hour

// OAuthRefreshMinExpiresIn is the minimum trusted lifetime of a refreshed
// access token.
// Tokens shorter than this would cross [OAuthRefreshThreshold] almost
// immediately and trigger a per-minute refresh storm; the IdP either
// misconfigured the client or is unhealthy, and the safe response is to
// reject the rotation and surface a clear error.
// The 2× factor preserves a useful working window: at least one
// OAuthRefreshThreshold of "fresh" plus another of "still serviceable".
const OAuthRefreshMinExpiresIn = 2 * OAuthRefreshThreshold

// validateRefreshResponse rejects responses that violate the basic invariants
// of an OAuth token-endpoint reply, and applies a conservative default for
// the recommended-but-omitted expires_in field.
// The IdP can return anything, including hostile or buggy values; trusting
// missing access_token, unexpected token_type, or a refresh-storm-inducing
// expires_in leads to a runaway refresh loop or silent authentication failures.
// On success the returned int is the trusted expires_in value in seconds
// (either the IdP-supplied value or [OAuthRefreshExpiresInDefault]).
func validateRefreshResponse(resp *dash0.OAuthTokenResponse) (int, error) {
	if resp == nil {
		return 0, fmt.Errorf("nil response from token endpoint")
	}
	if resp.AccessToken == "" {
		return 0, fmt.Errorf("token endpoint returned empty access_token")
	}
	if resp.TokenType != "" && !strings.EqualFold(resp.TokenType, "Bearer") {
		return 0, fmt.Errorf("token endpoint returned unsupported token_type %q (only Bearer is accepted)", resp.TokenType)
	}

	expiresIn := int64(resp.ExpiresIn)
	if expiresIn == 0 {
		// RFC 6749 §5.1: expires_in is RECOMMENDED, not required.
		// Fall back to a conservative default rather than reject.
		expiresIn = int64(OAuthRefreshExpiresInDefault / time.Second)
	}
	if expiresIn < 0 {
		return 0, fmt.Errorf("token endpoint returned negative expires_in (%d); refusing", resp.ExpiresIn)
	}
	if minSeconds := int64(OAuthRefreshMinExpiresIn / time.Second); expiresIn < minSeconds {
		return 0, fmt.Errorf(
			"token endpoint returned expires_in (%d s) below the minimum refresh-storm floor (%d s); refusing to enter a refresh loop",
			expiresIn, minSeconds,
		)
	}
	if maxSeconds := int64(OAuthRefreshMaxExpiresIn / time.Second); expiresIn > maxSeconds || expiresIn > math.MaxInt32 {
		return 0, fmt.Errorf("token endpoint returned implausibly large expires_in (%d); refusing", resp.ExpiresIn)
	}
	return int(expiresIn), nil
}

// ResolveOAuthClientID returns the client_id to send with token and revoke
// requests. The profile's stored value wins; if it is empty, the DCR cache
// for apiURL is consulted. An empty result means the caller must omit
// client_id rather than send an empty parameter.
func ResolveOAuthClientID(apiURL, stored string, opts ...StoreOption) string {
	if stored != "" {
		return stored
	}
	if apiURL == "" {
		return ""
	}
	store, err := NewOAuthClientStore(opts...)
	if err != nil {
		return ""
	}
	rec, ok, err := store.Get(apiURL)
	if err != nil || !ok {
		return ""
	}
	return rec.ClientID
}

// revokeOAuthTokens revokes the OAuth refresh token associated with cfg.
// Revoking the refresh token transitively revokes all descendant access tokens
// on the server side.
//
// The function is a no-op when cfg.OAuth is nil or cfg.ApiUrl is empty.
// Errors are returned so callers can decide whether to propagate or ignore them.
// configDir scopes the DCR-cache fallback used when cfg.OAuth.ClientID is empty.
func revokeOAuthTokens(ctx context.Context, cfg *Configuration, configDir string) error {
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
	req := &dash0.OAuthRevocationRequest{
		Token:         cfg.OAuth.RefreshToken,
		TokenTypeHint: &hint,
	}
	var opts []StoreOption
	if configDir != "" {
		opts = append(opts, WithConfigDir(configDir))
	}
	if clientID := ResolveOAuthClientID(cfg.ApiUrl, cfg.OAuth.ClientID, opts...); clientID != "" {
		req.ClientId = clientID
	}
	if err := oauthClient.RevokeToken(ctx, req); err != nil {
		return fmt.Errorf("OAuth token revocation failed: %w", err)
	}

	return nil
}
