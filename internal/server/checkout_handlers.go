package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v72"
	checkoutsession "github.com/stripe/stripe-go/v72/checkout/session"
)

// CreateItemCheckout handles POST /api/v1/checkout/item for one-time purchases
func (s *HTTPServer) CreateItemCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID     string `json:"user_id"`
		Email      string `json:"email"`
		ProductID  string `json:"product_id"`
		PriceID    string `json:"price_id"`
		SuccessURL string `json:"success_url"`
		CancelURL  string `json:"cancel_url"`
		Quantity   int64  `json:"quantity,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding item request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.UserID == "" || req.Email == "" || req.ProductID == "" ||
		req.PriceID == "" || req.SuccessURL == "" || req.CancelURL == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
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

	checkoutSession, err := checkoutsession.New(checkoutParams)
	if err != nil {
		log.Printf("Failed to create Stripe item session for user %s: %v", req.UserID, err)
		http.Error(w, "Failed to create item session", http.StatusInternalServerError)
		return
	}

	log.Printf("Created Stripe item session: %s for user: %s", checkoutSession.ID, req.UserID)

	// Return response
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
		return
	}
}

// CreateCartCheckout handles POST /api/v1/checkout/cart for e-commerce with multiple items
func (s *HTTPServer) CreateCartCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Items  []struct {
			PriceID   string `json:"price_id"`
			Quantity  int64  `json:"quantity"`
			ProductID string `json:"product_id,omitempty"`
		} `json:"items"`
		SuccessURL string `json:"success_url"`
		CancelURL  string `json:"cancel_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding cart request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.UserID == "" || req.Email == "" || len(req.Items) == 0 ||
		req.SuccessURL == "" || req.CancelURL == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Validate cart items
	if len(req.Items) > 20 {
		http.Error(w, "Cart cannot contain more than 20 items", http.StatusBadRequest)
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

	// Create line items from cart
	lineItems := make([]*stripe.CheckoutSessionLineItemParams, len(req.Items))
	for i, item := range req.Items {
		if item.Quantity <= 0 {
			http.Error(w, fmt.Sprintf("Item %d has invalid quantity", i+1), http.StatusBadRequest)
			return
		}

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

	checkoutSession, err := checkoutsession.New(checkoutParams)
	if err != nil {
		log.Printf("Failed to create Stripe cart session for user %s: %v", req.UserID, err)
		http.Error(w, "Failed to create cart session", http.StatusInternalServerError)
		return
	}

	log.Printf("Created Stripe cart session: %s for user: %s", checkoutSession.ID, req.UserID)

	// Return response
	response := struct {
		CheckoutSessionID string `json:"checkout_session_id"`
		CheckoutURL       string `json:"checkout_url"`
		ItemCount         int    `json:"item_count"`
	}{
		CheckoutSessionID: checkoutSession.ID,
		CheckoutURL:       checkoutSession.URL,
		ItemCount:         len(req.Items),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response for cart checkout: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
		return
	}
}
