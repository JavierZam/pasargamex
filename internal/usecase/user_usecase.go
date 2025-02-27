package usecase

import (
	"context"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

type UserUseCase struct {
    userRepo     repository.UserRepository
    firebaseAuth FirebaseAuthClient 
}

func NewUserUseCase(userRepo repository.UserRepository, firebaseAuth FirebaseAuthClient) *UserUseCase {
    return &UserUseCase{
        userRepo:     userRepo,
        firebaseAuth: firebaseAuth,
    }
}

type UpdateProfileInput struct {
	Username string
	Phone    string
	Bio      string
}

func (uc *UserUseCase) UpdateProfile(ctx context.Context, userId string, input UpdateProfileInput) (*entity.User, error) {
	// Get existing user
	user, err := uc.userRepo.GetByID(ctx, userId)
	if err != nil {
		return nil, errors.NotFound("User", err)
	}

	// Update fields if provided
	if input.Username != "" {
		user.Username = input.Username
	}
	if input.Phone != "" {
		user.Phone = input.Phone
	}
	if input.Bio != "" {
		user.Bio = input.Bio
	}

	user.UpdatedAt = time.Now()

	// Save to repository
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, errors.Internal("Failed to update user profile", err)
	}

	return user, nil
}

func (uc *UserUseCase) GetUserProfile(ctx context.Context, userId string) (*entity.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userId)
	if err != nil {
		return nil, errors.NotFound("User", err)
	}
	
	return user, nil
}

func (uc *UserUseCase) UpdatePassword(ctx context.Context, userId, currentPassword, newPassword string) error {
    // Dalam implementasi lengkap, kita perlu verifikasi current password
    // Namun Firebase Admin SDK tidak memiliki API untuk verifikasi password
    // Jadi kita langsung update password
    
    if err := uc.firebaseAuth.UpdateUserPassword(ctx, userId, newPassword); err != nil {
        return errors.Internal("Failed to update password", err)
    }
    
    return nil
}