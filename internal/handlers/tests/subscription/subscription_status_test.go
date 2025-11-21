package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers"
	"github.com/DraconDev/go-stripe-ms/internal/middleware"
)

// TestGetSubscriptionStatusIntegration tests subscription status retrieval with real database
func TestGetSubscriptionStatusIntegration(t *testing.T) {
	database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
		// Setup test data
		project, customer, err := testDB.CreateTestData()
		if err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}

		// Create HTTP server with real database
		stripeKey := os.Getenv("STRIPE_SECRET_KEY")
		server := handlers.NewHTTPServer(testDB.Repo, stripeKey)

		tests := []struct {
			name               string
			path               string
			expectedStatusCode int
			expectedResponse   map[string]interface{}
			expectedError      string
		}{
			// Note: path will be dynamically constructed in the test loop
			{
				name:               "Valid subscription request with real data",
				path:               "", // Will be set dynamically
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
				// Dynamically set path for valid subscription test
				path := tt.path
				if path == "" {
					path = fmt.Sprintf("/api/v1/subscriptions/%s/premium_plan", customer.UserID)
				}

				// Create request
				req := httptest.NewRequest(http.MethodGet, path, nil)

				// Inject project ID into context
				ctx := context.WithValue(req.Context(), middleware.ProjectIDKey, project.ID)
				req = req.WithContext(ctx)

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
