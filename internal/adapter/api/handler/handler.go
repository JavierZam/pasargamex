package handler

import (
	"pasargamex/internal/usecase"
)

var (
	authHandler        *AuthHandler
	userHandler        *UserHandler
	gameTitleHandler   *GameTitleHandler
	productHandler     *ProductHandler
	reviewHandler      *ReviewHandler
	transactionHandler *TransactionHandler
)

func Setup(
	authUseCase *usecase.AuthUseCase,
	userUseCase *usecase.UserUseCase,
	gameTitleUseCase *usecase.GameTitleUseCase,
	productUseCase *usecase.ProductUseCase,
	reviewUseCase *usecase.ReviewUseCase,
	transactionUseCase *usecase.TransactionUseCase,
) {
	authHandler = NewAuthHandler(authUseCase)
	userHandler = NewUserHandler(userUseCase)
	gameTitleHandler = NewGameTitleHandler(gameTitleUseCase)
	productHandler = NewProductHandler(productUseCase)
	reviewHandler = NewReviewHandler(reviewUseCase)
	transactionHandler = NewTransactionHandler(transactionUseCase)
}

func GetAuthHandler() *AuthHandler {
	return authHandler
}

func GetUserHandler() *UserHandler {
	return userHandler
}

func GetGameTitleHandler() *GameTitleHandler {
	return gameTitleHandler
}

func GetProductHandler() *ProductHandler {
	return productHandler
}

func GetReviewHandler() *ReviewHandler {
	return reviewHandler
}

func GetTransactionHandler() *TransactionHandler {
	return transactionHandler
}
