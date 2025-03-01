package handler

import (
	"net/http"
	"time"

	"pasargamex/internal/infrastructure/firebase"

	"github.com/labstack/echo/v4"
)

type HealthHandler struct {
	firebaseAuth *firebase.FirebaseAuthClient
}

var healthHandler *HealthHandler

func NewHealthHandler(firebaseAuth *firebase.FirebaseAuthClient) *HealthHandler {
	return &HealthHandler{
		firebaseAuth: firebaseAuth,
	}
}

func SetupHealthHandler(firebaseAuth *firebase.FirebaseAuthClient) {
	healthHandler = NewHealthHandler(firebaseAuth)
}

func GetHealthHandler() *HealthHandler {
	return healthHandler
}

func (h *HealthHandler) CheckHealth(c echo.Context) error {
    return c.JSON(http.StatusOK, map[string]string{
        "status": "Server is running",
        "time":   time.Now().Format(time.RFC3339),
    })
}

func (h *HealthHandler) CheckFirebaseHealth(c echo.Context) error {
	err := h.firebaseAuth.TestConnection(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"status": "Firebase Auth connection failed",
			"error":  err.Error(),
		})
	}
	
	return c.JSON(http.StatusOK, map[string]string{
		"status": "Firebase Auth connected successfully",
	})
}