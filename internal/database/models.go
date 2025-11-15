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
	ID                 uuid.UUID `json:"id"`
	CustomerID         uuid.UUID `json:"customer_id"`
	UserID             string    `json:"user_id"`
	ProductID          string    `json:"product_id"`
	PriceID            string    `json:"price_id"`
	StripeSubscriptionID string  `json:"stripe_subscription_id"`
	Status             string    `json:"status"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
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
// MULTI-PROJECT SUPPORT MODELS
// ========================

// Project represents a project configuration for multi-tenant payments
type Project struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	ProjectID        string    `json:"project_id"` // External identifier (e.g., "ecom_001")
	Domain           string    `json:"domain"`
	PaymentTypes     []string  `json:"payment_types"` // ["one_time"], ["subscription"], ["both"]
	DefaultCurrency  string    `json:"default_currency"`
	Features         []string  `json:"features"` // ["credits", "permissions", "metered"]
	SuccessURL       string    `json:"success_url"`
	CancelURL        string    `json:"cancel_url"`
	WebhookURL       string    `json:"webhook_url"`
	StripeMapping    JSONB     `json:"stripe_mapping"` // product_id -> price_id mapping
	Environment      string    `json:"environment"`    // "development", "production"
	Status           string    `json:"status"`         // "active", "inactive", "suspended"
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Product represents a product within a project
type Product struct {
	ID               uuid.UUID `json:"id"`
	ProjectID        string    `json:"project_id"`
	ProductID        string    `json:"product_id"` // External identifier
	Name             string    `json:"name"`
	Type             string    `json:"type"` // "one_time", "subscription", "credit", "permission"
	PriceID          string    `json:"price_id"` // Stripe price ID
	Recurring        bool      `json:"recurring"`
	Interval         string    `json:"interval"` // "month", "year" (for subscriptions)
	Currency         string    `json:"currency"`
	UnitAmount       int64     `json:"unit_amount"` // in cents
	Metadata         JSONB     `json:"metadata"` // Additional product data
	Active           bool      `json:"active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Payment represents a one-time payment transaction
type Payment struct {
	ID                uuid.UUID `json:"id"`
	ProjectID         string    `json:"project_id"`
	CustomerID        uuid.UUID `json:"customer_id"`
	UserID            string    `json:"user_id"`
	ProductID         string    `json:"product_id"`
	PriceID           string    `json:"price_id"`
	StripePaymentIntentID string `json:"stripe_payment_intent_id"`
	CheckoutSessionID string    `json:"checkout_session_id"`
	Amount            int64     `json:"amount"`
	Currency          string    `json:"currency"`
	Status            string    `json:"status"` // "pending", "succeeded", "failed", "canceled"
	PaymentType       string    `json:"payment_type"` // "one_time", "credit_purchase"
	Metadata          JSONB     `json:"metadata"` // Additional payment data
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Event represents events for event-driven architecture
type Event struct {
	ID          uuid.UUID  `json:"id"`
	Type        string     `json:"type"`        // "payment.succeeded", "subscription.created", etc.
	Source      string     `json:"source"`      // "payment-service", "auth-service"
	ProjectID   string     `json:"project_id"`  // Multi-project context
	UserID      string     `json:"user_id"`
	Data        JSONB      `json:"data"`        // Event payload
	Metadata    JSONB      `json:"metadata"`    // Additional event data
	Published   bool       `json:"published"`   // Whether event has been published
	ProcessedAt *time.Time `json:"processed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ProjectConfig represents project configuration for API responses
type ProjectConfig struct {
	ID              string            `json:"id"`
	ProjectID       string            `json:"project_id"`
	Name            string            `json:"name"`
	Domain          string            `json:"domain"`
	PaymentTypes    []string          `json:"payment_types"`
	DefaultCurrency string            `json:"default_currency"`
	Features        []string          `json:"features"`
	EnvironmentVars map[string]string `json:"environment_variables"`
	APIEndpoints    map[string]string `json:"api_endpoints"`
	StripeMapping   map[string]string `json:"stripe_mapping"`
}

// UnifiedPaymentRequest represents the universal payment request
type UnifiedPaymentRequest struct {
	UserID       string            `json:"user_id"`
	Email        string            `json:"email"`
	ProductID    string            `json:"product_id"`
	PaymentType  string            `json:"payment_type"` // "auto", "one_time", "subscription"
	PriceID      string            `json:"price_id"`
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
	PaymentType       string `json:"payment_type"`
	ProjectID         string `json:"project_id"`
	Amount            int64  `json:"amount"`
	Currency          string `json:"currency"`
}

// PaymentStatus represents universal payment status response
type PaymentStatus struct {
	PaymentID         string                 `json:"payment_id"`
	Status            string                 `json:"status"`
	Amount            int64                  `json:"amount"`
	Currency          string                 `json:"currency"`
	PaymentType       string                 `json:"payment_type"`
	ProjectID         string                 `json:"project_id"`
	CreatedAt         time.Time              `json:"created_at"`
	Metadata          map[string]interface{} `json:"metadata"`
	StripePaymentIntentID string             `json:"stripe_payment_intent_id,omitempty"`
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

// ========================
// SCAN FUNCTIONS FOR NEW MODELS
// ========================

// ScanProject scans a database row into a Project struct
func ScanProject(row pgx.Row) (*Project, error) {
	var project Project
	err := row.Scan(
		&project.ID,
		&project.Name,
		&project.ProjectID,
		&project.Domain,
		&project.PaymentTypes,
		&project.DefaultCurrency,
		&project.Features,
		&project.SuccessURL,
		&project.CancelURL,
		&project.WebhookURL,
		&project.StripeMapping,
		&project.Environment,
		&project.Status,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// ScanProduct scans a database row into a Product struct
func ScanProduct(row pgx.Row) (*Product, error) {
	var product Product
	err := row.Scan(
		&product.ID,
		&product.ProjectID,
		&product.ProductID,
		&product.Name,
		&product.Type,
		&product.PriceID,
		&product.Recurring,
		&product.Interval,
		&product.Currency,
		&product.UnitAmount,
		&product.Metadata,
			&product.Active,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// ScanPayment scans a database row into a Payment struct
func ScanPayment(row pgx.Row) (*Payment, error) {
	var payment Payment
	err := row.Scan(
		&payment.ID,
		&payment.ProjectID,
		&payment.CustomerID,
		&payment.UserID,
		&payment.ProductID,
		&payment.PriceID,
		&payment.StripePaymentIntentID,
		&payment.CheckoutSessionID,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.PaymentType,
		&payment.Metadata,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// ScanEvent scans a database row into an Event struct
func ScanEvent(row pgx.Row) (*Event, error) {
	var event Event
	err := row.Scan(
		&event.ID,
		&event.Type,
		&event.Source,
		&event.ProjectID,
		&event.UserID,
		&event.Data,
		&event.Metadata,
		&event.Published,
		&event.ProcessedAt,
		&event.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &event, nil
}
func (b sqlBool) Bool() bool     { return bool(b) }
