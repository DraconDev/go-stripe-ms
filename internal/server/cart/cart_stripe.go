package server

import (
	"fmt"
	"log"

	"github.com/stripe/stripe-go/v72"
	checkoutsession "github.com/stripe/stripe-go/v72/checkout/session"
)

// createCartStripeSession creates a Stripe checkout session for multiple items
func createCartStripeSession(req CartCheckoutRequest, stripeCustomerID string) (*stripe.CheckoutSession, error) {
	// Create line items from cart
	lineItems := make([]*stripe.CheckoutSessionLineItemParams, len(req.Items))
	for i, item := range req.Items {
		lineItems[i] = &stripe.CheckoutSessionLineItemParams{
			Price:    stripe.String(item.PriceID),
			Quantity: stripe.Int64(item.Quantity),
		}
	}

	// Create cart checkout session
	checkoutParams := &stripe.CheckoutSessionParams{
		Customer:                 stripe.String(stripeCustomerID),
		Mode:                     stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems:                lineItems,
		SuccessURL:               stripe.String(req.SuccessURL),
		CancelURL:                stripe.String(req.CancelURL),
		ClientReferenceID:        stripe.String(req.UserID),
		AllowPromotionCodes:      stripe.Bool(true),
		BillingAddressCollection: stripe.String(string(stripe.CheckoutSessionBillingAddressCollectionRequired)),
	}

	// Add metadata
	checkoutParams.AddMetadata("user_id", req.UserID)
	checkoutParams.AddMetadata("payment_type", "cart")
	checkoutParams.AddMetadata("item_count", fmt.Sprintf("%d", len(req.Items)))

	session, err := checkoutsession.New(checkoutParams)
	if err != nil {
		log.Printf("Failed to create Stripe cart session: %v", err)
		return nil, err
	}

	return session, nil
}