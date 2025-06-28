package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const version = "0.0.1"

// Config holds all configuration options
type Config struct {
	Role          string
	OutputFormat  string
	ToolExclude   string
	ProjectPath   string
	ToolFilter    string
	LookDirectory string
	ShowTiming    bool
	ShowCost      bool
	NoColor       bool
	ShowAllTools  bool
	Follow        bool
	StatsAll      bool
	StatsProjects bool
	StatsCurrent  bool
	ShowInfoAll   bool
	Compact       bool
}

var cfg Config

func init() {
	// Set default usage function
	flag.Usage = printUsage
}

// LogConfig holds flags specific to the log command
type LogConfig struct {
	jsonFlag bool
}

var logConfig LogConfig

// setupLogFlags sets up flags for the log subcommand
func setupLogFlags(logCmd *flag.FlagSet) {
	logCmd.StringVar(&cfg.ProjectPath, "p", "", "path to Claude Code project file")
	logCmd.BoolVar(&cfg.NoColor, "no-color", false, "disable color output")
	logCmd.BoolVar(&cfg.Compact, "compact", false, "compact output mode")
	logCmd.StringVar(&cfg.Role, "role", "", "filter by role (user,assistant,tool)")
	logCmd.StringVar(&cfg.ToolFilter, "tool", "", "filter by tool name (supports glob: Bash,*Edit,Todo*)")
	logCmd.BoolVar(&cfg.ShowAllTools, "tools", false, "show all tool calls (equivalent to --tool '*')")
	logCmd.StringVar(&cfg.ToolExclude, "tool-exclude", "", "exclude tools by name (supports glob)")
	logCmd.BoolVar(&cfg.ShowCost, "cost", false, "show token costs (fetches latest pricing)")
	logCmd.BoolVar(&cfg.ShowTiming, "timing", false, "show timing information between messages")
	logCmd.StringVar(&cfg.OutputFormat, "format", "text", "output format (text, json)")
	logCmd.BoolVar(&logConfig.jsonFlag, "json", false, "shortcut for --format json")
	logCmd.BoolVar(&cfg.Follow, "f", false, "follow mode - continuously monitor for new entries (like tail -f)")
	logCmd.BoolVar(&cfg.StatsProjects, "projects", false, "list project file paths only (for piping)")
	logCmd.BoolVar(&cfg.StatsCurrent, "current", false, "list current directory's project files only")
}

// setupStatusFlags sets up flags for the status subcommand
func setupStatusFlags(statusCmd *flag.FlagSet) {
	statusCmd.BoolVar(&cfg.StatsAll, "all", false, "show all projects")
	statusCmd.StringVar(&cfg.LookDirectory, "l", "", "output cd command for project directory")
	statusCmd.StringVar(&cfg.LookDirectory, "look", "", "output cd command for project directory")
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "ccl - Claude Code Log viewer (version %s)\n\n", version)
	fmt.Fprintf(os.Stderr, "A tool to display Claude Code project files in a human-readable format.\n\n")
	fmt.Fprintf(os.Stderr, "Usage: %s [command] [options]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  log      Display project logs (default)\n")
	fmt.Fprintf(os.Stderr, "  status   Show project status and information\n")
	fmt.Fprintf(os.Stderr, "  version  Show version information\n")
	fmt.Fprintf(os.Stderr, "  help     Show this help message\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  # Display conversation from current project\n")
	fmt.Fprintf(os.Stderr, "  ccl\n")
	fmt.Fprintf(os.Stderr, "  ccl log\n\n")
	fmt.Fprintf(os.Stderr, "  # Show project status\n")
	fmt.Fprintf(os.Stderr, "  ccl status\n\n")
	fmt.Fprintf(os.Stderr, "  # Show all tool calls\n")
	fmt.Fprintf(os.Stderr, "  ccl log --tools\n\n")
	fmt.Fprintf(os.Stderr, "  # Follow mode (like tail -f)\n")
	fmt.Fprintf(os.Stderr, "  ccl log -f\n\n")
	fmt.Fprintf(os.Stderr, "Use 'ccl [command] --help' for more information about a command.\n")
}

