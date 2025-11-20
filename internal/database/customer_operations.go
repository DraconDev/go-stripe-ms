package database

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
)

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