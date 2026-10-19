package dash0

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

const (
	testAuthTokenSecret = "auth_super_secret_value"
	testAuthTokenAPIVer = "dash0.com/v1alpha1"
)

var testAuthTokenUUID = openapi_types.UUID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

func newTestAuthToken(origin string) *AuthTokenDefinition {
	return &AuthTokenDefinition{
		ApiVersion: testAuthTokenAPIVer,
		Kind:       AuthTokenDefinitionKindDash0AuthToken,
		Metadata: AuthTokenMetadata{
			Name:   "CI token",
			Labels: &AuthTokenLabels{Dash0Comorigin: &origin},
		},
		Spec: AuthTokenSpec{
			Actions:   &[]Action{"ingest:write"},
			Ingestion: &AuthTokenIngestion{DefaultDataset: Ptr("staging")},
		},
	}
}

// newTestAuthTokenResponse builds a response document; token is nil for every response except the one that creates the token.
func newTestAuthTokenResponse(origin string, token *string) *AuthTokenResponseDefinition {
	def := newTestAuthToken(origin)
	return &AuthTokenResponseDefinition{
		ApiVersion: def.ApiVersion,
		Kind:       AuthTokenResponseDefinitionKindDash0AuthToken,
		Metadata:   def.Metadata,
		Spec:       AuthTokenResponseSpec{Actions: def.Spec.Actions, Ingestion: def.Spec.Ingestion, Token: token},
	}
}

func originOf(meta AuthTokenMetadata) string {
	if meta.Labels == nil || meta.Labels.Dash0Comorigin == nil {
		return ""
	}
	return *meta.Labels.Dash0Comorigin
}

func idOf(token *AuthTokenDefinition) string {
	if token.Metadata.Labels == nil || token.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return token.Metadata.Labels.Dash0Comid.String()
}

func writeAuthTokenJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func TestListAuthTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "method", r.Method, http.MethodGet)
		assertEqual(t, "path", r.URL.Path, "/api/auth-tokens")
		assertEqual(t, "authorization", r.Header.Get("Authorization"), "Bearer auth_test123")
		writeAuthTokenJSON(w, http.StatusOK, AuthTokenListResponse{*newTestAuthTokenResponse("a-tok", nil), *newTestAuthTokenResponse("b-tok", nil)})
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL).ListAuthTokens(context.Background())
	if err != nil {
		t.Fatalf("ListAuthTokens() error = %v", err)
	}
	if len(got) != 2 || originOf(got[1].Metadata) != "b-tok" || got[1].Spec.Token != nil {
		t.Errorf("ListAuthTokens() = %+v, want 2 tokens ending in b-tok", got)
	}
}

func TestListAuthTokensIter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAuthTokenJSON(w, http.StatusOK, AuthTokenListResponse{*newTestAuthTokenResponse("a-tok", nil)})
	}))
	defer server.Close()

	it := newTestClient(t, server.URL).ListAuthTokensIter(context.Background())
	if !it.Next() || originOf(it.Current().Metadata) != "a-tok" {
		t.Fatalf("iterator did not yield a-tok, err = %v", it.Err())
	}
	if it.Next() {
		t.Error("iterator yielded more than one token")
	}
}

func TestListAuthTokensIter_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusForbidden, "no permission")
	}))
	defer server.Close()

	it := newTestClient(t, server.URL).ListAuthTokensIter(context.Background())
	if it.Next() {
		t.Error("iterator yielded an item on error")
	}
	if !IsForbidden(it.Err()) {
		t.Errorf("iterator error = %v, want a forbidden error", it.Err())
	}
}

func TestGetAuthToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "method", r.Method, http.MethodGet)
		assertEqual(t, "path", r.URL.Path, "/api/auth-tokens/ci-token")
		assertEqual(t, "authorization", r.Header.Get("Authorization"), "Bearer auth_test123")
		writeAuthTokenJSON(w, http.StatusOK, newTestAuthTokenResponse("ci-token", nil))
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL).GetAuthToken(context.Background(), "ci-token")
	if err != nil {
		t.Fatalf("GetAuthToken() error = %v", err)
	}
	assertEqual(t, "name", GetAuthTokenName(got), "CI token")
	assertEqual(t, "apiVersion", got.ApiVersion, testAuthTokenAPIVer)
	if got.Spec.Token != nil {
		t.Errorf("token = %q, want nil: only the create response carries the secret", *got.Spec.Token)
	}
}

func TestGetAuthToken_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL).GetAuthToken(context.Background(), "ci-token")
	if err == nil || got != nil {
		t.Errorf("GetAuthToken() = %+v, %v, want nil and an error", got, err)
	}
}

