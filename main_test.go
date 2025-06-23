package main

import (
	"os"
	"testing"
)

// Test data
var testEntry = map[string]interface{}{
	"type":      "user",
	"timestamp": "2025-06-22T09:59:11.123Z",
	"message": map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": "Hello, world!",
			},
		},
	},
}

var testAssistantEntry = map[string]interface{}{
	"type":      "assistant",
	"timestamp": "2025-06-22T09:59:15.456Z",
	"message": map[string]interface{}{
		"model": "claude-sonnet-4-20250514",
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": "Hello! How can I help you today?",
			},
		},
		"usage": map[string]interface{}{
			"input_tokens":  float64(10),
			"output_tokens": float64(20),
		},
	},
}

var testToolUseEntry = map[string]interface{}{
	"type":      "assistant",
	"timestamp": "2025-06-22T09:59:20.789Z",
	"message": map[string]interface{}{
		"model": "claude-sonnet-4-20250514",
		"content": []interface{}{
			map[string]interface{}{
				"type": "tool_use",
				"id":   "toolu_01ABC",
				"name": "Bash",
				"input": map[string]interface{}{
					"command": "ls -la",
				},
			},
		},
	},
}

var testToolResultEntry = map[string]interface{}{
	"type":            "tool",
	"timestamp":       "2025-06-22T09:59:25.012Z",
	"parentMessageId": "toolu_01ABC",
	"toolUseResult": map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": "file1.txt\nfile2.txt",
			},
		},
		"isError": false,
	},
}

func TestFormatTimestamp(t *testing.T) {
	// Set timezone to UTC for consistent test results
	originalTZ := os.Getenv("TZ")
	os.Setenv("TZ", "UTC")
	defer func() {
		if originalTZ == "" {
			os.Unsetenv("TZ")
		} else {
			os.Setenv("TZ", originalTZ)
		}
	}()

	tests := []struct {
		input    string
		expected string
	}{
		{"2025-06-22T09:59:11.123Z", "09:59:11"},
		{"2025-06-22T23:45:30.999Z", "23:45:30"},
		{"invalid", "00:00:00"},
	}

	for _, tt := range tests {
		result := formatTimestamp(tt.input)
		if result != tt.expected {
			t.Errorf("formatTimestamp(%s) = %s; want %s", tt.input, result, tt.expected)
		}
	}
}

