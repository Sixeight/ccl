# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**ccl (Claude Code Log)** is a CLI tool that converts Claude Code project files from JSONL format to human-readable format. It automatically detects project files from `~/.config/claude/projects/` and provides filtering, streaming, and formatting capabilities.

## Development Commands

```bash
# Build the project
make build

# Run all checks and build (fmt, lint, test, build)
make all

# Run tests
make test

# Run a single test
go test -v -run TestFormatTimestamp

# Format code
make fmt

# Lint code
make lint

# Install globally
go install github.com/Sixeight/ccl@latest

# Run (after building)
./ccl [OPTIONS] [FILE]
```

## Architecture

The project has been refactored from a single-file design into a modular architecture (~1900 lines total):

### File Structure

- **main.go** (341 lines): Entry point, flag parsing, input detection, and orchestration
- **display.go** (738 lines): All display logic including formatting, color management, and tool-specific display handlers
- **filter.go** (360 lines): Filtering logic for roles and tools, including glob pattern matching
- **json_output.go** (44 lines): JSON output formatting
- **pricing.go** (116 lines): Token cost calculation (optional feature)
- **main_test.go** (312 lines): Test suite

### Core Processing Flow

1. **Input Detection** (`main.go`): Automatically detects stdin type (streaming vs buffered) or finds project files
2. **Two-Pass Processing**: 
   - Streaming mode (`processStreaming`): Real-time line-by-line processing
   - Buffered mode (`processBuffered`): First pass collects tool ID→name mappings, second pass displays
3. **Filtering** (`filter.go`): Multi-layered filtering with tool-based priority when `--tool` is specified
4. **Display** (`display.go`): Role-based handlers with special formatting for MCP tools

### Key Components

**Display Functions** (`display.go`):
- `displayEntryWithToolInfo()`: Main entry point for displaying
- `displayUserMessage()`, `displayAssistantMessage()`: Role-specific handlers
- `displayMCPToolInput()`: Special handler for MCP tools (key: value format with truncation)
- `displayToolUse()`: Formats tool invocations based on tool type

**Filtering Logic** (`filter.go`):
- `shouldDisplayEntryWithToolInfo()`: Main filter with tool-priority logic
- `shouldDisplayAssistantWithTools()`: Filters assistant messages containing tools
- `matchGlobPattern()`: Recursive glob matching supporting `*` and `?`

**Input/Output**:
- JSON output preserves original format when `--json` flag is used
- Streaming detection switches between real-time and buffered processing
- MCP tool inputs display in readable key: value format with long content truncation

## Important Implementation Details

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

### Test Timezone Handling
Tests set `TZ=UTC` to ensure consistent timestamp formatting across different environments.

## Development Notes

- No external dependencies - uses only Go standard library
- Binary file `ccl` is currently tracked in git - consider adding to .gitignore
- When adding new display formats, update both text display (`display.go`) and JSON output (`json_output.go`)
- MUST NOT commit without explicit instruction

