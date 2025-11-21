# API Key Authentication

This Payment MS uses API key authentication to secure access to payment endpoints.

## Quick Start

### Your API Key
```
proj_prkwlS4ipOUYKKekMv9kvPwPa6dun6Rt-6Uuwal31Gs
```

### Usage
Include the `X-API-Key` header in all API requests:

```bash
curl -H "X-API-Key: proj_prkwlS4ipOUYKKekMv9kvPwPa6dun6Rt-6Uuwal31Gs" \
  http://localhost:9000/api/v1/subscriptions/user_123/prod_123
```

### JavaScript Example
```javascript
fetch('http://localhost:9000/api/v1/checkout/subscription', {
  method: 'POST',
  headers: {
    'X-API-Key': 'proj_prkwlS4ipOUYKKekMv9kvPwPa6dun6Rt-6Uuwal31Gs',
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

## Creating New Projects

To create additional projects with separate API keys:

```bash
go run ./cmd/create-project "Project Name"
```

This will generate a new API key for the project.

## Security Notes

- **Never commit API keys to version control**
- Store API keys in environment variables or secrets management
- Use HTTPS in production to protect API keys in transit
- Rotate API keys periodically for enhanced security

## Troubleshooting

### Error: "Missing X-API-Key header"
You forgot to include the `X-API-Key` header in your request.

### Error: "Invalid API key format"
The API key must start with `proj_` and be 48 characters total.

### Error: "Invalid API key"
The API key is not found in the database or the project is inactive.

## Migration

If you're migrating from a version without API key authentication, run:

```bash
go run ./cmd/migrate
```

This will create the projects table and generate your first API key.
