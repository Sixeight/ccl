package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	type testCase struct {
		expected string
		duration time.Duration
	}

	tests := map[string]testCase{
		"30 minutes": {
			duration: time.Minute * 30,
			expected: "30 minutes",
		},
		"1 hour": {
			duration: time.Hour,
			expected: "1 hour",
		},
		"2 hours": {
			duration: time.Hour * 2,
			expected: "2 hours",
		},
		"1 day": {
			duration: time.Hour * 24,
			expected: "1 day",
		},
		"2 days": {
			duration: time.Hour * 48,
			expected: "2 days",
		},
		"1 month": {
			duration: time.Hour * 24 * 40,
			expected: "1 month",
		},
		"2 months": {
			duration: time.Hour * 24 * 65,
			expected: "2 months",
		},
		"1 year": {
			duration: time.Hour * 24 * 400,
			expected: "1 year",
		},
		"2 years": {
			duration: time.Hour * 24 * 800,
			expected: "2 years",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatDuration(tc.duration)
			if result != tc.expected {
				t.Errorf("formatDuration(%v) = %s, expected %s", tc.duration, result, tc.expected)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	type testCase struct {
		expected string
		count    int
	}

	tests := map[string]testCase{
		"zero": {
			count:    0,
			expected: "s",
		},
		"one": {
			count:    1,
			expected: "",
		},
		"two": {
			count:    2,
			expected: "s",
		},
		"many": {
			count:    100,
			expected: "s",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := pluralize(tc.count)
			if result != tc.expected {
				t.Errorf("pluralize(%d) = %s, expected %s", tc.count, result, tc.expected)
			}
		})
	}
}

func TestTruncateUTF8(t *testing.T) {
	type testCase struct {
		input    string
		expected string
		maxRunes int
	}

	tests := map[string]testCase{
		"short ASCII": {
			input:    "Hello",
			maxRunes: 10,
			expected: "Hello",
		},
		"long ASCII": {
			input:    "Hello World",
			maxRunes: 8,
			expected: "Hello...",
		},
		"short Japanese": {
			input:    "こんにちは世界",
			maxRunes: 10,
			expected: "こんにちは世界",
		},
		"long Japanese": {
			input:    "日本語のテスト文字列です",
			maxRunes: 10,
			expected: "日本語のテスト...",
		},
		"emoji": {
			input:    "🎉🎊🎈🎆🎇",
			maxRunes: 10,
			expected: "🎉🎊🎈🎆🎇",
		},
		"empty string": {
			input:    "",
			maxRunes: 5,
			expected: "",
		},
		"very short Japanese": {
			input:    "短い",
			maxRunes: 10,
			expected: "短い",
		},
		"very long Japanese": {
			input:    "非常に長い日本語のテキストです",
			maxRunes: 10,
			expected: "非常に長い日本...",
		},
		"long TODO text within limit": {
			input:    "TODOの内容の色は変えずにアイコン部分と優先度部分だけを変更するようにして",
			maxRunes: 50,
			expected: "TODOの内容の色は変えずにアイコン部分と優先度部分だけを変更するようにして",
		},
		"long TODO text truncated": {
			input:    "TODOの内容の色は変えずにアイコン部分と優先度部分だけを変更するようにして",
			maxRunes: 30,
			expected: "TODOの内容の色は変えずにアイコン部分と優先度部分だ...",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := truncateUTF8(tc.input, tc.maxRunes)
			if result != tc.expected {
				t.Errorf("truncateUTF8(%q, %d) = %q, expected %q",
					tc.input, tc.maxRunes, result, tc.expected)
			}
		})
	}
}

func TestEncodeDirectoryPath(t *testing.T) {
	// Test that the function exists and returns a string
	path := "/Users/test/project"
	encoded := encodeDirectoryPath(path)
	if encoded == "" {
		t.Error("encodeDirectoryPath returned empty string")
	}
}

func TestShowProjectInfo(t *testing.T) {
	// Create a temporary config directory
	tempDir := t.TempDir()
	os.Setenv("CLAUDE_CONFIG_DIR", tempDir)
	defer os.Unsetenv("CLAUDE_CONFIG_DIR")

	// Create a mock .claude.json
	config := ClaudeConfig{
		NumStartups:    10,
		InstallMethod:  "test",
		AutoUpdates:    true,
		FirstStartTime: time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		Projects: map[string]ProjectInfo{
			"/test/project": {
				History: []HistoryEntry{
					{Display: "test command 1"},
					{Display: "test command 2"},
				},
			},
		},
	}

	configData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	configPath := filepath.Join(tempDir, ".claude.json")
	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Redirect stdout to capture output
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// Test the function doesn't crash
	showProjectInfo("")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout
}

func TestFindProjectFileForPath(t *testing.T) {
	// Create a temporary config directory
	tempDir := t.TempDir()
	os.Setenv("CLAUDE_CONFIG_DIR", tempDir)
	defer os.Unsetenv("CLAUDE_CONFIG_DIR")

	// Create project directory structure
	projectPath := "/test/project"
	encoded := encodeDirectoryPath(projectPath)
	projectDir := filepath.Join(tempDir, "projects", encoded)

	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Create a test JSONL file
	testFile := filepath.Join(projectDir, "test.jsonl")
	if err := os.WriteFile(testFile, []byte("{}"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test finding the file
	result := findProjectFileForPath(projectPath)
	if result != testFile {
		t.Errorf("Expected %s, got %s", testFile, result)
	}

	// Test with non-existent project
	result = findProjectFileForPath("/nonexistent")
	if result != "" {
		t.Errorf("Expected empty string for non-existent project, got %s", result)
	}
}

func TestSearchHistory(t *testing.T) {
	// Create a temporary config directory
	tempDir := t.TempDir()
	os.Setenv("CLAUDE_CONFIG_DIR", tempDir)
	defer os.Unsetenv("CLAUDE_CONFIG_DIR")

	// Create a mock .claude.json with command history
	config := ClaudeConfig{
		Projects: map[string]ProjectInfo{
			"/test/project1": {
				History: []HistoryEntry{
					{Display: "git status"},
					{Display: "git commit -m test"},
					{Display: "make build"},
					{Display: "go test"},
				},
			},
			"/test/project2": {
				History: []HistoryEntry{
					{Display: "make test"},
					{Display: "git push"},
					{Display: "npm install"},
				},
			},
		},
	}

	configData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	configPath := filepath.Join(tempDir, ".claude.json")
	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test various search patterns
	testCases := []struct {
		query         string
		expectedFound bool
	}{
		{"git*", true},
		{"make*", true},
		{"*test*", true},
		{"nonexistent", false},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			// Redirect stdout to capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run search
			searchHistory(tc.query)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout

			output := make([]byte, 1024)
			n, _ := r.Read(output)
			outputStr := string(output[:n])

			if tc.expectedFound {
				if strings.Contains(outputStr, "No matching messages found") {
					t.Errorf("Expected to find matches for %s but found none", tc.query)
				}
			} else {
				if !strings.Contains(outputStr, "No matching messages found") {
					t.Errorf("Expected no matches for %s but found some", tc.query)
				}
			}
		})
	}
}
