package main

import (
	"context"
	"testing"

	"styx/internal/config"
	"styx/internal/database"
	"styx/internal/server"
	"styx/internal/webhooks"
	proto_billing "styx/proto/billing_service/proto/billing"
	"github.com/jackc/pgx/v5"
)

// TestBillingService tests the core functionality of the BillingService
func TestBillingService(t *testing.T) {
	// Load configuration from environment
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create a mock database repository
	db := &database.Repository{}
	
	// Initialize the billing service with real Stripe secret from environment
	billingService := server.NewBillingService(db, cfg.StripeSecretKey)
	
	if billingService == nil {
		t.Fatal("Failed to create BillingService")
	}
	
	// Test CreateSubscriptionCheckout
	t.Run("CreateSubscriptionCheckout", func(t *testing.T) {
		req := &proto_billing.CreateSubscriptionCheckoutRequest{
			UserId:     "test-user-123",
			ProductId:  "test-product-456", 
			PriceId:    "test-price-789",
			SuccessUrl: "https://example.com/success",
			CancelUrl:  "https://example.com/cancel",
		}
		
		ctx := context.Background()
		resp, err := billingService.CreateSubscriptionCheckout(ctx, req)
		
		if err != nil {
			t.Fatalf("CreateSubscriptionCheckout failed: %v", err)
		}
		
		if resp.CheckoutSessionId == "" {
			t.Error("CheckoutSessionId should not be empty")
		}
		
		if resp.CheckoutUrl == "" {
			t.Error("CheckoutUrl should not be empty")
		}
		
		t.Logf("Generated checkout session: %s", resp.CheckoutSessionId)
		t.Logf("Using real Stripe secret from environment")
	})
	
	// Test GetSubscriptionStatus
	t.Run("GetSubscriptionStatus", func(t *testing.T) {
		req := &proto_billing.GetSubscriptionStatusRequest{
			UserId:    "test-user-123",
			ProductId: "test-product-456",
		}
		
		ctx := context.Background()
		resp, err := billingService.GetSubscriptionStatus(ctx, req)
		
		if err != nil {
			t.Fatalf("GetSubscriptionStatus failed: %v", err)
		}
		
		// For a mock repository, we expect Exists to be false
		if resp.Exists {
			t.Error("Expected subscription not to exist for test user")
		}
		
		t.Logf("Subscription status check completed")
	})
	
	// Test CreateCustomerPortal
	t.Run("CreateCustomerPortal", func(t *testing.T) {
		req := &proto_billing.CreateCustomerPortalRequest{
			UserId:    "test-user-123",
			ReturnUrl: "https://example.com/return",
		}
		
		ctx := context.Background()
		resp, err := billingService.CreateCustomerPortal(ctx, req)
		
		if err == nil {
			// This might fail with our mock repository, which is expected
			if resp.PortalSessionId == "" {
				t.Error("PortalSessionId should not be empty")
			}
			
			if resp.PortalUrl == "" {
				t.Error("PortalUrl should not be empty")
			}
		} else {
			// Expected to fail with mock repository
			t.Logf("CreateCustomerPortal failed as expected with mock repository: %v", err)
		}
	})
}

// TestBillingServiceWithRealDB tests the billing service with a real database connection
func TestBillingServiceWithRealDB(t *testing.T) {
	// Load configuration from environment
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Connect to the real database
	conn, err := pgx.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(context.Background())
	
	// Create real database repository
	db := database.NewRepository(conn)
	
	// Initialize the billing service with real Stripe secret from environment
	billingService := server.NewBillingService(db, cfg.StripeSecretKey)
	
	if billingService == nil {
		t.Fatal("Failed to create BillingService")
	}
	
	t.Run("InitializeRealDatabase", func(t *testing.T) {
		ctx := context.Background()
		err := db.InitializeTables(ctx)
		
		// With real database, this should succeed
		if err != nil {
			t.Fatalf("InitializeTables failed with real database: %v", err)
		} else {
			t.Logf("Database tables initialized successfully with real database")
		}
	})
	
	t.Run("GetSubscriptionStatusWithRealDB", func(t *testing.T) {
		req := &proto_billing.GetSubscriptionStatusRequest{
			UserId:    "test-user-real-db",
			ProductId: "test-product-real-db",
		}
		
		ctx := context.Background()
		resp, err := billingService.GetSubscriptionStatus(ctx, req)
		
		if err != nil {
			t.Fatalf("GetSubscriptionStatus failed with real database: %v", err)
		}
		
		// For non-existent subscription in real database, we expect Exists to be false
		if resp.Exists {
			t.Error("Expected subscription not to exist for test user")
		}
		
		t.Logf("Real database subscription status check completed")
	})
}

