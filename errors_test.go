package dash0

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected string
	}{
		{
			name: "with message and trace ID",
			err: &APIError{
				StatusCode: 400,
				Status:     "400 Bad Request",
				Message:    "invalid request",
				TraceID:    "abc123",
			},
			expected: "dash0 api error: invalid request (status: 400, trace_id: abc123)",
		},
		{
			name: "with message only",
			err: &APIError{
				StatusCode: 400,
				Status:     "400 Bad Request",
				Message:    "invalid request",
			},
			expected: "dash0 api error: invalid request (status: 400)",
		},
		{
			name: "with body and trace ID when message is empty",
			err: &APIError{
				StatusCode: 400,
				Status:     "400 Bad Request",
				Body:       `{"details": "field is required"}`,
				TraceID:    "def456",
			},
			expected: `dash0 api error: {"details": "field is required"} (status: 400, trace_id: def456)`,
		},
		{
			name: "with body only when message is empty",
			err: &APIError{
				StatusCode: 400,
				Status:     "400 Bad Request",
				Body:       `{"details": "field is required"}`,
			},
			expected: `dash0 api error: {"details": "field is required"} (status: 400)`,
		},
		{
			name: "with status and trace ID when both message and body are empty",
			err: &APIError{
				StatusCode: 500,
				Status:     "500 Internal Server Error",
				TraceID:    "def456",
			},
			expected: "dash0 api error: 500 Internal Server Error (trace_id: def456)",
		},
		{
			name: "with status only when both message and body are empty",
			err: &APIError{
				StatusCode: 500,
				Status:     "500 Internal Server Error",
			},
			expected: "dash0 api error: 500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewAPIError(t *testing.T) {
	t.Run("extracts trace ID from header", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 400,
			Status:     "400 Bad Request",
			Header:     http.Header{"X-Trace-Id": []string{"trace-123"}},
			Body:       io.NopCloser(strings.NewReader("")),
		}

		apiErr := NewAPIError(resp)

		if apiErr.TraceID != "trace-123" {
			t.Errorf("TraceID = %q, want %q", apiErr.TraceID, "trace-123")
		}
	})

	t.Run("extracts message from JSON body", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 400,
			Status:     "400 Bad Request",
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(`{"message": "validation failed"}`)),
		}

		apiErr := NewAPIError(resp)

		if apiErr.Message != "validation failed" {
			t.Errorf("Message = %q, want %q", apiErr.Message, "validation failed")
		}
	})

	t.Run("extracts nested message and traceId from real Dash0 API error shape", func(t *testing.T) {
		// Matches the wire format produced by control-plane-api's utils.NewErrorResponse
		// (see common.ErrorResponse in dash0hq/openapi-types):
		//   {"error": {"code": 404, "message": "Check rule not found", "traceId": "abc123"}}
		resp := &http.Response{
			StatusCode: 404,
			Status:     "404 Not Found",
			Header:     http.Header{},
			Body: io.NopCloser(strings.NewReader(
				`{"error": {"code": 404, "message": "Check rule not found", "traceId": "abc123"}}`,
			)),
		}

		apiErr := NewAPIError(resp)

		if apiErr.Message != "Check rule not found" {
			t.Errorf("Message = %q, want %q", apiErr.Message, "Check rule not found")
		}
		if apiErr.TraceID != "abc123" {
			t.Errorf("TraceID = %q, want %q", apiErr.TraceID, "abc123")
		}
	})

	t.Run("header trace ID takes precedence over body traceId", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 404,
			Status:     "404 Not Found",
			Header:     http.Header{"X-Trace-Id": []string{"header-trace"}},
			Body: io.NopCloser(strings.NewReader(
				`{"error": {"code": 404, "message": "not found", "traceId": "body-trace"}}`,
			)),
		}

		apiErr := NewAPIError(resp)

		if apiErr.TraceID != "header-trace" {
			t.Errorf("TraceID = %q, want %q", apiErr.TraceID, "header-trace")
		}
	})

	t.Run("body traceId is preserved when nested message is empty", func(t *testing.T) {
		// A 5xx response may carry a trace ID without a human-readable message;
		// the trace ID is still the most useful debugging signal we can surface.
		resp := &http.Response{
			StatusCode: 500,
			Status:     "500 Internal Server Error",
			Header:     http.Header{},
			Body: io.NopCloser(strings.NewReader(
				`{"error": {"code": 500, "message": "", "traceId": "trace-xyz"}}`,
			)),
		}

		apiErr := NewAPIError(resp)

		if apiErr.TraceID != "trace-xyz" {
			t.Errorf("TraceID = %q, want %q", apiErr.TraceID, "trace-xyz")
		}
	})

	t.Run("malformed nested error field falls through to flat parser", func(t *testing.T) {
		// {"error": "string"} is the legacy flat shape — the nested unmarshal fails
		// (cannot decode string into struct) and the flat parser must still pick it up.
		resp := &http.Response{
			StatusCode: 400,
			Status:     "400 Bad Request",
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(`{"error": "legacy flat"}`)),
		}

		apiErr := NewAPIError(resp)

		if apiErr.Message != "legacy flat" {
			t.Errorf("Message = %q, want %q", apiErr.Message, "legacy flat")
		}
	})

	t.Run("extracts error from JSON body", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 400,
			Status:     "400 Bad Request",
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(`{"error": "something went wrong"}`)),
		}

		apiErr := NewAPIError(resp)

		if apiErr.Message != "something went wrong" {
			t.Errorf("Message = %q, want %q", apiErr.Message, "something went wrong")
		}
	})

	t.Run("prefers message over error in JSON", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 400,
			Status:     "400 Bad Request",
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(`{"message": "the message", "error": "the error"}`)),
		}

		apiErr := NewAPIError(resp)

		if apiErr.Message != "the message" {
			t.Errorf("Message = %q, want %q", apiErr.Message, "the message")
		}
	})

	t.Run("handles non-JSON body", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 500,
			Status:     "500 Internal Server Error",
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("plain text error")),
		}

		apiErr := NewAPIError(resp)

		if apiErr.Body != "plain text error" {
			t.Errorf("Body = %q, want %q", apiErr.Body, "plain text error")
		}
		if apiErr.Message != "" {
			t.Errorf("Message = %q, want empty", apiErr.Message)
		}
		// Body is included in Error() when Message is empty
		expected := "dash0 api error: plain text error (status: 500)"
		if apiErr.Error() != expected {
			t.Errorf("Error() = %q, want %q", apiErr.Error(), expected)
		}
	})

	t.Run("handles nil body", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 500,
			Status:     "500 Internal Server Error",
			Header:     http.Header{},
			Body:       nil,
		}

		apiErr := NewAPIError(resp)

		if apiErr.StatusCode != 500 {
			t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
		}
	})
}

