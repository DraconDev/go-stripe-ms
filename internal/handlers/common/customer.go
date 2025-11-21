package common

import (
	"context"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/google/uuid"
)

// FindOrCreateStripeCustomer finds an existing Stripe customer or creates a new one
// This wraps the database repository method for convenience
func FindOrCreateStripeCustomer(ctx context.Context, db database.RepositoryInterface, projectID uuid.UUID, userID, email string) (string, error) {
	return db.FindOrCreateStripeCustomer(ctx, projectID, userID, email)
}
