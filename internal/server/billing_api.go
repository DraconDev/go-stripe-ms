package server

import (
	"github.com/DraconDev/go-stripe-ms/internal/database"
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
