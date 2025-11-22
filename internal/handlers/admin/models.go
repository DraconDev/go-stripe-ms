package admin

import "time"

// ProductRegistrationRequest represents the request to register products
type ProductRegistrationRequest struct {
	ProjectName string `json:"project_name"`
	Plans       []Plan `json:"plans"`
}

// Plan represents a subscription plan to be created
type Plan struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Features    []string `json:"features,omitempty"`
	Pricing     Pricing  `json:"pricing"`
}

// Pricing represents the pricing structure for a plan
type Pricing struct {
	Monthly int64 `json:"monthly"` // in cents
	Yearly  int64 `json:"yearly,omitempty"`
}

// ProductResponse represents the response after creating a product
type ProductResponse struct {
	PlanName        string        `json:"plan_name"`
	StripeProductID string        `json:"stripe_product_id"`
	Prices          PriceResponse `json:"prices"`
	CreatedAt       time.Time     `json:"created_at"`
}

// PriceResponse contains the created price details
type PriceResponse struct {
	Monthly *PriceDetails `json:"monthly,omitempty"`
	Yearly  *PriceDetails `json:"yearly,omitempty"`
}

// PriceDetails contains details about a specific price
type PriceDetails struct {
	StripePriceID string `json:"stripe_price_id"`
	Amount        int64  `json:"amount"`
	Interval      string `json:"interval"`
	Currency      string `json:"currency"`
}

// RegistrationResponse is the top-level response
type RegistrationResponse struct {
	Success   bool              `json:"success"`
	ProjectID string            `json:"project_id"`
	Products  []ProductResponse `json:"products"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success           bool   `json:"success"`
	Error             string `json:"error"`
	Message           string `json:"message"`
	Details           string `json:"details,omitempty"`
	ExistingProductID string `json:"existing_product_id,omitempty"`
}
