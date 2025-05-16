package middleware

import (
	"context"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

type AuthMiddleware struct {
	authClient *auth.Client
}

func NewAuthMiddleware(authClient *auth.Client) *AuthMiddleware {
	return &AuthMiddleware{
		authClient: authClient,
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
