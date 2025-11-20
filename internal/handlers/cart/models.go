package cart

// CartCheckoutRequest represents the request structure for cart checkout
type CartCheckoutRequest struct {
	UserID     string     `json:"user_id"`
	Email      string     `json:"email"`
	Items      []CartItem `json:"items"`
	SuccessURL string     `json:"success_url"`
	CancelURL  string     `json:"cancel_url"`
}

// CartItem represents an individual item in a cart
type CartItem struct {
	PriceID   string `json:"price_id"`
	Quantity  int64  `json:"quantity"`
	ProductID string `json:"product_id,omitempty"`
}
