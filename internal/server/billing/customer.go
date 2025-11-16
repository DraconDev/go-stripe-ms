package billing

import (
	"context"
	"fmt"
	"log"

	"github.com/stripe/stripe-go/v72"
	customer "github.com/stripe/stripe-go/v72/customer"
)

// findOrCreateStripeCustomer finds an existing Stripe customer or creates a new one
func (s *HTTPServer) findOrCreateStripeCustomer(ctx context.Context, userID, email string) (string, error) {
	// Check database for existing customer
	existingCustomer, err := s.db.GetCustomerByUserID(ctx, userID)
	if err == nil && existingCustomer != nil && existingCustomer.StripeCustomerID != "" {
		log.Printf("Found existing Stripe customer for user %s: %s", userID, existingCustomer.StripeCustomerID)
		return existingCustomer.StripeCustomerID, nil
	}

	// Create new Stripe customer
	customerParams := &stripe.CustomerParams{
		Email: stripe.String(email),
	}

	// Add metadata
	customerParams.AddMetadata("user_id", userID)

	stripeCustomer, err := customer.New(customerParams)
	if err != nil {
		log.Printf("Failed to create Stripe customer: %v", err)
		return "", fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	// Update database with Stripe customer ID
	err = s.db.UpdateCustomerStripeID(ctx, userID, stripeCustomer.ID)
	if err != nil {
		log.Printf("Failed to update customer Stripe ID in database: %v", err)
		return "", fmt.Errorf("failed to update customer record: %w", err)
	}

	log.Printf("Created new Stripe customer: %s for user: %s", stripeCustomer.ID, userID)
	return stripeCustomer.ID, nil
}