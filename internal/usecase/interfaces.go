package usecase

import "context"

type FirebaseAuthClient interface {
    CreateUser(ctx context.Context, email, password, displayName string) (string, error)
    VerifyToken(ctx context.Context, token string) (string, error)
    GenerateToken(ctx context.Context, uid string) (string, error)
    SignInWithEmailPassword(email, password string) (string, error)
    UpdateUserPassword(ctx context.Context, uid, newPassword string) error
    TestConnection(ctx context.Context) error
    SignInWithEmailPasswordWithRefresh(email, password string) (string, string, error)
    RefreshIdToken(refreshToken string) (string, string, error)
}