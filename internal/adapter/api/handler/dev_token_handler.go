package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"pasargamex/internal/domain/repository"
	"pasargamex/internal/infrastructure/firebase"
	"pasargamex/pkg/response"
)

type DevTokenHandler struct {
	firebaseAuth *firebase.FirebaseAuthClient
	userRepo     repository.UserRepository
}

var devTokenHandler *DevTokenHandler

func NewDevTokenHandler(firebaseAuth *firebase.FirebaseAuthClient, userRepo repository.UserRepository) *DevTokenHandler {
	return &DevTokenHandler{
		firebaseAuth: firebaseAuth,
		userRepo:     userRepo,
	}
}

func SetupDevTokenHandler(firebaseAuth *firebase.FirebaseAuthClient, userRepo repository.UserRepository) {
	devTokenHandler = NewDevTokenHandler(firebaseAuth, userRepo)
}

func GetDevTokenHandler() *DevTokenHandler {
	return devTokenHandler
}

// GenerateUserToken menghasilkan token abadi untuk user biasa
func (h *DevTokenHandler) GenerateUserToken(c echo.Context) error {
	// Ambil user pertama dengan role user
	query := h.userRepo.GetUserByRole(c.Request().Context(), "user", 1)
	if len(query) == 0 {
		return response.Error(c, response.NewError("USER_NOT_FOUND", "No regular user found", http.StatusNotFound))
	}
	
	// Generate token
	token, err := h.firebaseAuth.GenerateLongLivedToken(c.Request().Context(), query[0].ID)
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Success(c, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id": query[0].ID,
			"email": query[0].Email,
			"username": query[0].Username,
			"role": query[0].Role,
		},
	})
}

// GenerateAdminToken menghasilkan token abadi untuk admin
func (h *DevTokenHandler) GenerateAdminToken(c echo.Context) error {
	// Ambil user pertama dengan role admin
	query := h.userRepo.GetUserByRole(c.Request().Context(), "admin", 1)
	if len(query) == 0 {
		return response.Error(c, response.NewError("ADMIN_NOT_FOUND", "No admin user found", http.StatusNotFound))
	}
	
	// Generate token
	token, err := h.firebaseAuth.GenerateLongLivedToken(c.Request().Context(), query[0].ID)
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Success(c, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id": query[0].ID,
			"email": query[0].Email,
			"username": query[0].Username,
			"role": query[0].Role,
		},
	})
}