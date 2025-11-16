package core

import (
	"fmt"

	"github.com/stripe/stripe-go/v72"
)

// CheckoutSessionBuilder builds Stripe checkout sessions with common configuration
type CheckoutSessionBuilder struct {
	customerID  string
	userID      string
	successURL  string
	cancelURL   string
	paymentType string
	lineItems   []*stripe.CheckoutSessionLineItemParams
}

// NewCheckoutSessionBuilder creates a new checkout session builder
func NewCheckoutSessionBuilder(customerID, userID, successURL, cancelURL, paymentType string) *CheckoutSessionBuilder {
	return &CheckoutSessionBuilder{
		customerID:  customerID,
		userID:      userID,
		successURL:  successURL,
		cancelURL:   cancelURL,
		paymentType: paymentType,
		lineItems:   make([]*stripe.CheckoutSessionLineItemParams, 0),
	}
}

// AddLineItem adds a line item to the checkout session
func (b *CheckoutSessionBuilder) AddLineItem(priceID string, quantity int64) *CheckoutSessionBuilder {
	b.lineItems = append(b.lineItems, &stripe.CheckoutSessionLineItemParams{
		Price:    stripe.String(priceID),
		Quantity: stripe.Int64(quantity),
	})
	return b
}

// Build creates the final checkout session parameters
func (b *CheckoutSessionBuilder) Build(mode stripe.CheckoutSessionMode) *stripe.CheckoutSessionParams {
	params := &stripe.CheckoutSessionParams{
		Customer:                 stripe.String(b.customerID),
		Mode:                     stripe.String(string(mode)),
		SuccessURL:               stripe.String(b.successURL),
		CancelURL:                stripe.String(b.cancelURL),
		ClientReferenceID:        stripe.String(b.userID),
		AllowPromotionCodes:      stripe.Bool(true),
		BillingAddressCollection: stripe.String(string(stripe.CheckoutSessionBillingAddressCollectionRequired)),
		LineItems:                b.lineItems,
	}

	// Add common metadata
	params.AddMetadata("user_id", b.userID)
	params.AddMetadata("payment_type", b.paymentType)

	return params
}

// Common checkout request validation
func validateCommonCheckoutRequest(req struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
}) error {
	if req.UserID == "" || req.Email == "" || req.SuccessURL == "" || req.CancelURL == "" {
		return &ValidationError{Field: "request", Message: "user_id, email, success_url, and cancel_url are required"}
	}
	return nil
}

// Common cart validation
func validateCartRequest(req struct {
	Items []struct {
		PriceID  string `json:"price_id"`
		Quantity int64  `json:"quantity"`
	} `json:"items"`
}) error {
	if len(req.Items) == 0 {
		return &ValidationError{Field: "items", Message: "at least one item is required"}
	}
	
	if len(req.Items) > 20 {
		return &ValidationError{Field: "items", Message: "cart cannot contain more than 20 items"}
	}
	
	for i, item := range req.Items {
		if item.PriceID == "" {
			return &ValidationError{Field: "items", Message: fmt.Sprintf("item %d price_id is required", i+1)}
		}
		if item.Quantity <= 0 {
			return &ValidationError{Field: "items", Message: fmt.Sprintf("item %d quantity must be greater than 0", i+1)}
		}
	}
	
	return nil
}