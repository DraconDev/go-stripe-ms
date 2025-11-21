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

// HandleRoot handles GET / - redirects to the documentation page
func HandleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Redirect to /docs which contains the full API documentation
	http.Redirect(w, r, "/docs", http.StatusFound)
}
