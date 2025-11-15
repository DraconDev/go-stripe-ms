package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DraconDev/go-stripe-ms/internal/database"

	"github.com/stripe/stripe-go/v72"
	billingportalsession "github.com/stripe/stripe-go/v72/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/price"
	"github.com/stripe/stripe-go/v72/product"
	"github.com/stripe/stripe-go/v72/sub"
)

// Simple rate limiter for basic protection
type RateLimiter struct {
	mu           sync.RWMutex
	requests     map[string][]time.Time
	limit        int
	window       time.Duration
}

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    requestsPerMinute,
		window:   time.Minute,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	requests := rl.requests[ip]

	// Remove old requests outside the window
	var validRequests []time.Time
	for _, reqTime := range requests {
		if now.Sub(reqTime) <= rl.window {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check limit
	if len(validRequests) >= rl.limit {
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[ip] = validRequests
	return true
}

// Enhanced input validation
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

func validateEmail(email string) error {
	if email == "" {
		return &ValidationError{Field: "email", Message: "email is required"}
	}
	
	if _, err := mail.ParseAddress(email); err != nil {
		return &ValidationError{Field: "email", Message: "invalid email format"}
	}
	
	return nil
}

		func validateURL(url string) error {
	if url == "" {
		return &ValidationError{Field: "url", Message: "url is required"}
	}
	
	// Basic URL validation
	_, err := net.LookupHost(strings.TrimPrefix(url, "https://"))
			if err != nil {
		return &ValidationError{Field: "url", Message: "invalid URL"}
	}
	
			return nil
}

func validateRequiredString(value, fieldName string) error {
	if value == "" {
		return &ValidationError{Field: fieldName, Message: fmt.Sprintf("%s is required", fieldName)}
	}
	return nil
}

func validateUserID(userID string) error {
	if err := validateRequiredString(userID, "user_id"); err != nil {
		return err
	}
	
	// Basic format validation
	userIDRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !userIDRegex.MatchString(userID) {
		return &ValidationError{Field: "user_id", Message: "user_id must contain only alphanumeric characters, hyphens, and underscores"}
	}
	
	if len(userID) > 100 {
		return &ValidationError{Field: "user_id", Message: "user_id must be less than 100 characters"}
	}
	
	return nil
}

// Enhanced error response
type ErrorResponse struct {
	Error   ErrorDetail `json:"error"`
	Meta    ErrorMeta   `json:"meta,omitempty"`
}

type ErrorDetail struct {
	Type        string `json:"type"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description,omitempty"`
	Field       string `json:"field,omitempty"`
}

type ErrorMeta struct {
	RequestID   string    `json:"request_id"`
	Timestamp   time.Time `json:"timestamp"`
	Environment string    `json:"environment,omitempty"`
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, errorType, code, message, description, field, requestID, environment string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	// Add security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	
	response := ErrorResponse{
		Error: ErrorDetail{
			Type:        errorType,
			Code:        code,
			Message:     message,
			Description: description,
			Field:       field,
		},
		Meta: ErrorMeta{
			RequestID:   requestID,
			Timestamp:   time.Now(),
			Environment: environment,
		},
	}
	
	json.NewEncoder(w).Encode(response)
}

func writeValidationError(w http.ResponseWriter, field, message, requestID, environment string) {
	writeErrorResponse(w, http.StatusBadRequest, "validation_error", "VALIDATION_FAILED",
		"Request validation failed", message, field, requestID, environment)
}

func writeAuthenticationError(w http.ResponseWriter, message, requestID, environment string) {
	writeErrorResponse(w, http.StatusUnauthorized, "authentication_error", "AUTHENTICATION_REQUIRED",
		message, "Valid API key required", "", requestID, environment)
}

func writeNotFoundError(w http.ResponseWriter, message, requestID, environment string) {
	writeErrorResponse(w, http.StatusNotFound, "not_found", "RESOURCE_NOT_FOUND",
		message, "The requested resource was not found", "", requestID, environment)
}

func writeRateLimitError(w http.ResponseWriter, requestID, environment string) {
	w.Header().Set("Retry-After", "60")
	writeErrorResponse(w, http.StatusTooManyRequests, "rate_limit_error", "RATE_LIMIT_EXCEEDED",
		"Too many requests", "Please try again later", "", requestID, environment)
}

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
		Customer: stripe.String(stripeCustomerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL:               stripe.String(req.SuccessURL),
		CancelURL:                stripe.String(req.CancelURL),
		ClientReferenceID:        stripe.String(req.UserID),
		AllowPromotionCodes:      stripe.Bool(true),
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

// CreateProduct handles POST /api/v1/products to create new Stripe products
func (s *HTTPServer) CreateProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Active      bool   `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding product request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.Name == "" {
		http.Error(w, "Product name is required", http.StatusBadRequest)
		return
	}

	// Create product in Stripe
	productParams := &stripe.ProductParams{
		Name:        stripe.String(req.Name),
		Active:      stripe.Bool(req.Active),
		Description: stripe.String(req.Description),
	}

	product, err := product.New(productParams)
	if err != nil {
		log.Printf("Failed to create Stripe product: %v", err)
		http.Error(w, "Failed to create product", http.StatusInternalServerError)
		return
	}

	response := struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Active      bool   `json:"active"`
		Created     int64  `json:"created"`
	}{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Active:      product.Active,
		Created:     product.Created,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// CreatePrice handles POST /api/v1/prices to create new Stripe prices
func (s *HTTPServer) CreatePrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProductID   string `json:"product_id"`
		Currency    string `json:"currency"`
		UnitAmount  int64  `json:"unit_amount"`
		Recurring   struct {
			Interval string `json:"interval"`
			Count    int64  `json:"count"`
		} `json:"recurring"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding price request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.ProductID == "" || req.Currency == "" || req.UnitAmount <= 0 {
		http.Error(w, "Product ID, currency, and positive unit amount are required", http.StatusBadRequest)
		return
	}

	// Create price in Stripe
	priceParams := &stripe.PriceParams{
		Product:     stripe.String(req.ProductID),
		Currency:    stripe.String(strings.ToLower(req.Currency)),
		UnitAmount:  stripe.Int64(req.UnitAmount),
	}

	if req.Recurring.Interval != "" {
		priceParams.Recurring = &stripe.PriceRecurringParams{
			Interval: stripe.String(req.Recurring.Interval),
		}
	}

	price, err := price.New(priceParams)
	if err != nil {
		log.Printf("Failed to create Stripe price: %v", err)
		http.Error(w, "Failed to create price", http.StatusInternalServerError)
		return
	}

	response := struct {
		ID             string `json:"id"`
		ProductID      string `json:"product_id"`
		Currency       string `json:"currency"`
		UnitAmount     int64  `json:"unit_amount"`
		Recurring      *struct {
			Interval string `json:"interval"`
		} `json:"recurring,omitempty"`
		Created int64 `json:"created"`
	}{
		ID:         price.ID,
		ProductID:  string(price.Product.ID),
		Currency:   string(price.Currency),
		UnitAmount: price.UnitAmount,
		Created:    price.Created,
	}

	if price.Recurring != nil {
		response.Recurring = &struct {
			Interval string `json:"interval"`
		}{
			Interval: string(price.Recurring.Interval),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetProducts handles GET /api/v1/products to list Stripe products
func (s *HTTPServer) GetProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	active := ""
	if activeStr := r.URL.Query().Get("active"); activeStr != "" {
		if activeStr == "true" || activeStr == "false" {
			active = activeStr
		}
	}

	// List products from Stripe
	params := &stripe.ProductListParams{}
	limitValue := int64(limit)
	params.Limit = &limitValue

	if active != "" {
		activeBool := active == "true"
		params.Active = stripe.Bool(activeBool)
	}

	iterator := product.List(params)
	productList := make([]map[string]interface{}, 0)
	
	for iterator.Next() {
		product := iterator.Product()
		productData := map[string]interface{}{
			"id":          product.ID,
			"name":        product.Name,
			"description": product.Description,
			"active":      product.Active,
			"created":     product.Created,
			"updated":     product.Updated,
		}
		productList = append(productList, productData)
	}

	if iterator.Err() != nil {
		log.Printf("Failed to list Stripe products: %v", iterator.Err())
		http.Error(w, "Failed to list products", http.StatusInternalServerError)
		return
	}

	response := struct {
		Data    []map[string]interface{} `json:"data"`
		Object  string                   `json:"object"`
		HasMore bool                     `json:"has_more"`
	}{
		Data:    productList,
		Object:  "list",
		HasMore: false, // Stripe iterator doesn't expose has_more directly
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
