package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Set a dummy Stripe key for testing
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_dummy_key")
	code := m.Run()
	os.Exit(code)
}

func TestHealthCheck(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthCheck)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "OK"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestCreatePaymentIntent(t *testing.T) {
	// Skip this test if running without proper Stripe key
	if os.Getenv("STRIPE_SECRET_KEY") == "sk_test_dummy_key" {
		t.Skip("Skipping integration test with dummy key")
	}

	payload := PaymentRequest{
		Amount:   1000,
		Currency: "usd",
	}

	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "/create-payment-intent", bytes.NewBuffer(jsonPayload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createPaymentIntent)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response PaymentResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.ClientSecret == "" {
		t.Error("Expected non-empty client_secret")
	}
}
