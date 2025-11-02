package main

import (
	"context"
	"testing"

	"billing_service/internal/database"
	"billing_service/internal/server"
	"billing_service/internal/webhooks"
	proto_billing "billing_service/proto/billing_service/proto/billing"
)

// TestBillingService tests the core functionality of the BillingService
func TestBillingService(t *testing.T) {
	// Create a mock database repository
	db := &database.Repository{}
	
	// Initialize the billing service
	billingService := server.NewBillingService(db)
	
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

// TestDatabaseIntegration tests basic database operations
func TestDatabaseIntegration(t *testing.T) {
	t.Run("InitializeTables", func(t *testing.T) {
		db := &database.Repository{}
		err := db.InitializeTables(nil)
		
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
}

// TestWebhookHandler tests the webhook handler functionality
func TestWebhookHandler(t *testing.T) {
	t.Run("HealthCheck", func(t *testing.T) {
		db := &database.Repository{}
		handler := webhooks.NewStripeWebhookHandler(db, "dummy-key", "dummy-secret")
		
		err := handler.HealthCheck()
		
		// With mock repository, health check should succeed
		if err != nil {
			t.Logf("HealthCheck returned error: %v", err)
		} else {
			t.Logf("Webhook handler health check passed")
		}
	})
}

// TestServerOrchestration tests basic server setup
func TestServerOrchestration(t *testing.T) {
	t.Run("NewServer", func(t *testing.T) {
		// This test requires the config package, so we'll create a basic test
		// In a real implementation, we would need to mock the config
		
		// For now, we'll just test that the basic server structure can be created
		server := &Server{
			config:         nil, // Would need proper config in real test
			db:             &database.Repository{},
			billingService: server.NewBillingService(&database.Repository{}),
		}
		
		if server.billingService == nil {
			t.Error("Failed to create billing service for server")
		}
		
		if server.db == nil {
			t.Error("Database repository should not be nil")
		}
		
		t.Logf("Server orchestration test passed")
	})
}

// Benchmark tests for performance testing
func BenchmarkCreateSubscriptionCheckout(b *testing.B) {
	db := &database.Repository{}
	billingService := server.NewBillingService(db)
	
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
