// Package httpresp provides a uniform JSON response envelope for the API.
//
// Every successful response carries {"data": <T>, "error": null}; every
// failure carries {"data": null, "error": {"code": "...", "message": "..."}}.
// This makes the contract trivial to express in OpenAPI and lets clients
// branch on a single field (`error == null`) without inspecting HTTP status.
package httpresp

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/quiqxiq/roskit/pkg/errors"
)

// Stable error codes used across handlers. Keep this list short and stable —
// clients may pattern-match on these values.
const (
	CodeBadRequest      = "bad_request"
	CodeUnauthorized    = "unauthorized"
	CodeForbidden       = "forbidden"
	CodeNotFound        = "not_found"
	CodeConflict        = "conflict"
	CodeRateLimited     = "rate_limited"
	CodeInternal        = "internal"
	CodeInvalidInput    = "invalid_input"
	CodeTenantMismatch  = "tenant_mismatch"
	CodeServiceFailure  = "service_failure"
)

// APIError is the JSON shape of an error inside the response envelope.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Envelope is the universal response shape. Exactly one of Data / Error is
// non-nil for any response. Keep this type stable — it appears verbatim in
// the OpenAPI schema as components.schemas.Envelope.
type Envelope struct {
	Data  any       `json:"data"`
	Error *APIError `json:"error"`
}

// Success writes a 2xx envelope with the given data payload.
func Success(c *gin.Context, status int, data any) {
	c.JSON(status, Envelope{Data: data, Error: nil})
}

// Error writes an error envelope with code+message.
func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, Envelope{Data: nil, Error: &APIError{Code: code, Message: message}})
}

// ErrorWithDetails writes an error envelope with structured details (e.g.
// per-field validation errors).
func ErrorWithDetails(c *gin.Context, status int, code, message string, details any) {
	c.JSON(status, Envelope{Data: nil, Error: &APIError{Code: code, Message: message, Details: details}})
}

// FromAppError translates a *errors.AppError into the matching response.
// Falls back to 500 internal when the error is unrecognised, exposing only
// a generic message to avoid leaking internals.
func FromAppError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		status := appErr.HTTPStatus
		if status == 0 {
			status = http.StatusInternalServerError
		}
		code := appErr.Code
		if code == "" {
			code = CodeInternal
		}
		Error(c, status, code, appErr.Message)
		return
	}
	Error(c, http.StatusInternalServerError, CodeInternal, "internal server error")
}
