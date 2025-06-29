package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsEmptyProjectFile(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "ccl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	type testCase struct {
		filename string
		content  string
		expected bool
	}

	tests := map[string]testCase{
		"empty file": {
			filename: "empty.jsonl",
			content:  "",
			expected: true,
		},
		"only summary": {
			filename: "summary-only.jsonl",
			content:  `{"type":"summary","summary":"Tool Result Display Enhancement and Style Cleanup","leafUuid":"6e6205f6-95e8-4012-8d06-1d4b9fb2ef6d"}`,
			expected: true,
		},
		"with user message": {
			filename: "with-user.jsonl",
			content: `{"type":"summary","summary":"Tool Result Display Enhancement and Style Cleanup","leafUuid":"6e6205f6-95e8-4012-8d06-1d4b9fb2ef6d"}
{"type":"user","message":{"role":"user","content":"Hello"}}`,
			expected: false,
		},
		"with assistant message": {
			filename: "with-assistant.jsonl",
			content: `{"type":"summary","summary":"Tool Result Display Enhancement and Style Cleanup","leafUuid":"6e6205f6-95e8-4012-8d06-1d4b9fb2ef6d"}
{"type":"assistant","message":{"role":"assistant","content":"Hi there"}}`,
			expected: false,
		},
		"with tool use only": {
			filename: "tool-only.jsonl",
			content: `{"type":"summary","summary":"Tool Result Display Enhancement and Style Cleanup","leafUuid":"6e6205f6-95e8-4012-8d06-1d4b9fb2ef6d"}
{"type":"tool","message":{"role":"tool","name":"Read","content":"File contents"}}`,
			expected: true,
		},
		"mixed content": {
			filename: "mixed.jsonl",
			content: `{"type":"summary","summary":"Tool Result Display Enhancement and Style Cleanup","leafUuid":"6e6205f6-95e8-4012-8d06-1d4b9fb2ef6d"}
{"type":"tool","message":{"role":"tool","name":"Read","content":"File contents"}}
{"type":"user","message":{"role":"user","content":"What's this?"}}
{"type":"assistant","message":{"role":"assistant","content":"This is a file"}}`,
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tmpDir, tc.filename)
			if err := os.WriteFile(testFile, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Test the function
			result := isEmptyProjectFile(testFile)
			if result != tc.expected {
				t.Errorf("isEmptyProjectFile(%s) = %v, want %v", tc.filename, result, tc.expected)
			}
		})
	}

	// Test non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		result := isEmptyProjectFile(filepath.Join(tmpDir, "does-not-exist.jsonl"))
		if !result {
			t.Error("isEmptyProjectFile(non-existent) = false, want true")
		}
	})
}

func TestProjectEncodeDirectoryPath(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"simple path": {
			input:    "/Users/sixeight/project",
			expected: "-Users-sixeight-project",
		},
		"path with dots": {
			input:    "/Users/sixeight/.config/claude",
			expected: "-Users-sixeight--config-claude",
		},
		"root path": {
			input:    "/",
			expected: "-",
		},
		"empty path": {
			input:    "",
			expected: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := encodeDirectoryPath(tc.input)
			if result != tc.expected {
				t.Errorf("encodeDirectoryPath(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestProjectDecodeDirectoryPath(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"simple encoded path": {
			input:    "-Users-sixeight-project",
			expected: "/Users/sixeight/project",
		},
		"encoded path with dots": {
			input:    "-Users-sixeight--config-claude",
			expected: "/Users/sixeight/.config/claude",
		},
		"encoded path with .ssh": {
			input:    "-Users-sixeight--ssh",
			expected: "/Users/sixeight/.ssh",
		},
		"encoded path with .local": {
			input:    "-Users-sixeight--local-share",
			expected: "/Users/sixeight/.local/share",
		},
		"encoded path with .cache": {
			input:    "-home-user--cache-app",
			expected: "/home/user/.cache/app",
		},
		"single dash": {
			input:    "-",
			expected: "/",
		},
		"empty": {
			input:    "",
			expected: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := decodeDirectoryPath(tc.input)
			if result != tc.expected {
				t.Errorf("decodeDirectoryPath(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
