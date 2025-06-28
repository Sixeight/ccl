# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**ccl (Claude Code Log)** is a CLI tool that converts Claude Code project files from JSONL format to human-readable format. It automatically detects project files from `~/.config/claude/projects/` and provides filtering, streaming, formatting capabilities, and project discovery features.

## Development Commands

```bash
# Build the project
make build

# Run all checks and build (fmt, lint, test, build)
make all

# Run tests
make test

# Run a single test
go test -run TestFormatTimestamp

# Format code
make fmt

# Lint code  
make lint

# Run tests with coverage
make test-coverage

# Clean build artifacts
make clean

# Install globally
go install github.com/Sixeight/ccl@latest

# Run (after building)
./ccl [OPTIONS] [FILE]
```

## Architecture

The project follows a modular architecture with clear separation of concerns:

### Core Modules

- **main.go**: Entry point, command-line parsing, and orchestration
  - Handles subcommands (`log` and `status`)
  - Manages global configuration (`cfg Config`)
  - Coordinates between different modules

- **project.go**: Project file discovery and management
  - `listProjectFiles()`: Lists all available project files
  - `listCurrentProjectFiles()`: Lists only current directory's project files
  - `encodeDirectoryPath()`: Encodes paths to safe directory names
  - `shortenProjectNames()`: Creates minimal unique display names

- **status.go**: Project status and information display
  - `showProjectInfo()`: Displays project history and settings
  - `searchHistory()`: Searches through project command history
  - `generateProjectID()`: Creates 7-character SHA256-based IDs

- **display.go**: All display logic and formatting
  - `displayEntryWithToolInfo()`: Main entry point for displaying
  - Role-specific handlers for user/assistant/tool messages
  - Special formatting for MCP tools (key: value format)

- **filter.go**: Filtering logic with glob pattern support
  - `shouldDisplayEntryWithToolInfo()`: Main filter with tool-priority logic
  - `matchGlobPattern()`: Recursive glob matching (`*` and `?`)

- **pricing.go**: Token cost calculation (optional feature)
- **json_output.go**: JSON output formatting

### Processing Modes

1. **Follow mode** (`-f`): Continuously monitors file changes
2. **Streaming mode**: Real-time line-by-line processing for piped input
3. **Buffered mode**: Two-pass processing (collect tool mappings → display)

### Data Flow

```
Input (stdin/file) → Detection → Processing Mode → Filtering → Display → Output
                                      ↓
                              Tool ID Mapping (buffered mode only)
```

## Key Implementation Details

### Project File Structure

Project files are stored in: `[config-dir]/projects/[encoded-path]/[UUID].jsonl`

The config directory is determined by (in order):
1. `$CLAUDE_CONFIG_DIR`
2. `$XDG_CONFIG_HOME/claude`
3. `~/.claude` (default)

### Path Encoding

Paths are encoded to safe directory names:
```go
/Users/sixeight/project → -Users-sixeight-project
```

### Project ID Generation

Uses first 7 characters of SHA256 hash:
```go
sha256("/path/to/project")[:7] → "abc1234"
```

### Tool Filtering Priority

When `--tool` filters are specified, they take priority over role filters. This allows filtering by specific tools regardless of message type.

### MCP Tool Display

Tools starting with `mcp__` get special formatting:
- Input parameters shown as `key: value` pairs
- Long strings truncated to 80 characters
- Arrays/objects shown as `[N items]` or `{N keys}`
- Multi-line strings show first line + line count

### Flag Ordering

Due to flag parsing, options must come before the file argument:
```bash
./ccl --tool "Bash" file.jsonl    # Correct
./ccl file.jsonl --tool "Bash"    # Won't work
```

## Code Quality Standards

### Linting Rules (golangci-lint)

- Functions should have cyclomatic complexity ≤ 15
- Pre-allocate slices when size is known
- Avoid variable shadowing
- Keep functions under 50 statements

### Test Structure

Tests use named struct types with explicit field names:
```go
type testCase struct {
    input    string
    expected string
}

tests := map[string]testCase{
    "test name": {
        input:    "value",
        expected: "result",
    },
}
```

### Test Timezone Handling

Tests set `TZ=UTC` to ensure consistent timestamp formatting across different environments.

## Development Notes

- No external dependencies - uses only Go standard library
- Binary file `ccl` is currently tracked in git - consider adding to .gitignore
- When adding new display formats, update both text display (`display.go`) and JSON output (`json_output.go`)
- The `projectFile` struct is used for both `listProjectFiles()` and `listCurrentProjectFiles()` to maintain consistency
- MUST NOT commit without explicit instruction