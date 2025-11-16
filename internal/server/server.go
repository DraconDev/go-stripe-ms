// server.go - Main package file for the server module
package server

import (
	"github.com/DraconDev/go-stripe-ms/internal/database"
)

// HTTPServer provides HTTP REST API for billing operations
type HTTPServer struct {
	db           database.RepositoryInterface
	stripeSecret string
}

// NewHTTPServer creates a new HTTP server instance
func NewHTTPServer(db database.RepositoryInterface, stripeSecret string) *HTTPServer {
	return &HTTPServer{
		db:           db,
		stripeSecret: stripeSecret,
	}
}