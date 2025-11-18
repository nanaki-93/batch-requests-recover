package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRemoveBOM(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "Content with UTF-8 BOM",
			input:    []byte{0xEF, 0xBB, 0xBF, 'H', 'e', 'l', 'l', 'o'},
			expected: []byte{'H', 'e', 'l', 'l', 'o'},
		},
		{
			name:     "Content without BOM",
			input:    []byte{'H', 'e', 'l', 'l', 'o'},
			expected: []byte{'H', 'e', 'l', 'l', 'o'},
		},
		{
			name:     "Empty content",
			input:    []byte{},
			expected: []byte{},
		},
		{
			name:     "Only BOM",
			input:    []byte{0xEF, 0xBB, 0xBF},
			expected: []byte{},
		},
		{
			name:     "Content shorter than BOM",
			input:    []byte{0xEF, 0xBB},
			expected: []byte{0xEF, 0xBB},
		},
		{
			name:     "Content starting with partial BOM pattern",
			input:    []byte{0xEF, 0xBB, 0x00, 'T', 'e', 's', 't'},
			expected: []byte{0xEF, 0xBB, 0x00, 'T', 'e', 's', 't'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveBOM(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("RemoveBOM() length = %v, expected %v", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("RemoveBOM() = %v, expected %v", result, tt.expected)
					return
				}
			}
		})
	}
}

func TestTrimQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "String with single quotes",
			input:    "'hello'",
			expected: "hello",
		},
		{
			name:     "String with double quotes",
			input:    `"hello"`,
			expected: "hello",
		},
		{
			name:     "String with whitespace and single quotes",
			input:    "  'hello'  ",
			expected: "hello",
		},
		{
			name:     "String with whitespace and double quotes",
			input:    `  "hello"  `,
			expected: "hello",
		},
		{
			name:     "String without quotes",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "String with only quotes",
			input:    "''",
			expected: "",
		},
		{
			name:     "String with mixed quotes",
			input:    `'"hello"'`,
			expected: "hello",
		},
		{
			name:     "String with quotes only on one side",
			input:    "'hello",
			expected: "hello",
		},
		{
			name:     "String with internal quotes",
			input:    "'hello'world'",
			expected: "hello'world",
		},
		{
			name:     "String with only whitespace",
			input:    "   ",
			expected: "",
		},
		{
			name:     "String with whitespace inside quotes",
			input:    "'  hello world  '",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TrimQuotes(tt.input)
			if result != tt.expected {
				t.Errorf("TrimQuotes() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestDelayFor(t *testing.T) {
	tests := []struct {
		name        string
		sleep       int
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{
			name:        "One second delay",
			sleep:       1000,
			minDuration: 1 * time.Second,
			maxDuration: 1*time.Second + 100*time.Millisecond,
		},
		{
			name:        "Zero seconds delay",
			sleep:       0,
			minDuration: 0,
			maxDuration: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			DelayFor(tt.sleep)
			elapsed := time.Since(start)

			if elapsed < tt.minDuration {
				t.Errorf("DelayFor(%d) took %v, expected at least %v", tt.sleep, elapsed, tt.minDuration)
			}
			if elapsed > tt.maxDuration {
				t.Errorf("DelayFor(%d) took %v, expected at most %v", tt.sleep, elapsed, tt.maxDuration)
			}
		})
	}
}

func TestWriteResponses(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name            string
		inputFilePath   string
		respList        []string
		suffix          string
		expectedContent string
	}{
		{
			name:            "Write single response",
			inputFilePath:   filepath.Join(tempDir, "test1.txt"),
			respList:        []string{"response1"},
			suffix:          ".resp",
			expectedContent: "response1",
		},
		{
			name:            "Write multiple responses",
			inputFilePath:   filepath.Join(tempDir, "test2.txt"),
			respList:        []string{"response1", "response2", "response3"},
			suffix:          ".err",
			expectedContent: "response1\nresponse2\nresponse3",
		},
		{
			name:            "Write empty list",
			inputFilePath:   filepath.Join(tempDir, "test3.txt"),
			respList:        []string{},
			suffix:          ".resp",
			expectedContent: "",
		},
		{
			name:            "Write with custom suffix",
			inputFilePath:   filepath.Join(tempDir, "test4.csv"),
			respList:        []string{"data1", "data2"},
			suffix:          ".output",
			expectedContent: "data1\ndata2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			WriteResponses(tt.inputFilePath, tt.respList, tt.suffix)

			// Check if file was created
			outputFile := tt.inputFilePath + tt.suffix
			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Errorf("WriteResponses() failed to create file: %v", err)
				return
			}

			// Verify content
			if string(content) != tt.expectedContent {
				t.Errorf("WriteResponses() wrote %q, expected %q", string(content), tt.expectedContent)
			}
		})
	}
}

func TestWriteResponses_LargeList(t *testing.T) {
	tempDir := t.TempDir()
	inputFilePath := filepath.Join(tempDir, "large_test.txt")

	// Create a large list of responses
	largeList := make([]string, 1000)
	for i := range largeList {
		largeList[i] = strings.Repeat("x", 100)
	}

	WriteResponses(inputFilePath, largeList, ".resp")

	outputFile := inputFilePath + ".resp"
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) != 1000 {
		t.Errorf("Expected 1000 lines, got %d", len(lines))
	}
}

func TestWriteResponses_SpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()
	inputFilePath := filepath.Join(tempDir, "special_test.txt")

	respList := []string{
		"Response with 日本語",
		"Response with émojis 🎉🎊",
		"Response with\ttabs",
		"Response with special chars: <>&\"'",
	}

	WriteResponses(inputFilePath, respList, ".resp")

	outputFile := inputFilePath + ".resp"
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	expected := strings.Join(respList, "\n")
	if string(content) != expected {
		t.Errorf("WriteResponses() incorrectly handled special characters")
	}
}
