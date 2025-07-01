package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSidechainIntegration(t *testing.T) {
	// Save original config and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()

	// Create temporary test file
	tmpDir, err := os.MkdirTemp("", "ccl_sidechain_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test_sidechain.jsonl")
	testData := `{"parentUuid":null,"isSidechain":false,"userType":"external","cwd":"/tmp","sessionId":"test-main","version":"1.0.35","type":"user","message":{"role":"user","content":[{"type":"text","text":"Main session user message"}]},"uuid":"main-1","timestamp":"2025-06-30T12:00:01.000Z"}
{"parentUuid":"main-1","isSidechain":false,"userType":"external","cwd":"/tmp","sessionId":"test-main","version":"1.0.35","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Main session assistant response"}]},"uuid":"main-2","timestamp":"2025-06-30T12:00:02.000Z"}
{"parentUuid":null,"isSidechain":true,"userType":"external","cwd":"/tmp","sessionId":"test-side","version":"1.0.35","type":"user","message":{"role":"user","content":[{"type":"text","text":"Sidechain user message"}]},"uuid":"side-1","timestamp":"2025-06-30T12:00:03.000Z"}
{"parentUuid":"side-1","isSidechain":true,"userType":"external","cwd":"/tmp","sessionId":"test-side","version":"1.0.35","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Sidechain assistant response"}]},"uuid":"side-2","timestamp":"2025-06-30T12:00:04.000Z"}
{"parentUuid":"side-1","isSidechain":true,"userType":"external","cwd":"/tmp","sessionId":"test-side","version":"1.0.35","type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"test-tool","content":"Tool output"}]},"toolUseResult":{"content":[{"type":"text","text":"Tool output"}]},"uuid":"side-3","timestamp":"2025-06-30T12:00:05.000Z"}`

	err = os.WriteFile(testFile, []byte(testData), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tests := map[string]struct {
		args        []string
		expected    []string
		notExpected []string
	}{
		"normal_mode_shows_sidechain_tags": {
			args: []string{"log", testFile},
			expected: []string{
				"Main session user message",
				"Main session assistant response",
				"[SIDECHAIN]",
				"Sidechain user message",
				"Sidechain assistant response",
			},
			notExpected: []string{
				"USER*SC",
				"ASST*SC",
			},
		},
		"compact_mode_shows_sidechain_suffixes": {
			args: []string{"log", "-compact", testFile},
			expected: []string{
				"USER     ",
				"ASSISTANT",
				"USER*SC  ",
				"ASST*SC  ",
				"TOOL*SC  ",
				"Main session user message",
				"Sidechain user message",
			},
			notExpected: []string{
				"[SIDECHAIN]",
			},
		},
		"no_color_mode": {
			args: []string{"log", "-no-color", testFile},
			expected: []string{
				"USER",
				"ASSISTANT",
				"[SIDECHAIN]",
				"Main session user message",
				"Sidechain user message",
			},
			notExpected: []string{
				"\033[34m", // No blue color codes for main session
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset global state
			cfg = Config{} // Reset to defaults

			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Capture stderr too
			oldErr := os.Stderr
			rErr, wErr, _ := os.Pipe()
			os.Stderr = wErr

			// Set up os.Args for argument parsing
			originalArgs := os.Args
			os.Args = append([]string{"ccl"}, test.args...)

			// Execute the log command directly with proper args
			runLogCommand(test.args[1:]) // Skip "log" subcommand

			// Restore everything
			w.Close()
			wErr.Close()
			os.Stdout = old
			os.Stderr = oldErr
			os.Args = originalArgs

			// Get output
			buf := new(bytes.Buffer)
			buf.ReadFrom(r)
			output := buf.String()

			bufErr := new(bytes.Buffer)
			bufErr.ReadFrom(rErr)
			errOutput := bufErr.String()

			fullOutput := output + errOutput

			// Check expected strings are present
			for _, expected := range test.expected {
				if !strings.Contains(fullOutput, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, fullOutput)
				}
			}

			// Check that not expected strings are absent
			for _, notExpected := range test.notExpected {
				if strings.Contains(fullOutput, notExpected) {
					t.Errorf("Expected output NOT to contain %q, but got:\n%s", notExpected, fullOutput)
				}
			}
		})
	}
}

