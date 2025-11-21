package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/middleware"
)

// TestAPIKeyAuth_Middleware tests the API key authentication middleware
func TestAPIKeyAuth_Middleware(t *testing.T) {
	database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
		// Create a test project with a known API key
		// Setup test database with a known project
		project, customer, err := testDB.CreateTestData()
		if err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}
		_ = customer // Not needed for auth tests

		authMiddleware := middleware.NewAPIKeyAuth(testDB.Repo)

		tests := []struct {
			name           string
			apiKey         string
			expectedStatus int
		}{
			{
				name:           "Valid API Key",
				apiKey:         project.APIKey, // Use the actual API key from created project
				expectedStatus: http.StatusOK,
			},
			{
				name:           "Missing API Key",
				apiKey:         "",
				expectedStatus: http.StatusUnauthorized,
			},
			{
				name:           "Invalid API Key",
				apiKey:         "invalid_key",
				expectedStatus: http.StatusUnauthorized,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, _ := http.NewRequest("GET", "/", nil)
				if tt.apiKey != "" {
					req.Header.Set("X-API-Key", tt.apiKey)
				}

				rr := httptest.NewRecorder()
				handler := authMiddleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					// Verify project ID is in context
					projectID, ok := middleware.GetProjectID(r.Context())
					if !ok {
						t.Error("Project ID not found in context")
					}
					// Check it matches the test project
					if projectID != project.ID {
						t.Errorf("Expected project ID %v, got %v", project.ID, projectID)
					}
				}))

				handler.ServeHTTP(rr, req)

				if status := rr.Code; status != tt.expectedStatus {
					t.Errorf("handler returned wrong status code: got %v want %v",
						status, tt.expectedStatus)
				}
			})
		}
	})
}
