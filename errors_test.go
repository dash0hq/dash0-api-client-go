package dash0

import (
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
			name: "with status and trace ID",
			err: &APIError{
				StatusCode: 500,
				Status:     "500 Internal Server Error",
				TraceID:    "def456",
			},
			expected: "dash0 api error: 500 Internal Server Error (trace_id: def456)",
		},
		{
			name: "with status only",
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
