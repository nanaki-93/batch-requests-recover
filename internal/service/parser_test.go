package service

import (
	"batchRequestsRecover/internal/model"
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// MockHttpService is a test mock for HttpService
type MockHttpService struct {
	callFunc func(record http.Request) ([]byte, int, error)
}

func (m *MockHttpService) call(record http.Request) ([]byte, int, error) {
	if m.callFunc != nil {
		return m.callFunc(record)
	}
	return []byte("default response"), 200, nil
}

func TestNewProcessService(t *testing.T) {
	config := model.Config{
		ApiEndpoint: "https://api.example.com",
		Method:      "POST",
	}
	args := model.CommandLineArgs{
		DryRun:      true,
		SleepMillis: 0,
	}

	service := NewProcessService(config, args)

	if service == nil {
		t.Fatal("NewProcessService returned nil")
	}

	if service.config.ApiEndpoint != config.ApiEndpoint {
		t.Errorf("Expected ApiEndpoint %s, got %s", config.ApiEndpoint, service.config.ApiEndpoint)
	}

	if service.args.DryRun != args.DryRun {
		t.Errorf("Expected DryRun %v, got %v", args.DryRun, service.args.DryRun)
	}

	if service.httpService == nil {
		t.Error("httpService should not be nil")
	}
}

func TestProcessService_processRecord_Success(t *testing.T) {
	tests := []struct {
		name         string
		responseBody []byte
		statusCode   int
		expectedType model.ResponseType
		expectedMsg  string
		mockError    error
	}{
		{
			name:         "Successful 200 response",
			responseBody: []byte(`{"status":"ok"}`),
			statusCode:   200,
			expectedType: model.SUCCESS,
			expectedMsg:  `0-200 - {"status":"ok"}`,
			mockError:    nil,
		},
		{
			name:         "Successful 201 response",
			responseBody: []byte(`{"id":"123"}`),
			statusCode:   201,
			expectedType: model.SUCCESS,
			expectedMsg:  `5-201 - {"id":"123"}`,
			mockError:    nil,
		},
		{
			name:         "Client error 400 response",
			responseBody: []byte("Bad Request"),
			statusCode:   400,
			expectedType: model.ERROR,
			expectedMsg:  "0-400 - Bad Request",
			mockError:    nil,
		},
		{
			name:         "Server error 500 response",
			responseBody: []byte("Internal Server Error"),
			statusCode:   500,
			expectedType: model.ERROR,
			expectedMsg:  "1-500 - Internal Server Error",
			mockError:    nil,
		},
		{
			name:         "Not found 404 response",
			responseBody: []byte("Not Found"),
			statusCode:   404,
			expectedType: model.ERROR,
			expectedMsg:  "2-404 - Not Found",
			mockError:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockHttpService{
				callFunc: func(record http.Request) ([]byte, int, error) {
					return tt.responseBody, tt.statusCode, tt.mockError
				},
			}

			service := &ProcessService{
				httpService: mockService,
			}

			req, _ := http.NewRequest("GET", "https://api.example.com/test", nil)

			// Extract index from expected message
			var index int
			if strings.Contains(tt.name, "201") {
				index = 5
			} else if strings.Contains(tt.name, "500") {
				index = 1
			} else if strings.Contains(tt.name, "404") {
				index = 2
			} else {
				index = 0
			}

			response, err := service.processRecord(*req, index)

			if tt.mockError != nil {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if response.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, response.Type)
			}

			if response.Message != tt.expectedMsg {
				t.Errorf("Expected message %q, got %q", tt.expectedMsg, response.Message)
			}
		})
	}
}

func TestProcessService_processRecord_Error(t *testing.T) {
	mockError := errors.New("network error")
	mockService := &MockHttpService{
		callFunc: func(record http.Request) ([]byte, int, error) {
			return nil, 0, mockError
		},
	}

	service := &ProcessService{
		httpService: mockService,
	}

	req, _ := http.NewRequest("GET", "https://api.example.com/test", nil)
	response, err := service.processRecord(*req, 0)

	if err == nil {
		t.Fatal("Expected error but got none")
	}

	if !strings.Contains(err.Error(), "error making request") {
		t.Errorf("Error message should contain 'error making request', got: %v", err)
	}

	if response.Type != model.ERROR {
		t.Errorf("Expected ERROR type, got %v", response.Type)
	}
}

