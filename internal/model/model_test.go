package model

import (
	"errors"
	"io"
	"testing"
)

func TestConfig_GetTotalColumns(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected int
	}{
		{
			name: "No path vars, no query vars, no body",
			config: Config{
				PathVars:  []string{},
				QueryVars: []string{},
				HasBody:   false,
			},
			expected: 0,
		},
		{
			name: "One path var only",
			config: Config{
				PathVars:  []string{"userId"},
				QueryVars: []string{},
				HasBody:   false,
			},
			expected: 1,
		},
		{
			name: "Multiple path vars and query vars",
			config: Config{
				PathVars:  []string{"userId", "resourceId"},
				QueryVars: []string{"status", "type"},
				HasBody:   false,
			},
			expected: 4,
		},
		{
			name: "Path vars, query vars, and body",
			config: Config{
				PathVars:  []string{"userId", "resourceId"},
				QueryVars: []string{"status"},
				HasBody:   true,
			},
			expected: 4,
		},
		{
			name: "Only body",
			config: Config{
				PathVars:  []string{},
				QueryVars: []string{},
				HasBody:   true,
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetTotalColumns()
			if result != tt.expected {
				t.Errorf("Expected %d columns, got %d", tt.expected, result)
			}
		})
	}
}

func TestConfig_GetPathVars(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		row           []string
		expected      string
		expectedError error
	}{
		{
			name: "Single path variable",
			config: Config{
				ApiEndpoint: "/{userId}",
				PathVars:    []string{"userId"},
				QueryVars:   []string{},
				HasBody:     false,
			},
			row:           []string{"123"},
			expected:      "/123",
			expectedError: nil,
		},
		{
			name: "Multiple path variables",
			config: Config{
				ApiEndpoint: "/{userId}/test/{resourceId}",
				PathVars:    []string{"userId", "resourceId"},
				QueryVars:   []string{},
				HasBody:     false,
			},
			row:           []string{"user123", "res456"},
			expected:      "/user123/test/res456",
			expectedError: nil,
		},
		{
			name: "No path variables",
			config: Config{
				PathVars:  []string{},
				QueryVars: []string{},
				HasBody:   false,
			},
			row:           []string{},
			expected:      "",
			expectedError: nil,
		},
		{
			name: "Path variables with quotes",
			config: Config{
				ApiEndpoint: "/{userId}",
				PathVars:    []string{"userId"},
				QueryVars:   []string{},
				HasBody:     false,
			},
			row:           []string{`"user123"`},
			expected:      "/user123",
			expectedError: nil,
		},
		{
			name: "Path variables with mixed data",
			config: Config{
				ApiEndpoint: "/{userId}/first/{age}/second/{status}",
				PathVars:    []string{"userId", "age", "status"},
				QueryVars:   []string{"type"},
				HasBody:     false,
			},
			row:           []string{"user1", "25", "active", "premium"},
			expected:      "/user1/first/25/second/active",
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.WithPathVars(tt.row)
			if !errors.Is(err, tt.expectedError) {
				t.Errorf("Expected error %v, got %v", tt.expectedError, err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestConfig_GetQueryVars(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		row           []string
		expected      string
		expectedError error
	}{
		{
			name: "No query variables",
			config: Config{
				PathVars:  []string{"userId"},
				QueryVars: []string{},
				HasBody:   false,
			},
			row:           []string{"123"},
			expected:      "",
			expectedError: nil,
		},
		{
			name: "Single query variable",
			config: Config{
				PathVars:  []string{"userId"},
				QueryVars: []string{"status"},
				HasBody:   false,
			},
			row:           []string{"123", "active"},
			expected:      "?status=active",
			expectedError: nil,
		},
		{
			name: "Multiple query variables",
			config: Config{
				PathVars:  []string{"userId"},
				QueryVars: []string{"status", "type", "limit"},
				HasBody:   false,
			},
			row:           []string{"123", "active", "premium", "10"},
			expected:      "?status=active&type=premium&limit=10",
			expectedError: nil,
		},
		{
			name: "Query variables with quotes",
			config: Config{
				PathVars:  []string{},
				QueryVars: []string{"name"},
				HasBody:   false,
			},
			row:           []string{`"test"`},
			expected:      "?name=test",
			expectedError: nil,
		},
		{
			name: "Query variables after path variables",
			config: Config{
				PathVars:  []string{"userId", "resourceId"},
				QueryVars: []string{"status", "type"},
				HasBody:   false,
			},
			row:           []string{"user1", "res1", "active", "premium"},
			expected:      "?status=active&type=premium",
			expectedError: nil,
		},
		{
			name: "Empty query values",
			config: Config{
				PathVars:  []string{},
				QueryVars: []string{"param1", "param2"},
				HasBody:   false,
			},
			row:           []string{"", "value2"},
			expected:      "?param1=&param2=value2",
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.GetQueryVars(tt.row)
			if !errors.Is(err, tt.expectedError) {
				t.Errorf("Expected error %v, got %v", tt.expectedError, err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestNewCsvRequest(t *testing.T) {
	tests := []struct {
		name           string
		options        []CsvRequestOption
		expectedMethod string
		expectedURL    string
		hasBody        bool
	}{
		{
			name:           "Default CsvRequest",
			options:        []CsvRequestOption{},
			expectedMethod: "GET",
			expectedURL:    "",
			hasBody:        false,
		},
		{
			name: "CsvRequest with URL only",
			options: []CsvRequestOption{
				WithRequestUrl("https://api.example.com/test"),
			},
			expectedMethod: "GET",
			expectedURL:    "https://api.example.com/test",
			hasBody:        false,
		},
		{
			name: "CsvRequest with method",
			options: []CsvRequestOption{
				WithMethod("POST"),
				WithRequestUrl("https://api.example.com/create"),
			},
			expectedMethod: "POST",
			expectedURL:    "https://api.example.com/create",
			hasBody:        false,
		},
		{
			name: "CsvRequest with body",
			options: []CsvRequestOption{
				WithMethod("POST"),
				WithRequestUrl("https://api.example.com/create"),
				WithBody(`{"key":"value"}`),
			},
			expectedMethod: "POST",
			expectedURL:    "https://api.example.com/create",
			hasBody:        true,
		},
		{
			name: "CsvRequest with headers",
			options: []CsvRequestOption{
				WithHeaders(map[string]string{
					"Content-Type": "application/json",
				}),
			},
			expectedMethod: "GET",
			expectedURL:    "",
			hasBody:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewCsvRequest(tt.options...)

			if req == nil {
				t.Fatal("NewCsvRequest returned nil")
			}

			if req.Method != tt.expectedMethod {
				t.Errorf("Expected method %s, got %s", tt.expectedMethod, req.Method)
			}

			if req.RequestUrl != tt.expectedURL {
				t.Errorf("Expected URL %s, got %s", tt.expectedURL, req.RequestUrl)
			}

			if tt.hasBody && req.Body == nil {
				t.Error("Expected body to be set, but it's nil")
			}

			if !tt.hasBody && req.Body != nil {
				t.Error("Expected body to be nil, but it's set")
			}

			if req.Headers == nil {
				t.Error("Headers should be initialized (not nil)")
			}
		})
	}
}

func TestWithRequestUrl(t *testing.T) {
	url := "https://api.example.com/test"
	req := NewCsvRequest(WithRequestUrl(url))

	if req.RequestUrl != url {
		t.Errorf("Expected URL %s, got %s", url, req.RequestUrl)
	}
}

func TestWithMethod(t *testing.T) {
	tests := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range tests {
		t.Run(method, func(t *testing.T) {
			req := NewCsvRequest(WithMethod(method))

			if req.Method != method {
				t.Errorf("Expected method %s, got %s", method, req.Method)
			}
		})
	}
}

func TestWithHeaders(t *testing.T) {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer token123",
		"X-Custom":      "custom-value",
	}

	req := NewCsvRequest(WithHeaders(headers))

	if req.Headers == nil {
		t.Fatal("Headers should not be nil")
	}

	for key, expectedValue := range headers {
		if actualValue, exists := req.Headers[key]; !exists {
			t.Errorf("Header %s not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("Header %s: expected %s, got %s", key, expectedValue, actualValue)
		}
	}
}

func TestWithBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "Simple JSON body",
			body: `{"key":"value"}`,
		},
		{
			name: "Complex JSON body",
			body: `{"name":"John","age":30,"active":true}`,
		},
		{
			name: "Empty body",
			body: "",
		},
		{
			name: "Text body",
			body: "plain text content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewCsvRequest(WithBody(tt.body))

			if req.Body == nil {
				t.Fatal("Body should not be nil")
			}

			// Read the body
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("Failed to read body: %v", err)
			}

			if string(bodyBytes) != tt.body {
				t.Errorf("Expected body %q, got %q", tt.body, string(bodyBytes))
			}
		})
	}
}

func TestResponse_Type(t *testing.T) {
	tests := []struct {
		name     string
		respType ResponseType
		expected ResponseType
	}{
		{
			name:     "SUCCESS type",
			respType: SUCCESS,
			expected: SUCCESS,
		},
		{
			name:     "ERROR type",
			respType: ERROR,
			expected: ERROR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := Response{
				Type:    tt.respType,
				Message: "test message",
			}

			if resp.Type != tt.expected {
				t.Errorf("Expected type %v, got %v", tt.expected, resp.Type)
			}
		})
	}
}

func TestResponseType_Values(t *testing.T) {
	// Verify the constant values
	if SUCCESS != 0 {
		t.Errorf("Expected SUCCESS to be 0, got %d", SUCCESS)
	}

	if ERROR != 1 {
		t.Errorf("Expected ERROR to be 1, got %d", ERROR)
	}
}

func TestCsvRequest_CompleteWorkflow(t *testing.T) {
	// Integration test combining all options
	req := NewCsvRequest(
		WithMethod("POST"),
		WithRequestUrl("https://api.example.com/users/123"),
		WithHeaders(map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer token",
		}),
		WithBody(`{"name":"test","value":123}`),
	)

	if req == nil {
		t.Fatal("Request is nil")
	}

	// Verify method
	if req.Method != "POST" {
		t.Errorf("Expected method POST, got %s", req.Method)
	}

	// Verify URL
	expectedURL := "https://api.example.com/users/123"
	if req.RequestUrl != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, req.RequestUrl)
	}

	// Verify headers
	if req.Headers["Content-Type"] != "application/json" {
		t.Error("Content-Type header not set correctly")
	}
	if req.Headers["Authorization"] != "Bearer token" {
		t.Error("Authorization header not set correctly")
	}

	// Verify body
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}
	expectedBody := `{"name":"test","value":123}`
	if string(bodyBytes) != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, string(bodyBytes))
	}
}

