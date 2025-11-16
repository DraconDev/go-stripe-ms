package subscription

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/sub"
)

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
		s.writeSubscriptionNotFoundResponse(w, userID, productID)
		return
	}

	// Get subscription from Stripe for latest status
	stripeSub, err := sub.Get(stripeSubID, nil)
	if err != nil {
		log.Printf("Failed to get Stripe subscription %s: %v", stripeSubID, err)
		// Return database info if Stripe call fails
		s.writeSubscriptionDatabaseResponse(w, stripeSubID, status, currentPeriodEnd)
		return
	}

	// Return active subscription details from Stripe
	s.writeSubscriptionStripeResponse(w, stripeSub)
}

// writeSubscriptionNotFoundResponse writes a response when no subscription is found
func (s *HTTPServer) writeSubscriptionNotFoundResponse(w http.ResponseWriter, userID, productID string) {
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
	}
}

// writeSubscriptionDatabaseResponse writes a response using database info when Stripe call fails
func (s *HTTPServer) writeSubscriptionDatabaseResponse(w http.ResponseWriter, stripeSubID, status string, currentPeriodEnd time.Time) {
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
	}
}

// writeSubscriptionStripeResponse writes a response using Stripe subscription data
func (s *HTTPServer) writeSubscriptionStripeResponse(w http.ResponseWriter, stripeSub *stripe.Subscription) {
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
	}
}
