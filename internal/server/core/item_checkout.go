package core

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v72"
	checkoutsession "github.com/stripe/stripe-go/v72/checkout/session"
)

// ItemCheckoutRequest represents the request structure for single item checkout
type ItemCheckoutRequest struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	ProductID  string `json:"product_id"`
	PriceID    string `json:"price_id"`
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
	Quantity   int64  `json:"quantity,omitempty"`
}

// CreateItemCheckout handles POST /api/v1/checkout/item for one-time purchases
func (s *HTTPServer) CreateItemCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ItemCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding item request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if err := s.validateItemCheckoutRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set default quantity if not provided
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	log.Printf("CreateItemCheckout called for user: %s, product: %s, quantity: %d", req.UserID, req.ProductID, req.Quantity)

	// Find or create Stripe customer
	stripeCustomerID, err := s.findOrCreateStripeCustomer(r.Context(), req.UserID, req.Email)
	if err != nil {
		log.Printf("Failed to find or create Stripe customer for user %s: %v", req.UserID, err)
		http.Error(w, "Failed to create or find customer", http.StatusInternalServerError)
		return
	}

	// Create one-time item checkout session
	checkoutSession, err := s.createItemCheckoutSession(req, stripeCustomerID)
	if err != nil {
		log.Printf("Failed to create Stripe item session for user %s: %v", req.UserID, err)
		http.Error(w, "Failed to create item session", http.StatusInternalServerError)
		return
	}

	log.Printf("Created Stripe item session: %s for user: %s", checkoutSession.ID, req.UserID)

	// Return response
	s.writeItemCheckoutResponse(w, checkoutSession)
}

// validateItemCheckoutRequest validates the item checkout request
func (s *HTTPServer) validateItemCheckoutRequest(req ItemCheckoutRequest) error {
	if req.UserID == "" || req.Email == "" || req.ProductID == "" ||
		req.PriceID == "" || req.SuccessURL == "" || req.CancelURL == "" {
		return fmt.Errorf("missing required fields")
	}
	return nil
}

// createItemCheckoutSession creates a Stripe checkout session for a single item
func (s *HTTPServer) createItemCheckoutSession(req ItemCheckoutRequest, stripeCustomerID string) (*stripe.CheckoutSession, error) {
	checkoutParams := &stripe.CheckoutSessionParams{
		Customer: stripe.String(stripeCustomerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.PriceID),
				Quantity: stripe.Int64(req.Quantity),
			},
		},
		SuccessURL:               stripe.String(req.SuccessURL),
		CancelURL:                stripe.String(req.CancelURL),
		ClientReferenceID:        stripe.String(req.UserID),
		AllowPromotionCodes:      stripe.Bool(true),
		BillingAddressCollection: stripe.String(string(stripe.CheckoutSessionBillingAddressCollectionRequired)),
	}

	// Add metadata
	checkoutParams.AddMetadata("user_id", req.UserID)
	checkoutParams.AddMetadata("product_id", req.ProductID)
	checkoutParams.AddMetadata("payment_type", "item")

	return checkoutsession.New(checkoutParams)
}

// writeItemCheckoutResponse writes the response for item checkout
func (s *HTTPServer) writeItemCheckoutResponse(w http.ResponseWriter, checkoutSession *stripe.CheckoutSession) {
	response := struct {
		CheckoutSessionID string `json:"checkout_session_id"`
		CheckoutURL       string `json:"checkout_url"`
	}{
		CheckoutSessionID: checkoutSession.ID,
		CheckoutURL:       checkoutSession.URL,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response for item checkout: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
	}
}