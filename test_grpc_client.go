package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"styx/proto/billing_service/proto/billing"

	"google.golang.org/grpc"
)

func main() {
	// Set up a connection to the server
	conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := billing.NewBillingServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test CreateSubscriptionCheckout
	fmt.Println("Testing CreateSubscriptionCheckout...")
	checkoutResp, err := c.CreateSubscriptionCheckout(ctx, &billing.CreateSubscriptionCheckoutRequest{
		UserId:    "test-user-123",
		ProductId: "prod_test_product",
		PriceId:   "price_test_price",
		SuccessUrl: "https://example.com/success",
		CancelUrl:  "https://example.com/cancel",
	})
	if err != nil {
		log.Printf("CreateSubscriptionCheckout failed: %v", err)
	} else {
		log.Printf("CreateSubscriptionCheckout success:")
		log.Printf("  Session ID: %s", checkoutResp.CheckoutSessionId)
		log.Printf("  Checkout URL: %s", checkoutResp.CheckoutUrl)
	}

	// Test GetSubscriptionStatus
	fmt.Println("Testing GetSubscriptionStatus...")
	statusResp, err := c.GetSubscriptionStatus(ctx, &billing.GetSubscriptionStatusRequest{
		UserId:    "test-user-123",
		ProductId: "prod_test_product",
	})
	if err != nil {
		log.Printf("GetSubscriptionStatus failed: %v", err)
	} else {
		log.Printf("GetSubscriptionStatus success:")
		log.Printf("  Subscription ID: %s", statusResp.SubscriptionId)
		log.Printf("  Status: %s", statusResp.Status)
		log.Printf("  Customer ID: %s", statusResp.CustomerId)
		log.Printf("  Exists: %t", statusResp.Exists)
	}

	// Test CreateCustomerPortal
	fmt.Println("Testing CreateCustomerPortal...")
	portalResp, err := c.CreateCustomerPortal(ctx, &billing.CreateCustomerPortalRequest{
		UserId:    "test-user-123",
		ReturnUrl: "https://example.com/dashboard",
	})
	if err != nil {
		log.Printf("CreateCustomerPortal failed: %v", err)
	} else {
		log.Printf("CreateCustomerPortal success:")
		log.Printf("  Portal Session ID: %s", portalResp.PortalSessionId)
		log.Printf("  Portal URL: %s", portalResp.PortalUrl)
	}

	fmt.Println("All tests completed!")
}
