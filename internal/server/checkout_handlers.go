package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v72"
	checkoutsession "github.com/stripe/stripe-go/v72/checkout/session"
)

// CartItem represents an individual item in a cart
type CartItem struct {
	PriceID   string `json:"price_id"`
	Quantity  int64  `json:"quantity"`
	ProductID string `json:"product_id,omitempty"`
}

// CartCheckoutRequest represents the request structure for cart checkout
type CartCheckoutRequest struct {
	UserID     string     `json:"user_id"`
	Email      string     `json:"email"`
	Items      []CartItem `json:"items"`
	SuccessURL string     `json:"success_url"`
	CancelURL  string     `json:"cancel_url"`
}

// CreateCartCheckout handles POST /api/v1/checkout/cart for e-commerce with multiple items
func (s *HTTPServer) CreateCartCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CartCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding cart request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if err := s.validateCartCheckoutRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("CreateCartCheckout called for user: %s, items: %d", req.UserID, len(req.Items))

	// Find or create Stripe customer
	stripeCustomerID, err := s.findOrCreateStripeCustomer(r.Context(), req.UserID, req.Email)
	if err != nil {
		log.Printf("Failed to find or create Stripe customer for user %s: %v", req.UserID, err)
		http.Error(w, "Failed to create or find customer", http.StatusInternalServerError)
		return
	}

	// Create cart checkout session
	checkoutSession, err := s.createCartCheckoutSession(req, stripeCustomerID)
	if err != nil {
		log.Printf("Failed to create Stripe cart session for user %s: %v", req.UserID, err)
		http.Error(w, "Failed to create cart session", http.StatusInternalServerError)
		return
	}

	log.Printf("Created Stripe cart session: %s for user: %s", checkoutSession.ID, req.UserID)

	// Return response
	s.writeCartCheckoutResponse(w, checkoutSession, len(req.Items))
}

// validateCartCheckoutRequest validates the cart checkout request
func (s *HTTPServer) validateCartCheckoutRequest(req CartCheckoutRequest) error {
	if req.UserID == "" || req.Email == "" || len(req.Items) == 0 ||
		req.SuccessURL == "" || req.CancelURL == "" {
		return fmt.Errorf("missing required fields")
	}

	// Validate cart items
	if len(req.Items) > 20 {
		return fmt.Errorf("cart cannot contain more than 20 items")
	}

	// Validate each item quantity
	for i, item := range req.Items {
		if item.Quantity <= 0 {
			return fmt.Errorf("item %d has invalid quantity", i+1)
		}
	}

	return nil
}

// createCartCheckoutSession creates a Stripe checkout session for multiple items
func (s *HTTPServer) createCartCheckoutSession(req CartCheckoutRequest, stripeCustomerID string) (*stripe.CheckoutSession, error) {
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

	return checkoutsession.New(checkoutParams)
}

// writeCartCheckoutResponse writes the response for cart checkout
func (s *HTTPServer) writeCartCheckoutResponse(w http.ResponseWriter, checkoutSession *stripe.CheckoutSession, itemCount int) {
	response := struct {
		CheckoutSessionID string `json:"checkout_session_id"`
		CheckoutURL       string `json:"checkout_url"`
		ItemCount         int    `json:"item_count"`
	}{
		CheckoutSessionID: checkoutSession.ID,
		CheckoutURL:       checkoutSession.URL,
		ItemCount:         itemCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response for cart checkout: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
	}
}
