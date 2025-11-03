# Styx Billing Microservice

A comprehensive Go microservice for handling Stripe subscription billing with **Neon DB**, gRPC API, and webhook event processing.

![Go](https://img.shields.io/badge/Go-1.22-blue)
![License](https://img.shields.io/badge/License-MIT-green)
![Status](https://img.shields.io/badge/Status-Production%20Ready-brightgreen)

## 🚀 Features

### Core Functionality
- **🔒 Secure Stripe Integration** - Real Stripe API integration with environment-based configuration
- **💳 Subscription Management** - Create checkout sessions, manage subscriptions, customer portals
- **📡 gRPC API** - High-performance gRPC service for billing operations
- **🔄 Webhook Processing** - Handle Stripe webhook events with context-aware processing
- **🗄️ Neon DB Integration** - PostgreSQL with pgx for subscription and customer data persistence
- **⚙️ Environment Configuration** - Full environment variable-based configuration

### Infrastructure
- **🐳 Docker Support** - Complete containerization with docker-compose
- **🏥 Health Monitoring** - Health check endpoints and graceful shutdown
- **📝 Comprehensive Logging** - Structured logging with configurable levels
- **🧪 Test Suite** - Both mock and real database integration tests

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   gRPC Clients  │────│   gRPC Server   │────│  PostgreSQL DB  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │ Stripe Webhooks │
                       │    HTTP API     │
                       └─────────────────┘
```

## 📋 Prerequisites

- **Go 1.23+**
- **PostgreSQL 14+** (or cloud database)
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
DATABASE_URL=postgresql://username:password@host:port/database
STRIPE_SECRET_KEY=sk_test_your_real_stripe_secret_key
STRIPE_WEBHOOK_SECRET=whsec_your_real_webhook_secret
GRPC_PORT=9090
HTTP_PORT=8080
LOG_LEVEL=info
CERBERUS_GRPC_DIAL_ADDRESS=cerberus-service:50051
```

### 2. Database Setup

```bash
# Initialize database schema
psql $DATABASE_URL -f init.sql
```

### 3. Run the Service

```bash
# Using Go directly
go run cmd/server/main.go

# Using Docker Compose (recommended)
docker-compose up --build
```

## 📚 API Documentation

### gRPC Service

The service implements the `BillingService` with the following methods:

#### CreateSubscriptionCheckout
Creates a Stripe Checkout Session for subscription purchases.

**Request:**
```proto
message CreateSubscriptionCheckoutRequest {
  string user_id = 1;
  string product_id = 2;
  string price_id = 3;
  string success_url = 4;
  string cancel_url = 5;
}
```

**Response:**
```proto
message CreateSubscriptionCheckoutResponse {
  string checkout_session_id = 1;
  string checkout_url = 2;
}
```

#### GetSubscriptionStatus
Retrieves the subscription status for a user/product combination.

**Request:**
```proto
message GetSubscriptionStatusRequest {
  string user_id = 1;
  string product_id = 2;
}
```

**Response:**
```proto
message GetSubscriptionStatusResponse {
  bool exists = 1;
  string status = 2;
  int64 current_period_end = 3;
}
```

#### CreateCustomerPortal
Creates a Stripe Customer Portal session for account management.

**Request:**
```proto
message CreateCustomerPortalRequest {
  string user_id = 1;
  string return_url = 2;
}
```

**Response:**
```proto
message CreateCustomerPortalResponse {
  string portal_session_id = 1;
  string portal_url = 2;
}
```

### HTTP Endpoints

#### Webhook Processing
```http
POST /webhooks/stripe
Content-Type: application/json
Stripe-Signature: <webhook-signature>
```

#### Health Check
```http
GET /health
```

## 🧪 Testing

### Test Structure

The project includes a comprehensive test suite with two testing approaches:

#### Mock Repository Tests (Fast)
Tests that use mock repositories and don't require database connection:
```bash
# Run only mock tests (no environment variables needed)
go test ./cmd/server/ -run "TestBillingService/CreateSubscriptionCheckout" -v
go test ./cmd/server/ -run "TestDatabaseIntegration" -v
go test ./cmd/server/ -run "TestWebhookHandler" -v
```

#### Real Database Tests (Integration)
Tests that use actual PostgreSQL database:
```bash
# Run tests with real database (requires environment variables)
DATABASE_URL="postgresql://..." STRIPE_SECRET_KEY="..." \
  go test ./cmd/server/ -run "TestBillingServiceWithRealDB" -v

# Run configuration tests
DATABASE_URL="..." STRIPE_SECRET_KEY="..." \
  go test ./cmd/server/ -run "TestConfiguration" -v
```

### Running All Tests

```bash
# Mock tests (recommended for development)
go test ./cmd/server/ -v

# With real database integration
source .env && go test ./cmd/server/ -v
```

### Test Categories

1. **TestBillingService** - Core billing functionality with mock repo
2. **TestBillingServiceWithRealDB** - Real database integration testing
3. **TestConfiguration** - Environment configuration validation
4. **TestDatabaseIntegration** - Database operations with mock repo
5. **TestWebhookHandler** - Webhook processing functionality
6. **TestServerOrchestration** - Server component integration
7. **TestErrorHandling** - Error scenario testing
8. **BenchmarkCreateSubscriptionCheckout** - Performance testing

## 🔧 Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | - | PostgreSQL connection string |
| `STRIPE_SECRET_KEY` | ✅ | - | Stripe secret API key |
| `STRIPE_WEBHOOK_SECRET` | ✅ | - | Stripe webhook signing secret |
| `GRPC_PORT` | ❌ | `50051` | gRPC server port |
| `HTTP_PORT` | ❌ | `8080` | HTTP server port |
| `LOG_LEVEL` | ❌ | `info` | Logging level (debug, info, warn, error) |
| `CERBERUS_GRPC_DIAL_ADDRESS` | ✅ | - | External service address |

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
# Build and start all services
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
docker run -p 8080:8080 -p 9090:9090 \
  --env-file .env \
  styx-billing
```

## 🔗 Integration Examples

### gRPC Client Example (Go)

```go
import (
    "context"
    "google.golang.org/grpc"
    billing "styx/proto/billing_service/proto/billing"
)

func main() {
    conn, _ := grpc.Dial("localhost:9090", grpc.WithInsecure())
    defer conn.Close()
    
    client := billing.NewBillingServiceClient(conn)
    
    resp, _ := client.CreateSubscriptionCheckout(context.Background(), &billing.CreateSubscriptionCheckoutRequest{
        UserId:     "user123",
        ProductId:  "prod_premium",
        PriceId:    "price_123",
        SuccessUrl: "https://example.com/success",
        CancelUrl:  "https://example.com/cancel",
    })
    
    fmt.Printf("Checkout URL: %s\n", resp.CheckoutUrl)
}
```

### cURL Examples

#### Health Check
```bash
curl http://localhost:8080/health
```

#### Test Stripe Webhook
```bash
curl -X POST http://localhost:8080/webhooks/stripe \
  -H "Content-Type: application/json" \
  -H "Stripe-Signature: test" \
  -d '{"type": "invoice.payment_succeeded"}'
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
├── cmd/server/           # Main server application
├── internal/
│   ├── config/           # Configuration management
│   ├── database/         # Database models and repository
│   ├── server/           # gRPC service implementation
│   └── webhooks/         # Stripe webhook handling
├── proto/                # Protocol buffer definitions
├── docker-compose.yml    # Docker orchestration
└── init.sql             # Database schema
```

### Adding New Features

1. **gRPC Methods:** Add to `proto/billing.proto` and regenerate
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

- **gRPC Performance:** Optimized for high-throughput scenarios
- **Database Connection Pooling:** Efficient connection management
- **Concurrent Request Handling:** Goroutine-based processing
- **Graceful Shutdown:** Proper resource cleanup

## 🔍 Troubleshooting

### Common Issues

#### Database Connection Failed
```bash
# Check DATABASE_URL format
echo $DATABASE_URL

# Test connection
psql $DATABASE_URL -c "SELECT 1;"
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

**Production Ready:** This microservice has been tested with both mock and real database scenarios, includes comprehensive error handling, and follows Go best practices for production deployment.
