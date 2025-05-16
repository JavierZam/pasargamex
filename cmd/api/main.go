package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"google.golang.org/api/option"

	fbapp "firebase.google.com/go/v4"

	"pasargamex/internal/adapter/api"
	"pasargamex/internal/adapter/api/handler"
	apimiddleware "pasargamex/internal/adapter/api/middleware"
	"pasargamex/internal/adapter/api/router"
	"pasargamex/internal/adapter/repository"
	"pasargamex/internal/infrastructure/firebase"
	"pasargamex/internal/usecase"
	"pasargamex/pkg/config"
)

func main() {

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx := context.Background()

	serviceAccountPath := os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH")
	if serviceAccountPath == "" {
		serviceAccountPath = "./pasargamex-458303-firebase-adminsdk-fbsvc-f079266cd9.json"
	}

	var firebaseApp *fbapp.App
	opt := option.WithCredentialsFile(serviceAccountPath)
	firebaseApp, err = fbapp.NewApp(ctx, &fbapp.Config{ProjectID: cfg.FirebaseProject}, opt)
	if err != nil {
		log.Fatalf("Failed to initialize Firebase: %v", err)
	}

	firestoreClient, err := firestore.NewClient(ctx, cfg.FirebaseProject, opt)
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer firestoreClient.Close()

	authClient, err := firebaseApp.Auth(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize Firebase Auth: %v", err)
	}

	userRepo := repository.NewFirestoreUserRepository(firestoreClient)
	gameTitleRepo := repository.NewFirestoreGameTitleRepository(firestoreClient)
	productRepo := repository.NewFirestoreProductRepository(firestoreClient)
	reviewRepo := repository.NewFirestoreReviewRepository(firestoreClient)
	transactionRepo := repository.NewFirestoreTransactionRepository(firestoreClient)

	firebaseAuthClient := firebase.NewFirebaseAuthClient(authClient, cfg.FirebaseApiKey)

	authUseCase := usecase.NewAuthUseCase(userRepo, firebaseAuthClient)
	userUseCase := usecase.NewUserUseCase(userRepo, firebaseAuthClient)
	gameTitleUseCase := usecase.NewGameTitleUseCase(gameTitleRepo)
	productUseCase := usecase.NewProductUseCase(productRepo, gameTitleRepo, userRepo, transactionRepo)
	reviewUseCase := usecase.NewReviewUseCase(reviewRepo, userRepo)
	transactionUseCase := usecase.NewTransactionUseCase(transactionRepo, productRepo, userRepo)

	handler.Setup(authUseCase, userUseCase, gameTitleUseCase, productUseCase, reviewUseCase, transactionUseCase)
	handler.SetupDevTokenHandler(firebaseAuthClient, userRepo)

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.Validator = api.NewValidator()

	authMiddleware := apimiddleware.NewAuthMiddleware(authClient)
	adminMiddleware := apimiddleware.NewAdminMiddleware(userRepo)

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	e.GET("/v1/debug/me", func(c echo.Context) error {

		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"error": "No authorization header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"error": "Invalid authorization format",
			})
		}

		idToken := parts[1]

		token, err := authClient.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"error":   "Invalid token",
				"details": err.Error(),
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"uid":            token.UID,
			"token_verified": true,
		})
	}, authMiddleware.Authenticate)

	router.Setup(e, authMiddleware, adminMiddleware, authClient)
	router.SetupDevRouter(e, cfg.Environment)

	log.Printf("Starting server on port %s...", cfg.ServerPort)
	e.Logger.Fatal(e.Start(":" + cfg.ServerPort))
}
