#!/bin/bash

# Test script for the Go Stripe microservice

echo "Testing Go Stripe Payment Microservice"
echo "======================================"

# Check if service is running
echo "1. Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health)
if [ "$HEALTH_RESPONSE" -eq 200 ]; then
    echo "✓ Health check passed"
else
    echo "✗ Health check failed (status: $HEALTH_RESPONSE)"
    echo "Make sure the service is running: go run main.go"
    exit 1
fi

# Test payment intent creation
echo "2. Testing payment intent creation..."
PAYMENT_RESPONSE=$(curl -s -X POST http://localhost:8080/create-payment-intent \
    -H "Content-Type: application/json" \
    -d '{"amount": 1000, "currency": "usd"}')

if echo "$PAYMENT_RESPONSE" | grep -q "client_secret"; then
    echo "✓ Payment intent creation passed"
    echo "Response: $PAYMENT_RESPONSE"
else
    echo "✗ Payment intent creation failed"
    echo "Response: $PAYMENT_RESPONSE"
fi

# Test test payment intent creation
echo "3. Testing test payment intent creation..."
TEST_PAYMENT_RESPONSE=$(curl -s -X POST http://localhost:8080/create-test-payment-intent \
    -H "Content-Type: application/json" \
    -d '{"amount": 1000, "currency": "usd"}')

if echo "$TEST_PAYMENT_RESPONSE" | grep -q "client_secret"; then
    echo "✓ Test payment intent creation passed"
    echo "Response: $TEST_PAYMENT_RESPONSE"
else
    echo "✗ Test payment intent creation failed"
    echo "Response: $TEST_PAYMENT_RESPONSE"
fi

echo "Testing complete!"
