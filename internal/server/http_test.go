package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"styx/internal/database"
)

// MockDatabase is a mock implementation of database.Repository for testing
type MockDatabase struct {
	customers    map[string]*database.Customer
	subscriptions map[string]*database.Subscription
	getCustomerError     error
	createCustomerError  error
	getSubscriptionError error
	createSubscriptionError error
	updateSubscriptionError error
}

func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		customers:    make(map[string]*database.Customer),
		subscriptions: make(map[string]*database.Subscription),
	}
}

func (m *MockDatabase) FindOrCreateStripeCustomer(ctx context.Context, userID, email string) (string, error) {
	if m.getCustomerError != nil {
		return "", m.getCustomerError
	}

	customer, exists := m.customers[userID]
	if exists && customer.StripeCustomerID != "" {
		return customer.StripeCustomerID, nil
	}

	return "", m.createCustomerError
}

func (m *MockDatabase) UpdateCustomerStripeID(ctx context.Context, userID, stripeCustomerID string) error {
	if m.updateSubscriptionError != nil {
		return m.updateSubscriptionError
	}

	if customer, exists := m.customers[userID]; exists {
		customer.StripeCustomerID = stripeCustomerID
		customer.UpdatedAt = time.Now()
	} else {
		// Create new customer if not exists
		m.customers[userID] = &database.Customer{
			UserID:           userID,
			Email:            "test@example.com",
			StripeCustomerID: stripeCustomerID,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
	}
	return nil
}

func (m *MockDatabase) GetSubscriptionStatus(ctx context.Context, userID, productID string) (string, string, time.Time, bool, error) {
	if m.getSubscriptionError != nil {
		return "", "", time.Time{}, false, m.getSubscriptionError
	}

	key := userID + "/" + productID
	if sub, exists := m.subscriptions[key]; exists {
		return sub.StripeSubscriptionID, sub.Status, sub.CurrentPeriodEnd, true, nil
	}

	return "", "", time.Time{}, false, nil
}

func (m *MockDatabase) CreateSubscription(ctx context.Context, customerID, stripeSubID, productID, priceID, userID, status string, periodStart, periodEnd time.Time) error {
	if m.createSubscriptionError != nil {
		return m.createSubscriptionError
	}

	key := userID + "/" + productID
	m.subscriptions[key] = &database.Subscription{
		CustomerID:           [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		UserID:               userID,
		ProductID:            productID,
		PriceID:              priceID,
		StripeSubscriptionID: stripeSubID,
		Status:               status,
		CurrentPeriodStart:   periodStart,
		CurrentPeriodEnd:     periodEnd,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
	return nil
}

func (m *MockDatabase) UpdateSubscriptionStatus(ctx context.Context, stripeSubID, status string, periodEnd time.Time) error {
	if m.updateSubscriptionError != nil {
		return m.updateSubscriptionError
	}

	// Find subscription by Stripe ID and update
	for _, sub := range m.subscriptions {
		if sub.StripeSubscriptionID == stripeSubID {
			sub.Status = status
			sub.CurrentPeriodEnd = periodEnd
			sub.UpdatedAt = time.Now()
			break
		}
	}
	return nil
}

func (m *MockDatabase) GetCustomerByStripeID(ctx context.Context, stripeCustomerID string) (*database.Customer, error) {
	if m.getCustomerError != nil {
		return nil, m.getCustomerError
	}

	for _, customer := range m.customers {
		if customer.StripeCustomerID == stripeCustomerID {
			return customer, nil
		}
	}
	return nil, nil
}

func (m *MockDatabase) GetCustomerByUserID(ctx context.Context, userID string) (*database.Customer, error) {
	if m.getCustomerError != nil {
		return nil, m.getCustomerError
	}

	customer, exists := m.customers[userID]
	if !exists {
		return nil, nil
	}
	return customer, nil
}

func (m *MockDatabase) GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*database.Subscription, error) {
	if m.getSubscriptionError != nil {
		return nil, m.getSubscriptionError
	}

	for _, sub := range m.subscriptions {
		if sub.StripeSubscriptionID == stripeSubID {
			return sub, nil
		}
	}
	return nil, nil
}

func (m *MockDatabase) InitializeTables(ctx context.Context) error {
	return nil
}

func (m *MockDatabase) AddTestCustomer(customer *database.Customer) {
	m.customers[customer.UserID] = customer
}

func (m *MockDatabase) AddTestSubscription(sub *database.Subscription) {
	key := sub.UserID + "/" + sub.ProductID
	m.subscriptions[key] = sub
}

func (m *MockDatabase) SetGetCustomerError(err error) {
	m.getCustomerError = err
}

func (m *MockDatabase) SetCreateCustomerError(err error) {
	m.createCustomerError = err
}

func (m *MockDatabase) SetGetSubscriptionError(err error) {
	m.getSubscriptionError = err
}

func (m *MockDatabase) SetCreateSubscriptionError(err error) {
	m.createSubscriptionError = err
}

func (m *MockDatabase) SetUpdateSubscriptionError(err error) {
	m.updateSubscriptionError = err
}

// Test Suite for CreateSubscriptionCheckout
func TestCreateSubscriptionCheckout(t *testing.T) {
	tests := []struct {
		name                string
		requestBody         map[string]interface{}
		setupMockDatabase   func(*MockDatabase)
		expectedStatusCode  int
		expectedResponse    map[string]interface{}
		expectedError       string
	}{
		{
			name: "Valid checkout request",
			requestBody: map[string]interface{}{
				"user_id":     "user123",
				"email":       "test@example.com",
				"product_id":  "premium_plan",
				"price_id":    "price_test123",
				"success_url": "https://example.com/success",
				"cancel_url":  "https://example.com/cancel",
			},
			setupMockDatabase: func(m *MockDatabase) {
				m.AddTestCustomer(&database.Customer{
					UserID:           "user123",
					Email:            "test@example.com",
					StripeCustomerID: "cus_test123",
					CreatedAt:        time.Now(),
					UpdatedAt:        time.Now(),
				})
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Missing user_id",
			requestBody:         map[string]interface{}{
				"email":       "test@example.com",
				"product_id":  "premium_plan",
				"price_id":    "price_test123",
				"success_url": "https://example.com/success",
				"cancel_url":  "https://example.com/cancel",
			},
			setupMockDatabase:   func(m *MockDatabase) {},
			expectedStatusCode:  http.StatusBadRequest,
			expectedError:       "Missing required fields",
		},
		{
			name:               "Invalid JSON body",
			requestBody:         map[string]interface{}{
				"user_id": "user123",
				"invalid": make(chan int), // Unmarshallable type
			},
			setupMockDatabase:   func(m *MockDatabase) {},
			expectedStatusCode:  http.StatusBadRequest,
			expectedError:       "Invalid request body",
		},
		{
			name: "Empty request body",
			requestBody: map[string]interface{}{
				"user_id": "",
				"email":   "",
				"product_id": "",
				"price_id": "",
				"success_url": "",
				"cancel_url": "",
			},
			setupMockDatabase:   func(m *MockDatabase) {},
			expectedStatusCode:  http.StatusBadRequest,
			expectedError:       "Missing required fields",
		},
		{
			name:               "Method not allowed",
			requestBody:         nil,
			setupMockDatabase:   func(m *MockDatabase) {},
			expectedStatusCode:  http.StatusMethodNotAllowed,
			expectedError:       "Method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockDB := NewMockDatabase()
			tt.setupMockDatabase(mockDB)
			
			server := NewHTTPServer(mockDB, "sk_test_123")

			// Create request
			var req *http.Request
			if tt.requestBody != nil {
				bodyBytes, _ := json.Marshal(tt.requestBody)
				req = httptest.NewRequest(http.MethodPost, "/api/v1/checkout", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.requestBody["method"].(string), "/api/v1/checkout", nil)
			}

			// Execute
			w := httptest.NewRecorder()
			server.CreateSubscriptionCheckout(w, req)

			// Assert
			if w.Code != tt.expectedStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatusCode, w.Code)
			}

			if tt.expectedError != "" {
				var errorResponse map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &errorResponse); err != nil {
					t.Errorf("Failed to unmarshal error response: %v", err)
				} else if errorResponse["error"] != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, errorResponse["error"])
				}
			}

			if tt.expectedResponse != nil && w.Code == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}
				// Add more specific assertions if needed
			}

			// Check Content-Type header
			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
			}
		})
	}
}

