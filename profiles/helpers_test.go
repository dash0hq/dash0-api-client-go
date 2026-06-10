package profiles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// tokenServerResponse configures the mock token endpoint response.
type tokenServerResponse struct {
	AccessToken  string
	ExpiresIn    int64
	RefreshToken *string
}

// newTokenServer creates an httptest.Server that responds to POST /oauth/token
// with the given token response.
// It also records the parsed form values from the most recent request into
// *lastRequest if non-nil.
func newTokenServer(t *testing.T, resp tokenServerResponse, lastRequest *map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth/token" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if lastRequest != nil {
			if err := r.ParseForm(); err == nil {
				m := make(map[string]string, len(r.PostForm))
				for k, v := range r.PostForm {
					if len(v) > 0 {
						m[k] = v[0]
					}
				}
				*lastRequest = m
			}
		}
		w.Header().Set("Content-Type", "application/json")
		body := map[string]any{
			"access_token": resp.AccessToken,
			"token_type":   "Bearer",
			"expires_in":   resp.ExpiresIn,
		}
		if resp.RefreshToken != nil {
			body["refresh_token"] = *resp.RefreshToken
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
}

// newRevokeServer creates an httptest.Server that responds to POST /oauth/revoke
// with 200 OK.
// It records the parsed form values from the most recent request into
// *lastRequest if non-nil.
func newRevokeServer(t *testing.T, lastRequest *map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth/revoke" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if lastRequest != nil {
			if err := r.ParseForm(); err == nil {
				m := make(map[string]string, len(r.PostForm))
				for k, v := range r.PostForm {
					if len(v) > 0 {
						m[k] = v[0]
					}
				}
				*lastRequest = m
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
}
