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

// getEnvOrDefault retrieves an environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsBool retrieves an environment variable as a boolean
func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("Warning: Invalid boolean value for %s, using default: %t", key, defaultValue)
		return defaultValue
	}
	
	return boolValue
}

// parseCSVEnv parses comma-separated environment variable into string slice
func parseCSVEnv(key string, defaultValues []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValues
	}
	
	values := strings.Split(value, ",")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	
	return values
}

// parseAPIKeys parses API key configuration from environment
func parseAPIKeys() map[string]APIKeyConfig {
	apiKeys := make(map[string]APIKeyConfig)
	
	// Parse main API key
	mainKey := getEnv("API_KEY")
	if mainKey != "" {
		apiKeys["main"] = APIKeyConfig{
			Name:      "Main API Key",
			KeyHash:   hashAPIKey(mainKey),
			Scopes:    parseCSVEnv("API_KEY_SCOPES", []string{"billing", "admin"}),
			RateLimit: getEnvAsInt("API_KEY_RATE_LIMIT", 100),
			Enabled:   true,
		}
	}
	
	// Parse additional API keys
	for i := 1; i <= 5; i++ {
		keyName := fmt.Sprintf("API_KEY_%d", i)
		keyValue := os.Getenv(keyName)
		if keyValue != "" {
			scopesEnv := fmt.Sprintf("API_KEY_%d_SCOPES", i)
			rateLimitEnv := fmt.Sprintf("API_KEY_%d_RATE_LIMIT", i)
			
			apiKeys[fmt.Sprintf("key_%d", i)] = APIKeyConfig{
				Name:      fmt.Sprintf("API Key %d", i),
				KeyHash:   hashAPIKey(keyValue),
				Scopes:    parseCSVEnv(scopesEnv, []string{"billing"}),
				RateLimit: getEnvAsInt(rateLimitEnv, 60),
				Enabled:   true,
			}
		}
	}
	
	return apiKeys
}

// hashAPIKey creates a SHA256 hash of an API key for secure storage
func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// ValidateConfig validates the configuration for required fields
func (c *Config) ValidateConfig() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("database URL is required")
	}
	
	if c.StripeSecretKey == "" {
		return fmt.Errorf("Stripe secret key is required")
	}
	
	if c.StripeWebhookSecret == "" {
		return fmt.Errorf("Stripe webhook secret is required")
	}
	
	if c.HTTPPort <= 0 || c.HTTPPort > 65535 {
		return fmt.Errorf("HTTP port must be between 1 and 65535")
	}
	
	return nil
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// GetAPILimits returns rate limiting configuration for a given scope
func (c *Config) GetAPILimits(scope string) (int, int) {
	// Check if specific API key has custom limits
	for _, keyConfig := range c.APIKeys {
		for _, keyScope := range keyConfig.Scopes {
			if keyScope == scope {
				return keyConfig.RateLimit, c.RateLimitBurstSize
			}
		}
	}
	
	// Return default limits
	return c.RateLimitRequestsPerMinute, c.RateLimitBurstSize
}
}
