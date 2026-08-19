package dash0

import (
	"context"
	"fmt"
	"strings"
)

// AuthTokenProvider supplies the bearer token used to authenticate a Dash0 API
// request.
// The client calls [AuthTokenProvider.AuthToken] once per request rather than
// caching a token for the client's lifetime, so a provider backed by a
// short-lived credential can renew it without the caller rebuilding the client.
//
// Implementations must be safe for concurrent use: the client issues up to
// [MaxConcurrentRequests] requests in parallel, and each one asks for a token.
type AuthTokenProvider interface {
	// AuthToken returns the bearer token to authenticate the next request.
	// The returned token must start with "auth_" or "dash0_at_"; the client
	// rejects anything else before the request leaves the process.
	AuthToken(ctx context.Context) (string, error)
}

// StaticAuthTokenProvider adapts a fixed auth token to [AuthTokenProvider].
// It exists so the request path has a single code path for authentication
// regardless of whether the caller supplied a static token or a renewing one.
func StaticAuthTokenProvider(authToken string) AuthTokenProvider {
	return staticAuthTokenProvider(authToken)
}

// staticAuthTokenProvider is the [StaticAuthTokenProvider] implementation.
// It is a string rather than a struct so it carries no mutable state and is
// trivially safe for concurrent use.
type staticAuthTokenProvider string

func (p staticAuthTokenProvider) AuthToken(context.Context) (string, error) {
	return string(p), nil
}

// Auth token prefixes. Every Dash0 credential carries one of these, and callers
// that validate a token before handing it over should compare against them
// rather than repeating the literals.
const (
	// AuthTokenPrefixStatic marks a long-lived organization token, created in
	// the Dash0 settings UI.
	AuthTokenPrefixStatic = "auth_"
	// AuthTokenPrefixOAuth marks an access token issued by the Dash0
	// authorization server, which expires and has to be refreshed.
	AuthTokenPrefixOAuth = "dash0_at_"
)

// validateAuthToken rejects a token whose shape cannot be a Dash0 credential.
//
// Checking at the point of use turns a would-be 401 from the server into an
// actionable local error, which matters most for provider-supplied tokens: the
// caller never sees those values, so an empty or malformed one is otherwise
// indistinguishable from a genuine authentication failure.
func validateAuthToken(authToken string) error {
	if authToken == "" {
		return fmt.Errorf("dash0: auth token is empty")
	}
	if !strings.HasPrefix(authToken, AuthTokenPrefixStatic) &&
		!strings.HasPrefix(authToken, AuthTokenPrefixOAuth) {
		return fmt.Errorf("dash0: auth token must start with %q or %q",
			AuthTokenPrefixStatic, AuthTokenPrefixOAuth)
	}
	return nil
}
