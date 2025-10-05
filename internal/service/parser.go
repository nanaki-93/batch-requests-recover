package service

import (
	"batchRequestsRecover/internal/model"
	"batchRequestsRecover/internal/util"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type ParserService struct {
	config model.Config
}

func NewParserService(config model.Config) *ParserService {
	return &ParserService{config: config}
}

func (s *ParserService) ReadAndParse(filePath string) ([]http.Request, error) {
	content, err := s.readFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}
	return s.parse(content)
}

func (s *ParserService) readFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return util.RemoveBOM(content), nil

}

func (s *ParserService) parse(content []byte) ([]http.Request, error) {

	reader := s.getReader(bytes.NewReader(content))

	var records []http.Request

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil && len(row) == 0 {
			return nil, fmt.Errorf("error reading row: %w", err)
		}

		if s.isAEmptyRow(row) {
			fmt.Println("Skipping empty row")
			continue
		}

		request, err := s.createRequest(row)
		if err != nil {
			return records, fmt.Errorf("error creating request: %w", err)
		}
		records = append(records, *request)
	}

	return records, nil
}

func (s *ParserService) createRequest(row []string) (*http.Request, error) {
	reqUrl := s.config.ApiEndpoint + s.config.GetPathVars(row) + s.config.GetQueryVars(row)

	var body string
	if s.config.HasBody {
		body = row[len(s.config.PathVars)+len(s.config.QueryVars)]
	}

	csvReq := model.NewCsvRequest(
		model.WithMethod(s.config.Method),
		model.WithHeaders(s.config.Headers),
		model.WithBody(body),
		model.WithRequestUrl(reqUrl),
	)
	request, err := createHttpRequest(csvReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	return request, nil
}

func (s *ParserService) isAEmptyRow(row []string) bool {
	return len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "")
}

func createHttpRequest(csvReq *model.CsvRequest) (*http.Request, error) {
	request, err := http.NewRequest(csvReq.Method, csvReq.RequestUrl, csvReq.Body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	for header, value := range csvReq.Headers {
		request.Header.Add(header, value)
	}
	return request, nil
}

func (s *ParserService) getReader(file io.Reader) *csv.Reader {
	reader := csv.NewReader(file)
	// Set delimiter (default to tab if not specified)
	if s.config.CSVDelimiter != "" && len(s.config.CSVDelimiter) > 0 {
		reader.Comma = rune(s.config.CSVDelimiter[0])
	} else {
		reader.Comma = '\t' // Tab separator
	}
	reader.LazyQuotes = true       // Allow lazy quotes
	reader.TrimLeadingSpace = true // Trim leading space
	return reader
}
