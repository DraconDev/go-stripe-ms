package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Database
	DatabaseURL string

	// Stripe Configuration
	StripeSecretKey     string
	StripeWebhookSecret string

	// Server Configuration
	HTTPPort int

	// Logging
	LogLevel string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		// Database
		DatabaseURL: getEnvOrError("DATABASE_URL"),

		// Stripe
		StripeSecretKey:     getEnvOrError("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: getEnvOrError("STRIPE_WEBHOOK_SECRET"),

		// Ports
		HTTPPort: getEnvAsInt("HTTP_PORT", 8080),

		// Logging
		LogLevel: getEnvOrError("LOG_LEVEL"),
	}

	return cfg, nil
}

// getEnv retrieves an environment variable, returns empty string if not set
func getEnv(key string) string {
	return os.Getenv(key)
}

// getEnvOrError retrieves an environment variable and logs an error if missing
func getEnvOrError(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Required environment variable %s is not set", key)
	}
	return value
}

// getEnvAsInt retrieves an environment variable as an integer, returns default if not set or invalid
func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Warning: Invalid integer value for %s, using default: %d", key, defaultValue)
		return defaultValue
	}

	return intValue
}

// GetDatabasePoolConfig returns pgx pool configuration
func (c *Config) GetDatabasePoolConfig() map[string]interface{} {
	return map[string]interface{}{
		"connString":        c.DatabaseURL,
		"minConns":          5,
		"maxConns":          25,
		"maxConnLifetime":   time.Hour,
		"maxConnIdleTime":   time.Minute * 30,
		"healthCheckPeriod": time.Minute * 5,
	}
}
