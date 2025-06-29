package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Global state for tracking timing
var lastTimestamp time.Time

// Color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// Helper function to apply color
func color(c string) string {
	if cfg.NoColor {
		return ""
	}
	return c
}

// Format timestamp for display
func formatTimestamp(timestamp string) string {
	t, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return "00:00:00"
	}
	// Convert to local time
	localTime := t.Local()

	// Calculate elapsed time if timing is enabled
	if cfg.ShowTiming && !lastTimestamp.IsZero() {
		elapsed := localTime.Sub(lastTimestamp)
		lastTimestamp = localTime

		// Format elapsed time
		var elapsedStr string
		switch {
		case elapsed < time.Second:
			elapsedStr = fmt.Sprintf("+%dms", elapsed.Milliseconds())
		case elapsed < time.Minute:
			elapsedStr = fmt.Sprintf("+%.1fs", elapsed.Seconds())
		default:
			elapsedStr = fmt.Sprintf("+%dm%ds", int(elapsed.Minutes()), int(elapsed.Seconds())%60)
		}
		return fmt.Sprintf("%s %s", localTime.Format("15:04:05"), elapsedStr)
	}

	lastTimestamp = localTime
	return localTime.Format("15:04:05")
}

// Extract content array from message
func extractContent(message map[string]interface{}) []map[string]interface{} {
	var result []map[string]interface{}

	// Handle string content (for regular user messages)
	if content, ok := message["content"].(string); ok {
		result = append(result, map[string]interface{}{
			"type": "text",
			"text": content,
		})
		return result
	}

	// Handle array content
	if content, ok := message["content"].([]interface{}); ok {
		for _, item := range content {
			if m, ok := item.(map[string]interface{}); ok {
				result = append(result, m)
			}
		}
	}

	return result
}

// Get brief summary of message for compact mode
func getMessageSummary(message map[string]interface{}) string {
	content := extractContent(message)
	if len(content) == 0 {
		return ""
	}

	var parts []string
	for _, item := range content {
		switch item["type"] {
		case "text":
			if text, ok := item["text"].(string); ok {
				// Take first line or 60 runes (for proper UTF-8 handling)
				lines := strings.Split(text, "\n")
				firstLine := strings.TrimSpace(lines[0])
				summary := truncateRunes(firstLine, 60)
				parts = append(parts, summary)
			}
		case "tool_use":
			if name, ok := item["name"].(string); ok {
				toolSummary := fmt.Sprintf("[Tool: %s]", name)
				// For Bash tool, include the command
				if input, ok := item["input"].(map[string]interface{}); ok {
					if name == "Bash" {
						if cmd, ok := input["command"].(string); ok {
							// Remove newlines and truncate command
							cmd = strings.ReplaceAll(cmd, "\n", " ")
							cmd = truncateRunes(strings.TrimSpace(cmd), 40)
							toolSummary = fmt.Sprintf("[Tool: Bash] %s", cmd)
						}
					} else {
						// Check for file_path in other tools
						if filePath, ok := input["file_path"].(string); ok {
							toolSummary = fmt.Sprintf("[Tool: %s] %s", name, filePath)
						}
					}
				}
				parts = append(parts, toolSummary)
			}
		case "tool_result":
			// Show tool result summary
			if content, ok := item["content"].(string); ok {
				lines := strings.Split(content, "\n")
				if len(lines) > 0 && lines[0] != "" {
					firstLine := strings.TrimSpace(lines[0])
					summary := truncateRunes(firstLine, 40)
					parts = append(parts, fmt.Sprintf("[Result: %s]", summary))
				} else {
					parts = append(parts, "[Tool Result]")
				}
			} else {
				parts = append(parts, "[Tool Result]")
			}
		}
	}

	return strings.Join(parts, " ")
}

