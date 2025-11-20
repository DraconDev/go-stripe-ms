package common

import (
	"context"

	"github.com/DraconDev/go-stripe-ms/internal/database"
)

// FindOrCreateStripeCustomer finds an existing Stripe customer or creates a new one
// This wraps the database repository method for convenience
func FindOrCreateStripeCustomer(ctx context.Context, db database.RepositoryInterface, userID, email string) (string, error) {
	return db.FindOrCreateStripeCustomer(ctx, userID, email)
}
