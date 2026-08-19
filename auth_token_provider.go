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

// RefreshingAuthTokenProvider is an [AuthTokenProvider] that can discard its
// current token and mint a replacement on demand.
//
// The client calls [RefreshingAuthTokenProvider.ForceRefreshAuthToken] after a
// request comes back 401, which recovers from a token the server stopped
// accepting earlier than its stated expiry — clock skew between the client and
// the authorization server, or a revocation performed elsewhere.
// Providers that hold an immutable token (see [StaticAuthTokenProvider]) do not
// implement this interface, and the client then surfaces the 401 unchanged.
type RefreshingAuthTokenProvider interface {
	AuthTokenProvider

	// ForceRefreshAuthToken returns a token to replace staleAuthToken, which the
	// server has just rejected.
	//
	// staleAuthToken lets an implementation tell "nobody has replaced this yet,
	// mint a new one" from "a concurrent caller already replaced it, hand that
	// result back". Without it, a single revocation 401s every request in
	// flight and each one mints again; for an OAuth provider that means one
	// refresh-token rotation per in-flight request, and every extra rotation is
	// another chance to land on invalid_grant and force an interactive login.
	//
	// Returning staleAuthToken unchanged is allowed and signals that no fresher
	// credential is available.
	ForceRefreshAuthToken(ctx context.Context, staleAuthToken string) (string, error)
}

// StaticAuthTokenProvider adapts a fixed auth token to [AuthTokenProvider].
// It exists so the request path has a single code path for authentication
// regardless of whether the caller supplied a static token or a renewing one.
//
// The returned provider deliberately does not implement
// [RefreshingAuthTokenProvider]: a static token cannot be refreshed, so a 401
// is surfaced to the caller instead of being retried.
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
