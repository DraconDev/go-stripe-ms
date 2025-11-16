package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/stripe/stripe-go/v72"
	billingportalsession "github.com/stripe/stripe-go/v72/billingportal/session"
	"github.com/stripe/stripe-go/v72/sub"
)

// HTTPServer provides HTTP REST API for billing operations
type HTTPServer struct {
	db           database.RepositoryInterface
	stripeSecret string
}

// NewHTTPServer creates a new HTTP server instance
func NewHTTPServer(db database.RepositoryInterface, stripeSecret string) *HTTPServer {
	stripe.Key = stripeSecret

	return &HTTPServer{
		db:           db,
		stripeSecret: stripeSecret,
	}
}

// GetSubscriptionStatus handles GET /api/v1/subscriptions/{user_id}/{product_id}
func (s *HTTPServer) GetSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse user_id and product_id from URL path
	userID := r.URL.Path[len("/api/v1/subscriptions/"):]
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	parts := splitURLPath(userID)
	if len(parts) != 2 {
		http.Error(w, "Invalid URL format. Expected /api/v1/subscriptions/{user_id}/{product_id}", http.StatusBadRequest)
		return
	}

	userID = parts[0]
	productID := parts[1]

	log.Printf("GetSubscriptionStatus called for user: %s, product: %s", userID, productID)

	// Check database for existing subscription using correct method
	stripeSubID, status, currentPeriodEnd, exists, err := s.db.GetSubscriptionStatus(r.Context(), userID, productID)
	if err != nil {
		log.Printf("Failed to get subscription status for user %s, product %s: %v", userID, productID, err)
		writeErrorResponse(w, http.StatusInternalServerError, "database_error", "DATABASE_QUERY_FAILED",
			"Failed to query subscription", "An error occurred while checking subscription status.", "", "", "")
		return
	}

	// Return response based on subscription status
	if !exists || stripeSubID == "" {
		response := struct {
			Exists     bool      `json:"exists"`
			Status     string    `json:"status"`
			Message    string    `json:"message"`
			UserID     string    `json:"user_id"`
			ProductID  string    `json:"product_id"`
		}{
			Exists:     false,
			Status:     "not_found",
			Message:    "No active subscription found",
			UserID:     userID,
			ProductID:  productID,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding response for subscription status: %v", err)
			writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
				"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
			return
		}
		return
	}

	// Get subscription from Stripe for latest status
	stripeSub, err := sub.Get(stripeSubID, nil)
	if err != nil {
		log.Printf("Failed to get Stripe subscription %s: %v", stripeSubID, err)
		// Return database info if Stripe call fails
		response := struct {
			SubscriptionID   string    `json:"subscription_id"`
			Status           string    `json:"status"`
			CustomerID       string    `json:"customer_id"`
			CurrentPeriodEnd time.Time `json:"current_period_end"`
			Exists           bool      `json:"exists"`
		}{
			SubscriptionID:   stripeSubID,
			Status:           status,
			CustomerID:       "", // We don't have customerID from the current method
			CurrentPeriodEnd: currentPeriodEnd,
			Exists:           true,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding response for subscription status: %v", err)
			writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
				"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
			return
		}
		return
	}

	// Return active subscription details from Stripe
	response := struct {
		SubscriptionID   string    `json:"subscription_id"`
		Status           string    `json:"status"`
		CustomerID       string    `json:"customer_id"`
		CurrentPeriodEnd time.Time `json:"current_period_end"`
		Exists           bool      `json:"exists"`
	}{
		SubscriptionID:   stripeSub.ID,
		Status:           string(stripeSub.Status),
		CustomerID:       stripeSub.Customer.ID,
		CurrentPeriodEnd: time.Unix(stripeSub.CurrentPeriodEnd, 0),
		Exists:           true,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response for subscription status: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
		return
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

	// Get user's Stripe customer ID
	customer, err := s.db.GetCustomerByUserID(r.Context(), req.UserID)
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
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
		return
	}
}

// splitURLPath splits a URL path by "/" and returns the parts
func splitURLPath(path string) []string {
	if path == "" {
		return []string{}
	}
	
	// Remove leading slash if present
	if path[0] == '/' {
		path = path[1:]
	}
	
	// Split by "/"
	parts := make([]string, 0)
	current := ""
	for _, char := range path {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	
	// Add the last part if it exists
	if current != "" {
		parts = append(parts, current)
	}
	
	return parts
}