package database

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Project represents a project that can use the payment service
type Project struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	APIKey     string    `json:"api_key"`
	WebhookURL string    `json:"webhook_url"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Customer represents a user customer record
type Customer struct {
	ID               uuid.UUID `json:"id"`
	ProjectID        uuid.UUID `json:"project_id"`
	UserID           string    `json:"user_id"`
	Email            string    `json:"email"`
	StripeCustomerID string    `json:"stripe_customer_id"` // May be empty until Stripe customer is created
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Subscription represents a subscription record
type Subscription struct {
	ID                   uuid.UUID `json:"id"`
	ProjectID            uuid.UUID `json:"project_id"`
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

// RegisteredProduct represents a product registered for a project
type RegisteredProduct struct {
	ID                 uuid.UUID `json:"id"`
	ProjectName        string    `json:"project_name"`
	PlanName           string    `json:"plan_name"`
	StripeProductID    string    `json:"stripe_product_id"`
	StripePriceMonthly string    `json:"stripe_price_monthly,omitempty"`
	StripePriceYearly  string    `json:"stripe_price_yearly,omitempty"`
	MonthlyAmount      int64     `json:"monthly_amount,omitempty"`
	YearlyAmount       int64     `json:"yearly_amount,omitempty"`
	Currency           string    `json:"currency"`
	Description        string    `json:"description,omitempty"`
	Features           []byte    `json:"features,omitempty"` // JSON bytes
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ScanProject scans a database row into a Project struct
func ScanProject(row pgx.Row) (*Project, error) {
	var project Project
	err := row.Scan(
		&project.ID,
		&project.Name,
		&project.APIKey,
		&project.WebhookURL,
		&project.IsActive,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// ScanCustomer scans a customer row from the database
func ScanCustomer(row pgx.Row) (*Customer, error) {
	var customer Customer
	var stripeCustomerID sql.NullString

	err := row.Scan(
		&customer.ID,
		&customer.ProjectID,
		&customer.UserID,
		&customer.Email,
		&stripeCustomerID,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert sql.NullString to string
	if stripeCustomerID.Valid {
		customer.StripeCustomerID = stripeCustomerID.String
	}

	return &customer, nil
}

// ScanSubscription scans a database row into a Subscription struct
func ScanSubscription(row pgx.Row) (*Subscription, error) {
	var sub Subscription
	err := row.Scan(
		&sub.ID,
		&sub.ProjectID,
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

// ScanRegisteredProduct scans a database row into a RegisteredProduct struct
func ScanRegisteredProduct(row pgx.Row) (*RegisteredProduct, error) {
	var product RegisteredProduct
	var monthlyAmount, yearlyAmount sql.NullInt64
	var stripePriceMonthly, stripePriceYearly sql.NullString

	err := row.Scan(
		&product.ID,
		&product.ProjectName,
		&product.PlanName,
		&product.StripeProductID,
		&stripePriceMonthly,
		&stripePriceYearly,
		&monthlyAmount,
		&yearlyAmount,
		&product.Currency,
		&product.Description,
		&product.Features,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if stripePriceMonthly.Valid {
		product.StripePriceMonthly = stripePriceMonthly.String
	}
	if stripePriceYearly.Valid {
		product.StripePriceYearly = stripePriceYearly.String
	}
	if monthlyAmount.Valid {
		product.MonthlyAmount = monthlyAmount.Int64
	}
	if yearlyAmount.Valid {
		product.YearlyAmount = yearlyAmount.Int64
	}

	return &product, nil
}

// ScanSubscriptionStatus scans a database row into subscription status fields
func ScanSubscriptionStatus(row pgx.Row) (string, string, time.Time, bool, error) {
	var stripeSubID, customerID sqlString
	var currentPeriodEnd sqlTime
	var exists sqlBool

	err := row.Scan(&stripeSubID, &customerID, &currentPeriodEnd, &exists)
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
func (b sqlBool) Bool() bool      { return bool(b) }