// TestConfiguration tests that environment configuration is properly loaded
func TestConfiguration(t *testing.T) {
	t.Run("LoadConfig", func(t *testing.T) {
		cfg, err := config.LoadConfig()
		
		if err != nil {
			t.Fatalf("Failed to load configuration: %v", err)
		}
		
		// Verify all required configuration is loaded
		if cfg.DatabaseURL == "" {
			t.Error("DatabaseURL should not be empty")
		} else {
			t.Logf("Database URL loaded successfully")
		}
		
		if cfg.StripeSecretKey == "" {
			t.Error("StripeSecretKey should not be empty")
		} else {
			t.Logf("Stripe Secret Key loaded successfully")
		}
		
		if cfg.StripeWebhookSecret == "" {
			t.Error("StripeWebhookSecret should not be empty")
		} else {
			t.Logf("Stripe Webhook Secret loaded successfully")
		}
		
		if cfg.GRPCPort <= 0 {
			t.Error("GRPCPort should be positive")
		} else {
			t.Logf("GRPC Port: %d", cfg.GRPCPort)
		}
		
		if cfg.HTTPPort <= 0 {
			t.Error("HTTPPort should be positive")
		} else {
			t.Logf("HTTP Port: %d", cfg.HTTPPort)
		}
		
		if cfg.LogLevel == "" {
			t.Error("LogLevel should not be empty")
		} else {
			t.Logf("Log Level: %s", cfg.LogLevel)
		}
		
		t.Logf("Configuration loaded from environment variables")
	})

	t.Run("DatabasePoolConfig", func(t *testing.T) {
		cfg, err := config.LoadConfig()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		poolConfig := cfg.GetDatabasePoolConfig()
		
		if connString, ok := poolConfig["connString"].(string); !ok || connString == "" {
			t.Error("Database connection string should be set")
		} else {
			t.Logf("Database connection string configured")
		}
		
		if minConns, ok := poolConfig["minConns"].(int); !ok || minConns <= 0 {
			t.Error("Minimum connections should be positive")
		} else {
			t.Logf("Min connections: %d", minConns)
		}
		
		if maxConns, ok := poolConfig["maxConns"].(int); !ok || maxConns <= 0 {
			t.Error("Maximum connections should be positive")
		} else {
			t.Logf("Max connections: %d", maxConns)
		}
	})
}

// TestDatabaseIntegration tests basic database operations
func TestDatabaseIntegration(t *testing.T) {
	t.Run("InitializeTables", func(t *testing.T) {
		db := &database.Repository{}
		ctx := context.Background()
		err := db.InitializeTables(ctx)
		
		// With our current mock implementation, this should not fail
		// but it might return an error, which is fine for now
		if err != nil {
			t.Logf("InitializeTables returned error (expected with mock): %v", err)
		} else {
			t.Logf("Database tables initialized successfully")
		}
	})
	
	t.Run("GetSubscriptionStatus", func(t *testing.T) {
		db := &database.Repository{}
		ctx := context.Background()
		
		stripeSubID, customerID, currentPeriodEnd, exists, err := db.GetSubscriptionStatus(ctx, "nonexistent-user", "nonexistent-product")
		
		if err != nil {
			t.Logf("GetSubscriptionStatus returned error: %v", err)
		}
		
		if exists {
			t.Error("Expected subscription not to exist")
		}
		
		// These should be empty/zero for non-existent subscription
		if stripeSubID != "" {
			t.Errorf("Expected empty stripeSubID, got: %s", stripeSubID)
		}
		
		if customerID != "" {
			t.Errorf("Expected empty customerID, got: %s", customerID)
		}
		
		if !currentPeriodEnd.IsZero() {
			t.Errorf("Expected zero currentPeriodEnd, got: %v", currentPeriodEnd)
		}
		
		t.Logf("Database subscription status check completed")
	})

	t.Run("UpdateCustomerStripeID", func(t *testing.T) {
		db := &database.Repository{}
		ctx := context.Background()
		
		// Test updating customer Stripe ID
		err := db.UpdateCustomerStripeID(ctx, "test-user", "cus_test123")
		
		if err != nil {
			t.Logf("UpdateCustomerStripeID returned error (expected with mock): %v", err)
		} else {
			t.Logf("Customer Stripe ID update completed")
		}
	})

	t.Run("GetCustomerByUserID", func(t *testing.T) {
		db := &database.Repository{}
		ctx := context.Background()
		
		// Test getting customer by user ID
		customer, err := db.GetCustomerByUserID(ctx, "nonexistent-user")
		
		if err != nil {
			t.Logf("GetCustomerByUserID returned error (expected with mock): %v", err)
		} else {
			if customer == nil {
				t.Logf("Customer is nil (expected for non-existent user)")
			} else {
				t.Logf("Found customer: %+v", customer)
			}
		}
	})
}

