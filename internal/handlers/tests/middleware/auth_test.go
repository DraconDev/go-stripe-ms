package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/middleware"
	"github.com/google/uuid"
)

// TestAPIKeyAuth_Middleware tests the API key authentication middleware
func TestAPIKeyAuth_Middleware(t *testing.T) {
	database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
		// Create a test project with a known API key
		apiKey := "sk_test_auth_middleware"
		project := &database.Project{
			ID:        uuid.New(),
			Name:      "Auth Test Project",
			APIKey:    apiKey,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := testDB.CreateTestProject(project); err != nil {
			t.Fatalf("Failed to create test project: %v", err)
		}

		authMiddleware := middleware.NewAPIKeyAuth(testDB.Repo)

		tests := []struct {
			name           string
			apiKey         string
			expectedStatus int
		}{
			{
				name:           "Valid API Key",
				apiKey:         apiKey,
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
