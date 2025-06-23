package main

import (
	"encoding/json"
	"fmt"
)

// Display entry as JSON - outputs the original JSON without modification
func displayEntryAsJSON(entry map[string]interface{}, toolUseMap map[string]string) {
	// For JSON output, output the original entry as-is without any processing
	if jsonBytes, err := json.Marshal(entry); err == nil {
		fmt.Println(string(jsonBytes))
	}
}
