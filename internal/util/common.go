package util

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// RemoveBOM removes UTF-8 BOM from the beginning of the byte slice
func RemoveBOM(content []byte) []byte {
	// UTF-8 BOM is: EF BB BF
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		return content[3:]
	}
	return content
}

// TrimQuotes removes surrounding single quotes and trims whitespace
func TrimQuotes(s string) string {
	// Trim whitespace first
	s = strings.TrimSpace(s)

	// Remove single quotes from both ends
	s = strings.Trim(s, "'")
	s = strings.Trim(s, "\"")

	// Trim whitespace again after removing quotes
	return strings.TrimSpace(s)
}

func DelayFor(sleep int) {
	sleepTime := time.Duration(sleep) * time.Second
	time.Sleep(sleepTime)
	println("Sleeping for " + sleepTime.String() + " second...")
}

func WriteResponses(inputFilePath string, respList []string, suffix string) {
	respFile := fmt.Sprint(inputFilePath, suffix)
	err := os.WriteFile(respFile, []byte(strings.Join(respList, "\n")), 0644)
	if err != nil {
		fmt.Println("Error writing "+inputFilePath+" file:", err)
	}
}
