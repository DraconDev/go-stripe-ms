package documentation

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
	html := `<!DOCTYPE html>
<html><head><title>Payment MS API</title>
<style>body{font-family:Arial;max-width:1200px;margin:0 auto;padding:20px}
.endpoint{background:#f5f5f5;padding:15px;margin:10px 0;border-left:4px solid #4CAF50}
.method{display:inline-block;padding:5px 10px;border-radius:3px;font-weight:bold;margin-right:10px}
.post{background:#4CAF50;color:white}.get{background:#2196F3;color:white}
code{background:#f0f0f0;padding:2px 6px;border-radius:3px}
pre{background:#f5f5f5;padding:15px;overflow-x:auto;border-radius:5px}</style></head>
<body><h1>🔐 Payment Microservice API</h1>
<p>All <code>/api/v1/*</code> endpoints require <code>X-API-Key</code> header</p>
<h2>Endpoints</h2>
<div class="endpoint"><span class="method get">GET</span><code>/health</code><p>Health check</p></div>
<div class="endpoint"><span class="method get">GET</span><code>/openapi.json</code><p>OpenAPI spec</p></div>
<div class="endpoint"><span class="method post">POST</span><code>/api/v1/checkout/subscription</code><p>🔒 Create subscription checkout</p></div>
<div class="endpoint"><span class="method post">POST</span><code>/api/v1/checkout/item</code><p>🔒 Single item checkout</p></div>
<div class="endpoint"><span class="method post">POST</span><code>/api/v1/checkout/cart</code><p>🔒 Cart checkout</p></div>
<div class="endpoint"><span class="method get">GET</span><code>/api/v1/subscriptions/{user_id}/{product_id}</code><p>🔒 Get subscription status</p></div>
<div class="endpoint"><span class="method post">POST</span><code>/api/v1/portal</code><p>🔒 Customer portal</p></div>
<p>See <a href="/openapi.json">OpenAPI Spec</a> | <a href="https://github.com/yourusername/go-stripe-ms/blob/main/API_REQUESTS.md">Full Docs</a></p>
</body></html>`

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")
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
