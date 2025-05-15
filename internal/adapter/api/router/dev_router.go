package router

import (
	"pasargamex/internal/adapter/api/handler"

	"github.com/labstack/echo/v4"
)

// SetupDevRouter initializes development-only routes
func SetupDevRouter(e *echo.Echo, environment string) {
	// Hanya aktifkan di environment development
	if environment != "development" {
		return
	}
	
	devTokenHandler := handler.GetDevTokenHandler()
	
	// Dev token endpoints
	e.GET("/_dev/token/user", devTokenHandler.GenerateUserToken)
	e.GET("/_dev/token/admin", devTokenHandler.GenerateAdminToken)
	e.POST("/_dev/long-lived-token", devTokenHandler.GetLongLivedToken)
}