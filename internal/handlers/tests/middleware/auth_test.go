package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DraconDev/go-stripe-ms/internal/middleware"
)

func TestAPIKeyAuth_Middleware(t *testing.T) {
	apiKey := "test-api-key"
	auth := middleware.NewAPIKeyAuth(apiKey)

	// Create a simple handler to wrap
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	tests := []struct {
		name           string
		apiKeyHeader   string
		expectedStatus int
	}{
		{
			name:           "Valid API Key",
			apiKeyHeader:   apiKey,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing API Key",
			apiKeyHeader:   "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid API Key",
			apiKeyHeader:   "wrong-key",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.apiKeyHeader != "" {
				req.Header.Set("X-API-Key", tt.apiKeyHeader)
			}

			w := httptest.NewRecorder()
			auth.Middleware(nextHandler).ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
