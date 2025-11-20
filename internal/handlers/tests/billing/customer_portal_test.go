package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DraconDev/go-stripe-ms/internal/database"
)

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