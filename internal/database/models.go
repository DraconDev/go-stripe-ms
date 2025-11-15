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
// UNIFIED MULTI-PROJECT MODELS
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

// Product represents a product within your unified Stripe account
type Product struct {
	ID               uuid.UUID `json:"id"`
	ProjectID        string    `json:"project_id"`
	ProductID        string    `json:"product_id"` // Internal identifier
	Name             string    `json:"name"`
	DisplayName      string    `json:"display_name"` // User-friendly name
	Type             string    `json:"type"` // "one_time", "subscription", "credit", "permission"
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

// Payment represents a transaction in your unified ecosystem
type Payment struct {
	ID                   uuid.UUID `json:"id"`
	ProjectID            string    `json:"project_id"`
	PaymentID            string    `json:"payment_id"` // Internal payment identifier
	CustomerID           uuid.UUID `json:"customer_id"`
	UserID               string    `json:"user_id"`
	ProductID            string    `json:"product_id"`
	StripePaymentIntentID string   `json:"stripe_payment_intent_id"`
	CheckoutSessionID    string    `json:"checkout_session_id"`
	Amount               int64     `json:"amount"`
	Currency             string    `json:"currency"`
	Status               string    `json:"status"` // "pending", "succeeded", "failed", "canceled"
	PaymentType          string    `json:"payment_type"` // "one_time", "subscription", "credit_purchase"
	Metadata             JSONB     `json:"metadata"` // Additional payment data
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// UnifiedPaymentRequest represents the universal payment request for your ecosystem
type UnifiedPaymentRequest struct {
	ProjectID    string            `json:"project_id"`
	UserID       string            `json:"user_id"`
	Email        string            `json:"email"`
	ProductID    string            `json:"product_id"`
	SuccessURL   string            `json:"success_url"`
	CancelURL    string            `json:"cancel_url"`
	Metadata     map[string]string `json:"metadata"`
	Quantity     int64             `json:"quantity,omitempty"`
}

// UnifiedPaymentResponse represents the universal payment response
type UnifiedPaymentResponse struct {
	PaymentID         string `json:"payment_id"`
	CheckoutSessionID string `json:"checkout_session_id"`
	CheckoutURL       string `json:"checkout_url"`
	ProjectID         string `json:"project_id"`
	Amount            int64  `json:"amount"`
	Currency          string `json:"currency"`
}

// ProjectRegistrationRequest represents project registration with automatic setup
type ProjectRegistrationRequest struct {
	ProjectID     string                `json:"project_id"`
	Name          string                `json:"name"`
	DisplayName   string                `json:"display_name"`
	Domain        string                `json:"domain"`
	Type          string                `json:"type"`
	Environment   string                `json:"environment"`
	Products      []ProductRegistration `json:"products"`
}

// ProductRegistration represents a product to be created in Stripe
type ProductRegistration struct {
	ProductID     string `json:"product_id"`
	Name          string `json:"name"`
	DisplayName   string `json:"display_name"`
	Type          string `json:"type"`
	Amount        int64  `json:"amount"` // in cents
	Currency      string `json:"currency"`
	Interval      string `json:"interval,omitempty"` // for subscriptions
	Recurring     bool   `json:"recurring"`
}

// ProjectRegistrationResponse represents the response after project registration
type ProjectRegistrationResponse struct {
	Project      Project           `json:"project"`
	Products     []Product         `json:"products"`
	APIKey       string            `json:"api_key"`
	Endpoints    map[string]string `json:"endpoints"`
	Environment  map[string]string `json:"environment_variables"`
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
