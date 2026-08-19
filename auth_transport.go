package dash0

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// authTransport sets the Authorization header on every outbound request from
// an [AuthTokenProvider].
//
// It is the outermost layer of the transport stack, above retry and rate
// limiting, because authentication is a property of the request rather than of
// the transport mechanics below it.
// Signing here rather than in a request editor means the OTLP path and the
// generated REST path share one implementation, and it means the replay after
// a 401 re-enters the full stack instead of bypassing rate limiting.
type authTransport struct {
	base     http.RoundTripper
	provider AuthTokenProvider

	// signedHosts are the hostnames the bearer token may be sent to, in
	// canonical lowercase form without a port.
	//
	// Signing inside RoundTrip loses a protection the standard library applies
	// on the way in: [http.Client] drops Authorization when a redirect crosses
	// to a host that is not the original or a subdomain of it, but it can only
	// drop headers that were already on the [http.Request]. A header added here
	// is invisible to it, and the Client re-enters the Transport for every
	// redirect hop, so without this list a 3xx from the configured host to any
	// other origin would hand that origin the token.
	signedHosts []string
}

// newAuthTransport wraps base so that every request it carries is authenticated
// with a token obtained from provider.
//
// endpoints are the URLs the client is configured to talk to. Only requests
// whose host matches one of them, or is a subdomain of one, are signed.
//
// An endpoint whose host cannot be determined is an error rather than a reason
// to skip the check. Deriving no host at all would leave the transport with
// nothing to compare against, and the safe reading of that is "sign nothing",
// which turns every request into a 401. Refusing to build the client says so
// directly instead.
func newAuthTransport(
	base http.RoundTripper,
	provider AuthTokenProvider,
	endpoints ...string,
) (*authTransport, error) {
	if base == nil {
		base = http.DefaultTransport
	}
	t := &authTransport{base: base, provider: provider}
	for _, endpoint := range endpoints {
		if endpoint == "" {
			continue
		}
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("dash0: cannot parse endpoint %q to scope the auth token to its host: %w", endpoint, err)
		}
		if parsed.Hostname() == "" {
			return nil, fmt.Errorf(
				"dash0: endpoint %q has no host, so the auth token cannot be scoped to it; "+
					"include a scheme, for example https://api.eu-west-1.aws.dash0.com", endpoint)
		}
		t.signedHosts = append(t.signedHosts, strings.ToLower(parsed.Hostname()))
	}
	if len(t.signedHosts) == 0 {
		return nil, fmt.Errorf("dash0: no endpoint host to scope the auth token to")
	}
	return t, nil
}

// maySign reports whether the bearer token may be sent to the request's host.
//
// The rule mirrors the standard library's isDomainOrSubdomain: an exact match or
// a subdomain of a configured host. Anything else is unsigned, including a
// request with no host at all, so a redirect off the configured origin cannot
// carry the credential.
func (t *authTransport) maySign(target *url.URL) bool {
	host := strings.ToLower(target.Hostname())
	if host == "" {
		return false
	}
	for _, allowed := range t.signedHosts {
		if host == allowed {
			return true
		}
		// An IPv6 literal or zone is never a subdomain of anything.
		if strings.ContainsAny(host, ":%") {
			continue
		}
		if strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	return false
}

// RoundTrip implements [http.RoundTripper].
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	// A redirect has taken this request off the configured host. Forward it
	// unsigned rather than leaking the token to another origin.
	if !t.maySign(req.URL) {
		return t.base.RoundTrip(req)
	}

	authToken, err := t.provider.AuthToken(ctx)
	if err != nil {
		closeRequestBody(req)
		return nil, fmt.Errorf("dash0: failed to obtain an auth token: %w", err)
	}
	if err := validateAuthToken(authToken); err != nil {
		closeRequestBody(req)
		return nil, err
	}

	// The caller's body is forwarded as-is, so the layer below closes it and
	// this transport upholds the RoundTripper contract.
	return t.base.RoundTrip(signed(req, authToken))
}

// signed returns a copy of req carrying the given bearer token.
//
// It clones rather than mutating, because an [http.RoundTripper] must not
// modify the request it is given, and because the replay path needs the
// original request intact.
// The clone shares Body and GetBody with the original, so the layers below can
// still retry; callers that need a second copy of the body replace Body
// themselves.
func signed(req *http.Request, authToken string) *http.Request {
	outbound := req.Clone(req.Context())
	outbound.Header.Set("Authorization", "Bearer "+authToken)
	return outbound
}

// closeRequestBody closes a request body that will never be sent.
// [http.RoundTripper] implementations own the body they are handed, including
// on the paths that fail before reaching the wire.
func closeRequestBody(req *http.Request) {
	if req.Body != nil {
		_ = req.Body.Close()
	}
}
