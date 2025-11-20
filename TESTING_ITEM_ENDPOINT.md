# Testing Your Item Endpoint

## Quick Answer: Customer ID
**You don't need to provide a customer ID!** The system automatically:
1. Checks your database for existing customer by `user_id`
2. Creates a new Stripe customer if none exists
3. Saves the Stripe customer ID back to your database

## Testing Approaches

### 1. curl (Direct HTTP Testing)

Since your server is running on port 8080:

```bash
# Basic test
curl -X POST http://localhost:8080/api/v1/checkout/item \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "email": "test@example.com",
    "product_id": "ebook_premium",
    "price_id": "price_1234567890",
    "quantity": 1,
    "success_url": "https://yourapp.com/success?session_id={CHECKOUT_SESSION_ID}",
    "cancel_url": "https://yourapp.com/cancel"
  }'

# With quantity
curl -X POST http://localhost:8080/api/v1/checkout/item \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_456",
    "email": "bulk@example.com",
    "product_id": "course_bundle",
    "price_id": "price_0987654321",
    "quantity": 5,
    "success_url": "https://yourapp.com/success?session_id={CHECKOUT_SESSION_ID}",
    "cancel_url": "https://yourapp.com/cancel"
  }'
```

### 2. Using the Integration Tests

```bash
# Run specific integration tests
go test -v -run "TestCreateItemCheckout" ./internal/server/
```

### 3. Postman/Insomnia

Create a POST request to:
- URL: `http://localhost:9000/api/v1/checkout/item`
- Headers: `Content-Type: application/json`
- Body: JSON as shown above

### 4. JavaScript Fetch

```javascript
fetch('http://localhost:9000/api/v1/checkout/item', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    user_id: 'user_js_123',
    email: 'javascript@example.com',
    product_id: 'digital_product',
    price_id: 'price_1234567890',
    quantity: 2,
    success_url: 'https://yourapp.com/success?session_id={CHECKOUT_SESSION_ID}',
    cancel_url: 'https://yourapp.com/cancel'
  })
})
.then(response => response.json())
.then(data => console.log('Checkout URL:', data.checkout_url));
```

## Required Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `user_id` | string | ✅ | Your internal user identifier |
| `email` | string | ✅ | User's email address |
| `product_id` | string | ✅ | Your internal product identifier |
| `price_id` | string | ✅ | Stripe Price ID (e.g., price_123...) |
| `success_url` | string | ✅ | Redirect URL on success |
| `cancel_url` | string | ✅ | Redirect URL on cancellation |
| `quantity` | integer | ❌ | Default: 1 |

## Expected Response

Success (200):
```json
{
  "checkout_session_id": "cs_test_1234567890",
  "checkout_url": "https://checkout.stripe.com/pay/cs_test_1234567890"
}
```

Error (400/500):
```json
{
  "error": {
    "type": "validation_error",
    "code": "VALIDATION_FAILED",
    "message": "Request validation failed",
    "description": "email is required",
    "field": "email"
  }
}
```

## Testing Different Scenarios

### 1. First-time Customer
```bash
curl -X POST http://localhost:9000/api/v1/checkout/item \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "new_user_123",
    "email": "new@example.com",
    "product_id": "starter_product",
    "price_id": "price_1111111111",
    "success_url": "https://yourapp.com/success?session_id={CHECKOUT_SESSION_ID}",
    "cancel_url": "https://yourapp.com/cancel"
  }'
```

### 2. Returning Customer
```bash
curl -X POST http://localhost:9000/api/v1/checkout/item \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "new_user_123",
    "email": "new@example.com",
    "product_id": "another_product",
    "price_id": "price_2222222222",
    "success_url": "https://yourapp.com/success?session_id={CHECKOUT_SESSION_ID}",
    "cancel_url": "https://yourapp.com/cancel"
  }'
```

### 3. Bulk Purchase
```bash
curl -X POST http://localhost:9000/api/v1/checkout/item \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "bulk_buyer",
    "email": "bulk@example.com",
    "product_id": "license_pack",
    "price_id": "price_3333333333",
    "quantity": 10,
    "success_url": "https://yourapp.com/success?session_id={CHECKOUT_SESSION_ID}",
    "cancel_url": "https://yourapp.com/cancel"
  }'
```

## Error Testing

### Missing Required Fields
```bash
curl -X POST http://localhost:9000/api/v1/checkout/item \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user"
    # Missing email, product_id, etc.
  }'
```

### Invalid Email
```bash
curl -X POST http://localhost:9000/api/v1/checkout/item \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
    "email": "invalid-email",
    "product_id": "test_product",
    "price_id": "price_test",
    "success_url": "https://example.com/success",
    "cancel_url": "https://example.com/cancel"
  }'
```

## What Happens During Testing

1. **Request Validation**: Checks all required fields
2. **Database Lookup**: Finds existing customer by `user_id`
3. **Customer Creation**: Creates Stripe customer if needed
4. **Database Update**: Saves Stripe customer ID
5. **Checkout Session**: Creates Stripe checkout session
6. **Response**: Returns checkout URL and session ID

## Troubleshooting

### Common Issues

1. **"Failed to create or find customer"**
   - Check Stripe API key configuration
   - Verify database connection

2. **"Missing required fields"**
   - Ensure all required fields are provided
   - Check JSON format

3. **"Invalid email format"**
   - Use valid email address
   - Check email field name (not `email_address`)

4. **Port issues**
   - Server runs on port 9000 (from your .env)
   - Health check: `curl http://localhost:9000/health`

### Check Server Status
```bash
curl http://localhost:9000/health
curl http://localhost:9000/debug
```

## Integration with Frontend

Once you get the `checkout_url`, redirect users:

```javascript
// After successful API call
window.location.href = response.checkout_url;
```

The Stripe checkout page will handle the payment process, and webhooks will update your database when payments complete.