package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"styx/internal/database"
	"github.com/stripe/stripe-go/v72"
	checkoutsession "github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/customer"
	billingportalsession "github.com/stripe/stripe-go/v72/billingportal/session"
	"github.com/stripe/stripe-go/v72/sub"
)

// HTTPServer provides HTTP REST API for billing operations
type HTTPServer struct {
	db           *database.Repository
	stripeSecret string
}

// NewHTTPServer creates a new HTTP server instance
func NewHTTPServer(db *database.Repository, stripeSecret string) *HTTPServer {
	stripe.Key = stripeSecret

	return &HTTPServer{
		db:           db,
		stripeSecret: stripeSecret,
	}
}

// CreateSubscriptionCheckout handles POST /api/v1/checkout
func (s *HTTPServer) CreateSubscriptionCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID     string `json:"user_id"`
		Email      string `json:"email"`
		ProductID  string `json:"product_id"`
		PriceID    string `json:"price_id"`
		SuccessURL string `json:"success_url"`
		CancelURL  string `json:"cancel_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.UserID == "" || req.Email == "" || req.ProductID == "" || 
	   req.PriceID == "" || req.SuccessURL == "" || req.CancelURL == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	log.Printf("CreateSubscriptionCheckout called for user: %s, product: %s", req.UserID, req.ProductID)

	// Find or create Stripe customer
	stripeCustomerID, err := s.findOrCreateStripeCustomer(r.Context(), req.UserID, req.Email)
	if err != nil {
		log.Printf("Failed to find or create Stripe customer for user %s: %v", req.UserID, err)
		http.Error(w, "Failed to create or find customer", http.StatusInternalServerError)
		return
	}

	// Create real Stripe Checkout Session
	checkoutParams := &stripe.CheckoutSessionParams{
		Customer:  stripe.String(stripeCustomerID),
		Mode:      stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL:            stripe.String(req.SuccessURL),
		CancelURL:             stripe.String(req.CancelURL),
		ClientReferenceID:     stripe.String(req.UserID),
		AllowPromotionCodes:   stripe.Bool(true),
		BillingAddressCollection: stripe.String(string(stripe.CheckoutSessionBillingAddressCollectionRequired)),
	}

	// Add metadata
	checkoutParams.AddMetadata("user_id", req.UserID)
	checkoutParams.AddMetadata("product_id", req.ProductID)

	checkoutSession, err := checkoutsession.New(checkoutParams)
	if err != nil {
		log.Printf("Failed to create Stripe checkout session for user %s: %v", req.UserID, err)
		http.Error(w, "Failed to create checkout session", http.StatusInternalServerError)
		return
	}

	log.Printf("Created Stripe checkout session: %s for user: %s", checkoutSession.ID, req.UserID)

	// Return response
	response := struct {
		CheckoutSessionID string `json:"checkout_session_id"`
		CheckoutURL       string `json:"checkout_url"`
	}{
		CheckoutSessionID: checkoutSession.ID,
		CheckoutURL:       checkoutSession.URL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSubscriptionStatus handles GET /api/v1/subscriptions/{user_id}/{product_id}
func (s *HTTPServer) GetSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user_id and product_id from URL path
	// Expected format: /api/v1/subscriptions/{user_id}/{product_id}
	path := r.URL.Path
	
	// Remove the base path
	expectedPrefix := "/api/v1/subscriptions/"
	if !strings.HasPrefix(path, expectedPrefix) {
		http.Error(w, "Invalid URL format. Expected: /api/v1/subscriptions/{user_id}/{product_id}", http.StatusBadRequest)
		return
	}

	subPath := path[len(expectedPrefix):]
	
	// Split by "/" to get user_id and product_id
	segments := strings.Split(subPath, "/")
	if len(segments) != 2 || segments[0] == "" || segments[1] == "" {
		http.Error(w, "Invalid URL format. Expected: /api/v1/subscriptions/{user_id}/{product_id}", http.StatusBadRequest)
		return
	}

	userID := segments[0]
	productID := segments[1]

	log.Printf("GetSubscriptionStatus called for user: %s, product: %s", userID, productID)

	// Get subscription status from database
	stripeSubID, customerID, currentPeriodEnd, exists, err := s.db.GetSubscriptionStatus(r.Context(), userID, productID)
	if err != nil {
		log.Printf("Failed to get subscription status from database: %v", err)
		http.Error(w, "Failed to get subscription status", http.StatusInternalServerError)
		return
	}

	if !exists {
		response := struct {
			Exists bool `json:"exists"`
		}{
			Exists: false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get current status from Stripe
	stripeSubscription, err := sub.Get(stripeSubID, nil)
	if err != nil {
		log.Printf("Failed to get Stripe subscription %s: %v", stripeSubID, err)
		// Return database status if Stripe call fails
		response := struct {
			SubscriptionID   string    `json:"subscription_id"`
			Status           string    `json:"status"`
			CustomerID       string    `json:"customer_id"`
			CurrentPeriodEnd time.Time `json:"current_period_end"`
			Exists           bool      `json:"exists"`
		}{
			SubscriptionID:   stripeSubID,
			Status:           "unknown",
			CustomerID:       customerID,
			CurrentPeriodEnd: currentPeriodEnd,
			Exists:           true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := struct {
		SubscriptionID   string    `json:"subscription_id"`
		Status           string    `json:"status"`
		CustomerID       string    `json:"customer_id"`
		CurrentPeriodEnd time.Time `json:"current_period_end"`
		Exists           bool      `json:"exists"`
	}{
		SubscriptionID:   stripeSubID,
		Status:           string(stripeSubscription.Status),
		CustomerID:       stripeSubscription.Customer.ID,
		CurrentPeriodEnd: time.Unix(stripeSubscription.CurrentPeriodEnd, 0),
		Exists:           true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
	json.NewEncoder(w).Encode(response)
}

// HealthCheck handles GET /health
func (s *HTTPServer) HealthCheck(w http.ResponseWriter, r *http.Request) {
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
	json.NewEncoder(w).Encode(response)
}

// findOrCreateStripeCustomer finds an existing Stripe customer or creates a new one
func (s *HTTPServer) findOrCreateStripeCustomer(ctx context.Context, userID, email string) (string, error) {
	// Check database for existing customer
	existingCustomer, err := s.db.GetCustomerByUserID(ctx, userID)
	if err == nil && existingCustomer != nil && existingCustomer.StripeCustomerID != "" {
		log.Printf("Found existing Stripe customer for user %s: %s", userID, existingCustomer.StripeCustomerID)
		return existingCustomer.StripeCustomerID, nil
	}

	// Create new Stripe customer
	customerParams := &stripe.CustomerParams{
		Email: stripe.String(email),
	}

	// Add metadata
	customerParams.AddMetadata("user_id", userID)

	stripeCustomer, err := customer.New(customerParams)
	if err != nil {
		log.Printf("Failed to create Stripe customer: %v", err)
		return "", fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	// Update database with Stripe customer ID
	err = s.db.UpdateCustomerStripeID(ctx, userID, stripeCustomer.ID)
	if err != nil {
		log.Printf("Failed to update customer Stripe ID in database: %v", err)
		return "", fmt.Errorf("failed to update customer record: %w", err)
	}

	log.Printf("Created new Stripe customer: %s for user: %s", stripeCustomer.ID, userID)
	return stripeCustomer.ID, nil
}