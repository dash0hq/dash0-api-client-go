package dash0

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError represents an error response from the Dash0 API.
type APIError struct {
	// StatusCode is the HTTP status code.
	StatusCode int

	// Status is the HTTP status text.
	Status string

	// Body is the raw response body.
	Body string

	// Message is the error message extracted from the response.
	Message string

	// TraceID is the trace ID from the x-trace-id header if available.
	TraceID string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Message != "" {
		if e.TraceID != "" {
			return fmt.Sprintf("dash0 api error: %s (status: %d, trace_id: %s)",
				e.Message, e.StatusCode, e.TraceID)
		}
		return fmt.Sprintf("dash0 api error: %s (status: %d)", e.Message, e.StatusCode)
	}
	if e.TraceID != "" {
		return fmt.Sprintf("dash0 api error: %s (trace_id: %s)", e.Status, e.TraceID)
	}
	return fmt.Sprintf("dash0 api error: %s", e.Status)
}

// NewAPIError creates an APIError from an HTTP response.
// Note: This function tries to read the response body. If the body has already
// been read (e.g., by oapi-codegen), use newAPIErrorWithBody instead.
func NewAPIError(resp *http.Response) *APIError {
	var body []byte
	if resp.Body != nil {
		body, _ = io.ReadAll(resp.Body)
	}
	return newAPIErrorWithBody(resp, body)
}

// newAPIErrorWithBody creates an APIError from an HTTP response and pre-read body bytes.
// This is used internally when the response body has already been consumed.
func newAPIErrorWithBody(resp *http.Response, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		TraceID:    resp.Header.Get("x-trace-id"),
		Body:       string(body),
	}

	// Try to extract message from JSON error response
	if len(body) > 0 {
		var errResp struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil {
			if errResp.Message != "" {
				apiErr.Message = errResp.Message
			} else if errResp.Error != "" {
				apiErr.Message = errResp.Error
			}
		}
	}

	return apiErr
}

// IsNotFound returns true if the error is a 404 Not Found.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// IsUnauthorized returns true if the error is a 401 Unauthorized.
func IsUnauthorized(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

// IsForbidden returns true if the error is a 403 Forbidden.
func IsForbidden(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusForbidden
	}
	return false
}

// IsRateLimited returns true if the error is a 429 Too Many Requests.
func IsRateLimited(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusTooManyRequests
	}
	return false
}

// IsServerError returns true if the error is a 5xx server error.
func IsServerError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode >= 500 && apiErr.StatusCode < 600
	}
	return false
}

// IsBadRequest returns true if the error is a 400 Bad Request.
func IsBadRequest(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusBadRequest
	}
	return false
}

// IsConflict returns true if the error is a 409 Conflict.
func IsConflict(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusConflict
	}
	return false
}
