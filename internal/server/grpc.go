package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"./internal/database"
	billing "./proto/billing"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BillingService implements the BillingService gRPC server
type BillingService struct {
	billing.UnimplementedBillingServiceServer
	db *database.Repository
}

// NewBillingService creates a new billing service instance
func NewBillingService(db *database.Repository) *BillingService {
	return &BillingService{
		db: db,
	}
}

// CreateSubscriptionCheckout initiates a Stripe subscription checkout session
func (s *BillingService) CreateSubscriptionCheckout(ctx context.Context, req *billing.CreateSubscriptionCheckoutRequest) (*billing.CreateSubscriptionCheckoutResponse, error) {
	log.Printf("CreateSubscriptionCheckout called for user: %s, product: %s", req.UserId, req.ProductId)

	// Get user details from metadata (assuming Cerberus service provides this)
	userDetails, err := s.getUserDetails(ctx, req.UserId)
	if err != nil {
		log.Printf("Failed to get user details: %v", err)
		return nil, status.Error(codes.FailedPrecondition, "failed to get user details")
	}

	// For now, create a mock checkout session - in production this would integrate with Stripe
	checkoutSessionID := fmt.Sprintf("cs_test_%s_%d", req.UserId, time.Now().Unix())
	checkoutURL := fmt.Sprintf("https://checkout.stripe.com/pay/%s", checkoutSessionID)

	log.Printf("Created checkout session: %s for user: %s", checkoutSessionID, req.UserId)

	return &billing.CreateSubscriptionCheckoutResponse{
		CheckoutSessionId: checkoutSessionID,
		CheckoutUrl:       checkoutURL,
	}, nil
}

// GetSubscriptionStatus retrieves the current status of a user's subscription
func (s *BillingService) GetSubscriptionStatus(ctx context.Context, req *billing.GetSubscriptionStatusRequest) (*billing.GetSubscriptionStatusResponse, error) {
	log.Printf("GetSubscriptionStatus called for user: %s, product: %s", req.UserId, req.ProductId)

	stripeSubID, customerID, currentPeriodEnd, exists, err := s.db.GetSubscriptionStatus(ctx, req.UserId, req.ProductId)
	if err != nil {
		log.Printf("Failed to get subscription status: %v", err)
		return nil, status.Error(codes.Internal, "failed to get subscription status")
	}

	if !exists {
		return &billing.GetSubscriptionStatusResponse{
			Exists: false,
		}, nil
	}

	return &billing.GetSubscriptionStatusResponse{
		SubscriptionId:    stripeSubID,
		Status:            "active", // Mock status - would get from Stripe API
		CustomerId:        customerID,
		CurrentPeriodEnd:  timestamppb.New(currentPeriodEnd),
		Exists:            true,
	}, nil
}

// CreateCustomerPortal creates a Stripe customer portal session for subscription management
func (s *BillingService) CreateCustomerPortal(ctx context.Context, req *billing.CreateCustomerPortalRequest) (*billing.CreateCustomerPortalResponse, error) {
	log.Printf("CreateCustomerPortal called for user: %s", req.UserId)

	// Get customer's Stripe ID from database
	customer, err := s.db.GetCustomerByStripeCustomerID(ctx, req.UserId)
	if err != nil {
		log.Printf("Failed to get customer: %v", err)
		return nil, status.Error(codes.NotFound, "customer not found")
	}

	// For now, create a mock portal session - in production this would integrate with Stripe
	portalSessionID := fmt.Sprintf("ps_test_%s_%d", req.UserId, time.Now().Unix())
	portalURL := fmt.Sprintf("https://billing.stripe.com/p/session/%s", portalSessionID)

	log.Printf("Created portal session: %s for user: %s", portalSessionID, req.UserId)

	return &billing.CreateCustomerPortalResponse{
		PortalSessionId: portalSessionID,
		PortalUrl:       portalURL,
	}, nil
}

// Helper methods

// getUserDetails retrieves user details from metadata (proxy for Cerberus service integration)
func (s *BillingService) getUserDetails(ctx context.Context, userID string) (*UserDetails, error) {
	// For now, extract from metadata (assuming Cerberus service provides this via metadata)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no metadata in context")
	}

	userEmails := md["user-email"]
	if len(userEmails) == 0 {
		// Fallback for development
		return &UserDetails{
			ID:    userID,
			Email: fmt.Sprintf("user+%s@example.com", userID),
		}, nil
	}

	return &UserDetails{
		ID:    userID,
		Email: userEmails[0],
	}, nil
}

// UserDetails represents user information
type UserDetails struct {
	ID    string
	Email string
}
