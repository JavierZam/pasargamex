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

// Modifikasi metode Login
func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (*AuthResult, error) {
    // Gunakan metode login dengan refresh token
    token, refreshToken, err := uc.firebaseAuth.SignInWithEmailPasswordWithRefresh(email, password)
    if err != nil {
        return nil, errors.Unauthorized("Invalid credentials", err)
    }

    // Verify token to get UID
    uid, err := uc.firebaseAuth.VerifyToken(ctx, token)
    if err != nil {
        return nil, errors.Internal("Failed to verify token", err)
    }

    // Get user data
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

// Modifikasi metode Register untuk juga mengembalikan refresh token
func (uc *AuthUseCase) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
    // Check if email already exists
    existingUser, err := uc.userRepo.GetByEmail(ctx, input.Email)
    if err == nil && existingUser != nil {
        return nil, errors.BadRequest("Email already in use", nil)
    }

    // Add check for username
    users, _, err := uc.userRepo.FindByField(ctx, "username", input.Username, 1, 0)
    if err == nil && len(users) > 0 {
        return nil, errors.BadRequest("Username already in use", nil)
    }
    
    // Add check for phone if provided
    if input.Phone != "" {
        users, _, err := uc.userRepo.FindByField(ctx, "phone", input.Phone, 1, 0)
        if err == nil && len(users) > 0 {
            return nil, errors.BadRequest("Phone number already in use", nil)
        }
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

// Tambahkan metode untuk refresh token
func (uc *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error) {
    // Refresh token dengan Firebase
    token, newRefreshToken, err := uc.firebaseAuth.RefreshIdToken(refreshToken)
    if err != nil {
        return nil, errors.Unauthorized("Invalid refresh token", err)
    }
    
    // Verify token to get UID
    uid, err := uc.firebaseAuth.VerifyToken(ctx, token)
    if err != nil {
        return nil, errors.Internal("Failed to verify token", err)
    }
    
    // Get user data
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
    // Untuk JWT, "logout" biasanya dihandle di client dengan menghapus token
    // Namun kita bisa memiliki implementasi server-side dengan token blacklisting
    // Untuk sederhananya, kita return success
    
    // Implementasi lengkap bisa menambahkan token ke blacklist di Redis/database
    return nil
}