func TestConfig_GetPathVars_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		row           []string
		expected      string
		expectedError string
	}{
		{
			name: "More path vars than columns",
			config: Config{
				PathVars:  []string{"var1", "var2", "var3"},
				QueryVars: []string{},
				HasBody:   false,
			},
			row:           []string{"val1"},
			expected:      "",
			expectedError: "not enough columns in the csv",
		},
		{
			name: "Special characters in path",
			config: Config{
				ApiEndpoint: "/{userId}",
				PathVars:    []string{"userId"},
				QueryVars:   []string{},
				HasBody:     false,
			},
			row:      []string{"user@123"},
			expected: "/user@123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.WithPathVars(tt.row)
			if tt.expectedError != "" && err.Error() != tt.expectedError {
				t.Errorf("Expected error: %v, got: %v", tt.expectedError, err)
			}
			if result != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, result)
			}
		})
	}
}

func TestCommandLineArgs_Structure(t *testing.T) {
	args := CommandLineArgs{
		CSVFilePath:    "/path/to/file.csv",
		ConfigFilePath: "/path/to/config.json",
		DryRun:         true,
		SleepMillis:    5,
	}

	if args.CSVFilePath != "/path/to/file.csv" {
		t.Error("CSVFilePath not set correctly")
	}
	if args.ConfigFilePath != "/path/to/config.json" {
		t.Error("ConfigFilePath not set correctly")
	}
	if !args.DryRun {
		t.Error("DryRun should be true")
	}
	if args.SleepMillis != 5 {
		t.Error("SleepMillis not set correctly")
	}
}

