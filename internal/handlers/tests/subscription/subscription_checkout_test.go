package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DraconDev/go-stripe-ms/internal/config"
	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers"
	"github.com/DraconDev/go-stripe-ms/internal/middleware"
)

// TestCreateSubscriptionCheckoutIntegration tests with real database
func TestCreateSubscriptionCheckoutIntegration(t *testing.T) {
	database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
		// Setup test data
		project, customer, err := testDB.CreateTestData()
		if err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}
		_ = customer // Will be used in future tests

		// Create HTTP server with real database
		stripeKey := os.Getenv("STRIPE_SECRET_KEY")
		server := handlers.NewHTTPServer(testDB.Repo, stripeKey)

		tests := []struct {
			name               string
			requestBody        map[string]interface{}
			expectedStatusCode int
			expectedResponse   map[string]interface{}
			expectedError      string
			setupAuth          bool // Added for authentication setup
		}{
			{
				name: "Valid checkout request with real database",
				requestBody: map[string]interface{}{
					"user_id":     "test_user_123",
					"email":       "test@example.com",
					"price_id":    config.TEST_SUBSCRIPTION_PRICE_ID,   // Use real test subscription price
					"product_id":  config.TEST_SUBSCRIPTION_PRODUCT_ID, // Use real test product
					"success_url": "http://localhost:3000/success",
					"cancel_url":  "http://localhost:3000/cancel",
				},
				expectedStatusCode: http.StatusOK,
				setupAuth:          true,
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
				// Use the actual customer UserID from test data for valid requests
				if tt.setupAuth {
					if userID, ok := tt.requestBody["user_id"].(string); !ok || userID == "test_user_123" {
						tt.requestBody["user_id"] = customer.UserID
					}
				}

				// Create request
				bodyBytes, _ := json.Marshal(tt.requestBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/subscription",
					bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")

				// Inject project context
				ctx := context.WithValue(req.Context(), middleware.ProjectIDKey, project.ID)
				req = req.WithContext(ctx)

				// Inject project ID into context
				ctx = context.WithValue(req.Context(), middleware.ProjectIDKey, project.ID)
				req = req.WithContext(ctx)

				// Execute
				w := httptest.NewRecorder()
				server.CreateSubscriptionCheckout(w, req)

				// Assert
				if w.Code != tt.expectedStatusCode {
					t.Errorf("Expected status code %d, got %d",
						tt.expectedStatusCode, w.Code)
				}

				if tt.expectedError != "" {
					var errorResponse map[string]interface{}
					if err := json.Unmarshal(w.Body.Bytes(), &errorResponse); err != nil {
						t.Errorf("Failed to unmarshal error response: %v", err)
					} else {
						// Check nested error object
						if errObj, ok := errorResponse["error"].(map[string]interface{}); ok {
							if msg, ok := errObj["message"].(string); ok {
								if msg != tt.expectedError {
									t.Errorf("Expected error message '%s', got '%s'",
										tt.expectedError, msg)
								}
							} else {
								t.Errorf("Error response missing 'message' field")
							}
						} else {
							// Fallback if error is just a string
							if errMsg, ok := errorResponse["error"].(string); ok {
								if errMsg != tt.expectedError {
									t.Errorf("Expected error '%s', got '%s'", tt.expectedError, errMsg)
								}
							} else {
								t.Errorf("Unexpected error format")
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
