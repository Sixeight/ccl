# ccl - Claude Code Log

A CLI tool that converts Claude Code project files from JSONL format to human-readable format. It automatically detects project files from `~/.config/claude/projects/` and provides filtering, streaming, and formatting capabilities.

## Features

- 🚀 **Auto-detection**: Automatically detects project files from `~/.config/claude/projects/` for the current directory
- 🎨 **Color-coded display**: Easy-to-read color coding by role (USER/ASSISTANT/TOOL)
- 🔍 **Flexible filtering**: Advanced filtering by role, tool name, and glob patterns
- 📊 **Real-time support**: Process streaming input (`tail -f`) in real-time
- 💰 **Cost calculation**: Calculate token count and costs (optional)
- 🔧 **MCP tool support**: Readable formatting for MCP (Model Context Protocol) tool inputs
- ⚡ **Fast**: Uses only Go standard library, no external dependencies

## Installation

```bash
go install github.com/Sixeight/ccl@latest
```

Or clone and build from source:

```bash
git clone https://github.com/Sixeight/ccl.git
cd ccl
make build
```

## Usage

### Basic Usage

```bash
# Auto-detect and display project file for current directory
ccl

# Specify a file
ccl path/to/project.jsonl

# Pipe input
cat project.jsonl | ccl

# Streaming input (real-time display)
tail -f project.jsonl | ccl
```

### Display Options

```bash
# Compact mode (omit tool details)
ccl --compact

# No color output
ccl --no-color

# JSON output (preserves original format)
ccl --json

# Show token count and costs
ccl --pricing
```

### Filtering

#### Filter by Role

```bash
# Show specific roles only
ccl --filter user           # USER messages only
ccl --filter tool           # TOOL messages only
ccl --filter user,assistant # USER and ASSISTANT

# Exclude specific roles
ccl --exclude tool          # Show everything except TOOL
ccl --exclude user,tool     # Show everything except USER and TOOL (ASSISTANT only)
```

#### Filter by Tool

```bash
# Show specific tools only
ccl --tool Bash             # Bash tool usage only
ccl --tool Bash,Edit        # Bash or Edit tools
ccl --filter tool --tool Bash  # Bash tool results only

# Filter tools with glob patterns
ccl --tool "*Edit"          # All Edit-related tools (Edit, MultiEdit)
ccl --tool "Todo*"          # All Todo-related tools (TodoRead, TodoWrite)
ccl --tool "mcp__*"         # All MCP tools

# Exclude specific tools
ccl --tool-exclude TodoWrite    # Everything except TodoWrite
ccl --tool-exclude Bash,Edit    # Everything except Bash and Edit
ccl --tool-exclude "*Edit"      # Exclude Edit-related tools
```

### Advanced Usage

```bash
# Extract specific messages with jq
cat project.jsonl | jq 'select(.type=="user")' | ccl

# Display logs for a specific date range
ccl --json | jq 'select(.timestamp >= "2024-01-01" and .timestamp <= "2024-01-31")'

# Show only Bash commands containing errors
ccl --tool Bash --json | jq 'select(.content.output | contains("error"))'
```

## Display Examples

```
[06:01:35] USER
  I want to read Claude Code project files and display them in a formatted way with search queries.

[06:01:40] ASSISTANT (claude-opus-4-20250514)
  I'll create a plan for a jq-like tool "ccq" for Claude Code project files.

[06:01:47] ASSISTANT (claude-opus-4-20250514)
  → Tool: TodoWrite
  Input:
    Todos: 7 items
      - Research project structure and understand Claude Code project file format [pending]
      - Design and implement basic CLI tool structure (argument parsing, help display) [pending]
      - Implement JSON file reading and formatted display functionality [pending]
      ... and 4 more

[06:01:48] TOOL (result for: toolu_01)
  ← Tool Result (toolu_01) [success]
    Todos have been modified successfully.
```

### MCP Tool Display Example

MCP tools (starting with `mcp__`) get special formatting:

```
[10:23:45] ASSISTANT (claude-opus-4-20250514)
  → Tool: mcp__github__create_issue
  Input:
    owner: Sixeight
    repo: ccl
    title: Add support for JSON output format
    body: We should add a --json flag to output the processed data in JSON format...
    labels: [2 items]
      - enhancement
      - feature-request
```

## Color Scheme

- **USER**: Blue (user input)
- **ASSISTANT**: Green (Claude Code responses)
- **TOOL**: Cyan (tool execution results)
- **Tool names**: Yellow
- **Timestamps**: Gray

## Architecture

Module structure (~1900 lines):

- **main.go** (341 lines): Entry point, flag parsing, input detection
- **display.go** (738 lines): Display logic, color management, tool-specific display handlers
- **filter.go** (360 lines): Role and tool filtering, glob pattern matching
- **json_output.go** (44 lines): JSON output formatting
- **pricing.go** (116 lines): Token cost calculation
- **main_test.go** (312 lines): Test suite

### Processing Flow

1. **Input Detection**: Auto-detect stdin (streaming/buffered) or project files
2. **Two-pass Processing**:
   - Streaming mode: Real-time line-by-line processing
   - Buffered mode: First pass collects tool ID→name mappings, second pass displays
3. **Filtering**: Tool-based priority filtering when `--tool` is specified
4. **Display**: Role-specific handlers with special formatting for MCP tools

## Development

```bash
# Build
make build

# Run tests
make test

# Run all checks and build (fmt, lint, test, build)
make all

# Format code
make fmt

# Lint code
make lint
```

### Notes

- No external dependencies (uses only Go standard library)
- Flags must be specified before the file argument
- Tests run with timezone set to `UTC`

## Project File Location

Claude Code project files are stored at:

```
~/.config/claude/projects/[project-path]/[UUID].jsonl
```

ccl automatically finds the project file corresponding to your current directory.

## License

MIT License

## Contributing

Issues and Pull Requests are welcome. For major changes, please open an issue first to discuss what you would like to change.