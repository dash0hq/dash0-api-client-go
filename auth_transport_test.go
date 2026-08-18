package dash0

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

// scriptedTokenProvider hands out a different token on each AuthToken call, so
// a test can tell which request carried which token.
type scriptedTokenProvider struct {
	tokens []string
	calls  atomic.Int32
}

func (p *scriptedTokenProvider) AuthToken(context.Context) (string, error) {
	i := int(p.calls.Add(1)) - 1
	if i >= len(p.tokens) {
		i = len(p.tokens) - 1
	}
	return p.tokens[i], nil
}

// staticOnlyProvider implements AuthTokenProvider but deliberately not
// RefreshingAuthTokenProvider.
type staticOnlyProvider struct{ token string }

func (p staticOnlyProvider) AuthToken(context.Context) (string, error) { return p.token, nil }

// failingProvider always fails to produce a token.
type failingProvider struct{ err error }

func (p failingProvider) AuthToken(context.Context) (string, error) { return "", p.err }

// recordingRoundTripper records the Authorization header of every request and
// replies with statuses taken from a script.
type recordingRoundTripper struct {
	statuses    []int
	authHeaders []string
	bodies      []string
}

func (rt *recordingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.authHeaders = append(rt.authHeaders, req.Header.Get("Authorization"))
	body := ""
	if req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		body = string(raw)
	}
	rt.bodies = append(rt.bodies, body)

	status := http.StatusOK
	if i := len(rt.authHeaders) - 1; i < len(rt.statuses) {
		status = rt.statuses[i]
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

// newTestRequest builds a request against the canonical test host. The method
// matters: authTransport only replays requests that are safe to send twice.
func newTestRequest(t *testing.T, method, body string) *http.Request {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	}
	req, err := http.NewRequestWithContext(
		context.Background(),
		method,
		"https://api.example.com/api/dashboards",
		reader,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return req
}

func TestAuthTransport(t *testing.T) {
	t.Run("sets the Authorization header from the provider", func(t *testing.T) {
		base := &recordingRoundTripper{}
		transport := newTestAuthTransport(t, base, StaticAuthTokenProvider("auth_static"))

		resp, err := transport.RoundTrip(newTestRequest(t, http.MethodGet, ""))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		drain(resp)

		if len(base.authHeaders) != 1 {
			t.Fatalf("base saw %d requests, want 1", len(base.authHeaders))
		}
		assertEqual(t, "Authorization", base.authHeaders[0], "Bearer auth_static")
	})

	t.Run("asks the provider for a token on every request", func(t *testing.T) {
		// This is the regression test for the reported bug: a client built once
		// and used for a long paginated operation must not keep re-sending the
		// token it started with.
		base := &recordingRoundTripper{}
		provider := &scriptedTokenProvider{tokens: []string{"dash0_at_first", "dash0_at_second", "dash0_at_third"}}
		transport := newTestAuthTransport(t, base, provider)

		for i := range 3 {
			resp, err := transport.RoundTrip(newTestRequest(t, http.MethodGet, ""))
			if err != nil {
				t.Fatalf("request %d: unexpected error: %v", i, err)
			}
			drain(resp)
		}

		want := []string{"Bearer dash0_at_first", "Bearer dash0_at_second", "Bearer dash0_at_third"}
		if len(base.authHeaders) != len(want) {
			t.Fatalf("base saw %d requests, want %d", len(base.authHeaders), len(want))
		}
		for i := range want {
			assertEqual(t, fmt.Sprintf("Authorization on request %d", i), base.authHeaders[i], want[i])
		}
	})

	t.Run("does not mutate the caller's request", func(t *testing.T) {
		base := &recordingRoundTripper{}
		transport := newTestAuthTransport(t, base, StaticAuthTokenProvider("auth_static"))
		req := newTestRequest(t, http.MethodGet, "")

		resp, err := transport.RoundTrip(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		drain(resp)

		if got := req.Header.Get("Authorization"); got != "" {
			t.Errorf("caller's request carries Authorization = %q, want it left unset", got)
		}
	})

	t.Run("propagates a provider failure without sending a request", func(t *testing.T) {
		sentinel := errors.New("no credential available")
		base := &recordingRoundTripper{}
		transport := newTestAuthTransport(t, base, failingProvider{err: sentinel})

		_, err := transport.RoundTrip(newTestRequest(t, http.MethodGet, ""))
		if !errors.Is(err, sentinel) {
			t.Fatalf("error = %v, want it to match the provider's error", err)
		}
		if len(base.authHeaders) != 0 {
			t.Errorf("base saw %d requests, want 0", len(base.authHeaders))
		}
	})

	t.Run("rejects a malformed provider token without sending a request", func(t *testing.T) {
		base := &recordingRoundTripper{}
		transport := newTestAuthTransport(t, base, staticOnlyProvider{token: "not-a-dash0-token"})

		_, err := transport.RoundTrip(newTestRequest(t, http.MethodGet, ""))
		if err == nil {
			t.Fatal("expected an error for a malformed token")
		}
		if !strings.Contains(err.Error(), "must start with") {
			t.Errorf("error = %q, want it to name the prefix requirement", err)
		}
		if len(base.authHeaders) != 0 {
			t.Errorf("base saw %d requests, want 0", len(base.authHeaders))
		}
	})
}

func TestClientAuthTokenProviderEndToEnd(t *testing.T) {
	t.Run("REST requests carry a freshly provided token each time", func(t *testing.T) {
		var authHeaders []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeaders = append(authHeaders, r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]any{})
		}))
		defer server.Close()

		provider := &scriptedTokenProvider{tokens: []string{"dash0_at_first", "dash0_at_second"}}
		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthTokenProvider(provider),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = client.Close(context.Background()) }()

		for i := range 2 {
			if _, err := client.ListDashboards(context.Background(), nil); err != nil {
				t.Fatalf("call %d: unexpected error: %v", i, err)
			}
		}

		if len(authHeaders) != 2 {
			t.Fatalf("server saw %d requests, want 2", len(authHeaders))
		}
		assertEqual(t, "Authorization on the first call", authHeaders[0], "Bearer dash0_at_first")
		assertEqual(t, "Authorization on the second call", authHeaders[1], "Bearer dash0_at_second")
	})

	t.Run("OTLP requests carry a freshly provided token each time", func(t *testing.T) {
		var authHeaders []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeaders = append(authHeaders, r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		provider := &scriptedTokenProvider{tokens: []string{"auth_first", "auth_second"}}
		client, err := NewClient(
			WithOtlpEndpoint(OtlpEncodingJson, server.URL),
			WithAuthTokenProvider(provider),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = client.Close(context.Background()) }()

		for i := range 2 {
			if err := client.SendLogs(context.Background(), newTestLogs(), nil); err != nil {
				t.Fatalf("call %d: unexpected error: %v", i, err)
			}
		}

		if len(authHeaders) != 2 {
			t.Fatalf("server saw %d requests, want 2", len(authHeaders))
		}
		assertEqual(t, "Authorization on the first call", authHeaders[0], "Bearer auth_first")
		assertEqual(t, "Authorization on the second call", authHeaders[1], "Bearer auth_second")
	})
}