// Extract text content from message - used only in tests
func extractTextContent(message map[string]interface{}) string {
	// Handle string content directly
	if content, ok := message["content"].(string); ok {
		return content
	}

	// Handle array content
	if content, ok := message["content"].([]interface{}); ok {
		for _, item := range content {
			if m, ok := item.(map[string]interface{}); ok {
				if m["type"] == "text" {
					if text, ok := m["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

func TestExtractTextContent(t *testing.T) {
	tests := []struct {
		name     string
		message  map[string]interface{}
		expected string
	}{
		{
			name:     "extract text from user message",
			message:  testEntry["message"].(map[string]interface{}),
			expected: "Hello, world!",
		},
		{
			name:     "extract text from assistant message",
			message:  testAssistantEntry["message"].(map[string]interface{}),
			expected: "Hello! How can I help you today?",
		},
		{
			name:     "no text content",
			message:  testToolUseEntry["message"].(map[string]interface{}),
			expected: "",
		},
		{
			name:     "empty message",
			message:  map[string]interface{}{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextContent(tt.message)
			if result != tt.expected {
				t.Errorf("extractTextContent() = %q; want %q", result, tt.expected)
			}
		})
	}
}

func TestGetTokenCount(t *testing.T) {
	usage := map[string]interface{}{
		"input_tokens":  float64(100),
		"output_tokens": 200, // int
	}

	tests := []struct {
		key      string
		expected int
		ok       bool
	}{
		{"input_tokens", 100, true},
		{"output_tokens", 200, true},
		{"missing_tokens", 0, false},
	}

	for _, tt := range tests {
		result, ok := getTokenCount(usage, tt.key)
		if ok != tt.ok {
			t.Errorf("getTokenCount(%s) ok = %v; want %v", tt.key, ok, tt.ok)
		}
		if result != tt.expected {
			t.Errorf("getTokenCount(%s) = %d; want %d", tt.key, result, tt.expected)
		}
	}
}

func TestMatchGlobPattern(t *testing.T) {
	tests := []struct {
		pattern  string
		str      string
		expected bool
	}{
		{"Bash", "Bash", true},
		{"*Edit", "MultiEdit", true},
		{"*Edit", "Edit", true},
		{"*Edit", "Editor", false},
		{"Todo*", "TodoWrite", true},
		{"Todo*", "MyTodo", false},
		{"*", "anything", true},
		{"", "something", false},
		{"test?", "test1", true},
		{"test?", "test12", false},
	}

	for _, tt := range tests {
		result := matchGlobPattern(tt.pattern, tt.str)
		if result != tt.expected {
			t.Errorf("matchGlobPattern(%q, %q) = %v; want %v",
				tt.pattern, tt.str, result, tt.expected)
		}
	}
}

func TestShouldDisplayEntry(t *testing.T) {
	// Save original values
	origCfg := cfg
	defer func() {
		cfg = origCfg
	}()

	tests := []struct {
		name     string
		filter   string
		msgType  string
		expected bool
	}{
		{"no filter", "", "user", true},
		{"filter user", "user", "user", true},
		{"filter user, message is assistant", "user", "assistant", false},
		{"filter multiple", "user,assistant", "assistant", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.Role = tt.filter
			result := shouldDisplayEntry(tt.msgType, nil)
			if result != tt.expected {
				t.Errorf("shouldDisplayEntry(%q) = %v; want %v",
					tt.msgType, result, tt.expected)
			}
		})
	}
}

func TestJSONOutput(t *testing.T) {
	// Test JSON output for user message
	old := cfg.OutputFormat
	cfg.OutputFormat = "json"
	defer func() { cfg.OutputFormat = old }()

	// Since we can't easily capture stdout in the test,
	// we'll just verify the function doesn't panic
	// In a real scenario, we'd refactor displayEntryAsJSON to use io.Writer
	displayEntryAsJSON(testEntry, nil)

	// Test with assistant message
	displayEntryAsJSON(testAssistantEntry, nil)

	// Test with tool result
	displayEntryAsJSON(testToolResultEntry, nil)
}

func TestParseRoles(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"user", []string{"user"}},
		{"user,assistant", []string{"user", "assistant"}},
		{"user,assistant,tool", []string{"user", "assistant", "tool"}},
		{"", []string{}},
		{"  user  ,  assistant  ", []string{"user", "assistant"}},
	}

	for _, tt := range tests {
		result := parseCommaSeparated(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseCommaSeparated(%q) length = %d; want %d",
				tt.input, len(result), len(tt.expected))
			continue
		}
		for i, role := range result {
			if role != tt.expected[i] {
				t.Errorf("parseCommaSeparated(%q)[%d] = %q; want %q",
					tt.input, i, role, tt.expected[i])
			}
		}
	}
}

func TestParseToolNames(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Bash", []string{"Bash"}},
		{"Bash,Edit", []string{"Bash", "Edit"}},
		{"*Edit,Todo*", []string{"*Edit", "Todo*"}},
		{"", []string{}},
	}

	for _, tt := range tests {
		result := parseCommaSeparated(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseCommaSeparated(%q) length = %d; want %d",
				tt.input, len(result), len(tt.expected))
			continue
		}
		for i, name := range result {
			if name != tt.expected[i] {
				t.Errorf("parseCommaSeparated(%q)[%d] = %q; want %q",
					tt.input, i, name, tt.expected[i])
			}
		}
	}
}

// Helper function to test color output
func TestColorFunction(t *testing.T) {
	// Test with color enabled
	cfg.NoColor = false
	if color(colorRed) != colorRed {
		t.Error("color() should return color code when cfg.NoColor=false")
	}

	// Test with color disabled
	cfg.NoColor = true
	if color(colorRed) != "" {
		t.Error("color() should return empty string when cfg.NoColor=true")
	}
	cfg.NoColor = false // Reset
}
