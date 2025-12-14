package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
	"pasargamex/internal/domain/repository"
)

type AuthMiddleware struct {
	authClient *auth.Client
	userRepo   repository.UserRepository
}

func NewAuthMiddleware(authClient *auth.Client, userRepo repository.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{
		authClient: authClient,
		userRepo:   userRepo,
	}
}

func (m *AuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {

		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Authorization header is required")
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization format")
		}

		idToken := parts[1]

		token, err := m.authClient.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or expired token")
		}

		c.Set("uid", token.UID)

		// Update user's last_seen timestamp on API activity
		go m.updateUserLastSeen(c.Request().Context(), token.UID)

		return next(c)
	}
}

func (m *AuthMiddleware) GetUIDFromToken(ctx context.Context, token string) (string, error) {

	firebaseToken, err := m.authClient.VerifyIDToken(ctx, token)
	if err != nil {
		return "", err
	}

	return firebaseToken.UID, nil
}

// updateUserLastSeen updates user's last_seen timestamp and online status
func (m *AuthMiddleware) updateUserLastSeen(ctx context.Context, userID string) {
	if m.userRepo == nil {
		return // Skip if no user repository
	}
	
	user, err := m.userRepo.GetByID(ctx, userID)
	if err != nil {
		// Log error but don't fail the request for this
		// log.Printf("Auth: Failed to get user %s for last_seen update: %v", userID, err)
		return
	}
	
	now := time.Now()
	user.LastSeen = now
	user.OnlineStatus = "online"
	user.UpdatedAt = now
	
	// Fix any users with zero-value last_seen (from before the update)
	if user.LastSeen.IsZero() || user.LastSeen.Year() == 1 {
		user.LastSeen = now
	}
	if user.OnlineStatus == "" {
		user.OnlineStatus = "online"
	}
	
	if err := m.userRepo.Update(ctx, user); err != nil {
		// Log error but don't fail the request for this
		// log.Printf("Auth: Failed to update last_seen for user %s: %v", userID, err)
		return
	}
}

// RequireAuth middleware for Gorilla Mux
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		idToken := parts[1]

		token, err := m.authClient.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Add user ID to request header for downstream handlers
		r.Header.Set("X-User-ID", token.UID)

		next.ServeHTTP(w, r)
	})
}
