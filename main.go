package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const version = "0.3.0"

// Config holds all configuration options
type Config struct {
	ProjectPath  string
	ShowVersion  bool
	NoColor      bool
	Compact      bool
	Verbose      bool
	Role         string
	ToolFilter   string
	ToolExclude  string
	ShowCost     bool
	ShowTiming   bool
	OutputFormat string
	ShowAllTools bool // Show all tools when --tools flag is used
	Follow       bool // Follow mode like tail -f
}

var cfg Config
var (
	textFlag   bool
	jsonFlag   bool
	followFlag bool
)

func init() {
	flag.StringVar(&cfg.ProjectPath, "p", "", "path to Claude Code project file")
	flag.BoolVar(&cfg.ShowVersion, "version", false, "show version")
	flag.BoolVar(&cfg.NoColor, "no-color", false, "disable color output")
	flag.BoolVar(&cfg.Compact, "compact", false, "compact output mode")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "verbose output (show tool input details)")
	flag.StringVar(&cfg.Role, "role", "", "filter by role (user,assistant,tool)")
	flag.StringVar(&cfg.ToolFilter, "tool", "", "filter by tool name (supports glob: Bash,*Edit,Todo*)")
	flag.BoolVar(&cfg.ShowAllTools, "tools", false, "show all tool calls (equivalent to --tool '*')")
	flag.StringVar(&cfg.ToolExclude, "tool-exclude", "", "exclude tools by name (supports glob)")
	flag.BoolVar(&cfg.ShowCost, "cost", false, "show token costs (fetches latest pricing)")
	flag.BoolVar(&cfg.ShowTiming, "timing", false, "show timing information between messages")
	flag.StringVar(&cfg.OutputFormat, "format", "text", "output format (text, json)")
	flag.BoolVar(&textFlag, "text", false, "shortcut for --format text")
	flag.BoolVar(&jsonFlag, "json", false, "shortcut for --format json")
	flag.BoolVar(&cfg.Follow, "f", false, "follow mode - continuously monitor for new entries (like tail -f)")
	flag.BoolVar(&followFlag, "follow", false, "follow mode - continuously monitor for new entries (like tail -f)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "ccl - Claude Code Log viewer (version %s)\n\n", version)
		fmt.Fprintf(os.Stderr, "A tool to display Claude Code project files in a human-readable format.\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] [FILE]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Display conversation from current project\n")
		fmt.Fprintf(os.Stderr, "  ccl\n\n")
		fmt.Fprintf(os.Stderr, "  # Display specific project file\n")
		fmt.Fprintf(os.Stderr, "  ccl project.jsonl\n\n")
		fmt.Fprintf(os.Stderr, "  # Compact mode (less verbose)\n")
		fmt.Fprintf(os.Stderr, "  ccl --compact\n\n")
		fmt.Fprintf(os.Stderr, "  # Filter by role\n")
		fmt.Fprintf(os.Stderr, "  ccl --role user,assistant\n\n")
		fmt.Fprintf(os.Stderr, "  # Show all tool calls\n")
		fmt.Fprintf(os.Stderr, "  ccl --tools\n")
		fmt.Fprintf(os.Stderr, "  ccl --tool '*'\n\n")
		fmt.Fprintf(os.Stderr, "  # Filter by tool name\n")
		fmt.Fprintf(os.Stderr, "  ccl --tool Bash,Edit\n\n")
		fmt.Fprintf(os.Stderr, "  # Use glob patterns\n")
		fmt.Fprintf(os.Stderr, "  ccl --tool \"*Edit\"     # All Edit tools\n")
		fmt.Fprintf(os.Stderr, "  ccl --tool \"Todo*\"     # All Todo tools\n\n")
		fmt.Fprintf(os.Stderr, "  # Show only Bash tool results\n")
		fmt.Fprintf(os.Stderr, "  ccl --tool Bash\n\n")
		fmt.Fprintf(os.Stderr, "  # JSON output\n")
		fmt.Fprintf(os.Stderr, "  ccl --json\n\n")
		fmt.Fprintf(os.Stderr, "  # Follow mode (like tail -f)\n")
		fmt.Fprintf(os.Stderr, "  ccl -f\n")
	}
}

