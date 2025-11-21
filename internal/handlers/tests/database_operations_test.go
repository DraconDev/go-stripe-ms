package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/google/uuid"
)

// TestDatabaseOperationsIntegration tests database operations directly
func TestDatabaseOperationsIntegration(t *testing.T) {
	database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
		ctx := context.Background()

		// Create a test project
		projectID := uuid.New()
		project := &database.Project{
			ID:        projectID,
			Name:      "Test Project Ops",
			APIKey:    "sk_test_ops_" + uuid.New().String(),
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := testDB.CreateTestProject(project); err != nil {
			t.Fatalf("Failed to create test project: %v", err)
		}

		// Generate unique identifiers for this test run to avoid conflicts
		timestamp := time.Now().Unix()
		testUserID1 := fmt.Sprintf("test_user_%d_1", timestamp)
		testUserID2 := fmt.Sprintf("test_user_%d_2", timestamp)
		testUserID3 := fmt.Sprintf("test_user_%d_3", timestamp)
		stripeCustomerID1 := fmt.Sprintf("cus_test_%d_1", timestamp)
		stripeCustomerID2 := fmt.Sprintf("cus_test_%d_2", timestamp)
		stripeSubID := fmt.Sprintf("sub_test_%d", timestamp)

		t.Run("FindOrCreateStripeCustomer", func(t *testing.T) {
			// Test customer creation
			_, err := testDB.Repo.FindOrCreateStripeCustomer(ctx, projectID,
				testUserID1, fmt.Sprintf("test%d@example.com", timestamp))
			if err != nil {
				t.Fatalf("Failed to create customer: %v", err)
			}

			// Test customer retrieval
			customer, err := testDB.Repo.GetCustomerByUserID(ctx, projectID, testUserID1)
			if err != nil {
				t.Fatalf("Failed to get customer: %v", err)
			}
			if customer == nil {
				t.Fatal("Customer not found")
			}
			if customer.UserID != testUserID1 {
				t.Errorf("Expected user ID '%s', got '%s'", testUserID1, customer.UserID)
			}
		})

		t.Run("CreateSubscription", func(t *testing.T) {
			// First create a customer
			_, err := testDB.Repo.FindOrCreateStripeCustomer(ctx, projectID,
				testUserID2, fmt.Sprintf("test%d_2@example.com", timestamp))
			if err != nil {
				t.Fatalf("Failed to create customer: %v", err)
			}

			// Update customer with Stripe ID (simulating the flow)
			err = testDB.Repo.UpdateCustomerStripeID(ctx, projectID, testUserID2, stripeCustomerID1)
			if err != nil {
				t.Fatalf("Failed to update customer Stripe ID: %v", err)
			}

			// Create subscription
			now := time.Now()
			err = testDB.Repo.CreateSubscription(ctx, projectID, stripeCustomerID1, stripeSubID,
				"pro_plan", "price_789", testUserID2, "active", now, now.AddDate(0, 0, 30))
			if err != nil {
				t.Fatalf("Failed to create subscription: %v", err)
			}

			// Retrieve subscription
			stripeSubIDRetrieved, status, _, exists, err := testDB.Repo.GetSubscriptionStatus(ctx, projectID,
				testUserID2, "pro_plan")
			if err != nil {
				t.Fatalf("Failed to get subscription status: %v", err)
			}
			if !exists {
				t.Fatal("Subscription not found")
			}
			if stripeSubIDRetrieved != stripeSubID {
				t.Errorf("Expected Stripe subscription ID '%s', got '%s'", stripeSubID, stripeSubIDRetrieved)
			}
			if status != "active" {
				t.Errorf("Expected status 'active', got '%s'", status)
			}
		})

		t.Run("UpdateSubscriptionStatus", func(t *testing.T) {
			// Create customer and subscription first
			_, err := testDB.Repo.FindOrCreateStripeCustomer(ctx, projectID,
				testUserID3, fmt.Sprintf("test%d_3@example.com", timestamp))
			if err != nil {
				t.Fatalf("Failed to create customer: %v", err)
			}

			err = testDB.Repo.UpdateCustomerStripeID(ctx, projectID, testUserID3, stripeCustomerID2)
			if err != nil {
				t.Fatalf("Failed to update customer Stripe ID: %v", err)
			}

			now := time.Now()
			err = testDB.Repo.CreateSubscription(ctx, projectID, stripeCustomerID2, stripeSubID,
				"enterprise_plan", "price_999", testUserID3, "active", now, now.AddDate(0, 0, 30))
			if err != nil {
				t.Fatalf("Failed to create subscription: %v", err)
			}

			// Update subscription status
			newPeriodEnd := now.AddDate(0, 1, 0)
			err = testDB.Repo.UpdateSubscriptionStatus(ctx, stripeSubID, "canceled", newPeriodEnd)
			if err != nil {
				t.Fatalf("Failed to update subscription status: %v", err)
			}

			// Verify update
			_, status, periodEnd, exists, err := testDB.Repo.GetSubscriptionStatus(ctx, projectID,
				testUserID3, "enterprise_plan")
			if err != nil {
				t.Fatalf("Failed to get subscription status: %v", err)
			}
			if !exists {
				t.Fatal("Subscription not found")
			}
			if status != "canceled" {
				t.Errorf("Expected status 'canceled', got '%s'", status)
			}
			if periodEnd != newPeriodEnd {
				t.Errorf("Expected period end %v, got %v", newPeriodEnd, periodEnd)
			}
		})
	})
}
