package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"styx/internal/config"
	"styx/internal/database"
	"styx/internal/server"
	"styx/internal/webhooks"
	proto_billing "styx/proto/billing"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server handles the orchestration of gRPC and HTTP servers
type Server struct {
	config         *config.Config
	grpcServer     *grpc.Server
	httpServer     *http.Server
	db             *database.Repository
	billingService *server.BillingService
	webhookHandler *webhooks.StripeWebhookHandler
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize database connection
	db, err := initDatabase(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize billing service
	billingService := server.NewBillingService(db, cfg.StripeSecretKey)

	// Initialize webhook handler
	webhookHandler := webhooks.NewStripeWebhookHandler(db, cfg.StripeSecretKey, cfg.StripeWebhookSecret)

	return &Server{
		config:         cfg,
		db:             db,
		billingService: billingService,
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

// StartGRPCServer starts the gRPC server
func (s *Server) StartGRPCServer() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %d: %w", s.config.GRPCPort, err)
	}

	// Create gRPC server with reflection for debugging
	s.grpcServer = grpc.NewServer()
	
	// Register billing service
	billing.RegisterBillingServiceServer(s.grpcServer, s.billingService)
	
	// Enable reflection for debugging
	reflection.Register(s.grpcServer)

	log.Printf("Starting gRPC server on port %d", s.config.GRPCPort)
	
	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	return nil
}

// StartHTTPServer starts the HTTP server for webhooks and health checks
func (s *Server) StartHTTPServer() error {
	mux := http.NewServeMux()
	
	// Set up routes
	s.setupRoutes(mux)
	
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

// setupRoutes sets up HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// Health check endpoint
	mux.HandleFunc("/health", s.healthCheckHandler)
	
	// Stripe webhook endpoint
	s.webhookHandler.SetupRoutes(mux)
	
	// Debug endpoint (development only)
	if true { // Default to enabled for now
		mux.HandleFunc("/debug", s.debugHandler)
	}
}

// debugHandler provides debug information
func (s *Server) debugHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	info := map[string]interface{}{
		"service": "billing-service",
		"status":  "running",
		"time":    time.Now().UTC().Format(time.RFC3339),
	}
	
	jsonData, _ := json.Marshal(info)
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// healthCheckHandler handles health check requests
func (s *Server) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Check webhook handler health
	if err := s.webhookHandler.HealthCheck(); err != nil {
		log.Printf("Health check failed: webhook handler error: %v", err)
		http.Error(w, `{"status": "unhealthy", "error": "webhook handler failed"}`, http.StatusServiceUnavailable)
		return
	}

	// Return healthy status
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy", "timestamp": "` + time.Now().UTC().Format(time.RFC3339) + `"}`))
}

// Start starts all servers
func (s *Server) Start() error {
	log.Printf("Starting billing service")
	
	// Start servers
	if err := s.StartGRPCServer(); err != nil {
		return fmt.Errorf("failed to start gRPC server: %w", err)
	}

	if err := s.StartHTTPServer(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	log.Println("All servers started successfully")
	return nil
}

// Stop gracefully shuts down all servers
func (s *Server) Stop() error {
	log.Println("Shutting down servers...")

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

	// Shutdown gRPC server
	if s.grpcServer != nil {
		log.Println("Shutting down gRPC server...")
		s.grpcServer.GracefulStop()
	}

	log.Println("Servers shutdown complete")
	return nil
}

// Run runs the server with graceful shutdown handling
func (s *Server) Run() error {
	// Start servers
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start servers: %w", err)
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
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Print configuration (remove sensitive data)
	log.Printf("Configuration loaded:")
	log.Printf("  HTTP Port: %d", cfg.HTTPPort)
	log.Printf("  gRPC Port: %d", cfg.GRPCPort)
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
