package handler

import (
	"pasargamex/internal/usecase"
)

var (
	authHandler      *AuthHandler
	userHandler      *UserHandler
	gameTitleHandler *GameTitleHandler
	productHandler   *ProductHandler
	reviewHandler    *ReviewHandler
)

// Setup initializes all handlers
func Setup(
	authUseCase *usecase.AuthUseCase,
	userUseCase *usecase.UserUseCase,
	gameTitleUseCase *usecase.GameTitleUseCase,
	productUseCase *usecase.ProductUseCase,
	reviewUseCase *usecase.ReviewUseCase,
) {
	authHandler = NewAuthHandler(authUseCase)
	userHandler = NewUserHandler(userUseCase)
	gameTitleHandler = NewGameTitleHandler(gameTitleUseCase)
	productHandler = NewProductHandler(productUseCase)
	reviewHandler = NewReviewHandler(reviewUseCase)
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

func GetProductHandler() *ProductHandler {
	return productHandler
}

func GetReviewHandler() *ReviewHandler {
	return reviewHandler
}