// Test Suite for GetSubscriptionStatus
func TestGetSubscriptionStatus(t *testing.T) {
	tests := []struct {
		name               string
		path               string
		setupMockDatabase  func(*MockDatabase)
		expectedStatusCode int
		expectedResponse   map[string]interface{}
		expectedError      string
	}{
		{
			name:               "Valid subscription request",
			path:               "/api/v1/subscriptions/user123/premium_plan",
			setupMockDatabase:  func(m *MockDatabase) {
				m.AddTestSubscription(&database.Subscription{
					UserID:               "user123",
					ProductID:            "premium_plan",
					StripeSubscriptionID: "sub_test123",
					Status:               "active",
					CurrentPeriodEnd:     time.Now().AddDate(0, 0, 30),
				})
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Non-existent subscription",
			path:               "/api/v1/subscriptions/user456/premium_plan",
			setupMockDatabase:  func(m *MockDatabase) {},
			expectedStatusCode: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"exists": false,
			},
		},
		{
			name:               "Invalid URL format - missing product_id",
			path:               "/api/v1/subscriptions/user123",
			setupMockDatabase:  func(m *MockDatabase) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "Invalid URL format",
		},
		{
			name:               "Empty user_id",
			path:               "/api/v1/subscriptions//premium_plan",
			setupMockDatabase:  func(m *MockDatabase) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "Invalid URL format",
		},
		{
			name:               "Empty product_id",
			path:               "/api/v1/subscriptions/user123/",
			setupMockDatabase:  func(m *MockDatabase) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "Invalid URL format",
		},
		{
			name:               "Method not allowed",
			path:               "/api/v1/subscriptions/user123/premium_plan",
			setupMockDatabase:  func(m *MockDatabase) {},
			expectedStatusCode: http.StatusMethodNotAllowed,
			expectedError:      "Method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockDB := NewMockDatabase()
			tt.setupMockDatabase(mockDB)
			
			server := NewHTTPServer(mockDB, "sk_test_123")

			// Create request
			method := http.MethodGet
			if strings.Contains(tt.name, "Method not allowed") {
				method = http.MethodPost
			}
			req := httptest.NewRequest(method, tt.path, nil)

			// Execute
			w := httptest.NewRecorder()
			server.GetSubscriptionStatus(w, req)

			// Assert
			if w.Code != tt.expectedStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatusCode, w.Code)
			}

			if tt.expectedError != "" {
				var errorResponse map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &errorResponse); err != nil {
					t.Errorf("Failed to unmarshal error response: %v", err)
				} else if errorResponse["error"] != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, errorResponse["error"])
				}
			}

			if tt.expectedResponse != nil {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				} else {
					for key, expectedValue := range tt.expectedResponse {
						if actualValue, exists := response[key]; !exists || actualValue != expectedValue {
							t.Errorf("Expected response[%s] = %v, got %v", key, expectedValue, actualValue)
						}
					}
				}
			}

			// Check Content-Type header
			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
			}
		})
	}
}

