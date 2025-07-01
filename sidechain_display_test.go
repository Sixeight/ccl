package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSidechainDisplay(t *testing.T) {
	// Save original config and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()

	tests := map[string]struct {
		entry    map[string]interface{}
		expected []string
		compact  bool
		noColor  bool
	}{
		"main_session_user_normal": {
			entry: map[string]interface{}{
				"type":        "user",
				"isSidechain": false,
				"timestamp":   "2025-06-30T12:00:00.000Z",
				"version":     "1.0.35",
				"message": map[string]interface{}{
					"role": "user",
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Main session message",
						},
					},
				},
			},
			compact: false,
			noColor: true,
			expected: []string{
				"USER",
				"Main session message",
			},
		},
		"sidechain_user_normal": {
			entry: map[string]interface{}{
				"type":        "user",
				"isSidechain": true,
				"timestamp":   "2025-06-30T12:00:00.000Z",
				"version":     "1.0.35",
				"message": map[string]interface{}{
					"role": "user",
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Sidechain message",
						},
					},
				},
			},
			compact: false,
			noColor: true,
			expected: []string{
				"USER",
				"[SIDECHAIN]",
				"Sidechain message",
			},
		},
		"main_session_assistant_compact": {
			entry: map[string]interface{}{
				"type":        "assistant",
				"isSidechain": false,
				"timestamp":   "2025-06-30T12:00:00.000Z",
				"version":     "1.0.35",
				"message": map[string]interface{}{
					"role": "assistant",
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Main assistant response",
						},
					},
				},
			},
			compact: true,
			noColor: true,
			expected: []string{
				"ASSISTANT",
				"Main assistant response",
			},
		},
		"sidechain_assistant_compact": {
			entry: map[string]interface{}{
				"type":        "assistant",
				"isSidechain": true,
				"timestamp":   "2025-06-30T12:00:00.000Z",
				"version":     "1.0.35",
				"message": map[string]interface{}{
					"role": "assistant",
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Sidechain response",
						},
					},
				},
			},
			compact: true,
			noColor: true,
			expected: []string{
				"ASST*SC",
				"Sidechain response",
			},
		},
		"sidechain_with_colors": {
			entry: map[string]interface{}{
				"type":        "user",
				"isSidechain": true,
				"timestamp":   "2025-06-30T12:00:00.000Z",
				"version":     "1.0.35",
				"message": map[string]interface{}{
					"role": "user",
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Colored sidechain",
						},
					},
				},
			},
			compact: false,
			noColor: false,
			expected: []string{
				"\033[94m\033[1m", // Light blue + bold color code
				"[SIDECHAIN]",
				"\033[33m", // Yellow color code for tag
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset global state
			lastTimestamp = time.Time{}

			// Configure test settings
			cfg = Config{
				Compact:      test.compact,
				NoColor:      test.noColor,
				OutputFormat: "text",
			}

			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Execute display function
			displayEntryWithToolInfo(test.entry, make(map[string]string), make(map[string]map[string]interface{}))

			// Restore stdout and get output
			w.Close()
			os.Stdout = old
			buf := new(bytes.Buffer)
			buf.ReadFrom(r)
			output := buf.String()

			// Check expected strings are present
			for _, expected := range test.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestSidechainColors(t *testing.T) {
	// Save original config and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()

	tests := map[string]struct {
		entryType      string
		expectedColors []string
		isSidechain    bool
		compact        bool
	}{
		"main_user_normal": {
			entryType:      "user",
			isSidechain:    false,
			compact:        false,
			expectedColors: []string{"\033[34m\033[1m"}, // Blue + bold
		},
		"sidechain_user_normal": {
			entryType:   "user",
			isSidechain: true,
			compact:     false,
			expectedColors: []string{
				"\033[94m\033[1m", // Light blue + bold
				"\033[33m",        // Yellow for [SIDECHAIN] tag
			},
		},
		"main_assistant_compact": {
			entryType:      "assistant",
			isSidechain:    false,
			compact:        true,
			expectedColors: []string{"\033[32m\033[1m"}, // Green + bold
		},
		"sidechain_assistant_compact": {
			entryType:      "assistant",
			isSidechain:    true,
			compact:        true,
			expectedColors: []string{"\033[92m\033[1m"}, // Light green + bold
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset global state
			lastTimestamp = time.Time{}

			// Configure test settings
			cfg = Config{
				Compact:      test.compact,
				NoColor:      false, // Enable colors for this test
				OutputFormat: "text",
			}

			entry := map[string]interface{}{
				"type":        test.entryType,
				"isSidechain": test.isSidechain,
				"timestamp":   "2025-06-30T12:00:00.000Z",
				"version":     "1.0.35",
				"message": map[string]interface{}{
					"role": test.entryType,
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Test message",
						},
					},
				},
			}

			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Execute display function
			displayEntryWithToolInfo(entry, make(map[string]string), make(map[string]map[string]interface{}))

			// Restore stdout and get output
			w.Close()
			os.Stdout = old
			buf := new(bytes.Buffer)
			buf.ReadFrom(r)
			output := buf.String()

			// Check expected color codes are present
			for _, expectedColor := range test.expectedColors {
				if !strings.Contains(output, expectedColor) {
					t.Errorf("Expected output to contain color code %q, but got:\n%s", expectedColor, output)
				}
			}
		})
	}
}

func TestSidechainToolDisplay(t *testing.T) {
	// Save original config and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()

	tests := map[string]struct {
		expected    []string
		isSidechain bool
		compact     bool
	}{
		"main_tool_normal": {
			isSidechain: false,
			compact:     false,
			expected:    []string{"TOOL"},
		},
		"sidechain_tool_normal": {
			isSidechain: true,
			compact:     false,
			expected:    []string{"TOOL", "[SIDECHAIN]"},
		},
		"main_tool_compact": {
			isSidechain: false,
			compact:     true,
			expected:    []string{"TOOL"},
		},
		"sidechain_tool_compact": {
			isSidechain: true,
			compact:     true,
			expected:    []string{"TOOL*SC"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset global state
			lastTimestamp = time.Time{}

			// Configure test settings
			cfg = Config{
				Compact:      test.compact,
				NoColor:      true,
				OutputFormat: "text",
			}

			entry := map[string]interface{}{
				"type":        "user",
				"isSidechain": test.isSidechain,
				"timestamp":   "2025-06-30T12:00:00.000Z",
				"version":     "1.0.35",
				"message": map[string]interface{}{
					"role": "user",
					"content": []interface{}{
						map[string]interface{}{
							"type":        "tool_result",
							"tool_use_id": "test-tool-id",
							"content":     "Tool result content",
						},
					},
				},
				"toolUseResult": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Tool result content",
						},
					},
				},
			}

			toolUseMap := map[string]string{
				"test-tool-id": "TestTool",
			}
			toolInputMap := map[string]map[string]interface{}{
				"test-tool-id": {
					"param": "value",
				},
			}

			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Execute display function
			displayEntryWithToolInfo(entry, toolUseMap, toolInputMap)

			// Restore stdout and get output
			w.Close()
			os.Stdout = old
			buf := new(bytes.Buffer)
			buf.ReadFrom(r)
			output := buf.String()

			// Check expected strings are present
			for _, expected := range test.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, output)
				}
			}
		})
	}
}
