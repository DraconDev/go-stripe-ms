package main

import (
	"encoding/json"
	"net/http"
)

// OpenAPI/Swagger specification for the Payment MS
var openAPISpec = map[string]interface{}{
	"openapi": "3.0.0",
	"info": map[string]interface{}{
		"title":       "Payment Microservice API",
		"description": "Stripe-based payment processing service",
		"version":     "1.0.0",
	},
	"servers": []map[string]string{
		{"url": "http://localhost:9000", "description": "Local development"},
	},
	"security": []map[string][]string{
		{"ApiKeyAuth": {}},
	},
	"components": map[string]interface{}{
		"securitySchemes": map[string]interface{}{
			"ApiKeyAuth": map[string]string{
				"type": "apiKey",
				"in":   "header",
				"name": "X-API-Key",
			},
		},
		"schemas": map[string]interface{}{
			"SubscriptionCheckoutRequest": map[string]interface{}{
				"type":     "object",
				"required": []string{"user_id", "email", "product_id", "price_id", "success_url", "cancel_url"},
				"properties": map[string]interface{}{
					"user_id":     map[string]string{"type": "string", "example": "user_123"},
					"email":       map[string]string{"type": "string", "example": "customer@example.com"},
					"product_id":  map[string]string{"type": "string", "example": "prod_RZaVDAN6Uf4Qfb"},
					"price_id":    map[string]string{"type": "string", "example": "price_1QhEBSFhH6dwUiIH"},
					"success_url": map[string]string{"type": "string", "example": "https://yourapp.com/success"},
					"cancel_url":  map[string]string{"type": "string", "example": "https://yourapp.com/cancel"},
				},
			},
			"CheckoutResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"checkout_session_id": map[string]string{"type": "string"},
					"checkout_url":        map[string]string{"type": "string"},
				},
			},
		},
	},
	"paths": map[string]interface{}{
		"/health": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":  "Health check",
				"security": []map[string][]string{}, // No auth required
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Service is healthy",
					},
				},
			},
		},
		"/api/v1/checkout/subscription": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Create subscription checkout session",
				"description": "Creates a Stripe checkout session for a subscription",
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]string{
								"$ref": "#/components/schemas/SubscriptionCheckoutRequest",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Checkout session created",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]string{
									"$ref": "#/components/schemas/CheckoutResponse",
								},
							},
						},
					},
					"401": map[string]interface{}{
						"description": "Unauthorized - Invalid or missing API key",
					},
				},
			},
		},
		"/api/v1/checkout/item": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Create single item checkout",
				"description": "Creates a Stripe checkout session for a single item",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{"description": "Success"},
				},
			},
		},
		"/api/v1/checkout/cart": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Create cart checkout",
				"description": "Creates a Stripe checkout session for multiple items",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{"description": "Success"},
				},
			},
		},
		"/api/v1/subscriptions/{user_id}/{product_id}": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "Get subscription status",
				"description": "Retrieves the subscription status for a user and product",
				"parameters": []map[string]interface{}{
					{
						"name":     "user_id",
						"in":       "path",
						"required": true,
						"schema":   map[string]string{"type": "string"},
					},
					{
						"name":     "product_id",
						"in":       "path",
						"required": true,
						"schema":   map[string]string{"type": "string"},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{"description": "Subscription status"},
				},
			},
		},
		"/api/v1/portal": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Create customer portal session",
				"description": "Creates a Stripe customer portal session for managing subscriptions",
				"responses": map[string]interface{}{
					"200": map[string]interface{}{"description": "Portal URL created"},
				},
			},
		},
	},
}

// OpenAPIHandler serves the OpenAPI specification
func (s *Server) openAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow CORS for docs
	json.NewEncoder(w).Encode(openAPISpec)
}
