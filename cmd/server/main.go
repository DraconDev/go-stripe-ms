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
	handlerSvc "github.com/DraconDev/go-stripe-ms/internal/handlers"
	"github.com/DraconDev/go-stripe-ms/internal/middleware"
	"github.com/DraconDev/go-stripe-ms/internal/webhooks"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

// Server handles the HTTP-only billing service
type Server struct {
	config         *config.Config
	httpServer     *http.Server
	db             *database.Repository
	apiServer      *handlerSvc.HTTPServer
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
	apiServer := handlerSvc.NewHTTPServer(db, cfg.StripeSecretKey)

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
	// Create API key middleware
	apiKeyAuth := middleware.NewAPIKeyAuth(s.config.APIKey)

	// Public endpoints (no authentication required)
	mux.HandleFunc("/", s.apiServer.RootHandler)
	mux.HandleFunc("/health", s.apiServer.HealthCheck)

	// API Documentation endpoints (public)
	mux.HandleFunc("/openapi.json", s.openAPIHandler)
	mux.HandleFunc("/docs", s.docsHandler)

	// Webhook endpoint (authenticated by Stripe signature, not API key)
	s.webhookHandler.SetupRoutes(mux)

	// Protected API endpoints (require X-API-Key header)
	// Wrap each handler with the middleware
	mux.Handle("POST /api/v1/checkout/subscription",
		apiKeyAuth.Middleware(http.HandlerFunc(s.apiServer.CreateSubscriptionCheckout)))
	mux.Handle("POST /api/v1/checkout/item",
		apiKeyAuth.Middleware(http.HandlerFunc(s.apiServer.CreateItemCheckout)))
	mux.Handle("POST /api/v1/checkout/cart",
		apiKeyAuth.Middleware(http.HandlerFunc(s.apiServer.CreateCartCheckout)))
	mux.Handle("GET /api/v1/subscriptions/{user_id}/{product_id}",
		apiKeyAuth.Middleware(http.HandlerFunc(s.apiServer.GetSubscriptionStatus)))
	mux.Handle("POST /api/v1/portal",
		apiKeyAuth.Middleware(http.HandlerFunc(s.apiServer.CreateCustomerPortal)))

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
package main

import (
	"encoding/json"
	"net/http"
)

// OpenAPI/Swagger specification for the Payment MS
var openAPISpec = map[string]interface{}{
	"openapi": "3.0.0",
	"info": map[string]interface{}{
		"title":       "Payment Microservice API",
		"description": "Stripe-based payment processing service",
		"version":     "1.0.0",
	},
	"servers": []map[string]string{
		{"url": "http://localhost:9000", "description": "Local development"},
	},
	"security": []map[string][]string{
		{"ApiKeyAuth": {}},
	},
	"components": map[string]interface{}{
		"securitySchemes": map[string]interface{}{
			"ApiKeyAuth": map[string]string{
				"type": "apiKey",
				"in":   "header",
				"name": "X-API-Key",
			},
		},
		"schemas": map[string]interface{}{
			"SubscriptionCheckoutRequest": map[string]interface{}{
				"type":     "object",
				"required": []string{"user_id", "email", "product_id", "price_id", "success_url", "cancel_url"},
				"properties": map[string]interface{}{
					"user_id":     map[string]string{"type": "string", "example": "user_123"},
					"email":       map[string]string{"type": "string", "example": "customer@example.com"},
					"product_id":  map[string]string{"type": "string", "example": "prod_RZaVDAN6Uf4Qfb"},
					"price_id":    map[string]string{"type": "string", "example": "price_1QhEBSFhH6dwUiIH"},
					"success_url": map[string]string{"type": "string", "example": "https://yourapp.com/success"},
					"cancel_url":  map[string]string{"type": "string", "example": "https://yourapp.com/cancel"},
				},
			},
			"CheckoutResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"checkout_session_id": map[string]string{"type": "string"},
					"checkout_url":        map[string]string{"type": "string"},
				},
			},
		},
	},
	"paths": map[string]interface{}{
		"/health": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":  "Health check",
				"security": []map[string][]string{}, // No auth required
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Service is healthy",
					},
				},
			},
		},
		"/api/v1/checkout/subscription": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Create subscription checkout session",
				"description": "Creates a Stripe checkout session for a subscription",
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]string{
								"$ref": "#/components/schemas/SubscriptionCheckoutRequest",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Checkout session created",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]string{
									"$ref": "#/components/schemas/CheckoutResponse",
								},
							},
						},
					},
					"401": map[string]interface{}{
						"description": "Unauthorized - Invalid or missing API key",
					},
				},
			},
		},
		"/api/v1/checkout/item": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Create single item checkout",
				"description": "Creates a Stripe checkout session for a single item",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{"description": "Success"},
				},
			},
		},
		"/api/v1/checkout/cart": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Create cart checkout",
				"description": "Creates a Stripe checkout session for multiple items",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{"description": "Success"},
				},
			},
		},
		"/api/v1/subscriptions/{user_id}/{product_id}": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "Get subscription status",
				"description": "Retrieves the subscription status for a user and product",
				"parameters": []map[string]interface{}{
					{
						"name":     "user_id",
						"in":       "path",
						"required": true,
						"schema":   map[string]string{"type": "string"},
					},
					{
						"name":     "product_id",
						"in":       "path",
						"required": true,
						"schema":   map[string]string{"type": "string"},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{"description": "Subscription status"},
				},
			},
		},
		"/api/v1/portal": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Create customer portal session",
				"description": "Creates a Stripe customer portal session for managing subscriptions",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{"description": "Portal URL created"},
				},
			},
		},
	},
}

