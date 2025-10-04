package main

// Config represents the structure for configuration related to API calls.
// ApiEndpoint specifies the API endpoint URL.
// Method defines the HTTP method for the API request.
// Headers contains the key-value pairs of HTTP headers.
// PathVars holds the dynamic segments for the URL path.
// QueryVars represents the query parameters in the request URL.
// HasBody indicates whether the request includes a payload body.
// The order in the csv file is important.
// The first n columns are the PathVars, the next n columns are the QueryVars,
// and the last column is the body, if the request has a body (hasBody = true).
type Config struct {
	ApiEndpoint string            `json:"api_endpoint"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	PathVars    []string          `json:"path_vars"`
	QueryVars   []string          `json:"query_vars"`
	HasBody     bool              `json:"has_body"`
}