// Test Suite for CreateCustomerPortal
func TestCreateCustomerPortal(t *testing.T) {
	tests := []struct {
		name                string
		requestBody         map[string]interface{}
		setupMockDatabase   func(*MockDatabase)
		expectedStatusCode  int
		expectedResponse    map[string]interface{}
		expectedError       string
	}{
		{
			name: "Valid portal request",
			requestBody: map[string]interface{}{
				"user_id":    "user123",
				"return_url": "https://example.com/account",
			},
			setupMockDatabase: func(m *MockDatabase) {
				m.AddTestCustomer(&database.Customer{
					UserID:           "user123",
					Email:            "test@example.com",
					StripeCustomerID: "cus_test123",
					CreatedAt:        time.Now(),
					UpdatedAt:        time.Now(),
				})
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Missing user_id",
			requestBody: map[string]interface{}{
				"return_url": "https://example.com/account",
			},
			setupMockDatabase:   func(m *MockDatabase) {},
			expectedStatusCode:  http.StatusBadRequest,
			expectedError:       "Missing required fields",
		},
		{
			name:               "Missing return_url",
			requestBody: map[string]interface{}{
				"user_id": "user123",
			},
			setupMockDatabase:   func(m *MockDatabase) {},
			expectedStatusCode:  http.StatusBadRequest,
			expectedError:       "Missing required fields",
		},
		{
			name:               "Customer not found",
			requestBody: map[string]interface{}{
				"user_id":    "nonexistent",
				"return_url": "https://example.com/account",
			},
			setupMockDatabase:   func(m *MockDatabase) {},
			expectedStatusCode:  http.StatusNotFound,
			expectedError:       "Customer not found",
		},
		{
			name:               "Customer has no Stripe customer ID",
			requestBody: map[string]interface{}{
				"user_id":    "user123",
				"return_url": "https://example.com/account",
			},
			setupMockDatabase: func(m *MockDatabase) {
				m.AddTestCustomer(&database.Customer{
					UserID:           "user123",
					Email:            "test@example.com",
					StripeCustomerID: "",
					CreatedAt:        time.Now(),
					UpdatedAt:        time.Now(),
				})
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "Customer has no Stripe customer ID",
		},
		{
			name:               "Invalid JSON body",
			requestBody: map[string]interface{}{
				"user_id":    "user123",
				"invalid":    make(chan int),
			},
			setupMockDatabase:   func(m *MockDatabase) {},
			expectedStatusCode:  http.StatusBadRequest,
			expectedError:       "Invalid request body",
		},
		{
			name:               "Method not allowed",
			requestBody:         nil,
			setupMockDatabase:   func(m *MockDatabase) {},
			expectedStatusCode:  http.StatusMethodNotAllowed,
			expectedError:       "Method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockDB := NewMockDatabase()
			tt.setupMockDatabase(mockDB)
			
			server := NewHTTPServer(mockDB, "sk_test_123")

			// Create request
			var req *http.Request
			if tt.requestBody != nil {
				bodyBytes, _ := json.Marshal(tt.requestBody)
				req = httptest.NewRequest(http.MethodPost, "/api/v1/portal", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.requestBody["method"].(string), "/api/v1/portal", nil)
			}

			// Execute
			w := httptest.NewRecorder()
			server.CreateCustomerPortal(w, req)

			// Assert
			if w.Code != tt.expectedStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatusCode, w.Code)
			}

			if tt.expectedError != "" {
				var errorResponse map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &errorResponse); err != nil {
					t.Errorf("Failed to unmarshal error response: %v", err)
				} else if errorResponse["error"] != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, errorResponse["error"])
				}
			}

			if tt.expectedResponse != nil && w.Code == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}
				// Add more specific assertions if needed
			}

			// Check Content-Type header
			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
			}
		})
	}
}

