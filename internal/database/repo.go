package database

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Repository handles all database operations for billing service
type Repository struct {
	db *pgx.Conn
}

// NewRepository creates a new database repository
func NewRepository(db *pgx.Conn) *Repository {
	return &Repository{db: db}
}

// FindOrCreateStripeCustomer finds an existing customer or creates a new one
func (r *Repository) FindOrCreateStripeCustomer(ctx context.Context, userID, email string) (string, error) {
	// First, try to find existing customer
	var existingCustomerID string
	err := r.db.QueryRow(ctx, `
		SELECT stripe_customer_id 
		FROM customers 
		WHERE user_id = $1
	`, userID).Scan(&existingCustomerID)
	
	if err == nil && existingCustomerID != "" {
		log.Printf("Found existing customer for user %s: %s", userID, existingCustomerID)
		return existingCustomerID, nil
	}

	// If not found, create new customer in Stripe (this would be done by the calling service)
	// For now, we'll create a placeholder entry that will be updated when Stripe customer is created
	customerID := uuid.New()
	_, err = r.db.Exec(ctx, `
		INSERT INTO customers (id, user_id, email, stripe_customer_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			email = EXCLUDED.email,
			updated_at = EXCLUDED.updated_at
	`, customerID, userID, email, "", time.Now(), time.Now())

	if err != nil {
		return "", err
	}

	return "", nil
}

// UpdateCustomerStripeID updates the Stripe customer ID for an existing customer
func (r *Repository) UpdateCustomerStripeID(ctx context.Context, userID, stripeCustomerID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customers 
		SET stripe_customer_id = $1, updated_at = $2
		WHERE user_id = $3
	`, stripeCustomerID, time.Now(), userID)
	return err
}

// GetSubscriptionStatus retrieves subscription status for a user/product
func (r *Repository) GetSubscriptionStatus(ctx context.Context, userID, productID string) (string, string, time.Time, bool, error) {
	row := r.db.QueryRow(ctx, `
		SELECT 
			stripe_subscription_id,
			customer_id,
			status,
			current_period_end,
			true as exists
		FROM subscriptions 
		WHERE user_id = $1 AND product_id = $2
	`, userID, productID)

	return ScanSubscriptionStatus(row)
}

// CreateSubscription creates a new subscription record
func (r *Repository) CreateSubscription(ctx context.Context, customerID, stripeSubID, productID, priceID, userID, status string, periodStart, periodEnd time.Time) error {
	ID := uuid.New()
	
	// Get customer database ID
	var customerDBID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT id 
		FROM customers 
		WHERE stripe_customer_id = $1
	`, customerID).Scan(&customerDBID)
	
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO subscriptions (
			id, customer_id, user_id, product_id, price_id,
			stripe_subscription_id, status, current_period_start, current_period_end,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id, product_id) DO UPDATE SET
			stripe_subscription_id = EXCLUDED.stripe_subscription_id,
			status = EXCLUDED.status,
			current_period_start = EXCLUDED.current_period_start,
			current_period_end = EXCLUDED.current_period_end,
			updated_at = EXCLUDED.updated_at
	`, ID, customerDBID, userID, productID, priceID, stripeSubID, status, periodStart, periodEnd, time.Now(), time.Now())

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

// GetCustomerByStripeID retrieves customer by Stripe customer ID
func (r *Repository) GetCustomerByStripeID(ctx context.Context, stripeCustomerID string) (*Customer, error) {
	return ScanCustomer(r.db.QueryRow(ctx, `
		SELECT id, user_id, email, stripe_customer_id, created_at, updated_at
		FROM customers 
		WHERE stripe_customer_id = $1
	`, stripeCustomerID))
}

// GetCustomerByUserID retrieves customer by user ID
func (r *Repository) GetCustomerByUserID(ctx context.Context, userID string) (*Customer, error) {
	return ScanCustomer(r.db.QueryRow(ctx, `
		SELECT id, user_id, email, stripe_customer_id, created_at, updated_at
		FROM customers 
		WHERE user_id = $1
	`, userID))
}

// GetSubscriptionByStripeID retrieves subscription by Stripe subscription ID
func (r *Repository) GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*Subscription, error) {
	return ScanSubscription(r.db.QueryRow(ctx, `
		SELECT id, customer_id, user_id, product_id, price_id,
			stripe_subscription_id, status, current_period_start, current_period_end,
			created_at, updated_at
		FROM subscriptions 
		WHERE stripe_subscription_id = $1
	`, stripeSubID))
}

// InitializeTables creates the necessary database tables
func (r *Repository) InitializeTables(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS customers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id VARCHAR(255) UNIQUE NOT NULL,
			email VARCHAR(255) NOT NULL,
			stripe_customer_id VARCHAR(255) UNIQUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
			UNIQUE(user_id, product_id)
		)`,
		
		`CREATE INDEX IF NOT EXISTS idx_customers_user_id ON customers(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_customers_stripe_id ON customers(stripe_customer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe_id ON subscriptions(stripe_subscription_id)`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_product_id ON subscriptions(product_id)`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(ctx, query); err != nil {
			return err
		}
	}

	return nil
}
