package main

import (
	"testing"

	"styx/internal/config"
)

// TestCerberusAddressHealthCheck tests the Cerberus address configuration health check
// This test automatically loads environment variables from .env files
func TestCerberusAddressHealthCheck(t *testing.T) {
	// Automatically load real environment variables
	withRealEnv(t, func(t *testing.T) {
		t.Run("CerberusAddressConfiguration", func(t *testing.T) {
			cfg, err := config.LoadConfig()
			if err != nil {
				t.Fatalf("Failed to load configuration: %v", err)
			}

			// Check if Cerberus address is configured (it's optional, so empty is valid)
			t.Logf("Cerberus GRPC Dial Address: %s", cfg.CerberusGRPCDialAddress)
			
			if cfg.CerberusGRPCDialAddress == "" {
				t.Logf("Cerberus address not configured (optional integration)")
			} else {
				t.Logf("Cerberus address configured: %s", cfg.CerberusGRPCDialAddress)
				// Basic validation that it looks like a URL or address
				if len(cfg.CerberusGRPCDialAddress) < 5 {
					t.Errorf("Cerberus address appears too short: %s", cfg.CerberusGRPCDialAddress)
				}
			}
			
			t.Logf("Cerberus health check completed with real environment")
		})
	})
}
