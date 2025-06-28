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
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{time.Minute * 30, "30 minutes"},
		{time.Hour, "1 hour"},
		{time.Hour * 2, "2 hours"},
		{time.Hour * 24, "1 day"},
		{time.Hour * 48, "2 days"},
		{time.Hour * 24 * 40, "1 month"},
		{time.Hour * 24 * 65, "2 months"},
		{time.Hour * 24 * 400, "1 year"},
		{time.Hour * 24 * 800, "2 years"},
	}

	for _, test := range tests {
		result := formatDuration(test.duration)
		if result != test.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", test.duration, result, test.expected)
		}
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{0, "s"},
		{1, ""},
		{2, "s"},
		{100, "s"},
	}

	for _, test := range tests {
		result := pluralize(test.count)
		if result != test.expected {
			t.Errorf("pluralize(%d) = %s, expected %s", test.count, result, test.expected)
		}
	}
}

func TestTruncateUTF8(t *testing.T) {
	tests := []struct {
		input    string
		maxRunes int
		expected string
	}{
		{"Hello", 10, "Hello"},
		{"Hello World", 8, "Hello..."},
		{"こんにちは世界", 10, "こんにちは世界"},
		{"日本語のテスト文字列です", 10, "日本語のテスト..."},
		{"🎉🎊🎈🎆🎇", 10, "🎉🎊🎈🎆🎇"},
		{"", 5, ""},
		{"短い", 10, "短い"},
		{"非常に長い日本語のテキストです", 10, "非常に長い日本..."},
		{"TODOの内容の色は変えずにアイコン部分と優先度部分だけを変更するようにして", 50, "TODOの内容の色は変えずにアイコン部分と優先度部分だけを変更するようにして"},
		{"TODOの内容の色は変えずにアイコン部分と優先度部分だけを変更するようにして", 30, "TODOの内容の色は変えずにアイコン部分と優先度部分だ..."},
	}

	for _, test := range tests {
		result := truncateUTF8(test.input, test.maxRunes)
		if result != test.expected {
			t.Errorf("truncateUTF8(%q, %d) = %q, expected %q",
				test.input, test.maxRunes, result, test.expected)
		}
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
