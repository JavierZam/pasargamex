package firebase

import (
	"context"

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
	params := (&auth.UserToCreate{}).
		Email(email).
		Password(password).
		DisplayName(displayName)

	user, err := f.client.CreateUser(ctx, params)
	if err != nil {
		return "", err
	}
	
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