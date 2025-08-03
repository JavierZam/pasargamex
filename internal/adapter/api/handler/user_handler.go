package handler

import (
	"log"
	"pasargamex/internal/usecase"
	"pasargamex/pkg/errors"
	"pasargamex/pkg/response"
	"time"

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

type verifyIdentityRequest struct {
	FullName    string `json:"full_name" validate:"required"`
	Address     string `json:"address" validate:"required"`
	DateOfBirth string `json:"date_of_birth" validate:"required"`
	IdNumber    string `json:"id_number" validate:"required"`
	IdCardImage string `json:"id_card_image" validate:"required,url"`
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

	uid := c.Get("uid").(string)
	log.Printf("Updating profile for user ID: %s", uid)

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

	uid := c.Get("uid").(string)

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
		"role":     user.Role,
		"status":   user.Status,
	})
}

// New: GetUserByID - get any user profile by ID (for chat purposes)
func (h *UserHandler) GetUserByID(c echo.Context) error {
	targetUserID := c.Param("id")
	if targetUserID == "" {
		return response.Error(c, errors.BadRequest("User ID is required", nil))
	}

	user, err := h.userUseCase.GetUserProfile(c.Request().Context(), targetUserID)
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

	uid := c.Get("uid").(string)

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

func (h *UserHandler) SubmitVerification(c echo.Context) error {
	var req verifyIdentityRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return response.Error(c, errors.BadRequest("Invalid date format", err))
	}

	userID := c.Get("uid").(string)

	user, err := h.userUseCase.SubmitVerification(c.Request().Context(), userID, usecase.VerifyIdentityInput{
		FullName:    req.FullName,
		Address:     req.Address,
		DateOfBirth: dob,
		IdNumber:    req.IdNumber,
		IdCardImage: req.IdCardImage,
	})

	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, user)
}

type processVerificationRequest struct {
	Status string `json:"status" validate:"required,oneof=verified rejected"`
}

func (h *UserHandler) ProcessVerification(c echo.Context) error {

	userID := c.Param("userId")
	if userID == "" {
		return response.Error(c, errors.BadRequest("User ID is required", nil))
	}

	var req processVerificationRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	adminID := c.Get("uid").(string)

	user, err := h.userUseCase.ProcessVerification(c.Request().Context(), adminID, userID, req.Status)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, user)
}

type updateUserRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=user admin seller buyer"`
}

// UpdateUserRole allows admin to update any user's role
func (h *UserHandler) UpdateUserRole(c echo.Context) error {
	userID := c.Param("userId")
	if userID == "" {
		return response.Error(c, errors.BadRequest("User ID is required", nil))
	}

	var req updateUserRoleRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	adminID := c.Get("uid").(string)

	user, err := h.userUseCase.UpdateUserRole(c.Request().Context(), adminID, userID, req.Role)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]interface{}{
		"message": "User role updated successfully",
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}
