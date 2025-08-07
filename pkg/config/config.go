package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort         string
	FirebaseProject    string
	Environment        string
	JWTSecret          string
	JWTExpiry          string
	FirebaseApiKey     string
	FirebaseAuthDomain string
	StorageBucket      string
	
	// Midtrans Configuration
	MidtransServerKey   string
	MidtransClientKey   string
	MidtransEnvironment string // sandbox or production
}

func Load() (*Config, error) {
	godotenv.Load()

	config := &Config{
		ServerPort:         getEnv("PORT", getEnv("SERVER_PORT", "8080")), // Cloud Run uses PORT env var
		FirebaseProject:    getEnv("FIREBASE_PROJECT_ID", ""),
		Environment:        getEnv("ENVIRONMENT", "development"),
		JWTSecret:          getEnv("JWT_SECRET", "default-secret"),
		JWTExpiry:          getEnv("JWT_EXPIRY", "86400"),
		FirebaseApiKey:     getEnv("FIREBASE_API_KEY", ""),
		FirebaseAuthDomain: getEnv("FIREBASE_AUTH_DOMAIN", ""),
		StorageBucket:      getEnv("STORAGE_BUCKET", ""),
		
		// Midtrans Configuration (Sandbox by default)
		MidtransServerKey:   getEnv("MIDTRANS_SERVER_KEY", "SB-Mid-server-your-sandbox-server-key"),
		MidtransClientKey:   getEnv("MIDTRANS_CLIENT_KEY", "SB-Mid-client-your-sandbox-client-key"),
		MidtransEnvironment: getEnv("MIDTRANS_ENVIRONMENT", "sandbox"),
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
