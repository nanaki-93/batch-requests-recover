package service

import (
	"batchRequestsRecover/internal/model"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewParserService(t *testing.T) {
	config := model.Config{
		ApiEndpoint: "https://api.example.com",
		Method:      "POST",
	}

	service := NewParserService(config)

	if service == nil {
		t.Fatal("NewParserService returned nil")
	}

	if service.config.ApiEndpoint != config.ApiEndpoint {
		t.Errorf("Expected ApiEndpoint %s, got %s", config.ApiEndpoint, service.config.ApiEndpoint)
	}
}

func TestParserService_isAEmptyRow(t *testing.T) {
	service := &ParserService{}

	tests := []struct {
		name     string
		row      []string
		expected bool
	}{
		{
			name:     "Empty slice",
			row:      []string{},
			expected: true,
		},
		{
			name:     "Single empty string",
			row:      []string{""},
			expected: true,
		},
		{
			name:     "Single whitespace string",
			row:      []string{"   "},
			expected: true,
		},
		{
			name:     "Non-empty row",
			row:      []string{"value1", "value2"},
			expected: false,
		},
		{
			name:     "Row with data",
			row:      []string{"test"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isAEmptyRow(tt.row)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for row %v", tt.expected, result, tt.row)
			}
		})
	}
}

func TestParserService_getReader(t *testing.T) {
	tests := []struct {
		name      string
		config    model.Config
		csvData   string
		wantComma rune
	}{
		{
			name: "Default tab delimiter",
			config: model.Config{
				CSVDelimiter: "",
			},
			csvData:   "col1\tcol2\tcol3",
			wantComma: '\t',
		},
		{
			name: "Comma delimiter",
			config: model.Config{
				CSVDelimiter: ",",
			},
			csvData:   "col1,col2,col3",
			wantComma: ',',
		},
		{
			name: "Pipe delimiter",
			config: model.Config{
				CSVDelimiter: "|",
			},
			csvData:   "col1|col2|col3",
			wantComma: '|',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ParserService{config: tt.config}
			reader := service.getReader(strings.NewReader(tt.csvData))

			if reader == nil {
				t.Fatal("getReader returned nil")
			}

			if reader.Comma != tt.wantComma {
				t.Errorf("Expected delimiter %q, got %q", tt.wantComma, reader.Comma)
			}

			if !reader.LazyQuotes {
				t.Error("Expected LazyQuotes to be true")
			}

			if !reader.TrimLeadingSpace {
				t.Error("Expected TrimLeadingSpace to be true")
			}
		})
	}
}

func TestParserService_createRequest(t *testing.T) {
	tests := []struct {
		name        string
		config      model.Config
		row         []string
		expectedURL string
		expectedErr bool
	}{
		{
			name: "Simple GET request without body",
			config: model.Config{
				ApiEndpoint: "https://api.example.com",
				Method:      "GET",
				PathVars:    []string{"userId"},
				QueryVars:   []string{"status"},
				HasBody:     false,
			},
			row:         []string{"123", "active"},
			expectedURL: "https://api.example.com/123?status=active",
			expectedErr: false,
		},
		{
			name: "POST request with body",
			config: model.Config{
				ApiEndpoint: "https://api.example.com",
				Method:      "POST",
				PathVars:    []string{"userId"},
				QueryVars:   []string{},
				HasBody:     true,
			},
			row:         []string{"456", `{"name":"test"}`},
			expectedURL: "https://api.example.com/456",
			expectedErr: false,
		},
		{
			name: "Request with multiple path and query vars",
			config: model.Config{
				ApiEndpoint: "https://api.example.com",
				Method:      "PUT",
				PathVars:    []string{"userId", "resourceId"},
				QueryVars:   []string{"status", "type"},
				HasBody:     false,
			},
			row:         []string{"user1", "res2", "active", "premium"},
			expectedURL: "https://api.example.com/user1/res2?status=active&type=premium",
			expectedErr: false,
		},
		{
			name: "Request with headers",
			config: model.Config{
				ApiEndpoint: "https://api.example.com",
				Method:      "POST",
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer token123",
				},
				PathVars:  []string{"id"},
				QueryVars: []string{},
				HasBody:   false,
			},
			row:         []string{"789"},
			expectedURL: "https://api.example.com/789",
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ParserService{config: tt.config}
			request, err := service.createRequest(tt.row)

			if tt.expectedErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if request == nil {
				t.Fatal("Request is nil")
			}

			if request.URL.String() != tt.expectedURL {
				t.Errorf("Expected URL %s, got %s", tt.expectedURL, request.URL.String())
			}

			if request.Method != tt.config.Method {
				t.Errorf("Expected method %s, got %s", tt.config.Method, request.Method)
			}

			// Check headers
			for key, value := range tt.config.Headers {
				if request.Header.Get(key) != value {
					t.Errorf("Expected header %s=%s, got %s", key, value, request.Header.Get(key))
				}
			}

			// Check body if present
			if tt.config.HasBody {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Fatalf("Error reading body: %v", err)
				}
				expectedBody := tt.row[len(tt.config.PathVars)+len(tt.config.QueryVars)]
				if string(body) != expectedBody {
					t.Errorf("Expected body %s, got %s", expectedBody, string(body))
				}
			}
		})
	}
}

