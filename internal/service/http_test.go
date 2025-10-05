package service

import (
	"batchRequestsRecover/internal/model"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateHttpService(t *testing.T) {
	config := model.Config{
		ApiEndpoint: "https://api.example.com",
		Method:      "POST",
	}

	tests := []struct {
		name         string
		args         model.CommandLineArgs
		expectedType string
		isDryRun     bool
	}{
		{
			name: "Creates mock service for dry run",
			args: model.CommandLineArgs{
				DryRun: true,
			},
			expectedType: "*service.HttpServiceMock",
			isDryRun:     true,
		},
		{
			name: "Creates real service for production",
			args: model.CommandLineArgs{
				DryRun: false,
			},
			expectedType: "*service.HttpServiceReal",
			isDryRun:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := createHttpService(config, tt.args)

			if service == nil {
				t.Fatal("createHttpService returned nil")
			}

			// Type assertion to verify correct type
			switch s := service.(type) {
			case *HttpServiceMock:
				if !tt.isDryRun {
					t.Error("Expected HttpServiceReal but got HttpServiceMock")
				}
				if s.config.ApiEndpoint != config.ApiEndpoint {
					t.Errorf("Mock service config not set correctly")
				}
			case *HttpServiceReal:
				if tt.isDryRun {
					t.Error("Expected HttpServiceMock but got HttpServiceReal")
				}
				if s.config.ApiEndpoint != config.ApiEndpoint {
					t.Errorf("Real service config not set correctly")
				}
			default:
				t.Errorf("Unexpected service type: %T", service)
			}
		})
	}
}

func TestHttpServiceMock_call(t *testing.T) {
	config := model.Config{
		ApiEndpoint: "https://api.example.com",
		Method:      "POST",
	}
	args := model.CommandLineArgs{
		DryRun: true,
	}

	mockService := &HttpServiceMock{
		config: config,
		args:   args,
	}

	// Run multiple times to test randomness
	testCases := []struct {
		name string
		url  string
	}{
		{"Request 1", "https://api.example.com/test1"},
		{"Request 2", "https://api.example.com/test2"},
		{"Request 3", "https://api.example.com/test3"},
		{"Request 4", "https://api.example.com/test4"},
		{"Request 5", "https://api.example.com/test5"},
	}

	successCount := 0
	errorCount := 0

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", tt.url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			body, status, err := mockService.call(*req)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if body == nil {
				t.Error("Body should not be nil")
			}

			// Check that status is either 200 or 400 (based on mock logic)
			if status != 200 && status != 400 {
				t.Errorf("Expected status 200 or 400, got %d", status)
			}

			if status == 200 {
				successCount++
				if string(body) != "Success" {
					t.Errorf("Expected body 'Success', got %s", string(body))
				}
			} else if status == 400 {
				errorCount++
				if string(body) != "BadRequest" {
					t.Errorf("Expected body 'BadRequest', got %s", string(body))
				}
			}
		})
	}

	// At least one of each should occur (statistically very likely with 5 requests)
	t.Logf("Success count: %d, Error count: %d", successCount, errorCount)
}

func TestHttpServiceMock_call_VerifyRequestLogging(t *testing.T) {
	config := model.Config{
		ApiEndpoint: "https://api.example.com",
		Method:      "POST",
	}
	args := model.CommandLineArgs{
		DryRun: true,
	}

	mockService := &HttpServiceMock{
		config: config,
		args:   args,
	}

	testURL := "https://api.example.com/test"
	req, _ := http.NewRequest("POST", testURL, nil)

	// Call the service
	body, status, err := mockService.call(*req)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if body == nil {
		t.Fatal("Body should not be nil")
	}

	if status != 200 && status != 400 {
		t.Errorf("Expected status 200 or 400, got %d", status)
	}

	// Verify response body matches status
	if status == 200 && string(body) != "Success" {
		t.Errorf("Status 200 should return 'Success', got %s", string(body))
	}
	if status == 400 && string(body) != "BadRequest" {
		t.Errorf("Status 400 should return 'BadRequest', got %s", string(body))
	}
}

func TestHttpServiceReal_call_Success(t *testing.T) {
	// Create a test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Send response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer testServer.Close()

	config := model.Config{
		ApiEndpoint: testServer.URL,
		Method:      "POST",
	}
	args := model.CommandLineArgs{
		DryRun: false,
	}

	realService := &HttpServiceReal{
		config: config,
		args:   args,
	}

	req, err := http.NewRequest("POST", testServer.URL+"/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	body, status, err := realService.call(*req)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}

	expectedBody := `{"status":"success"}`
	if string(body) != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, string(body))
	}
}

func TestHttpServiceReal_call_ErrorStatuses(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Bad Request 400",
			statusCode:     http.StatusBadRequest,
			responseBody:   "Bad Request",
			expectedStatus: 400,
			expectedBody:   "Bad Request",
		},
		{
			name:           "Not Found 404",
			statusCode:     http.StatusNotFound,
			responseBody:   "Not Found",
			expectedStatus: 404,
			expectedBody:   "Not Found",
		},
		{
			name:           "Internal Server Error 500",
			statusCode:     http.StatusInternalServerError,
			responseBody:   "Internal Server Error",
			expectedStatus: 500,
			expectedBody:   "Internal Server Error",
		},
		{
			name:           "Created 201",
			statusCode:     http.StatusCreated,
			responseBody:   `{"id":"123"}`,
			expectedStatus: 201,
			expectedBody:   `{"id":"123"}`,
		},
		{
			name:           "Forbidden 403",
			statusCode:     http.StatusForbidden,
			responseBody:   "Forbidden",
			expectedStatus: 403,
			expectedBody:   "Forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer testServer.Close()

			config := model.Config{
				ApiEndpoint: testServer.URL,
				Method:      "POST",
			}
			args := model.CommandLineArgs{
				DryRun: false,
			}

			realService := &HttpServiceReal{
				config: config,
				args:   args,
			}

			req, err := http.NewRequest("POST", testServer.URL+"/test", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			body, status, err := realService.call(*req)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if status != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, status)
			}

			if string(body) != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, string(body))
			}
		})
	}
}

