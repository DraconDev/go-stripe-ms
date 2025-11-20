package webhooks

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/stripe/stripe-go/v72"
)

// handleCustomerSubscriptionCreated processes subscription creation events
func (h *StripeWebhookHandler) handleCustomerSubscriptionCreated(ctx context.Context, event stripe.Event) {
	var subscription struct {
		ID               string                 `json:"id"`
		Customer         struct{ ID string }   `json:"customer"`
		Status           string                 `json:"status"`
		Items            struct {
			Data []struct {
				Price struct {
					ID      string `json:"id"`
					Product string `json:"product"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
		CurrentPeriodStart int64 `json:"current_period_start"`
		CurrentPeriodEnd   int64 `json:"current_period_end"`
	}
	
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		log.Printf("Error unmarshaling subscription event: %v", err)
		return
	}

	log.Printf("Subscription created: %s for customer: %s", subscription.ID, subscription.Customer.ID)
	
	// Get customer details
	customer, err := h.getCustomerByStripeID(ctx, subscription.Customer.ID)
	if err != nil {
		log.Printf("Customer not found: %s", subscription.Customer.ID)
		return
	}

	// Extract product and price information
	var productID, priceID string
	if len(subscription.Items.Data) > 0 {
		priceID = subscription.Items.Data[0].Price.ID
		productID = subscription.Items.Data[0].Price.Product
	}

	// Create subscription in database
	err = h.db.CreateSubscription(
		ctx,
		subscription.Customer.ID,
		subscription.ID,
		productID,
		priceID,
		customer.UserID,
		subscription.Status,
		time.Unix(subscription.CurrentPeriodStart, 0),
		time.Unix(subscription.CurrentPeriodEnd, 0),
	)
	
	if err != nil {
		log.Printf("Error creating subscription in database: %v", err)
	} else {
		log.Printf("Successfully created subscription in database")
	}
}

// handleCustomerSubscriptionUpdated processes subscription update events
func (h *StripeWebhookHandler) handleCustomerSubscriptionUpdated(ctx context.Context, event stripe.Event) {
	var subscription struct {
		ID               string `json:"id"`
		Status           string `json:"status"`
		CurrentPeriodEnd int64  `json:"current_period_end"`
	}
	
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		log.Printf("Error unmarshaling subscription event: %v", err)
		return
	}

	log.Printf("Subscription updated: %s, status: %s", subscription.ID, subscription.Status)

	// Update subscription status with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	err := h.db.UpdateSubscriptionStatus(
		timeoutCtx,
		subscription.ID,
		subscription.Status,
		time.Unix(subscription.CurrentPeriodEnd, 0),
	)
	
	if err != nil {
		log.Printf("Error updating subscription in database: %v", err)
	} else {
		log.Printf("Successfully updated subscription in database")
	}
}

// handleCustomerSubscriptionDeleted processes subscription deletion events
func (h *StripeWebhookHandler) handleCustomerSubscriptionDeleted(ctx context.Context, event stripe.Event) {
	var subscription struct {
		ID               string `json:"id"`
		CurrentPeriodEnd int64  `json:"current_period_end"`
	}
	
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		log.Printf("Error unmarshaling subscription event: %v", err)
		return
	}

	log.Printf("Subscription deleted: %s", subscription.ID)

	// Update subscription status to canceled with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	err := h.db.UpdateSubscriptionStatus(
		timeoutCtx,
		subscription.ID,
		"canceled",
		time.Unix(subscription.CurrentPeriodEnd, 0),
	)
	
	if err != nil {
		log.Printf("Error updating subscription status to canceled: %v", err)
	} else {
		log.Printf("Successfully marked subscription as canceled")
	}
}

// getCustomerByStripeID retrieves customer from database by Stripe customer ID
func (h *StripeWebhookHandler) getCustomerByStripeID(ctx context.Context, stripeCustomerID string) (*database.Customer, error) {
	return h.db.GetCustomerByStripeID(ctx, stripeCustomerID)
}