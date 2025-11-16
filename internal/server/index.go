// index.go - Main package file for the server module
// This file establishes the server package in the root directory
// All files in subdirectories also declare "package server"
// making them part of the same logical package
package server

// Export main types and functions for external use
// These are defined in subdirectories but accessible throughout the server package

// HTTPServer is the main HTTP server struct
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