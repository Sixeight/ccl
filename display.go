package main

import (
	"encoding/json"
	"fmt"
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
				if name == "Bash" {
					if input, ok := item["input"].(map[string]interface{}); ok {
						if cmd, ok := input["command"].(string); ok {
							// Remove newlines and truncate command
							cmd = strings.ReplaceAll(cmd, "\n", " ")
							cmd = truncateRunes(strings.TrimSpace(cmd), 40)
							toolSummary = fmt.Sprintf("[Tool: Bash] %s", cmd)
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
		displayToolResult(message, timeStr, versionStr, toolUseMap, toolInputMap, toolUseResult)
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
		fmt.Printf("%s[%s]%s %sUSER%s",
			color(colorGray), timeStr, versionStr,
			color(colorBlue+colorBold), colorReset)

		// Add [COMMAND] label for slash commands
		if isSlashCommand {
			fmt.Printf(" %s[COMMAND]%s", color(colorPurple), colorReset)
		}

		// Display content
		if !cfg.Compact {
			fmt.Println()
			displayMessageContent(message, "  ")
			fmt.Println()
		} else {
			// Show brief summary in compact mode
			summary := getMessageSummary(message)
			if summary != "" {
				fmt.Printf(" - %s\n", summary)
			} else {
				fmt.Println()
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

	// Display content
	if !cfg.Compact {
		fmt.Println()
		displayMessageContent(message, "  ")
		fmt.Println()
	} else {
		// Show brief summary in compact mode on same line
		summary := getMessageSummary(message)
		if summary != "" {
			fmt.Printf(" - %s\n", summary)
		} else {
			fmt.Println()
		}
	}
}

// Display tool result from user message
func displayToolResult(message map[string]interface{}, timeStr, versionStr string, toolUseMap map[string]string, toolInputMap map[string]map[string]interface{}, toolUseResult map[string]interface{}) {
	// Display header
	fmt.Printf("%s[%s]%s %sTOOL%s",
		color(colorGray), timeStr, versionStr,
		color(colorCyan+colorBold), colorReset)

	// Try to get tool name from tool_result content
	contents := extractContent(message)
	toolName := ""
	for _, content := range contents {
		if content["type"] == "tool_result" {
			if id, ok := content["tool_use_id"].(string); ok {
				if name, exists := toolUseMap[id]; exists {
					toolName = name
					fmt.Printf(" %s(%s)%s", color(colorGray), toolName, colorReset)

				}
			}
			break
		}
	}

	// Get tool input data for this tool use
	var toolInput map[string]interface{}
	for _, content := range contents {
		if content["type"] == "tool_result" {
			if id, ok := content["tool_use_id"].(string); ok {
				if input, exists := toolInputMap[id]; exists {
					toolInput = input
				}
			}
			break
		}
	}

	// Display content
	if !cfg.Compact {
		fmt.Println()
		displayMessageContentFull(message, "  ", toolName, toolUseResult, toolInput)
		fmt.Println()
	} else {
		// Show brief status in compact mode
		for _, content := range contents {
			if content["type"] == "tool_result" {
				if isError, ok := content["is_error"].(bool); ok && isError {
					fmt.Printf(" - [ERROR]\n")
				} else {
					fmt.Printf(" - [OK]\n")
				}
				return
			}
		}
		fmt.Println()
	}
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
		displayToolInputString(key, v)
	case []interface{}:
		displayToolInputArray(v)
	case map[string]interface{}:
		displayToolInputObject(v)
	case bool, float64, int:
		fmt.Printf("%v\n", v)
	case nil:
		fmt.Printf("null\n")
	default:
		displayToolInputDefault(value)
	}
}

// Display string value with appropriate truncation
func displayToolInputString(key, value string) {
	// Always show full path for file-related keys
	isPathKey := key == "file_path" || key == "path" || strings.HasSuffix(key, "_path")

	switch {
	case isPathKey:
		// Show full content for paths
		fmt.Printf("%s\n", value)
	case len(value) > 100:
		// Truncate very long strings
		fmt.Printf("%s\n", truncateRunes(value, 80))
	case strings.Contains(value, "\n"):
		// For multi-line strings, show first line only
		lines := strings.Split(value, "\n")
		firstLine := strings.TrimSpace(lines[0])
		if len(lines) > 1 {
			fmt.Printf("%s... (%d more lines)\n", truncateRunes(firstLine, 60), len(lines)-1)
		} else {
			fmt.Printf("%s\n", firstLine)
		}
	default:
		fmt.Printf("%s\n", value)
	}
}

// Display array values
func displayToolInputArray(v []interface{}) {
	// For arrays, show count and type
	fmt.Printf("[%d items]\n", len(v))
}

// Display object values
func displayToolInputObject(v map[string]interface{}) {
	// For objects, show key count
	fmt.Printf("{%d keys}\n", len(v))
}

// Display default/unknown type values
func displayToolInputDefault(value interface{}) {
	// For other types, convert to JSON
	if data, err := json.Marshal(value); err == nil {
		jsonStr := string(data)
		if len(jsonStr) <= 100 {
			fmt.Printf("%s\n", jsonStr)
		} else {
			fmt.Printf("%s\n", truncateRunes(jsonStr, 80))
		}
	} else {
		fmt.Printf("%v\n", value)
	}
}

// Display tool result string content
func displayToolResultString(content, indent string) bool {
	if content == "" {
		return false
	}
	// Show first 10 lines with truncation notice
	displayTextTruncated(content, indent, 10)
	return true
}

// Display tool result array content
func displayToolResultArray(content []interface{}, indent string) bool {
	hasContent := false
	for _, item := range content {
		if m, ok := item.(map[string]interface{}); ok {
			if m["type"] == "text" {
				if text, ok := m["text"].(string); ok && text != "" {
					hasContent = true
					displayTextTruncated(text, indent, 10)
				}
			}
		}
	}
	return hasContent
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

	// Display content - tool_result content is a string, not an array
	hasContent := false
	switch content := result["content"].(type) {
	case string:
		hasContent = displayToolResultString(content, indent)
	case []interface{}:
		hasContent = displayToolResultArray(content, indent)
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
