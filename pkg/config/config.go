package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort      string
	FirebaseProject string
	Environment     string
	JWTSecret       string
	JWTExpiry       int64
}

func Load() (*Config, error) {
	godotenv.Load()

	config := &Config{
		ServerPort:      getEnv("SERVER_PORT", "8080"),
		FirebaseProject: getEnv("FIREBASE_PROJECT_ID", ""),
		Environment:     getEnv("ENVIRONMENT", "development"),
		JWTSecret:       getEnv("JWT_SECRET", "your-secret-key"),
		JWTExpiry:       getEnvAsInt64("JWT_EXPIRY", 24*60*60), // 24 hours
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return intValue
		}
	}
	return defaultValue
}