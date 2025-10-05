package cmd

import (
	"batchRequestsRecover/internal/model"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Success(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	testConfig := model.Config{
		ApiEndpoint: "https://api.example.com",
		Method:      "POST",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		PathVars:     []string{"userId"},
		QueryVars:    []string{"status"},
		HasBody:      true,
		CSVDelimiter: "\t",
	}

	configBytes, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	err = os.WriteFile(configPath, configBytes, 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test loading the config
	config := loadConfig(configPath)

	if config.ApiEndpoint != testConfig.ApiEndpoint {
		t.Errorf("ApiEndpoint = %v, want %v", config.ApiEndpoint, testConfig.ApiEndpoint)
	}
	if config.Method != testConfig.Method {
		t.Errorf("Method = %v, want %v", config.Method, testConfig.Method)
	}
	if config.HasBody != testConfig.HasBody {
		t.Errorf("HasBody = %v, want %v", config.HasBody, testConfig.HasBody)
	}
	if config.CSVDelimiter != testConfig.CSVDelimiter {
		t.Errorf("CsvDelimiter = %v, want %v", config.CSVDelimiter, testConfig.CSVDelimiter)
	}
	if len(config.Headers) != len(testConfig.Headers) {
		t.Errorf("Headers length = %v, want %v", len(config.Headers), len(testConfig.Headers))
	}
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	config := loadConfig("nonexistent_file.json")

	// Should return empty config on error
	if config == nil {
		t.Error("Expected non-nil config on error")
	}

	// Empty config should have empty values
	if config.ApiEndpoint != "" {
		t.Errorf("Expected empty ApiEndpoint, got %v", config.ApiEndpoint)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid_config.json")

	// Write invalid JSON
	err := os.WriteFile(configPath, []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}

	config := loadConfig(configPath)

	// Should return empty config on parsing error
	if config == nil {
		t.Error("Expected non-nil config on error")
	}
	if config.ApiEndpoint != "" {
		t.Errorf("Expected empty ApiEndpoint on invalid JSON, got %v", config.ApiEndpoint)
	}
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "empty_config.json")

	// Write empty file
	err := os.WriteFile(configPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write empty config file: %v", err)
	}

	config := loadConfig(configPath)

	if config == nil {
		t.Error("Expected non-nil config on empty file")
	}
}

func TestLoadConfig_ComplexConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "complex_config.json")

	testConfig := model.Config{
		ApiEndpoint: "https://api.example.com/v2",
		Method:      "PUT",
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer token123",
			"X-Custom":      "value",
		},
		PathVars:     []string{"userId", "resourceId", "itemId"},
		QueryVars:    []string{"status", "type", "format"},
		HasBody:      true,
		CSVDelimiter: ",",
	}

	configBytes, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal complex config: %v", err)
	}

	err = os.WriteFile(configPath, configBytes, 0644)
	if err != nil {
		t.Fatalf("Failed to write complex config file: %v", err)
	}

	config := loadConfig(configPath)

	if len(config.PathVars) != 3 {
		t.Errorf("PathVars length = %v, want 3", len(config.PathVars))
	}
	if len(config.QueryVars) != 3 {
		t.Errorf("QueryVars length = %v, want 3", len(config.QueryVars))
	}
	if len(config.Headers) != 3 {
		t.Errorf("Headers length = %v, want 3", len(config.Headers))
	}
}

func TestCheckAndParseArgs_MissingInputFile(t *testing.T) {
	// Save original os.Args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Save original exit function
	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()

	// Mock os.Exit
	var exitCode int
	osExit = func(code int) {
		exitCode = code
	}

	// Reset flags before test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Set args without inputFile
	os.Args = []string{"cmd"}

	checkAndParseArgs()

	if exitCode != 1 {
		t.Errorf("Expected exit code 1 when inputFile is missing, got %d", exitCode)
	}
}