func TestCreateAuthToken(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assertEqual(t, "method", r.Method, http.MethodPost)
				assertEqual(t, "path", r.URL.Path, "/api/auth-tokens")
				assertEqual(t, "authorization", r.Header.Get("Authorization"), "Bearer auth_test123")
				var raw map[string]any
				if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				assertEqual(t, "request apiVersion", fmt.Sprint(raw["apiVersion"]), testAuthTokenAPIVer)
				assertEqual(t, "request kind", fmt.Sprint(raw["kind"]), "Dash0AuthToken")
				spec, _ := raw["spec"].(map[string]any)
				if _, ok := spec["token"]; ok {
					t.Errorf("request spec = %v, want no token key", spec)
				}
				meta, _ := raw["metadata"].(map[string]any)
				assertEqual(t, "request name", fmt.Sprint(meta["name"]), "CI token")
				labels, _ := meta["labels"].(map[string]any)
				assertEqual(t, "request origin", fmt.Sprint(labels["dash0.com/origin"]), "ci-token")
				actions, _ := spec["actions"].([]any)
				if len(actions) != 1 || actions[0] != "ingest:write" {
					t.Errorf("request actions = %v, want [ingest:write]", actions)
				}
				writeAuthTokenJSON(w, status, newTestAuthTokenResponse("ci-token", Ptr(testAuthTokenSecret)))
			}))
			defer server.Close()

			got, err := newTestClient(t, server.URL).CreateAuthToken(context.Background(), newTestAuthToken("ci-token"))
			if err != nil {
				t.Fatalf("CreateAuthToken() error = %v", err)
			}
			assertEqual(t, "origin", originOf(got.Metadata), "ci-token")
			assertPtrEqual(t, "token", got.Spec.Token, testAuthTokenSecret)
		})
	}
}

// The token is returned only when the PUT creates the token (201), so a plain update (200) parses with a nil token.
func TestUpdateAuthToken(t *testing.T) {
	tests := []struct {
		status    int
		wantToken *string
	}{
		{http.StatusOK, nil},
		{http.StatusCreated, Ptr(testAuthTokenSecret)},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assertEqual(t, "method", r.Method, http.MethodPut)
				assertEqual(t, "path", r.URL.Path, "/api/auth-tokens/ci-token")
				assertEqual(t, "authorization", r.Header.Get("Authorization"), "Bearer auth_test123")
				var raw map[string]any
				if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				assertEqual(t, "request apiVersion", fmt.Sprint(raw["apiVersion"]), testAuthTokenAPIVer)
				meta, _ := raw["metadata"].(map[string]any)
				assertEqual(t, "request name", fmt.Sprint(meta["name"]), "CI token")
				if spec, _ := raw["spec"].(map[string]any); spec["token"] != nil {
					t.Errorf("request spec = %v, want no token key", spec)
				}
				writeAuthTokenJSON(w, tt.status, newTestAuthTokenResponse("ci-token", tt.wantToken))
			}))
			defer server.Close()

			got, err := newTestClient(t, server.URL).UpdateAuthToken(context.Background(), "ci-token", newTestAuthToken("ci-token"))
			if err != nil {
				t.Fatalf("UpdateAuthToken() error = %v", err)
			}
			assertEqual(t, "origin", originOf(got.Metadata), "ci-token")
			if tt.wantToken == nil {
				if got.Spec.Token != nil {
					t.Errorf("token = %q, want nil", *got.Spec.Token)
				}
			} else {
				assertPtrEqual(t, "token", got.Spec.Token, *tt.wantToken)
			}
		})
	}
}

func TestDeleteAuthToken(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assertEqual(t, "method", r.Method, http.MethodDelete)
				assertEqual(t, "path", r.URL.Path, "/api/auth-tokens/ci-token")
				assertEqual(t, "authorization", r.Header.Get("Authorization"), "Bearer auth_test123")
				w.WriteHeader(status)
			}))
			defer server.Close()

			if err := newTestClient(t, server.URL).DeleteAuthToken(context.Background(), "ci-token"); err != nil {
				t.Fatalf("DeleteAuthToken() error = %v", err)
			}
		})
	}
}

func TestAuthTokens_NilArgument(t *testing.T) {
	c := newTestClient(t, "http://127.0.0.1:1")
	if _, err := c.CreateAuthToken(context.Background(), nil); err == nil {
		t.Error("CreateAuthToken(nil) error = nil, want an error")
	}
	if _, err := c.UpdateAuthToken(context.Background(), "ci-token", nil); err == nil {
		t.Error("UpdateAuthToken(nil) error = nil, want an error")
	}
}

