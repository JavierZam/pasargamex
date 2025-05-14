package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"pasargamex/internal/usecase"
	"pasargamex/pkg/errors"
	"pasargamex/pkg/response"
)

type AuthHandler struct {
	authUseCase *usecase.AuthUseCase
}

func NewAuthHandler(authUseCase *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{
		authUseCase: authUseCase,
	}
}

type registerRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Username string `json:"username" validate:"required,min=3"`
	Phone    string `json:"phone" validate:"omitempty,e164"`
}

type userResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Phone    string `json:"phone,omitempty"`
}

type authResponse struct {
    Token        string      `json:"token"`
    RefreshToken string      `json:"refresh_token"`
    User         userResponse `json:"user"`
}

// Modifikasi handler Login
func (h *AuthHandler) Login(c echo.Context) error {
    var req struct {
        Email    string `json:"email" validate:"required,email"`
        Password string `json:"password" validate:"required"`
    }
    
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }
    
    if err := c.Validate(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }
    
    result, err := h.authUseCase.Login(c.Request().Context(), req.Email, req.Password)
    if err != nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
    }
    
    return c.JSON(http.StatusOK, authResponse{
        Token:        result.Token,
        RefreshToken: result.RefreshToken,
        User: userResponse{
            ID:       result.User.ID,
            Email:    result.User.Email,
            Username: result.User.Username,
            Phone:    result.User.Phone,
        },
    })
}

// Modifikasi handler Register
func (h *AuthHandler) Register(c echo.Context) error {
    var req registerRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    if err := c.Validate(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    input := usecase.RegisterInput{
        Email:    req.Email,
        Password: req.Password,
        Username: req.Username,
        Phone:    req.Phone,
    }

    result, err := h.authUseCase.Register(c.Request().Context(), input)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "Failed to register: "+err.Error())
    }

    return c.JSON(http.StatusCreated, authResponse{
        Token:        result.Token,
        RefreshToken: result.RefreshToken,
        User: userResponse{
            ID:       result.User.ID,
            Email:    result.User.Email,
            Username: result.User.Username,
            Phone:    result.User.Phone,
        },
    })
}

// Modifikasi handler RefreshToken
func (h *AuthHandler) RefreshToken(c echo.Context) error {
    // Parse request
    var req struct {
        RefreshToken string `json:"refresh_token" validate:"required"`
    }
    
    if err := c.Bind(&req); err != nil {
        return response.Error(c, err)
    }
    
    if err := c.Validate(&req); err != nil {
        return response.Error(c, err)
    }
    
    // Call usecase untuk refresh token
    result, err := h.authUseCase.RefreshToken(c.Request().Context(), req.RefreshToken)
    if err != nil {
        return response.Error(c, err)
    }
    
    return response.Success(c, authResponse{
        Token:        result.Token,
        RefreshToken: result.RefreshToken,
        User: userResponse{
            ID:       result.User.ID,
            Email:    result.User.Email,
            Username: result.User.Username,
            Phone:    result.User.Phone,
        },
    })
}

func (h *AuthHandler) Logout(c echo.Context) error {
    // Get authorization header
    authHeader := c.Request().Header.Get("Authorization")
    if authHeader == "" {
        return response.Error(c, errors.Unauthorized("Authorization header required", nil))
    }
    
    // Extract token
    parts := strings.Split(authHeader, " ")
    if len(parts) != 2 || parts[0] != "Bearer" {
        return response.Error(c, errors.Unauthorized("Invalid authorization format", nil))
    }
    token := parts[1]
    
    // Call usecase
    if err := h.authUseCase.Logout(c.Request().Context(), token); err != nil {
        return response.Error(c, err)
    }
    
    return response.Success(c, map[string]string{
        "message": "Successfully logged out",
    })
}
