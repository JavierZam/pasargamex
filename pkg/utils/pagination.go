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

// DefaultPage is the default page number
const DefaultPage = 1

// DefaultPageSize is the default number of items per page
const DefaultPageSize = 20

// MaxPageSize is the maximum allowed page size
const MaxPageSize = 100

// GetPaginationParams extracts pagination parameters from request
func GetPaginationParams(c echo.Context) PaginationParams {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("limit"))

	return NewPaginationParams(page, pageSize)
}

// NewPaginationParams creates a new pagination parameter set with validation
func NewPaginationParams(page, pageSize int) PaginationParams {
	if page <= 0 {
		page = DefaultPage
	}

	if pageSize <= 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	offset := (page - 1) * pageSize

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Offset:   offset,
	}
}

// ToMap converts pagination parameters to a map for query filters
func (p PaginationParams) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"limit":  p.PageSize,
		"offset": p.Offset,
	}
}