func TestNewClientAuthTokenValidation(t *testing.T) {
	t.Run("the last of the two auth options applied wins", func(t *testing.T) {
		// ClientOptions() documents that callers may append options to override
		// what it returned, so the two auth options have to be alternatives
		// rather than a conflict.
		for _, tc := range []struct {
			name string
			opts []ClientOption
			want string
		}{
			{
				name: "provider then token",
				opts: []ClientOption{
					WithAuthTokenProvider(StaticAuthTokenProvider("dash0_at_provider")),
					WithAuthToken("auth_token_wins"),
				},
				want: "Bearer auth_token_wins",
			},
			{
				name: "token then provider",
				opts: []ClientOption{
					WithAuthToken("auth_token"),
					WithAuthTokenProvider(StaticAuthTokenProvider("dash0_at_provider_wins")),
				},
				want: "Bearer dash0_at_provider_wins",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var got string
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					got = r.Header.Get("Authorization")
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode([]any{})
				}))
				defer server.Close()

				client, err := NewClient(append([]ClientOption{WithApiUrl(server.URL)}, tc.opts...)...)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				defer func() { _ = client.Close(context.Background()) }()

				if _, err := client.ListDashboards(context.Background(), nil); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				assertEqual(t, "Authorization", got, tc.want)
			})
		}
	})

	t.Run("rejects neither option being set", func(t *testing.T) {
		_, err := NewClient(WithApiUrl("https://api.example.com"))
		if err == nil {
			t.Fatal("expected an error when no credential is configured")
		}
		if !strings.Contains(err.Error(), "WithAuthTokenProvider") {
			t.Errorf("error = %q, want it to name WithAuthTokenProvider as an option", err)
		}
	})

	t.Run("accepts WithAuthTokenProvider alone", func(t *testing.T) {
		client, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthTokenProvider(StaticAuthTokenProvider("dash0_at_abc")),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = client.Close(context.Background()) }()
	})

	t.Run("NewOAuthClient rejects WithAuthTokenProvider", func(t *testing.T) {
		_, err := NewOAuthClient(
			WithApiUrl("https://api.example.com"),
			WithAuthTokenProvider(StaticAuthTokenProvider("auth_static")),
		)
		if err == nil {
			t.Fatal("expected an error: the OAuth client is unauthenticated by design")
		}
		if !strings.Contains(err.Error(), "WithAuthTokenProvider") {
			t.Errorf("error = %q, want it to name the rejected option", err)
		}
	})
}