func TestErrorHelpers(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		check      func(error) bool
		expected   bool
	}{
		{"IsNotFound with 404", 404, IsNotFound, true},
		{"IsNotFound with 200", 200, IsNotFound, false},
		{"IsUnauthorized with 401", 401, IsUnauthorized, true},
		{"IsUnauthorized with 200", 200, IsUnauthorized, false},
		{"IsForbidden with 403", 403, IsForbidden, true},
		{"IsForbidden with 200", 200, IsForbidden, false},
		{"IsRateLimited with 429", 429, IsRateLimited, true},
		{"IsRateLimited with 200", 200, IsRateLimited, false},
		{"IsBadRequest with 400", 400, IsBadRequest, true},
		{"IsBadRequest with 200", 200, IsBadRequest, false},
		{"IsConflict with 409", 409, IsConflict, true},
		{"IsConflict with 200", 200, IsConflict, false},
		{"IsServerError with 500", 500, IsServerError, true},
		{"IsServerError with 502", 502, IsServerError, true},
		{"IsServerError with 599", 599, IsServerError, true},
		{"IsServerError with 400", 400, IsServerError, false},
		{"IsServerError with 600", 600, IsServerError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode}
			got := tt.check(err)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorHelpers_NonAPIError(t *testing.T) {
	err := io.EOF // A non-APIError

	checks := []struct {
		name  string
		check func(error) bool
	}{
		{"IsNotFound", IsNotFound},
		{"IsUnauthorized", IsUnauthorized},
		{"IsForbidden", IsForbidden},
		{"IsRateLimited", IsRateLimited},
		{"IsBadRequest", IsBadRequest},
		{"IsConflict", IsConflict},
		{"IsServerError", IsServerError},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if c.check(err) {
				t.Errorf("%s returned true for non-APIError", c.name)
			}
		})
	}
}

