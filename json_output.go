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

// Format usage information for JSON output
func formatUsageJSON(usage map[string]interface{}, message map[string]interface{}) map[string]interface{} {
	usageOutput := make(map[string]interface{})

	if v, ok := getTokenCount(usage, "input_tokens"); ok {
		usageOutput["input_tokens"] = v
	}
	if v, ok := getTokenCount(usage, "output_tokens"); ok {
		usageOutput["output_tokens"] = v
	}
	if v, ok := getTokenCount(usage, "cache_read_input_tokens"); ok && v > 0 {
		usageOutput["cache_read_tokens"] = v
	}
	if v, ok := getTokenCount(usage, "cache_creation_input_tokens"); ok && v > 0 {
		usageOutput["cache_creation_tokens"] = v
	}

	// Add cost if requested
	if cfg.ShowCost {
		if model, ok := message["model"].(string); ok {
			cost := calculateCost(usage, model)
			if cost > 0 {
				usageOutput["cost"] = cost
			}
		}
	}

	return usageOutput
}
