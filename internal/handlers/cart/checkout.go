package cart

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/common"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/utils"
	"github.com/DraconDev/go-stripe-ms/internal/middleware"
)

// HandleCartCheckout handles POST /api/v1/checkout/cart for e-commerce with multiple items
func HandleCartCheckout(db database.RepositoryInterface, stripeSecret string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "METHOD_NOT_ALLOWED", "Method not allowed", "Only POST method is allowed", "", "", "")
		return
	}

	var req CartCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding cart request: %v", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, "invalid_request", "INVALID_BODY", "Invalid request body", "Failed to decode JSON body", "", "", "")
		return
	}

	// Validate cart checkout request
	if err := validateCartCheckoutRequest(req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "invalid_request", "VALIDATION_FAILED", err.Error(), "Request validation failed", "", "", "")
		return
	}

	projectID, ok := middleware.GetProjectID(r.Context())
	if !ok {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "authentication_error", "UNAUTHORIZED", "Unauthorized", "Missing or invalid authentication", "", "", "")
		return
	}

	log.Printf("HandleCartCheckout called for user: %s, items: %d", req.UserID, len(req.Items))

	// Find or create Stripe customer
	stripeCustomerID, err := common.FindOrCreateStripeCustomer(r.Context(), db, projectID, req.UserID, req.Email)
	if err != nil {
		log.Printf("Failed to find or create Stripe customer for user %s: %v", req.UserID, err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "api_error", "CUSTOMER_ERROR", "Failed to create or find customer", err.Error(), "user_id", "", "")
		return
	}

	// Create cart checkout session
	checkoutSession, err := createCartStripeSession(req, stripeCustomerID)
	if err != nil {
		log.Printf("Failed to create Stripe cart session for user %s: %v", req.UserID, err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "api_error", "STRIPE_ERROR", "Failed to create cart session", err.Error(), "", "", "")
		return
	}

	log.Printf("Created Stripe cart session: %s for user: %s", checkoutSession.ID, req.UserID)

	// Return response
	writeCartCheckoutResponse(w, checkoutSession, len(req.Items))
}
