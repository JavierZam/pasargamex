package middleware

import (
	"net/http"

	"pasargamex/internal/domain/repository"

	"github.com/labstack/echo/v4"
)

type AdminMiddleware struct {
	userRepo repository.UserRepository
}

func NewAdminMiddleware(userRepo repository.UserRepository) *AdminMiddleware {
	return &AdminMiddleware{
		userRepo: userRepo,
	}
}

func (m *AdminMiddleware) AdminOnly(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {

		uid, ok := c.Get("uid").(string)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required")
		}

		user, err := m.userRepo.GetByID(c.Request().Context(), uid)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify admin privileges")
		}

		if user.Role != "admin" {
			return echo.NewHTTPError(http.StatusForbidden, "Admin privileges required")
		}

		return next(c)
	}
}
