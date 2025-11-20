package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DraconDev/go-stripe-ms/internal/config"
	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers"
	"github.com/DraconDev/go-stripe-ms/internal/webhooks"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

// Server handles the HTTP-only billing service
type Server struct {
	config         *config.Config
	httpServer     *http.Server
	db             *database.Repository
	apiServer      *handlers.HTTPServer
	webhookHandler *webhooks.StripeWebhookHandler
}

// NewServer creates a new HTTP-only server instance
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize database connection
	db, err := initDatabase(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize HTTP API server
	apiServer := handlers.NewHTTPServer(db, cfg.StripeSecretKey)

	// Initialize webhook handler
	webhookHandler := webhooks.NewStripeWebhookHandler(db, cfg.StripeSecretKey, cfg.StripeWebhookSecret)

	return &Server{
		config:         cfg,
		db:             db,
		apiServer:      apiServer,
		webhookHandler: webhookHandler,
	}, nil
}

// initDatabase initializes the database connection
func initDatabase(cfg *config.Config) (*database.Repository, error) {
	log.Printf("Using database URL: %s", cfg.DatabaseURL)

	// Connect to the actual database
	conn, err := pgx.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize database tables
	repo := database.NewRepository(conn)
	if err := repo.InitializeTables(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize database tables: %w", err)
	}

	log.Println("Database connection initialized successfully")

	return repo, nil
}

// StartHTTPServer starts the HTTP server with all endpoints
func (s *Server) StartHTTPServer() error {
	mux := http.NewServeMux()

	// Set up API routes
	s.setupAPIRoutes(mux)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.HTTPPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting HTTP server on port %d", s.config.HTTPPort)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// setupAPIRoutes sets up all HTTP routes for the billing API
func (s *Server) setupAPIRoutes(mux *http.ServeMux) {
	// Health check
// Root endpoint
	mux.HandleFunc("/", s.apiServer.RootHandler)
	
	mux.HandleFunc("/health", s.apiServer.HealthCheck)

	// Checkout endpoints - split by payment type
	mux.HandleFunc("POST /api/v1/checkout/subscription", s.apiServer.CreateSubscriptionCheckout)
	mux.HandleFunc("POST /api/v1/checkout/item", s.apiServer.CreateItemCheckout)
	mux.HandleFunc("POST /api/v1/checkout/cart", s.apiServer.CreateCartCheckout)

	// Billing API endpoints
	mux.HandleFunc("GET /api/v1/subscriptions/{user_id}/{product_id}", s.apiServer.GetSubscriptionStatus)
	mux.HandleFunc("POST /api/v1/portal", s.apiServer.CreateCustomerPortal)

	// Webhook endpoint
	s.webhookHandler.SetupRoutes(mux)

	// Debug endpoint (development only)
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}
	if env != "production" {
		mux.HandleFunc("/debug", s.debugHandler)
	}
}

// debugHandler provides debug information
func (s *Server) debugHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"service":      "billing-service",
		"status":       "running",
		"time":         time.Now().UTC().Format(time.RFC3339),
		"environment":  getEnvironment(),
		"database_url": "configured",
		"stripe_key":   "configured",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	jsonData, _ := json.Marshal(info)
	if _, err := w.Write(jsonData); err != nil {
		log.Printf("Error writing debug response: %v", err)
	}
}

// getEnvironment returns the current environment with default
func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		return "development"
	}
	return env
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting billing service (HTTP-only)")

	// Start HTTP server
	if err := s.StartHTTPServer(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	log.Println("HTTP server started successfully")
	return nil
}

// Stop gracefully shuts down the server
func (s *Server) Stop() error {
	log.Println("Shutting down server...")

	// Create context for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if s.httpServer != nil {
		log.Println("Shutting down HTTP server...")
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}

	log.Println("Server shutdown complete")
	return nil
}

// Run runs the server with graceful shutdown handling
func (s *Server) Run() error {
	// Start server
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Set up signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-quit
	log.Println("Received shutdown signal")

	// Graceful shutdown
	return s.Stop()
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Print configuration (remove sensitive data)
	log.Printf("Configuration loaded:")
	log.Printf("  HTTP Port: %d", cfg.HTTPPort)
	log.Printf("  Environment: %s", getEnvironment())
	log.Printf("  Log Level: %s", cfg.LogLevel)

	// Create and run server
	srv, err := NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Run with graceful shutdown
	if err := srv.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
