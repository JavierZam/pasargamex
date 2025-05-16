package router

import (
	"pasargamex/internal/adapter/api/handler"

	"github.com/labstack/echo/v4"
)

func SetupHealthRouter(e *echo.Echo) {
	healthHandler := handler.GetHealthHandler()
	e.GET("/health", healthHandler.CheckHealth)
	e.GET("/firebase-health", healthHandler.CheckFirebaseHealth)
}