func TestAuthTokens_Errors(t *testing.T) {
	ctx := context.Background()
	ops := map[string]func(Client) error{
		"List": func(c Client) error { _, err := c.ListAuthTokens(ctx); return err },
		"Get":  func(c Client) error { _, err := c.GetAuthToken(ctx, "ci-token"); return err },
		"Create": func(c Client) error {
			_, err := c.CreateAuthToken(ctx, newTestAuthToken("ci-token"))
			return err
		},
		"Update": func(c Client) error {
			_, err := c.UpdateAuthToken(ctx, "ci-token", newTestAuthToken("ci-token"))
			return err
		},
		"Delete": func(c Client) error { return c.DeleteAuthToken(ctx, "ci-token") },
	}
	statuses := []struct {
		status int
		check  func(error) bool
	}{
		{http.StatusBadRequest, IsBadRequest},
		{http.StatusForbidden, IsForbidden},
		{http.StatusNotFound, IsNotFound},
	}
	for name, op := range ops {
		for _, s := range statuses {
			t.Run(fmt.Sprintf("%s/%d", name, s.status), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					writeAPIError(w, s.status, "rejected")
				}))
				defer server.Close()

				err := op(newTestClient(t, server.URL))
				if !s.check(err) {
					t.Fatalf("error = %v, want status %d", err, s.status)
				}
				var apiErr *APIError
				if !errors.As(err, &apiErr) || apiErr.StatusCode != s.status || apiErr.Message != "rejected" {
					t.Errorf("error = %#v, want an *APIError with status %d and message %q", err, s.status, "rejected")
				}
			})
		}
	}
}

func TestDeleteAuthToken_LastToken(t *testing.T) {
	const message = "cannot delete the last auth token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusBadRequest, message)
	}))
	defer server.Close()

	err := newTestClient(t, server.URL).DeleteAuthToken(context.Background(), "ci-token")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("DeleteAuthToken() error = %v, want an *APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest || apiErr.Message != message {
		t.Errorf("APIError = status %d, message %q, want 400 and %q", apiErr.StatusCode, apiErr.Message, message)
	}
}

func TestAuthTokenResponse_Redaction(t *testing.T) {
	def := newTestAuthTokenResponse("ci-token", Ptr(testAuthTokenSecret))
	// fmt prints a pointer field below the top level as an address, so the nested case holds the response by value.
	wrapper := struct{ Def AuthTokenResponseDefinition }{*def}

	values := map[string]any{
		"spec value":   def.Spec,
		"spec pointer": &def.Spec,
		"response":     *def,
		"response ptr": def,
		"nested":       wrapper,
		"list body":    AuthTokenListResponse{*def},
	}
	for name, v := range values {
		for _, verb := range []string{"%v", "%+v", "%#v", "%s"} {
			t.Run(name+" "+verb, func(t *testing.T) {
				got := fmt.Sprintf(verb, v)
				if strings.Contains(got, testAuthTokenSecret) {
					t.Errorf("Sprintf(%q) leaked the token: %s", verb, got)
				}
				if !strings.Contains(got, "[REDACTED]") {
					t.Errorf("Sprintf(%q) = %s, want [REDACTED]", verb, got)
				}
			})
		}
	}
}

func TestAuthTokenResponse_RedactionNilToken(t *testing.T) {
	got := fmt.Sprintf("%+v", newTestAuthTokenResponse("ci-token", nil))
	if strings.Contains(got, "[REDACTED]") {
		t.Errorf("Sprintf() = %s, want no [REDACTED] marker for a nil token", got)
	}
}

func TestAuthTokenResponse_RedactionSlog(t *testing.T) {
	def := newTestAuthTokenResponse("ci-token", Ptr(testAuthTokenSecret))
	handlers := map[string]func(*bytes.Buffer) slog.Handler{
		"json": func(b *bytes.Buffer) slog.Handler { return slog.NewJSONHandler(b, nil) },
		"text": func(b *bytes.Buffer) slog.Handler { return slog.NewTextHandler(b, nil) },
	}
	attrs := map[string]slog.Attr{
		"spec":         slog.Any("spec", def.Spec),
		"spec ptr":     slog.Any("spec", &def.Spec),
		"response":     slog.Any("def", *def),
		"response ptr": slog.Any("def", def),
	}
	for hName, newHandler := range handlers {
		for aName, attr := range attrs {
			t.Run(hName+" "+aName, func(t *testing.T) {
				var buf bytes.Buffer
				slog.New(newHandler(&buf)).Info("auth token", attr)
				if strings.Contains(buf.String(), testAuthTokenSecret) {
					t.Errorf("slog leaked the token: %s", buf.String())
				}
				if !strings.Contains(buf.String(), "REDACTED") {
					t.Errorf("slog output = %s, want REDACTED", buf.String())
				}
			})
		}
	}
}

// Redaction must not leak into the wire format: the secret is still received as is.
func TestAuthTokenResponse_JSONKeepsToken(t *testing.T) {
	data, err := json.Marshal(newTestAuthTokenResponse("ci-token", Ptr(testAuthTokenSecret)))
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(data), testAuthTokenSecret) || strings.Contains(string(data), "REDACTED") {
		t.Errorf("json.Marshal() = %s, want the real token", data)
	}
	var back AuthTokenResponseDefinition
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	assertPtrEqual(t, "round-tripped token", back.Spec.Token, testAuthTokenSecret)
}

