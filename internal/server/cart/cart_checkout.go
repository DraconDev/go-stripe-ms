package server

import (
	"encoding/json"
	"log"
	"net/http"
)

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

	// Validate cart checkout request
	if err := validateCartCheckoutRequest(req); err != nil {
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
	checkoutSession, err := createCartStripeSession(req, stripeCustomerID)
	if err != nil {
		log.Printf("Failed to create Stripe cart session for user %s: %v", req.UserID, err)
		http.Error(w, "Failed to create cart session", http.StatusInternalServerError)
		return
	}

	log.Printf("Created Stripe cart session: %s for user: %s", checkoutSession.ID, req.UserID)

	// Return response
	writeCartCheckoutResponse(w, checkoutSession, len(req.Items))
}