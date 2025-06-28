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

## Features

- Auto-detect project files for current directory
- Filter by role or tool with glob patterns
- Stream conversations in real-time
- Output as colored text or JSON
- Navigate to project directories quickly

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
ccl --compact    # Minimal output
ccl --json       # JSON format
ccl -f           # Follow mode
```

### Project Files

```bash
# List all projects
ccl log --projects

# List current directory's projects
ccl log --current  

# Open specific project
ccl log --projects | grep "myproject" | cut -f1 | xargs ccl
```


### Project Navigation

```bash
# Show project status
ccl status
ccl status --all      # All projects with IDs
ccl status abc123     # Specific project

# Jump to project directory  
cd $(ccl status -l abc123)
```


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

