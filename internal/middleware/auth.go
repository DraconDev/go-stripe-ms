package middleware

import (
	"net/http"
)

// APIKeyAuth middleware validates API keys
type APIKeyAuth struct {
	apiKey string
}

// NewAPIKeyAuth creates a new API key authentication middleware
func NewAPIKeyAuth(apiKey string) *APIKeyAuth {
	return &APIKeyAuth{apiKey: apiKey}
}

// Middleware validates the X-API-Key header
func (a *APIKeyAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract API key from header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			http.Error(w, `{"error":"Missing X-API-Key header"}`, http.StatusUnauthorized)
			return
		}

		// Check if API key matches
		if apiKey != a.apiKey {
			http.Error(w, `{"error":"Invalid API key"}`, http.StatusUnauthorized)
			return
		}

		// Call next handler
		next.ServeHTTP(w, r)
	})
}