func main() {
	flag.Parse()

	// Handle shortcut flags
	if jsonFlag {
		cfg.OutputFormat = "json"
	} else if textFlag {
		cfg.OutputFormat = "text"
	}

	// Handle follow flag aliases
	if followFlag {
		cfg.Follow = true
	}

	// If --tools was set, set tool filter to show all tools
	if cfg.ShowAllTools {
		cfg.ToolFilter = "*"
	}

	// Disable colors for JSON output
	if cfg.OutputFormat == "json" {
		cfg.NoColor = true
	}

	if cfg.ShowVersion {
		fmt.Printf("ccl version %s\n", version)
		os.Exit(0)
	}

	// Fetch pricing data if cost flag is set
	if cfg.ShowCost && cfg.OutputFormat == "text" {
		if err := fetchModelPricing(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to fetch pricing data: %v\n", err)
			// Continue without cost display
			cfg.ShowCost = false
		}
	}

	// Get input reader
	reader, cleanup, err := getInputReader()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Process and display conversation
	if err := processConversation(reader); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Get input source
func getInputReader() (io.Reader, func(), error) {
	// Check if stdin has data (pipe or redirect)
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Data is being piped in
		return os.Stdin, nil, nil
	}

	// Check for file path from -p flag
	if cfg.ProjectPath != "" {
		file, err := os.Open(cfg.ProjectPath)
		if err != nil {
			return nil, nil, fmt.Errorf("opening file: %w", err)
		}
		return file, func() { file.Close() }, nil
	}

	// Check for file path from command line argument
	if flag.NArg() > 0 {
		file, err := os.Open(flag.Arg(0))
		if err != nil {
			return nil, nil, fmt.Errorf("opening file: %w", err)
		}
		return file, func() { file.Close() }, nil
	}

	// Try to find project file for current directory
	projectFile := findProjectFile()
	if projectFile == "" {
		return nil, nil, fmt.Errorf("no input provided and no project file found for current directory")
	}

	file, err := os.Open(projectFile)
	if err != nil {
		return nil, nil, fmt.Errorf("opening project file: %w", err)
	}
	return file, func() { file.Close() }, nil
}

// Find project file in Claude Code config
func findProjectFile() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Encode the current directory path for matching
	encoded := encodeDirectoryPath(cwd)

	// Look in Claude Code config directory
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "claude", "projects")
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return ""
	}

	// Find matching directory
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == encoded {
			// Look for JSONL files in this directory
			projectDir := filepath.Join(configDir, entry.Name())
			files, err := os.ReadDir(projectDir)
			if err != nil {
				continue
			}

			// Find the most recent JSONL file
			var newestFile string
			var newestTime int64
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".jsonl") {
					info, err := file.Info()
					if err != nil {
						continue
					}
					if info.ModTime().Unix() > newestTime {
						newestTime = info.ModTime().Unix()
						newestFile = filepath.Join(projectDir, file.Name())
					}
				}
			}

			if newestFile != "" {
				return newestFile
			}
		}
	}

	return ""
}

// Encode path for project directory name
func encodeDirectoryPath(path string) string {
	// Replace path separators and dots with dashes
	encoded := strings.ReplaceAll(path, "/", "-")
	encoded = strings.ReplaceAll(encoded, ".", "-")
	// Claude Code keeps the leading dash
	return encoded
}

// Process the conversation from reader
func processConversation(reader io.Reader) error {
	// In follow mode, we need special handling for file input
	if cfg.Follow {
		// Follow mode only works with files, not stdin
		if file, ok := reader.(*os.File); ok && file != os.Stdin {
			return processFollowMode(file)
		} else {
			return fmt.Errorf("follow mode (-f) only works with file input, not stdin")
		}
	}

	// Check if stdin is a terminal (for streaming mode detection)
	stat, _ := os.Stdin.Stat()
	isStreaming := reader == os.Stdin && (stat.Mode()&os.ModeCharDevice) == 0

	if isStreaming {
		// Streaming mode: process line by line without buffering
		return processStreaming(reader)
	} else {
		// Regular mode: two-pass processing for tool name mapping
		return processBuffered(reader)
	}
}