// Display entry with tool information
func displayEntryWithToolInfo(entry map[string]interface{}, toolUseMap map[string]string, toolInputMap map[string]map[string]interface{}) {
	msgType, _ := entry["type"].(string)
	timestamp, _ := entry["timestamp"].(string)
	version, _ := entry["version"].(string)

	// Check if this entry should be displayed based on filters
	if !shouldDisplayEntryWithToolInfo(msgType, entry, toolUseMap) {
		return
	}

	// JSON output mode
	if cfg.OutputFormat == "json" {
		displayEntryAsJSON(entry, toolUseMap)
		return
	}

	// Format timestamp and version info
	timeStr := formatTimestamp(timestamp)
	versionStr := formatVersionInfo(version)

	// Route to appropriate display function
	// Note: "tool" type doesn't exist in the data, tool results are in "user" messages
	switch msgType {
	case "user":
		displayUserMessage(entry, timeStr, versionStr, toolUseMap, toolInputMap)
	case "assistant":
		displayAssistantMessage(entry, timeStr, versionStr)
	}
}

// Format version info for display
func formatVersionInfo(version string) string {
	if version == "" || cfg.Compact {
		return ""
	}
	return fmt.Sprintf(" %sv%s%s", color(colorGray), version, colorReset)
}

// Display user message
func displayUserMessage(entry map[string]interface{}, timeStr, versionStr string, toolUseMap map[string]string, toolInputMap map[string]map[string]interface{}) {
	message, ok := entry["message"].(map[string]interface{})
	if !ok {
		return
	}

	// Check if this is a tool result message
	contents := extractContent(message)
	isToolResult := false
	for _, content := range contents {
		if content["type"] == "tool_result" {
			isToolResult = true
			break
		}
	}

	if isToolResult {
		// Display as TOOL message
		toolUseResult, _ := entry["toolUseResult"].(map[string]interface{})
		displayToolResultSimple(message, timeStr, versionStr, toolUseMap, toolInputMap, toolUseResult)
	} else {
		// Check if this is a slash command
		isSlashCommand := false
		contents := extractContent(message)
		for _, content := range contents {
			if content["type"] == "text" {
				if text, ok := content["text"].(string); ok {
					// Slash commands are wrapped in <command-name> tags
					if strings.Contains(text, "<command-name>") && strings.Contains(text, "</command-name>") {
						isSlashCommand = true
					}
					break
				}
			}
		}

		// Display as regular USER message
		if !cfg.Compact {
			fmt.Printf("%s[%s]%s %sUSER%s",
				color(colorGray), timeStr, versionStr,
				color(colorBlue+colorBold), colorReset)

			// Add [COMMAND] label for slash commands
			if isSlashCommand {
				fmt.Printf(" %s[COMMAND]%s", color(colorPurple), colorReset)
			}

			fmt.Println()
			displayMessageContent(message, "  ")
			fmt.Println()
		} else {
			// Compact mode: fixed width role display
			fmt.Printf("%s[%s]%s %s%-9s%s - ",
				color(colorGray), timeStr, colorReset,
				color(colorBlue+colorBold), "USER", colorReset)

			summary := getMessageSummary(message)
			if summary != "" {
				fmt.Printf("%s\n", summary)
			} else {
				fmt.Printf("\n")
			}
		}
	}
}