// closeTrackingBody reports whether it was closed, so a test can assert that
// authTransport upholds the RoundTripper contract of always closing the body.
type closeTrackingBody struct {
	io.Reader
	closed atomic.Bool
}

func (b *closeTrackingBody) Close() error {
	b.closed.Store(true)
	return nil
}

func TestAuthTransportClosesRequestBody(t *testing.T) {
	t.Run("closes the body when the provider fails", func(t *testing.T) {
		body := &closeTrackingBody{Reader: strings.NewReader(`{"name":"test"}`)}
		req := newTestRequest(t, http.MethodPut, `{"name":"test"}`)
		req.Body = body

		transport := newTestAuthTransport(t, &recordingRoundTripper{}, failingProvider{err: errors.New("boom")})
		if _, err := transport.RoundTrip(req); err == nil {
			t.Fatal("expected an error")
		}

		if !body.closed.Load() {
			t.Error("request body was not closed")
		}
	})

	t.Run("closes the body when the provider token is malformed", func(t *testing.T) {
		body := &closeTrackingBody{Reader: strings.NewReader(`{"name":"test"}`)}
		req := newTestRequest(t, http.MethodPut, `{"name":"test"}`)
		req.Body = body

		transport := newTestAuthTransport(t, &recordingRoundTripper{}, staticOnlyProvider{token: "bogus"})
		if _, err := transport.RoundTrip(req); err == nil {
			t.Fatal("expected an error")
		}

		if !body.closed.Load() {
			t.Error("request body was not closed")
		}
	})

	t.Run("forwards the caller's body on the first attempt", func(t *testing.T) {
		// The layer below owns closing it, which is only true when the first
		// attempt carries the original body rather than a rewound copy.
		base := &recordingRoundTripper{}
		transport := newTestAuthTransport(t, base, StaticAuthTokenProvider("auth_static"))
		req := newTestRequest(t, http.MethodPut, `{"name":"test"}`)
		original := req.Body

		resp, err := transport.RoundTrip(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		drain(resp)

		if len(base.bodies) != 1 {
			t.Fatalf("base saw %d requests, want 1", len(base.bodies))
		}
		assertEqual(t, "body", base.bodies[0], `{"name":"test"}`)
		if req.Body != original {
			t.Error("the caller's request body was replaced")
		}
	})
}

// TestAuthTransportDoesNotSignOtherHosts covers the protection that moving the
// signing out of a request editor and into RoundTrip would otherwise lose:
// http.Client strips Authorization across a cross-origin redirect, but only for
// headers it can see on the Request.
func TestAuthTransportDoesNotSignOtherHosts(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		target   string
		wantAuth bool
	}{
		{name: "exact host", endpoint: "https://api.example.com", target: "https://api.example.com/api/x", wantAuth: true},
		{name: "subdomain of the configured host", endpoint: "https://example.com", target: "https://api.example.com/api/x", wantAuth: true},
		{name: "host differing in case", endpoint: "https://API.example.com", target: "https://api.example.com/api/x", wantAuth: true},
		{name: "unrelated host", endpoint: "https://api.example.com", target: "https://evil.test/api/x", wantAuth: false},
		{name: "parent of the configured host", endpoint: "https://api.example.com", target: "https://example.com/api/x", wantAuth: false},
		{name: "suffix but not a subdomain", endpoint: "https://example.com", target: "https://notexample.com/api/x", wantAuth: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			base := &recordingRoundTripper{}
			transport := newTestAuthTransport(t, base, StaticAuthTokenProvider("auth_secret"), tc.endpoint)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, tc.target, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			drain(resp)

			if len(base.authHeaders) != 1 {
				t.Fatalf("base saw %d requests, want 1", len(base.authHeaders))
			}
			gotAuth := base.authHeaders[0] != ""
			if gotAuth != tc.wantAuth {
				t.Errorf("Authorization sent = %v (%q), want %v", gotAuth, base.authHeaders[0], tc.wantAuth)
			}
		})
	}
}

