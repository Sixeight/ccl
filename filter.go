package main

import "strings"

// Parse comma-separated strings
func parseCommaSeparated(str string) []string {
	if str == "" {
		return []string{}
	}

	parts := strings.Split(str, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// Check if an entry should be displayed based on role filters
func shouldDisplayEntry(msgType string, entry map[string]interface{}) bool {
	// Parse filter list
	filterRoles := parseCommaSeparated(cfg.Role)

	// If no filter specified, display all
	if len(filterRoles) == 0 {
		return true
	}

	// Check if message type is in filter list
	for _, role := range filterRoles {
		if msgType == role {
			return true
		}
	}

	return false
}

// Check if an entry should be displayed based on all filters
func shouldDisplayEntryWithToolInfo(msgType string, entry map[string]interface{}, toolUseMap map[string]string) bool {
	// Check if tool filters are specified
	toolFilterList := parseCommaSeparated(cfg.ToolFilter)
	toolExcludeList := parseCommaSeparated(cfg.ToolExclude)
	hasToolFilters := len(toolFilterList) > 0 || len(toolExcludeList) > 0

	// If tool filters are specified, prioritize tool-based filtering
	if hasToolFilters {
		switch msgType {
		case "user":
			return shouldDisplayUserWithToolResult(entry, toolUseMap)
		case "assistant":
			return shouldDisplayAssistantWithTools(entry, toolUseMap)
		case "tool":
			return shouldDisplayToolResult(entry, toolUseMap)
		default:
			// For other message types, don't display when tool filters are active
			return false
		}
	}

	// If no tool filters, fall back to role-based filtering
	// Special handling for user messages that might contain tool results
	if msgType == "user" {
		return shouldDisplayUserWithToolResult(entry, toolUseMap)
	}

	// First check role filters for non-user messages
	if !shouldDisplayEntry(msgType, entry) {
		return false
	}

	// For tool messages, check tool filters
	if msgType == "tool" {
		return shouldDisplayToolResult(entry, toolUseMap)
	}

	// For assistant messages, check if they contain filtered tools
	if msgType == "assistant" {
		return shouldDisplayAssistantWithTools(entry, toolUseMap)
	}

	// For other message types, display if role filter passed
	return true
}

// Check if a tool result should be displayed
func shouldDisplayToolResult(entry map[string]interface{}, toolUseMap map[string]string) bool {
	toolFilterList := parseCommaSeparated(cfg.ToolFilter)
	toolExcludeList := parseCommaSeparated(cfg.ToolExclude)

	// Get tool name from parent message ID
	parentID, _ := entry["parentMessageId"].(string)
	toolName := toolUseMap[parentID]

	// If we couldn't determine tool name, apply default behavior
	if toolName == "" {
		// If there's a tool filter, don't show unknown tools
		if len(toolFilterList) > 0 {
			return false
		}
		// If there's no filter, show it (unless excluded)
		return true
	}

	return applyToolFilters(toolName, toolFilterList, toolExcludeList)
}

// Get tool name from content item
func getToolName(m map[string]interface{}, toolUseMap map[string]string) string {
	if id, ok := m["id"].(string); ok {
		if toolName := toolUseMap[id]; toolName != "" {
			return toolName
		}
	}
	if name, ok := m["name"].(string); ok {
		return name
	}
	return ""
}

// Check if tool is excluded
func isToolExcluded(toolName string, excludeList []string) bool {
	for _, pattern := range excludeList {
		if matchGlobPattern(pattern, toolName) {
			return true
		}
	}
	return false
}

// Check if tool matches filter
func isToolFiltered(toolName string, filterList []string) bool {
	if len(filterList) == 0 {
		return true // No filters means include all
	}
	for _, pattern := range filterList {
		if matchGlobPattern(pattern, toolName) {
			return true
		}
	}
	return false
}

// applyToolFilters checks if a tool name passes include/exclude filters
func applyToolFilters(toolName string, includeList, excludeList []string) bool {
	// Check excludes first
	if isToolExcluded(toolName, excludeList) {
		return false
	}

	// Check if tool matches include filter
	return isToolFiltered(toolName, includeList)
}

// Check if an assistant message with tools should be displayed
func shouldDisplayAssistantWithTools(entry map[string]interface{}, toolUseMap map[string]string) bool {
	toolFilterList := parseCommaSeparated(cfg.ToolFilter)
	toolExcludeList := parseCommaSeparated(cfg.ToolExclude)

	// If no tool filters, show all assistant messages
	if len(toolFilterList) == 0 && len(toolExcludeList) == 0 {
		return true
	}

	// Extract tool uses from message
	message, ok := entry["message"].(map[string]interface{})
	if !ok {
		return true
	}

	content, ok := message["content"].([]interface{})
	if !ok {
		return true
	}

	// Check if any tool passes filters
	for _, item := range content {
		m, ok := item.(map[string]interface{})
		if !ok || m["type"] != "tool_use" {
			continue
		}

		toolName := getToolName(m, toolUseMap)
		if applyToolFilters(toolName, toolFilterList, toolExcludeList) {
			return true
		}
	}

	return false
}

// Check if a user message with tool results should be displayed
func shouldDisplayUserWithToolResult(entry map[string]interface{}, toolUseMap map[string]string) bool {
	filterRoles := parseCommaSeparated(cfg.Role)

	// If no role filters, show all
	if len(filterRoles) == 0 {
		return true
	}

	// Check if "tool" is in filter and this has tool result
	hasToolFilter := false
	hasUserFilter := false
	for _, role := range filterRoles {
		if role == "tool" {
			hasToolFilter = true
		}
		if role == "user" {
			hasUserFilter = true
		}
	}

	if hasToolResult(entry) {
		// This is a tool result, show if tool is in filter
		if hasToolFilter {
			return shouldDisplayToolResultInUser(entry, toolUseMap)
		}
	} else {
		// This is a regular user message, show if user is in filter
		if hasUserFilter {
			return true
		}
	}
	return false
}

// Check if a user message contains tool results
func hasToolResult(entry map[string]interface{}) bool {
	message, ok := entry["message"].(map[string]interface{})
	if !ok {
		return false
	}

	content, ok := message["content"].([]interface{})
	if !ok {
		return false
	}

	for _, item := range content {
		if m, ok := item.(map[string]interface{}); ok {
			if m["type"] == "tool_result" {
				return true
			}
		}
	}
	return false
}

// Check if a tool result in a user message should be displayed
func shouldDisplayToolResultInUser(entry map[string]interface{}, toolUseMap map[string]string) bool {
	toolFilterList := parseCommaSeparated(cfg.ToolFilter)
	toolExcludeList := parseCommaSeparated(cfg.ToolExclude)

	message, ok := entry["message"].(map[string]interface{})
	if !ok {
		return true
	}

	content, ok := message["content"].([]interface{})
	if !ok {
		return true
	}

	// Find tool_result and get tool name
	var toolName string
	for _, item := range content {
		if m, ok := item.(map[string]interface{}); ok {
			if m["type"] == "tool_result" {
				if toolUseID, ok := m["tool_use_id"].(string); ok {
					toolName = toolUseMap[toolUseID]
				}
				break
			}
		}
	}

	// If we couldn't determine tool name, apply default behavior
	if toolName == "" {
		// If there's a tool filter, don't show unknown tools
		if len(toolFilterList) > 0 {
			return false
		}
		// If there's no filter, show it (unless excluded)
		return true
	}

	return applyToolFilters(toolName, toolFilterList, toolExcludeList)
}

// Match glob pattern against string
func matchGlobPattern(pattern, str string) bool {
	return matchGlobRecursive(pattern, str, 0, 0)
}

func matchGlobRecursive(pattern, str string, pIdx, sIdx int) bool {
	// Both exhausted, match
	if pIdx == len(pattern) && sIdx == len(str) {
		return true
	}

	// Pattern exhausted, no match
	if pIdx == len(pattern) {
		return false
	}

	// Handle * wildcard
	if pattern[pIdx] == '*' {
		// Try matching 0 or more characters
		for i := sIdx; i <= len(str); i++ {
			if matchGlobRecursive(pattern, str, pIdx+1, i) {
				return true
			}
		}
		return false
	}

	// String consumed
	if sIdx == len(str) {
		return false
	}

	// Handle ? wildcard or exact match
	if pattern[pIdx] == '?' || pattern[pIdx] == str[sIdx] {
		return matchGlobRecursive(pattern, str, pIdx+1, sIdx+1)
	}

	return false
}
