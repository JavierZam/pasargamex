package main

import (
	"context"
	"log"
	"os"

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
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Inisialisasi context
	ctx := context.Background()

	// Setup Firebase App
	var firebaseApp *fbapp.App

	// Gunakan err yang sudah ada tanpa mendeklarasikan ulang
	if os.Getenv("ENVIRONMENT") == "production" {
		firebaseApp, err = fbapp.NewApp(ctx, &fbapp.Config{ProjectID: cfg.FirebaseProject})
	} else {
		opt := option.WithCredentialsFile("pasargamex-firebase-adminsdk-fbsvc-9cdda86f4c.json")
		firebaseApp, err = fbapp.NewApp(ctx, &fbapp.Config{ProjectID: cfg.FirebaseProject}, opt)
	}

	if err != nil {
		log.Fatalf("Failed to initialize Firebase: %v", err)
	}

	// Initialize Firestore
	firestoreClient, err := firestore.NewClient(ctx, cfg.FirebaseProject)
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer firestoreClient.Close()

	// Initialize Firebase Auth
	authClient, err := firebaseApp.Auth(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize Firebase Auth: %v", err)
	}

	// Initialize repositories
	userRepo := repository.NewFirestoreUserRepository(firestoreClient)
	gameTitleRepo := repository.NewFirestoreGameTitleRepository(firestoreClient)
	productRepo := repository.NewFirestoreProductRepository(firestoreClient)

	// Initialize Firebase auth client adapter
	firebaseAuthClient := firebase.NewFirebaseAuthClient(authClient)

	// Initialize use cases
	authUseCase := usecase.NewAuthUseCase(userRepo, firebaseAuthClient)
	userUseCase := usecase.NewUserUseCase(userRepo, firebaseAuthClient)
	gameTitleUseCase := usecase.NewGameTitleUseCase(gameTitleRepo)
	productUseCase := usecase.NewProductUseCase(productRepo, gameTitleRepo, userRepo)

	// Setup handlers
	handler.Setup(authUseCase, userUseCase, gameTitleUseCase, productUseCase)

	// Initialize Echo
	e := echo.New()
	
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	
	// Initialize validator
	e.Validator = api.NewValidator()

	// Auth middleware
	authMiddleware := apimiddleware.NewAuthMiddleware(authClient)
	
	// Add health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})
	
	// Setup routers
	router.Setup(e, authMiddleware)
	
	// Start server
	e.Logger.Fatal(e.Start(":" + cfg.ServerPort))
}