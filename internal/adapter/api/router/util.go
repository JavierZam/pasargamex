package router

import (
	"context"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

// VerifyToken adalah fungsi helper untuk memverifikasi token tanpa middleware authentication
func VerifyToken(authClient *auth.Client) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Get authorization header
            authHeader := c.Request().Header.Get("Authorization")
            if authHeader == "" {
                // Tidak ada token, lanjutkan tanpa autentikasi
                return next(c)
            }

            // Validasi format token
            parts := strings.Split(authHeader, " ")
            if len(parts) != 2 || parts[0] != "Bearer" {
                // Format token salah, lanjutkan tanpa autentikasi
                return next(c)
            }

            // Extract token
            token := parts[1]

            // Verify token
            firebaseToken, err := authClient.VerifyIDToken(context.Background(), token)
            if err != nil {
                // Token tidak valid, lanjutkan tanpa autentikasi
                return next(c)
            }

            // Set user ID in context
            c.Set("uid", firebaseToken.UID)

            // Lanjutkan ke handler berikutnya
            return next(c)
        }
    }
}