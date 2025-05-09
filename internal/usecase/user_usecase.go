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

type VerifyIdentityInput struct {
	FullName    string
	Address     string
	DateOfBirth time.Time
	IdNumber    string
	IdCardImage string
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
    user, err := uc.userRepo.GetByID(ctx, userId)
    if err != nil {
        return errors.NotFound("User", err)
    }
    
    _, err = uc.firebaseAuth.SignInWithEmailPassword(user.Email, currentPassword)
    if err != nil {
        return errors.Unauthorized("Current password is incorrect", err)
    }
    
    if err := uc.firebaseAuth.UpdateUserPassword(ctx, userId, newPassword); err != nil {
        return errors.Internal("Failed to update password", err)
    }
    
    return nil
}

func (uc *UserUseCase) SubmitVerification(ctx context.Context, userID string, input VerifyIdentityInput) (*entity.User, error) {
	// Get user
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	
	// Validasi status
	if user.VerificationStatus == "verified" {
		return nil, errors.BadRequest("User already verified", nil)
	}
	
	// Update user info
	user.FullName = input.FullName
	user.Address = input.Address
	user.DateOfBirth = input.DateOfBirth
	user.IdNumber = input.IdNumber
	user.IdCardImage = input.IdCardImage
	user.VerificationStatus = "pending"
	user.UpdatedAt = time.Now()
	
	// Save ke repository
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	
	return user, nil
}

// Untuk admin menyetujui/menolak verifikasi
func (uc *UserUseCase) ProcessVerification(ctx context.Context, adminID, userID, status string) (*entity.User, error) {
	// TODO: Validate admin
	
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	
	if user.VerificationStatus != "pending" {
		return nil, errors.BadRequest("Verification is not pending", nil)
	}
	
	user.VerificationStatus = status // "verified" atau "rejected"
	user.UpdatedAt = time.Now()
	
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	
	return user, nil
}

func (uc *UserUseCase) GetUserRepository() repository.UserRepository {
    return uc.userRepo
}