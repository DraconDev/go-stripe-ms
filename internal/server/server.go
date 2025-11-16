// server.go - Main package file for the reorganized server module
package server

import (
	"github.com/DraconDev/go-stripe-ms/internal/database"
)

// HTTPServer provides HTTP REST API for billing operations
// This type is defined in the billing package but exported here for convenience
type HTTPServer = BillingHTTPServer

// NewHTTPServer creates a new HTTP server instance
// This function is defined in the billing package but exported here
func NewHTTPServer(db database.RepositoryInterface, stripeSecret string) *HTTPServer {
	return NewBillingHTTPServer(db, stripeSecret)
}