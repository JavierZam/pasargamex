package usecase

import (
	"context"
	"log"
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

    // Generate ID token directly
    token, err := uc.firebaseAuth.SignInWithEmailPassword(input.Email, input.Password)
    if err != nil {
        return nil, errors.Internal("Failed to generate authentication token", err)
    }

    return &AuthResult{
        User:  user,
        Token: token,
    }, nil
}

func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (*AuthResult, error) {
    // Gunakan metode login dengan email/password langsung
    token, err := uc.firebaseAuth.SignInWithEmailPassword(email, password)
    if err != nil {
        // Pastikan error dilog dan dikembalikan dengan benar
        log.Printf("Login failed: %v", err)
        return nil, errors.Unauthorized("Invalid credentials", err)
    }

    // Verify token to get UID
    uid, err := uc.firebaseAuth.VerifyToken(ctx, token)
    if err != nil {
        log.Printf("Token verification failed: %v", err)
        return nil, errors.Internal("Failed to verify token", err)
    }

    // Get user data
    user, err := uc.userRepo.GetByID(ctx, uid)
    if err != nil {
        log.Printf("Failed to get user by ID: %v", err)
        return nil, errors.NotFound("User", err)
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