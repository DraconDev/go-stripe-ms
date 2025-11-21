package database

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// autoLoadEnv automatically loads environment variables from .env files
func autoLoadEnv() {
	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	// Traverse up the directory tree
	for {
		// Check for .env files in current directory
		envFiles := []string{
			filepath.Join(dir, ".env"),
			filepath.Join(dir, ".env.local"),
		}

		found := false
		for _, envFile := range envFiles {
			if err := loadEnvFile(envFile); err == nil {
				log.Printf("Loaded environment from: %s", envFile)
				found = true
			}
		}

		if found {
			// If we found env files, we can stop searching up
			// Or we might want to continue if we want to support cascading configs
			// For now, let's stop as usually .env is at root
			break
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}
}

// loadEnvFile loads environment variables from a file
func loadEnvFile(envFilePath string) error {
	file, err := os.Open(envFilePath)
	if err != nil {
		return fmt.Errorf("failed to open env file %s: %w", envFilePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
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
	return scanner.Err()
}
