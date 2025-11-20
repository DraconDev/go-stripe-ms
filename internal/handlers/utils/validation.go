package utils

import (
	"net"
	"net/mail"
	"regexp"
	"strings"
)

// ValidationError and related types are defined in errors.go
// This file contains validation helper functions

func validateEmail(email string) error {
	if email == "" {
		return &ValidationError{Field: "email", Message: "email is required"}
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return &ValidationError{Field: "email", Message: "invalid email format"}
	}

	return nil
}

func validateURL(url string) error {
	if url == "" {
		return &ValidationError{Field: "url", Message: "url is required"}
	}

	// Basic URL validation
	_, err := net.LookupHost(strings.TrimPrefix(url, "https://"))
	if err != nil {
		return &ValidationError{Field: "url", Message: "invalid URL"}
	}

	return nil
}

func validateRequiredString(value, fieldName string) error {
	if value == "" {
		return &ValidationError{Field: fieldName, Message: fieldName + " is required"}
	}
	return nil
}

func validateUserID(userID string) error {
	if err := validateRequiredString(userID, "user_id"); err != nil {
		return err
	}

	// Basic format validation
	userIDRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !userIDRegex.MatchString(userID) {
		return &ValidationError{Field: "user_id", Message: "user_id must contain only alphanumeric characters, hyphens, and underscores"}
	}

	if len(userID) > 100 {
		return &ValidationError{Field: "user_id", Message: "user_id must be less than 100 characters"}
	}

	return nil
}

func validateCheckoutRequest(userID, email, productID, priceID, successURL, cancelURL string) error {
	if err := validateUserID(userID); err != nil {
		return err
	}
	if err := validateEmail(email); err != nil {
		return err
	}
	if err := validateRequiredString(productID, "product_id"); err != nil {
		return err
	}
	if err := validateRequiredString(priceID, "price_id"); err != nil {
		return err
	}
	if err := validateURL(successURL); err != nil {
		return err
	}
	if err := validateURL(cancelURL); err != nil {
		return err
	}
	return nil
}

func validatePortalRequest(userID, returnURL string) error {
	if err := validateUserID(userID); err != nil {
		return err
	}
	if err := validateURL(returnURL); err != nil {
		return err
	}
	return nil
}
