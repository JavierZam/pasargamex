package router

import (
	"pasargamex/internal/adapter/api/handler"

	"github.com/labstack/echo/v4"
)

func SetupDevRouter(e *echo.Echo, environment string) {
	if environment != "development" {
		return
	}
	devTokenHandler := handler.GetDevTokenHandler()

	e.GET("/_dev/token/user", devTokenHandler.GenerateUserToken)
	e.GET("/_dev/token/admin", devTokenHandler.GenerateAdminToken)
	e.POST("/_dev/long-lived-token", devTokenHandler.GetLongLivedToken)
	e.POST("/_dev/test-refresh", devTokenHandler.TestRefreshToken)
}
