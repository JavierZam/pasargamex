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
	User  *entity.User
	Token string
}

func (uc *AuthUseCase) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	// Check if email already exists
	existingUser, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err == nil && existingUser != nil {
		return nil, errors.BadRequest("Email already in use", nil)
	}

	// Create user in Firebase Auth
	uid, err := uc.firebaseAuth.CreateUser(ctx, input.Email, input.Password, input.Username)
	if err != nil {
		return nil, errors.Internal("Failed to create user in authentication provider", err)
	}

	// Create user in repository
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
		// Consider cleanup in Firebase if this fails
		return nil, errors.Internal("Failed to create user record", err)
	}

	// Generate token
	token, err := uc.firebaseAuth.GenerateToken(ctx, uid)
	if err != nil {
		return nil, errors.Internal("Failed to generate authentication token", err)
	}

	return &AuthResult{
		User:  user,
		Token: token,
	}, nil
}

func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	// Note: Firebase Admin SDK doesn't support password sign-in directly
	// We would need Firebase Auth REST API for this
	// For now, we'll just mock this with a "successful" login

	// Get user by email
	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Generate token
	token, err := uc.firebaseAuth.GenerateToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:  user,
		Token: token,
	}, nil
}

func (uc *AuthUseCase) GetUserByID(ctx context.Context, id string) (*entity.User, error) {
	user, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.NotFound("User", err)
	}
	return user, nil
}

func (uc *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	uid, err := uc.firebaseAuth.VerifyToken(ctx, refreshToken)
	if err != nil {
		return "", errors.Unauthorized("Invalid refresh token", err)
	}
	
	newToken, err := uc.firebaseAuth.GenerateToken(ctx, uid)
	if err != nil {
		return "", errors.Internal("Failed to generate new token", err)
	}
	
	return newToken, nil
}

func (uc *AuthUseCase) Logout(ctx context.Context, token string) error {
    // Untuk JWT, "logout" biasanya dihandle di client dengan menghapus token
    // Namun kita bisa memiliki implementasi server-side dengan token blacklisting
    // Untuk sederhananya, kita return success
    
    // Implementasi lengkap bisa menambahkan token ke blacklist di Redis/database
    return nil
}