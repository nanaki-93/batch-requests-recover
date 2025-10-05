package model

import (
	"batchRequestsRecover/internal/util"
	"bytes"
	"fmt"
	"io"
	"strings"
)

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
	ApiEndpoint  string            `json:"api_endpoint"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers"`
	PathVars     []string          `json:"path_vars"`
	QueryVars    []string          `json:"query_vars"`
	HasBody      bool              `json:"has_body"`
	CSVDelimiter string            `json:"csv_delimiter"`
}

type CommandLineArgs struct {
	CSVFilePath    string
	ConfigFilePath string
	DryRun         bool
	SleepSeconds   int
}

type CsvRequest struct {
	RequestUrl string
	Method     string
	Headers    map[string]string
	Body       io.Reader
}
type Response struct {
	Type    ResponseType
	Message string
}

type ResponseType int

const (
	SUCCESS ResponseType = iota
	ERROR
)

type CsvRequestOption func(*CsvRequest)

// NewCsvRequest creates a new CsvRequest with the given options
func NewCsvRequest(opts ...CsvRequestOption) *CsvRequest {
	// Set defaults
	req := &CsvRequest{
		Method:  "GET",
		Headers: make(map[string]string),
	}

	// Apply all options
	for _, opt := range opts {
		opt(req)
	}

	return req
}

// WithRequestUrl sets the request URL
func WithRequestUrl(url string) CsvRequestOption {
	return func(cr *CsvRequest) {
		cr.RequestUrl = url
	}
}

// WithMethod sets the HTTP method
func WithMethod(method string) CsvRequestOption {
	return func(cr *CsvRequest) {
		cr.Method = method
	}
}

// WithHeaders sets the request headers
func WithHeaders(headers map[string]string) CsvRequestOption {
	return func(cr *CsvRequest) {
		cr.Headers = headers
	}
}

// WithBody sets the request body from a string
func WithBody(body string) CsvRequestOption {
	return func(cr *CsvRequest) {
		cr.Body = bytes.NewBuffer([]byte(body))
	}
}

func (conf *Config) GetTotalColumns() int {

	totalColumns := len(conf.PathVars) + len(conf.QueryVars)
	if conf.HasBody {
		totalColumns++
	}
	return totalColumns
}

func (conf *Config) GetPathVars(row []string) string {
	var urlBuilder strings.Builder

	columnsToProcess := min(len(conf.PathVars), conf.GetTotalColumns())

	for j := 0; j < columnsToProcess; j++ {
		urlBuilder.WriteString("/")
		urlBuilder.WriteString(util.TrimQuotes(row[j]))
	}

	if len(conf.PathVars) > conf.GetTotalColumns() {
		fmt.Println("Too many columns in the csv:", row)
	}

	return urlBuilder.String()
}

func (conf *Config) GetQueryVars(row []string) string {

	if len(conf.QueryVars) == 0 {
		return ""
	}

	var urlBuilder strings.Builder
	urlBuilder.WriteString("?")

	pathVarsOffset := len(conf.PathVars)

	for j, queryVar := range conf.QueryVars {
		columnIndex := j + pathVarsOffset
		if columnIndex >= conf.GetTotalColumns() {
			fmt.Println("Too many columns in the csv:", row)
			break
		}

		if j > 0 {
			urlBuilder.WriteString("&")
		}
		urlBuilder.WriteString(util.TrimQuotes(queryVar))
		urlBuilder.WriteString("=")
		urlBuilder.WriteString(util.TrimQuotes(row[columnIndex]))
	}

	return urlBuilder.String()
}
