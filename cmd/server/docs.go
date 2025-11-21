package main

import (
	"fmt"
	"net/http"
)

// docsHandler serves a simple HTML page with API documentation
func (s *Server) docsHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Payment MS API Documentation</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }
        h1 { color: #333; }
        h2 { color: #666; margin-top: 30px; }
        .endpoint { background: #f5f5f5; padding: 15px; margin: 10px 0; border-left: 4px solid #4CAF50; }
        .method { display: inline-block; padding: 5px 10px; border-radius: 3px; font-weight: bold; margin-right: 10px; }
        .post { background: #4CAF50; color: white; }
        .get { background: #2196F3; color: white; }
        code { background: #f0f0f0; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f5f5f5; padding: 15px; overflow-x: auto; border-radius: 5px; }
        .auth-required { color: #ff9800; font-weight: bold; }
    </style>
</head>
<body>
    <h1>🔐 Payment Microservice API</h1>
    <p>Stripe-based payment processing service</p>
    
    <h2>Authentication</h2>
    <p>All <code>/api/v1/*</code> endpoints require the <code>X-API-Key</code> header:</p>
    <pre>X-API-Key: your_api_key_here</pre>
    
    <h2>Base URL</h2>
    <pre>http://localhost:9000</pre>
    
    <h2>Endpoints</h2>
    
    <div class="endpoint">
        <span class="method get">GET</span>
        <code>/health</code>
        <p>Health check endpoint (no auth required)</p>
    </div>
    
    <div class="endpoint">
        <span class="method get">GET</span>
        <code>/openapi.json</code>
        <p>OpenAPI specification (no auth required)</p>
    </div>
    
    <div class="endpoint">
        <span class="method post">POST</span>
        <code>/api/v1/checkout/subscription</code>
        <span class="auth-required">🔒 Auth Required</span>
        <p>Create a subscription checkout session</p>
        <pre>{
  "user_id": "user_123",
  "email": "customer@example.com",
  "product_id": "prod_RZaVDAN6Uf4Qfb",
  "price_id": "price_1QhEBSFhH6dwUiIH",
  "success_url": "https://yourapp.com/success",
  "cancel_url": "https://yourapp.com/cancel"
}</pre>
    </div>
    
    <div class="endpoint">
        <span class="method post">POST</span>
        <code>/api/v1/checkout/item</code>
        <span class="auth-required">🔒 Auth Required</span>
        <p>Create a single item checkout session</p>
    </div>
    
    <div class="endpoint">
        <span class="method post">POST</span>
        <code>/api/v1/checkout/cart</code>
        <span class="auth-required">🔒 Auth Required</span>
        <p>Create a cart checkout session with multiple items</p>
    </div>
    
    <div class="endpoint">
        <span class="method get">GET</span>
        <code>/api/v1/subscriptions/{user_id}/{product_id}</code>
        <span class="auth-required">🔒 Auth Required</span>
        <p>Get subscription status for a user and product</p>
    </div>
    
    <div class="endpoint">
        <span class="method post">POST</span>
        <code>/api/v1/portal</code>
        <span class="auth-required">🔒 Auth Required</span>
        <p>Create a Stripe customer portal session</p>
    </div>
    
    <h2>Full Documentation</h2>
    <p>For complete API documentation, see:</p>
    <ul>
        <li><a href="/openapi.json">OpenAPI Specification (JSON)</a></li>
        <li><a href="https://github.com/yourusername/go-stripe-ms/blob/main/API_REQUESTS.md">API_REQUESTS.md</a></li>
    </ul>
    
    <h2>Example Request</h2>
    <pre>curl -X POST http://localhost:9000/api/v1/checkout/subscription \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "email": "customer@example.com",
    "product_id": "prod_RZaVDAN6Uf4Qfb",
    "price_id": "price_1QhEBSFhH6dwUiIH",
    "success_url": "https://yourapp.com/success",
    "cancel_url": "https://yourapp.com/cancel"
  }'</pre>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprint(w, html)
}
