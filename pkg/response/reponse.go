package response

import (
	"errors"
	"net/http"
	"strings"
	"time"

	apperrors "pasargamex/pkg/errors"
	"github.com/go-playground/validator/v10"

	"github.com/labstack/echo/v4"
)

type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}

type MetaInfo struct {
	RequestID string `json:"request_id,omitempty"`
	Version   string `json:"version,omitempty"`
}

type CustomError struct {
	Code    string
	Message string
	Status  int
}

func Success(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func Created(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func Paginated(c echo.Context, items interface{}, total int64, page, pageSize int) error {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return c.JSON(http.StatusOK, Response{
		Success:   true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data: PaginatedResponse{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

// New method: SuccessPaginated for chat handler compatibility
func SuccessPaginated(c echo.Context, items interface{}, total int64, limit, offset int) error {
	// Calculate page and pageSize from limit and offset
	page := (offset / limit) + 1
	if offset == 0 {
		page = 1
	}
	pageSize := limit

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return c.JSON(http.StatusOK, Response{
		Success:   true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data: PaginatedResponse{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

func Error(c echo.Context, err error) error {
	// Handle validation errors
	var validationErr validator.ValidationErrors
	if errors.As(err, &validationErr) {
		return handleValidationError(c, validationErr)
	}

	// Handle application errors
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return c.JSON(appErr.Status, Response{
			Success:   false,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Error: &ErrorInfo{
				Code:    appErr.Code,
				Message: appErr.Message,
			},
		})
	}

	return c.JSON(http.StatusInternalServerError, Response{
		Success:   false,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Error: &ErrorInfo{
			Code:    "INTERNAL_ERROR",
			Message: "An unexpected error occurred",
		},
	})
}

func handleValidationError(c echo.Context, validationErr validator.ValidationErrors) error {
	for _, err := range validationErr {
		field := strings.ToLower(err.Field())
		tag := err.Tag()
		param := err.Param()

		var message string
		switch tag {
		case "required":
			message = field + " is required"
		case "min":
			if field == "amount" {
				if param == "10000" {
					message = "Minimum amount is 10,000 IDR"
				} else {
					message = field + " must be at least " + param
				}
			} else {
				message = field + " must be at least " + param
			}
		case "max":
			if field == "amount" {
				if param == "50000000" {
					message = "Maximum withdrawal amount is 50,000,000 IDR"
				} else if param == "100000000" {
					message = "Maximum topup amount is 100,000,000 IDR"
				} else {
					message = field + " must be at most " + param
				}
			} else {
				message = field + " must be at most " + param
			}
		case "oneof":
			message = field + " must be one of: " + param
		case "email":
			message = field + " must be a valid email address"
		default:
			message = field + " is invalid"
		}

		return c.JSON(http.StatusBadRequest, Response{
			Success:   false,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Error: &ErrorInfo{
				Code:    "VALIDATION_ERROR",
				Message: message,
			},
		})
	}

	return c.JSON(http.StatusBadRequest, Response{
		Success:   false,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Error: &ErrorInfo{
			Code:    "VALIDATION_ERROR",
			Message: "Invalid input data",
		},
	})
}

func (e *CustomError) Error() string {
	return e.Message
}

func NewError(code, message string, status int) error {
	return &CustomError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}
