package errors

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new application error
func New(code string, message string, status int, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
		Err:     err,
	}
}

// NotFound creates a new not found error
func NotFound(resource string, err error) *AppError {
	return &AppError{
		Code:    "NOT_FOUND",
		Message: fmt.Sprintf("%s not found", resource),
		Status:  http.StatusNotFound,
		Err:     err,
	}
}

// BadRequest creates a new bad request error
func BadRequest(message string, err error) *AppError {
	return &AppError{
		Code:    "BAD_REQUEST",
		Message: message,
		Status:  http.StatusBadRequest,
		Err:     err,
	}
}

// Unauthorized creates a new unauthorized error
func Unauthorized(message string, err error) *AppError {
	return &AppError{
		Code:    "UNAUTHORIZED",
		Message: message,
		Status:  http.StatusUnauthorized,
		Err:     err,
	}
}

// Internal creates a new internal server error
func Internal(message string, err error) *AppError {
	return &AppError{
		Code:    "INTERNAL_ERROR",
		Message: message,
		Status:  http.StatusInternalServerError,
		Err:     err,
	}
}

// Is check if the error is of type AppError and matches the given code
func Is(err error, code string) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// Forbidden 
func Forbidden(message string, err error) *AppError {
    return &AppError{
        Code:    "FORBIDDEN",
        Message: message,
        Status:  http.StatusForbidden, // 403
        Err:     err,
    }
}