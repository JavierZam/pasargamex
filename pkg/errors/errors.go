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

func New(code string, message string, status int, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
		Err:     err,
	}
}

func NotFound(resource string, err error) *AppError {
	return &AppError{
		Code:    "NOT_FOUND",
		Message: fmt.Sprintf("%s not found", resource),
		Status:  http.StatusNotFound,
		Err:     err,
	}
}

func BadRequest(message string, err error) *AppError {
	return &AppError{
		Code:    "BAD_REQUEST",
		Message: message,
		Status:  http.StatusBadRequest,
		Err:     err,
	}
}

func Unauthorized(message string, err error) *AppError {
	return &AppError{
		Code:    "UNAUTHORIZED",
		Message: message,
		Status:  http.StatusUnauthorized,
		Err:     err,
	}
}

func Internal(message string, err error) *AppError {
	return &AppError{
		Code:    "INTERNAL_ERROR",
		Message: message,
		Status:  http.StatusInternalServerError,
		Err:     err,
	}
}

func Is(err error, code string) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

func Forbidden(message string, err error) *AppError {
	return &AppError{
		Code:    "FORBIDDEN",
		Message: message,
		Status:  http.StatusForbidden,
		Err:     err,
	}
}

func Conflict(message string) *AppError {
	return &AppError{
		Code:    "CONFLICT",
		Message: message,
		Status:  http.StatusConflict,
		Err:     nil,
	}
}

func InternalServer(message string, err error) *AppError {
	return &AppError{
		Code:    "INTERNAL_SERVER_ERROR",
		Message: message,
		Status:  http.StatusInternalServerError,
		Err:     err,
	}
}

func TooManyRequests(message string, waitTime interface{}) *AppError {
	return &AppError{
		Code:    "TOO_MANY_REQUESTS",
		Message: message,
		Status:  http.StatusTooManyRequests,
		Err:     nil,
	}
}
