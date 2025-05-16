package handler

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

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

func (h *DevTokenHandler) GenerateUserToken(c echo.Context) error {
	// Hanya berfungsi di mode development
	if c.Get("environment") != "development" {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Endpoint not found",
		})
	}
	
	// Ambil user pertama dengan role user
	users, total, err := h.userRepo.FindByField(c.Request().Context(), "role", "user", 1, 0)
	if err != nil || total == 0 || len(users) == 0 {
		return response.Error(c, response.NewError("USER_NOT_FOUND", "No regular user found", http.StatusNotFound))
	}
	
	user := users[0]
	
	// Generate token pair
	token, refreshToken, err := h.firebaseAuth.GenerateDevTokenPair(c.Request().Context(), user.Email)
	if err != nil {
		// Log error tapi jangan expose detail error ke response
		log.Printf("Error generating token pair: %v", err)
		
		// Fallback ke metode lama jika error
		customToken, err := h.firebaseAuth.GenerateLongLivedToken(c.Request().Context(), user.ID)
		if err != nil {
			return response.Error(c, err)
		}
		
		return response.Success(c, map[string]interface{}{
			"token": customToken,
			"user": map[string]interface{}{
				"id":       user.ID,
				"email":    user.Email,
				"username": user.Username,
				"role":     user.Role,
			},
			"note": "Using fallback token method (custom token only). Might need to exchange for ID token manually.",
		})
	}
	
	// Token info untuk debugging
	tokenInfo := parseJWTWithoutVerification(token)
	expiryTime := time.Unix(tokenInfo["exp"].(int64), 0)
	
	return response.Success(c, map[string]interface{}{
		"token":         token,
		"refresh_token": refreshToken,
		"expires_at":    expiryTime.Format(time.RFC3339),
		"expires_in":    expiryTime.Sub(time.Now()).String(),
		"user": map[string]interface{}{
			"id":       user.ID,
			"email":    user.Email,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

// GenerateAdminToken generates tokens for an admin user for development
func (h *DevTokenHandler) GenerateAdminToken(c echo.Context) error {
	// Hanya berfungsi di mode development
	if c.Get("environment") != "development" {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Endpoint not found",
		})
	}
	
	// Ambil admin pertama
	users, total, err := h.userRepo.FindByField(c.Request().Context(), "role", "admin", 1, 0)
	if err != nil || total == 0 || len(users) == 0 {
		return response.Error(c, response.NewError("ADMIN_NOT_FOUND", "No admin user found", http.StatusNotFound))
	}
	
	user := users[0]
	
	// Generate token pair
	token, refreshToken, err := h.firebaseAuth.GenerateDevTokenPair(c.Request().Context(), user.Email)
	if err != nil {
		// Log error tapi jangan expose detail error ke response
		log.Printf("Error generating token pair: %v", err)
		
		// Fallback ke metode lama jika error
		customToken, err := h.firebaseAuth.GenerateLongLivedToken(c.Request().Context(), user.ID)
		if err != nil {
			return response.Error(c, err)
		}
		
		return response.Success(c, map[string]interface{}{
			"token": customToken,
			"user": map[string]interface{}{
				"id":       user.ID,
				"email":    user.Email,
				"username": user.Username,
				"role":     user.Role,
			},
			"note": "Using fallback token method (custom token only). Might need to exchange for ID token manually.",
		})
	}
	
	// Token info untuk debugging
	tokenInfo := parseJWTWithoutVerification(token)
	expiryTime := time.Unix(tokenInfo["exp"].(int64), 0)
	
	return response.Success(c, map[string]interface{}{
		"token":         token,
		"refresh_token": refreshToken,
		"expires_at":    expiryTime.Format(time.RFC3339),
		"expires_in":    expiryTime.Sub(time.Now()).String(),
		"user": map[string]interface{}{
			"id":       user.ID,
			"email":    user.Email,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

// Helper function to decode JWT without verification
func parseJWTWithoutVerification(tokenString string) map[string]interface{} {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return map[string]interface{}{}
	}
	
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return map[string]interface{}{}
	}
	
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return map[string]interface{}{}
	}
	
	return claims
}

func (h *DevTokenHandler) GetLongLivedToken(c echo.Context) error {
    // Parse request
    var req struct {
        Email    string `json:"email" validate:"required,email"`
        UserID   string `json:"user_id" validate:"required"`
    }
    
    if err := c.Bind(&req); err != nil {
        return response.Error(c, err)
    }
    
    // Generate token langsung dengan Firebase
    token, refreshToken, err := h.firebaseAuth.GenerateDevTokenPair(c.Request().Context(), req.Email)
    if err != nil {
        return response.Error(c, err)
    }
    
    return response.Success(c, map[string]interface{}{
        "token": token,
        "refresh_token": refreshToken,
    })
}

func (h *DevTokenHandler) TestRefreshToken(c echo.Context) error {
    // Parse request
    var req struct {
        RefreshToken string `json:"refresh_token" validate:"required"`
    }
    
    if err := c.Bind(&req); err != nil {
        return response.Error(c, err)
    }
    
    // Try refreshing the token
    token, refreshToken, err := h.firebaseAuth.RefreshIdToken(req.RefreshToken)
    if err != nil {
        log.Printf("Refresh token error: %v", err)
        return response.Error(c, err)
    }
    
    // Return success
    return response.Success(c, map[string]interface{}{
        "token":         token,
        "refresh_token": refreshToken,
        "message":       "Token refreshed successfully",
    })
}