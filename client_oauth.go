package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// OAuthClient provides methods for the Dash0 OAuth 2.0 authorization flow.
// These endpoints do not require API key authentication.
// Use [NewOAuthClient] to create an instance.
type OAuthClient interface {
	// GetAuthorizationServerMetadata retrieves the OAuth 2.0 authorization
	// server metadata (RFC 8414) from the well-known endpoint.
	GetAuthorizationServerMetadata(ctx context.Context) (*OAuthAuthorizationServerMetadata, error)

	// GetProtectedResourceMetadata retrieves the OAuth 2.0 protected resource
	// metadata (RFC 9728) from the well-known endpoint.
	GetProtectedResourceMetadata(ctx context.Context) (*OAuthProtectedResourceMetadata, error)

	// AuthorizeURL builds the OAuth 2.0 authorization URL for the
	// authorization code flow with PKCE.
	// This method does not make an HTTP call; it constructs the URL from the
	// given parameters and the client's API URL.
	AuthorizeURL(params *AuthorizeURLParams) (string, error)

	// RegisterClient performs OAuth 2.0 dynamic client registration (RFC 7591).
	RegisterClient(ctx context.Context, request *OAuthClientRegistrationRequest) (*OAuthClientRegistrationResponse, error)

	// ExchangeToken exchanges an authorization code or refresh token for an
	// access token at the OAuth 2.0 token endpoint.
	// Set [OAuthTokenRequest].GrantType to select the grant type.
	// On a 400 response the error is an [*OAuthTokenError] with the structured
	// OAuth error fields.
	ExchangeToken(ctx context.Context, request *OAuthTokenRequest) (*OAuthTokenResponse, error)

	// RevokeToken revokes an access or refresh token (RFC 7009).
	RevokeToken(ctx context.Context, request *OAuthRevocationRequest) error

	// Close releases resources associated with the client.
	Close(ctx context.Context) error
}

// AuthorizeURLParams contains the parameters for building an OAuth 2.0
// authorization URL.
type AuthorizeURLParams struct {
	// ResponseType must be "code" for the authorization code flow.
	ResponseType string

	// ClientID is the client identifier obtained during registration.
	ClientID string

	// RedirectURI must match one of the client's registered redirect URIs.
	RedirectURI string

	// Scope is an optional space-separated list of requested scopes.
	Scope *string

	// State is an optional opaque value for CSRF protection, returned
	// unchanged in the redirect.
	State *string

	// CodeChallenge is the PKCE code challenge (RFC 7636).
	CodeChallenge string

	// CodeChallengeMethod must be "S256".
	CodeChallengeMethod string

	// Prompt is an optional space-separated list of prompt directives.
	// Supported value: "consent".
	Prompt *string
}

// oauthClient is the concrete implementation of OAuthClient.
type oauthClient struct {
	inner  *ClientWithResponses
	apiURL string
}

// NewOAuthClient creates a new OAuth client for the Dash0 API.
// It accepts the same [ClientOption] functions as [NewClient].
// [WithApiUrl] is required; [WithAuthToken] is not needed and is ignored.
// Only [WithApiUrl], [WithHTTPClient], [WithTimeout], and [WithUserAgent] are
// meaningful; other options are accepted but have no effect.
//
// Example:
//
//	client, err := dash0.NewOAuthClient(
//	    dash0.WithApiUrl("https://api.eu-west-1.aws.dash0.com"),
//	)
func NewOAuthClient(opts ...ClientOption) (OAuthClient, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.apiUrl == "" {
		return nil, fmt.Errorf("dash0: API URL is required for OAuthClient (use WithApiUrl)")
	}

	var httpClient *http.Client
	if cfg.httpClient != nil {
		httpClient = &http.Client{
			Transport:     cfg.httpClient.Transport,
			Timeout:       cfg.httpClient.Timeout,
			CheckRedirect: cfg.httpClient.CheckRedirect,
			Jar:           cfg.httpClient.Jar,
		}
		if cfg.timeoutSet {
			httpClient.Timeout = cfg.timeout
		}
	} else {
		httpClient = &http.Client{
			Timeout: cfg.timeout,
		}
	}

	userAgentEditor := func(_ context.Context, req *http.Request) error {
		req.Header.Set("User-Agent", cfg.userAgent)
		return nil
	}

	inner, err := NewClientWithResponses(
		cfg.apiUrl,
		withGeneratedHTTPClient(httpClient),
		WithRequestEditorFn(userAgentEditor),
	)
	if err != nil {
		return nil, fmt.Errorf("dash0: failed to create OAuth client: %w", err)
	}

	return &oauthClient{
		inner:  inner,
		apiURL: cfg.apiUrl,
	}, nil
}

