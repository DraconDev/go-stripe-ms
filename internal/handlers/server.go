// server.go - Main server type definitions and HTTP handler wrappers
package handlers

import (
	"net/http"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/billing"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/cart"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/core"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/subscription"
	"github.com/stripe/stripe-go/v72"
)

// HTTPServer provides HTTP REST API for billing operations
type HTTPServer struct {
	db           database.RepositoryInterface
	stripeSecret string
}

// NewHTTPServer creates a new HTTP server instance
func NewHTTPServer(db database.RepositoryInterface, stripeSecret string) *HTTPServer {
	stripe.Key = stripeSecret

	return &HTTPServer{
		db:           db,
		stripeSecret: stripeSecret,
	}
}

// Wrapper methods that delegate to sub-package handler functions

// HealthCheck handles GET /health
func (s *HTTPServer) HealthCheck(w http.ResponseWriter, r *http.Request) {
	core.HandleHealth(w, r)
}

// RootHandler handles GET /
func (s *HTTPServer) RootHandler(w http.ResponseWriter, r *http.Request) {
	core.HandleRoot(w, r)
}

// CreateItemCheckout handles POST /api/v1/checkout/item
func (s *HTTPServer) CreateItemCheckout(w http.ResponseWriter, r *http.Request) {
	core.HandleItemCheckout(s.db, s.stripeSecret, w, r)
}

// CreateCartCheckout handles POST /api/v1/checkout/cart
func (s *HTTPServer) CreateCartCheckout(w http.ResponseWriter, r *http.Request) {
	cart.HandleCartCheckout(s.db, s.stripeSecret, w, r)
}

// DebugHandler handles GET /debug
func (s *HTTPServer) DebugHandler(w http.ResponseWriter, r *http.Request) {
	documentation.HandleDebug(w, r)
}

// OpenAPIHandler serves the OpenAPI specification
func (s *HTTPServer) OpenAPIHandler(w http.ResponseWriter, r *http.Request) {
	documentation.HandleOpenAPI(w, r)
}

// DocsHandler serves a simple HTML documentation page
func (s *HTTPServer) DocsHandler(w http.ResponseWriter, r *http.Request) {
	documentation.HandleDocs(w, r)
}

// CreateSubscriptionCheckout handles POST /api/v1/checkout/subscription
func (s *HTTPServer) CreateSubscriptionCheckout(w http.ResponseWriter, r *http.Request) {
	subscription.HandleSubscriptionCheckout(s.db, s.stripeSecret, w, r)
}

// GetSubscriptionStatus handles GET /api/v1/subscriptions/{user_id}/{product_id}
func (s *HTTPServer) GetSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	subscription.HandleSubscriptionStatus(s.db, w, r)
}

// CreateCustomerPortal handles POST /api/v1/portal
func (s *HTTPServer) CreateCustomerPortal(w http.ResponseWriter, r *http.Request) {
	billing.HandleCustomerPortal(s.db, s.stripeSecret, w, r)
}
