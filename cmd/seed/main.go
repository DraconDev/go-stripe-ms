package main

import (
	"context"
	"fmt"
	"log"

	"styx/internal/config"
	"styx/internal/database"
	
	"github.com/jackc/pgx/v5"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	conn, err := pgx.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(context.Background())

	// Initialize database repository
	repo := database.NewRepository(conn)

	// Create tables if they don't exist
	if err := repo.InitializeTables(context.Background()); err != nil {
		log.Fatalf("Failed to initialize tables: %v", err)
	}

	// Insert test user
	userID := "test-user-001"
	email := "test@example.com"
	stripeCustomerID := "cus_test_001"

	// Find or create customer
	fmt.Printf("Creating test user: %s (%s)\n", userID, email)
	
	// Insert customer directly
	customerID := "123e4567-e89b-12d3-a456-426614174000" // UUID for test
	_, err = conn.Exec(context.Background(), `
		INSERT INTO customers (id, user_id, email, stripe_customer_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			email = EXCLUDED.email,
			stripe_customer_id = EXCLUDED.stripe_customer_id,
			updated_at = EXCLUDED.updated_at
	`, customerID, userID, email, stripeCustomerID)
	
	if err != nil {
		log.Fatalf("Failed to insert customer: %v", err)
	}

	// Insert test subscription
	productID := "prod_premium"
	priceID := "price_premium_monthly"
	stripeSubID := "sub_test_001"
	status := "active"

	_, err = conn.Exec(context.Background(), `
		INSERT INTO subscriptions (customer_id, user_id, product_id, price_id, stripe_subscription_id, status, current_period_start, current_period_end, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', NOW(), NOW())
		ON CONFLICT (user_id, product_id) DO UPDATE SET
			stripe_subscription_id = EXCLUDED.stripe_subscription_id,
			status = EXCLUDED.status,
			current_period_start = EXCLUDED.current_period_start,
			current_period_end = EXCLUDED.current_period_end,
			updated_at = EXCLUDED.updated_at
	`, customerID, userID, productID, priceID, stripeSubID, status)

	if err != nil {
		log.Fatalf("Failed to insert subscription: %v", err)
	}

	// Verify the data
	fmt.Println("\n✅ Test data inserted successfully!")
	fmt.Println("Test User Details:")
	fmt.Printf("  User ID: %s\n", userID)
	fmt.Printf("  Email: %s\n", email)
	fmt.Printf("  Stripe Customer ID: %s\n", stripeCustomerID)
	fmt.Printf("  Product: %s\n", productID)
	fmt.Printf("  Price: %s\n", priceID)
	fmt.Printf("  Subscription ID: %s\n", stripeSubID)
	fmt.Printf("  Status: %s\n", status)

	// Query to show current data
	rows, err := conn.Query(context.Background(), `
		SELECT 
			c.user_id,
			c.email,
			c.stripe_customer_id,
			s.product_id,
			s.price_id,
			s.stripe_subscription_id,
			s.status,
			s.current_period_start,
			s.current_period_end
		FROM customers c
		LEFT JOIN subscriptions s ON c.id = s.customer_id
		WHERE c.user_id = $1
	`, userID)

	if err != nil {
		log.Printf("Failed to query data: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("\n📊 Database verification:")
	for rows.Next() {
		var userID, email, stripeCustomerID, productID, priceID, stripeSubID, status string
		var periodStart, periodEnd interface{}

		err := rows.Scan(&userID, &email, &stripeCustomerID, &productID, &priceID, &stripeSubID, &status, &periodStart, &periodEnd)
		if err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}

		fmt.Printf("  %s - %s (Stripe: %s)\n", userID, email, stripeCustomerID)
		if productID != "" {
			fmt.Printf("    Subscription: %s (%s) - %s\n", productID, priceID, status)
		}
	}

	fmt.Println("\n🎉 Seed data setup complete! Your test user is ready for testing.")
}