func main() {
	// Handle subcommands
	if len(os.Args) < 2 {
		// No subcommand provided, default to "log"
		runLogCommand(os.Args[1:])
		return
	}

	subcommand := os.Args[1]

	// Check if first argument looks like a flag or file
	if strings.HasPrefix(subcommand, "-") || fileExists(subcommand) {
		// It's a flag or file, treat as log command
		runLogCommand(os.Args[1:])
		return
	}

	// Handle subcommands
	switch subcommand {
	case "log":
		runLogCommand(os.Args[2:])
	case "status":
		runStatusCommand(os.Args[2:])
	case "version":
		fmt.Printf("ccl version %s\n", version)
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// runLogCommand runs the log subcommand
func runLogCommand(args []string) {
	logCmd := flag.NewFlagSet("log", flag.ExitOnError)
	setupLogFlags(logCmd)

	logCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ccl log [options] [file]\n\n")
		fmt.Fprintf(os.Stderr, "Display Claude Code project logs.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		logCmd.PrintDefaults()
	}

	if err := logCmd.Parse(args); err != nil {
		return
	}

	// Handle shortcut flags
	if logConfig.jsonFlag {
		cfg.OutputFormat = "json"
	}

	// Handle project listing flags first
	if cfg.StatsProjects {
		listProjectFiles()
		return
	}

	if cfg.StatsCurrent {
		listCurrentProjectFiles()
		return
	}

	// If --tools was set, set tool filter to show all tools
	if cfg.ShowAllTools {
		cfg.ToolFilter = "*"
	}

	// Disable colors for JSON output
	if cfg.OutputFormat == "json" {
		cfg.NoColor = true
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
	reader, cleanup, err := getInputReaderForLog(logCmd)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	// Process and display conversation
	if err := processConversation(reader); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

// runStatusCommand runs the status subcommand
func runStatusCommand(args []string) {
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	setupStatusFlags(statusCmd)

	statusCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ccl status [options] [PROJECT_ID]\n\n")
		fmt.Fprintf(os.Stderr, "Show project status and information.\n")
		fmt.Fprintf(os.Stderr, "PROJECT_ID can be a project ID prefix from 'ccl status --all'.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		statusCmd.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  ccl status -l 3cdee5a    # Output: /path/to/project\n")
		fmt.Fprintf(os.Stderr, "  cd $(ccl status -l abc)  # Change to project directory\n")
	}

	if err := statusCmd.Parse(args); err != nil {
		return
	}

	// Get remaining arguments (non-flag arguments)
	remainingArgs := statusCmd.Args()
	var projectID string
	if len(remainingArgs) > 0 {
		projectID = remainingArgs[0]
	}

	// If -l/--look option is used, projectID is required if no ID provided as argument
	if cfg.LookDirectory != "" && projectID == "" {
		projectID = cfg.LookDirectory
	}

	// Determine which stats command to run
	if cfg.StatsAll {
		// Show all projects info
		cfg.ShowInfoAll = true
		showProjectInfo("")
	} else {
		// Default: show current project info or specified project
		showProjectInfo(projectID)
	}
}

// Get input source for log command
func getInputReaderForLog(logCmd *flag.FlagSet) (io.Reader, func(), error) {
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
		return file, func() { _ = file.Close() }, nil
	}

	// Check for file path from command line argument
	args := logCmd.Args()
	if len(args) > 0 {
		file, err := os.Open(args[0])
		if err != nil {
			return nil, nil, fmt.Errorf("opening file: %w", err)
		}
		return file, func() { _ = file.Close() }, nil
	}

	// Try to find project file for current directory
	projectFile := findProjectFile()
	if projectFile == "" {
		configDir := getClaudeConfigDir()
		return nil, nil, fmt.Errorf("no input provided and no project file found for current directory in %s/projects/", configDir)
	}

	file, err := os.Open(projectFile)
	if err != nil {
		return nil, nil, fmt.Errorf("opening project file: %w", err)
	}
	return file, func() { _ = file.Close() }, nil
}

// Process the conversation from reader
func processConversation(reader io.Reader) error {
	// Determine processing mode
	file, isFile := reader.(*os.File)
	isStdin := isFile && file == os.Stdin

	// Follow mode only works with files (not stdin)
	if cfg.Follow {
		if !isFile || isStdin {
			return fmt.Errorf("follow mode (-f) only works with file input, not stdin")
		}
		return processFollowMode(file)
	}

	// Check if we should use streaming mode
	if isStdin {
		stat, _ := os.Stdin.Stat()
		isStreaming := (stat.Mode() & os.ModeCharDevice) == 0
		if isStreaming {
			return processStreaming(reader)
		}
	}

	// Default: buffered processing
	return processBuffered(reader)
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
	if fileErr := processBuffered(file); fileErr != nil {
		return fileErr
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
