package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/webhooks"
	"github.com/stripe/stripe-go/v72"
)

func TestWebhookIntegration(t *testing.T) {
	database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
		// Setup test data
		project, customer, err := testDB.CreateTestData()
		if err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}

		// Initialize webhook handler
		// We use a dummy secret for testing since we're not verifying signatures in this test
		handler := webhooks.NewStripeWebhookHandler(testDB.Repo, "sk_test_dummy", "whsec_dummy")

		t.Run("Handle customer.subscription.created", func(t *testing.T) {
			// Create a mock event
			event := stripe.Event{
				Type: "customer.subscription.created",
				Data: &stripe.EventData{
					Raw: json.RawMessage(`{
						"id": "sub_test_created",
						"customer": {"id": "` + customer.StripeCustomerID + `"},
						"status": "active",
						"current_period_start": ` + createTimestamp(time.Now()) + `,
						"current_period_end": ` + createTimestamp(time.Now().Add(30*24*time.Hour)) + `,
						"items": {
							"data": [{
								"price": {
									"id": "price_test_123",
									"product": "prod_test_123"
								}
							}]
						}
					}`),
				},
			}

			// Create request
			bodyBytes, _ := json.Marshal(event)
			req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// We skip signature verification in the handler if secret is empty,
			// but here we initialized it. However, the handler logic for verification
			// is currently: "if !h.verifySignature(...)".
			// And verifySignature returns true in the current implementation.
			// So we don't need to generate a valid signature for now.

			w := httptest.NewRecorder()
			handler.HandleWebhook(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
			}

			// Check if subscription exists
			_, _, _, exists, err := testDB.Repo.GetSubscriptionStatus(req.Context(), project.ID, customer.UserID, "prod_test_123")
			if err != nil {
				t.Errorf("Failed to get subscription status: %v", err)
			}
			if !exists {
				t.Errorf("Expected subscription to be active/exist")
			}
		})

		t.Run("Handle customer.subscription.updated", func(t *testing.T) {
			// First ensure we have a subscription to update
			// (Re-using the one created above or creating a new one)
			// Let's update the one created above "sub_test_created"

			event := stripe.Event{
				Type: "customer.subscription.updated",
				Data: &stripe.EventData{
					Raw: json.RawMessage(`{
						"id": "sub_test_created",
						"status": "past_due",
						"current_period_end": ` + createTimestamp(time.Now().Add(30*24*time.Hour)) + `
					}`),
				},
			}

			bodyBytes, _ := json.Marshal(event)
			req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.HandleWebhook(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
			}

			// Verify status update
			// Note: GetSubscriptionStatus returns exists=true for any status if record exists.
			// To verify status change, we might need GetSubscriptionByStripeID
			sub, err := testDB.Repo.GetSubscriptionByStripeID(req.Context(), "sub_test_created")
			if err != nil {
				t.Fatalf("Failed to get subscription: %v", err)
			}
			if sub.Status != "past_due" {
				t.Errorf("Expected status 'past_due', got '%s'", sub.Status)
			}
		})

		t.Run("Handle customer.subscription.deleted", func(t *testing.T) {
			event := stripe.Event{
				Type: "customer.subscription.deleted",
				Data: &stripe.EventData{
					Raw: json.RawMessage(`{
						"id": "sub_test_created",
						"current_period_end": ` + createTimestamp(time.Now()) + `
					}`),
				},
			}

			bodyBytes, _ := json.Marshal(event)
			req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.HandleWebhook(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
			}

			// Verify cancellation
			sub, err := testDB.Repo.GetSubscriptionByStripeID(req.Context(), "sub_test_created")
			if err != nil {
				t.Fatalf("Failed to get subscription: %v", err)
			}
			if sub.Status != "canceled" {
				t.Errorf("Expected status 'canceled', got '%s'", sub.Status)
			}
		})
	})
}

func createTimestamp(t time.Time) string {
	return fmt.Sprintf("%d", t.Unix())
}
