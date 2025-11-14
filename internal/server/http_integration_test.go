package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"styx/internal/database"
)

// TestCreateSubscriptionCheckoutIntegration tests with real database
func TestCreateSubscriptionCheckoutIntegration(t *testing.T) {
	database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
		// Setup test data
		if err := testDB.CreateTestData(); err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}

		// Create HTTP server with real database
		server := NewHTTPServer(testDB.Repo, "sk_test_123")

		tests := []struct {
			name               string
			requestBody        map[string]interface{}
			expectedStatusCode int
			expectedResponse   map[string]interface{}
			expectedError      string
		}{
			{
				name: "Valid checkout request with real database",
				requestBody: map[string]interface{}{
					"user_id":     "test_user_123",
					"email":       "test@example.com",
					"product_id":  "premium_plan",
					"price_id":    "price_test123",
					"success_url": "https://example.com/success",
					"cancel_url":  "https://example.com/cancel",
				},
				expectedStatusCode: http.StatusOK,
			},
			{
				name: "Missing user_id",
				requestBody: map[string]interface{}{
					"email":       "test@example.com",
					"product_id":  "premium_plan",
					"price_id":    "price_test123",
					"success_url": "https://example.com/success",
					"cancel_url":  "https://example.com/cancel",
				},
				expectedStatusCode: http.StatusBadRequest,
				expectedError:      "Missing required fields",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create request
				bodyBytes, _ := json.Marshal(tt.requestBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", 
					bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")

				// Execute
				w := httptest.NewRecorder()
				server.CreateSubscriptionCheckout(w, req)

				// Assert
				if w.Code != tt.expectedStatusCode {
					t.Errorf("Expected status code %d, got %d", 
						tt.expectedStatusCode, w.Code)
				}

				if tt.expectedError != "" {
					var errorResponse map[string]string
					if err := json.Unmarshal(w.Body.Bytes(), &errorResponse); err != nil {
						t.Errorf("Failed to unmarshal error response: %v", err)
					} else if errorResponse["error"] != tt.expectedError {
						t.Errorf("Expected error '%s', got '%s'", 
							tt.expectedError, errorResponse["error"])
					}
				}

				// Check Content-Type header
				if w.Header().Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type 'application/json', got '%s'", 
						w.Header().Get("Content-Type"))
				}
			})
		}
	})
}

// TestGetSubscriptionStatusIntegration tests subscription status retrieval with real database
func TestGetSubscriptionStatusIntegration(t *testing.T) {
	database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
		// Setup test data
		if err := testDB.CreateTestData(); err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}

		// Create HTTP server with real database
		server := NewHTTPServer(testDB.Repo, "sk_test_123")

		tests := []struct {
			name               string
			path               string
			expectedStatusCode int
			expectedResponse   map[string]interface{}
			expectedError      string
		}{
			{
				name:               "Valid subscription request with real data",
				path:               "/api/v1/subscriptions/test_user_123/premium_plan",
				expectedStatusCode: http.StatusOK,
			},
			{
				name:               "Non-existent subscription",
				path:               "/api/v1/subscriptions/nonexistent/premium_plan",
				expectedStatusCode: http.StatusOK,
				expectedResponse: map[string]interface{}{
					"exists": false,
				},
			},
			{
				name:               "Invalid URL format",
				path:               "/api/v1/subscriptions/test_user_123",
				expectedStatusCode: http.StatusBadRequest,
				expectedError:      "Invalid URL format",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create request
				req := httptest.NewRequest(http.MethodGet, tt.path, nil)

				// Execute
				w := httptest.NewRecorder()
				server.GetSubscriptionStatus(w, req)

				// Assert
				if w.Code != tt.expectedStatusCode {
					t.Errorf("Expected status code %d, got %d", 
						tt.expectedStatusCode, w.Code)
				}

				if tt.expectedError != "" {
					var errorResponse map[string]string
					if err := json.Unmarshal(w.Body.Bytes(), &errorResponse); err != nil {
						t.Errorf("Failed to unmarshal error response: %v", err)
					} else if errorResponse["error"] != tt.expectedError {
						t.Errorf("Expected error '%s', got '%s'", 
							tt.expectedError, errorResponse["error"])
					}
				}

				if tt.expectedResponse != nil {
					var response map[string]interface{}
					if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
						t.Errorf("Failed to unmarshal response: %v", err)
					} else {
						for key, expectedValue := range tt.expectedResponse {
							if actualValue, exists := response[key]; !exists || actualValue != expectedValue {
								t.Errorf("Expected response[%s] = %v, got %v", 
									key, expectedValue, actualValue)
							}
						}
					}
				}

				// Check Content-Type header
				if w.Header().Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type 'application/json', got '%s'", 
						w.Header().Get("Content-Type"))
				}
			})
		}
	})
}

