package webhooks

import (
	"encoding/json"
	"log"

	"github.com/stripe/stripe-go/v72"
)

// handleInvoicePaymentSucceeded processes successful payment events
func (h *StripeWebhookHandler) handleInvoicePaymentSucceeded(event stripe.Event) {
	var invoice struct {
		ID         string `json:"id"`
		AmountPaid int64  `json:"amount_paid"`
	}
	
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		log.Printf("Error unmarshaling invoice event: %v", err)
		return
	}

	log.Printf("Payment succeeded for invoice: %s, amount: %d", invoice.ID, invoice.AmountPaid)
	// Additional logic for successful payments can be added here
}

// handleInvoicePaymentFailed processes failed payment events
func (h *StripeWebhookHandler) handleInvoicePaymentFailed(event stripe.Event) {
	var invoice struct {
		ID        string `json:"id"`
		AmountDue int64  `json:"amount_due"`
	}
	
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		log.Printf("Error unmarshaling invoice event: %v", err)
		return
	}

	log.Printf("Payment failed for invoice: %s, amount: %d", invoice.ID, invoice.AmountDue)
	// Additional logic for failed payments can be added here (e.g., notifications)
}

// handlePaymentMethodAttached processes payment method attachment events
func (h *StripeWebhookHandler) handlePaymentMethodAttached(event stripe.Event) {
	var paymentMethod struct {
		ID       string `json:"id"`
		Customer string `json:"customer"`
	}
	
	if err := json.Unmarshal(event.Data.Raw, &paymentMethod); err != nil {
		log.Printf("Error unmarshaling payment method event: %v", err)
		return
	}

	log.Printf("Payment method attached: %s for customer: %s", paymentMethod.ID, paymentMethod.Customer)
	// Additional logic for payment method updates can be added here
}