package usecase

import "context"

type FirebaseAuthClient interface {
    CreateUser(ctx context.Context, email, password, displayName string) (string, error)
    VerifyToken(ctx context.Context, token string) (string, error)
    GenerateToken(ctx context.Context, uid string) (string, error)
    UpdateUserPassword(ctx context.Context, uid, newPassword string) error
}