func TestOAuthTokenError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *OAuthTokenError
		expected string
	}{
		{
			name: "with description",
			err: &OAuthTokenError{
				StatusCode:  400,
				Code:        "invalid_grant",
				Description: "authorization code has expired",
			},
			expected: "dash0 oauth error: invalid_grant: authorization code has expired (status: 400)",
		},
		{
			name: "without description",
			err: &OAuthTokenError{
				StatusCode: 400,
				Code:       "invalid_request",
			},
			expected: "dash0 oauth error: invalid_request (status: 400)",
		},
		{
			name: "with description and URI",
			err: &OAuthTokenError{
				StatusCode:  400,
				Code:        "invalid_grant",
				Description: "authorization code has expired",
				URI:         "https://docs.example.com/errors/invalid_grant",
			},
			expected: "dash0 oauth error: invalid_grant: authorization code has expired (status: 400) (see: https://docs.example.com/errors/invalid_grant)",
		},
		{
			name: "with URI but no description",
			err: &OAuthTokenError{
				StatusCode: 400,
				Code:       "invalid_request",
				URI:        "https://docs.example.com/errors/invalid_request",
			},
			expected: "dash0 oauth error: invalid_request (status: 400) (see: https://docs.example.com/errors/invalid_request)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsOAuthTokenError(t *testing.T) {
	t.Run("returns true for OAuthTokenError", func(t *testing.T) {
		err := &OAuthTokenError{StatusCode: 400, Code: "invalid_grant"}
		if !IsOAuthTokenError(err) {
			t.Error("expected true for *OAuthTokenError")
		}
	})

	t.Run("returns true for wrapped OAuthTokenError", func(t *testing.T) {
		inner := &OAuthTokenError{StatusCode: 400, Code: "invalid_grant"}
		err := fmt.Errorf("wrapper: %w", inner)
		if !IsOAuthTokenError(err) {
			t.Error("expected true for wrapped *OAuthTokenError")
		}
	})

	t.Run("returns false for APIError", func(t *testing.T) {
		err := &APIError{StatusCode: 400}
		if IsOAuthTokenError(err) {
			t.Error("expected false for *APIError")
		}
	})

	t.Run("returns false for non-API error", func(t *testing.T) {
		err := errors.New("something else")
		if IsOAuthTokenError(err) {
			t.Error("expected false for plain error")
		}
	})
}

func TestIsOAuthInvalidGrant(t *testing.T) {
	t.Run("returns true for invalid_grant", func(t *testing.T) {
		err := &OAuthTokenError{StatusCode: 400, Code: "invalid_grant"}
		if !IsOAuthInvalidGrant(err) {
			t.Error("expected true for invalid_grant")
		}
	})

	t.Run("returns true for wrapped invalid_grant", func(t *testing.T) {
		inner := &OAuthTokenError{StatusCode: 400, Code: "invalid_grant"}
		err := fmt.Errorf("wrapper: %w", inner)
		if !IsOAuthInvalidGrant(err) {
			t.Error("expected true for wrapped invalid_grant")
		}
	})

	t.Run("returns false for other OAuth error codes", func(t *testing.T) {
		for _, code := range []string{"invalid_request", "invalid_client", "unauthorized_client", "unsupported_grant_type", "invalid_scope"} {
			err := &OAuthTokenError{StatusCode: 400, Code: code}
			if IsOAuthInvalidGrant(err) {
				t.Errorf("expected false for code %q", code)
			}
		}
	})

	t.Run("returns false for APIError", func(t *testing.T) {
		err := &APIError{StatusCode: 400}
		if IsOAuthInvalidGrant(err) {
			t.Error("expected false for *APIError")
		}
	})

	t.Run("returns false for plain error", func(t *testing.T) {
		if IsOAuthInvalidGrant(errors.New("boom")) {
			t.Error("expected false for plain error")
		}
	})
}

// TestStatusCodeHelpers_AcceptOAuthTokenError pins the behaviour added when
// the Is* family was refactored to route through statusCodeOf, which unwraps
// *OAuthTokenError in addition to *APIError. Without this test the contract is
// implicit and a future regression could narrow the helpers back to *APIError
// only.
func TestStatusCodeHelpers_AcceptOAuthTokenError(t *testing.T) {
	cases := []struct {
		status int
		check  func(error) bool
		name   string
	}{
		{http.StatusBadRequest, IsBadRequest, "IsBadRequest"},
		{http.StatusUnauthorized, IsUnauthorized, "IsUnauthorized"},
		{http.StatusForbidden, IsForbidden, "IsForbidden"},
		{http.StatusNotFound, IsNotFound, "IsNotFound"},
		{http.StatusConflict, IsConflict, "IsConflict"},
		{http.StatusTooManyRequests, IsRateLimited, "IsRateLimited"},
		{http.StatusInternalServerError, IsServerError, "IsServerError"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := &OAuthTokenError{StatusCode: tc.status, Code: "test"}
			if !tc.check(err) {
				t.Errorf("%s(*OAuthTokenError{StatusCode: %d}) = false, want true", tc.name, tc.status)
			}
			// Wrapped via fmt.Errorf must still match (statusCodeOf uses errors.As).
			wrapped := fmt.Errorf("wrapped: %w", err)
			if !tc.check(wrapped) {
				t.Errorf("%s(wrapped *OAuthTokenError{StatusCode: %d}) = false, want true", tc.name, tc.status)
			}
		})
	}

	t.Run("mismatched status returns false", func(t *testing.T) {
		err := &OAuthTokenError{StatusCode: http.StatusBadRequest, Code: "test"}
		if IsNotFound(err) {
			t.Error("IsNotFound should be false for a 400-status OAuthTokenError")
		}
	})
}
