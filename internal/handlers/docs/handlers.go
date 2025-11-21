// Package docs provides handlers for API documentation endpoints
package docs

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// HandleDebug provides debug information
func HandleDebug(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"service":      "billing-service",
		"status":       "running",
		"time":         time.Now().UTC().Format(time.RFC3339),
		"environment":  getEnvironment(),
		"database_url": "configured",
		"stripe_key":   "configured",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	jsonData, _ := json.Marshal(info)
	if _, err := w.Write(jsonData); err != nil {
		log.Printf("Error writing debug response: %v", err)
	}
}

// HandleOpenAPI serves the OpenAPI specification
func HandleOpenAPI(w http.ResponseWriter, r *http.Request) {
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "Payment Microservice API",
			"description": "Stripe-based payment processing service with API key authentication",
			"version":     "1.0.0",
		},
		"servers": []map[string]string{
			{"url": "http://localhost:9000", "description": "Local development"},
		},
		"paths": map[string]interface{}{
			"/health":                       map[string]interface{}{"get": map[string]string{"summary": "Health check"}},
			"/docs":                         map[string]interface{}{"get": map[string]string{"summary": "API documentation"}},
			"/api/v1/checkout/subscription": map[string]interface{}{"post": map[string]string{"summary": "Create subscription checkout"}},
			"/api/v1/checkout/item":         map[string]interface{}{"post": map[string]string{"summary": "Create item checkout"}},
			"/api/v1/checkout/cart":         map[string]interface{}{"post": map[string]string{"summary": "Create cart checkout"}},
			"/api/v1/subscriptions/{user_id}/{product_id}": map[string]interface{}{"get": map[string]string{"summary": "Get subscription status"}},
			"/api/v1/portal": map[string]interface{}{"post": map[string]string{"summary": "Create customer portal"}},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(spec); err != nil {
		log.Printf("Error encoding OpenAPI spec: %v", err)
	}
}

