package profiles

import (
	"context"
	"fmt"
	"time"

	dash0 "github.com/dash0hq/dash0-api-client-go"
)

// OAuthRefreshThreshold is how far before expiry a token refresh is attempted.
const OAuthRefreshThreshold = 5 * time.Minute

// refreshOAuthToken checks whether the OAuth access token in cfg needs
// refreshing and, if so, performs a token refresh using the Dash0 OAuth API.
// On success it updates cfg in place and persists the new tokens via
// [Store.UpdateProfile].
//
// The function is a no-op when cfg.OAuth is nil or the token is not close to
// expiry.
//
// Concurrent calls for the same store are serialized via store.refreshMu so
// that two goroutines do not race to refresh and end up invalidating each
// other's rotated refresh token. After acquiring the lock the profile is
// re-read from disk so a goroutine that lost the race picks up the freshly
// persisted tokens instead of refreshing again with a stale refresh token.
func refreshOAuthToken(store *Store, profileName string, cfg *Configuration) error {
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

	// Re-read the profile from disk; another goroutine may have refreshed
	// the token while we were waiting for the lock.
	profiles, err := store.GetProfiles()
	if err != nil {
		return fmt.Errorf("failed to re-read profiles for OAuth refresh: %w", err)
	}
	var persistedOAuth *OAuthState
	var persistedAuthToken string
	for _, p := range profiles {
		if p.Name == profileName {
			persistedAuthToken = p.Configuration.AuthToken
			persistedOAuth = p.Configuration.OAuth
			break
		}
	}
	if persistedOAuth == nil {
		// Profile no longer has OAuth state (removed or cleared
		// concurrently); reflect that in cfg and bail out.
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
	defer func() { _ = oauthClient.Close(context.Background()) }()

	resp, err := oauthClient.ExchangeToken(context.Background(), &dash0.OAuthTokenRequest{
		GrantType:    dash0.OAuthGrantTypeRefreshToken,
		RefreshToken: dash0.Ptr(cfg.OAuth.RefreshToken),
		ClientId:     dash0.Ptr(cfg.OAuth.ClientID),
	})
	if err != nil {
		return fmt.Errorf("OAuth token refresh failed: %w", err)
	}

	newExpiresAt := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	newRefreshToken := cfg.OAuth.RefreshToken
	if resp.RefreshToken != nil {
		newRefreshToken = *resp.RefreshToken
	}

	cfg.AuthToken = resp.AccessToken
	cfg.OAuth.ExpiresAt = newExpiresAt
	cfg.OAuth.RefreshToken = newRefreshToken

	return store.UpdateProfile(profileName, func(persisted *Configuration) {
		persisted.AuthToken = resp.AccessToken
		if persisted.OAuth != nil {
			persisted.OAuth.ExpiresAt = newExpiresAt
			persisted.OAuth.RefreshToken = newRefreshToken
		}
	})
}

// revokeOAuthTokens revokes the OAuth refresh token associated with cfg.
// Revoking the refresh token transitively revokes all descendant access tokens
// on the server side.
//
// The function is a no-op when cfg.OAuth is nil or cfg.ApiUrl is empty.
// Errors are returned so callers can decide whether to propagate or ignore them.
func revokeOAuthTokens(cfg *Configuration) error {
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
	defer func() { _ = oauthClient.Close(context.Background()) }()

	hint := dash0.OAuthTokenTypeRefreshToken
	if err := oauthClient.RevokeToken(context.Background(), &dash0.OAuthRevocationRequest{
		Token:         cfg.OAuth.RefreshToken,
		TokenTypeHint: &hint,
	}); err != nil {
		return fmt.Errorf("OAuth token revocation failed: %w", err)
	}

	return nil
}
