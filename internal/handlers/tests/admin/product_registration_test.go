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

func TestProductRegistrationIntegration(t *testing.T) {
	database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
		// Setup test data
		project, _, err := testDB.CreateTestData()
		if err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}

		// Create HTTP server with real database
		stripeKey := os.Getenv("STRIPE_SECRET_KEY")
		server := handlers.NewHTTPServer(testDB.Repo, stripeKey)

		tests := []struct {
			name               string
			requestBody        map[string]interface{}
			expectedStatusCode int
			setupAuth          bool
		}{
			{
				name: "Valid product registration",
				requestBody: map[string]interface{}{
					"project_name": "test-project",
					"plans": []map[string]interface{}{
						{
							"name":        "Pro Plan",
							"description": "Professional features",
							"features":    []string{"feature1", "feature2"},
							"pricing": map[string]interface{}{
								"monthly": 2900,
								"yearly":  29000,
							},
						},
					},
				},
				expectedStatusCode: http.StatusCreated,
				setupAuth:          true,
			},
			{
				name: "Missing project_name",
				requestBody: map[string]interface{}{
					"plans": []map[string]interface{}{
						{
							"name": "Pro Plan",
							"pricing": map[string]interface{}{
								"monthly": 2900,
							},
						},
					},
				},
				expectedStatusCode: http.StatusBadRequest,
				setupAuth:          true,
			},
			{
				name: "Missing plans",
				requestBody: map[string]interface{}{
					"project_name": "test-project",
					"plans":        []map[string]interface{}{},
				},
				expectedStatusCode: http.StatusBadRequest,
				setupAuth:          true,
			},
			{
				name: "Invalid pricing (no monthly or yearly)",
				requestBody: map[string]interface{}{
					"project_name": "test-project",
					"plans": []map[string]interface{}{
						{
							"name": "Pro Plan",
							"pricing": map[string]interface{}{
								"monthly": 0,
								"yearly":  0,
							},
						},
					},
				},
				expectedStatusCode: http.StatusBadRequest,
				setupAuth:          true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create request
				bodyBytes, _ := json.Marshal(tt.requestBody)
				req := httptest.NewRequest(http.MethodPost, "/admin/products/register",
					bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")

				if tt.setupAuth {
					// Inject project context
					ctx := context.WithValue(req.Context(), middleware.ProjectIDKey, project.ID)
					req = req.WithContext(ctx)
				}

				// Execute
				w := httptest.NewRecorder()
				server.RegisterProducts(w, req)

				// Assert status code
				if w.Code != tt.expectedStatusCode {
					t.Errorf("Expected status code %d, got %d. Response: %s",
						tt.expectedStatusCode, w.Code, w.Body.String())
				}

				// Check Content-Type header
				if w.Header().Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type 'application/json', got '%s'",
						w.Header().Get("Content-Type"))
				}

				// For successful creation, verify response structure
				if tt.expectedStatusCode == http.StatusCreated {
					var response map[string]interface{}
					if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
						t.Errorf("Failed to unmarshal response: %v", err)
					}

					if success, ok := response["success"].(bool); !ok || !success {
						t.Errorf("Expected success=true in response")
					}

					if products, ok := response["products"].([]interface{}); !ok || len(products) == 0 {
						t.Error("Expected products array in response")
					}
				}
			})
		}
	})
}