func TestStripAuthTokenServerFields(t *testing.T) {
	now := time.Now()
	source := CrdSource("api")
	origin := "ci-token"
	token := newTestAuthToken(origin)
	token.Metadata.Annotations = &AuthTokenAnnotations{Dash0ComcreatedAt: &now}
	token.Metadata.Labels = &AuthTokenLabels{Dash0Comid: &testAuthTokenUUID, Dash0Comorigin: &origin, Dash0Comsource: &source}

	StripAuthTokenServerFields(token)

	if token.Metadata.Annotations != nil || token.Metadata.Labels.Dash0Comid != nil || token.Metadata.Labels.Dash0Comsource != nil {
		t.Errorf("server fields were not stripped: %+v", token.Metadata)
	}
	assertEqual(t, "origin", originOf(token.Metadata), "ci-token")
	assertEqual(t, "name", token.Metadata.Name, "CI token")
	if token.Spec.Actions == nil || token.Spec.Ingestion == nil {
		t.Errorf("spec was stripped, want it kept: %+v", token.Spec)
	}
}

func TestConvertAuthTokenResponseToDefinition(t *testing.T) {
	resp := newTestAuthTokenResponse("ci-token", Ptr(testAuthTokenSecret))
	resp.Spec.Scope = &AuthTokenScope{Signals: &AuthTokenSignalScope{Types: &[]SignalType{"spans"}}}

	got := ConvertAuthTokenResponseToDefinition(resp)

	assertEqual(t, "apiVersion", got.ApiVersion, testAuthTokenAPIVer)
	assertEqual(t, "kind", string(got.Kind), "Dash0AuthToken")
	assertEqual(t, "name", got.Metadata.Name, "CI token")
	assertEqual(t, "origin", originOf(got.Metadata), "ci-token")
	if got.Spec.Actions == nil || (*got.Spec.Actions)[0] != "ingest:write" ||
		got.Spec.Ingestion == nil || *got.Spec.Ingestion.DefaultDataset != "staging" ||
		got.Spec.Scope == nil || (*got.Spec.Scope.Signals.Types)[0] != "spans" {
		t.Errorf("spec not copied: %+v", got.Spec)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(data), testAuthTokenSecret) || strings.Contains(string(data), `"token"`) {
		t.Errorf("converted definition = %s, want no token", data)
	}
}

func TestAuthTokenHelpers(t *testing.T) {
	const other = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	token := &AuthTokenDefinition{Metadata: AuthTokenMetadata{Name: "CI token"}}
	assertEqual(t, "id before set", idOf(token), "")

	SetAuthTokenIDIfAbsent(token, testAuthTokenUUID.String())
	SetAuthTokenIDIfAbsent(token, other)
	assertEqual(t, "id after IfAbsent", idOf(token), testAuthTokenUUID.String())

	SetAuthTokenID(token, other)
	assertEqual(t, "id after Set", idOf(token), other)

	SetAuthTokenID(token, "not-a-uuid")
	assertEqual(t, "id after invalid Set", idOf(token), other)

	ClearAuthTokenID(token)
	assertEqual(t, "id after Clear", idOf(token), "")
}

func TestGetAuthTokenIDAndName(t *testing.T) {
	resp := newTestAuthTokenResponse("ci-token", nil)
	assertEqual(t, "id before set", GetAuthTokenID(resp), "")
	resp.Metadata.Labels.Dash0Comid = &testAuthTokenUUID
	assertEqual(t, "id", GetAuthTokenID(resp), testAuthTokenUUID.String())
	assertEqual(t, "name", GetAuthTokenName(resp), "CI token")
}

func TestAuthTokenHelpers_Nil(t *testing.T) {
	StripAuthTokenServerFields(nil)
	StripAuthTokenServerFields(&AuthTokenDefinition{})
	SetAuthTokenID(nil, testAuthTokenUUID.String())
	SetAuthTokenIDIfAbsent(nil, testAuthTokenUUID.String())
	ClearAuthTokenID(nil)
	ClearAuthTokenID(&AuthTokenDefinition{})
	if got := ConvertAuthTokenResponseToDefinition(nil); got != nil {
		t.Errorf("ConvertAuthTokenResponseToDefinition(nil) = %+v, want nil", got)
	}
	_ = ConvertAuthTokenResponseToDefinition(&AuthTokenResponseDefinition{})
	assertEqual(t, "id", GetAuthTokenID(nil), "")
	assertEqual(t, "id of zero value", GetAuthTokenID(&AuthTokenResponseDefinition{}), "")
	assertEqual(t, "name", GetAuthTokenName(nil), "")
}
