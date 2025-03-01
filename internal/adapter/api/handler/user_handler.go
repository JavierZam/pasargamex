package handler

import (
	"log"
	"pasargamex/internal/usecase"
	"pasargamex/pkg/response"

	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	userUseCase *usecase.UserUseCase
}

func NewUserHandler(userUseCase *usecase.UserUseCase) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
	}
}

type updateProfileRequest struct {
	Username string `json:"username" validate:"omitempty,min=3"`
	Phone    string `json:"phone" validate:"omitempty,e164"`
	Bio      string `json:"bio" validate:"omitempty,max=500"`
}

func (h *UserHandler) UpdateProfile(c echo.Context) error {
    var req updateProfileRequest
    if err := c.Bind(&req); err != nil {
        log.Printf("Error binding request: %v", err)
        return response.Error(c, err)
    }

    log.Printf("Update profile request: %+v", req)
    
    if err := c.Validate(&req); err != nil {
        log.Printf("Validation error: %v", err)
        return response.Error(c, err)
    }

    // Get user ID from context
    uid := c.Get("uid").(string)
    log.Printf("Updating profile for user ID: %s", uid)
    
    // Call use case
    user, err := h.userUseCase.UpdateProfile(c.Request().Context(), uid, usecase.UpdateProfileInput{
        Username: req.Username,
        Phone:    req.Phone,
        Bio:      req.Bio,
    })

    if err != nil {
        log.Printf("Error updating profile: %v", err)
        return response.Error(c, err)
    }

    log.Printf("Profile updated successfully")
    return response.Success(c, map[string]interface{}{
        "id":       user.ID,
        "email":    user.Email,
        "username": user.Username,
        "phone":    user.Phone,
        "bio":      user.Bio,
    })
}

func (h *UserHandler) GetProfile(c echo.Context) error {
	// Get user ID from context
	uid := c.Get("uid").(string)
	
	// Call use case
	user, err := h.userUseCase.GetUserProfile(c.Request().Context(), uid)
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Success(c, map[string]interface{}{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
		"phone":    user.Phone,
		"bio":      user.Bio,
	})
}

func (h *UserHandler) UpdatePassword(c echo.Context) error {
    var req struct {
        CurrentPassword string `json:"current_password" validate:"required"`
        NewPassword     string `json:"new_password" validate:"required,min=8"`
    }
    
    if err := c.Bind(&req); err != nil {
        return response.Error(c, err)
    }
    
    if err := c.Validate(&req); err != nil {
        return response.Error(c, err)
    }
    
    // Get user ID from context
    uid := c.Get("uid").(string)
    
    // Call usecase
    err := h.userUseCase.UpdatePassword(
        c.Request().Context(),
        uid,
        req.CurrentPassword,
        req.NewPassword,
    )
    
    if err != nil {
        return response.Error(c, err)
    }
    
    return response.Success(c, map[string]string{
        "message": "Password updated successfully",
    })
}