// Display assistant message
func displayAssistantMessage(entry map[string]interface{}, timeStr, versionStr string) {
	message, ok := entry["message"].(map[string]interface{})
	if !ok {
		return
	}

	// Display header
	if !cfg.Compact {
		fmt.Printf("%s[%s]%s %sASSISTANT%s",
			color(colorGray), timeStr, versionStr,
			color(colorGreen+colorBold), colorReset)

		// Check for model info
		if model, ok := message["model"].(string); ok {
			fmt.Printf(" %s(%s)%s", color(colorGray), model, colorReset)
		}

		// Display usage info if available
		if usage, ok := message["usage"].(map[string]interface{}); ok {
			// Always show brief token info
			if inputTokens, ok := getTokenCount(usage, "input_tokens"); ok {
				if outputTokens, ok := getTokenCount(usage, "output_tokens"); ok {
					fmt.Printf(" [↑%d ↓%d", inputTokens, outputTokens)

					// Show cache info if available
					if cacheRead, ok := getTokenCount(usage, "cache_read_input_tokens"); ok && cacheRead > 0 {
						fmt.Printf(" *%d", cacheRead)
					}
					if cacheCreate, ok := getTokenCount(usage, "cache_creation_input_tokens"); ok && cacheCreate > 0 {
						fmt.Printf(" +%d", cacheCreate)
					}

					// Calculate and show cost if requested
					if cfg.ShowCost {
						modelName := ""
						if model, ok := message["model"].(string); ok {
							modelName = model
						}
						cost := calculateCost(usage, modelName)
						if cost > 0 {
							fmt.Printf(" $%.4f", cost)
						}
					}
					fmt.Printf("]")
				}
			}
		}

		fmt.Println()
		displayMessageContent(message, "  ")
		fmt.Println()
	} else {
		// Compact mode: fixed width role display, no metadata
		fmt.Printf("%s[%s]%s %s%-9s%s - ",
			color(colorGray), timeStr, colorReset,
			color(colorGreen+colorBold), "ASSISTANT", colorReset)

		// Show brief summary in compact mode
		summary := getMessageSummary(message)
		if summary != "" {
			fmt.Printf("%s\n", summary)
		} else {
			fmt.Printf("\n")
		}
	}
}

// Display tool result from user message

// Get tool name from tool result content
func getToolNameFromResult(message map[string]interface{}, toolUseMap map[string]string) string {
	contents := extractContent(message)
	for _, content := range contents {
		if content["type"] == "tool_result" {
			if id, ok := content["tool_use_id"].(string); ok {
				if name, exists := toolUseMap[id]; exists {
					return name
				}
			}
			break
		}
	}
	return ""
}

// Get tool input for tool result
func getToolInputForResult(message map[string]interface{}, toolInputMap map[string]map[string]interface{}) map[string]interface{} {
	contents := extractContent(message)
	for _, content := range contents {
		if content["type"] == "tool_result" {
			if id, ok := content["tool_use_id"].(string); ok {
				if input, exists := toolInputMap[id]; exists {
					return input
				}
			}
			break
		}
	}
	return nil
}

// Extract tool result info from contents
func extractToolResult(contents []map[string]interface{}) (isError bool, resultContent string) {
	for _, content := range contents {
		if content["type"] == "tool_result" {
			if err, ok := content["is_error"].(bool); ok {
				isError = err
			}
			if contentStr, ok := content["content"].(string); ok {
				resultContent = contentStr
			}
			return
		}
	}
	return
}

// Display error or OK status
func displayCompactStatus(isError bool) {
	if isError {
		fmt.Printf("[ERROR]")
	} else {
		fmt.Printf("[OK]")
	}
}

// Display tool result in compact mode
func displayToolResultCompact(message map[string]interface{}, toolName string, toolInput map[string]interface{}) {
	contents := extractContent(message)

	// Route to specific handlers
	switch {
	case toolName == "TodoWrite" && toolInput != nil:
		displayTodoWriteResultCompact(contents, toolInput)
	case toolName == "Bash" && toolInput != nil:
		displayBashResultCompact(contents, toolInput)
	case isFileOperationTool(toolName):
		displayFileToolResultCompact(contents, toolName, toolInput)
	case toolName == "WebFetch" || toolName == "WebSearch":
		displayWebToolResultCompact(contents, toolName, toolInput)
	case strings.HasPrefix(toolName, "mcp__"):
		displayMCPToolResultCompact(contents, toolName, toolInput)
	default:
		displayDefaultToolResultCompact(contents)
	}
}

// Check if tool is a file operation tool
func isFileOperationTool(toolName string) bool {
	switch toolName {
	case "Read", "Grep", "Glob", "Write", "Edit", "MultiEdit":
		return true
	}
	return false
}

// Display default tool result in compact mode
func displayDefaultToolResultCompact(contents []map[string]interface{}) {
	isError, _ := extractToolResult(contents)
	displayCompactStatus(isError)
	fmt.Println()
}

