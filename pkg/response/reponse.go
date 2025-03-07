package response

import (
	"errors"
	"net/http"

	apperrors "pasargamex/pkg/errors"

	"github.com/labstack/echo/v4"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}

// Success returns a success response with data
func Success(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// Created returns a success response for resource creation
func Created(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// Paginated returns a paginated response
func Paginated(c echo.Context, items interface{}, total int64, page, pageSize int) error {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: PaginatedResponse{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

// Error returns an error response
func Error(c echo.Context, err error) error {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return c.JSON(appErr.Status, Response{
			Success: false,
			Error: &ErrorInfo{
				Code:    appErr.Code,
				Message: appErr.Message,
			},
		})
	}

	// Default to internal server error
	return c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    "INTERNAL_ERROR",
			Message: "An unexpected error occurred",
		},
	})
}