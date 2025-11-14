# Styx Billing Microservice (HTTP-Only)

A comprehensive HTTP-only Go microservice for handling Stripe subscription billing with **Neon DB**, REST API, and webhook event processing.

![Go](https://img.shields.io/badge/Go-1.22-blue)
![License](https://img.shields.io/badge/License-MIT-green)
![Status](https://img.shields.io/badge/Status-Production%20Ready-brightgreen)

## 🚀 Features

### Core Functionality

- **🔒 Secure Stripe Integration** - Real Stripe API integration with environment-based configuration
- **💳 Subscription Management** - Create checkout sessions, manage subscriptions, customer portals
- **🌐 HTTP REST API** - Universal compatibility with any application via HTTP
- **🔄 Webhook Processing** - Handle Stripe webhook events with context-aware processing
- **🗄️ Neon DB Integration** - PostgreSQL with pgx for subscription and customer data persistence
- **⚙️ Environment Configuration** - Full environment variable-based configuration

### Infrastructure

- **🐳 Docker Support** - Complete containerization with docker-compose
- **🏥 Health Monitoring** - Health check endpoints and graceful shutdown
- **📝 Comprehensive Logging** - Structured logging with configurable levels
- **🧪 Test Suite** - Both mock and real database integration tests
- **📚 API Documentation** - OpenAPI/Swagger specification included

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Clients  │────│   HTTP Server   │────│   Neon DB       │
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

### 2. Database Setup

**Neon DB (Recommended):**
The database tables are automatically created when the service starts. For manual setup, run:

```bash
# Using the seed tool
cd cmd/seed && go run main.go

# Or manually via psql
psql "$DATABASE_URL" -f scripts/create_tables.sql
```

### 3. Run the Service

```bash
# Using Go directly
go run cmd/server/main.go

# Using Docker Compose (recommended)
docker-compose up --build
```

## 📚 API Documentation

### HTTP REST API

The service provides the following HTTP endpoints:

#### Health Check

```http
GET /health
```

**Response:**

```json
{
  "status": "healthy",
  "timestamp": "2024-11-14T04:00:00Z",
  "service": "billing-service"
}
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

**Response:**

```json
{
  "checkout_session_id": "cs_test_1234567890",
  "checkout_url": "https://checkout.stripe.com/pay/cs_test_1234567890"
}
```

#### Get Subscription Status

```http
GET /api/v1/subscriptions/{user_id}/{product_id}
```

**Response (Subscription Exists):**

```json
{
  "subscription_id": "sub_1234567890",
  "status": "active",
  "customer_id": "cus_1234567890",
  "current_period_end": "2024-12-31T23:59:59Z",
  "exists": true
}
```

**Response (No Subscription):**

```json
{
  "exists": false
}
```

#### Create Customer Portal

```http
POST /api/v1/portal
Content-Type: application/json
```

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
  "portal_session_id": "ps_test_1234567890",
  "portal_url": "https://billing.stripe.com/p/session/ps_test_1234567890"
}
```

#### Webhook Processing

```http
POST /webhooks/stripe
Content-Type: application/json
Stripe-Signature: <webhook-signature>
```

## 🧪 Testing

### Test Structure

The project includes comprehensive testing:

#### Mock Repository Tests (Fast)

```bash
# Run tests without database
go test ./cmd/server/ -v
```

#### Database Integration Tests

```bash
# Run tests with real database (requires environment variables)
DATABASE_URL="postgresql://..." STRIPE_SECRET_KEY="..." \
  go test ./cmd/server/ -run "TestWithDB" -v
```

### Adding Test Data

Use the seed tool to add test users and subscriptions:

```bash
cd cmd/seed && go run main.go
```

### Testing with cURL

#### Health Check

```bash
curl http://localhost:8080/health
```

#### Create Checkout Session

```bash
curl -X POST http://localhost:8080/api/v1/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "email": "user@example.com",
    "product_id": "premium_plan",
    "price_id": "price_1234567890",
    "success_url": "https://yourapp.com/success?session_id={CHECKOUT_SESSION_ID}",
    "cancel_url": "https://yourapp.com/cancel"
  }'
```

#### Get Subscription Status

```bash
curl http://localhost:8080/api/v1/subscriptions/user_123/premium_plan
```

#### Test Stripe Webhook

```bash
curl -X POST http://localhost:8080/webhooks/stripe \
  -H "Content-Type: application/json" \
  -H "Stripe-Signature: test" \
  -d '{"type": "invoice.payment_succeeded"}'
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

### Database Configuration

The service automatically configures connection pooling:

- **Min Connections:** 5
- **Max Connections:** 25
- **Connection Lifetime:** 1 hour
- **Idle Timeout:** 30 minutes
- **Health Check:** 5 minutes

## 🐳 Docker Deployment

### Using Docker Compose

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

### Manual Docker Build

```bash
# Build image
docker build -t styx-billing .

# Run container
docker run -p 8080:8080 \
  --env-file .env \
  styx-billing
```

## 🔗 Integration Examples

### JavaScript/Node.js

```javascript
// Create checkout session
const response = await fetch("http://localhost:8080/api/v1/checkout", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    user_id: "user_123",
    email: "user@example.com",
    product_id: "premium_plan",
    price_id: "price_1234567890",
    success_url: "https://yourapp.com/success?session_id={CHECKOUT_SESSION_ID}",
    cancel_url: "https://yourapp.com/cancel",
  }),
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

### cURL Examples

#### Health Check

```bash
curl http://localhost:8080/health
```

#### Create Portal Session

```bash
curl -X POST http://localhost:8080/api/v1/portal \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user_123", "return_url": "https://yourapp.com/account"}'
```

## 📊 Monitoring

### Health Checks

The service provides multiple health check endpoints:

- **HTTP:** `GET /health`
- **Database:** Connection and query health
- **Stripe API:** API connectivity validation
- **System:** Memory and resource usage

### Logging

Structured logging with configurable levels:

```bash
# Set log level via environment
LOG_LEVEL=debug go run cmd/server/main.go
```

Log levels:

- `debug` - Detailed debugging information
- `info` - General operational messages
- `warn` - Warning conditions
- `error` - Error conditions

## 🛡️ Security

- **Environment-based secrets** - No hardcoded credentials
- **Stripe signature verification** - Webhook authenticity validation
- **Database connection security** - SSL/TLS connection support
- **Graceful shutdown** - Proper connection cleanup
- **Input validation** - Request parameter validation

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

### Adding New Features

1. **HTTP Endpoints:** Add to `internal/server/http.go`
2. **Database Operations:** Extend `internal/database/repo.go`
3. **Configuration:** Update `internal/config/config.go`
4. **Tests:** Add to `cmd/server/main_test.go`

### Code Quality

```bash
# Run all tests
go test ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage
go tool cover -html=coverage.out

# Format code
go fmt ./...

# Run linter
go vet ./...
```

## 📈 Performance

- **HTTP Performance:** Optimized for high-throughput scenarios with proper timeouts
- **Database Connection Pooling:** Efficient connection management
- **Concurrent Request Handling:** Goroutine-based processing
- **Graceful Shutdown:** Proper resource cleanup

## 🔍 Troubleshooting

### Common Issues

#### Database Connection Failed

```bash
# Check DATABASE_URL format
echo $DATABASE_URL

# Test connection with Neon DB
psql "$DATABASE_URL" -c "SELECT 1;"
```

#### Environment Variables Not Loaded

```bash
# Verify .env file exists
ls -la .env

# Load environment manually
source .env && echo $DATABASE_URL
```

#### Stripe API Errors

- Verify `STRIPE_SECRET_KEY` is valid test/production key
- Check network connectivity to Stripe API
- Review application logs for specific error messages

### Debug Mode

```bash
# Enable debug logging
LOG_LEVEL=debug go run cmd/server/main.go

# Run tests with verbose output
go test ./cmd/server/ -v -count=1

# Use debug endpoint (development only)
curl http://localhost:8080/debug
```

## 📝 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Run the test suite
5. Submit a pull request

## 📞 Support

For issues and questions:

- Create an issue on GitHub
- Check existing documentation
- Review test examples for usage patterns

---

**Production Ready:** This HTTP-only microservice has been designed for universal compatibility and tested with both mock and real database scenarios, includes comprehensive error handling, and follows Go best practices for production deployment with Neon DB.
