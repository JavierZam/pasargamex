package router

import (
	"context"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func VerifyToken(authClient *auth.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {

				return next(c)
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {

				return next(c)
			}

			token := parts[1]

			firebaseToken, err := authClient.VerifyIDToken(context.Background(), token)
			if err != nil {

				return next(c)
			}

			c.Set("uid", firebaseToken.UID)

			return next(c)
		}
	}
}