// Process follow mode - continuously monitor file for new entries
func processFollowMode(file *os.File) error {
	// Tool maps that persist across all entries
	toolUseMap := make(map[string]string)
	toolInputMap := make(map[string]map[string]interface{})

	// First pass: collect tool information from existing content
	scanner := bufio.NewScanner(file)
	const maxScanTokenSize = 1024 * 1024 * 10 // 10MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		line := scanner.Text()
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		// Collect tool use information
		if msgType, _ := entry["type"].(string); msgType == "assistant" {
			collectToolUseInfo(entry, toolUseMap, toolInputMap)
		}
	}

	// Reset to beginning for display pass
	_, err := file.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("seeking to beginning: %w", err)
	}

	// Display all existing content
	if err := processBuffered(file); err != nil {
		return err
	}

	// Get current position (end of file)
	currentPos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("getting current position: %w", err)
	}

	// Continuously monitor for new content
	for {
		// Check current file size
		stat, err := file.Stat()
		if err != nil {
			return fmt.Errorf("getting file stats: %w", err)
		}

		if stat.Size() > currentPos {
			// New content available
			_, err := file.Seek(currentPos, 0)
			if err != nil {
				return fmt.Errorf("seeking to position: %w", err)
			}

			// Process new lines
			scanner := bufio.NewScanner(file)
			const maxScanTokenSize = 1024 * 1024 * 10 // 10MB
			buf := make([]byte, maxScanTokenSize)
			scanner.Buffer(buf, maxScanTokenSize)

			for scanner.Scan() {
				line := scanner.Text()

				var entry map[string]interface{}
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					continue // Skip malformed lines
				}

				// Collect tool use information
				if msgType, _ := entry["type"].(string); msgType == "assistant" {
					collectToolUseInfo(entry, toolUseMap, toolInputMap)
				}

				// Display immediately
				displayEntryWithToolInfo(entry, toolUseMap, toolInputMap)
			}

			if err := scanner.Err(); err != nil {
				return err
			}

			// Update current position
			currentPos = stat.Size()
		}

		// Sleep for a short interval before checking again
		time.Sleep(100 * time.Millisecond)
	}
}

// Process streaming input
func processStreaming(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	const maxScanTokenSize = 1024 * 1024 * 10 // 10MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	toolUseMap := make(map[string]string)                   // toolUseID -> toolName
	toolInputMap := make(map[string]map[string]interface{}) // toolUseID -> input data

	for scanner.Scan() {
		line := scanner.Text()

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines
		}

		// Collect tool use information for future reference
		if msgType, _ := entry["type"].(string); msgType == "assistant" {
			collectToolUseInfo(entry, toolUseMap, toolInputMap)
		}

		// Display immediately
		displayEntryWithToolInfo(entry, toolUseMap, toolInputMap)
	}

	return scanner.Err()
}

// Process buffered input
func processBuffered(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	const maxScanTokenSize = 1024 * 1024 * 10 // 10MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	var entries []map[string]interface{}
	toolUseMap := make(map[string]string)                   // toolUseID -> toolName
	toolInputMap := make(map[string]map[string]interface{}) // toolUseID -> input data

	// First pass: collect all entries and build tool name map
	for scanner.Scan() {
		line := scanner.Text()

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines
		}

		// Collect tool use information
		if msgType, _ := entry["type"].(string); msgType == "assistant" {
			collectToolUseInfo(entry, toolUseMap, toolInputMap)
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Second pass: display entries with tool name information
	for _, entry := range entries {
		displayEntryWithToolInfo(entry, toolUseMap, toolInputMap)
	}

	return nil
}

// Collect tool use information from assistant messages
func collectToolUseInfo(entry map[string]interface{}, toolUseMap map[string]string, toolInputMap map[string]map[string]interface{}) {
	if message, ok := entry["message"].(map[string]interface{}); ok {
		if content, ok := message["content"].([]interface{}); ok {
			for _, item := range content {
				if m, ok := item.(map[string]interface{}); ok {
					if m["type"] == "tool_use" {
						if toolID, ok := m["id"].(string); ok {
							if toolName, ok := m["name"].(string); ok {
								toolUseMap[toolID] = toolName
							}
							if input, ok := m["input"].(map[string]interface{}); ok {
								toolInputMap[toolID] = input
							}
						}
					}
				}
			}
		}
	}
}