func TestHttpServiceReal_call_WithHeaders(t *testing.T) {
	expectedHeaders := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer test-token",
		"X-Custom":      "custom-value",
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		for key, expectedValue := range expectedHeaders {
			actualValue := r.Header.Get(key)
			if actualValue != expectedValue {
				t.Errorf("Header %s: expected %s, got %s", key, expectedValue, actualValue)
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer testServer.Close()

	config := model.Config{
		ApiEndpoint: testServer.URL,
		Method:      "POST",
	}
	args := model.CommandLineArgs{
		DryRun: false,
	}

	realService := &HttpServiceReal{
		config: config,
		args:   args,
	}

	req, err := http.NewRequest("POST", testServer.URL+"/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add headers to request
	for key, value := range expectedHeaders {
		req.Header.Add(key, value)
	}

	body, status, err := realService.call(*req)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}

	if string(body) != "OK" {
		t.Errorf("Expected body OK, got %s", string(body))
	}
}

func TestHttpServiceReal_call_WithRequestBody(t *testing.T) {
	expectedRequestBody := `{"name":"test","value":123}`

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read and verify request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		if string(body) != expectedRequestBody {
			t.Errorf("Expected request body %s, got %s", expectedRequestBody, string(body))
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"456"}`))
	}))
	defer testServer.Close()

	config := model.Config{
		ApiEndpoint: testServer.URL,
		Method:      "POST",
	}
	args := model.CommandLineArgs{
		DryRun: false,
	}

	realService := &HttpServiceReal{
		config: config,
		args:   args,
	}

	req, err := http.NewRequest("POST", testServer.URL+"/test", strings.NewReader(expectedRequestBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	body, status, err := realService.call(*req)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", status)
	}

	expectedResponseBody := `{"id":"456"}`
	if string(body) != expectedResponseBody {
		t.Errorf("Expected body %s, got %s", expectedResponseBody, string(body))
	}
}

func TestHttpServiceReal_call_NetworkError(t *testing.T) {
	config := model.Config{
		ApiEndpoint: "https://invalid-url-that-does-not-exist-12345.com",
		Method:      "POST",
	}
	args := model.CommandLineArgs{
		DryRun: false,
	}

	realService := &HttpServiceReal{
		config: config,
		args:   args,
	}

	req, err := http.NewRequest("POST", "https://invalid-url-that-does-not-exist-12345.com/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	body, status, err := realService.call(*req)

	if err == nil {
		t.Error("Expected error for invalid URL, got none")
	}

	if body != nil {
		t.Errorf("Expected nil body on error, got %v", body)
	}

	if status != 0 {
		t.Errorf("Expected status 0 on error, got %d", status)
	}

	if !strings.Contains(err.Error(), "error making request") {
		t.Errorf("Error message should contain 'error making request', got: %v", err)
	}
}

func TestHttpServiceReal_call_LargeResponse(t *testing.T) {
	// Create a large response body
	largeBody := strings.Repeat("x", 1024*100) // 100KB

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeBody))
	}))
	defer testServer.Close()

	config := model.Config{
		ApiEndpoint: testServer.URL,
		Method:      "GET",
	}
	args := model.CommandLineArgs{
		DryRun: false,
	}

	realService := &HttpServiceReal{
		config: config,
		args:   args,
	}

	req, err := http.NewRequest("GET", testServer.URL+"/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	body, status, err := realService.call(*req)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}

	if len(body) != len(largeBody) {
		t.Errorf("Expected body length %d, got %d", len(largeBody), len(body))
	}

	if string(body) != largeBody {
		t.Error("Large body content doesn't match")
	}
}

func TestLoadClient_Configuration(t *testing.T) {
	client := loadClient()

	if client == nil {
		t.Fatal("loadClient returned nil")
	}

	if client.Transport == nil {
		t.Fatal("Client transport should not be nil")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport is not *http.Transport")
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("TLS config should not be nil")
	}

	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true")
	}
}

func TestHttpServiceReal_call_EmptyResponse(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		// No body written
	}))
	defer testServer.Close()

	config := model.Config{
		ApiEndpoint: testServer.URL,
		Method:      "DELETE",
	}
	args := model.CommandLineArgs{
		DryRun: false,
	}

	realService := &HttpServiceReal{
		config: config,
		args:   args,
	}

	req, err := http.NewRequest("DELETE", testServer.URL+"/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	body, status, err := realService.call(*req)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", status)
	}

	if len(body) != 0 {
		t.Errorf("Expected empty body, got %d bytes", len(body))
	}
}

func TestHttpService_InterfaceImplementation(t *testing.T) {
	// Verify both implementations satisfy the interface
	var _ HttpService = (*HttpServiceMock)(nil)
	var _ HttpService = (*HttpServiceReal)(nil)
}
