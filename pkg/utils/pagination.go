package utils

import (
	"strconv"

	"github.com/labstack/echo/v4"
)

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int
	PageSize int
	Offset   int
}

// GetPaginationParams extracts pagination parameters from request
func GetPaginationParams(c echo.Context) PaginationParams {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("limit"))

	if page <= 0 {
		page = 1
	}

	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20 // Default page size
	}

	offset := (page - 1) * pageSize

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Offset:   offset,
	}
}