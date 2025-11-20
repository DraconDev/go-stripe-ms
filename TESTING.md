# Real Database Testing Guide

**Document Type**: Technical Documentation - Testing  
**Purpose**: Guide for implementing real database integration testing alongside mock testing

## Overview

This project now supports testing with the real PostgreSQL database instead of mocks. This provides more realistic testing that catches integration issues and works with your actual database schema.

You have two testing approaches:

1. **Mock Database Testing** (existing) - Fast unit tests using in-memory mocks
2. **Real Database Testing** (new) - Integration tests using your actual PostgreSQL database

## Setup

### Environment Configuration

The real database tests use your existing Neon `DATABASE_URL`. Make sure your `.env` file has the connection string:

```bash
# Database Configuration
DATABASE_URL=postgresql://neondb_owner:npg_4zngha5HGpNK@ep-dark-queen-adrvy2wc-pooler.c-2.us-east-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require
```

The individual `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` variables are not needed since the full connection info is in the `DATABASE_URL`.

## Test Files

### Mock Tests (Existing)
- `internal/server/http_test.go` - Unit tests using mocks
- These are fast but don't test actual database operations

### Integration Tests (New)  
- `internal/server/tests/` - Real database integration tests organized by functionality
- `internal/database/test_database.go` - Database testing utilities

## Running Tests

### Run All Tests
```bash
go test -v ./...
```

### Run Only Integration Tests
```bash
go test -v -run Integration ./...
```

### Run Only Mock Tests
```bash
go test -v ./internal/server/
```

### Run Database Operations Only
```bash
go test -v -run "TestDatabaseOperationsIntegration" ./...
```

## Using Real Database in Your Tests

### Basic Pattern

```go
package yourpackage

import (
    "testing"
    "styx/internal/database"
)

func TestYourFeatureWithRealDB(t *testing.T) {
    database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
        // Your test code here
        // testDB.Repo is a *database.Repository connected to real DB
        // testDB.Conn is a *pgx.Conn for direct queries if needed
        // Use testDB.CreateTestCustomer() and testDB.CreateTestSubscription() for test data
    })
}
```

### Creating Test Data

```go
// Create a test customer
customer := &database.Customer{
    UserID:           "test_user_123",
    Email:            "test@example.com",
    StripeCustomerID: "cus_test123",
    CreatedAt:        time.Now(),
    UpdatedAt:        time.Now(),
}

if err := testDB.CreateTestCustomer(customer); err != nil {
    t.Fatalf("Failed to create test customer: %v", err)
}

// Create a test subscription
subscription := &database.Subscription{
    UserID:               "test_user_123",
    ProductID:            "premium_plan",
    PriceID:              "price_test123",
    StripeSubscriptionID: "sub_test123",
    Status:               "active",
    CurrentPeriodStart:   time.Now(),
    CurrentPeriodEnd:     time.Now().AddDate(0, 0, 30),
    CreatedAt:            time.Now(),
    UpdatedAt:            time.Now(),
}

if err := testDB.CreateTestSubscription(subscription); err != nil {
    t.Fatalf("Failed to create test subscription: %v", err)
}
```

### Testing HTTP Endpoints

```go
func TestYourEndpointIntegration(t *testing.T) {
    database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
        // Setup
        if err := testDB.CreateTestData(); err != nil {
            t.Fatalf("Failed to create test data: %v", err)
        }

        // Create HTTP server with real database
        server := NewHTTPServer(testDB.Repo, "sk_test_your_key")

        // Create request
        req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoint", nil)
        
        // Execute
        w := httptest.NewRecorder()
        server.YourEndpoint(w, req)

        // Assert
        if w.Code != http.StatusOK {
            t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
        }
        
        // Parse response
        var response YourResponseType
        if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
            t.Fatalf("Failed to unmarshal response: %v", err)
        }
    })
}
```

## Testing Utilities

### database.WithTestDatabase

This is the main function for running tests with a real database:

```go
database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
    // Test code here
})
```

### database.TestDatabase Methods

- `testDB.Repo` - Database repository for testing business logic
- `testDB.Conn` - Direct database connection for custom queries
- `testDB.CreateTestCustomer(customer)` - Creates a test customer
- `testDB.CreateTestSubscription(subscription)` - Creates a test subscription
- `testDB.CreateTestData()` - Creates standard test data

### Test Skipping

Tests automatically skip if:
- `DATABASE_URL` environment variable is not set
- Database connection fails

This means your CI/CD can run mock tests by default and integration tests when the database is available.

## Migration from Mock to Real Database

### Step 1: Identify Tests to Migrate

Look for tests that:
- Test actual database operations
- Need to verify data persistence
- Test complex queries or transactions
- Need integration validation

### Step 2: Create Integration Test

```go
// Example migration from mock to real database test
func TestCreateSubscriptionIntegration(t *testing.T) {
    database.WithTestDatabase(t, func(t *testing.T, testDB *database.TestDatabase) {
        // Setup test data
        // Instead of mockDB.AddTestCustomer()
        customer := &database.Customer{
            UserID: "test_user",
            Email:  "test@example.com",
        }
        if err := testDB.CreateTestCustomer(customer); err != nil {
            t.Fatalf("Setup failed: %v", err)
        }

        // Test with real repository
        result, err := testDB.Repo.FindOrCreateStripeCustomer(ctx, "test_user", "test@example.com")
        // Your assertions here
    })
}
```

### Step 3: Keep Mock Tests for Speed

Mock tests are still valuable for:
- Fast feedback during development
- Testing edge cases
- Testing error conditions
- Unit testing individual functions

Use both approaches:
- Mock tests for development and quick feedback
- Integration tests for CI/CD and validation

## Best Practices

### 1. Test Isolation
Each test runs in isolation with its own database connection. Tests don't interfere with each other.

### 2. Test Data Management
- Use unique identifiers for test data
- Clean up after tests (handled automatically by the test framework)
- Don't rely on existing data in the database

### 3. Performance
- Real database tests are slower than mocks
- Use them for critical paths and integration validation
- Keep unit tests as mocks for development speed

### 4. CI/CD Integration
```bash
# In your CI pipeline
go test -v -short ./...                    # Fast unit tests
go test -v -run Integration ./...          # Integration tests with real DB
```

## Examples in the Codebase

### HTTP Integration Tests
See `internal/server/tests/` directory for organized test examples:
- `billing_api_test.go` - Customer portal and billing operations
- `subscription_checkout_test.go` - Subscription checkout workflows  
- `subscription_status_test.go` - Subscription status management
- `database_operations_test.go` - Direct database operations

### Test Organization Benefits
- Each test file focuses on specific functionality
- Tests use real database connections with proper cleanup
- Both mock and integration testing patterns available
- Clear separation between unit and integration testing

## Troubleshooting

### Database Connection Issues
- Check your `.env` file has correct `DATABASE_URL`
- Ensure your database is accessible
- Check network connectivity

### Test Timeouts
- Integration tests with real database are slower
- Adjust timeout: `go test -timeout 60s ./...`

### Test Data Conflicts
- Tests use unique identifiers to avoid conflicts
- If you see conflicts, check for hardcoded IDs in tests

## Next Steps

1. **Start with Integration Tests**: Run the existing integration tests to verify they work with your database
2. **Gradual Migration**: Convert mock tests to integration tests gradually for critical features
3. **CI/CD Integration**: Add integration tests to your CI pipeline
4. **Performance Monitoring**: Monitor test execution times and optimize as needed

The real database testing approach gives you more confidence that your application works correctly in production environments.