func TestProcessService_ProcessAll(t *testing.T) {
	tests := []struct {
		name          string
		records       []http.Request
		mockResponses []struct {
			body   []byte
			status int
			err    error
		}
		expectedRespCount int
		expectedErrCount  int
		expectError       bool
	}{
		{
			name: "All successful requests",
			records: []http.Request{
				*createTestRequest("https://api.example.com/1"),
				*createTestRequest("https://api.example.com/2"),
				*createTestRequest("https://api.example.com/3"),
			},
			mockResponses: []struct {
				body   []byte
				status int
				err    error
			}{
				{[]byte("ok1"), 200, nil},
				{[]byte("ok2"), 200, nil},
				{[]byte("ok3"), 201, nil},
			},
			expectedRespCount: 3,
			expectedErrCount:  0,
			expectError:       false,
		},
		{
			name: "Mixed success and error responses",
			records: []http.Request{
				*createTestRequest("https://api.example.com/1"),
				*createTestRequest("https://api.example.com/2"),
				*createTestRequest("https://api.example.com/3"),
				*createTestRequest("https://api.example.com/4"),
			},
			mockResponses: []struct {
				body   []byte
				status int
				err    error
			}{
				{[]byte("ok"), 200, nil},
				{[]byte("bad request"), 400, nil},
				{[]byte("ok"), 200, nil},
				{[]byte("server error"), 500, nil},
			},
			expectedRespCount: 2,
			expectedErrCount:  2,
			expectError:       false,
		},
		{
			name: "All error responses",
			records: []http.Request{
				*createTestRequest("https://api.example.com/1"),
				*createTestRequest("https://api.example.com/2"),
			},
			mockResponses: []struct {
				body   []byte
				status int
				err    error
			}{
				{[]byte("not found"), 404, nil},
				{[]byte("forbidden"), 403, nil},
			},
			expectedRespCount: 0,
			expectedErrCount:  2,
			expectError:       false,
		},
		{
			name: "Network error during processing",
			records: []http.Request{
				*createTestRequest("https://api.example.com/1"),
			},
			mockResponses: []struct {
				body   []byte
				status int
				err    error
			}{
				{nil, 0, errors.New("network timeout")},
			},
			expectedRespCount: 0,
			expectedErrCount:  0,
			expectError:       true,
		},
		{
			name:    "Empty records list",
			records: []http.Request{},
			mockResponses: []struct {
				body   []byte
				status int
				err    error
			}{},
			expectedRespCount: 0,
			expectedErrCount:  0,
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			mockService := &MockHttpService{
				callFunc: func(record http.Request) ([]byte, int, error) {
					if callCount >= len(tt.mockResponses) {
						t.Fatalf("Unexpected call count: %d", callCount)
					}
					resp := tt.mockResponses[callCount]
					callCount++
					return resp.body, resp.status, resp.err
				},
			}

			service := &ProcessService{
				args: model.CommandLineArgs{
					SleepMillis: 0, // No sleep for tests
				},
				httpService: mockService,
			}

			respList, errList, err := service.ProcessAll(tt.records)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(respList) != tt.expectedRespCount {
				t.Errorf("Expected %d successful responses, got %d", tt.expectedRespCount, len(respList))
			}

			if len(errList) != tt.expectedErrCount {
				t.Errorf("Expected %d error responses, got %d", tt.expectedErrCount, len(errList))
			}

			// Verify response format
			for _, resp := range respList {
				if !strings.Contains(resp, "-") {
					t.Errorf("Response format incorrect: %s", resp)
				}
			}

			for _, errResp := range errList {
				if !strings.Contains(errResp, "-") {
					t.Errorf("Error response format incorrect: %s", errResp)
				}
			}
		})
	}
}

func TestCreateResponseFromStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		message      string
		expectedType model.ResponseType
	}{
		{
			name:         "Status 200 is success",
			status:       200,
			message:      "OK",
			expectedType: model.SUCCESS,
		},
		{
			name:         "Status 201 is success",
			status:       201,
			message:      "Created",
			expectedType: model.SUCCESS,
		},
		{
			name:         "Status 299 is success",
			status:       299,
			message:      "Custom success",
			expectedType: model.SUCCESS,
		},
		{
			name:         "Status 199 is error",
			status:       199,
			message:      "Informational",
			expectedType: model.ERROR,
		},
		{
			name:         "Status 300 is error",
			status:       300,
			message:      "Redirect",
			expectedType: model.ERROR,
		},
		{
			name:         "Status 400 is error",
			status:       400,
			message:      "Bad Request",
			expectedType: model.ERROR,
		},
		{
			name:         "Status 404 is error",
			status:       404,
			message:      "Not Found",
			expectedType: model.ERROR,
		},
		{
			name:         "Status 500 is error",
			status:       500,
			message:      "Internal Server Error",
			expectedType: model.ERROR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := createResponseFromStatus(tt.status, tt.message)

			if response.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, response.Type)
			}

			if response.Message != tt.message {
				t.Errorf("Expected message %q, got %q", tt.message, response.Message)
			}
		})
	}
}