// Test Suite for HealthCheck
func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		expectedStatusCode int
		expectedResponse   map[string]interface{}
	}{
		{
			name:               "Valid health check",
			method:             http.MethodGet,
			expectedStatusCode: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"status":  "healthy",
				"service": "billing-service",
			},
		},
		{
			name:               "Method not allowed",
			method:             http.MethodPost,
			expectedStatusCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockDB := NewMockDatabase()
			server := NewHTTPServer(mockDB, "sk_test_123")

			// Create request
			req := httptest.NewRequest(tt.method, "/health", nil)

			// Execute
			w := httptest.NewRecorder()
			server.HealthCheck(w, req)

			// Assert
			if w.Code != tt.expectedStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatusCode, w.Code)
			}

			if tt.expectedResponse != nil {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				} else {
					for key, expectedValue := range tt.expectedResponse {
						if actualValue, exists := response[key]; !exists || actualValue != expectedValue {
							t.Errorf("Expected response[%s] = %v, got %v", key, expectedValue, actualValue)
						}
					}
				}

				// Check that timestamp exists and is valid
				if timestamp, exists := response["timestamp"]; exists {
					if _, ok := timestamp.(string); !ok {
						t.Errorf("Expected timestamp to be a string, got %T", timestamp)
					}
				}
			}

			// Check Content-Type header
			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
			}
		})
	}
}

// Benchmark tests for performance testing
func BenchmarkCreateSubscriptionCheckout(b *testing.B) {
	mockDB := NewMockDatabase()
	mockDB.AddTestCustomer(&database.Customer{
		UserID:           "benchmark-user",
		Email:            "benchmark@example.com",
		StripeCustomerID: "cus_benchmark",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	})

	server := NewHTTPServer(mockDB, "sk_test_123")
	
	requestBody := map[string]interface{}{
		"user_id":     "benchmark-user",
		"email":       "benchmark@example.com",
		"product_id":  "premium_plan",
		"price_id":    "price_benchmark",
		"success_url": "https://example.com/success",
		"cancel_url":  "https://example.com/cancel",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		server.CreateSubscriptionCheckout(w, req)
	}
}

func BenchmarkGetSubscriptionStatus(b *testing.B) {
	mockDB := NewMockDatabase()
	mockDB.AddTestSubscription(&database.Subscription{
		UserID:               "benchmark-user",
		ProductID:            "premium_plan",
		StripeSubscriptionID: "sub_benchmark",
		Status:               "active",
		CurrentPeriodEnd:     time.Now().AddDate(0, 0, 30),
	})

	server := NewHTTPServer(mockDB, "sk_test_123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/benchmark-user/premium_plan", nil)
		w := httptest.NewRecorder()
		server.GetSubscriptionStatus(w, req)
	}
}