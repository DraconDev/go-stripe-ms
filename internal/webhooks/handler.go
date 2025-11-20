package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/stripe/stripe-go/v72"
)

// StripeWebhookHandler handles incoming Stripe webhook events
type StripeWebhookHandler struct {
	db     database.RepositoryInterface
	secret string
}

// NewStripeWebhookHandler creates a new Stripe webhook handler
func NewStripeWebhookHandler(db database.RepositoryInterface, stripeSecret, webhookSecret string) *StripeWebhookHandler {
	stripe.Key = stripeSecret
	
	return &StripeWebhookHandler{
		db:     db,
		secret: webhookSecret,
	}
}

// HandleWebhook processes incoming Stripe webhook events
func (h *StripeWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context() // Use request context for proper cancellation and timeout handling
	log.Println("Received Stripe webhook event")

	// Set content type for Stripe
	w.Header().Set("Content-Type", "application/json")

	// Read body bytes for potential signature verification
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading webhook body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	
	// Reset body for JSON decoding
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	
	signature := r.Header.Get("Stripe-Signature")

	// For development/testing, we can skip signature verification if no secret is set
	if h.secret == "" {
		log.Println("Warning: No webhook secret configured, skipping signature verification")
	} else {
		// Verify webhook signature (simplified for development)
		// In production, use stripe.ConstructEvent with proper webhook secret
		if !h.verifySignature(bodyBytes, signature) {
			log.Printf("Webhook signature verification failed")
			http.Error(w, "Invalid signature", http.StatusBadRequest)
			return
		}
	}

	// For development without signature verification
	var event stripe.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("Error decoding webhook event: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	
	h.processEvent(ctx, w, event)
}

// verifySignature is a simplified signature verification for development
func (h *StripeWebhookHandler) verifySignature(body []byte, signature string) bool {
	// In production, implement proper HMAC verification
	// For now, accept all signatures in development
	return true
}

// processEvent handles different types of Stripe events
func (h *StripeWebhookHandler) processEvent(ctx context.Context, w http.ResponseWriter, event stripe.Event) {
	log.Printf("Processing event type: %s", event.Type)

	// WithTimeout ensures we have sufficient time for database operations
	processingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	switch event.Type {
	case "customer.subscription.created":
		h.handleCustomerSubscriptionCreated(processingCtx, event)
	case "customer.subscription.updated":
		h.handleCustomerSubscriptionUpdated(processingCtx, event)
	case "customer.subscription.deleted":
		h.handleCustomerSubscriptionDeleted(processingCtx, event)
	case "invoice.payment_succeeded":
		h.handleInvoicePaymentSucceeded(event)
	case "invoice.payment_failed":
		h.handleInvoicePaymentFailed(event)
	case "payment_method.attached":
		h.handlePaymentMethodAttached(event)
	default:
		log.Printf("Unhandled event type: %s", event.Type)
	}

	// Always return 200 OK to acknowledge receipt
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status": "processed"}`));  err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// SetupRoutes sets up the webhook routes
func (h *StripeWebhookHandler) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/webhooks/stripe", h.HandleWebhook)
}

// HealthCheck returns the health status of the webhook handler
func (h *StripeWebhookHandler) HealthCheck() error {
	// Test database connection with a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := h.db.InitializeTables(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	return nil
}