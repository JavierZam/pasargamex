package usecase

import (
	"context"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

type AuthUseCase struct {
	userRepo     repository.UserRepository
	firebaseAuth FirebaseAuthClient
}

func NewAuthUseCase(userRepo repository.UserRepository, firebaseAuth FirebaseAuthClient) *AuthUseCase {
	return &AuthUseCase{
		userRepo:     userRepo,
		firebaseAuth: firebaseAuth,
	}
}

type RegisterInput struct {
	Email    string
	Password string
	Username string
	Phone    string
}

type AuthResult struct {
	User         *entity.User
	Token        string
	RefreshToken string
}

func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (*AuthResult, error) {

	token, refreshToken, err := uc.firebaseAuth.SignInWithEmailPasswordWithRefresh(email, password)
	if err != nil {
		return nil, errors.Unauthorized("Invalid credentials", err)
	}

	uid, err := uc.firebaseAuth.VerifyToken(ctx, token)
	if err != nil {
		return nil, errors.Internal("Failed to verify token", err)
	}

	user, err := uc.userRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, errors.NotFound("User", err)
	}

	return &AuthResult{
		User:         user,
		Token:        token,
		RefreshToken: refreshToken,
	}, nil
}

func (uc *AuthUseCase) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	existingUser, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err == nil && existingUser != nil {
		return nil, errors.BadRequest("Email already in use", nil)
	}

	users, _, err := uc.userRepo.FindByField(ctx, "username", input.Username, 1, 0)
	if err == nil && len(users) > 0 {
		return nil, errors.BadRequest("Username already in use", nil)
	}

	if input.Phone != "" {
		users, _, err := uc.userRepo.FindByField(ctx, "phone", input.Phone, 1, 0)
		if err == nil && len(users) > 0 {
			return nil, errors.BadRequest("Phone number already in use", nil)
		}
	}

	uid, err := uc.firebaseAuth.CreateUser(ctx, input.Email, input.Password, input.Username)
	if err != nil {
		return nil, errors.Internal("Failed to create user in authentication provider", err)
	}

	now := time.Now()
	user := &entity.User{
		ID:        uid,
		Email:     input.Email,
		Username:  input.Username,
		Phone:     input.Phone,
		Role:      "user",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {

		return nil, errors.Internal("Failed to create user record", err)
	}

	token, refreshToken, err := uc.firebaseAuth.SignInWithEmailPasswordWithRefresh(input.Email, input.Password)
	if err != nil {
		return nil, errors.Internal("Failed to generate authentication token", err)
	}

	return &AuthResult{
		User:         user,
		Token:        token,
		RefreshToken: refreshToken,
	}, nil
}

func (uc *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error) {

	token, newRefreshToken, err := uc.firebaseAuth.RefreshIdToken(refreshToken)
	if err != nil {
		return nil, errors.Unauthorized("Invalid refresh token", err)
	}

	uid, err := uc.firebaseAuth.VerifyToken(ctx, token)
	if err != nil {
		return nil, errors.Internal("Failed to verify token", err)
	}

	user, err := uc.userRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, errors.NotFound("User", err)
	}

	return &AuthResult{
		User:         user,
		Token:        token,
		RefreshToken: newRefreshToken,
	}, nil
}

func (uc *AuthUseCase) Logout(ctx context.Context, token string) error {

	return nil
}
