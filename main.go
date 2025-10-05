package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {

	config := getConfig()
	fmt.Println(config)
	client := RecoverClient()

	csvFilePath, dryRun, sleep := checkArgs()

	fmt.Printf("Processing inputFile: %s\n", *csvFilePath)

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
	records, err := parseCSV(bytes.NewReader(content), config)
	if err != nil {
		fmt.Println("Error reading CSV:", err)
		return
	}
	respList := make([]string, 0, len(records))
	errList := make([]string, 0)
	for i, record := range records {

		if *dryRun {
			println("Dry run, skipping request")
			println("Request URL: ", record.URL.String())
			println("Request Method: ", record.Method)
			println("--- End Request ---")

		} else {
			resp, err := client.Do(&record)
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

func checkArgs() (*string, *bool, *int) {
	csvFilePath := flag.String("inputFile", "", "Path to CSV inputFile")
	//todo change it to prod -> defaultValue := false
	dryRun := flag.Bool("dry", true, "Dry run")
	sleep := flag.Int("sleep", 1, "Sleep seconds between requests")

	flag.Parse()

	if *csvFilePath == "" {
		println(" inputFile is required")
		os.Exit(1)
	}
	return csvFilePath, dryRun, sleep
}

// parseCSV parses the CSV file with tab separator and removes single quotes
func parseCSV(file io.Reader, config *Config) ([]http.Request, error) {

	reader := getCsvReader(file, config)

	var records []http.Request

	totalColumns := len(config.PathVars) + len(config.QueryVars)
	if config.HasBody {
		totalColumns++
	}

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

		reqUrl := config.ApiEndpoint

		reqUrl = processPathVars(config, totalColumns, row, reqUrl)

		reqUrl = processQueryVars(config, reqUrl, totalColumns, row)

		var body io.Reader = nil
		if config.HasBody {
			body = bytes.NewBuffer([]byte(row[len(config.PathVars)+len(config.QueryVars)]))
		}

		request, err2 := createRequest(err, config, reqUrl, body)
		if err2 != nil {
			return nil, err2
		}

		records = append(records, *request)
	}

	return records, nil
}

func createRequest(err error, config *Config, reqUrl string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(config.Method, reqUrl, body)
	if err != nil {
		fmt.Printf("error creating request: %s", err)
		return nil, err
	}
	for header, value := range config.Headers {
		request.Header.Add(header, value)
	}
	return request, nil
}

func processQueryVars(config *Config, reqUrl string, totalColumns int, row []string) string {
	if len(config.QueryVars) > 0 {
		reqUrl += "?"
	}
	for j := 0; j < len(config.QueryVars); j++ {
		if (j + len(config.PathVars)) >= totalColumns {
			fmt.Println("Too many columns in the csv:", row)
			break
		}
		reqUrl += trimQuotes(config.QueryVars[j]) + "=" + trimQuotes(row[j+len(config.PathVars)]) + "&"
	}
	reqUrl = strings.TrimSuffix(reqUrl, "&")
	return reqUrl
}

func processPathVars(config *Config, totalColumns int, row []string, reqUrl string) string {
	for j := 0; j < len(config.PathVars); j++ {
		if j >= totalColumns {
			fmt.Println("Too many columns in the csv:", row)
			break
		}
		if j < len(config.PathVars) {
			reqUrl += "/" + trimQuotes(row[j])
		}
	}
	return reqUrl
}

func getCsvReader(file io.Reader, config *Config) *csv.Reader {
	reader := csv.NewReader(file)
	// Set delimiter (default to tab if not specified)
	if config.CSVDelimiter != "" && len(config.CSVDelimiter) > 0 {
		reader.Comma = rune(config.CSVDelimiter[0])
	} else {
		reader.Comma = '\t' // Tab separator
	}
	reader.LazyQuotes = true       // Allow lazy quotes
	reader.TrimLeadingSpace = true // Trim leading space
	return reader
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

func getConfig() *Config {
	config := Config{}
	file, err := os.ReadFile("config.json")

	err = json.Unmarshal(file, &config)
	fmt.Println("config:", config)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		return &Config{}
	}
	return &config
}