func TestSidechainColorCodes(t *testing.T) {
	// Save original config and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()

	// Create temporary test file
	tmpDir, err := os.MkdirTemp("", "ccl_color_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test_colors.jsonl")
	testData := `{"parentUuid":null,"isSidechain":false,"userType":"external","cwd":"/tmp","sessionId":"test-main","version":"1.0.35","type":"user","message":{"role":"user","content":[{"type":"text","text":"Main message"}]},"uuid":"main-1","timestamp":"2025-06-30T12:00:01.000Z"}
{"parentUuid":null,"isSidechain":true,"userType":"external","cwd":"/tmp","sessionId":"test-side","version":"1.0.35","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Sidechain message"}]},"uuid":"side-1","timestamp":"2025-06-30T12:00:02.000Z"}`

	err = os.WriteFile(testFile, []byte(testData), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tests := map[string]struct {
		args           []string
		expectedColors []string
	}{
		"main_session_colors": {
			args: []string{"log", testFile},
			expectedColors: []string{
				"\033[34m\033[1m", // Blue + bold for main USER
				"\033[92m\033[1m", // Light green + bold for sidechain ASSISTANT
				"\033[33m",        // Yellow for [SIDECHAIN] tag
			},
		},
		"compact_mode_colors": {
			args: []string{"log", "-compact", testFile},
			expectedColors: []string{
				"\033[34m\033[1m", // Blue + bold for main USER
				"\033[92m\033[1m", // Light green + bold for sidechain ASSISTANT
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset global state
			cfg = Config{} // Reset to defaults (colors enabled)

			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Set up os.Args for argument parsing
			originalArgs := os.Args
			os.Args = append([]string{"ccl"}, test.args...)

			// Execute the log command directly with proper args
			runLogCommand(test.args[1:]) // Skip "log" subcommand

			// Restore everything
			w.Close()
			os.Stdout = old
			os.Args = originalArgs

			// Get output
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

func TestSidechainJSONOutput(t *testing.T) {
	// Save original config and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()

	// Create temporary test file
	tmpDir, err := os.MkdirTemp("", "ccl_json_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test_json.jsonl")
	testData := `{"parentUuid":null,"isSidechain":true,"userType":"external","cwd":"/tmp","sessionId":"test-side","version":"1.0.35","type":"user","message":{"role":"user","content":[{"type":"text","text":"Sidechain JSON test"}]},"uuid":"side-1","timestamp":"2025-06-30T12:00:01.000Z"}`

	err = os.WriteFile(testFile, []byte(testData), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Reset global state
	cfg = Config{} // Reset to defaults

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set up os.Args for argument parsing
	originalArgs := os.Args
	os.Args = []string{"ccl", "log", "-json", testFile}

	// Execute the log command
	runLogCommand([]string{"-json", testFile})

	// Restore everything
	w.Close()
	os.Stdout = old
	os.Args = originalArgs

	// Get output
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	output := buf.String()

	// JSON output should preserve the isSidechain field
	expectedJSONFields := []string{
		`"isSidechain":true`,
		`"type":"user"`,
		`"Sidechain JSON test"`,
	}

	for _, expected := range expectedJSONFields {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected JSON output to contain %q, but got:\n%s", expected, output)
		}
	}

	// Should not contain text mode formatting
	notExpectedInJSON := []string{
		"[SIDECHAIN]",
		"USER*SC",
		"\033[", // ANSI color codes
	}

	for _, notExpected := range notExpectedInJSON {
		if strings.Contains(output, notExpected) {
			t.Errorf("Expected JSON output NOT to contain %q, but got:\n%s", notExpected, output)
		}
	}
}
