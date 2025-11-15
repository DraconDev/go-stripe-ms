package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration for production-ready multi-tenant service
type Config struct {
	// Multi-tenancy
	TenantID string
	Environment string

	// Database
	DatabaseURL string

	// Stripe Configuration
	StripeSecretKey     string
	StripeWebhookSecret string

	// Server Configuration
	HTTPPort int
	ReadTimeout time.Duration
	WriteTimeout time.Duration
	IdleTimeout time.Duration

	// Security & Authentication
	APIKeys map[string]APIKeyConfig
	CORSAllowedOrigins []string

	// Rate Limiting
	RateLimitRequestsPerMinute int
	RateLimitBurstSize int

	// Logging
	LogLevel string
	EnableStructuredLogging bool

	// Monitoring
	EnableMetrics bool
	MetricsPort int

	// Health Checks
	HealthCheckTimeout time.Duration
	DatabaseHealthCheck bool

	// Retry Configuration
	WebhookRetryAttempts int
	WebhookRetryDelay time.Duration
}

// APIKeyConfig holds API key configuration for different tenants/endpoints
type APIKeyConfig struct {
	Name      string   `json:"name"`
	KeyHash   string   `json:"-"`
	Scopes    []string `json:"scopes"`
	RateLimit int      `json:"rate_limit"`
	Enabled   bool     `json:"enabled"`
}

// LoadConfig loads configuration from environment variables with multi-tenant support
func LoadConfig() (*Config, error) {
	cfg := &Config{
		// Multi-tenancy
		TenantID: getEnvOrDefault("TENANT_ID", "default"),
		Environment: getEnvOrDefault("ENVIRONMENT", "development"),
		
		// Database
		DatabaseURL: getEnvOrError("DATABASE_URL"),
		
		// Stripe
		StripeSecretKey:     getEnvOrError("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: getEnvOrError("STRIPE_WEBHOOK_SECRET"),
		
		// Server Configuration
		HTTPPort: getEnvAsInt("HTTP_PORT", 8080),
		ReadTimeout: time.Duration(getEnvAsInt("READ_TIMEOUT_SECONDS", 30)) * time.Second,
		WriteTimeout: time.Duration(getEnvAsInt("WRITE_TIMEOUT_SECONDS", 30)) * time.Second,
		IdleTimeout: time.Duration(getEnvAsInt("IDLE_TIMEOUT_SECONDS", 120)) * time.Second,

		// Security & Authentication
		APIKeys: parseAPIKeys(),
		CORSAllowedOrigins: parseCSVEnv("CORS_ALLOWED_ORIGINS", []string{"*"}),

		// Rate Limiting
		RateLimitRequestsPerMinute: getEnvAsInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 60),
		RateLimitBurstSize: getEnvAsInt("RATE_LIMIT_BURST_SIZE", 10),

		// Logging
		LogLevel: getEnvOrDefault("LOG_LEVEL", "info"),
		EnableStructuredLogging: getEnvAsBool("ENABLE_STRUCTURED_LOGGING", true),

		// Monitoring
		EnableMetrics: getEnvAsBool("ENABLE_METRICS", true),
		MetricsPort: getEnvAsInt("METRICS_PORT", 9090),

		// Health Checks
		HealthCheckTimeout: time.Duration(getEnvAsInt("HEALTH_CHECK_TIMEOUT_SECONDS", 5)) * time.Second,
		DatabaseHealthCheck: getEnvAsBool("DATABASE_HEALTH_CHECK", true),

		// Retry Configuration
		WebhookRetryAttempts: getEnvAsInt("WEBHOOK_RETRY_ATTEMPTS", 3),
		WebhookRetryDelay: time.Duration(getEnvAsInt("WEBHOOK_RETRY_DELAY_SECONDS", 5)) * time.Second,
	}

	return cfg, nil
}

// getEnv retrieves an environment variable, returns empty string if not set
func getEnv(key string) string {
	return os.Getenv(key)
}

// getEnvOrDefault retrieves an environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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

// GetDatabasePoolConfig returns pgx pool configuration
func (c *Config) GetDatabasePoolConfig() map[string]interface{} {
	maxConns := 25
	if c.IsProduction() {
		maxConns = 50
	}
	
	return map[string]interface{}{
		"connString":          c.DatabaseURL,
		"minConns":            5,
		"maxConns":            maxConns,
		"maxConnLifetime":     time.Hour,
		"maxConnIdleTime":     time.Minute * 30,
		"healthCheckPeriod":   time.Minute * 5,
	}
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
