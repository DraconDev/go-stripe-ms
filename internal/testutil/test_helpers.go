package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"styx/internal/database"
)

// HTTPTestRequest represents a test HTTP request with expected response
type HTTPTestRequest struct {
	Method       string
	Path         string
	Body         interface{}
	Headers      map[string]string
	StatusCode   int
	ResponseType interface{}
}

// HTTPResponse captures the response from an HTTP handler
type HTTPResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// CreateTestRequest creates an HTTP test request
func CreateTestRequest(req HTTPTestRequest) *http.Request {
	var bodyReader *strings.Reader
	if req.Body != nil {
		bodyBytes, _ := json.Marshal(req.Body)
		bodyReader = strings.NewReader(string(bodyBytes))
	} else {
		bodyReader = strings.NewReader("")
	}
	
	reqHTTP := httptest.NewRequest(req.Method, req.Path, bodyReader)
	
	// Add headers
	for key, value := range req.Headers {
		reqHTTP.Header.Set(key, value)
	}
	
	// Set default content type if not provided
	if _, exists := req.Headers["Content-Type"]; !exists && req.Body != nil {
		reqHTTP.Header.Set("Content-Type", "application/json")
	}
	
	return reqHTTP
}

// ExecuteTestRequest executes an HTTP handler test and returns the response
func ExecuteTestRequest(handler http.Handler, req HTTPTestRequest) HTTPResponse {
	httpReq := CreateTestRequest(req)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, httpReq)
	
	// Capture headers
	headers := make(map[string]string)
	for key, values := range w.Header() {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	
	return HTTPResponse{
		StatusCode: w.Code,
		Body:       w.Body.String(),
		Headers:    headers,
	}
}

// AssertJSONResponse validates that the response matches expected JSON structure
func AssertJSONResponse(t *testing.T, response HTTPResponse, expectedStatusCode int, expectedResponse interface{}) {
	t.Helper()
	
	if response.StatusCode != expectedStatusCode {
		t.Errorf("Expected status code %d, got %d", expectedStatusCode, response.StatusCode)
	}
	
	if expectedResponse != nil {
		var actual interface{}
		if err := json.Unmarshal([]byte(response.Body), &actual); err != nil {
			t.Errorf("Failed to unmarshal response body: %v", err)
			return
		}
		
		expectedJSON, _ := json.Marshal(expectedResponse)
		actualJSON, _ := json.Marshal(actual)
		
		if string(expectedJSON) != string(actualJSON) {
			t.Errorf("Response mismatch:\nExpected: %s\nActual: %s", string(expectedJSON), string(actualJSON))
		}
	}
}

// AssertErrorResponse validates error response structure
func AssertErrorResponse(t *testing.T, response HTTPResponse, expectedStatusCode int, expectedError string) {
	t.Helper()
	
	if response.StatusCode != expectedStatusCode {
		t.Errorf("Expected status code %d, got %d", expectedStatusCode, response.StatusCode)
	}
	
	var errorResponse struct {
		Error string `json:"error"`
	}
	
	if err := json.Unmarshal([]byte(response.Body), &errorResponse); err != nil {
		t.Errorf("Failed to unmarshal error response: %v", err)
		return
	}
	
	if errorResponse.Error != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, errorResponse.Error)
	}
}

// CreateMockCustomer creates a mock customer for testing
func CreateMockCustomer() *database.Customer {
	return &database.Customer{
		ID:               [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		UserID:           "test-user-123",
		Email:            "test@example.com",
		StripeCustomerID: "cus_test123456789",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// CreateMockSubscription creates a mock subscription for testing
func CreateMockSubscription() *database.Subscription {
	return &database.Subscription{
		ID:                   [16]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17},
		CustomerID:           [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		UserID:               "test-user-123",
		ProductID:            "premium_plan",
		PriceID:              "price_test123456789",
		StripeSubscriptionID: "sub_test123456789",
		Status:               "active",
		CurrentPeriodStart:   time.Now(),
		CurrentPeriodEnd:     time.Now().AddDate(0, 0, 30),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}

// NormalizeJSON normalizes JSON for comparison
func NormalizeJSON(t *testing.T, data interface{}) string {
	t.Helper()
	
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Errorf("Failed to marshal JSON: %v", err)
		return ""
	}
	
	var normalized interface{}
	if err := json.Unmarshal(jsonBytes, &normalized); err != nil {
		t.Errorf("Failed to normalize JSON: %v", err)
		return string(jsonBytes)
	}
	
	normalizedBytes, _ := json.Marshal(normalized)
	return string(normalizedBytes)
}

// CapturePanic captures panics from test functions
func CapturePanic(t *testing.T, testFunc func()) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Test panicked: %v", r)
		}
	}()
	testFunc()
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func init() {
	// This will make tests work without imports from other packages
}