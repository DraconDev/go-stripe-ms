# Go Stripe Payment Microservice

A simple Go microservice for handling Stripe payments using Payment Intents.

## Features

- Create Payment Intents
- Health check endpoint
- Docker support
- Environment-based configuration

## Prerequisites

- Go 1.21+
- Stripe account and secret key

## Setup

1. Clone the repository
2. Set your Stripe secret key as an environment variable:
   ```bash
   export STRIPE_SECRET_KEY=sk_test_...
   ```
3. Run the service:
   ```bash
   go run main.go
   ```

## Docker

Build and run with Docker Compose:

```bash
docker-compose up --build
```

## API Endpoints

### POST /create-payment-intent

Creates a new Stripe Payment Intent.

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

## Usage

This microservice can be integrated into larger applications to handle payment processing securely on the server side.
