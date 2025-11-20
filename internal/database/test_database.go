package database

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// Cleanup closes connections
func (td *TestDatabase) Cleanup(t *testing.T) {
	t.Helper()
	
	if td.Conn != nil {
		td.Conn.Close(td.ctx)
	}
	
	log.Println("Test database cleanup completed")
}

// CreateTestCustomer creates a test customer in the database
func (td *TestDatabase) CreateTestCustomer(customer *Customer) error {
	_, err := td.Conn.Exec(td.ctx, `
		INSERT INTO customers (user_id, email, stripe_customer_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			email = EXCLUDED.email,
			stripe_customer_id = EXCLUDED.stripe_customer_id,
			updated_at = EXCLUDED.updated_at
	`, customer.UserID, customer.Email, customer.StripeCustomerID, customer.CreatedAt, customer.UpdatedAt)
	return err
}

// CreateTestSubscription creates a test subscription in the database
func (td *TestDatabase) CreateTestSubscription(subscription *Subscription) error {
	// First ensure customer exists
	customer := &Customer{
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
		SELECT id::text FROM customers WHERE user_id = $1
	`, subscription.UserID).Scan(&customerID)
	if err != nil {
		return err
	}

	_, err = td.Conn.Exec(td.ctx, `
		INSERT INTO subscriptions (
			customer_id, user_id, product_id, price_id,
			stripe_subscription_id, status, current_period_start, current_period_end,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
		ON CONFLICT (user_id, product_id) DO UPDATE SET
			stripe_subscription_id = EXCLUDED.stripe_subscription_id,
			status = EXCLUDED.status,
			current_period_start = EXCLUDED.current_period_start,
			current_period_end = EXCLUDED.current_period_end,
			updated_at = EXCLUDED.updated_at
	`, customerID, subscription.UserID, subscription.ProductID, subscription.PriceID,
		subscription.StripeSubscriptionID, subscription.Status,
		subscription.CurrentPeriodStart, subscription.CurrentPeriodEnd,
		subscription.CreatedAt, subscription.UpdatedAt)
	
	return err
}





// CreateTestData creates test customers and subscriptions
func (td *TestDatabase) CreateTestData() error {
	// Create test customer
	customer := &Customer{
		UserID:           "test_user_123",
		Email:            "test@example.com",
		StripeCustomerID: "cus_test123",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	
	if err := td.CreateTestCustomer(customer); err != nil {
		return err
	}
	
	// Create test subscription
	subscription := &Subscription{
		UserID:               "test_user_123",
		ProductID:            "premium_plan",
		PriceID:              "price_test123",
		StripeSubscriptionID: "sub_test123",
		Status:               "active",
		CurrentPeriodStart:   time.Now(),
		CurrentPeriodEnd:     time.Now().AddDate(0, 0, 30),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
	
	return td.CreateTestSubscription(subscription)
}
