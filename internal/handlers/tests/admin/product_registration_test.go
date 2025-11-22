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
					return
				}

				// Verify success field
				if success, ok := response["success"].(bool); !ok || !success {
					t.Errorf("Expected success=true in response, got: %v", response)
					return
				}

				// Verify project_id field
				if projectID, ok := response["project_id"].(string); !ok || projectID != "test-project" {
					t.Errorf("Expected project_id='test-project', got: %v", response["project_id"])
				}

				// Verify products array
				products, ok := response["products"].([]interface{})
				if !ok || len(products) == 0 {
					t.Error("Expected products array in response")
					return
				}

				// Verify first product structure
				product := products[0].(map[string]interface{})
				
				if planName, ok := product["plan_name"].(string); !ok || planName != "Pro Plan" {
					t.Errorf("Expected plan_name='Pro Plan', got: %v", product["plan_name"])
				}

				if stripeProductID, ok := product["stripe_product_id"].(string); !ok || stripeProductID == "" {
					t.Error("Expected non-empty stripe_product_id")
				}

				// Verify prices structure
				prices, ok := product["prices"].(map[string]interface{})
				if !ok {
					t.Error("Expected prices object in product")
					return
				}

				// Verify monthly price
				if monthly, ok := prices["monthly"].(map[string]interface{}); ok {
					if priceID, ok := monthly["stripe_price_id"].(string); !ok || priceID == "" {
						t.Error("Expected non-empty monthly stripe_price_id")
					}
					if amount, ok := monthly["amount"].(float64); !ok || amount != 2900 {
						t.Errorf("Expected monthly amount=2900, got: %v", monthly["amount"])
					}
				} else {
					t.Error("Expected monthly price in response")
				}

				// Verify yearly price
				if yearly, ok := prices["yearly"].(map[string]interface{}); ok {
					if priceID, ok := yearly["stripe_price_id"].(string); !ok || priceID == "" {
						t.Error("Expected non-empty yearly stripe_price_id")
					}
					if amount, ok := yearly["amount"].(float64); !ok || amount != 29000 {
						t.Errorf("Expected yearly amount=29000, got: %v", yearly["amount"])
					}
				} else {
					t.Error("Expected yearly price in response")
				}
			}	}
			})
		}
	})
}
