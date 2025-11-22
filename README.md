# Styx Billing Microservice (HTTP-Only) !

A comprehensive **HTTP-only** Go microservice for handling Stripe subscription billing with **Neon DB**, REST API, and webhook event processing. Built for universal compatibility.

![Go](https://img.shields.io/badge/Go-1.22-blue)
![License](https://img.shields.io/badge/License-MIT-green)
![Status](https://img.shields.io/badge/Status-Production%20Ready-brightgreen)

## ✅ Architecture: HTTP-Only Design

This service is built with a pure HTTP REST API architecture, providing universal compatibility with any application while maintaining all functionality.

## 🚀 Features

- **🔒 Secure Stripe Integration** - Real Stripe API integration with environment-based configuration
- **💳 Subscription Management** - Create checkout sessions, manage subscriptions, customer portals
- **🌐 HTTP REST API** - Universal compatibility with any application via HTTP
- **🔄 Webhook Processing** - Handle Stripe webhook events with context-aware processing
- **🗄️ Neon DB Integration** - PostgreSQL with pgx for subscription and customer data persistence
- **🐳 Docker Support** - Complete containerization with docker-compose
- **🏥 Health Monitoring** - Service health check endpoint
- **📝 Comprehensive Logging** - Structured logging with configurable levels
- **📚 OpenAPI Documentation** - Complete API specification

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Clients  │────│   HTTP Server   │────│   Neon DB       │
│  (Any Language) │    │   (Go/Gin)      │    │   PostgreSQL    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                         ┌─────────────────┐
                         │ Stripe Webhooks │
                         │    HTTP API     │
                         └─────────────────┘
```

## 📋 Prerequisites

- **Go 1.22+**
- **Neon DB** (PostgreSQL) - Cloud-based PostgreSQL database
- **Stripe Account** with API keys
- **Docker & Docker Compose** (optional)

## 🚀 Quick Start

### 1. Environment Setup

```bash
# Copy environment template
cp .env.example .env

# Edit .env with your actual values
nano .env
```

Required environment variables:

```bash
# Neon DB Configuration
DATABASE_URL=postgresql://neondb_owner:your_password@ep-your-db.neon.tech/neondb?sslmode=require

# Stripe Configuration
STRIPE_SECRET_KEY=sk_test_your_real_stripe_secret_key
STRIPE_WEBHOOK_SECRET=whsec_your_real_webhook_secret

# Server Configuration
HTTP_PORT=8080
LOG_LEVEL=info
```

### 2. Run the Service

```bash
# Using Go directly
go run cmd/server/main.go

# Using Docker Compose
docker-compose up --build
```

The database tables are automatically created when the service starts.

## 📚 API Documentation

### HTTP REST API

#### Health Check

```http
GET /health
```

#### Create Subscription Checkout

```http
POST /api/v1/checkout
Content-Type: application/json
```

**Request Body:**
```json
{
  "user_id": "user_123",
  "email": "user@example.com",
  "product_id": "premium_plan",
  "price_id": "price_1234567890",
  "success_url": "https://yourapp.com/success?session_id={CHECKOUT_SESSION_ID}",
  "cancel_url": "https://yourapp.com/cancel"
}
```

#### Get Subscription Status

```http
GET /api/v1/subscriptions/{user_id}/{product_id}
```

#### Create Customer Portal

```http
POST /api/v1/portal
Content-Type: application/json
```

#### Webhook Processing

```http
POST /webhooks/stripe
Content-Type: application/json
Stripe-Signature: <webhook-signature>
```

## 🧪 Testing

### Run Tests

```bash
# Test all endpoints
go test ./...

# Integration tests with real database
DATABASE_URL="postgresql://..." STRIPE_SECRET_KEY="..." go test -v ./...
```

### Test Data

```bash
# Add test users and subscriptions
cd cmd/seed && go run main.go
```

### Quick Test

```bash
# Health check
curl http://localhost:8080/health

# Create checkout session
curl -X POST http://localhost:8080/api/v1/checkout \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user_123","email":"test@example.com","product_id":"premium_plan","price_id":"price_123","success_url":"https://example.com/success","cancel_url":"https://example.com/cancel"}'
```

## 🔧 Configuration

### Environment Variables

| Variable                | Required | Default | Description                              |
| ----------------------- | -------- | ------- | ---------------------------------------- |
| `DATABASE_URL`          | ✅       | -       | Neon DB PostgreSQL connection string     |
| `STRIPE_SECRET_KEY`     | ✅       | -       | Stripe secret API key                    |
| `STRIPE_WEBHOOK_SECRET` | ✅       | -       | Stripe webhook signing secret            |
| `HTTP_PORT`             | ❌       | `8080`  | HTTP server port                         |
| `LOG_LEVEL`             | ❌       | `info`  | Logging level (debug, info, warn, error) |

## 🐳 Docker Deployment

```bash
# Build and start service
docker-compose up --build

# Run in background
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

## 🔗 Integration Examples

### JavaScript/Node.js

```javascript
// Create checkout session
const response = await fetch("http://localhost:8080/api/v1/checkout", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    user_id: "user_123",
    email: "user@example.com",
    product_id: "premium_plan",
    price_id: "price_1234567890",
    success_url: "https://yourapp.com/success?session_id={CHECKOUT_SESSION_ID}",
    cancel_url: "https://yourapp.com/cancel"
  })
});

const { checkout_url } = await response.json();
window.location.href = checkout_url;
```

### Python

```python
import requests

# Get subscription status
response = requests.get('http://localhost:8080/api/v1/subscriptions/user_123/premium_plan')
data = response.json()

if data.get('exists'):
    print(f"Subscription status: {data['status']}")
else:
    print("No active subscription")
```

## 📊 Monitoring

### Health Checks

The service provides health check endpoints:
- **HTTP:** `GET /health`
- **Database:** Connection and query health
- **Stripe API:** API connectivity validation

### Logging

Structured logging with configurable levels:

```bash
LOG_LEVEL=debug go run cmd/server/main.go
```

Log levels: `debug`, `info`, `warn`, `error`

## 🛡️ Security

- **Environment-based secrets** - No hardcoded credentials
- **Stripe signature verification** - Webhook authenticity validation
- **Database connection security** - SSL/TLS connection support
- **Input validation** - Request parameter validation
- **Graceful shutdown** - Proper connection cleanup

## 🤝 Development

### Project Structure

```
styx/
├── cmd/
│   ├── server/          # Main HTTP server application
│   └── seed/           # Database seeding tool
├── internal/
│   ├── config/         # Configuration management
│   ├── database/       # Database models and repository
│   ├── server/         # HTTP service implementation
│   └── webhooks/       # Stripe webhook handling
├── api/                # API documentation (OpenAPI/Swagger)
├── docker-compose.yml  # Docker orchestration
└── .env.example        # Environment template
```

### Code Quality

```bash
# Run all tests
go test ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📖 Documentation

- **OpenAPI Specification:** `api/openapi.yaml`
- **Testing Guide:** `TESTING.md`
- **Environment Template:** `.env.example`

## Production Status

✅ **Production Ready** - The billing microservice is fully functional with:
- Real Stripe API integration
- HTTP REST API with JSON responses
- PostgreSQL database persistence
- Webhook event processing
- Docker deployment support
- Health monitoring and logging
- OpenAPI documentation

## License

MIT License - see LICENSE file for details.
