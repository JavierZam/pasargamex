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
}

func Load() (*Config, error) {
	godotenv.Load()

	config := &Config{
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		FirebaseProject:    getEnv("FIREBASE_PROJECT_ID", ""),
		Environment:        getEnv("ENVIRONMENT", "development"),
		JWTSecret:          getEnv("JWT_SECRET", "default-secret"),
		JWTExpiry:          getEnv("JWT_EXPIRY", "86400"),
		FirebaseApiKey:     getEnv("FIREBASE_API_KEY", ""),
		FirebaseAuthDomain: getEnv("FIREBASE_AUTH_DOMAIN", ""),
		StorageBucket:      getEnv("STORAGE_BUCKET", ""),
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