// HandleDocs serves a simple HTML documentation page
func HandleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	html := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Payment MS API Documentation</title>
<style>
* { box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; background: #f9f9f9; }
h1 { color: #333; border-bottom: 3px solid #4CAF50; padding-bottom: 10px; }
h2 { color: #555; margin-top: 40px; }
h3 { color: #666; margin-top: 25px; font-size: 1.1em; }
.header-auth { background: #fff3cd; border-left: 4px solid #ffc107; padding: 15px; margin: 20px 0; border-radius: 4px; }
.endpoint { background: white; padding: 20px; margin: 20px 0; border-left: 4px solid #4CAF50; border-radius: 4px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
.endpoint-header { display: flex; align-items: center; margin-bottom: 15px; }
.method { display: inline-block; padding: 6px 12px; border-radius: 4px; font-weight: bold; margin-right: 12px; font-size: 0.9em; }
.get { background: #2196F3; color: white; }
.post { background: #4CAF50; color: white; }
.path { font-family: 'Courier New', monospace; font-size: 1.1em; color: #333; }
.lock { color: #ff9800; margin-left: 8px; }
.description { color: #666; margin: 10px 0; }
.section { margin: 15px 0; }
.section-title { font-weight: bold; color: #555; margin-bottom: 8px; }
code { background: #f0f0f0; padding: 2px 6px; border-radius: 3px; font-family: 'Courier New', monospace; }
pre { background: #2d2d2d; color: #f8f8f2; padding: 15px; overflow-x: auto; border-radius: 4px; font-size: 0.9em; }
.response { background: #e8f5e9; }
.footer { margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; color: #666; }
</style>
</head>
<body>
<h1>🔐 Payment Microservice API</h1>

<div class="header-auth">
<strong>Authentication Required:</strong> All <code>/api/v1/*</code> endpoints require the <code>X-API-Key</code> header.
<br><br>
<code>X-API-Key: your_api_key_here</code>
</div>

<h2>Public Endpoints</h2>

<div class="endpoint">
<div class="endpoint-header">
<span class="method get">GET</span>
<span class="path">/health</span>
</div>
<div class="description">Health check endpoint (no authentication required)</div>
<div class="section">
<div class="section-title">Response:</div>
<pre>{"status":"healthy","timestamp":"2025-11-21T10:00:00Z","service":"billing-service"}</pre>
</div>
</div>

<div class="endpoint">
<div class="endpoint-header">
<span class="method get">GET</span>
<span class="path">/openapi.json</span>
</div>
<div class="description">OpenAPI 3.0 specification (no authentication required)</div>
</div>

<h2>Protected Endpoints</h2>

<div class="endpoint">
<div class="endpoint-header">
<span class="method post">POST</span>
<span class="path">/api/v1/checkout/subscription</span>
<span class="lock">🔒</span>
</div>
<div class="description">Create a Stripe checkout session for a subscription</div>
<div class="section">
<div class="section-title">Request Body:</div>
<pre>{
  "user_id": "user_123",
  "email": "customer@example.com",
  "product_id": "prod_RZaVDAN6Uf4Qfb",
  "price_id": "price_1QhEBSFhH6dwUiIH",
  "success_url": "https://yourapp.com/success",
  "cancel_url": "https://yourapp.com/cancel"
}</pre>
</div>
<div class="section response">
<div class="section-title">Response:</div>
<pre>{
  "checkout_session_id": "cs_test_...",
  "checkout_url": "https://checkout.stripe.com/c/pay/cs_test_..."
}</pre>
</div>
</div>

<div class="endpoint">
<div class="endpoint-header">
<span class="method post">POST</span>
<span class="path">/api/v1/checkout/item</span>
<span class="lock">🔒</span>
</div>
<div class="description">Create a Stripe checkout session for a single item purchase</div>
<div class="section">
<div class="section-title">Request Body:</div>
<pre>{
  "user_id": "user_123",
  "email": "customer@example.com",
  "price_id": "price_1QhEBSFhH6dwUiIH",
  "quantity": 1,
  "success_url": "https://yourapp.com/success",
  "cancel_url": "https://yourapp.com/cancel"
}</pre>
</div>
<div class="section response">
<div class="section-title">Response:</div>
<pre>{
  "checkout_session_id": "cs_test_...",
  "checkout_url": "https://checkout.stripe.com/..."
}</pre>
</div>
</div>

<div class="endpoint">
<div class="endpoint-header">
<span class="method post">POST</span>
<span class="path">/api/v1/checkout/cart</span>
<span class="lock">🔒</span>
</div>
<div class="description">Create a checkout session for multiple items (shopping cart)</div>
<div class="section">
<div class="section-title">Request Body:</div>
<pre>{
  "user_id": "user_123",
  "email": "customer@example.com",
  "items": [
    {"price_id": "price_ABC", "quantity": 2},
    {"price_id": "price_XYZ", "quantity": 1}
  ],
  "success_url": "https://yourapp.com/success",
  "cancel_url": "https://yourapp.com/cancel"
}</pre>
</div>
</div>

<div class="endpoint">
<div class="endpoint-header">
<span class="method get">GET</span>
<span class="path">/api/v1/subscriptions/{user_id}/{product_id}</span>
<span class="lock">🔒</span>
</div>
<div class="description">Get the subscription status for a user and product</div>
<div class="section">
<div class="section-title">URL Parameters:</div>
<code>user_id</code> - User identifier<br>
<code>product_id</code> - Stripe product ID
</div>
<div class="section response">
<div class="section-title">Response (Active):</div>
<pre>{
  "status": "active",
  "subscription_id": "sub_1QhEBS...",
  "current_period_end": "2025-12-21T10:00:00Z",
  "product_id": "prod_RZaVDAN6Uf4Qfb"
}</pre>
</div>
<div class="section response">
<div class="section-title">Response (No Subscription):</div>
<pre>{
  "status": "none",
  "subscription_id": "",
  "current_period_end": "0001-01-01T00:00:00Z",
  "product_id": ""
}</pre>
</div>
</div>

<div class="endpoint">
<div class="endpoint-header">
<span class="method post">POST</span>
<span class="path">/api/v1/portal</span>
<span class="lock">🔒</span>
</div>
<div class="description">Create a Stripe customer portal session for managing subscriptions</div>
<div class="section">
<div class="section-title">Request Body:</div>
<pre>{
  "user_id": "user_123",
  "return_url": "https://yourapp.com/account"
}</pre>
</div>
<div class="section response">
<div class="section-title">Response:</div>
<pre>{
  "portal_url": "https://billing.stripe.com/p/session/..."
}</pre>
</div>
</div>

<div class="footer">
<p><strong>Additional Resources:</strong></p>
<ul>
<li><a href="/openapi.json">OpenAPI Specification (JSON)</a></li>
<li><a href="https://github.com/yourusername/go-stripe-ms/blob/main/API_REQUESTS.md">Detailed API Documentation</a></li>
</ul>
<p><em>All timestamps are in ISO 8601 format (UTC)</em></p>
</div>

</body>
</html>`

	fmt.Fprint(w, html)
}

// getEnvironment returns the current environment with default
func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		return "development"
	}
	return env
}
