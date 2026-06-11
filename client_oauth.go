package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// OAuthResponseTypeCode is the prefixed alias for the [Code] response type
// (RFC 6749 §4.1).
// Prefer this name in user code — the bare [Code] survives only for the
// oapi-codegen-generated constant set.
const OAuthResponseTypeCode = Code

// OAuthCodeChallengeMethodS256 is the prefixed alias for the [S256] PKCE
// code-challenge method (RFC 7636).
// Prefer this name in user code — the bare [S256] survives only for the
// oapi-codegen-generated constant set.
const OAuthCodeChallengeMethodS256 = S256

// OAuthClient provides methods for the Dash0 OAuth 2.0 authorization flow.
// These endpoints do not require API key authentication.
// Use [NewOAuthClient] to create an instance.
//
// For the inputs to [OAuthClient.AuthorizeURL] and [OAuthClient.ExchangeToken],
// the recommended way to obtain them is:
//   - [GeneratePKCEPair] for the PKCE verifier and S256 challenge (RFC 7636).
//   - [GenerateOAuthState] for the `state` parameter (RFC 6749 §10.12).
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
	// Implementations should at minimum drop idle HTTP connections held by
	// the underlying transport; the ctx is reserved for future
	// graceful-shutdown work and is currently unused.
	Close(ctx context.Context) error
}

// AuthorizeURLParams contains the parameters for building an OAuth 2.0
// authorization URL.
type AuthorizeURLParams struct {
	// ResponseType must be [OAuthResponseTypeCode].
	// Defaults to [OAuthResponseTypeCode] when empty.
	ResponseType OAuthResponseType

	// ClientID is the client identifier obtained during registration.
	ClientID string

	// RedirectURI must match one of the client's registered redirect URIs.
	RedirectURI string

	// Scope is an optional space-separated list of requested scopes.
	Scope *string

	// State is an opaque value bound to the request for CSRF protection
	// (RFC 6749 §10.12); the authorization server returns it unchanged in
	// the redirect.
	// State is mandatory: use [GenerateOAuthState] to produce a fresh value.
	State string

	// CodeChallenge is the PKCE code challenge (RFC 7636).
	CodeChallenge string

	// CodeChallengeMethod must be [OAuthCodeChallengeMethodS256].
	// Defaults to [OAuthCodeChallengeMethodS256] when empty.
	CodeChallengeMethod OAuthCodeChallengeMethod

	// Prompt is an optional space-separated list of prompt directives.
	// Supported value: "consent".
	Prompt *string
}

// oauthClient is the concrete implementation of OAuthClient.
type oauthClient struct {
	inner      *ClientWithResponses
	httpClient *http.Client
	apiURL     string
}

// NewOAuthClient creates a new OAuth client for the Dash0 API.
// It accepts a subset of the [ClientOption] functions used by [NewClient];
// only [WithApiUrl], [WithHTTPClient], [WithTimeout], and [WithUserAgent] are
// supported.
// [WithApiUrl] is required.
// Passing any other option (notably [WithAuthToken], retry/concurrency
// tuning, [WithTransport], or [WithOtlpEndpoint]) returns an error so the
// caller does not silently rely on a no-op.
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

	if err := validateOAuthOptions(cfg); err != nil {
		return nil, err
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
		inner:      inner,
		httpClient: httpClient,
		apiURL:     cfg.apiUrl,
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
//
// State is mandatory: an empty value is rejected because it is the only CSRF
// defense in the authorization-code-with-PKCE flow targeted by this client
// (RFC 6749 §10.12).
// ResponseType defaults to [OAuthResponseTypeCode] when empty; any other value
// is rejected.
// CodeChallengeMethod defaults to [OAuthCodeChallengeMethodS256] when empty;
// any other value is rejected to prevent PKCE downgrade.
func (c *oauthClient) AuthorizeURL(params *AuthorizeURLParams) (string, error) {
	if params == nil {
		return "", fmt.Errorf("dash0: AuthorizeURL requires non-nil params")
	}
	if params.ClientID == "" {
		return "", fmt.Errorf("dash0: AuthorizeURL requires a non-empty ClientID")
	}
	if params.RedirectURI == "" {
		return "", fmt.Errorf("dash0: AuthorizeURL requires a non-empty RedirectURI")
	}
	if params.State == "" {
		return "", fmt.Errorf("dash0: AuthorizeURL requires a non-empty State for CSRF protection; use GenerateOAuthState to produce one")
	}
	if params.CodeChallenge == "" {
		return "", fmt.Errorf("dash0: AuthorizeURL requires a non-empty CodeChallenge; use GeneratePKCEPair to produce one")
	}
	responseType := params.ResponseType
	if responseType == "" {
		responseType = OAuthResponseTypeCode
	} else if responseType != OAuthResponseTypeCode {
		return "", fmt.Errorf("dash0: AuthorizeURL ResponseType must be %q, got %q", OAuthResponseTypeCode, responseType)
	}
	challengeMethod := params.CodeChallengeMethod
	if challengeMethod == "" {
		challengeMethod = OAuthCodeChallengeMethodS256
	} else if challengeMethod != OAuthCodeChallengeMethodS256 {
		return "", fmt.Errorf("dash0: AuthorizeURL CodeChallengeMethod must be %q, got %q", OAuthCodeChallengeMethodS256, challengeMethod)
	}
	state := params.State
	genParams := &GetOauthAuthorizeParams{
		ResponseType:        string(responseType),
		ClientId:            params.ClientID,
		RedirectUri:         params.RedirectURI,
		Scope:               params.Scope,
		State:               &state,
		CodeChallenge:       params.CodeChallenge,
		CodeChallengeMethod: string(challengeMethod),
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
	if request == nil {
		return nil, fmt.Errorf("dash0: RegisterClient requires non-nil request")
	}
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
	if request == nil {
		return nil, fmt.Errorf("dash0: ExchangeToken requires non-nil request")
	}
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
// Per RFC 7009 §2.2 the authorization server returns 200 on success; some
// servers return 204 instead. Any 2xx status is treated as success.
func (c *oauthClient) RevokeToken(ctx context.Context, request *OAuthRevocationRequest) error {
	if request == nil {
		return fmt.Errorf("dash0: RevokeToken requires non-nil request")
	}
	resp, err := c.inner.PostOauthRevokeWithFormdataBodyWithResponse(ctx, *request)
	if err != nil {
		return fmt.Errorf("dash0: revoke token failed: %w", err)
	}
	if status := resp.StatusCode(); status < 200 || status >= 300 {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// Close releases idle HTTP connections held by the OAuth client's underlying
// [*http.Client].
// The provided context is currently unused; the signature accepts one for
// symmetry with [Client.Close] and to leave room for future graceful-shutdown
// work.
// Calling Close on an [OAuthClient] returned by [NewOAuthClient] is safe and
// idempotent.
func (c *oauthClient) Close(_ context.Context) error {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
	return nil
}
