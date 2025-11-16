// server.go - Main package index for the reorganized server module
package server

import (
	"github.com/DraconDev/go-stripe-ms/internal/database"
	
	// Import from subdirectories
	billing "github.com/DraconDev/go-stripe-ms/internal/server/billing"
	core "github.com/DraconDev/go-stripe-ms/internal/server/core"
)

// Export main types and functions from sub-packages for external use
// This allows cmd/server/main.go to import "github.com/DraconDev/go-stripe-ms/internal/server"

// HTTPServer is the main HTTP server struct
type HTTPServer = billing.HTTPServer

// NewHTTPServer creates a new HTTP server instance  
func NewHTTPServer(db database.RepositoryInterface, stripeSecret string) *HTTPServer {
	return billing.NewHTTPServer(db, stripeSecret)
}

// Export core handler functions for external use
var (
	CreateItemCheckout = core.CreateItemCheckout
	HealthCheck = core.HealthCheck
	RootHandler = core.RootHandler
	CreateSubscriptionCheckout = core.CreateSubscriptionCheckout
)