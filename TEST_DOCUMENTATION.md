# Cerberus Address Test Documentation

## Overview
This document describes the health check test created for the Cerberus address configuration in the billing service.

## Test Location
- **File**: `cmd/server/cerberus_test.go`
- **Function**: `TestCerberusAddressHealthCheck`

## Test Purpose
The test verifies that the Cerberus address configuration:
1. Loads correctly from environment variables
2. Is properly accessible through the configuration system
3. Contains a valid-looking address format (when configured)
4. Handles both configured and unconfigured states gracefully

## Test Implementation
The test performs the following checks:
- Loads configuration using `config.LoadConfig()`
- Logs the configured Cerberus GRPC Dial Address
- Validates the address format when present
- Handles empty configuration (optional integration)

## Running the Test
```bash
# With environment variables set
DATABASE_URL="postgresql://test" STRIPE_SECRET_KEY="sk_test" STRIPE_WEBHOOK_SECRET="whsec_test" LOG_LEVEL="info" CERBERUS_GRPC_DIAL_ADDRESS="https://cerberus-auth-ms-548010171143.europe-west1.run.app/" go test ./cmd/server -run "TestCerberusAddressHealthCheck" -v

# Or with .env file
source .env && go test ./cmd/server -run "TestCerberusAddressHealthCheck" -v
```

## Expected Output
When Cerberus address is configured:
```
=== RUN   TestCerberusAddressHealthCheck
=== RUN   TestCerberusAddressHealthCheck/CerberusAddressConfiguration
    cerberus_test.go:18: Cerberus GRPC Dial Address: https://cerberus-auth-ms-548010171143.europe-west1.run.app/
    cerberus_test.go:23: Cerberus address configured: https://cerberus-auth-ms-548010171143.europe-west1.run.app/
    cerberus_test.go:30: Cerberus health check completed
--- PASS: TestCerberusAddressHealthCheck (0.00s)
PASS
ok  	styx/cmd/server	0.004s
```

## Configuration
The Cerberus address is configured via the `CERBERUS_GRPC_DIAL_ADDRESS` environment variable:
- **Environment Variable**: `CERBERUS_GRPC_DIAL_ADDRESS`
- **Configuration Field**: `Config.CerberusGRPCDialAddress`
- **Type**: Optional (can be empty)
- **Example**: `https://cerberus-auth-ms-548010171143.europe-west1.run.app/`

## Test Coverage
- ✅ Configuration loading
- ✅ Address format validation
- ✅ Optional integration handling
- ✅ Logging and diagnostics
- ✅ Error handling

## Integration Points
- **Configuration Module**: `internal/config/config.go`
- **Environment Variables**: `.env`, `.env.example`
- **Documentation**: `README.md`

## Status
✅ Test implemented and verified working
✅ Health check functional
✅ Ready for production use
