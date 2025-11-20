package subscription

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/common"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/core"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/utils"
	"github.com/stripe/stripe-go/v72"
	checkoutsession "github.com/stripe/stripe-go/v72/checkout/session"
)

// SubscriptionCheckoutRequest represents the request for subscription checkout
type SubscriptionCheckoutRequest struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	ProductID  string `json:"product_id"`
	PriceID    string `json:"price_id"`
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
}

// HandleSubscriptionCheckout handles POST /api/v1/checkout/subscription
func HandleSubscriptionCheckout(db database.RepositoryInterface, stripeSecret string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubscriptionCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding subscription request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate using common validation
	if err := validateSubscriptionRequest(req); err != nil {
		log.Printf("Validation error for subscription request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("HandleSubscriptionCheckout called for user: %s, product: %s", req.UserID, req.ProductID)

	// Find or create Stripe customer
	stripeCustomerID, err := common.FindOrCreateStripeCustomer(r.Context(), db, req.UserID, req.Email)
	if err != nil {
		log.Printf("Failed to find or create Stripe customer for user %s: %v", req.UserID, err)
		http.Error(w, "Failed to create or find customer", http.StatusInternalServerError)
		return
	}

	// Use checkout session builder
	builder := core.NewCheckoutSessionBuilder(stripeCustomerID, req.UserID, req.SuccessURL, req.CancelURL, "subscription")
	builder.AddLineItem(req.PriceID, 1)

	checkoutParams := builder.Build(stripe.CheckoutSessionModeSubscription)
	checkoutParams.AddMetadata("product_id", req.ProductID)

	checkoutSession, err := checkoutsession.New(checkoutParams)
	if err != nil {
		log.Printf("Failed to create Stripe checkout session for user %s: %v", req.UserID, err)
		http.Error(w, "Failed to create checkout session", http.StatusInternalServerError)
		return
	}

	log.Printf("Created Stripe checkout session: %s for user: %s", checkoutSession.ID, req.UserID)

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
		log.Printf("Error encoding response for subscription checkout: %v", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
		return
	}
}

// validateSubscriptionRequest validates subscription checkout requests
func validateSubscriptionRequest(req SubscriptionCheckoutRequest) error {
	if req.UserID == "" || req.Email == "" || req.ProductID == "" ||
		req.PriceID == "" || req.SuccessURL == "" || req.CancelURL == "" {
		return &utils.ValidationError{Field: "request", Message: "user_id, email, product_id, price_id, success_url, and cancel_url are required"}
	}
	return nil
}
