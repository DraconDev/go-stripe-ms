package core

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/common"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/utils"
	"github.com/DraconDev/go-stripe-ms/internal/middleware"
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

// HandleItemCheckout handles POST /api/v1/checkout/item for one-time purchases
func HandleItemCheckout(db database.RepositoryInterface, stripeSecret string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "METHOD_NOT_ALLOWED", "Method not allowed", "Only POST method is allowed", "", "", "")
		return
	}

	var req ItemCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding item request: %v", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, "invalid_request", "INVALID_BODY", "Invalid request body", "Failed to decode JSON body", "", "", "")
		return
	}

	// Validate required fields
	if err := validateItemCheckoutRequest(req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "invalid_request", "VALIDATION_FAILED", err.Error(), "Request validation failed", "", "", "")
		return
	}

	// Set default quantity if not provided
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	projectID, ok := middleware.GetProjectID(r.Context())
	if !ok {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "authentication_error", "UNAUTHORIZED", "Unauthorized", "Missing or invalid authentication", "", "", "")
		return
	}

	log.Printf("HandleItemCheckout called for user: %s, product: %s, quantity: %d", req.UserID, req.ProductID, req.Quantity)

	// Find or create Stripe customer
	stripeCustomerID, err := common.FindOrCreateStripeCustomer(r.Context(), db, projectID, req.UserID, req.Email)
	if err != nil {
		log.Printf("Failed to find or create Stripe customer for user %s: %v", req.UserID, err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "api_error", "CUSTOMER_ERROR", "Failed to create or find customer", err.Error(), "user_id", "", "")
		return
	}

	// Create one-time item checkout session
	checkoutSession, err := createItemCheckoutSession(req, stripeCustomerID)
	if err != nil {
		log.Printf("Failed to create Stripe item session for user %s: %v", req.UserID, err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "api_error", "STRIPE_ERROR", "Failed to create item session", err.Error(), "", "", "")
		return
	}

	log.Printf("Created Stripe item session: %s for user: %s", checkoutSession.ID, req.UserID)

	// Return response
	writeItemCheckoutResponse(w, checkoutSession)
}

// validateItemCheckoutRequest validates the item checkout request
func validateItemCheckoutRequest(req ItemCheckoutRequest) error {
	if req.UserID == "" || req.Email == "" || req.ProductID == "" ||
		req.PriceID == "" || req.SuccessURL == "" || req.CancelURL == "" {
		return fmt.Errorf("missing required fields")
	}
	return nil
}

// createItemCheckoutSession creates a Stripe checkout session for a single item
func createItemCheckoutSession(req ItemCheckoutRequest, stripeCustomerID string) (*stripe.CheckoutSession, error) {
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
func writeItemCheckoutResponse(w http.ResponseWriter, checkoutSession *stripe.CheckoutSession) {
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
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
	}
}
