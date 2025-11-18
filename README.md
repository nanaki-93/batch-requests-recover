# Batch Requests Recover

A command-line tool for processing batch HTTP requests from CSV/TSV files. 
This tool is designed to recover or replay HTTP requests in bulk, with support for custom configurations, dry-run mode, and detailed response logging.

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

- Valid configuration file (JSON format)
- Input file with request data (CSV or TSV format)

## Usage
- The directory contains the executable binary, an example config file, and sample input data.

### Basic Command
```
bash
./batch-requests-recover -inputFile=<path-to-csv> [options]
```
### Command-Line Flags

| Flag          | Description                                      | Default       | Required |
|---------------|--------------------------------------------------|---------------|----------|
| `-inputFile`  | Path to CSV/TSV input file                       | -             | ✅ Yes   |
| `-configPath` | Path to configuration file                       | `config.json` | No       |
| `-dry`        | Enable dry-run mode (no actual requests)         | `true`        | No       |
| `-sleep`      | Sleep duration in milliseconds between requests  | `1000`        | No       |

### Example Commands
```
bash
# Process requests with default config
./batch-requests-recover -inputFile=input.tsv

# Production run with custom config and rate limiting
./batch-requests-recover -inputFile=data.csv -configPath=prod-config.json -dry=false -sleep=2000

# Dry run to test configuration
./batch-requests-recover -inputFile=test.tsv -dry=true
```
## Configuration

### Config File Structure (`config.json`)
```
json
{
"api_endpoint": "https://api.example.com/{userId}/{resourceId}",
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

- **api_endpoint**: Base URL for API requests, with path variables in {}
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
  "api_endpoint": "https://api.example.com/{userId}/{resourceId}",
  "method": "POST",
  "headers": {
    "Content-Type": "application/json"
  },
  "csv_delimiter": "\t",
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

## Best Practices

1. **Always test with dry-run first** - Validate your configuration before making real requests
2. **Use appropriate sleep intervals** - Respect API rate limits with the `-sleep` flag
3. **Monitor output files** - Check `.err` files for failed requests



