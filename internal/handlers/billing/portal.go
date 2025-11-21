package billing

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/utils"
	"github.com/DraconDev/go-stripe-ms/internal/middleware"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/billingportal/session"
)

// HandleCustomerPortal handles POST /api/v1/portal
func HandleCustomerPortal(db database.RepositoryInterface, stripeSecret string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "METHOD_NOT_ALLOWED", "Method not allowed", "Only POST method is allowed", "", "", "")
		return
	}

	var req struct {
		UserID    string `json:"user_id"`
		ReturnURL string `json:"return_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding portal request: %v", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, "invalid_request", "INVALID_BODY", "Invalid request body", "Failed to decode JSON body", "", "", "")
		return
	}

	if req.UserID == "" || req.ReturnURL == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "invalid_request", "MISSING_FIELDS", "Missing user_id or return_url", "Both user_id and return_url are required", "", "", "")
		return
	}

	projectID, ok := middleware.GetProjectID(r.Context())
	if !ok {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "authentication_error", "UNAUTHORIZED", "Unauthorized", "Missing or invalid authentication", "", "", "")
		return
	}

	log.Printf("HandleCustomerPortal called for user: %s", req.UserID)

	// Get customer from database
	customer, err := db.GetCustomerByUserID(r.Context(), projectID, req.UserID)
	if err != nil {
		log.Printf("Failed to get customer for user %s: %v", req.UserID, err)
		utils.WriteErrorResponse(w, http.StatusNotFound, "not_found", "CUSTOMER_NOT_FOUND", "Customer not found", "The specified customer could not be found", "user_id", "", "")
		return
	}

	if customer.StripeCustomerID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "invalid_request", "NO_STRIPE_CUSTOMER", "Customer has no Stripe customer ID", "The customer does not have a linked Stripe account", "user_id", "", "")
		return
	}

	// Create real Stripe Billing Portal session
	portalParams := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customer.StripeCustomerID),
		ReturnURL: stripe.String(req.ReturnURL),
	}

	portalSession, err := session.New(portalParams)
	if err != nil {
		log.Printf("Failed to create portal session: %v", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "api_error", "STRIPE_ERROR", "Failed to create portal session", err.Error(), "", "", "")
		return
	}

	log.Printf("Created Stripe portal session: %s for user: %s", portalSession.ID, req.UserID)

	response := struct {
		PortalSessionID string `json:"portal_session_id"`
		PortalURL       string `json:"portal_url"`
	}{
		PortalSessionID: portalSession.ID,
		PortalURL:       portalSession.URL,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response for customer portal: %v", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
		return
	}
}
