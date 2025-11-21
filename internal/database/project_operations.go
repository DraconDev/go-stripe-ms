package database

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
)

// CreateProject creates a new project with a generated API key
func (r *Repository) CreateProject(ctx context.Context, name, webhookURL string) (*Project, error) {
	apiKey, err := GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	id := uuid.New()
	project := &Project{
		ID:         id,
		Name:       name,
		APIKey:     apiKey,
		WebhookURL: webhookURL,
		IsActive:   true,
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO projects (id, name, api_key, webhook_url, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`, project.ID, project.Name, project.APIKey, project.WebhookURL, project.IsActive)

	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return project, nil
}

// GetProjectByAPIKey retrieves a project by its API key
func (r *Repository) GetProjectByAPIKey(ctx context.Context, apiKey string) (*Project, error) {
	return ScanProject(r.db.QueryRow(ctx, `
		SELECT id, name, api_key, webhook_url, is_active, created_at, updated_at
		FROM projects
		WHERE api_key = $1 AND is_active = true
	`, apiKey))
}

// GetProjectByID retrieves a project by its ID
func (r *Repository) GetProjectByID(ctx context.Context, projectID uuid.UUID) (*Project, error) {
	return ScanProject(r.db.QueryRow(ctx, `
		SELECT id, name, api_key, webhook_url, is_active, created_at, updated_at
		FROM projects
		WHERE id = $1
	`, projectID))
}

// ListProjects retrieves all projects
func (r *Repository) ListProjects(ctx context.Context) ([]*Project, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, api_key, webhook_url, is_active, created_at, updated_at
		FROM projects
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		project, err := ScanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	return projects, rows.Err()
}

// GenerateAPIKey generates a secure random API key
func GenerateAPIKey() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	// Format: proj_<43_random_chars>
	encoded := base64.URLEncoding.EncodeToString(b)
	return "proj_" + encoded[:43], nil
}