func TestFormatResponse(t *testing.T) {
	tests := []struct {
		name           string
		index          int
		status         int
		response       []byte
		expectedFormat string
	}{
		{
			name:           "Simple response",
			index:          0,
			status:         200,
			response:       []byte("OK"),
			expectedFormat: "0-200 - OK",
		},
		{
			name:           "JSON response",
			index:          5,
			status:         201,
			response:       []byte(`{"id":"123","status":"created"}`),
			expectedFormat: `5-201 - {"id":"123","status":"created"}`,
		},
		{
			name:           "Error response",
			index:          10,
			status:         404,
			response:       []byte("Resource not found"),
			expectedFormat: "10-404 - Resource not found",
		},
		{
			name:           "Empty response",
			index:          1,
			status:         204,
			response:       []byte(""),
			expectedFormat: "1-204 - ",
		},
		{
			name:           "Large index",
			index:          9999,
			status:         200,
			response:       []byte("test"),
			expectedFormat: "9999-200 - test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatResponse(tt.index, tt.status, tt.response)

			if result != tt.expectedFormat {
				t.Errorf("Expected format %q, got %q", tt.expectedFormat, result)
			}

			// Verify format structure
			parts := strings.Split(result, " - ")
			if len(parts) != 2 {
				t.Errorf("Expected format 'index-status - body', got %q", result)
			}

			indexStatus := strings.Split(parts[0], "-")
			if len(indexStatus) != 2 {
				t.Errorf("Expected 'index-status' format, got %q", parts[0])
			}
		})
	}
}

func TestLoadClient(t *testing.T) {
	client := loadClient()

	if client == nil {
		t.Fatal("loadClient returned nil")
	}

	if client.Transport == nil {
		t.Fatal("Client transport should not be nil")
	}

	// Verify TLS configuration
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

func TestCreateHttpServiceInner(t *testing.T) {
	config := model.Config{
		ApiEndpoint: "https://api.example.com",
		Method:      "POST",
	}

	tests := []struct {
		name         string
		args         model.CommandLineArgs
		expectedType string
	}{
		{
			name: "Dry run creates mock service",
			args: model.CommandLineArgs{
				DryRun: true,
			},
			expectedType: "*service.HttpServiceMock",
		},
		{
			name: "Non-dry run creates real service",
			args: model.CommandLineArgs{
				DryRun: false,
			},
			expectedType: "*service.HttpServiceReal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := createHttpService(config, tt.args)

			if service == nil {
				t.Fatal("createHttpService returned nil")
			}

			serviceType := fmt.Sprintf("%T", service)
			if serviceType != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, serviceType)
			}
		})
	}
}

// Helper function to create test HTTP requests
func createTestRequest(url string) *http.Request {
	req, _ := http.NewRequest("GET", url, bytes.NewBuffer([]byte("")))
	return req
}

func TestProcessService_Integration(t *testing.T) {
	// Integration test simulating real workflow
	config := model.Config{
		ApiEndpoint: "https://api.example.com",
		Method:      "POST",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	args := model.CommandLineArgs{
		DryRun:      false,
		SleepMillis: 0,
	}

	// Create mock responses
	mockResponses := []struct {
		body   []byte
		status int
	}{
		{[]byte(`{"result":"success"}`), 200},
		{[]byte(`{"error":"invalid"}`), 400},
		{[]byte(`{"result":"created"}`), 201},
	}

	callCount := 0
	mockService := &MockHttpService{
		callFunc: func(record http.Request) ([]byte, int, error) {
			if callCount >= len(mockResponses) {
				return nil, 0, errors.New("unexpected call")
			}
			resp := mockResponses[callCount]
			callCount++
			return resp.body, resp.status, nil
		},
	}

	service := &ProcessService{
		config:      config,
		args:        args,
		httpService: mockService,
	}

	// Create test requests
	requests := []http.Request{
		*createTestRequest("https://api.example.com/test1"),
		*createTestRequest("https://api.example.com/test2"),
		*createTestRequest("https://api.example.com/test3"),
	}

	respList, errList, err := service.ProcessAll(requests)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(respList) != 2 {
		t.Errorf("Expected 2 successful responses, got %d", len(respList))
	}

	if len(errList) != 1 {
		t.Errorf("Expected 1 error response, got %d", len(errList))
	}

	// Verify response content
	if !strings.Contains(respList[0], "success") {
		t.Errorf("First response should contain 'success', got: %s", respList[0])
	}

	if !strings.Contains(errList[0], "invalid") {
		t.Errorf("Error response should contain 'invalid', got: %s", errList[0])
	}
}
