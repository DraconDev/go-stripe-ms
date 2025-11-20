package database

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

// WithTestDatabase runs a test with a real database
func WithTestDatabase(t *testing.T, testFunc func(*testing.T, *TestDatabase)) {
	t.Helper()
	
	// Check if database is configured
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database tests")
	}

	testDB := NewTestDatabase(t)
	defer testDB.Cleanup(t)
	
	// Setup database
	testDB.Setup(t)
	
	// Run test
	testFunc(t, testDB)
}

// WithRealDatabase runs a test with the real production database
func WithRealDatabase(t *testing.T, testFunc func(*testing.T, *Repository)) {
	t.Helper()
	
	// Check if database is configured
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping real database tests")
	}

	ctx := context.Background()
	
	// Connect to real database
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to real database: %v", err)
	}
	defer conn.Close(ctx)

	// Create repository
	repo := NewRepository(conn)
	
	// Run test
	testFunc(t, repo)
}