// TestAuthTransportRedirectDoesNotLeakToken is the end-to-end version: a real
// http.Client following a 3xx off the configured host must not carry the token.
//
// The two servers need distinct hostnames, so a dialer maps both names onto the
// test listeners. Two httptest servers would otherwise share 127.0.0.1 and
// differ only by port, which the rule deliberately ignores: the standard
// library's isDomainOrSubdomain compares hostnames and not ports, and this
// mirrors it.
func TestAuthTransportRedirectDoesNotLeakToken(t *testing.T) {
	var elsewhereAuth string
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		elsewhereAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]any{})
	}))
	defer elsewhere.Close()

	var configuredAuth string
	configured := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		configuredAuth = r.Header.Get("Authorization")
		http.Redirect(w, r, "http://evil.test/api/dashboards", http.StatusFound)
	}))
	defer configured.Close()

	hosts := map[string]string{
		"configured.test:80": strings.TrimPrefix(configured.URL, "http://"),
		"evil.test:80":       strings.TrimPrefix(elsewhere.URL, "http://"),
	}
	dialer := &net.Dialer{}
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if mapped, ok := hosts[addr]; ok {
					addr = mapped
				}
				return dialer.DialContext(ctx, network, addr)
			},
		},
	}

	client, err := NewClient(
		WithApiUrl("http://configured.test"),
		WithAuthToken("auth_secret"),
		WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	if _, err := client.ListDashboards(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Authorization at the configured host", configuredAuth, "Bearer auth_secret")
	if elsewhereAuth != "" {
		t.Errorf("the redirect target received %q, want no Authorization header", elsewhereAuth)
	}
}

// drain closes a response body the test is done with, so the connection is not
// left dangling.
func drain(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// newTestAuthTransport builds an authTransport scoped to the host the test
// requests use, and fails the test if construction is refused.
func newTestAuthTransport(t *testing.T, base http.RoundTripper, provider AuthTokenProvider, endpoints ...string) *authTransport {
	t.Helper()
	if len(endpoints) == 0 {
		endpoints = []string{"https://api.example.com"}
	}
	transport, err := newAuthTransport(base, provider, endpoints...)
	if err != nil {
		t.Fatalf("unexpected error building the transport: %v", err)
	}
	return transport
}

// TestNewAuthTransportRefusesUnscopeableEndpoints pins the fail-closed
// behaviour: without a host to compare against there is no safe default, so
// construction is refused rather than signing everything or nothing.
func TestNewAuthTransportRefusesUnscopeableEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name      string
		endpoints []string
		wantErr   string
	}{
		{name: "no endpoints at all", endpoints: nil, wantErr: "no endpoint host"},
		{name: "only empty endpoints", endpoints: []string{"", ""}, wantErr: "no endpoint host"},
		{name: "missing scheme", endpoints: []string{"api.example.com"}, wantErr: "has no host"},
		{name: "path only", endpoints: []string{"/api"}, wantErr: "has no host"},
		{name: "unparseable", endpoints: []string{"http://[::1"}, wantErr: "cannot parse endpoint"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newAuthTransport(&recordingRoundTripper{}, StaticAuthTokenProvider("auth_x"), tc.endpoints...)
			if err == nil {
				t.Fatal("expected construction to be refused")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

// TestNewClientRejectsApiUrlWithoutAHost covers the same refusal through the
// public constructor, which is where a caller would actually hit it.
func TestNewClientRejectsApiUrlWithoutAHost(t *testing.T) {
	_, err := NewClient(WithApiUrl("api.example.com"), WithAuthToken("auth_x"))
	if err == nil {
		t.Fatal("expected an error for an API URL with no host")
	}
	if !strings.Contains(err.Error(), "has no host") {
		t.Errorf("error = %q, want it to name the missing host", err)
	}
}

// TestAuthTransportDoesNotSignAHostlessRequest is the last line of defence: a
// request whose URL carries no host is never signed.
func TestAuthTransportDoesNotSignAHostlessRequest(t *testing.T) {
	base := &recordingRoundTripper{}
	transport := newTestAuthTransport(t, base, StaticAuthTokenProvider("auth_secret"))

	req := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: "/api/dashboards"},
		Header: make(http.Header),
	}
	resp, err := transport.RoundTrip(req.WithContext(context.Background()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	drain(resp)

	if len(base.authHeaders) != 1 {
		t.Fatalf("base saw %d requests, want 1", len(base.authHeaders))
	}
	if base.authHeaders[0] != "" {
		t.Errorf("a hostless request was signed with %q", base.authHeaders[0])
	}
}