// GetAuthorizationServerMetadata retrieves the OAuth 2.0 authorization server
// metadata from /.well-known/oauth-authorization-server.
func (c *oauthClient) GetAuthorizationServerMetadata(ctx context.Context) (*OAuthAuthorizationServerMetadata, error) {
	resp, err := c.inner.GetWellKnownOauthAuthorizationServerWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("dash0: get authorization server metadata failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// GetProtectedResourceMetadata retrieves the OAuth 2.0 protected resource
// metadata from /.well-known/oauth-protected-resource.
func (c *oauthClient) GetProtectedResourceMetadata(ctx context.Context) (*OAuthProtectedResourceMetadata, error) {
	resp, err := c.inner.GetWellKnownOauthProtectedResourceWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("dash0: get protected resource metadata failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// AuthorizeURL builds the OAuth 2.0 authorization URL without making an HTTP
// request.
func (c *oauthClient) AuthorizeURL(params *AuthorizeURLParams) (string, error) {
	genParams := &GetOauthAuthorizeParams{
		ResponseType:        params.ResponseType,
		ClientId:            params.ClientID,
		RedirectUri:         params.RedirectURI,
		Scope:               params.Scope,
		State:               params.State,
		CodeChallenge:       params.CodeChallenge,
		CodeChallengeMethod: params.CodeChallengeMethod,
		Prompt:              params.Prompt,
	}
	req, err := NewGetOauthAuthorizeRequest(c.apiURL, genParams)
	if err != nil {
		return "", fmt.Errorf("dash0: build authorize URL failed: %w", err)
	}
	return req.URL.String(), nil
}

// RegisterClient performs OAuth 2.0 dynamic client registration.
func (c *oauthClient) RegisterClient(ctx context.Context, request *OAuthClientRegistrationRequest) (*OAuthClientRegistrationResponse, error) {
	resp, err := c.inner.PostOauthRegisterWithResponse(ctx, *request)
	if err != nil {
		return nil, fmt.Errorf("dash0: register client failed: %w", err)
	}
	if resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON201 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON201, nil
}

// ExchangeToken exchanges an authorization code or refresh token for an access
// token.
func (c *oauthClient) ExchangeToken(ctx context.Context, request *OAuthTokenRequest) (*OAuthTokenResponse, error) {
	resp, err := c.inner.PostOauthTokenWithFormdataBodyWithResponse(ctx, *request)
	if err != nil {
		return nil, fmt.Errorf("dash0: exchange token failed: %w", err)
	}
	if resp.StatusCode() == http.StatusBadRequest && resp.JSON400 != nil {
		return nil, &OAuthTokenError{
			StatusCode:  resp.StatusCode(),
			Code:        resp.JSON400.Error,
			Description: StringValue(resp.JSON400.ErrorDescription),
			URI:         StringValue(resp.JSON400.ErrorUri),
		}
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// RevokeToken revokes an access or refresh token.
func (c *oauthClient) RevokeToken(ctx context.Context, request *OAuthRevocationRequest) error {
	resp, err := c.inner.PostOauthRevokeWithFormdataBodyWithResponse(ctx, *request)
	if err != nil {
		return fmt.Errorf("dash0: revoke token failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// Close is a no-op for OAuthClient.
func (c *oauthClient) Close(_ context.Context) error {
	return nil
}
