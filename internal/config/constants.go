package config

// Stripe Constants
// These are identifiers, not secrets, so they are safe to commit to code.
const (
	// // Subscription Plans
	// SubscriptionPriceID_Monthly = "price_1QVMBhFhH6dwUiIHygh8SRUY"
	// SubscriptionPriceID_Yearly  = "price_1QVMgeFhH6dwUiIHTm5oaxxJ"

	// // Products
	// ProductID_Premium = "prod_RZaVDAN6Uf4Qfb"

	// Test Data (Optional - useful for integration tests)
	TestCustomerID                    = "cus_TQk7UsTFXZcMYC"
	TestProductID                     = "prod_RZaVDAN6Uf4Qfb"
	TestProductPriceID                = "price_1QhEBSFhH6dwUiIHSUnHP957"
	TestSubscriptionProductID_Monthly = "prod_RO8SNJEePdNojr"
	TestSubscriptionPriceID_Monthly   = "price_1QVMBhFhH6dwUiIHygh8SRUY"
	TestSubscriptionProductID_Yearly  = "prod_RO8yZ5Z9yzvuuY"
	TestSubscriptionPriceID_Yearly    = "price_1QVMgeFhH6dwUiIHTm5oaxxJ"

	// Convenience aliases for tests
	TEST_PRICE_ID                = TestProductPriceID                // For item/cart checkout tests
	TEST_SUBSCRIPTION_PRICE_ID   = TestSubscriptionPriceID_Monthly   // For subscription checkout tests
	TEST_SUBSCRIPTION_PRODUCT_ID = TestSubscriptionProductID_Monthly // For subscription checkout tests
)