func TestParserService_parse(t *testing.T) {
	tests := []struct {
		name          string
		config        model.Config
		csvContent    string
		expectedCount int
		expectedErr   bool
	}{
		{
			name: "Parse simple TSV file",
			config: model.Config{
				ApiEndpoint:  "https://api.example.com",
				Method:       "POST",
				PathVars:     []string{"userId"},
				QueryVars:    []string{"status"},
				HasBody:      true,
				CSVDelimiter: "\t",
			},
			csvContent:    "user1\tactive\t{\"name\":\"John\"}\nuser2\tinactive\t{\"name\":\"Jane\"}",
			expectedCount: 2,
			expectedErr:   false,
		},
		{
			name: "Parse CSV with empty lines",
			config: model.Config{
				ApiEndpoint:  "https://api.example.com",
				Method:       "GET",
				PathVars:     []string{"id"},
				QueryVars:    []string{},
				HasBody:      false,
				CSVDelimiter: ",",
			},
			csvContent:    "123\n\n456\n",
			expectedCount: 2,
			expectedErr:   false,
		},
		{
			name: "Parse CSV with comma delimiter",
			config: model.Config{
				ApiEndpoint:  "https://api.example.com",
				Method:       "GET",
				PathVars:     []string{"userId"},
				QueryVars:    []string{"type"},
				HasBody:      false,
				CSVDelimiter: ",",
			},
			csvContent:    "user1,premium\nuser2,basic",
			expectedCount: 2,
			expectedErr:   false,
		},
		{
			name: "Parse empty content",
			config: model.Config{
				ApiEndpoint:  "https://api.example.com",
				Method:       "GET",
				PathVars:     []string{"id"},
				QueryVars:    []string{},
				HasBody:      false,
				CSVDelimiter: "\t",
			},
			csvContent:    "",
			expectedCount: 0,
			expectedErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ParserService{config: tt.config}
			content := []byte(tt.csvContent)

			requests, err := service.parse(content)

			if tt.expectedErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(requests) != tt.expectedCount {
				t.Errorf("Expected %d requests, got %d", tt.expectedCount, len(requests))
			}
		})
	}
}

func TestParserService_readFile(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.csv")

	testContent := "col1\tcol2\tcol3\nval1\tval2\tval3"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		filePath    string
		expectError bool
	}{
		{
			name:        "Read existing file",
			filePath:    testFile,
			expectError: false,
		},
		{
			name:        "Read non-existent file",
			filePath:    filepath.Join(tmpDir, "nonexistent.csv"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ParserService{}
			content, err := service.readFile(tt.filePath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(content) == 0 {
				t.Error("Expected content but got empty")
			}
		})
	}
}

func TestParserService_readFile_WithBOM(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_bom.csv")

	// Create file with UTF-8 BOM
	bom := []byte{0xEF, 0xBB, 0xBF}
	testContent := []byte("col1\tcol2")
	contentWithBOM := append(bom, testContent...)

	err := os.WriteFile(testFile, contentWithBOM, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	service := &ParserService{}
	content, err := service.readFile(testFile)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify BOM was removed
	if bytes.HasPrefix(content, bom) {
		t.Error("BOM was not removed from content")
	}

	if !bytes.Equal(content, testContent) {
		t.Errorf("Expected content %v, got %v", testContent, content)
	}
}

func TestParserService_ReadAndParse(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_integration.tsv")

	testContent := "user1\t21\tM\t{\"firstName\":\"John\",\"lastName\":\"Doe\"}\n" +
		"user2\t22\tF\t{\"firstName\":\"Jane\",\"lastName\":\"Smith\"}"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := model.Config{
		ApiEndpoint:  "https://api.example.com",
		Method:       "POST",
		PathVars:     []string{"userId", "age", "gender"},
		QueryVars:    []string{},
		HasBody:      true,
		CSVDelimiter: "\t",
	}

	service := NewParserService(config)
	requests, err := service.ReadAndParse(testFile)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(requests) != 2 {
		t.Errorf("Expected 2 requests, got %d", len(requests))
	}

	// Verify first request
	if len(requests) > 0 {
		expectedURL := "https://api.example.com/user1/21/M"
		if requests[0].URL.String() != expectedURL {
			t.Errorf("Expected URL %s, got %s", expectedURL, requests[0].URL.String())
		}

		if requests[0].Method != "POST" {
			t.Errorf("Expected method POST, got %s", requests[0].Method)
		}

		body, _ := io.ReadAll(requests[0].Body)
		expectedBody := `{"firstName":"John","lastName":"Doe"}`
		if string(body) != expectedBody {
			t.Errorf("Expected body %s, got %s", expectedBody, string(body))
		}
	}
}

func TestCreateHttpRequest(t *testing.T) {
	tests := []struct {
		name        string
		csvRequest  *model.CsvRequest
		expectError bool
	}{
		{
			name: "Valid GET request",
			csvRequest: model.NewCsvRequest(
				model.WithMethod("GET"),
				model.WithRequestUrl("https://api.example.com/test"),
			),
			expectError: false,
		},
		{
			name: "Valid POST request with body",
			csvRequest: model.NewCsvRequest(
				model.WithMethod("POST"),
				model.WithRequestUrl("https://api.example.com/test"),
				model.WithBody(`{"key":"value"}`),
				model.WithHeaders(map[string]string{
					"Content-Type": "application/json",
				}),
			),
			expectError: false,
		},
		{
			name: "Invalid URL",
			csvRequest: model.NewCsvRequest(
				model.WithMethod("GET"),
				model.WithRequestUrl("://invalid-url"),
			),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := createHttpRequest(tt.csvRequest)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if request == nil {
				t.Fatal("Request is nil")
			}

			if request.Method != tt.csvRequest.Method {
				t.Errorf("Expected method %s, got %s", tt.csvRequest.Method, request.Method)
			}

			for key, value := range tt.csvRequest.Headers {
				if request.Header.Get(key) != value {
					t.Errorf("Expected header %s=%s, got %s", key, value, request.Header.Get(key))
				}
			}
		})
	}
}
