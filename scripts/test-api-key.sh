#!/bin/bash

# Load API_KEY from .env
if [ -f .env ]; then
    export $(grep "^API_KEY=" .env | xargs)
else
    echo "Error: .env file not found"
    exit 1
fi

if [ -z "$API_KEY" ]; then
    echo "Error: API_KEY not found in .env"
    exit 1
fi

echo "🔑 Using API Key: $API_KEY"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Running API Key Authentication Tests"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# Test 1: Health check (public endpoint)
echo "✅ Test 1: Health Check (Public - No Auth Required)"
curl -s http://localhost:9000/health | jq
echo ""

# Test 2: No API key
echo "❌ Test 2: Missing API Key (Should Fail)"
curl -s -X POST http://localhost:9000/api/v1/checkout/subscription | jq
echo ""

# Test 3: Wrong API key
echo "❌ Test 3: Wrong API Key (Should Fail)"
curl -s -X POST -H "X-API-Key: proj_wrong_key_123" http://localhost:9000/api/v1/checkout/subscription | jq
echo ""

# Test 4: Valid API key from .env
echo "✅ Test 4: Valid API Key from .env (Should Pass Auth)"
curl -s -X POST \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  http://localhost:9000/api/v1/checkout/subscription | jq
echo ""

# Test 5: Get subscription status with valid key
echo "✅ Test 5: Get Subscription Status (With Valid Key)"
curl -s -H "X-API-Key: $API_KEY" \
  http://localhost:9000/api/v1/subscriptions/test_user/prod_test | jq
echo ""

echo "═══════════════════════════════════════════════════════════════"
echo "Tests Complete!"
echo "═══════════════════════════════════════════════════════════════"
