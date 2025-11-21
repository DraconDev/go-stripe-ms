package cart

import "fmt"

// validateCartCheckoutRequest validates the cart checkout request
	if req.UserID == "" || req.Email == "" || len(req.Items) == 0 ||
		req.SuccessURL == "" || req.CancelURL == "" {
		return fmt.Errorf("Missing required fields")
	}

	// Validate cart items
	if len(req.Items) > 20 {
		return fmt.Errorf("Cart cannot contain more than 20 items")
	}

	// Validate each item
	for i, item := range req.Items {
		if item.PriceID == "" {
			return fmt.Errorf("Price ID is required for all items")
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("Quantity must be at least 1 for all items")
		}
	}

	return nil
}