// Display TodoWrite result in compact mode with special handling
func displayTodoWriteResultCompact(contents []map[string]interface{}, toolInput map[string]interface{}) {
	isError, _ := extractToolResult(contents)
	displayCompactStatus(isError)
	fmt.Printf(" ")
	displayTodoWriteCompact(toolInput)
	fmt.Println()
}

// Display Bash result in compact mode
func displayBashResultCompact(contents []map[string]interface{}, toolInput map[string]interface{}) {
	isError, resultContent := extractToolResult(contents)

	// Try to extract exit code from the output
	exitCode := extractExitCode(resultContent)

	// Display status with exit code
	displayCompactStatus(isError)

	if exitCode >= 0 {
		fmt.Printf(" exit %d", exitCode)
	}

	// Display first line of output if available
	if resultContent != "" {
		lines := strings.Split(resultContent, "\n")
		if len(lines) > 0 && lines[0] != "" {
			firstLine := strings.TrimSpace(lines[0])
			if firstLine != "" {
				fmt.Printf(": %s", truncateRunes(firstLine, 50))
			}
		}
	}

	fmt.Println()
}

// Extract exit code from bash output (looks for common patterns)
func extractExitCode(output string) int {
	// Look for patterns like "exit status 1" or "exit code: 1"
	if strings.Contains(output, "exit status ") {
		parts := strings.Split(output, "exit status ")
		if len(parts) > 1 {
			codeStr := strings.Fields(parts[1])[0]
			if code, err := strconv.Atoi(codeStr); err == nil {
				return code
			}
		}
	}

	// If error output is empty, it usually means success
	if output == "" {
		return 0
	}

	return -1 // Unknown
}

// Display file operation tool results in compact mode
func displayFileToolResultCompact(contents []map[string]interface{}, toolName string, toolInput map[string]interface{}) {
	isError, resultContent := extractToolResult(contents)
	displayCompactStatus(isError)

	if !isError {
		displayFileToolInfo(toolName, resultContent, toolInput)
	}

	fmt.Println()
}

// Display file tool specific info
func displayFileToolInfo(toolName, resultContent string, toolInput map[string]interface{}) {
	switch toolName {
	case "Read":
		if resultContent != "" {
			lines := strings.Split(resultContent, "\n")
			fmt.Printf(" %d lines", len(lines))
		}
	case "Grep", "Glob":
		displayCountInfo(toolName, resultContent)
	case "Write":
		fmt.Printf(" file created")
	case "Edit":
		fmt.Printf(" file updated")
	case "MultiEdit":
		if edits, ok := toolInput["edits"].([]interface{}); ok {
			fmt.Printf(" %d edits applied", len(edits))
		}
	}
}

// Display count info for Grep and Glob
func displayCountInfo(toolName, resultContent string) {
	lines := strings.Split(strings.TrimSpace(resultContent), "\n")
	if lines[0] != "" {
		if toolName == "Grep" {
			fmt.Printf(" %d matches", len(lines))
		} else {
			fmt.Printf(" %d files found", len(lines))
		}
	}
}

// Display web tool results in compact mode
func displayWebToolResultCompact(contents []map[string]interface{}, toolName string, toolInput map[string]interface{}) {
	isError, resultContent := extractToolResult(contents)
	displayCompactStatus(isError)

	// Display tool-specific info
	switch toolName {
	case "WebFetch":
		if !isError && resultContent != "" {
			// Extract first meaningful line
			lines := strings.Split(resultContent, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					fmt.Printf(" %s", truncateRunes(line, 50))
					break
				}
			}
		}
	case "WebSearch":
		if !isError && resultContent != "" {
			// Count search results
			resultCount := strings.Count(resultContent, "<search_result>")
			if resultCount > 0 {
				fmt.Printf(" %d results", resultCount)
			}
		}
	}

	fmt.Println()
}

// Display MCP tool results in compact mode
func displayMCPToolResultCompact(contents []map[string]interface{}, toolName string, toolInput map[string]interface{}) {
	isError, resultContent := extractToolResult(contents)
	displayCompactStatus(isError)

	if !isError && resultContent != "" {
		displayMCPToolInfo(toolName, resultContent)
	}

	fmt.Println()
}

