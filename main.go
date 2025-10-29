package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
)

type PaymentRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type PaymentResponse struct {
	ClientSecret string `json:"client_secret"`
}

func main() {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	http.HandleFunc("/create-payment-intent", createPaymentIntent)
	http.HandleFunc("/create-test-payment-intent", createTestPaymentIntent)
	http.HandleFunc("/health", healthCheck)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func createPaymentIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(req.Amount),
		Currency: stripe.String(req.Currency),
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		http.Error(w, "Failed to create payment intent", http.StatusInternalServerError)
		log.Printf("Error creating payment intent: %v", err)
		return
	}

	response := PaymentResponse{
		ClientSecret: pi.ClientSecret,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func createTestPaymentIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Use test mode - this will only work with test keys
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(req.Amount),
		Currency: stripe.String(req.Currency),
		// Additional test-specific parameters can be added here
		Metadata: map[string]string{
			"test_mode": "true",
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		http.Error(w, "Failed to create test payment intent", http.StatusInternalServerError)
		log.Printf("Error creating test payment intent: %v", err)
		return
	}

	response := PaymentResponse{
		ClientSecret: pi.ClientSecret,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
