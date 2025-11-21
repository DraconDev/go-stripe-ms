package cart

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers"
)

// TestCreateCartCheckoutIntegration tests the cart checkout endpoint with a real database
func TestCreateCartCheckoutIntegration(t *testing.T) {
	database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
		// Setup test data
		project, err := testDB.CreateTestData()
		if err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}

		// Create HTTP server with real database
		server := handlers.NewHTTPServer(testDB.Repo, "sk_test_123")

		tests := []struct {
			name               string
			requestBody        map[string]interface{}
			expectedStatusCode int
			expectedError      string
		}{
			{
				name: "Valid cart checkout request",
				requestBody: map[string]interface{}{
					"user_id": "test_user_123",
					"email":   "test@example.com",
					"items": []map[string]interface{}{
						{
							"product_id": "prod_test123",
							"price_id":   "price_test123",
							"quantity":   2,
						},
					},
					"success_url": "https://example.com/success",
					"cancel_url":  "https://example.com/cancel",
				},
				expectedStatusCode: http.StatusOK,
			},
			{
				name: "Empty items list",
				requestBody: map[string]interface{}{
					"user_id":     "test_user_123",
					"email":       "test@example.com",
					"items":       []map[string]interface{}{},
					"success_url": "https://example.com/success",
					"cancel_url":  "https://example.com/cancel",
				},
				expectedStatusCode: http.StatusBadRequest,
				expectedError:      "Cart cannot be empty",
			},
			{
				name: "Item missing price_id",
				requestBody: map[string]interface{}{
					"user_id": "test_user_123",
					"email":   "test@example.com",
					"items": []map[string]interface{}{
						{"quantity": 1},
					},
					"success_url": "https://example.com/success",
					"cancel_url":  "https://example.com/cancel",
				},
				expectedStatusCode: http.StatusBadRequest,
				expectedError:      "Price ID is required for all items",
			},
			{
				name: "Item with invalid quantity",
				requestBody: map[string]interface{}{
					"user_id": "test_user_123",
					"email":   "test@example.com",
					"items": []map[string]interface{}{
						{"price_id": "price_1", "quantity": 0},
					},
					"success_url": "https://example.com/success",
					"cancel_url":  "https://example.com/cancel",
				},
				expectedStatusCode: http.StatusBadRequest,
				expectedError:      "Quantity must be at least 1 for all items",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create request
				bodyBytes, _ := json.Marshal(tt.requestBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/cart",
					bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")

				// Execute
				w := httptest.NewRecorder()
				server.CreateCartCheckout(w, req)

				// Assert
				if w.Code != tt.expectedStatusCode {
					t.Errorf("Expected status code %d, got %d",
						tt.expectedStatusCode, w.Code)
				}

				if tt.expectedError != "" {
					if !bytes.Contains(w.Body.Bytes(), []byte(tt.expectedError)) {
						t.Errorf("Expected error '%s' in response body", tt.expectedError)
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
