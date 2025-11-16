package cart

import "fmt"

// validateCartCheckoutRequest validates the cart checkout request
func validateCartCheckoutRequest(req CartCheckoutRequest) error {
	if req.UserID == "" || req.Email == "" || len(req.Items) == 0 ||
		req.SuccessURL == "" || req.CancelURL == "" {
		return fmt.Errorf("missing required fields")
	}

	// Validate cart items
	if len(req.Items) > 20 {
		return fmt.Errorf("cart cannot contain more than 20 items")
	}

	// Validate each item quantity
	for i, item := range req.Items {
		if item.Quantity <= 0 {
			return fmt.Errorf("item %d has invalid quantity", i+1)
		}
	}

	return nil
}