package errors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
	Err        error
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
}

func New(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func NewNotFound(entity, id string) *AppError {
	return &AppError{
		Code:       "NOT_FOUND",
		Message:    fmt.Sprintf("%s %s not found", entity, id),
		HTTPStatus: http.StatusNotFound,
		Err:        nil,
	}
}

func NewUnauthorized(message string) *AppError {
	return &AppError{
		Code:       "UNAUTHORIZED",
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
		Err:        nil,
	}
}

func NewBadRequest(message string) *AppError {
	return &AppError{
		Code:       "BAD_REQUEST",
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Err:        nil,
	}
}

func NewInternal(message string, err error) *AppError {
	return &AppError{
		Code:       "INTERNAL",
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

func NewForbidden(message string) *AppError {
	return &AppError{
		Code:       "FORBIDDEN",
		Message:    message,
		HTTPStatus: http.StatusForbidden,
		Err:        nil,
	}
}
