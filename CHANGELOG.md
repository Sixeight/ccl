# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.1] - 2025-06-28

### Added
- Initial release of ccl (Claude Code Log)
- Convert Claude Code project files from JSONL to human-readable format
- Automatic project file detection from `~/.config/claude/projects/`
- Multiple filtering options:
  - Role-based filtering (user/assistant/tool)
  - Tool name filtering with glob patterns
  - Tool exclusion
- Multiple output formats:
  - Default colored terminal output
  - JSON output format
  - Compact mode
- Project management features:
  - List all projects with `ccl status`
  - Show project information with `ccl status -i`
  - Search project history
  - Project statistics
- Real-time features:
  - Follow mode (`-f`) for monitoring file changes
  - Streaming support for piped input
- Display options:
  - Syntax highlighting for code blocks
  - Token usage and cost calculation
  - Timing information
  - Tool input/output display
- MCP (Model Context Protocol) tool support with special formatting
- Comprehensive test suite
- Full documentation in README.md

[0.0.1]: https://github.com/Sixeight/ccl/releases/tag/v0.0.1