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
func refreshOAuthToken(store *Store, profileName string, cfg *Configuration) error {
	oauth := cfg.OAuth
	if oauth == nil {
		return nil
	}

	if time.Until(oauth.ExpiresAt) > OAuthRefreshThreshold {
		return nil
	}

	if cfg.ApiUrl == "" {
		return fmt.Errorf("OAuth token refresh requires an API URL")
	}

	oauthClient, err := dash0.NewOAuthClient(dash0.WithApiUrl(cfg.ApiUrl))
	if err != nil {
		return fmt.Errorf("failed to create OAuth client for token refresh: %w", err)
	}
	defer func() { _ = oauthClient.Close(context.Background()) }()

	resp, err := oauthClient.ExchangeToken(context.Background(), &dash0.OAuthTokenRequest{
		GrantType:    dash0.OAuthGrantTypeRefreshToken,
		RefreshToken: dash0.Ptr(oauth.RefreshToken),
		ClientId:     dash0.Ptr(oauth.ClientID),
	})
	if err != nil {
		return fmt.Errorf("OAuth token refresh failed: %w", err)
	}

	newExpiresAt := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	newRefreshToken := oauth.RefreshToken
	if resp.RefreshToken != nil {
		newRefreshToken = *resp.RefreshToken
	}

	cfg.AuthToken = resp.AccessToken
	oauth.ExpiresAt = newExpiresAt
	oauth.RefreshToken = newRefreshToken

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
