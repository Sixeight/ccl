package main

import (
	"strings"
	"testing"
)

// Test basic display functionality without worrying about exact formatting
func TestDisplayFunctionality(t *testing.T) {
	// Save original config
	origCfg := cfg
	defer func() {
		cfg = origCfg
	}()

	// Disable colors for cleaner test output
	cfg.NoColor = true

	t.Run("extractContent handles different formats", func(t *testing.T) {
		// Test string content
		msg1 := map[string]interface{}{
			"content": "Simple string",
		}
		content1 := extractContent(msg1)
		if len(content1) != 1 {
			t.Errorf("Expected 1 content item for string content, got %d", len(content1))
		}

		// Test array content
		msg2 := map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello",
				},
			},
		}
		content2 := extractContent(msg2)
		if len(content2) != 1 {
			t.Errorf("Expected 1 content item for array content, got %d", len(content2))
		}
	})

	t.Run("getMessageSummary creates proper summaries", func(t *testing.T) {
		// Test text summary
		msg1 := map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "This is a test message",
				},
			},
		}
		summary1 := getMessageSummary(msg1)
		if !strings.Contains(summary1, "This is a test message") {
			t.Errorf("Expected text summary to contain message, got: %s", summary1)
		}

		// Test tool use summary
		msg2 := map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "tool_use",
					"name": "Bash",
					"input": map[string]interface{}{
						"command": "ls -la",
					},
				},
			},
		}
		summary2 := getMessageSummary(msg2)
		if !strings.Contains(summary2, "[Tool: Bash]") {
			t.Errorf("Expected tool summary to contain tool name, got: %s", summary2)
		}
		if !strings.Contains(summary2, "ls -la") {
			t.Errorf("Expected Bash tool summary to contain command, got: %s", summary2)
		}
	})

	t.Run("truncateRunes handles UTF-8 properly", func(t *testing.T) {
		// Test ASCII
		result1 := truncateRunes("Hello World", 5)
		if result1 != "Hello..." {
			t.Errorf("Expected 'Hello...', got: %s", result1)
		}

		// Test Japanese
		result2 := truncateRunes("こんにちは世界", 5)
		if result2 != "こんにちは..." {
			t.Errorf("Expected 'こんにちは...', got: %s", result2)
		}

		// Test short string
		result3 := truncateRunes("Hi", 5)
		if result3 != "Hi" {
			t.Errorf("Expected 'Hi', got: %s", result3)
		}
	})

	t.Run("compact mode message summaries", func(t *testing.T) {
		// Save original compact setting
		origCompact := cfg.Compact
		cfg.Compact = true
		defer func() {
			cfg.Compact = origCompact
		}()

		// Test file_path in non-Bash tools
		msg := map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "tool_use",
					"name": "Read",
					"input": map[string]interface{}{
						"file_path": "/path/to/file.go",
					},
				},
			},
		}
		summary := getMessageSummary(msg)
		if !strings.Contains(summary, "[Tool: Read]") {
			t.Errorf("Expected tool name in summary, got: %s", summary)
		}
		if !strings.Contains(summary, "/path/to/file.go") {
			t.Errorf("Expected file path in summary for Read tool, got: %s", summary)
		}

		// Test that Write tool also shows file_path
		msg2 := map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "tool_use",
					"name": "Write",
					"input": map[string]interface{}{
						"file_path": "/another/path/file.txt",
						"content":   "file content here",
					},
				},
			},
		}
		summary2 := getMessageSummary(msg2)
		if !strings.Contains(summary2, "/another/path/file.txt") {
			t.Errorf("Expected file path in summary for Write tool, got: %s", summary2)
		}
	})
}
