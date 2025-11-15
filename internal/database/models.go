package database

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Customer represents a user customer record
type Customer struct {
	ID               uuid.UUID `json:"id"`
	UserID           string    `json:"user_id"`
	Email            string    `json:"email"`
	StripeCustomerID string    `json:"stripe_customer_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Subscription represents a subscription record
type Subscription struct {
	ID                   uuid.UUID `json:"id"`
	CustomerID           uuid.UUID `json:"customer_id"`
	UserID               string    `json:"user_id"`
	ProductID            string    `json:"product_id"`
	PriceID              string    `json:"price_id"`
	StripeSubscriptionID string    `json:"stripe_subscription_id"`
	Status               string    `json:"status"`
	CurrentPeriodStart   time.Time `json:"current_period_start"`
	CurrentPeriodEnd     time.Time `json:"current_period_end"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// ScanCustomer scans a database row into a Customer struct
func ScanCustomer(row pgx.Row) (*Customer, error) {
	var customer Customer
	err := row.Scan(
		&customer.ID,
		&customer.UserID,
		&customer.Email,
		&customer.StripeCustomerID,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// ScanSubscription scans a database row into a Subscription struct
func ScanSubscription(row pgx.Row) (*Subscription, error) {
	var sub Subscription
	err := row.Scan(
		&sub.ID,
		&sub.CustomerID,
		&sub.UserID,
		&sub.ProductID,
		&sub.PriceID,
		&sub.StripeSubscriptionID,
		&sub.Status,
		&sub.CurrentPeriodStart,
		&sub.CurrentPeriodEnd,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// ScanSubscriptionStatus scans a database row into subscription status fields
func ScanSubscriptionStatus(row pgx.Row) (string, string, time.Time, bool, error) {
	var stripeSubID, customerID, status sqlString
	var currentPeriodEnd sqlTime
	var exists sqlBool

	err := row.Scan(&stripeSubID, &customerID, &status, &currentPeriodEnd, &exists)
	if err != nil {
		return "", "", time.Time{}, false, err
	}

	return string(stripeSubID), string(customerID), time.Time(currentPeriodEnd), bool(exists), nil
}

// Helper types for database scanning
type sqlString string
type sqlTime time.Time
type sqlBool bool

func (s *sqlString) Scan(src interface{}) error {
	if src == nil {
		*s = sqlString("")
		return nil
	}
	*s = sqlString(src.(string))
	return nil
}

func (t *sqlTime) Scan(src interface{}) error {
	if src == nil {
		*t = sqlTime(time.Time{})
		return nil
	}
	*t = sqlTime(src.(time.Time))
	return nil
}

func (b *sqlBool) Scan(src interface{}) error {
	if src == nil {
		*b = sqlBool(false)
		return nil
	}
	*b = sqlBool(src.(bool))
	return nil
}

func (t sqlTime) Time() time.Time { return time.Time(t) }
// ========================
// MULTI-PROJECT EXTENSIONS (Keep Existing + Add Project Support)
// ========================

import "encoding/json"

// Project represents a project configuration for your unified ecosystem
type Project struct {
	ID              uuid.UUID `json:"id"`
	ProjectID       string    `json:"project_id"` // Internal identifier (e.g., "ecommerce_001")
	Name            string    `json:"name"`
	DisplayName     string    `json:"display_name"` // User-friendly name
	Domain          string    `json:"domain"`
	Environment     string    `json:"environment"` // "development", "production"
	Type            string    `json:"type"` // "ecommerce", "saas", "marketplace", "platform"
	PaymentTypes    []string  `json:"payment_types"` // ["one_time"], ["subscription"], ["both"]
	DefaultCurrency string    `json:"default_currency"`
	Features        []string  `json:"features"` // ["credits", "permissions", "analytics"]
	SuccessURL      string    `json:"success_url"`
	CancelURL       string    `json:"cancel_url"`
	WebhookURL      string    `json:"webhook_url"`
	Status          string    `json:"status"` // "active", "inactive", "suspended"
	APIKey          string    `json:"api_key"` // For project authentication
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Product represents a product (can be shared across projects or project-specific)
type Product struct {
	ID               uuid.UUID `json:"id"`
	ProjectID        string    `json:"project_id"` // Which project owns this product (can be null for shared)
	ProductID        string    `json:"product_id"` // Internal identifier
	Name             string    `json:"name"`
	DisplayName      string    `json:"display_name"` // User-friendly name
	Type             string    `json:"type"` // "one_time", "subscription", "credit", "permission"
	Scope            string    `json:"scope"` // "shared" (cross-project) or "project_specific"
	EligibleProjects []string  `json:"eligible_projects"` // Which projects can use this (empty = all)
	StripeProductID  string    `json:"stripe_product_id"` // Created in your Stripe account
	StripePriceID    string    `json:"stripe_price_id"` // Created in your Stripe account
	Recurring        bool      `json:"recurring"`
	Interval         string    `json:"interval"` // "month", "year" (for subscriptions)
	Currency         string    `json:"currency"`
	UnitAmount       int64     `json:"unit_amount"` // in cents
	Metadata         JSONB     `json:"metadata"` // Additional product data
	Active           bool      `json:"active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ProjectProduct represents which products are available to which projects
type ProjectProduct struct {
	ID           uuid.UUID `json:"id"`
	ProjectID    string    `json:"project_id"`
	ProductID    string    `json:"product_id"`
	Enabled      bool      `json:"enabled"`
	CustomName   *string   `json:"custom_name,omitempty"` // Project can override display name
	CustomPrice  *int64    `json:"custom_price,omitempty"` // Project can override price
	CreatedAt    time.Time `json:"created_at"`
}

// Payment represents a transaction in your unified ecosystem
type Payment struct {
	ID                   uuid.UUID `json:"id"`
	ProjectID            string    `json:"project_id"` // NEW: Track which project this payment belongs to
	CustomerID           uuid.UUID `json:"customer_id"`
	UserID               string    `json:"user_id"`
	ProductID            string    `json:"product_id"`
	PriceID              string    `json:"price_id"`
	StripePaymentIntentID string   `json:"stripe_payment_intent_id"`
	CheckoutSessionID    string    `json:"checkout_session_id"`
	Amount               int64     `json:"amount"`
	Currency             string    `json:"currency"`
	Status               string    `json:"status"` // "pending", "succeeded", "failed", "canceled"
	PaymentType          string    `json:"payment_type"` // "item", "subscription", "cart"
	Metadata             JSONB     `json:"metadata"` // Additional payment data
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Subscription represents a user subscription (can work across multiple projects)
type Subscription struct {
	ID                   uuid.UUID `json:"id"`
	CustomerID           uuid.UUID `json:"customer_id"`
	UserID               string    `json:"user_id"`
	ProductID            string    `json:"product_id"` // Shared product ID
	EligibleProjects     []string  `json:"eligible_projects"` // Projects where this subscription works
	ActiveProjects       []string  `json:"active_projects"` // Currently active in these projects
	StripeSubscriptionID string    `json:"stripe_subscription_id"`
	Status               string    `json:"status"` // "active", "canceled", "past_due", "unpaid"
	CurrentPeriodStart   time.Time `json:"current_period_start"`
	CurrentPeriodEnd     time.Time `json:"current_period_end"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// CheckoutRequest represents a checkout request with project context
type CheckoutRequest struct {
	ProjectID string            `json:"project_id"`
	UserID    string            `json:"user_id"`
	Email     string            `json:"email"`
	SuccessURL string           `json:"success_url"`
	CancelURL string            `json:"cancel_url"`
	Metadata  map[string]string `json:"metadata"`
	Quantity  int64             `json:"quantity,omitempty"`
}

// SubscriptionCheckoutRequest extends CheckoutRequest for subscriptions
type SubscriptionCheckoutRequest struct {
	CheckoutRequest
	ProductID string `json:"product_id"`
}

// ItemCheckoutRequest extends CheckoutRequest for one-time purchases
type ItemCheckoutRequest struct {
	CheckoutRequest
	ProductID string `json:"product_id"`
}

// CartCheckoutRequest extends CheckoutRequest for cart purchases
type CartCheckoutRequest struct {
	CheckoutRequest
	Items []CartItem `json:"items"`
}

type CartItem struct {
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

// CheckoutResponse represents checkout response
type CheckoutResponse struct {
	CheckoutSessionID string `json:"checkout_session_id"`
	CheckoutURL       string `json:"checkout_url"`
	ProjectID         string `json:"project_id"`
	PaymentID         string `json:"payment_id"`
}

// ProjectRegistrationRequest represents project registration
type ProjectRegistrationRequest struct {
	ProjectID      string   `json:"project_id"`
	Name           string   `json:"name"`
	DisplayName    string   `json:"display_name"`
	Domain         string   `json:"domain"`
	Type           string   `json:"type"`
	Environment    string   `json:"environment"`
	SharedProducts []string `json:"shared_products"` // IDs of shared products to enable
}

// ProjectRegistrationResponse represents project registration response
type ProjectRegistrationResponse struct {
	Project     Project  `json:"project"`
	APIKey      string   `json:"api_key"`
	SharedProducts []Product `json:"shared_products"`
	Endpoints   map[string]string `json:"endpoints"`
}

// JSONB is a helper type for JSON data in PostgreSQL
type JSONB map[string]interface{}

// Scan implements sql.Scanner for JSONB
func (j *JSONB) Scan(src interface{}) error {
	if src == nil {
		*j = nil
		return nil
	}
	
	bytes, ok := src.([]byte)
	if !ok {
		return nil
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}
	
	*j = result
	return nil
}

// Value implements driver.Valuer for JSONB
func (j JSONB) Value() (interface{}, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}
func (b sqlBool) Bool() bool      { return bool(b) }
