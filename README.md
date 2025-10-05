# Batch Requests Recover

A Go-based command-line tool for processing batch HTTP requests from CSV/TSV files. This tool is designed to recover or replay HTTP requests in bulk, with support for custom configurations, dry-run mode, and detailed response logging.

## Features

- 📁 **CSV/TSV File Processing** - Read and parse delimited files containing request parameters
- 🔧 **Configurable Requests** - Define API endpoints, HTTP methods, headers, and parameters via JSON config
- 🧪 **Dry Run Mode** - Test your configuration without making actual HTTP requests
- ⏱️ **Rate Limiting** - Control request frequency with configurable sleep intervals
- 📝 **Response Logging** - Separate successful responses and errors into distinct files
- 🔒 **TLS Support** - Handle HTTPS requests with custom TLS configuration
- 🌐 **Flexible URL Construction** - Support for path variables and query parameters
- 📊 **UTF-8 BOM Handling** - Automatically removes UTF-8 BOM from input files

## Prerequisites

- Go 1.24 or higher
- Valid configuration file (JSON format)
- Input file with request data (CSV or TSV format)

## Installation
```
bash
# Clone the repository
git clone <repository-url>
cd batch-requests-recover

# Build the application
go build -o batch-requests-recover

# Or run directly
go run main.go [flags]
```
## Usage

### Basic Command
```
bash
./batch-requests-recover -inputFile=<path-to-csv> [options]
```
### Command-Line Flags

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `-inputFile` | Path to CSV/TSV input file | - | ✅ Yes |
| `-configPath` | Path to configuration file | `config.json` | No |
| `-dry` | Enable dry-run mode (no actual requests) | `true` | No |
| `-sleep` | Sleep duration in seconds between requests | `1` | No |

### Example Commands
```
bash
# Process requests with default config
./batch-requests-recover -inputFile=input.tsv

# Production run with custom config and rate limiting
./batch-requests-recover -inputFile=data.csv -configPath=prod-config.json -dry=false -sleep=2

# Dry run to test configuration
./batch-requests-recover -inputFile=test.tsv -dry=true
```
## Configuration

### Config File Structure (`config.json`)
```
json
{
"api_endpoint": "https://api.example.com",
"method": "POST",
"headers": {
"Content-Type": "application/json",
"Authorization": "Bearer YOUR_TOKEN"
},
"path_vars": ["userId", "resourceId"],
"query_vars": ["status", "type"],
"has_body": true,
"csv_delimiter": "\t",
}
```
### Configuration Parameters

- **api_endpoint**: Base URL for API requests
- **method**: HTTP method (GET, POST, PUT, DELETE, etc.)
- **headers**: Map of HTTP headers to include in requests
- **path_vars**: List of column names used as path variables
- **query_vars**: List of column names used as query parameters
- **has_body**: Whether requests include a body (last column)
- **csv_delimiter**: Field delimiter character (default: tab)

## Input File Format

The CSV/TSV file should follow this column order:

1. **Path Variables** - Columns matching `path_vars` in config
2. **Query Variables** - Columns matching `query_vars` in config
3. **Request Body** (optional) - Last column if `has_body` is `true`

### Example Input File (`input.tsv`)
```
tsv
userId123	resource456	active	user	{"firstName":"John","lastName":"Doe"}
userId789	resource012	inactive	admin	{"firstName":"Jane","lastName":"Smith"}
```
With this config:
```json
{
  "api_endpoint": "https://api.example.com",
  "path_vars": ["userId", "resourceId"],
  "query_vars": ["status", "type"],
  "has_body": true
}
```
```


Generates requests like:
```
POST https://api.example.com/userId123/resource456?status=active&type=user
Body: {"firstName":"John","lastName":"Doe"}
```


## Output Files

After processing, the tool generates two files:

- **`<inputFile>.resp`** - Contains successful responses (HTTP 2xx)
- **`<inputFile>.err`** - Contains error responses (non-2xx status codes)

### Output Format

Each line follows this pattern:
```
<index>-<status_code> - <response_body>
```


Example:
```
0-200 - {"success": true, "id": "123"}
1-201 - {"success": true, "id": "456"}
```


## Project Structure

```
batch-requests-recover/
├── cmd/
│   └── root.go              # Main command logic
├── internal/
│   ├── model/
│   │   └── model.go         # Data models and structs
│   ├── service/
│   │   ├── csv.go           # CSV file operations
│   │   ├── http.go          # HTTP client and requests
│   │   └── processor.go     # Request processing logic
│   └── util/
│       └── common.go        # Utility functions
├── config.json              # Default configuration
├── main.go                  # Application entry point
└── README.md               # This file
```


## Error Handling

The application handles various error scenarios:

- Missing required command-line arguments
- Invalid or missing configuration file
- File read/write errors
- CSV parsing errors
- HTTP request failures
- Invalid response handling

Errors are logged to the console and error responses are saved to `<inputFile>.err`.

## Dry Run Mode

Use dry-run mode to validate your configuration without making actual HTTP requests:

```shell script
./batch-requests-recover -inputFile=input.tsv -dry=true
```


Dry-run output shows:
- Request URL
- HTTP method
- Headers (if applicable)

## Best Practices

1. **Always test with dry-run first** - Validate your configuration before making real requests
2. **Use appropriate sleep intervals** - Respect API rate limits with the `-sleep` flag
3. **Monitor output files** - Check `.err` files for failed requests
4. **Secure your tokens** - Keep configuration files with sensitive data out of version control
5. **Backup your data** - Keep copies of input files before processing

## Troubleshooting

### Common Issues

**"inputFile is required" error**
- Ensure you provide the `-inputFile` flag with a valid path

**Empty output files**
- Check if input file has valid data
- Verify CSV delimiter matches your file format
- Enable dry-run mode to debug request construction

**All requests failing**
- Verify API endpoint is accessible
- Check authentication headers in config
- Review network/firewall settings

**TLS/SSL errors**
- The tool uses `InsecureSkipVerify: true` by default
- Modify `service.LoadClient()` for production TLS verification

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

