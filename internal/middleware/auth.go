package middleware

import (
	"context"
	"net/http"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/google/uuid"
)

type contextKey string

const (
	// ProjectIDKey is the context key for project ID
	ProjectIDKey contextKey = "projectID"
)

// APIKeyAuth middleware validates API keys
type APIKeyAuth struct {
	repo database.RepositoryInterface
}

// NewAPIKeyAuth creates a new API key authentication middleware
func NewAPIKeyAuth(repo database.RepositoryInterface) *APIKeyAuth {
	return &APIKeyAuth{repo: repo}
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

		// Check if API key matches a project
		project, err := a.repo.GetProjectByAPIKey(r.Context(), apiKey)
		if err != nil {
			http.Error(w, `{"error":"Invalid API key"}`, http.StatusUnauthorized)
			return
		}

		if !project.IsActive {
			http.Error(w, `{"error":"Project is inactive"}`, http.StatusUnauthorized)
			return
		}

		// Store project ID in context
		ctx := context.WithValue(r.Context(), ProjectIDKey, project.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetProjectID retrieves the project ID from the context
func GetProjectID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ProjectIDKey).(uuid.UUID)
	return id, ok
}