// OpenAPIHandler serves the OpenAPI specification
func (s *Server) openAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow CORS for docs
	json.NewEncoder(w).Encode(openAPISpec)
}
package main

import (
	"fmt"
	"net/http"
)

// docsHandler serves a simple HTML page with API documentation
func (s *Server) docsHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Payment MS API Documentation</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }
        h1 { color: #333; }
        h2 { color: #666; margin-top: 30px; }
        .endpoint { background: #f5f5f5; padding: 15px; margin: 10px 0; border-left: 4px solid #4CAF50; }
        .method { display: inline-block; padding: 5px 10px; border-radius: 3px; font-weight: bold; margin-right: 10px; }
        .post { background: #4CAF50; color: white; }
        .get { background: #2196F3; color: white; }
        code { background: #f0f0f0; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f5f5f5; padding: 15px; overflow-x: auto; border-radius: 5px; }
        .auth-required { color: #ff9800; font-weight: bold; }
    </style>
</head>
<body>
    <h1>🔐 Payment Microservice API</h1>
    <p>Stripe-based payment processing service</p>
    
    <h2>Authentication</h2>
    <p>All <code>/api/v1/*</code> endpoints require the <code>X-API-Key</code> header:</p>
    <pre>X-API-Key: your_api_key_here</pre>
    
    <h2>Base URL</h2>
    <pre>http://localhost:9000</pre>
    
    <h2>Endpoints</h2>
    
    <div class="endpoint">
        <span class="method get">GET</span>
        <code>/health</code>
        <p>Health check endpoint (no auth required)</p>
    </div>
    
    <div class="endpoint">
        <span class="method get">GET</span>
        <code>/openapi.json</code>
        <p>OpenAPI specification (no auth required)</p>
    </div>
    
    <div class="endpoint">
        <span class="method post">POST</span>
        <code>/api/v1/checkout/subscription</code>
        <span class="auth-required">🔒 Auth Required</span>
        <p>Create a subscription checkout session</p>
        <pre>{
  "user_id": "user_123",
  "email": "customer@example.com",
  "product_id": "prod_RZaVDAN6Uf4Qfb",
  "price_id": "price_1QhEBSFhH6dwUiIH",
  "success_url": "https://yourapp.com/success",
  "cancel_url": "https://yourapp.com/cancel"
}</pre>
    </div>
    
    <div class="endpoint">
        <span class="method post">POST</span>
        <code>/api/v1/checkout/item</code>
        <span class="auth-required">🔒 Auth Required</span>
        <p>Create a single item checkout session</p>
    </div>
    
    <div class="endpoint">
        <span class="method post">POST</span>
        <code>/api/v1/checkout/cart</code>
        <span class="auth-required">🔒 Auth Required</span>
        <p>Create a cart checkout session with multiple items</p>
    </div>
    
    <div class="endpoint">
        <span class="method get">GET</span>
        <code>/api/v1/subscriptions/{user_id}/{product_id}</code>
        <span class="auth-required">🔒 Auth Required</span>
        <p>Get subscription status for a user and product</p>
    </div>
    
    <div class="endpoint">
        <span class="method post">POST</span>
        <code>/api/v1/portal</code>
        <span class="auth-required">🔒 Auth Required</span>
        <p>Create a Stripe customer portal session</p>
    </div>
    
    <h2>Full Documentation</h2>
    <p>For complete API documentation, see:</p>
    <ul>
        <li><a href="/openapi.json">OpenAPI Specification (JSON)</a></li>
        <li><a href="https://github.com/yourusername/go-stripe-ms/blob/main/API_REQUESTS.md">API_REQUESTS.md</a></li>
    </ul>
    
    <h2>Example Request</h2>
    <pre>curl -X POST http://localhost:9000/api/v1/checkout/subscription \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "email": "customer@example.com",
    "product_id": "prod_RZaVDAN6Uf4Qfb",
    "price_id": "price_1QhEBSFhH6dwUiIH",
    "success_url": "https://yourapp.com/success",
    "cancel_url": "https://yourapp.com/cancel"
  }'</pre>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprint(w, html)
}