// TestCreateCustomerPortalIntegration tests customer portal creation with real database
func TestCreateCustomerPortalIntegration(t *testing.T) {
	database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
		// Setup test data
		if err := testDB.CreateTestData(); err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}

		// Create HTTP server with real database
		server := NewHTTPServer(testDB.Repo, "sk_test_123")

		tests := []struct {
			name               string
			requestBody        map[string]interface{}
			expectedStatusCode int
			expectedError      string
		}{
			{
				name: "Valid portal request with real database",
				requestBody: map[string]interface{}{
					"user_id":    "test_user_123",
					"return_url": "https://example.com/account",
				},
				expectedStatusCode: http.StatusOK,
			},
			{
				name: "Missing user_id",
				requestBody: map[string]interface{}{
					"return_url": "https://example.com/account",
				},
				expectedStatusCode: http.StatusBadRequest,
				expectedError:      "Missing required fields",
			},
			{
				name: "Customer not found",
				requestBody: map[string]interface{}{
					"user_id":    "nonexistent",
					"return_url": "https://example.com/account",
				},
				expectedStatusCode: http.StatusNotFound,
				expectedError:      "Customer not found",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create request
				bodyBytes, _ := json.Marshal(tt.requestBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/portal", 
					bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")

				// Execute
				w := httptest.NewRecorder()
				server.CreateCustomerPortal(w, req)

				// Assert
				if w.Code != tt.expectedStatusCode {
					t.Errorf("Expected status code %d, got %d", 
						tt.expectedStatusCode, w.Code)
				}

				if tt.expectedError != "" {
					var errorResponse map[string]string
					if err := json.Unmarshal(w.Body.Bytes(), &errorResponse); err != nil {
						t.Errorf("Failed to unmarshal error response: %v", err)
					} else if errorResponse["error"] != tt.expectedError {
						t.Errorf("Expected error '%s', got '%s'", 
							tt.expectedError, errorResponse["error"])
					}
				}

				// Check Content-Type header
				if w.Header().Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type 'application/json', got '%s'", 
						w.Header().Get("Content-Type"))
				}
			})
		}
	})
}

// TestDatabaseOperationsIntegration tests database operations directly
func TestDatabaseOperationsIntegration(t *testing.T) {
	database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
		ctx := context.Background()

		t.Run("FindOrCreateStripeCustomer", func(t *testing.T) {
			// Test customer creation
			_, err := testDB.Repo.FindOrCreateStripeCustomer(ctx, 
				"test_user_456", "test456@example.com")
			if err != nil {
				t.Fatalf("Failed to create customer: %v", err)
			}

			// Test customer retrieval
			customer, err := testDB.Repo.GetCustomerByUserID(ctx, "test_user_456")
			if err != nil {
				t.Fatalf("Failed to get customer: %v", err)
			}
			if customer == nil {
				t.Fatal("Customer not found")
			}
			if customer.UserID != "test_user_456" {
				t.Errorf("Expected user ID 'test_user_456', got '%s'", customer.UserID)
			}
			if customer.Email != "test456@example.com" {
				t.Errorf("Expected email 'test456@example.com', got '%s'", customer.Email)
			}
		})

		t.Run("CreateSubscription", func(t *testing.T) {
			// First create a customer
			_, err := testDB.Repo.FindOrCreateStripeCustomer(ctx, 
				"test_user_789", "test789@example.com")
			if err != nil {
				t.Fatalf("Failed to create customer: %v", err)
			}

			// Update customer with Stripe ID (simulating the flow)
			err = testDB.Repo.UpdateCustomerStripeID(ctx, "test_user_789", "cus_test789")
			if err != nil {
				t.Fatalf("Failed to update customer Stripe ID: %v", err)
			}

			// Create subscription
			now := time.Now()
			err = testDB.Repo.CreateSubscription(ctx, "cus_test789", "sub_test789", 
				"pro_plan", "price_789", "test_user_789", "active", now, now.AddDate(0, 0, 30))
			if err != nil {
				t.Fatalf("Failed to create subscription: %v", err)
			}

			// Retrieve subscription
			stripeSubID, status, _, exists, err := testDB.Repo.GetSubscriptionStatus(ctx, 
				"test_user_789", "pro_plan")
			if err != nil {
				t.Fatalf("Failed to get subscription status: %v", err)
			}
			if !exists {
				t.Fatal("Subscription not found")
			}
			if stripeSubID != "sub_test789" {
				t.Errorf("Expected Stripe subscription ID 'sub_test789', got '%s'", stripeSubID)
			}
			if status != "active" {
				t.Errorf("Expected status 'active', got '%s'", status)
			}
		})

		t.Run("UpdateSubscriptionStatus", func(t *testing.T) {
			// Create customer and subscription first
			_, err := testDB.Repo.FindOrCreateStripeCustomer(ctx, 
				"test_user_999", "test999@example.com")
			if err != nil {
				t.Fatalf("Failed to create customer: %v", err)
			}

			err = testDB.Repo.UpdateCustomerStripeID(ctx, "test_user_999", "cus_test999")
			if err != nil {
				t.Fatalf("Failed to update customer Stripe ID: %v", err)
			}

			now := time.Now()
			err = testDB.Repo.CreateSubscription(ctx, "cus_test999", "sub_test999", 
				"enterprise_plan", "price_999", "test_user_999", "active", now, now.AddDate(0, 0, 30))
			if err != nil {
				t.Fatalf("Failed to create subscription: %v", err)
			}

			// Update subscription status
			newPeriodEnd := now.AddDate(0, 1, 0)
			err = testDB.Repo.UpdateSubscriptionStatus(ctx, "sub_test999", "canceled", newPeriodEnd)
			if err != nil {
				t.Fatalf("Failed to update subscription status: %v", err)
			}

			// Verify update
			_, status, periodEnd, exists, err := testDB.Repo.GetSubscriptionStatus(ctx, 
				"test_user_999", "enterprise_plan")
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

// Note: Benchmark tests would need a separate WithTestDatabase function for *testing.B
// For now, we focus on the integration tests with real database
