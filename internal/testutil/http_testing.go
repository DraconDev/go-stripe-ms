package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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