package middleware

import (
	"net/http"
	"strings"
)

// CORS middleware handles Cross-Origin Resource Sharing
// Only needed if you're calling the API from a web browser (frontend app)
type CORSMiddleware struct {
	allowedOrigins []string
	enabled        bool
}

// NewCORSMiddleware creates a CORS middleware
// Set allowedOrigins to empty slice to allow all origins (NOT recommended for production)
// Example: []string{"https://myapp.com", "https://app.myapp.com"}
func NewCORSMiddleware(allowedOrigins []string) *CORSMiddleware {
	return &CORSMiddleware{
		allowedOrigins: allowedOrigins,
		enabled:        len(allowedOrigins) > 0,
	}
}

// Handler wraps an HTTP handler with CORS support
func (c *CORSMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CORS if not enabled
		if !c.enabled {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if c.isOriginAllowed(origin) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "3600")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isOriginAllowed checks if an origin is in the allowed list
func (c *CORSMiddleware) isOriginAllowed(origin string) bool {
	// If no origins specified, allow all (NOT recommended for production)
	if len(c.allowedOrigins) == 0 {
		return true
	}

	// Check if origin is in allowed list
	for _, allowed := range c.allowedOrigins {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}

	return false
}
