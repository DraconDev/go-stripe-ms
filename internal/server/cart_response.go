package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v72"
)

// writeCartCheckoutResponse writes the response for cart checkout
func writeCartCheckoutResponse(w http.ResponseWriter, checkoutSession *stripe.CheckoutSession, itemCount int) {
	response := struct {
		CheckoutSessionID string `json:"checkout_session_id"`
		CheckoutURL       string `json:"checkout_url"`
		ItemCount         int    `json:"item_count"`
	}{
		CheckoutSessionID: checkoutSession.ID,
		CheckoutURL:       checkoutSession.URL,
		ItemCount:         itemCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response for cart checkout: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "ENCODING_FAILED",
			"Failed to encode response", "An unexpected error occurred while preparing the response.", "", "", "")
	}
}