// Display MCP tool specific info
func displayMCPToolInfo(toolName, resultContent string) {
	parts := strings.Split(toolName, "__")
	if len(parts) <= 1 {
		return
	}

	action := parts[len(parts)-1]

	switch {
	case strings.HasPrefix(action, "create_"):
		displayMCPCreateInfo(resultContent)
	case strings.HasPrefix(action, "list_"):
		displayMCPListInfo(resultContent)
	case strings.HasPrefix(action, "get_"):
		displayMCPGetInfo(resultContent)
	}
}

// Display info for MCP create actions
func displayMCPCreateInfo(resultContent string) {
	if match := extractJSONValue(resultContent, "id"); match != "" {
		fmt.Printf(" Created: %s", match)
	} else if match := extractJSONValue(resultContent, "title"); match != "" {
		fmt.Printf(" Created: %s", truncateRunes(match, 30))
	}
}

// Display info for MCP list actions
func displayMCPListInfo(resultContent string) {
	if count := countJSONArrayItems(resultContent); count > 0 {
		fmt.Printf(" Found %d items", count)
	}
}

// Display info for MCP get actions
func displayMCPGetInfo(resultContent string) {
	if match := extractJSONValue(resultContent, "title"); match != "" {
		fmt.Printf(" %s", truncateRunes(match, 40))
	} else if match := extractJSONValue(resultContent, "name"); match != "" {
		fmt.Printf(" %s", truncateRunes(match, 40))
	}
}

// Extract a simple value from JSON-like content
func extractJSONValue(content, key string) string {
	// Simple pattern matching for common JSON patterns
	if idx := strings.Index(content, fmt.Sprintf("%q", key)); idx >= 0 {
		// Find the value after the key
		substr := content[idx:]
		if valueStart := strings.Index(substr, `:"`); valueStart >= 0 {
			valueStart += 2
			valueEnd := strings.Index(substr[valueStart:], `"`)
			if valueEnd >= 0 {
				return substr[valueStart : valueStart+valueEnd]
			}
		}
	}
	return ""
}

// Count items in JSON arrays
func countJSONArrayItems(content string) int {
	// Count occurrences of common item patterns
	count := 0

	// Try to count by looking for repeated patterns
	if strings.Contains(content, "[{") {
		// Count objects in arrays
		count = strings.Count(content, "},{") + 1
	} else if strings.Contains(content, `"id":`) {
		// Count by ID fields
		count = strings.Count(content, `"id":`)
	}

	return count
}

// Display TodoWrite in compact mode
func displayTodoWriteCompact(toolInput map[string]interface{}) {
	if todos, ok := toolInput["todos"].([]interface{}); ok && len(todos) > 0 {
		// Find the in_progress or most recently changed todo
		var focusedTodo map[string]interface{}
		for _, todoItem := range todos {
			if todo, ok := todoItem.(map[string]interface{}); ok {
				if status, ok := todo["status"].(string); ok && status == "in_progress" {
					focusedTodo = todo
					break
				}
				// If no in_progress, use the first todo
				if focusedTodo == nil {
					focusedTodo = todo
				}
			}
		}
		if focusedTodo != nil {
			if content, ok := focusedTodo["content"].(string); ok {
				status, _ := focusedTodo["status"].(string)
				statusIcon, statusColor := getTodoStatusIcon(status)
				fmt.Printf("%s%s%s %s", color(statusColor), statusIcon, colorReset, truncateRunes(content, 50))
			}
		}
	}
}

