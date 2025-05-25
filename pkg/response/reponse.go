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

type CustomError struct {
	Code    string
	Message string
	Status  int
}

func Success(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

func Created(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

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

	return c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    "INTERNAL_ERROR",
			Message: "An unexpected error occurred",
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
