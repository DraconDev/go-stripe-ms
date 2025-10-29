# Go Stripe Payment Microservice

A simple Go microservice for handling Stripe payments using Payment Intents.

## Features

- Create Payment Intents
- Health check endpoint
- Docker support
- Environment-based configuration

## Prerequisites

- Go 1.21
- Docker (optional, for containerized deployment)
- Stripe account and secret key

## Setup

1. Clone the repository
2. Copy the environment file and set your Stripe secret key:
   ```bash
   cp .env.example .env
   # Edit .env and replace with your actual Stripe secret key
   ```
3. Run the service:
   ```bash
   go run main.go
   ```

   Or with Docker Compose:
   ```bash
   docker-compose up --build
   ```

## Docker

Build and run with Docker Compose:

```bash
docker-compose up --build
```

## API Endpoints

### POST /create-payment-intent

Creates a new Stripe Payment Intent for live payments.

**Request Body:**
```json
{
  "amount": 1000,
  "currency": "usd"
}
```

**Response:**
```json
{
  "client_secret": "pi_..."
}
```

### POST /create-test-payment-intent

Creates a new Stripe Payment Intent for testing (works with test keys only).

**Request Body:**
```json
{
  "amount": 1000,
  "currency": "usd"
}
```

**Response:**
```json
{
  "client_secret": "pi_..."
}
```

### GET /health

Health check endpoint.

**Response:** `OK`

## Testing

### Unit Tests
Run the unit tests:
```bash
go test -v
```

### Integration Tests
1. Start the service:
   ```bash
   export STRIPE_SECRET_KEY=sk_test_...
   go run main.go
   ```

2. Run the test script in another terminal:
   ```bash
   ./test_client.sh
   ```

### Manual Testing
Test the endpoints manually:

Health check:
```bash
curl http://localhost:8080/health
```

Create payment intent:
```bash
curl -X POST http://localhost:8080/create-payment-intent \
  -H "Content-Type: application/json" \
  -d '{"amount": 1000, "currency": "usd"}'
```

## Usage

This microservice can be integrated into larger applications to handle payment processing securely on the server side.
