package firebase

import (
	"context"
	"log"
	"strings"

	"firebase.google.com/go/v4/auth"
)

type FirebaseAuthClient struct {
	client *auth.Client
}

func NewFirebaseAuthClient(client *auth.Client) *FirebaseAuthClient {
	return &FirebaseAuthClient{
		client: client,
	}
}

func (f *FirebaseAuthClient) CreateUser(ctx context.Context, email, password, displayName string) (string, error) {
    log.Printf("Creating Firebase Auth user with email: %s", email)
    
    params := (&auth.UserToCreate{}).
        Email(email).
        Password(password).
        DisplayName(displayName)

    user, err := f.client.CreateUser(ctx, params)
    if err != nil {
        // Log error type dan detail
        log.Printf("Firebase Auth error type: %T", err)
        log.Printf("Firebase Auth error detail: %v", err)
        
        // Tidak perlu type assertion ke auth.Error
        // Cukup log error saja
        return "", err
    }
    
    log.Printf("Firebase Auth user created successfully with UID: %s", user.UID)
    return user.UID, nil
}

func (f *FirebaseAuthClient) VerifyToken(ctx context.Context, token string) (string, error) {
	result, err := f.client.VerifyIDToken(ctx, token)
	if err != nil {
		return "", err
	}
	
	return result.UID, nil
}

func (f *FirebaseAuthClient) GenerateToken(ctx context.Context, uid string) (string, error) {
	token, err := f.client.CustomToken(ctx, uid)
	if err != nil {
		return "", err
	}
	
	return token, nil
}

func (f *FirebaseAuthClient) UpdateUserPassword(ctx context.Context, uid, newPassword string) error {
    params := (&auth.UserToUpdate{}).
        Password(newPassword)
    
    _, err := f.client.UpdateUser(ctx, uid, params)
    if err != nil {
        return err
    }
    
    return nil
}

func (f *FirebaseAuthClient) TestConnection(ctx context.Context) error {
	// Coba mengambil user yang tidak ada
	// Jika error adalah "user not found", berarti koneksi berhasil
	_, err := f.client.GetUser(ctx, "non-existent-uid")
	if err != nil {
		// Error user not found adalah expected dan menunjukkan koneksi berhasil
		if strings.Contains(err.Error(), "user not found") {
			return nil
		}
		// Error lain menunjukkan masalah koneksi
		return err
	}
	// Tidak ada error tapi user ditemukan - unlikely scenario
	return nil
}