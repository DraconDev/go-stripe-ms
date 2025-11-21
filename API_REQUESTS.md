# API Request Examples

Quick reference for all Payment MS endpoints and their expected request formats.

---

## Authentication

All `/api/v1/*` endpoints require the `X-API-Key` header:
```bash
X-API-Key: your_api_key_here
```

---

## Checkout Endpoints

### 1. Subscription Checkout
**Endpoint:** `POST /api/v1/checkout/subscription`

**Request Body:**
```json
{
  "user_id": "user_123",
  "email": "customer@example.com",
  "product_id": "prod_RZaVDAN6Uf4Qfb",
  "price_id": "price_1QhEBSFhH6dwUiIHSUnHP957",
  "success_url": "https://yourapp.com/success",
  "cancel_url": "https://yourapp.com/cancel"
}
```

**Response:**
```json
{
  "checkout_session_id": "cs_test_...",
  "checkout_url": "https://checkout.stripe.com/c/pay/cs_test_..."
}
```

**Example:**
```bash
curl -X POST http://localhost:9000/api/v1/checkout/subscription \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "email": "customer@example.com",
    "product_id": "prod_RZaVDAN6Uf4Qfb",
    "price_id": "price_1QhEBSFhH6dwUiIHSUnHP957",
    "success_url": "https://yourapp.com/success",
    "cancel_url": "https://yourapp.com/cancel"
  }'
```

---

### 2. Single Item Checkout
**Endpoint:** `POST /api/v1/checkout/item`

**Request Body:**
```json
{
  "user_id": "user_123",
  "email": "customer@example.com",
  "price_id": "price_1QhEBSFhH6dwUiIHSUnHP957",
  "quantity": 1,
  "success_url": "https://yourapp.com/success",
  "cancel_url": "https://yourapp.com/cancel"
}
```

**Response:**
```json
{
  "checkout_session_id": "cs_test_...",
  "checkout_url": "https://checkout.stripe.com/c/pay/cs_test_..."
}
```

---

### 3. Cart Checkout (Multiple Items)
**Endpoint:** `POST /api/v1/checkout/cart`

**Request Body:**
```json
{
  "user_id": "user_123",
  "email": "customer@example.com",
  "items": [
    {
      "price_id": "price_1QhEBSFhH6dwUiIHSUnHP957",
      "quantity": 2
    },
    {
      "price_id": "price_1QhEBSFhH6dwUiIHAnotherID",
      "quantity": 1
    }
  ],
  "success_url": "https://yourapp.com/success",
  "cancel_url": "https://yourapp.com/cancel"
}
```

**Response:**
```json
{
  "checkout_session_id": "cs_test_...",
  "checkout_url": "https://checkout.stripe.com/c/pay/cs_test_..."
}
```

---

## Subscription Management

### 4. Get Subscription Status
**Endpoint:** `GET /api/v1/subscriptions/{user_id}/{product_id}`

**URL Parameters:**
- `user_id`: The user's ID
- `product_id`: The Stripe product ID

**Response (Active Subscription):**
```json
{
  "status": "active",
  "subscription_id": "sub_1QhEBSFhH6dwUiIH...",
  "current_period_end": "2025-12-21T10:00:00Z",
  "product_id": "prod_RZaVDAN6Uf4Qfb"
}
```

**Response (No Subscription):**
```json
{
  "status": "none",
  "subscription_id": "",
  "current_period_end": "0001-01-01T00:00:00Z",
  "product_id": ""
}
```

**Example:**
```bash
curl -H "X-API-Key: YOUR_API_KEY" \
  http://localhost:9000/api/v1/subscriptions/user_123/prod_RZaVDAN6Uf4Qfb
```

---

### 5. Customer Portal
**Endpoint:** `POST /api/v1/portal`

**Request Body:**
```json
{
  "user_id": "user_123",
  "return_url": "https://yourapp.com/account"
}
```

**Response:**
```json
{
  "portal_url": "https://billing.stripe.com/p/session/..."
}
```

**Example:**
```bash
curl -X POST http://localhost:9000/api/v1/portal \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "return_url": "https://yourapp.com/account"
  }'
```

---

## Public Endpoints (No Auth Required)

### 6. Health Check
**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-11-21T10:00:00Z",
  "service": "billing-service"
}
```

**Example:**
```bash
curl http://localhost:9000/health
```

---

### 7. Stripe Webhooks
**Endpoint:** `POST /webhooks/stripe`

**Authentication:** Stripe signature verification (not API key)

This endpoint is called by Stripe, not your application.

---

## Common Fields Explained

### Required in All Checkout Requests:
- `user_id`: Your internal user identifier
- `email`: Customer's email address
- `success_url`: Where to redirect after successful payment
- `cancel_url`: Where to redirect if user cancels

### Stripe Identifiers:
- `product_id`: Stripe Product ID (e.g., `prod_RZaVDAN6Uf4Qfb`)
- `price_id`: Stripe Price ID (e.g., `price_1QhEBSFhH6dwUiIHSUnHP957`)

**Where to find these:**
- Stripe Dashboard → Products → Click product → Copy IDs
- Or use the constants in `internal/config/constants.go`

---

## Error Responses

All endpoints return errors in this format:

```json
{
  "error": {
    "type": "validation_error",
    "code": "INVALID_REQUEST",
    "message": "user_id is required",
    "description": "The request is missing required fields."
  },
  "meta": {
    "request_id": "",
    "timestamp": "2025-11-21T10:00:00Z"
  }
}
```

**Common Error Codes:**
- `401 Unauthorized`: Missing or invalid API key
- `400 Bad Request`: Invalid request body or missing fields
- `500 Internal Server Error`: Server-side error

---

## Testing

Use the test script to verify API key authentication:
```bash
./scripts/test-api-key.sh
```

Or test individual endpoints:
```bash
# Test subscription checkout
curl -X POST http://localhost:9000/api/v1/checkout/subscription \
  -H "X-API-Key: $(grep '^API_KEY=' .env | cut -d '=' -f2)" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
    "email": "test@example.com",
    "product_id": "prod_RZaVDAN6Uf4Qfb",
    "price_id": "price_1QhEBSFhH6dwUiIHSUnHP957",
    "success_url": "http://localhost:3000/success",
    "cancel_url": "http://localhost:3000/cancel"
  }'
```
