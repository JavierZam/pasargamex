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

	// Update last seen on login
	now := time.Now()
	user.LastSeen = now
	user.OnlineStatus = "online"
	user.UpdatedAt = now
	
	if err := uc.userRepo.Update(ctx, user); err != nil {
		// Log error but don't fail login for this
		// log.Printf("Failed to update user last seen: %v", err)
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
		ID:           uid,
		Email:        input.Email,
		Username:     input.Username,
		Phone:        input.Phone,
		Role:         "user",
		Status:       "active",
		LastSeen:     now,
		OnlineStatus: "offline",
		Provider:     "email",
		CreatedAt:    now,
		UpdatedAt:    now,
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

// CreateOrUpdateUserFromFirebaseOAuth creates or updates user from Firebase OAuth (Google, etc.)
func (uc *AuthUseCase) CreateOrUpdateUserFromFirebaseOAuth(ctx context.Context, uid string) (*AuthResult, error) {
	// Get Firebase profile data
	firebaseUser, err := uc.firebaseAuth.GetUserProfile(ctx, uid)
	if err != nil {
		return nil, errors.Internal("Failed to get Firebase user profile", err)
	}

	now := time.Now()
	
	// Check if user already exists
	existingUser, err := uc.userRepo.GetByID(ctx, uid)
	if err == nil && existingUser != nil {
		// Update existing user with latest profile data
		existingUser.LastSeen = now
		existingUser.OnlineStatus = "online"
		existingUser.UpdatedAt = now
		
		// Update profile data if available
		if firebaseUser.PhotoURL != "" {
			existingUser.PhotoURL = firebaseUser.PhotoURL
			existingUser.AvatarURL = firebaseUser.PhotoURL
		}
		if firebaseUser.DisplayName != "" && existingUser.Username == "" {
			existingUser.Username = firebaseUser.DisplayName
		}
		
		if err := uc.userRepo.Update(ctx, existingUser); err != nil {
			return nil, errors.Internal("Failed to update user profile", err)
		}
		
		// Generate tokens for existing user
		token, err := uc.firebaseAuth.GenerateToken(ctx, uid)
		if err != nil {
			return nil, errors.Internal("Failed to generate token", err)
		}
		
		return &AuthResult{
			User:  existingUser,
			Token: token,
		}, nil
	}
	
	// Create new user from Firebase profile
	// For now, determine provider based on available data
	provider := "firebase"
	if firebaseUser.Email != "" && firebaseUser.PhotoURL != "" {
		provider = "google.com" // Likely Google OAuth if has photo
	}
	
	user := &entity.User{
		ID:           firebaseUser.UID,
		Email:        firebaseUser.Email,
		Username:     firebaseUser.DisplayName,
		Role:         "user",
		Status:       "active",
		PhotoURL:     firebaseUser.PhotoURL,
		AvatarURL:    firebaseUser.PhotoURL,
		LastSeen:     now,
		OnlineStatus: "online",
		Provider:     provider,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	
	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, errors.Internal("Failed to create user from OAuth profile", err)
	}
	
	// Generate tokens for new user
	token, err := uc.firebaseAuth.GenerateToken(ctx, uid)
	if err != nil {
		return nil, errors.Internal("Failed to generate token", err)
	}
	
	return &AuthResult{
		User:  user,
		Token: token,
	}, nil
}

// VerifyToken verifies a Firebase token and returns the UID
func (uc *AuthUseCase) VerifyToken(ctx context.Context, token string) (string, error) {
	uid, err := uc.firebaseAuth.VerifyToken(ctx, token)
	if err != nil {
		return "", errors.Unauthorized("Invalid or expired token", err)
	}
	return uid, nil
}