func TestCheckAndParseArgs_AllDefaults(t *testing.T) {
	// Save original os.Args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Save original exit function
	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()

	// Mock os.Exit to prevent actual exit
	osExit = func(code int) {}

	// Reset flags before test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Set args with only required inputFile
	os.Args = []string{"cmd", "-inputFile=test.csv"}

	args := checkAndParseArgs()

	if args.CSVFilePath != "test.csv" {
		t.Errorf("CSVFilePath = %v, want test.csv", args.CSVFilePath)
	}
	if args.ConfigFilePath != "config.json" {
		t.Errorf("ConfigFilePath = %v, want config.json", args.ConfigFilePath)
	}
	if args.DryRun != true {
		t.Errorf("DryRun = %v, want true", args.DryRun)
	}
	if args.SleepSeconds != 1 {
		t.Errorf("SleepSeconds = %v, want 1", args.SleepSeconds)
	}
}

func TestCheckAndParseArgs_CustomValues(t *testing.T) {
	// Save original os.Args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Save original exit function
	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()

	// Mock os.Exit to prevent actual exit
	osExit = func(code int) {}

	// Reset flags before test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Set args with custom values
	os.Args = []string{"cmd", "-inputFile=data.tsv", "-configPath=custom.json", "-dry=false", "-sleep=5"}

	args := checkAndParseArgs()

	if args.CSVFilePath != "data.tsv" {
		t.Errorf("CSVFilePath = %v, want data.tsv", args.CSVFilePath)
	}
	if args.ConfigFilePath != "custom.json" {
		t.Errorf("ConfigFilePath = %v, want custom.json", args.ConfigFilePath)
	}
	if args.DryRun != false {
		t.Errorf("DryRun = %v, want false", args.DryRun)
	}
	if args.SleepSeconds != 5 {
		t.Errorf("SleepSeconds = %v, want 5", args.SleepSeconds)
	}
}

func TestCheckAndParseArgs_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected model.CommandLineArgs
	}{
		{
			name: "Zero sleep time",
			args: []string{"cmd", "-inputFile=test.csv", "-sleep=0"},
			expected: model.CommandLineArgs{
				CSVFilePath:    "test.csv",
				ConfigFilePath: "config.json",
				DryRun:         true,
				SleepSeconds:   0,
			},
		},
		{
			name: "Long sleep time",
			args: []string{"cmd", "-inputFile=test.csv", "-sleep=3600"},
			expected: model.CommandLineArgs{
				CSVFilePath:    "test.csv",
				ConfigFilePath: "config.json",
				DryRun:         true,
				SleepSeconds:   3600,
			},
		},
		{
			name: "File with absolute path",
			args: []string{"cmd", "-inputFile=/absolute/path/to/file.csv"},
			expected: model.CommandLineArgs{
				CSVFilePath:    "/absolute/path/to/file.csv",
				ConfigFilePath: "config.json",
				DryRun:         true,
				SleepSeconds:   1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original os.Args and restore after test
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Save original exit function
			oldOsExit := osExit
			defer func() { osExit = oldOsExit }()

			// Mock os.Exit to prevent actual exit
			osExit = func(code int) {}

			// Reset flags before test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			os.Args = tt.args

			args := checkAndParseArgs()

			if args.CSVFilePath != tt.expected.CSVFilePath {
				t.Errorf("CSVFilePath = %v, want %v", args.CSVFilePath, tt.expected.CSVFilePath)
			}
			if args.ConfigFilePath != tt.expected.ConfigFilePath {
				t.Errorf("ConfigFilePath = %v, want %v", args.ConfigFilePath, tt.expected.ConfigFilePath)
			}
			if args.DryRun != tt.expected.DryRun {
				t.Errorf("DryRun = %v, want %v", args.DryRun, tt.expected.DryRun)
			}
			if args.SleepSeconds != tt.expected.SleepSeconds {
				t.Errorf("SleepSeconds = %v, want %v", args.SleepSeconds, tt.expected.SleepSeconds)
			}
		})
	}
}
