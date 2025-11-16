// server.go - Main server type definitions and core utilities
package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/stripe/stripe-go/v72"
)

// HTTPServer provides HTTP REST API for billing operations
type HTTPServer struct {
	db           database.RepositoryInterface
	stripeSecret string
}

// NewHTTPServer creates a new HTTP server instance
func NewHTTPServer(db database.RepositoryInterface, stripeSecret string) *HTTPServer {
	stripe.Key = stripeSecret

	return &HTTPServer{
		db:           db,
		stripeSecret: stripeSecret,
	}
}

// Core types and functions needed across the server
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, errorType, code, message, description, field, requestID, environment string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"type":        errorType,
			"code":        code,
			"message":     message,
			"description": description,
			"field":       field,
			"request_id":  requestID,
		},
	}

	if environment != "" {
		errorResponse["environment"] = environment
	}

	jsonData, _ := json.Marshal(errorResponse)
	if _, err := w.Write(jsonData); err != nil {
		log.Printf("Error writing error response: %v", err)
	}
}

func writeValidationError(w http.ResponseWriter, field, message, requestID, environment string) {
	writeErrorResponse(w, http.StatusBadRequest, "validation_error", "VALIDATION_FAILED",
		"Request validation failed", message, field, requestID, environment)
}

func writeAuthenticationError(w http.ResponseWriter, message, requestID, environment string) {
	writeErrorResponse(w, http.StatusUnauthorized, "authentication_error", "AUTHENTICATION_REQUIRED",
		message, "Valid API key required", "", requestID, environment)
}

func writeNotFoundError(w http.ResponseWriter, message, requestID, environment string) {
	writeErrorResponse(w, http.StatusNotFound, "not_found", "RESOURCE_NOT_FOUND",
		message, "The requested resource was not found", "", requestID, environment)
}

// Utility functions for common operations
func splitURLPath(path string) []string {
	// Simple path splitting utility
	if path == "" {
		return []string{}
	}
	// Remove leading slash and split
	if path[0] == '/' {
		path = path[1:]
	}
	if path == "" {
		return []string{}
	}
	return []string{path}
}
