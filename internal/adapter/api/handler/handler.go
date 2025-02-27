package handler

import (
	"pasargamex/internal/usecase"
)

var (
	authHandler      *AuthHandler
	userHandler      *UserHandler
	gameTitleHandler *GameTitleHandler
)

// Setup initializes all handlers
func Setup(
	authUseCase *usecase.AuthUseCase,
	userUseCase *usecase.UserUseCase,
	gameTitleUseCase *usecase.GameTitleUseCase,
) {
	authHandler = NewAuthHandler(authUseCase)
	userHandler = NewUserHandler(userUseCase)
	gameTitleHandler = NewGameTitleHandler(gameTitleUseCase)
}

// GetAuthHandler returns the auth handler
func GetAuthHandler() *AuthHandler {
	return authHandler
}

// GetUserHandler returns the user handler
func GetUserHandler() *UserHandler {
	return userHandler
}

// GetGameTitleHandler returns the game title handler
func GetGameTitleHandler() *GameTitleHandler {
	return gameTitleHandler
}