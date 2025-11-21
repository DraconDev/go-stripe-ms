package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/google/uuid"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// ProjectContextKey is the key for storing project in context
	ProjectContextKey contextKey = "project"
	// ProjectIDContextKey is the key for storing project ID in context
	ProjectIDContextKey contextKey = "project_id"
)

// APIKeyAuth middleware validates API keys and attaches project to context
type APIKeyAuth struct {
	repo database.RepositoryInterface
}

// NewAPIKeyAuth creates a new API key authentication middleware
func NewAPIKeyAuth(repo database.RepositoryInterface) *APIKeyAuth {
	return &APIKeyAuth{repo: repo}
}

// Middleware validates the X-API-Key header and attaches project to request context
func (a *APIKeyAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract API key from header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			http.Error(w, `{"error":"Missing X-API-Key header"}`, http.StatusUnauthorized)
			return
		}

		// Validate API key format
		if !strings.HasPrefix(apiKey, "proj_") {
			http.Error(w, `{"error":"Invalid API key format"}`, http.StatusUnauthorized)
			return
		}

		// Look up project by API key
		project, err := a.repo.GetProjectByAPIKey(r.Context(), apiKey)
		if err != nil {
			http.Error(w, `{"error":"Invalid API key"}`, http.StatusUnauthorized)
			return
		}

		// Check if project is active
		if !project.IsActive {
			http.Error(w, `{"error":"Project is inactive"}`, http.StatusForbidden)
			return
		}

		// Attach project to context
		ctx := context.WithValue(r.Context(), ProjectContextKey, project)
		ctx = context.WithValue(ctx, ProjectIDContextKey, project.ID)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetProjectFromContext retrieves the project from request context
func GetProjectFromContext(ctx context.Context) (*database.Project, bool) {
	project, ok := ctx.Value(ProjectContextKey).(*database.Project)
	return project, ok
}

// GetProjectIDFromContext retrieves the project ID from request context
func GetProjectIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	projectID, ok := ctx.Value(ProjectIDContextKey).(uuid.UUID)
	return projectID, ok
}
