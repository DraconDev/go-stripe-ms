package common

import (
	"context"
	"log"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
)

// FindOrCreateStripeCustomer finds an existing Stripe customer or creates a new one
// This is a shared utility function used by multiple handlers
func FindOrCreateStripeCustomer(ctx context.Context, db database.RepositoryInterface, userID, email string) (string, error) {
	// Check if customer already exists in database
	existingCustomerID, err := db.GetStripeCustomerID(ctx, userID)
	if err != nil {
		log.Printf("Error checking for existing customer: %v", err)
		return "", err
	}

	// If customer exists, return their ID
	if existingCustomerID != "" {
		log.Printf("Found existing Stripe customer: %s for user: %s", existingCustomerID, userID)
		return existingCustomerID, nil
	}

	// Create new Stripe customer
	log.Printf("Creating new Stripe customer for user: %s", userID)
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
	}
	params.AddMetadata("user_id", userID)

	newCustomer, err := customer.New(params)
	if err != nil {
		log.Printf("Failed to create Stripe customer: %v", err)
		return "", err
	}

	// Store customer ID in database
	if err := db.SaveStripeCustomer(ctx, userID, newCustomer.ID); err != nil {
		log.Printf("Failed to save customer ID to database: %v", err)
		return "", err
	}

	log.Printf("Created new Stripe customer: %s", newCustomer.ID)
	return newCustomer.ID, nil
}
