package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"styx/internal/database"
	billing "styx/proto"

	"github.com/stripe/stripe-go/v72"
	checkoutsession "github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/sub"
	billingportal "github.com/stripe/stripe-go/v72/billing_portal/session"
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
	if req.SuccessUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "success_url is required")
	}
	if req.CancelUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "cancel_url is required")
	}

	// Find or create Stripe customer using the email field
	stripeCustomerID, err := s.findOrCreateStripeCustomer(ctx, req.UserId, req.Email)
	if err != nil {
		log.Printf("Failed to find or create Stripe customer for user %s: %v", req.UserId, err)
		return nil, status.Error(codes.Internal, "failed to create or find customer")
	}

	// Create real Stripe Checkout Session
	checkoutParams := &checkoutsession.CheckoutSessionParams{
		Customer:  stripe.String(stripeCustomerID),
		Mode:      stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*checkoutsession.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.PriceId),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL:            stripe.String(req.SuccessUrl),
		CancelURL:             stripe.String(req.CancelUrl),
		ClientReferenceID:     stripe.String(req.UserId),
		AllowPromotionCodes:   stripe.Bool(true),
		BillingAddressCollection: stripe.String(string(stripe.CheckoutSessionBillingAddressCollectionRequired)),
	}

	// Add metadata
	checkoutParams.AddMetadata("user_id", req.UserId)
	checkoutParams.AddMetadata("product_id", req.ProductId)

	checkoutSession, err := checkoutsession.New(checkoutParams)
	if err != nil {
		log.Printf("Failed to create Stripe checkout session for user %s: %v", req.UserId, err)
		return nil, status.Errorf(codes.Internal, "failed to create checkout session: %v", err)
	}

	log.Printf("Created Stripe checkout session: %s for user: %s with Stripe customer: %s", checkoutSession.ID, req.UserId, stripeCustomerID)

	return &billing.CreateSubscriptionCheckoutResponse{
		CheckoutSessionId: checkoutSession.ID,
		CheckoutUrl:       checkoutSession.URL,
	}, nil
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

	// Validate required fields
	if req.ReturnUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "return_url is required")
	}

	// Get user's Stripe customer ID
	customer, err := s.db.GetCustomerByUserID(ctx, req.UserId)
	if err != nil {
		log.Printf("Failed to get customer for user %s: %v", req.UserId, err)
		return nil, status.Error(codes.NotFound, "customer not found")
	}

	if customer.StripeCustomerID == "" {
		return nil, status.Error(codes.FailedPrecondition, "customer has no Stripe customer ID")
	}

	// Create real Stripe Billing Portal session
	portalParams := &billingportal.SessionParams{
		Customer:  stripe.String(customer.StripeCustomerID),
		ReturnURL: stripe.String(req.ReturnUrl),
	}

	portalSession, err := billingportal.Session.New(portalParams)
	if err != nil {
		log.Printf("Failed to create Stripe portal session for customer %s: %v", customer.StripeCustomerID, err)
		return nil, status.Errorf(codes.Internal, "failed to portal session: %v", err)
	}

	log.Printf("Created Stripe portal session: %s for user: %s", portalSession.ID, req.UserId)

	return &billing.CreateCustomerPortalResponse{
		PortalSessionId: portalSession.ID,
		PortalUrl:       portalSession.URL,
	}, nil
}

// CancelSubscription cancels a user's subscription
func (s *BillingService) CancelSubscription(ctx context.Context, req *billing.CancelSubscriptionRequest) (*billing.CancelSubscriptionResponse, error) {
	log.Printf("CancelSubscription called for user: %s, product: %s", req.UserId, req.ProductId)

	// Validate required fields
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	// Get user's subscription from database
	stripeSubID, _, _, exists, err := s.db.GetSubscriptionStatus(ctx, req.UserId, req.ProductId)
	if err != nil {
		log.Printf("Failed to get subscription status from database: %v", err)
		return nil, status.Error(codes.Internal, "failed to get subscription status")
	}

	if !exists {
		return nil, status.Error(codes.NotFound, "subscription not found")
	}

	// Cancel the Stripe subscription
	cancelParams := &sub.CancelParams{
		InvoiceNow: stripe.Bool(req.InvoiceNow),
		Prorate:    stripe.Bool(req.Prorate),
	}

	// If not specified, use default values
	if !req.InvoiceNow {
		cancelParams.InvoiceNow = stripe.Bool(false)
	}
	if !req.Prorate {
		cancelParams.Prorate = stripe.Bool(true)
	}

	cancelledSubscription, err := sub.Cancel(stripeSubID, cancelParams)
	if err != nil {
		log.Printf("Failed to cancel Stripe subscription %s: %v", stripeSubID, err)
		return nil, status.Errorf(codes.Internal, "failed to cancel subscription: %v", err)
	}

	// Update database with cancellation status
	err = s.db.UpdateSubscriptionStatus(ctx, req.UserId, req.ProductId, time.Now())
	if err != nil {
		log.Printf("Failed to update subscription status in database: %v", err)
		// Don't return error here as Stripe cancellation succeeded
	}

	log.Printf("Cancelled Stripe subscription: %s for user: %s", cancelledSubscription.ID, req.UserId)

	return &billing.CancelSubscriptionResponse{
		SubscriptionId: cancelledSubscription.ID,
		Status:         string(cancelledSubscription.Status),
		CanceledAt:     timestamppb.New(time.Unix(cancelledSubscription.CanceledAt, 0)),
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