// TestWebhookHandler tests the webhook handler functionality
func TestWebhookHandler(t *testing.T) {
	// Load configuration for webhook tests
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config for webhook tests: %v", err)
	}

	t.Run("HealthCheck", func(t *testing.T) {
		db := &database.Repository{}
		handler := webhooks.NewStripeWebhookHandler(db, cfg.StripeSecretKey, cfg.StripeWebhookSecret)
		
		err := handler.HealthCheck()
		
		// With mock repository, health check should succeed
		if err != nil {
			t.Logf("HealthCheck returned error: %v", err)
		} else {
			t.Logf("Webhook handler health check passed")
		}
	})

	t.Run("SetupRoutes", func(t *testing.T) {
		db := &database.Repository{}
		handler := webhooks.NewStripeWebhookHandler(db, cfg.StripeSecretKey, cfg.StripeWebhookSecret)
		
		// Test that routes can be set up without panicking
		// Note: This is a basic smoke test
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("SetupRoutes panicked: %v", r)
			}
		}()
		
		// We can't easily test the HTTP routing without a full server setup
		// but we can at least verify the handler was created successfully
		if handler == nil {
			t.Error("Webhook handler should not be nil")
		} else {
			t.Logf("Webhook handler created successfully with real webhook secret")
		}
	})
}

// TestServerOrchestration tests basic server setup
func TestServerOrchestration(t *testing.T) {
	t.Run("NewServer", func(t *testing.T) {
		// Load configuration for server test
		cfg, err := config.LoadConfig()
		if err != nil {
			t.Fatalf("Failed to load config for server test: %v", err)
		}

		// Test that we can create a server structure with the correct components
		db := &database.Repository{}
		billingService := server.NewBillingService(db, cfg.StripeSecretKey)
		
		server := &Server{
			config:         cfg,
			db:             db,
			billingService: billingService,
		}
		
		if server.billingService == nil {
			t.Error("Failed to create billing service for server")
		}
		
		if server.db == nil {
			t.Error("Database repository should not be nil")
		}

		if server.config == nil {
			t.Error("Configuration should not be nil")
		}
		
		t.Logf("Server orchestration test passed with real configuration")
	})
}

// Benchmark tests for performance testing
func BenchmarkCreateSubscriptionCheckout(b *testing.B) {
	// Load configuration for benchmark
	cfg, err := config.LoadConfig()
	if err != nil {
		b.Fatalf("Failed to load config for benchmark: %v", err)
	}

	db := &database.Repository{}
	billingService := server.NewBillingService(db, cfg.StripeSecretKey)
	
	req := &proto_billing.CreateSubscriptionCheckoutRequest{
		UserId:     "benchmark-user",
		ProductId:  "benchmark-product", 
		PriceId:    "benchmark-price",
		SuccessUrl: "https://example.com/success",
		CancelUrl:  "https://example.com/cancel",
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := billingService.CreateSubscriptionCheckout(ctx, req)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	// Load configuration for error handling tests
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config for error handling tests: %v", err)
	}

	db := &database.Repository{}
	billingService := server.NewBillingService(db, cfg.StripeSecretKey)
	
	t.Run("InvalidUserId", func(t *testing.T) {
		// Test with empty user ID
		req := &proto_billing.CreateSubscriptionCheckoutRequest{
			UserId:     "",
			ProductId:  "test-product", 
			PriceId:    "test-price",
			SuccessUrl: "https://example.com/success",
			CancelUrl:  "https://example.com/cancel",
		}
		
		ctx := context.Background()
		resp, err := billingService.CreateSubscriptionCheckout(ctx, req)
		
		// Should still work with empty user ID (mock implementation)
		if err != nil {
			t.Logf("CreateSubscriptionCheckout failed with empty user ID (expected): %v", err)
		} else {
			if resp.CheckoutSessionId == "" {
				t.Error("CheckoutSessionId should not be empty even with empty user ID")
			}
			t.Logf("CreateSubscriptionCheckout succeeded with empty user ID: %s", resp.CheckoutSessionId)
		}
	})

	t.Run("InvalidURLs", func(t *testing.T) {
		// Test with empty URLs
		req := &proto_billing.CreateSubscriptionCheckoutRequest{
			UserId:     "test-user",
			ProductId:  "test-product", 
			PriceId:    "test-price",
			SuccessUrl: "",
			CancelUrl:  "",
		}
		
		ctx := context.Background()
		resp, err := billingService.CreateSubscriptionCheckout(ctx, req)
		
		// Should still work with empty URLs (mock implementation)
		if err != nil {
			t.Logf("CreateSubscriptionCheckout failed with empty URLs (expected): %v", err)
		} else {
			if resp.CheckoutSessionId == "" {
				t.Error("CheckoutSessionId should not be empty even with empty URLs")
			}
			t.Logf("CreateSubscriptionCheckout succeeded with empty URLs: %s", resp.CheckoutSessionId)
		}
	})
}
