package main

type Config struct {
	ApiEndpoint string            `json:"api_endpoint"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	CsvColumns  []Column          `json:"csv_columns"`
	HasBody     bool              `json:"has_body"`
}

type Column struct {
	Name string
	Type ColumnType
}
type ColumnType string

const (
	Path  ColumnType = "path"
	Query ColumnType = "query"
	Body  ColumnType = "body"
)