// Display tool result from user message (simplified version)
func displayToolResultSimple(message map[string]interface{}, timeStr, versionStr string, toolUseMap map[string]string, toolInputMap map[string]map[string]interface{}, toolUseResult map[string]interface{}) {
	// Get tool name and input
	toolName := getToolNameFromResult(message, toolUseMap)
	toolInput := getToolInputForResult(message, toolInputMap)

	// Display header
	if !cfg.Compact {
		fmt.Printf("%s[%s]%s %sTOOL%s",
			color(colorGray), timeStr, versionStr,
			color(colorCyan+colorBold), colorReset)
		if toolName != "" {
			fmt.Printf(" %s(%s)%s", color(colorGray), toolName, colorReset)
		}
		fmt.Println()
		displayMessageContentFull(message, "  ", toolName, toolUseResult, toolInput)
		fmt.Println()
		return
	}

	// Compact mode
	fmt.Printf("%s[%s]%s %s%-9s%s - ",
		color(colorGray), timeStr, colorReset,
		color(colorCyan+colorBold), "TOOL", colorReset)
	displayToolResultCompact(message, toolName, toolInput)
}

// Display message content
func displayMessageContent(message map[string]interface{}, indent string) {
	displayMessageContentFull(message, indent, "", nil, nil)
}

// Display message content with full context
func displayMessageContentFull(message map[string]interface{}, indent, toolName string, toolUseResult, toolInput map[string]interface{}) {
	content := extractContent(message)

	for _, item := range content {
		switch item["type"] {
		case "text":
			if text, ok := item["text"].(string); ok {
				displayText(text, indent)
			}
		case "tool_use":
			displayToolUse(item, indent)
		case "tool_result":
			displayToolResultFull(item, indent, toolName, toolUseResult, toolInput)
		}
	}
}

// Display text content
func displayText(text, indent string) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		fmt.Printf("%s%s\n", indent, line)
	}
}

// Display text content with line limit
func displayTextTruncated(text, indent string, maxLines int) {
	lines := strings.Split(text, "\n")
	totalLines := len(lines)

	// Show all lines if within limit
	if totalLines <= maxLines+2 { // +2 for better UX (don't truncate if we're close)
		for _, line := range lines {
			fmt.Printf("%s%s\n", indent, line)
		}
		return
	}

	// Show first maxLines lines
	for i := 0; i < maxLines && i < totalLines; i++ {
		fmt.Printf("%s%s\n", indent, lines[i])
	}

	// Show truncation notice
	remaining := totalLines - maxLines
	fmt.Printf("%s%s... (%d more lines)%s\n",
		indent, color(colorGray), remaining, colorReset)
}

// Truncate string by rune count (for proper UTF-8 handling)
func truncateRunes(s string, maxRunes int) string {
	if s == "" {
		return s
	}

	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}

	return string(runes[:maxRunes]) + "..."
}

// Display tool use
func displayToolUse(tool map[string]interface{}, indent string) {
	fmt.Printf("%s%s[Tool Use]%s", indent, color(colorYellow), colorReset)

	if name, ok := tool["name"].(string); ok {
		fmt.Printf(" %s", name)
		// Add MCP label for MCP tools
		if strings.HasPrefix(name, "mcp__") {
			fmt.Printf(" %s(MCP)%s", color(colorCyan), colorReset)
		}
	}

	if id, ok := tool["id"].(string); ok {
		fmt.Printf(" %s(ID: %s)%s", color(colorGray), id, colorReset)
	}

	fmt.Println()

	// Display input as key: value format for all tools
	if input, ok := tool["input"].(map[string]interface{}); ok && len(input) > 0 {
		displayToolInputAsKeyValue(input, indent+"  ")
	}
}

// Display tool input as key: value format with appropriate formatting
func displayToolInputAsKeyValue(input map[string]interface{}, indent string) {
	for key, value := range input {
		fmt.Printf("%s%s%s:%s ", indent, color(colorGray), key, colorReset)
		displayToolInputValue(key, value)
	}
}

// Display individual tool input value based on type
func displayToolInputValue(key string, value interface{}) {
	switch v := value.(type) {
	case string:
		// Path keys get special treatment
		if isPathKey(key) {
			fmt.Printf("%s\n", v)
		} else {
			fmt.Printf("%s\n", formatStringValue(v, 100))
		}
	case []interface{}:
		fmt.Printf("[%d items]\n", len(v))
	case map[string]interface{}:
		fmt.Printf("{%d keys}\n", len(v))
	case bool, float64, int:
		fmt.Printf("%v\n", v)
	case nil:
		fmt.Printf("null\n")
	default:
		// JSON fallback for complex types
		if data, err := json.Marshal(value); err == nil {
			fmt.Printf("%s\n", formatStringValue(string(data), 100))
		} else {
			fmt.Printf("%v\n", value)
		}
	}
}

