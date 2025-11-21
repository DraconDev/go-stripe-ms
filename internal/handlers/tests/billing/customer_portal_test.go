package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers"
	"github.com/DraconDev/go-stripe-ms/internal/middleware"
)

// TestCreateCustomerPortalIntegration tests with real database
func TestCreateCustomerPortalIntegration(t *testing.T) {
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
			expectedError      string
		}{
			{
				name: "Valid portal request",
				requestBody: map[string]interface{}{
					"user_id":    "test_user_123",
					"return_url": "http://localhost:3000/account",
				},
				expectedStatusCode: http.StatusOK,
			},
			{
				name: "Missing user_id",
				requestBody: map[string]interface{}{
					"return_url": "https://example.com/account",
				},
				expectedStatusCode: http.StatusBadRequest,
				expectedError:      "Missing user_id or return_url",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Use the actual customer UserID from test data for valid requests
				if tt.name == "Valid portal request" {
					tt.requestBody["user_id"] = customer.UserID
				}

				// Create request
				bodyBytes, _ := json.Marshal(tt.requestBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/portal",
					bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")

				// Inject project ID into context
				ctx := context.WithValue(req.Context(), middleware.ProjectIDKey, project.ID)
				req = req.WithContext(ctx)

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
