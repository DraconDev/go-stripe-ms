package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Enhanced input validation
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// Enhanced error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
	Meta  ErrorMeta   `json:"meta,omitempty"`
}

type ErrorDetail struct {
	Type        string `json:"type"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description,omitempty"`
	Field       string `json:"field,omitempty"`
}

type ErrorMeta struct {
	RequestID   string    `json:"request_id"`
	Timestamp   time.Time `json:"timestamp"`
	Environment string    `json:"environment,omitempty"`
}

// WriteErrorResponse writes a standardized JSON error response
func WriteErrorResponse(w http.ResponseWriter, statusCode int, errorType, code, message, description, field, requestID, environment string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Add security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	response := ErrorResponse{
		Error: ErrorDetail{
			Type:        errorType,
			Code:        code,
			Message:     message,
			Description: description,
			Field:       field,
		},
		Meta: ErrorMeta{
			RequestID:   requestID,
			Timestamp:   time.Now(),
			Environment: environment,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding error response: %v", err)
		// Fallback to a plain text error if JSON encoding fails
		http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
	}
}

func writeValidationError(w http.ResponseWriter, field, message, requestID, environment string) {
	WriteErrorResponse(w, http.StatusBadRequest, "validation_error", "VALIDATION_FAILED",
		"Request validation failed", message, field, requestID, environment)
}

func writeAuthenticationError(w http.ResponseWriter, message, requestID, environment string) {
	WriteErrorResponse(w, http.StatusUnauthorized, "authentication_error", "AUTHENTICATION_REQUIRED",
		message, "Valid API key required", "", requestID, environment)
}

func writeNotFoundError(w http.ResponseWriter, message, requestID, environment string) {
	WriteErrorResponse(w, http.StatusNotFound, "not_found", "RESOURCE_NOT_FOUND",
		message, "The requested resource was not found", "", requestID, environment)
}

func writeRateLimitError(w http.ResponseWriter, requestID, environment string) {
	w.Header().Set("Retry-After", "60")
	WriteErrorResponse(w, http.StatusTooManyRequests, "rate_limit_error", "RATE_LIMIT_EXCEEDED",
		"Too many requests", "Please try again later", "", requestID, environment)
}
