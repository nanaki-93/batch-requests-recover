package main

import (
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// CSVRecord represents a row from the CSV file
type CSVRecord struct {
	JSONPayload   string
	OptionalField string
}

func main() {

	// Create a custom transport that skips TLS verification
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	csvFilePath := flag.String("inputFile", "", "Path to CSV inputFile")
	apiURL := flag.String("url", "", "API endpoint")
	dryRun := flag.Bool("dry", false, "Dry run")
	sleep := flag.Int("sleep", 1, "Sleep seconds between requests")

	flag.Parse()

	if *csvFilePath == "" || *apiURL == "" {
		println(" inputFile and url are required")
		os.Exit(1)
	}

	fmt.Printf("Processing inputFile: %s\n", *csvFilePath)
	fmt.Printf("Sending to: %s\n", *apiURL)

	inputFile, err := os.Open(*csvFilePath)
	if err != nil {
		fmt.Println("Error opening inputFile:", err)
		return
	}
	defer inputFile.Close()

	// Read all content
	content, err := io.ReadAll(inputFile)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	// Remove BOM if present
	content = removeBOM(content)

	// Parse CSV records
	records, err := parseCSV(bytes.NewReader(content))
	if err != nil {
		fmt.Println("Error reading CSV:", err)
		return
	}
	respList := make([]string, 0, len(records))
	errList := make([]string, 0, 0)
	for i, record := range records {

		fmt.Printf("\n--- Request %d ---\n", i+1)
		payloadPreview := record.JSONPayload[0:100]
		fmt.Printf("JSON Payload: %s\n", payloadPreview)
		fmt.Printf("Optional Field: %s\n", record.OptionalField)

		requestUrl := *apiURL

		req, err := http.NewRequest("POST", requestUrl, bytes.NewBufferString(record.JSONPayload))
		if err != nil {
			fmt.Println("Error creating request:", err)
			continue
		}

		req.Header.Add("Content-Type", "application/json")
		client := &http.Client{Transport: tr}

		if *dryRun {
			println("Dry run, skipping request")
			println("Request URL: ", requestUrl)
			println("Request Body: ", payloadPreview)
			println("--- End Request ---")
			if (i % 10) != 0 {
				respList = append(respList, fmt.Sprint(strconv.Itoa(i)+"-"+"200 - "+payloadPreview))
			} else {
				errList = append(errList, fmt.Sprint(strconv.Itoa(i)+"-"+"500 - "+payloadPreview))
			}
		} else {
			resp, err := client.Do(req)
			if err != nil {
				fmt.Println("Error making request:", err)
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				fmt.Println("Error reading response:", err)
				continue
			}
			fmt.Println("Status:", resp.Status)
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				respList = append(respList, fmt.Sprint(strconv.Itoa(i)+"-"+resp.Status+" - "+string(body)))
			} else {
				errList = append(respList, fmt.Sprint(strconv.Itoa(i)+"-"+resp.Status+" - "+string(body)))
			}

		}
		sleepTime := time.Duration(*sleep) * time.Second
		time.Sleep(sleepTime)
		println("Sleeping for " + sleepTime.String() + " second...")
	}

	errFile := fmt.Sprint(inputFile.Name(), ".err")
	err = os.WriteFile(errFile, []byte(strings.Join(errList, "\n")), 0644)
	if err != nil {
		fmt.Println("Error writing error file:", err)
	}

	respFile := fmt.Sprint(inputFile.Name(), ".resp")
	err = os.WriteFile(respFile, []byte(strings.Join(respList, "\n")), 0644)
	if err != nil {
		fmt.Println("Error writing response file:", err)
	}
	fmt.Println("Done")
}

// parseCSV parses the CSV file with tab separator and removes single quotes
func parseCSV(file io.Reader) ([]CSVRecord, error) {

	reader := csv.NewReader(file)
	reader.Comma = '\t'            // Tab separator
	reader.LazyQuotes = true       // Allow lazy quotes
	reader.TrimLeadingSpace = true // Trim leading space

	var records []CSVRecord

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil && len(row) == 0 {
			return nil, fmt.Errorf("error reading row: %w", err)
		}

		// Skip empty rows
		if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
			continue
		}

		record := CSVRecord{}

		// First column (mandatory): JSON payload
		if len(row) > 0 {
			record.JSONPayload = trimQuotes(row[0])
		}

		// Second column (optional): string field
		if len(row) > 1 {
			record.OptionalField = trimQuotes(row[1])
		}

		// Only add if JSON payload is not empty
		if record.JSONPayload != "" {
			records = append(records, record)
		}
	}

	return records, nil
}

// trimQuotes removes surrounding single quotes and trims whitespace
func trimQuotes(s string) string {
	// Trim whitespace first
	s = strings.TrimSpace(s)

	// Remove single quotes from both ends
	s = strings.Trim(s, "'")

	// Trim whitespace again after removing quotes
	return strings.TrimSpace(s)
}

// removeBOM removes UTF-8 BOM from the beginning of the byte slice
func removeBOM(content []byte) []byte {
	// UTF-8 BOM is: EF BB BF
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		return content[3:]
	}
	return content
}
