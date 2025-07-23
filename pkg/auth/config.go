package auth

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

// JWTConfig holds the configuration for JWT tokens
type JWTConfig struct {
	SecretKey     string
	TokenDuration time.Duration
}

// DefaultConfig returns the default JWT configuration
func DefaultConfig() JWTConfig {
	// Load environment variables from .env file if it exists
	godotenv.Load()

	// Get JWT secret from environment variable or use a default (for development only)
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "your-256-bit-secret-key-for-development-only"
	}

	// Get token duration from environment variable or use a default
	tokenDuration := 24 * time.Hour // Default: 24 hours

	return JWTConfig{
		SecretKey:     secretKey,
		TokenDuration: tokenDuration,
	}
}