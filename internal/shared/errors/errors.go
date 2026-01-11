package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Common error types
var (
	ErrNotFound       = errors.New("resource not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrBadRequest     = errors.New("bad request")
	ErrConflict       = errors.New("conflict")
	ErrInternal       = errors.New("internal error")
	ErrValidation     = errors.New("validation error")
)

// AppError represents an application error with context
type AppError struct {
	Err        error             `json:"-"`
	Message    string            `json:"message"`
	Code       string            `json:"code"`
	HTTPStatus int               `json:"-"`
	Details    map[string]string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// NotFound creates a not found error
func NotFound(resource string, id string) *AppError {
	return &AppError{
		Err:        ErrNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		Code:       "NOT_FOUND",
		HTTPStatus: http.StatusNotFound,
		Details:    map[string]string{"resource": resource, "id": id},
	}
}

// Unauthorized creates an unauthorized error
func Unauthorized(message string) *AppError {
	return &AppError{
		Err:        ErrUnauthorized,
		Message:    message,
		Code:       "UNAUTHORIZED",
		HTTPStatus: http.StatusUnauthorized,
	}
}

// Forbidden creates a forbidden error
func Forbidden(message string) *AppError {
	return &AppError{
		Err:        ErrForbidden,
		Message:    message,
		Code:       "FORBIDDEN",
		HTTPStatus: http.StatusForbidden,
	}
}

// BadRequest creates a bad request error
func BadRequest(message string) *AppError {
	return &AppError{
		Err:        ErrBadRequest,
		Message:    message,
		Code:       "BAD_REQUEST",
		HTTPStatus: http.StatusBadRequest,
	}
}

// Validation creates a validation error with field details
func Validation(message string, details map[string]string) *AppError {
	return &AppError{
		Err:        ErrValidation,
		Message:    message,
		Code:       "VALIDATION_ERROR",
		HTTPStatus: http.StatusBadRequest,
		Details:    details,
	}
}

// Conflict creates a conflict error
func Conflict(message string) *AppError {
	return &AppError{
		Err:        ErrConflict,
		Message:    message,
		Code:       "CONFLICT",
		HTTPStatus: http.StatusConflict,
	}
}

// Internal creates an internal error
func Internal(err error) *AppError {
	return &AppError{
		Err:        err,
		Message:    "internal server error",
		Code:       "INTERNAL_ERROR",
		HTTPStatus: http.StatusInternalServerError,
	}
}

// Wrap wraps an error with additional context
func Wrap(err error, message string) *AppError {
	if appErr, ok := err.(*AppError); ok {
		appErr.Message = fmt.Sprintf("%s: %s", message, appErr.Message)
		return appErr
	}
	return &AppError{
		Err:        err,
		Message:    message,
		Code:       "INTERNAL_ERROR",
		HTTPStatus: http.StatusInternalServerError,
	}
}
