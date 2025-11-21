package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
		server := handlers.NewHTTPServer(testDB.Repo, "sk_test_123")

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

				// Inject project ID into context
				ctx := context.WithValue(req.Context(), middleware.ProjectIDKey, project.ID)
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