// formatStringValue formats strings with truncation and multi-line handling
func formatStringValue(s string, maxLen int) string {
	if strings.Contains(s, "\n") {
		lines := strings.Split(s, "\n")
		firstLine := strings.TrimSpace(lines[0])
		if len(lines) > 1 {
			return fmt.Sprintf("%s... (%d more lines)", truncateRunes(firstLine, 60), len(lines)-1)
		}
		return firstLine
	}
	if len(s) > maxLen {
		return truncateRunes(s, 80)
	}
	return s
}

// isPathKey checks if a key represents a file path
func isPathKey(key string) bool {
	return key == "file_path" || key == "path" || strings.HasSuffix(key, "_path")
}

// Display tool result content with full context
func displayToolResultFull(result map[string]interface{}, indent, toolName string, toolUseResult, toolInput map[string]interface{}) {
	// Check if it's an error
	if isError, ok := result["is_error"].(bool); ok && isError {
		fmt.Printf("%s%s[ERROR]%s\n", indent, color(colorRed), colorReset)
	}

	// Special handling for TodoWrite
	if toolName == "TodoWrite" && toolUseResult != nil {
		displayTodoWriteResultWithData(result, indent, toolUseResult)
		return
	}

	// Display content - handle both string and array types
	hasContent := false
	switch content := result["content"].(type) {
	case string:
		if content != "" {
			displayTextTruncated(content, indent, 10)
			hasContent = true
		}
	case []interface{}:
		for _, item := range content {
			if m, ok := item.(map[string]interface{}); ok && m["type"] == "text" {
				if text, ok := m["text"].(string); ok && text != "" {
					displayTextTruncated(text, indent, 10)
					hasContent = true
				}
			}
		}
	}

	// Show "(No content)" if no content was displayed
	if !hasContent {
		fmt.Printf("%s%s(No content)%s\n", indent, color(colorGray), colorReset)
	}
}

// Get status icon and color
func getTodoStatusIcon(status string) (icon, color string) {
	switch status {
	case "completed":
		return "✓", colorGreen
	case "in_progress":
		return "→", colorYellow
	case "pending":
		return "□", colorGray
	default:
		return "•", colorGray
	}
}

// Display a single todo item
func displayTodoItem(todo map[string]interface{}, indent string) {
	content, _ := todo["content"].(string)
	status, _ := todo["status"].(string)
	priority, _ := todo["priority"].(string)

	statusIcon, statusColor := getTodoStatusIcon(status)

	// Display the todo item
	fmt.Printf("%s%s%s%s %s", indent, color(statusColor), statusIcon, colorReset, content)

	// Add priority indicator
	switch priority {
	case "high":
		fmt.Printf(" %s[HIGH]%s", color(colorRed), colorReset)
	case "medium":
		fmt.Printf(" %s[MED]%s", color(colorYellow), colorReset)
	}

	fmt.Println()
}

// Display TodoWrite result with structured data
func displayTodoWriteResultWithData(result map[string]interface{}, indent string, toolUseResult map[string]interface{}) {
	// Check for newTodos in the result
	if newTodos, ok := toolUseResult["newTodos"].([]interface{}); ok {
		// Display each todo item
		for _, todoItem := range newTodos {
			if todo, ok := todoItem.(map[string]interface{}); ok {
				displayTodoItem(todo, indent)
			}
		}

		// Changes are no longer shown since verbose mode is removed
	} else {
		// Fallback to content display if no structured data
		if content, ok := result["content"].(string); ok && content != "" {
			// Suppress the default message
			if !strings.Contains(content, "Todos have been modified successfully") {
				displayText(content, indent)
			}
		}
	}
}
