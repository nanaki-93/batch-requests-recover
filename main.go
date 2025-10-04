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

	csvFilePath, apiURL, dryRun, sleep := checkArgs()

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
			println("Request URL: ", record)
			println("--- End Request ---")

		} else {
			resp, err := client.Do(record)
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

func checkArgs() (*string, *string, *bool, *int) {
	csvFilePath := flag.String("inputFile", "", "Path to CSV inputFile")
	apiURL := flag.String("url", "", "API endpoint")
	//todo change it to prod -> defaultValue := false
	dryRun := flag.Bool("dry", true, "Dry run")
	sleep := flag.Int("sleep", 1, "Sleep seconds between requests")

	flag.Parse()

	if *csvFilePath == "" || *apiURL == "" {
		println(" inputFile and url are required")
		os.Exit(1)
	}
	return csvFilePath, apiURL, dryRun, sleep
}

// parseCSV parses the CSV file with tab separator and removes single quotes
func parseCSV(file io.Reader, config *Config) ([]*http.Request, error) {

	reader := csv.NewReader(file)
	reader.Comma = '\t'            // Tab separator
	reader.LazyQuotes = true       // Allow lazy quotes
	reader.TrimLeadingSpace = true // Trim leading space

	var records []*http.Request

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

		req := http.Request{}

		req.Method = config.Method
		reqUrl := config.ApiEndpoint
		for header, value := range config.Headers {
			req.Header.Add(header, value)
		}

		for _, pathVar := range config.PathVars {
			reqUrl += "/" + trimQuotes(pathVar)
		}
		req.URL.Path = reqUrl

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

func getConfig() *Config {
	config := Config{}
	file, err := os.ReadFile("config.json")

	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		return &Config{}
	}
	return &config
}
