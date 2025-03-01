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

type authResponse struct {
	Token string      `json:"token"`
	User  userResponse `json:"user"`
}

type userResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Phone    string `json:"phone,omitempty"`
}

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
		Token: result.Token,
		User: userResponse{
			ID:       result.User.ID,
			Email:    result.User.Email,
			Username: result.User.Username,
			Phone:    result.User.Phone,
		},
	})
}

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
	
	// Login implementation - this will need a corresponding usecase method
	result, err := h.authUseCase.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
	}
	
	return c.JSON(http.StatusOK, authResponse{
		Token: result.Token,
		User: userResponse{
			ID:       result.User.ID,
			Email:    result.User.Email,
			Username: result.User.Username,
			Phone:    result.User.Phone,
		},
	})
}

func (h *AuthHandler) GetCurrentUser(c echo.Context) error {
	// Get user ID from context (set by auth middleware)
	uid := c.Get("uid").(string)
	
	// Get user details - this will need a corresponding usecase method
	user, err := h.authUseCase.GetUserByID(c.Request().Context(), uid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get user details")
	}
	
	return c.JSON(http.StatusOK, userResponse{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
		Phone:    user.Phone,
	})
}

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
    
    // Call usecase
    newToken, err := h.authUseCase.RefreshToken(c.Request().Context(), req.RefreshToken)
    if err != nil {
        return response.Error(c, err)
    }
    
    return response.Success(c, map[string]string{
        "token": newToken,
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

