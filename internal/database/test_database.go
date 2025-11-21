package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TestDatabase manages database testing with real PostgreSQL
type TestDatabase struct {
	Conn *pgx.Conn
	Repo *Repository
	ctx  context.Context
}

// NewTestDatabase creates a new test database connection
func NewTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	ctx := context.Background()

	// Automatically load environment variables
	autoLoadEnv()

	// Get database connection string
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		t.Fatal("DATABASE_URL environment variable is required for integration tests")
	}

	// Connect to test database
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Create repository
	repo := NewRepository(conn)

	return &TestDatabase{
		Conn: conn,
		Repo: repo,
		ctx:  ctx,
	}
}

// Setup creates tables and initializes the test database
func (td *TestDatabase) Setup(t *testing.T) {
	t.Helper()

	// Initialize database tables
	if err := td.Repo.InitializeTables(td.ctx); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	log.Println("Test database setup completed")
}

// Cleanup closes connections and cleans up test data
func (td *TestDatabase) Cleanup(t *testing.T) {
	t.Helper()

	if td.Conn != nil {
		// Clean up test data by truncating tables in reverse dependency order
		// This ensures foreign key constraints are respected
		_, err := td.Conn.Exec(td.ctx, `
			TRUNCATE TABLE subscriptions, customers, projects CASCADE;
		`)
		if err != nil {
			t.Logf("Warning: failed to clean up test data: %v", err)
		}

		td.Conn.Close(td.ctx)
	}

	log.Println("Test database cleanup completed")
}

// CreateTestProject creates a test project in the database
func (td *TestDatabase) CreateTestProject(project *Project) error {
	_, err := td.Conn.Exec(td.ctx, `
		INSERT INTO projects (id, name, api_key, webhook_url, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (api_key) DO UPDATE SET
			name = EXCLUDED.name,
			webhook_url = EXCLUDED.webhook_url,
			updated_at = EXCLUDED.updated_at
	`, project.ID, project.Name, project.APIKey, project.WebhookURL, project.IsActive, project.CreatedAt, project.UpdatedAt)
	return err
}

// CreateTestCustomer creates a test customer in the database
func (td *TestDatabase) CreateTestCustomer(customer *Customer) error {
	err := td.Conn.QueryRow(td.ctx, `
		INSERT INTO customers (id, project_id, user_id, email, stripe_customer_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (project_id, user_id) DO UPDATE SET
			id = EXCLUDED.id,
			email = EXCLUDED.email,
			stripe_customer_id = EXCLUDED.stripe_customer_id,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`, customer.ID, customer.ProjectID, customer.UserID, customer.Email, customer.StripeCustomerID, customer.CreatedAt, customer.UpdatedAt).Scan(&customer.ID)
	return err
}

// CreateTestSubscription creates a test subscription in the database
func (td *TestDatabase) CreateTestSubscription(subscription *Subscription) error {
	// First ensure customer exists
	customer := &Customer{
		ProjectID:        subscription.ProjectID,
		UserID:           subscription.UserID,
		Email:            "test@example.com",
		StripeCustomerID: "cus_test123",
		CreatedAt:        subscription.CreatedAt,
		UpdatedAt:        subscription.UpdatedAt,
	}
	if err := td.CreateTestCustomer(customer); err != nil {
		return fmt.Errorf("failed to create test customer: %w", err)
	}

	// Get customer ID
	var customerID string
	err := td.Conn.QueryRow(td.ctx, `
		SELECT id::text FROM customers WHERE project_id = $1 AND user_id = $2
	`, subscription.ProjectID, subscription.UserID).Scan(&customerID)
	if err != nil {
		return err
	}

	_, err = td.Conn.Exec(td.ctx, `
		INSERT INTO subscriptions (
			project_id, customer_id, user_id, product_id, price_id,
			stripe_subscription_id, status, current_period_start, current_period_end,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		ON CONFLICT (project_id, user_id, product_id) DO UPDATE SET
			stripe_subscription_id = EXCLUDED.stripe_subscription_id,
			status = EXCLUDED.status,
			current_period_start = EXCLUDED.current_period_start,
			current_period_end = EXCLUDED.current_period_end,
			updated_at = EXCLUDED.updated_at
	`, subscription.ProjectID, customerID, subscription.UserID, subscription.ProductID, subscription.PriceID,
		subscription.StripeSubscriptionID, subscription.Status,
		subscription.CurrentPeriodStart, subscription.CurrentPeriodEnd,
		subscription.CreatedAt, subscription.UpdatedAt)

	return err
}

// CreateTestData creates test customers and subscriptions
func (td *TestDatabase) CreateTestData() (*Project, error) {
	// Use timestamp to ensure uniqueness across parallel test runs
	timestamp := time.Now().UnixNano()

	// Create test project
	projectID := uuid.New()
	project := &Project{
		ID:        projectID,
		Name:      fmt.Sprintf("Test Project %d", timestamp),
		APIKey:    fmt.Sprintf("sk_test_%d_%s", timestamp, uuid.New().String()[:8]),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := td.CreateTestProject(project); err != nil {
		return nil, fmt.Errorf("failed to create test project: %w", err)
	}

	// Create test customer
	customerID := uuid.New()
	customer := &Customer{
		ID:               customerID,
		ProjectID:        projectID,
		UserID:           fmt.Sprintf("test_user_%d", timestamp),
		Email:            fmt.Sprintf("test_%d@example.com", timestamp),
		StripeCustomerID: fmt.Sprintf("cus_test_%d", timestamp),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := td.CreateTestCustomer(customer); err != nil {
		return nil, fmt.Errorf("failed to create test customer: %w", err)
	}

	// Create test subscription
	subscription := &Subscription{
		ProjectID:            projectID,
		CustomerID:           customerID,
		UserID:               customer.UserID,
		StripeSubscriptionID: fmt.Sprintf("sub_test_%d", timestamp),
		ProductID:            "premium_plan",
		PriceID:              "price_123",
		Status:               "active",
		CurrentPeriodStart:   time.Now(),
		CurrentPeriodEnd:     time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := td.CreateTestSubscription(subscription); err != nil {
		return nil, fmt.Errorf("failed to create test subscription: %w", err)
	}

	return project, nil
}
