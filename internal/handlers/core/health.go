package core

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/DraconDev/go-stripe-ms/internal/handlers/utils"
)

// HandleHealth handles GET /health - simple health check
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := struct {
		Status    string    `json:"status"`
		Timestamp time.Time `json:"timestamp"`
		Service   string    `json:"service"`
	}{
		Status:    "healthy",
		Timestamp: time.Now(),
		Service:   "billing-service",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response for health check: %v", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
		return
	}
}

// HandleRoot handles GET / - main health check endpoint with service information
func HandleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := struct {
		Status    string    `json:"status"`
		Timestamp time.Time `json:"timestamp"`
		Service   string    `json:"service"`
		Version   string    `json:"version"`
		Message   string    `json:"message"`
		Endpoints []string  `json:"endpoints"`
	}{
		Status:    "healthy",
		Timestamp: time.Now(),
		Service:   "billing-service",
		Version:   "1.0.0",
		Message:   "Billing microservice is running",
		Endpoints: []string{
			"GET /",
			"GET /health",
			"POST /api/v1/checkout/subscription",
			"POST /api/v1/checkout/item",
			"POST /api/v1/checkout/cart",
			"GET /api/v1/subscriptions/{user_id}/{product_id}",
			"POST /api/v1/portal",
			"POST /webhooks/stripe",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response for root handler: %v", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
		return
	}
}
