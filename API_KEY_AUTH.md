# API Key Authentication

This Payment MS uses API key authentication to secure access to payment endpoints. The API key is stored in your `.env` file and validated on every request.

## Quick Start

### 1. Set Your API Key
Add to your `.env` file:
```bash
API_KEY=your_api_key_here
```

### 2. Use in Requests
Include the `X-API-Key` header in all API requests:

```bash
curl -H "X-API-Key: your_api_key_here" \
  http://localhost:9000/api/v1/subscriptions/user_123/prod_123
```

### JavaScript Example
```javascript
fetch('http://localhost:9000/api/v1/checkout/subscription', {
  method: 'POST',
  headers: {
    'X-API-Key': process.env.API_KEY,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    user_id: 'user_123',
    email: 'customer@example.com',
    product_id: 'prod_RZaVDAN6Uf4Qfb',
    price_id: 'price_1QhEBSFhH6dwUiIHSUnHP957',
    success_url: 'https://yourapp.com/success',
    cancel_url: 'https://yourapp.com/cancel'
  })
});
```

## Protected Endpoints

The following endpoints require the `X-API-Key` header:
- `POST /api/v1/checkout/subscription`
- `POST /api/v1/checkout/item`
- `POST /api/v1/checkout/cart`
- `GET /api/v1/subscriptions/{user_id}/{product_id}`
- `POST /api/v1/portal`

## Public Endpoints

These endpoints do NOT require authentication:
- `GET /health` - Health check
- `POST /webhooks/stripe` - Stripe webhooks (authenticated by Stripe signature)

## Testing

Run the automated test script:
```bash
./scripts/test-api-key.sh
```

This will test:
- ✅ Public endpoints work without auth
- ❌ Requests without API key are blocked
- ❌ Requests with wrong API key are blocked
- ✅ Requests with valid API key pass through

## Generating a New API Key

You can generate a secure random API key using:
```bash
openssl rand -base64 32
```

Or use the migration script which generates one automatically:
```bash
go run ./cmd/migrate
```

## Security Notes

- **Never commit API keys to version control** - `.env` is in `.gitignore`
- Store API keys in environment variables or secrets management
- Use HTTPS in production to protect API keys in transit
- Rotate API keys periodically for enhanced security
- Use different API keys for different environments (dev, staging, prod)

## Troubleshooting

### Error: "Missing X-API-Key header"
You forgot to include the `X-API-Key` header in your request.

**Fix:**
```bash
curl -H "X-API-Key: YOUR_KEY" http://localhost:9000/api/...
```

### Error: "Invalid API key"
The API key in your request doesn't match the one in `.env`.

**Fix:**
1. Check your `.env` file for the correct API_KEY value
2. Ensure you're using the same key in your request
3. Restart the server after changing `.env`

### Error: "Required environment variable API_KEY is not set"
The server can't find the API_KEY in your environment.

**Fix:**
1. Add `API_KEY=your_key_here` to your `.env` file
2. Restart the server

## Production Deployment

### Environment Variables
Set the API_KEY in your production environment:

**Heroku:**
```bash
heroku config:set API_KEY=your_production_key
```

**Docker:**
```bash
docker run -e API_KEY=your_production_key ...
```

**AWS/GCP:**
Use their secrets management services (AWS Secrets Manager, GCP Secret Manager)

### Key Rotation
To rotate your API key:
1. Generate a new key
2. Update `.env` or environment variable
3. Restart the server
4. Update all client applications with the new key

