package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
)

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(envFilePath string) error {
	file, err := os.Open(envFilePath)
	if err != nil {
		return fmt.Errorf("failed to open env file %s: %w", envFilePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Split on first = only
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes if present
		if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
		   (strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
			value = value[1 : len(value)-1]
		}
		
		// Only set if not already set (preserve existing environment)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading env file: %w", err)
	}
	
	return nil
}

// autoLoadTestEnv automatically loads environment from .env files for tests
func autoLoadTestEnv() {
	// Try to load .env file from current directory first
	envFiles := []string{
		".env",
		".env.local",
		"../.env",
		"../../.env",
	}
	
	for _, envFile := range envFiles {
		if _, err := os.Stat(envFile); err == nil {
			if err := loadEnvFile(envFile); err == nil {
				fmt.Printf("Loaded environment from: %s\n", envFile)
				break
			}
		}
	}
}

// withRealEnv runs a test with real environment variables automatically loaded
func withRealEnv(t *testing.T, testFunc func(*testing.T)) {
	t.Helper()
	
	// Automatically load real environment variables
	autoLoadTestEnv()
	
	// Run the test
	testFunc(t)
}