func TestConfig_CompleteStructure(t *testing.T) {
	config := Config{
		ApiEndpoint: "https://api.example.com",
		Method:      "POST",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		PathVars:     []string{"userId", "resourceId"},
		QueryVars:    []string{"status", "type"},
		HasBody:      true,
		CSVDelimiter: "\t",
	}

	if config.ApiEndpoint != "https://api.example.com" {
		t.Error("ApiEndpoint not set correctly")
	}
	if config.Method != "POST" {
		t.Error("Method not set correctly")
	}
	if len(config.Headers) != 1 {
		t.Error("Headers not set correctly")
	}
	if len(config.PathVars) != 2 {
		t.Error("PathVars not set correctly")
	}
	if len(config.QueryVars) != 2 {
		t.Error("QueryVars not set correctly")
	}
	if !config.HasBody {
		t.Error("HasBody should be true")
	}
	if config.CSVDelimiter != "\t" {
		t.Error("CSVDelimiter not set correctly")
	}
}

func TestCsvRequest_MultipleOptionsOrder(t *testing.T) {
	// Test that options can be applied in any order
	req1 := NewCsvRequest(
		WithBody("body1"),
		WithMethod("POST"),
		WithRequestUrl("url1"),
	)

	req2 := NewCsvRequest(
		WithRequestUrl("url1"),
		WithMethod("POST"),
		WithBody("body1"),
	)

	if req1.Method != req2.Method {
		t.Error("Method should be the same regardless of option order")
	}
	if req1.RequestUrl != req2.RequestUrl {
		t.Error("URL should be the same regardless of option order")
	}
}
