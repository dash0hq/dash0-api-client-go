package dash0

import (
	"encoding/json"
	"errors"
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
	detail := e.Message
	if detail == "" {
		detail = e.Body
	}
	if detail != "" {
		if e.TraceID != "" {
			return fmt.Sprintf("dash0 api error: %s (status: %d, trace_id: %s)",
				detail, e.StatusCode, e.TraceID)
		}
		return fmt.Sprintf("dash0 api error: %s (status: %d)", detail, e.StatusCode)
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

	if len(body) > 0 {
		// The Dash0 API returns errors as { "error": { "code": int, "message": string, "traceId": string } }
		// (see common.ErrorResponse in dash0hq/openapi-types). Parse that shape first.
		// "code" is intentionally omitted: StatusCode already carries the HTTP status.
		var nested struct {
			Error struct {
				Message string `json:"message"`
				TraceID string `json:"traceId"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &nested) == nil {
			// Always backfill the trace ID from the body when the header didn't provide one,
			// even if the message is empty — a trace ID is more valuable than nothing for
			// the support-debugging case that motivated this code path.
			if apiErr.TraceID == "" {
				apiErr.TraceID = nested.Error.TraceID
			}
			if nested.Error.Message != "" {
				apiErr.Message = nested.Error.Message
				return apiErr
			}
		}

		// Fallback for non-standard shapes: top-level "message" or "error" string.
		var flat struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		if json.Unmarshal(body, &flat) == nil {
			if flat.Message != "" {
				apiErr.Message = flat.Message
			} else if flat.Error != "" {
				apiErr.Message = flat.Error
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

// OAuthTokenError represents an OAuth 2.0 token endpoint error response
// (RFC 6749 section 5.2).
// The token endpoint returns a structured error body with an error code,
// optional description, and optional URI, rather than the generic JSON error
// format used by other API endpoints.
type OAuthTokenError struct {
	// StatusCode is the HTTP status code (typically 400).
	StatusCode int

	// Code is the OAuth 2.0 error code (e.g., "invalid_grant",
	// "invalid_request").
	Code string

	// Description is the optional human-readable error description.
	Description string

	// URI is the optional URI identifying a human-readable web page with
	// error information.
	URI string
}

// Error implements the error interface.
func (e *OAuthTokenError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("dash0 oauth error: %s: %s (status: %d)",
			e.Code, e.Description, e.StatusCode)
	}
	return fmt.Sprintf("dash0 oauth error: %s (status: %d)",
		e.Code, e.StatusCode)
}

// IsOAuthTokenError returns true if the error is an OAuthTokenError.
func IsOAuthTokenError(err error) bool {
	var oauthErr *OAuthTokenError
	return errors.As(err, &oauthErr)
}
