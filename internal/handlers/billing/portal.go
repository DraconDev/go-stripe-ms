package billing

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/utils"
	"github.com/stripe/stripe-go/v72"
	billingportalsession "github.com/stripe/stripe-go/v72/billingportal/session"
)

// CustomerPortalRequest represents the request for creating a customer portal session
type CustomerPortalRequest struct {
	UserID    string `json:"user_id"`
	ReturnURL string `json:"return_url"`
}

// HandleCustomerPortal handles POST /api/v1/portal
func HandleCustomerPortal(db database.RepositoryInterface, stripeSecret string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CustomerPortalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.UserID == "" || req.ReturnURL == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	log.Printf("HandleCustomerPortal called for user: %s", req.UserID)

	// Get user's Stripe customer ID
	customer, err := db.GetCustomerByUserID(r.Context(), req.UserID)
	if err != nil {
		log.Printf("Failed to get customer for user %s: %v", req.UserID, err)
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	if customer.StripeCustomerID == "" {
		http.Error(w, "Customer has no Stripe customer ID", http.StatusBadRequest)
		return
	}

	// Create real Stripe Billing Portal session
	portalParams := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customer.StripeCustomerID),
		ReturnURL: stripe.String(req.ReturnURL),
	}

	portalSession, err := billingportalsession.New(portalParams)
	if err != nil {
		log.Printf("Failed to create Stripe portal session for customer %s: %v", customer.StripeCustomerID, err)
		http.Error(w, "Failed to create portal session", http.StatusInternalServerError)
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
