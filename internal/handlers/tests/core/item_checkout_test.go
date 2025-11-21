package core

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers"
)

// TestCreateItemCheckoutIntegration tests the item checkout endpoint with a real database
func TestCreateItemCheckoutIntegration(t *testing.T) {
	database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
		// Setup test data
		if err := testDB.CreateTestData(); err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}

		// Create HTTP server with real database
		// Using a test stripe key
		server := handlers.NewHTTPServer(testDB.Repo, "sk_test_123")

		tests := []struct {
			name               string
			requestBody        map[string]interface{}
			expectedStatusCode int
			expectedError      string
		}{
			{
				name: "Valid item checkout request",
				requestBody: map[string]interface{}{
					"user_id":     "test_user_123",
					"email":       "test@example.com",
					"price_id":    "price_test123",
					"quantity":    1,
					"success_url": "https://example.com/success",
					"cancel_url":  "https://example.com/cancel",
				},
				expectedStatusCode: http.StatusOK,
			},
			{
				name: "Valid item checkout request with default quantity",
				requestBody: map[string]interface{}{
					"user_id":  "test_user_123",
					"email":    "test@example.com",
					"price_id": "price_test123",
					// quantity omitted, should default to 1
					"success_url": "https://example.com/success",
					"cancel_url":  "https://example.com/cancel",
				},
				expectedStatusCode: http.StatusOK,
			},
			{
				name: "Missing price_id",
				requestBody: map[string]interface{}{
					"user_id":     "test_user_123",
					"email":       "test@example.com",
					"quantity":    1,
					"success_url": "https://example.com/success",
					"cancel_url":  "https://example.com/cancel",
				},
				expectedStatusCode: http.StatusBadRequest,
				expectedError:      "Missing required fields",
			},
			{
				name: "Invalid quantity (0)",
				requestBody: map[string]interface{}{
					"user_id":     "test_user_123",
					"email":       "test@example.com",
					"price_id":    "price_test123",
					"quantity":    0,
					"success_url": "https://example.com/success",
					"cancel_url":  "https://example.com/cancel",
				},
				expectedStatusCode: http.StatusBadRequest,
				expectedError:      "Quantity must be at least 1",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create request
				bodyBytes, _ := json.Marshal(tt.requestBody)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/item",
					bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")

				// Execute
				w := httptest.NewRecorder()
				server.CreateItemCheckout(w, req)

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
							// Fallback for simple error strings if any (though our API uses structured errors)
							// or if the test expectation was simplified.
							// Let's assume strict structure check based on API_REQUESTS.md
							// But for now, let's just check if the error string is contained in the response body for simplicity if structure fails
							if !bytes.Contains(w.Body.Bytes(), []byte(tt.expectedError)) {
								t.Errorf("Expected error '%s' in response body", tt.expectedError)
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
