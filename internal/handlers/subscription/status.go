package subscription

import (
	"encoding/json"
	"log"
	"net/http"
	"strings" // Added for URL parsing
	"time"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/utils" // Kept for helper functions

	// Added, though not directly used in this snippet, it's in the provided import block
	"github.com/DraconDev/go-stripe-ms/internal/middleware"
	"github.com/jackc/pgx/v5"         // Added for pgx.ErrNoRows check
	"github.com/stripe/stripe-go/v72" // Kept for helper functions
	// Kept for helper functions
)

// HandleSubscriptionStatus handles GET /api/v1/subscriptions/{user_id}/{product_id}
func HandleSubscriptionStatus(db database.RepositoryInterface, stripeSecret string, w http.ResponseWriter, r *http.Request) {
	// Parse URL path to extract user_id and product_id
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 { // e.g., "api/v1/subscriptions/user_id/product_id" -> 5 parts
		utils.WriteErrorResponse(w, http.StatusBadRequest, "invalid_request", "INVALID_URL", "Invalid URL format", "URL must be in format /api/v1/subscriptions/{user_id}/{product_id}", "", "", "")
		return
	}

	userID := pathParts[len(pathParts)-2]
	productID := pathParts[len(pathParts)-1]

	log.Printf("HandleSubscriptionStatus called for user: %s, product: %s", userID, productID)

	// Get projectID from context
	projectID, ok := middleware.GetProjectID(r.Context())
	if !ok {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "authentication_error", "UNAUTHORIZED", "Unauthorized", "Missing or invalid authentication", "", "", "")
		return
	}

	// Get subscription status from database
	// The signature of GetSubscriptionStatus in the database interface might need to be updated
	// to return customerID and periodEnd instead of status and currentPeriodEnd.
	// For now, assuming it returns stripeSubID, customerID, periodEnd, exists, err
	stripeSubID, customerID, periodEnd, exists, err := db.GetSubscriptionStatus(r.Context(), projectID, userID, productID)
	if err != nil {
		// Check if it's a "no rows" error - this is normal when subscription doesn't exist
		if err == pgx.ErrNoRows {
			// Return exists: false
			response := map[string]interface{}{
				"exists": false,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		// For other errors, return 500
		log.Printf("Failed to get subscription status for user %s, product %s: %v", userID, productID, err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "api_error", "DATABASE_ERROR", "Internal server error", "Failed to retrieve subscription status", "", "", "")
		return
	}

	// If no subscription found (exists is false), return appropriate response
	if !exists {
		response := map[string]interface{}{
			"exists": false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return subscription details
	response := map[string]interface{}{
		"exists":                 true,
		"stripe_subscription_id": stripeSubID,
		"customer_id":            customerID,
		"period_end":             periodEnd,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding subscription status response: %v", err)
	}
}

// writeSubscriptionNotFoundResponse writes a response when no subscription is found
func writeSubscriptionNotFoundResponse(w http.ResponseWriter, userID, productID string) {
	response := struct {
		Exists    bool   `json:"exists"`
		Status    string `json:"status"`
		Message   string `json:"message"`
		UserID    string `json:"user_id"`
		ProductID string `json:"product_id"`
	}{
		Exists:    false,
		Status:    "not_found",
		Message:   "No active subscription found",
		UserID:    userID,
		ProductID: productID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response for subscription status: %v", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
	}
}

// writeSubscriptionDatabaseResponse writes a response using database info when Stripe call fails
func writeSubscriptionDatabaseResponse(w http.ResponseWriter, stripeSubID, status string, currentPeriodEnd time.Time) {
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
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
	}
}

// writeSubscriptionStripeResponse writes a response using Stripe subscription data
func writeSubscriptionStripeResponse(w http.ResponseWriter, stripeSub *stripe.Subscription) {
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
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
	}
}
