// server.go - Main package index for the server module
package server

import (
	"github.com/DraconDev/go-stripe-ms/internal/database"
)

// Export main types and functions from sub-packages for external use
// This allows cmd/server/main.go to import "github.com/DraconDev/go-stripe-ms/internal/server"

// HTTPServer is the main HTTP server struct
type HTTPServer = billing.HTTPServer

// NewHTTPServer creates a new HTTP server instance  
func NewHTTPServer(db database.RepositoryInterface, stripeSecret string) *HTTPServer {
	return billing.NewHTTPServer(db, stripeSecret)
}

// Export types from different modules for external import
type (
	// Cart types
	CartCheckoutRequest = cart.CartCheckoutRequest
	CartCheckoutResponse = cart.CartCheckoutResponse
	
	// Item checkout types
	ItemCheckoutRequest = core.ItemCheckoutRequest
	ItemCheckoutResponse = core.ItemCheckoutResponse
	
	// Subscription types
	SubscriptionStatusRequest = subscription.SubscriptionStatusRequest
	SubscriptionStatusResponse = subscription.SubscriptionStatusResponse
	
	// Customer portal types
	CustomerPortalRequest = billing.CustomerPortalRequest
	CustomerPortalResponse = billing.CustomerPortalResponse
)