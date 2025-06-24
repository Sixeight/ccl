# ccl - Claude Code Log

Simple Claude Code project files viewer.

## Quick Start

```bash
# Install
go install github.com/Sixeight/ccl@latest

# View current project's conversation
ccl

# View specific file
ccl project.jsonl

# Stream real-time updates
tail -f project.jsonl | ccl
```

## Key Features

- **Auto-detection** - Finds project files for your current directory automatically
- **Smart filtering** - Filter by role (user/assistant/tool) or specific tools with glob patterns
- **Real-time streaming** - Watch conversations as they happen
- **Multiple formats** - Text (colored), JSON, or compact output
- **Cost tracking** - Optional token usage and pricing information
- **Project discovery** - List and access all Claude Code project files easily

## Common Usage

### Filtering Examples

```bash
# Show only user messages
ccl --role user

# Show Bash commands and their results
ccl --tool Bash

# Show all Edit operations (Edit, MultiEdit)
ccl --tool "*Edit"

# Exclude todo-related messages
ccl --tool-exclude "Todo*"

# Combine filters
ccl --role assistant --tool "mcp__*"  # MCP tools used by assistant
```

### Output Options

```bash
ccl --compact    # Minimal output, no tool details
ccl --no-color   # Plain text without colors
ccl --json       # Original JSON format for piping
ccl --cost       # Include token costs
ccl -f           # Follow mode (like tail -f)
ccl -v           # Verbose output with tool details
```

### Project Management

```bash
# List all project files
ccl --list-projects
ccl -ls

# List current directory's project files
ccl --list-current
ccl -lc

# Verbose listing with timestamps and sizes
ccl -v -ls
ccl -v -lc

# Open most recent project
ccl -ls | head -1 | xargs ccl

# Search and open specific project
ccl -ls | grep "myproject" | xargs ccl

# Get project info as JSON
ccl --json -v -ls | jq '.[0]'
```

### Advanced Queries

```bash
# Find errors in Bash commands
ccl --tool Bash --json | jq 'select(.content.output | contains("error"))'

# Extract specific date range
ccl --json | jq 'select(.timestamp >= "2024-01-01")'

# Count messages by type
ccl --json | jq -s 'group_by(.type) | map({type: .[0].type, count: length})'

# Find largest project files
ccl --json -v -ls | jq 'sort_by(.size) | reverse | .[0:5]'
```

## Example Output

```
[06:01:35] USER
  Help me create a CLI tool to view Claude Code logs.

[06:01:40] ASSISTANT (claude-opus-4-20250514)
  I'll help you create that tool. Let me start by examining the project structure.

[06:01:47] ASSISTANT (claude-opus-4-20250514)
  → Tool: Bash
  Input:
    command: ls -la
  
[06:01:48] TOOL (result for: toolu_01)
  ← Tool Result (toolu_01) [success]
    total 16
    drwxr-xr-x  4 user  staff   128 Jan 15 10:00 .
    drwxr-xr-x  5 user  staff   160 Jan 15 09:00 ..
    -rw-r--r--  1 user  staff  1024 Jan 15 10:00 main.go
    -rw-r--r--  1 user  staff   512 Jan 15 10:00 go.mod
```

## Configuration

ccl looks for project files in the Claude configuration directory:

1. `$CLAUDE_CONFIG_DIR` (if set)
2. `$XDG_CONFIG_HOME/claude` (if XDG_CONFIG_HOME is set)
3. `~/.claude` (default)

Project files are stored as: `[config-dir]/projects/[encoded-path]/[UUID].jsonl`

## Development

```bash
git clone https://github.com/Sixeight/ccl.git
cd ccl
make all     # fmt, lint, test, and build
make test    # run tests only
make build   # build binary only
```

## License

MIT License

