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

	"styx/internal/database"
	"github.com/stripe/stripe-go/v72"
)

// StripeWebhookHandler handles incoming Stripe webhook events
type StripeWebhookHandler struct {
	db     *database.Repository
	secret string
}

// NewStripeWebhookHandler creates a new Stripe webhook handler
func NewStripeWebhookHandler(db *database.Repository, stripeSecret, webhookSecret string) *StripeWebhookHandler {
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
	w.Write([]byte(`{"status": "processed"}`))
}

// handleCustomerSubscriptionCreated processes subscription creation events
func (h *StripeWebhookHandler) handleCustomerSubscriptionCreated(ctx context.Context, event stripe.Event) {
	var subscription struct {
		ID               string                 `json:"id"`
		Customer         struct{ ID string }   `json:"customer"`
		Status           string                 `json:"status"`
		Items            struct {
			Data []struct {
				Price struct {
					ID      string `json:"id"`
					Product string `json:"product"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
		CurrentPeriodStart int64 `json:"current_period_start"`
		CurrentPeriodEnd   int64 `json:"current_period_end"`
	}
	
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		log.Printf("Error unmarshaling subscription event: %v", err)
		return
	}

	log.Printf("Subscription created: %s for customer: %s", subscription.ID, subscription.Customer.ID)
	
	// Get customer details
	customer, err := h.getCustomerByStripeID(ctx, subscription.Customer.ID)
	if err != nil {
		log.Printf("Customer not found: %s", subscription.Customer.ID)
		return
	}

	// Extract product and price information
	var productID, priceID string
	if len(subscription.Items.Data) > 0 {
		priceID = subscription.Items.Data[0].Price.ID
		productID = subscription.Items.Data[0].Price.Product
	}

	// Create subscription in database
	err = h.db.CreateSubscription(
		ctx,
		subscription.Customer.ID,
		subscription.ID,
		productID,
		priceID,
		customer.UserID,
		subscription.Status,
		time.Unix(subscription.CurrentPeriodStart, 0),
		time.Unix(subscription.CurrentPeriodEnd, 0),
	)
	
	if err != nil {
		log.Printf("Error creating subscription in database: %v", err)
	} else {
		log.Printf("Successfully created subscription in database")
	}
}

// handleCustomerSubscriptionUpdated processes subscription update events
func (h *StripeWebhookHandler) handleCustomerSubscriptionUpdated(ctx context.Context, event stripe.Event) {
	var subscription struct {
		ID               string `json:"id"`
		Status           string `json:"status"`
		CurrentPeriodEnd int64  `json:"current_period_end"`
	}
	
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		log.Printf("Error unmarshaling subscription event: %v", err)
		return
	}

	log.Printf("Subscription updated: %s, status: %s", subscription.ID, subscription.Status)

	// Update subscription status with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	err := h.db.UpdateSubscriptionStatus(
		timeoutCtx,
		subscription.ID,
		subscription.Status,
		time.Unix(subscription.CurrentPeriodEnd, 0),
	)
	
	if err != nil {
		log.Printf("Error updating subscription in database: %v", err)
	} else {
		log.Printf("Successfully updated subscription in database")
	}
}

// handleCustomerSubscriptionDeleted processes subscription deletion events
func (h *StripeWebhookHandler) handleCustomerSubscriptionDeleted(ctx context.Context, event stripe.Event) {
	var subscription struct {
		ID               string `json:"id"`
		CurrentPeriodEnd int64  `json:"current_period_end"`
	}
	
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		log.Printf("Error unmarshaling subscription event: %v", err)
		return
	}

	log.Printf("Subscription deleted: %s", subscription.ID)

	// Update subscription status to canceled with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	err := h.db.UpdateSubscriptionStatus(
		timeoutCtx,
		subscription.ID,
		"canceled",
		time.Unix(subscription.CurrentPeriodEnd, 0),
	)
	
	if err != nil {
		log.Printf("Error updating subscription status to canceled: %v", err)
	} else {
		log.Printf("Successfully marked subscription as canceled")
	}
}

// handleInvoicePaymentSucceeded processes successful payment events
func (h *StripeWebhookHandler) handleInvoicePaymentSucceeded(event stripe.Event) {
	var invoice struct {
		ID         string `json:"id"`
		AmountPaid int64  `json:"amount_paid"`
	}
	
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		log.Printf("Error unmarshaling invoice event: %v", err)
		return
	}

	log.Printf("Payment succeeded for invoice: %s, amount: %d", invoice.ID, invoice.AmountPaid)
	// Additional logic for successful payments can be added here
}

// handleInvoicePaymentFailed processes failed payment events
func (h *StripeWebhookHandler) handleInvoicePaymentFailed(event stripe.Event) {
	var invoice struct {
		ID        string `json:"id"`
		AmountDue int64  `json:"amount_due"`
	}
	
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		log.Printf("Error unmarshaling invoice event: %v", err)
		return
	}

	log.Printf("Payment failed for invoice: %s, amount: %d", invoice.ID, invoice.AmountDue)
	// Additional logic for failed payments can be added here (e.g., notifications)
}

// handlePaymentMethodAttached processes payment method attachment events
func (h *StripeWebhookHandler) handlePaymentMethodAttached(event stripe.Event) {
	var paymentMethod struct {
		ID       string `json:"id"`
		Customer string `json:"customer"`
	}
	
	if err := json.Unmarshal(event.Data.Raw, &paymentMethod); err != nil {
		log.Printf("Error unmarshaling payment method event: %v", err)
		return
	}

	log.Printf("Payment method attached: %s for customer: %s", paymentMethod.ID, paymentMethod.Customer)
	// Additional logic for payment method updates can be added here
}

// getCustomerByStripeID retrieves customer from database by Stripe customer ID
func (h *StripeWebhookHandler) getCustomerByStripeID(ctx context.Context, stripeCustomerID string) (*database.Customer, error) {
	return h.db.GetCustomerByStripeID(ctx, stripeCustomerID)
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
