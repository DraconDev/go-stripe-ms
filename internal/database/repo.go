package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RepositoryInterface defines the interface for database operations
type RepositoryInterface interface {
	// Customer operations
	FindOrCreateStripeCustomer(ctx context.Context, projectID uuid.UUID, userID, email string) (string, error)
	UpdateCustomerStripeID(ctx context.Context, projectID uuid.UUID, userID, stripeCustomerID string) error
	GetCustomerByStripeID(ctx context.Context, stripeCustomerID string) (*Customer, error)
	GetCustomerByUserID(ctx context.Context, projectID uuid.UUID, userID string) (*Customer, error)

	// Subscription operations
	GetSubscriptionStatus(ctx context.Context, projectID uuid.UUID, userID, productID string) (string, string, time.Time, bool, error)
	CreateSubscription(ctx context.Context, projectID uuid.UUID, customerID, stripeSubID, productID, priceID, userID, status string, periodStart, periodEnd time.Time) error
	UpdateSubscriptionStatus(ctx context.Context, stripeSubID, status string, periodEnd time.Time) error
	GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*Subscription, error)

	// Registered products operations
	CreateRegisteredProduct(ctx context.Context, product *RegisteredProduct) error
	GetRegisteredProductsByProject(ctx context.Context, projectName string) ([]*RegisteredProduct, error)
	ProductExistsForProject(ctx context.Context, projectName, planName string) (bool, string, error)
	GetRegisteredProductByStripeID(ctx context.Context, stripeProductID string) (*RegisteredProduct, error)

	// Project operations
	CreateProject(ctx context.Context, name, webhookURL string) (*Project, error)
	GetProjectByAPIKey(ctx context.Context, apiKey string) (*Project, error)
	GetProjectByID(ctx context.Context, projectID uuid.UUID) (*Project, error)
	ListProjects(ctx context.Context) ([]*Project, error)

	// Database initialization
	InitializeTables(ctx context.Context) error
}

// Repository handles all database operations for billing service
type Repository struct {
	db *pgx.Conn
}

// NewRepository creates a new database repository
func NewRepository(db *pgx.Conn) *Repository {
	return &Repository{db: db}
}

// GetSubscriptionStatus retrieves subscription status for a user/product
func (r *Repository) GetSubscriptionStatus(ctx context.Context, projectID uuid.UUID, userID, productID string) (string, string, time.Time, bool, error) {
	row := r.db.QueryRow(ctx, `
		SELECT 
			stripe_subscription_id,
			customer_id,
			current_period_end,
			TRUE as exists
		FROM subscriptions 
		WHERE project_id = $1 AND user_id = $2 AND product_id = $3
	`, projectID, userID, productID)

	return ScanSubscriptionStatus(row)
}

// CreateSubscription creates or updates a subscription
func (r *Repository) CreateSubscription(ctx context.Context, projectID uuid.UUID, customerID, stripeSubID, productID, priceID, userID, status string, periodStart, periodEnd time.Time) error {
	_, err := r.db.Exec(ctx, `
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
	`, projectID, customerID, userID, productID, priceID, stripeSubID, status, periodStart, periodEnd, time.Now(), time.Now())

	return err
}

// UpdateSubscriptionStatus updates subscription status and period end
func (r *Repository) UpdateSubscriptionStatus(ctx context.Context, stripeSubID, status string, periodEnd time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE subscriptions 
		SET status = $1, current_period_end = $2, updated_at = $3
		WHERE stripe_subscription_id = $4
	`, status, periodEnd, time.Now(), stripeSubID)

	return err
}

// GetSubscriptionByStripeID retrieves subscription by Stripe subscription ID
func (r *Repository) GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*Subscription, error) {
	return ScanSubscription(r.db.QueryRow(ctx, `
		SELECT id, project_id, customer_id, user_id, product_id, price_id,
			stripe_subscription_id, status, current_period_start, current_period_end,
			created_at, updated_at
		FROM subscriptions 
		WHERE stripe_subscription_id = $1
	`, stripeSubID))
}

// GetCustomerByUserID retrieves a customer by User ID
func (r *Repository) GetCustomerByUserID(ctx context.Context, projectID uuid.UUID, userID string) (*Customer, error) {
	return ScanCustomer(r.db.QueryRow(ctx, `
		SELECT id, project_id, user_id, email, stripe_customer_id, created_at, updated_at
		FROM customers 
		WHERE project_id = $1 AND user_id = $2
	`, projectID, userID))
}

// InitializeTables creates the necessary database tables
func (r *Repository) InitializeTables(ctx context.Context) error {
	queries := []string{
		// Projects table for multi-tenant API key authentication
		`CREATE TABLE IF NOT EXISTS projects (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			api_key VARCHAR(64) UNIQUE NOT NULL,
			webhook_url TEXT,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS customers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
			user_id VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			stripe_customer_id VARCHAR(255) UNIQUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(project_id, user_id)
		)`,

		`CREATE TABLE IF NOT EXISTS subscriptions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
			customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
			user_id VARCHAR(255) NOT NULL,
			product_id VARCHAR(255) NOT NULL,
			price_id VARCHAR(255) NOT NULL,
			stripe_subscription_id VARCHAR(255) UNIQUE NOT NULL,
			status VARCHAR(100) NOT NULL,
			current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
			current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(project_id, user_id, product_id)
		)`,

		`CREATE TABLE IF NOT EXISTS registered_products (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			project_name VARCHAR(255) NOT NULL,
			plan_name VARCHAR(255) NOT NULL,
			stripe_product_id VARCHAR(255) NOT NULL UNIQUE,
			stripe_price_monthly VARCHAR(255),
			stripe_price_yearly VARCHAR(255),
			monthly_amount INT,
			yearly_amount INT,
			currency VARCHAR(10) DEFAULT 'usd',
			description TEXT,
			features JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(project_name, plan_name)
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_projects_api_key ON projects(api_key)`,
		`CREATE INDEX IF NOT EXISTS idx_customers_project_id ON customers(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_customers_user_id ON customers(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_customers_stripe_id ON customers(stripe_customer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_project_id ON subscriptions(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe_id ON subscriptions(stripe_subscription_id)`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_product_id ON subscriptions(product_id)`,
		`CREATE INDEX IF NOT EXISTS idx_registered_products_project ON registered_products(project_name)`,
		`CREATE INDEX IF NOT EXISTS idx_registered_products_stripe_id ON registered_products(stripe_product_id)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(ctx, query); err != nil {
			return err
		}
	}

	return nil
}
