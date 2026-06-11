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

// statusCodeOf extracts the HTTP status code from either an [*APIError] or an
// [*OAuthTokenError], following the error chain via [errors.As].
// Returns (0, false) when neither type is in the chain.
func statusCodeOf(err error) (int, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode, true
	}
	var oauthErr *OAuthTokenError
	if errors.As(err, &oauthErr) {
		return oauthErr.StatusCode, true
	}
	return 0, false
}

// IsNotFound returns true if the error is a 404 Not Found.
func IsNotFound(err error) bool {
	code, ok := statusCodeOf(err)
	return ok && code == http.StatusNotFound
}

// IsUnauthorized returns true if the error is a 401 Unauthorized.
func IsUnauthorized(err error) bool {
	code, ok := statusCodeOf(err)
	return ok && code == http.StatusUnauthorized
}

// IsForbidden returns true if the error is a 403 Forbidden.
func IsForbidden(err error) bool {
	code, ok := statusCodeOf(err)
	return ok && code == http.StatusForbidden
}

// IsRateLimited returns true if the error is a 429 Too Many Requests.
func IsRateLimited(err error) bool {
	code, ok := statusCodeOf(err)
	return ok && code == http.StatusTooManyRequests
}

// IsServerError returns true if the error is a 5xx server error.
func IsServerError(err error) bool {
	code, ok := statusCodeOf(err)
	return ok && code >= 500 && code < 600
}

// IsBadRequest returns true if the error is a 400 Bad Request.
func IsBadRequest(err error) bool {
	code, ok := statusCodeOf(err)
	return ok && code == http.StatusBadRequest
}

// IsConflict returns true if the error is a 409 Conflict.
func IsConflict(err error) bool {
	code, ok := statusCodeOf(err)
	return ok && code == http.StatusConflict
}

// OAuthTokenError represents an OAuth 2.0 token endpoint error response
// (RFC 6749 section 5.2).
// The token endpoint returns a structured error body with an error code,
// optional description, and optional URI, rather than the generic JSON error
// format used by other API endpoints.
//
// Security note: the Description and URI fields are returned verbatim by the
// authorization server and may carry attacker-influenced content if the IdP
// itself is compromised or operates a relay.
// Surfaces that auto-render the URI (CLI hyperlinks, log scrapers that follow
// links) should treat it as untrusted; surfaces that log [OAuthTokenError.Error]
// should escape control characters in the description before display.
type OAuthTokenError struct {
	// StatusCode is the HTTP status code (typically 400).
	StatusCode int

	// Code is the OAuth 2.0 error code (e.g., "invalid_grant",
	// "invalid_request").
	Code string

	// Description is the optional human-readable error description, supplied
	// verbatim by the authorization server.
	// See the type-level security note.
	Description string

	// URI is the optional URI identifying a human-readable web page with
	// error information, supplied verbatim by the authorization server.
	// See the type-level security note.
	URI string
}

// Error implements the error interface.
// The format is "dash0 oauth error: <code>[: <description>] (status: <code>)[ (see: <uri>)]",
// preserving the RFC 6749 §5.2 error_uri field when the IdP provides one.
func (e *OAuthTokenError) Error() string {
	var msg string
	if e.Description != "" {
		msg = fmt.Sprintf("dash0 oauth error: %s: %s (status: %d)",
			e.Code, e.Description, e.StatusCode)
	} else {
		msg = fmt.Sprintf("dash0 oauth error: %s (status: %d)",
			e.Code, e.StatusCode)
	}
	if e.URI != "" {
		msg += fmt.Sprintf(" (see: %s)", e.URI)
	}
	return msg
}

// IsOAuthTokenError returns true if the error is an OAuthTokenError.
func IsOAuthTokenError(err error) bool {
	var oauthErr *OAuthTokenError
	return errors.As(err, &oauthErr)
}

// IsOAuthInvalidGrant returns true if the error is an [*OAuthTokenError] whose
// Code is "invalid_grant".
// This is the OAuth 2.0 signal that the refresh token is no longer accepted by
// the authorization server (rotated, revoked, expired, or the user's session
// was terminated) and the caller must initiate a fresh interactive login.
// See RFC 6749 §5.2.
//
// Note: IsOAuthInvalidGrant is the narrow predicate for the single
// "invalid_grant" code. Callers that want the broader "should I prompt for
// re-authentication?" check should use [IsOAuthTerminalError], which also
// covers "invalid_client" and "unauthorized_client" — codes the library
// itself treats as terminal for the stored credential.
func IsOAuthInvalidGrant(err error) bool {
	var oauthErr *OAuthTokenError
	if !errors.As(err, &oauthErr) {
		return false
	}
	return oauthErr.Code == "invalid_grant"
}

// IsOAuthTerminalError reports whether err carries an [*OAuthTokenError]
// whose Code identifies a terminal credential rejection.
// A terminal rejection means the stored credential is no longer accepted by
// the authorization server and the caller must initiate a fresh interactive
// login; no amount of retry, refresh, or back-off will recover the session.
// The current terminal set per RFC 6749 §5.2 is "invalid_grant",
// "invalid_client", and "unauthorized_client".
//
// This is the predicate the [github.com/dash0hq/dash0-api-client-go/profiles]
// package uses to decide when to clear stored OAuth state from disk; callers
// driving a login UI should align with the same set so the UI prompt matches
// the library's recovery model.
func IsOAuthTerminalError(err error) bool {
	var oauthErr *OAuthTokenError
	if !errors.As(err, &oauthErr) {
		return false
	}
	switch oauthErr.Code {
	case "invalid_grant", "invalid_client", "unauthorized_client":
		return true
	}
	return false
}
