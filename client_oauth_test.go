package dash0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestNewOAuthClient(t *testing.T) {
	t.Run("requires API URL", func(t *testing.T) {
		_, err := NewOAuthClient()
		if err == nil {
			t.Fatal("expected error for missing API URL")
		}
	})

	t.Run("succeeds with valid API URL", func(t *testing.T) {
		client, err := NewOAuthClient(WithApiUrl("https://api.example.com"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("accepts custom timeout", func(t *testing.T) {
		client, err := NewOAuthClient(
			WithApiUrl("https://api.example.com"),
			WithTimeout(5*time.Second),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("accepts custom user agent", func(t *testing.T) {
		client, err := NewOAuthClient(
			WithApiUrl("https://api.example.com"),
			WithUserAgent("test-agent/1.0"),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("ignores auth token", func(t *testing.T) {
		client, err := NewOAuthClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_ignored"),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
	})

	t.Run("accepts custom HTTP client", func(t *testing.T) {
		customClient := &http.Client{
			Timeout: 42 * time.Second,
		}
		client, err := NewOAuthClient(
			WithApiUrl("https://api.example.com"),
			WithHTTPClient(customClient),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
	})
}

func TestOAuthClient_NoAuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("unexpected Authorization header: %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(OAuthAuthorizationServerMetadata{
			Issuer:                 "https://auth.example.com",
			AuthorizationEndpoint:  "https://auth.example.com/oauth/authorize",
			TokenEndpoint:          "https://auth.example.com/oauth/token",
			ResponseTypesSupported: []OAuthResponseType{Code},
		})
	}))
	defer server.Close()

	client, err := NewOAuthClient(WithApiUrl(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.GetAuthorizationServerMetadata(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOAuthClient_UserAgentHeader(t *testing.T) {
	t.Run("sends default user agent", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ua := r.Header.Get("User-Agent")
			if ua != DefaultUserAgent {
				t.Errorf("User-Agent = %q, want %q", ua, DefaultUserAgent)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(OAuthAuthorizationServerMetadata{
				Issuer:                 "https://auth.example.com",
				AuthorizationEndpoint:  "https://auth.example.com/oauth/authorize",
				TokenEndpoint:          "https://auth.example.com/oauth/token",
				ResponseTypesSupported: []OAuthResponseType{Code},
			})
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.GetAuthorizationServerMetadata(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("sends custom user agent", func(t *testing.T) {
		customUA := "my-cli/2.0"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ua := r.Header.Get("User-Agent")
			if ua != customUA {
				t.Errorf("User-Agent = %q, want %q", ua, customUA)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(OAuthAuthorizationServerMetadata{
				Issuer:                 "https://auth.example.com",
				AuthorizationEndpoint:  "https://auth.example.com/oauth/authorize",
				TokenEndpoint:          "https://auth.example.com/oauth/token",
				ResponseTypesSupported: []OAuthResponseType{Code},
			})
		}))
		defer server.Close()

		client, err := NewOAuthClient(
			WithApiUrl(server.URL),
			WithUserAgent(customUA),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.GetAuthorizationServerMetadata(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestOAuthClient_GetAuthorizationServerMetadata(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := OAuthAuthorizationServerMetadata{
			Issuer:                 "https://auth.example.com",
			AuthorizationEndpoint:  "https://auth.example.com/oauth/authorize",
			TokenEndpoint:          "https://auth.example.com/oauth/token",
			ResponseTypesSupported: []OAuthResponseType{Code},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/.well-known/oauth-authorization-server" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method: %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expected)
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		metadata, err := client.GetAuthorizationServerMetadata(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if metadata.Issuer != expected.Issuer {
			t.Errorf("Issuer = %q, want %q", metadata.Issuer, expected.Issuer)
		}
		if metadata.TokenEndpoint != expected.TokenEndpoint {
			t.Errorf("TokenEndpoint = %q, want %q", metadata.TokenEndpoint, expected.TokenEndpoint)
		}
	})

	t.Run("nil body on 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.GetAuthorizationServerMetadata(context.Background())
		if err == nil {
			t.Fatal("expected error for nil response body")
		}
	})

	t.Run("error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "internal error"})
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.GetAuthorizationServerMetadata(context.Background())
		if err == nil {
			t.Fatal("expected error")
		}
		if !IsServerError(err) {
			t.Errorf("expected server error, got: %v", err)
		}
	})
}

func TestOAuthClient_GetProtectedResourceMetadata(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := OAuthProtectedResourceMetadata{
			Resource:             "https://api.example.com",
			AuthorizationServers: []string{"https://auth.example.com"},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/.well-known/oauth-protected-resource" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method: %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expected)
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		metadata, err := client.GetProtectedResourceMetadata(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if metadata.Resource != expected.Resource {
			t.Errorf("Resource = %q, want %q", metadata.Resource, expected.Resource)
		}
	})

	t.Run("nil body on 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.GetProtectedResourceMetadata(context.Background())
		if err == nil {
			t.Fatal("expected error for nil response body")
		}
	})

	t.Run("error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.GetProtectedResourceMetadata(context.Background())
		if err == nil {
			t.Fatal("expected error")
		}
		if !IsNotFound(err) {
			t.Errorf("expected not found error, got: %v", err)
		}
	})
}

func TestOAuthClient_AuthorizeURL(t *testing.T) {
	client, err := NewOAuthClient(WithApiUrl("https://api.example.com"))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	t.Run("builds URL with all parameters", func(t *testing.T) {
		scope := "openid profile"
		state := "random-state"
		prompt := "consent"
		result, err := client.AuthorizeURL(&AuthorizeURLParams{
			ResponseType:        "code",
			ClientID:            "my-client-id",
			RedirectURI:         "http://localhost:8080/callback",
			Scope:               &scope,
			State:               &state,
			CodeChallenge:       "abc123challenge",
			CodeChallengeMethod: "S256",
			Prompt:              &prompt,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parsed, err := url.Parse(result)
		if err != nil {
			t.Fatalf("failed to parse URL: %v", err)
		}

		if parsed.Scheme != "https" {
			t.Errorf("scheme = %q, want %q", parsed.Scheme, "https")
		}
		if parsed.Host != "api.example.com" {
			t.Errorf("host = %q, want %q", parsed.Host, "api.example.com")
		}
		if parsed.Path != "/oauth/authorize" {
			t.Errorf("path = %q, want %q", parsed.Path, "/oauth/authorize")
		}

		query := parsed.Query()
		if got := query.Get("response_type"); got != "code" {
			t.Errorf("response_type = %q, want %q", got, "code")
		}
		if got := query.Get("client_id"); got != "my-client-id" {
			t.Errorf("client_id = %q, want %q", got, "my-client-id")
		}
		if got := query.Get("redirect_uri"); got != "http://localhost:8080/callback" {
			t.Errorf("redirect_uri = %q, want %q", got, "http://localhost:8080/callback")
		}
		if got := query.Get("scope"); got != "openid profile" {
			t.Errorf("scope = %q, want %q", got, "openid profile")
		}
		if got := query.Get("state"); got != "random-state" {
			t.Errorf("state = %q, want %q", got, "random-state")
		}
		if got := query.Get("code_challenge"); got != "abc123challenge" {
			t.Errorf("code_challenge = %q, want %q", got, "abc123challenge")
		}
		if got := query.Get("code_challenge_method"); got != "S256" {
			t.Errorf("code_challenge_method = %q, want %q", got, "S256")
		}
		if got := query.Get("prompt"); got != "consent" {
			t.Errorf("prompt = %q, want %q", got, "consent")
		}
	})

	t.Run("builds URL with required parameters only", func(t *testing.T) {
		result, err := client.AuthorizeURL(&AuthorizeURLParams{
			ResponseType:        "code",
			ClientID:            "client-1",
			RedirectURI:         "http://localhost/cb",
			CodeChallenge:       "challenge",
			CodeChallengeMethod: "S256",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parsed, err := url.Parse(result)
		if err != nil {
			t.Fatalf("failed to parse URL: %v", err)
		}

		query := parsed.Query()
		if got := query.Get("scope"); got != "" {
			t.Errorf("scope should be absent, got %q", got)
		}
		if got := query.Get("state"); got != "" {
			t.Errorf("state should be absent, got %q", got)
		}
		if got := query.Get("prompt"); got != "" {
			t.Errorf("prompt should be absent, got %q", got)
		}
	})
}

func TestOAuthClient_RegisterClient(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/oauth/register" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("unexpected method: %s", r.Method)
			}
			if ct := r.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			var req OAuthClientRegistrationRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}
			if req.ClientName != "My CLI" {
				t.Errorf("ClientName = %q, want %q", req.ClientName, "My CLI")
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(OAuthClientRegistrationResponse{
				ClientId:                "generated-client-id",
				ClientName:              req.ClientName,
				RedirectUris:            req.RedirectUris,
				RegistrationAccessToken: "reg-token",
			})
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := client.RegisterClient(context.Background(), &OAuthClientRegistrationRequest{
			ClientName:   "My CLI",
			RedirectUris: []string{"http://localhost:8080/callback"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ClientId != "generated-client-id" {
			t.Errorf("ClientId = %q, want %q", resp.ClientId, "generated-client-id")
		}
		if resp.RegistrationAccessToken != "reg-token" {
			t.Errorf("RegistrationAccessToken = %q, want %q", resp.RegistrationAccessToken, "reg-token")
		}
	})

	t.Run("nil body on 201", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.RegisterClient(context.Background(), &OAuthClientRegistrationRequest{
			ClientName:   "Test",
			RedirectUris: []string{"http://localhost/cb"},
		})
		if err == nil {
			t.Fatal("expected error for nil response body")
		}
	})

	t.Run("error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid redirect_uri"})
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.RegisterClient(context.Background(), &OAuthClientRegistrationRequest{
			ClientName:   "Bad Client",
			RedirectUris: []string{"not-a-url"},
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !IsBadRequest(err) {
			t.Errorf("expected bad request error, got: %v", err)
		}
	})
}

func TestOAuthClient_ExchangeToken(t *testing.T) {
	t.Run("success with authorization code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/oauth/token" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("unexpected method: %s", r.Method)
			}

			if err := r.ParseForm(); err != nil {
				t.Fatalf("failed to parse form: %v", err)
			}
			if got := r.PostFormValue("grant_type"); got != "authorization_code" {
				t.Errorf("grant_type = %q, want %q", got, "authorization_code")
			}
			if got := r.PostFormValue("code"); got != "auth-code-xyz" {
				t.Errorf("code = %q, want %q", got, "auth-code-xyz")
			}
			if got := r.PostFormValue("redirect_uri"); got != "http://localhost:8080/callback" {
				t.Errorf("redirect_uri = %q, want %q", got, "http://localhost:8080/callback")
			}
			if got := r.PostFormValue("code_verifier"); got != "verifier" {
				t.Errorf("code_verifier = %q, want %q", got, "verifier")
			}
			if got := r.PostFormValue("client_id"); got != "client-1" {
				t.Errorf("client_id = %q, want %q", got, "client-1")
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(OAuthTokenResponse{
				AccessToken: "access-token-123",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			})
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := client.ExchangeToken(context.Background(), &OAuthTokenRequest{
			GrantType:    OAuthGrantTypeAuthorizationCode,
			Code:         Ptr("auth-code-xyz"),
			RedirectUri:  Ptr("http://localhost:8080/callback"),
			CodeVerifier: Ptr("verifier"),
			ClientId:     Ptr("client-1"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.AccessToken != "access-token-123" {
			t.Errorf("AccessToken = %q, want %q", resp.AccessToken, "access-token-123")
		}
		if resp.TokenType != "Bearer" {
			t.Errorf("TokenType = %q, want %q", resp.TokenType, "Bearer")
		}
		if resp.ExpiresIn != 3600 {
			t.Errorf("ExpiresIn = %d, want %d", resp.ExpiresIn, 3600)
		}
	})

	t.Run("success with refresh token", func(t *testing.T) {
		datasets := []string{"default"}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseForm(); err != nil {
				t.Fatalf("failed to parse form: %v", err)
			}
			if got := r.PostFormValue("grant_type"); got != "refresh_token" {
				t.Errorf("grant_type = %q, want %q", got, "refresh_token")
			}
			if got := r.PostFormValue("refresh_token"); got != "dash0_rt_old" {
				t.Errorf("refresh_token = %q, want %q", got, "dash0_rt_old")
			}
			if got := r.PostFormValue("client_id"); got != "client-1" {
				t.Errorf("client_id = %q, want %q", got, "client-1")
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			restriction := DatasetRestriction("restricted")
			_ = json.NewEncoder(w).Encode(OAuthTokenResponse{
				AccessToken:        "dash0_at_new",
				TokenType:          "Bearer",
				ExpiresIn:          900,
				RefreshToken:       Ptr("dash0_rt_new"),
				Scope:              Ptr("*"),
				DatasetRestriction: &restriction,
				Datasets:           &datasets,
			})
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := client.ExchangeToken(context.Background(), &OAuthTokenRequest{
			GrantType:    OAuthGrantTypeRefreshToken,
			RefreshToken: Ptr("dash0_rt_old"),
			ClientId:     Ptr("client-1"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.AccessToken != "dash0_at_new" {
			t.Errorf("AccessToken = %q, want %q", resp.AccessToken, "dash0_at_new")
		}
		if resp.RefreshToken == nil || *resp.RefreshToken != "dash0_rt_new" {
			t.Errorf("RefreshToken = %v, want %q", resp.RefreshToken, "dash0_rt_new")
		}
		if resp.DatasetRestriction == nil || string(*resp.DatasetRestriction) != "restricted" {
			t.Errorf("DatasetRestriction = %v, want %q", resp.DatasetRestriction, "restricted")
		}
		if resp.Datasets == nil || len(*resp.Datasets) != 1 || (*resp.Datasets)[0] != "default" {
			t.Errorf("Datasets = %v, want [default]", resp.Datasets)
		}
	})

	t.Run("nil body on 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ExchangeToken(context.Background(), &OAuthTokenRequest{
			GrantType:    OAuthGrantTypeAuthorizationCode,
			Code:         Ptr("code"),
			RedirectUri:  Ptr("http://localhost/cb"),
			CodeVerifier: Ptr("verifier"),
			ClientId:     Ptr("client-1"),
		})
		if err == nil {
			t.Fatal("expected error for nil response body")
		}
	})

	t.Run("bad request returns OAuthTokenError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(OAuthTokenErrorResponse{
				Error:            "invalid_grant",
				ErrorDescription: Ptr("authorization code has expired"),
				ErrorUri:         Ptr("https://docs.example.com/errors/invalid_grant"),
			})
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ExchangeToken(context.Background(), &OAuthTokenRequest{
			GrantType:    OAuthGrantTypeAuthorizationCode,
			Code:         Ptr("expired-code"),
			RedirectUri:  Ptr("http://localhost:8080/callback"),
			CodeVerifier: Ptr("verifier"),
			ClientId:     Ptr("client-1"),
		})
		if err == nil {
			t.Fatal("expected error")
		}

		if !IsOAuthTokenError(err) {
			t.Fatalf("expected OAuthTokenError, got %T: %v", err, err)
		}

		oauthErr, ok := err.(*OAuthTokenError)
		if !ok {
			t.Fatalf("expected *OAuthTokenError, got %T", err)
		}
		if oauthErr.StatusCode != 400 {
			t.Errorf("StatusCode = %d, want 400", oauthErr.StatusCode)
		}
		if oauthErr.Code != "invalid_grant" {
			t.Errorf("Code = %q, want %q", oauthErr.Code, "invalid_grant")
		}
		if oauthErr.Description != "authorization code has expired" {
			t.Errorf("Description = %q, want %q", oauthErr.Description, "authorization code has expired")
		}
		if oauthErr.URI != "https://docs.example.com/errors/invalid_grant" {
			t.Errorf("URI = %q, want %q", oauthErr.URI, "https://docs.example.com/errors/invalid_grant")
		}
	})

	t.Run("bad request without structured body returns APIError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad request"))
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ExchangeToken(context.Background(), &OAuthTokenRequest{
			GrantType: OAuthGrantTypeAuthorizationCode,
			Code:      Ptr("some-code"),
			ClientId:  Ptr("client-1"),
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if IsOAuthTokenError(err) {
			t.Error("expected APIError, not OAuthTokenError")
		}
		if !IsBadRequest(err) {
			t.Errorf("expected bad request error, got: %v", err)
		}
	})

	t.Run("server error returns APIError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "internal error"})
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ExchangeToken(context.Background(), &OAuthTokenRequest{
			GrantType:    OAuthGrantTypeAuthorizationCode,
			Code:         Ptr("some-code"),
			RedirectUri:  Ptr("http://localhost:8080/callback"),
			CodeVerifier: Ptr("verifier"),
			ClientId:     Ptr("client-1"),
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !IsServerError(err) {
			t.Errorf("expected server error, got: %v", err)
		}
	})
}

func TestOAuthClient_RevokeToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		hint := OAuthTokenTypeAccessToken
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/oauth/revoke" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("unexpected method: %s", r.Method)
			}

			if err := r.ParseForm(); err != nil {
				t.Fatalf("failed to parse form: %v", err)
			}
			if got := r.PostFormValue("token"); got != "token-to-revoke" {
				t.Errorf("token = %q, want %q", got, "token-to-revoke")
			}
			if got := r.PostFormValue("token_type_hint"); got != "access_token" {
				t.Errorf("token_type_hint = %q, want %q", got, "access_token")
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = client.RevokeToken(context.Background(), &OAuthRevocationRequest{
			Token:         "token-to-revoke",
			TokenTypeHint: &hint,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "internal error"})
		}))
		defer server.Close()

		client, err := NewOAuthClient(WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = client.RevokeToken(context.Background(), &OAuthRevocationRequest{
			Token: "token-to-revoke",
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !IsServerError(err) {
			t.Errorf("expected server error, got: %v", err)
		}
	})
}

func TestOAuthClient_Close(t *testing.T) {
	client, err := NewOAuthClient(WithApiUrl("https://api.example.com"))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.Close(context.Background())
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
