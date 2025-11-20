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
	// Try multiple locations for .env file
	envFiles := []string{
		".env",
		".env.local",
	}
	
	// Also check if we're running from a subdirectory
	if cwd, err := os.Getwd(); err == nil {
		// Try relative paths from current working directory
		relativePaths := []string{
			filepath.Join(cwd, ".env"),
			filepath.Join(cwd, ".env.local"),
		}
		envFiles = append(envFiles, relativePaths...)
	}
	
	for _, envFile := range envFiles {
		if err := loadEnvFile(envFile); err == nil {
			log.Printf("Loaded environment from: %s", envFile)
			break
		}
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