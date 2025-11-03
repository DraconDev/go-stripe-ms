-- Seed Single Test User for Stripe Billing Service
-- This script adds one test customer to your Neon DB

-- Insert test customer
INSERT INTO customers (user_id, email, stripe_customer_id) VALUES 
  ('test-user-001', 'test@example.com', 'cus_test_001')
ON CONFLICT (user_id) DO UPDATE SET
  email = EXCLUDED.email,
  stripe_customer_id = EXCLUDED.stripe_customer_id,
  updated_at = NOW();

-- Add active subscription for the test user
INSERT INTO subscriptions (customer_id, user_id, product_id, price_id, stripe_subscription_id, status, current_period_start, current_period_end) 
SELECT 
  c.id,
  'test-user-001',
  'prod_premium' as product_id,
  'price_premium_monthly' as price_id,
  'sub_test_001' as stripe_subscription_id,
  'active' as status,
  NOW() - INTERVAL '15 days' as current_period_start,
  NOW() + INTERVAL '15 days' as current_period_end
FROM customers c
WHERE c.user_id = 'test-user-001'
ON CONFLICT (user_id, product_id) DO UPDATE SET
  stripe_subscription_id = EXCLUDED.stripe_subscription_id,
  status = EXCLUDED.status,
  current_period_start = EXCLUDED.current_period_start,
  current_period_end = EXCLUDED.current_period_end,
  updated_at = NOW();

-- Verify the test data
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
WHERE c.user_id = 'test-user-001';
