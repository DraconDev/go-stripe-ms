package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v72"
	billingportalsession "github.com/stripe/stripe-go/v72/billingportal/session"
)

// HTTPServer provides HTTP REST API for billing operations
type HTTPServer struct {
	db           interface{}
	stripeSecret string
}

// NewHTTPServer creates a new HTTP server instance
func NewHTTPServer(db interface{}, stripeSecret string) *HTTPServer {
	stripe.Key = stripeSecret

	return &HTTPServer{
		db:           db,
		stripeSecret: stripeSecret,
	}
}

// CreateCustomerPortal handles POST /api/v1/portal
func (s *HTTPServer) CreateCustomerPortal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID    string `json:"user_id"`
		ReturnURL string `json:"return_url"`
	}

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

	log.Printf("CreateCustomerPortal called for user: %s", req.UserID)

	// Get user's Stripe customer ID from database
	db := s.db.(interface{ GetCustomerByUserID(context interface{}, userID string) (interface{}, error) })
	customer, err := db.GetCustomerByUserID(r.Context(), req.UserID)
	if err != nil {
		log.Printf("Failed to get customer for user %s: %v", req.UserID, err)
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	customerStruct := customer.(struct {
		StripeCustomerID string
	})

	if customerStruct.StripeCustomerID == "" {
		http.Error(w, "Customer has no Stripe customer ID", http.StatusBadRequest)
		return
	}

	// Create real Stripe Billing Portal session
	portalParams := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerStruct.StripeCustomerID),
		ReturnURL: stripe.String(req.ReturnURL),
	}

	portalSession, err := billingportalsession.New(portalParams)
	if err != nil {
		log.Printf("Failed to create Stripe portal session for customer %s: %v", customerStruct.StripeCustomerID, err)
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
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
		return
	}
}