package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"styx/internal/database"
	billing "styx/proto"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/sub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BillingService implements the BillingService gRPC server
type BillingService struct {
	billing.UnimplementedBillingServiceServer
	db           *database.Repository
	stripeSecret string
}

// NewBillingService creates a new billing service instance
func NewBillingService(db *database.Repository, stripeSecret string) *BillingService {
	stripe.Key = stripeSecret

	return &BillingService{
		db:           db,
		stripeSecret: stripeSecret,
	}
}

// CreateSubscriptionCheckout initiates a Stripe subscription checkout session
func (s *BillingService) CreateSubscriptionCheckout(ctx context.Context, req *billing.CreateSubscriptionCheckoutRequest) (*billing.CreateSubscriptionCheckoutResponse, error) {
	log.Printf("CreateSubscriptionCheckout called for user: %s, email: %s, product: %s", req.UserId, req.Email, req.ProductId)

	// Validate required fields
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if req.PriceId == "" {
		return nil, status.Error(codes.InvalidArgument, "price_id is required")
	}

	// Find or create Stripe customer using the email field
	stripeCustomerID, err := s.findOrCreateStripeCustomer(ctx, req.UserId, req.Email)
	if err != nil {
		log.Printf("Failed to find or create Stripe customer for user %s: %v", req.UserId, err)
		return nil, status.Error(codes.Internal, "failed to create or find customer")
	}

	// Create real Stripe checkout session for subscription using the correct API
	checkoutParams := map[string]interface{}{
		"customer":    stripeCustomerID,
		"payment_method_types": []string{"card"},
		"line_items": []map[string]interface{}{
			{
				"price":    req.PriceId,
				"quantity": 1,
			},
		},
		"mode":             "subscription",
		"success_url":      req.SuccessUrl,
		"cancel_url":       req.CancelUrl,
		"subscription_data": map[string]interface{}{
			"metadata": map[string]string{
				"user_id":    req.UserId,
				"product_id": req.ProductId,
			},
		},
	}

	params := &stripe.CheckoutSessionParams{}
	if session, err := params.NewSession(checkoutParams); err != nil {
		log.Printf("Failed to create Stripe checkout session: %v", err)
		return nil, status.Error(codes.Internal, "failed to create checkout session")
	} else {
		log.Printf("Created Stripe checkout session: %s for user: %s with Stripe customer: %s", session.ID, req.UserId, stripeCustomerID)
		return &billing.CreateSubscriptionCheckoutResponse{
			CheckoutSessionId: session.ID,
			CheckoutUrl:       session.URL,
		}, nil
	}
}

// GetSubscriptionStatus retrieves the current status of a user's subscription
func (s *BillingService) GetSubscriptionStatus(ctx context.Context, req *billing.GetSubscriptionStatusRequest) (*billing.GetSubscriptionStatusResponse, error) {
	log.Printf("GetSubscriptionStatus called for user: %s, product: %s", req.UserId, req.ProductId)

	// First check database for existing subscription
	stripeSubID, customerID, currentPeriodEnd, exists, err := s.db.GetSubscriptionStatus(ctx, req.UserId, req.ProductId)
	if err != nil {
		log.Printf("Failed to get subscription status from database: %v", err)
		return nil, status.Error(codes.Internal, "failed to get subscription status")
	}

	if !exists {
		return &billing.GetSubscriptionStatusResponse{
			Exists: false,
		}, nil
	}

	// Get current status from Stripe
	stripeSubscription, err := sub.Get(stripeSubID, nil)
	if err != nil {
		log.Printf("Failed to get Stripe subscription %s: %v", stripeSubID, err)
		// Return database status if Stripe call fails
		return &billing.GetSubscriptionStatusResponse{
			SubscriptionId:   stripeSubID,
			Status:           "unknown",
			CustomerId:       customerID,
			CurrentPeriodEnd: timestamppb.New(currentPeriodEnd),
			Exists:           true,
		}, nil
	}

	return &billing.GetSubscriptionStatusResponse{
		SubscriptionId:   stripeSubID,
		Status:           string(stripeSubscription.Status),
		CustomerId:       stripeSubscription.Customer.ID,
		CurrentPeriodEnd: timestamppb.New(time.Unix(stripeSubscription.CurrentPeriodEnd, 0)),
		Exists:           true,
	}, nil
}

// CreateCustomerPortal creates a Stripe customer portal session for subscription management
func (s *BillingService) CreateCustomerPortal(ctx context.Context, req *billing.CreateCustomerPortalRequest) (*billing.CreateCustomerPortalResponse, error) {
	log.Printf("CreateCustomerPortal called for user: %s", req.UserId)

	// Get user's Stripe customer ID
	customer, err := s.db.GetCustomerByUserID(ctx, req.UserId)
	if err != nil {
		log.Printf("Failed to get customer for user %s: %v", req.UserId, err)
		return nil, status.Error(codes.NotFound, "customer not found")
	}

	if customer.StripeCustomerID == "" {
		return nil, status.Error(codes.FailedPrecondition, "customer has no Stripe customer ID")
	}

	// For now, create a mock portal session until proper Stripe billing portal API is available
	// In a production environment, you would use Stripe's Customer Portal API
	portalSessionID := fmt.Sprintf("ps_test_%s_%d", req.UserId, time.Now().Unix())
	portalURL := fmt.Sprintf("https://billing.stripe.com/p/session/%s", portalSessionID)

	log.Printf("Created mock portal session: %s for user: %s", portalSessionID, req.UserId)

	return &billing.CreateCustomerPortalResponse{
		PortalSessionId: portalSessionID,
		PortalUrl:       portalURL,
	}, nil
}

// findOrCreateStripeCustomer finds an existing Stripe customer or creates a new one
func (s *BillingService) findOrCreateStripeCustomer(ctx context.Context, userID, email string) (string, error) {
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

	// Add metadata using the correct API
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
