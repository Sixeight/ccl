package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ClaudeConfig represents the structure of .claude.json
type ClaudeConfig struct {
	Projects       map[string]ProjectInfo `json:"projects"`
	MCPServers     map[string]MCPServer   `json:"mcpServers"`
	InstallMethod  string                 `json:"installMethod"`
	DiffTool       string                 `json:"diffTool"`
	FirstStartTime string                 `json:"firstStartTime"`
	NumStartups    int                    `json:"numStartups"`
	AutoUpdates    bool                   `json:"autoUpdates"`
}

// ProjectInfo represents project-specific information in .claude.json
type ProjectInfo struct {
	AllowedTools []string       `json:"allowedTools"`
	History      []HistoryEntry `json:"history"`
}

// HistoryEntry represents a message history entry
type HistoryEntry struct {
	PastedContents map[string]interface{} `json:"pastedContents"`
	Display        string                 `json:"display"`
}

// MCPServer represents MCP server configuration
type MCPServer struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// LocalSettings represents .claude/settings.local.json
type LocalSettings struct {
	Permissions struct {
		Allow []string `json:"allow"`
		Deny  []string `json:"deny"`
	} `json:"permissions"`
}

// loadClaudeConfig loads the global .claude.json configuration
func loadClaudeConfig() (*ClaudeConfig, error) {
	configDir := getClaudeConfigDir()
	configPath := filepath.Join(configDir, ".claude.json")

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", configPath, err)
	}

	var config ClaudeConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("parsing .claude.json: %w", err)
	}

	return &config, nil
}

// getProjectPathAndInfo retrieves project path and info based on projectID or current directory
func getProjectPathAndInfo(config *ClaudeConfig, projectID string) (string, ProjectInfo, error) {
	var projectPath string
	var projectInfo ProjectInfo
	var found bool

	if projectID != "" {
		path, err := findProjectByID(*config, projectID)
		if err != nil {
			return "", ProjectInfo{}, err
		}
		projectPath = path
		projectInfo = config.Projects[path]
		found = true
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return "", ProjectInfo{}, fmt.Errorf("getting current directory: %w", err)
		}
		projectPath = cwd
		projectInfo, found = config.Projects[cwd]
	}

	if !found {
		return projectPath, ProjectInfo{}, fmt.Errorf("no project history found")
	}

	return projectPath, projectInfo, nil
}

// displayProjectMessages displays recent messages from project history
func displayProjectMessages(projectInfo ProjectInfo) {
	totalCommands := len(projectInfo.History)
	if totalCommands == 0 {
		fmt.Println("No messages yet")
		return
	}

	fmt.Println("Recent messages (last 5):")
	displayLimit := 5
	if totalCommands < displayLimit {
		displayLimit = totalCommands
	}

	for i := 0; i < displayLimit; i++ {
		entry := projectInfo.History[i]
		display := truncateAtNewline(entry.Display, 70)
		fmt.Printf("  %3d. %s\n", i+1, display)
	}
}

// showProjectInfo displays project information from .claude.json
func showProjectInfo(projectID string) {
	config, err := loadClaudeConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	// If --all flag is set, show all projects
	if cfg.ShowInfoAll {
		showAllProjectsInfo(*config)
		return
	}

	projectPath, projectInfo, err := getProjectPathAndInfo(config, projectID)
	if err != nil {
		if err.Error() == "no project history found" {
			fmt.Println("No project history found.")
			fmt.Printf("\nAvailable projects: %d\n", len(config.Projects))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return
	}

	// If -l/--look option is set, output directory path and return
	if cfg.LookDirectory != "" {
		fmt.Println(projectPath)
		return
	}

	// Display project info
	if projectID != "" {
		fmt.Printf("Project: %s\n", projectPath)
		fmt.Printf("ID: %s\n\n", generateProjectID(projectPath))
	}

	// Display recent messages
	displayProjectMessages(projectInfo)

	// Load and display permissions
	localSettingsPath := filepath.Join(projectPath, ".claude", "settings.local.json")
	if settingsData, err := os.ReadFile(localSettingsPath); err == nil {
		var localSettings LocalSettings
		if err := json.Unmarshal(settingsData, &localSettings); err == nil {
			displayLocalSettings(localSettings)
		}
	}

	// Display MCP servers if configured
	if len(config.MCPServers) > 0 {
		displayMCPServersInfo(config.MCPServers)
	}
}

// displayLocalSettings displays permission settings from settings.local.json
func displayLocalSettings(settings LocalSettings) {
	if len(settings.Permissions.Allow) > 0 || len(settings.Permissions.Deny) > 0 {
		fmt.Println("\nPermissions:")

		if len(settings.Permissions.Allow) > 0 {
			fmt.Println("  Allowed:")
			for _, perm := range settings.Permissions.Allow {
				fmt.Printf("    ✓ %s\n", perm)
			}
		}

		if len(settings.Permissions.Deny) > 0 {
			fmt.Println("  Denied:")
			for _, perm := range settings.Permissions.Deny {
				fmt.Printf("    ✗ %s\n", perm)
			}
		}
	}
}

// displayMCPServersInfo displays MCP servers in a format similar to permissions
func displayMCPServersInfo(servers map[string]MCPServer) {
	fmt.Println("\nMCP Servers:")
	fmt.Println("  Enabled:")

	// Sort server names for consistent output
	serverNames := make([]string, 0, len(servers))
	for name := range servers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	// Display each server
	for _, name := range serverNames {
		fmt.Printf("    ✓ %s\n", name)
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 365 {
		years := days / 365
		return fmt.Sprintf("%d year%s", years, pluralize(years))
	}
	if days > 30 {
		months := days / 30
		return fmt.Sprintf("%d month%s", months, pluralize(months))
	}
	if days > 0 {
		return fmt.Sprintf("%d day%s", days, pluralize(days))
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%d hour%s", hours, pluralize(hours))
	}
	minutes := int(d.Minutes())
	return fmt.Sprintf("%d minute%s", minutes, pluralize(minutes))
}

// pluralize returns "s" if count is not 1
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// truncateUTF8 truncates a UTF-8 string to the specified rune count
func truncateUTF8(s string, maxRunes int) string {
	if maxRunes <= 3 {
		// If maxRunes is too small, just return the ellipsis
		return "..."
	}

	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}

	return string(runes[:maxRunes-3]) + "..."
}

// truncateAtNewline truncates a string at the first newline or max length
func truncateAtNewline(s string, maxRunes int) string {
	// Find the first newline
	if idx := strings.IndexByte(s, '\n'); idx != -1 {
		// Truncate at newline
		s = s[:idx]
	}

	// Then apply regular truncation if still too long
	return truncateUTF8(s, maxRunes)
}

// projectStat holds project statistics for status display
type projectStat struct {
	id       string
	path     string
	display  string
	lastCmd  string
	commands int
}

// shortenProjectPaths generates shortened display names for project paths
func shortenProjectPaths(projects []projectStat) {
	// First pass: count occurrences of last directory names
	lastDirCount := make(map[string]int)
	lastDirOnly := make([]string, len(projects))

	for i, p := range projects {
		parts := strings.Split(p.path, "/")
		if len(parts) > 0 {
			lastDir := parts[len(parts)-1]
			lastDirOnly[i] = lastDir
			lastDirCount[lastDir]++
		}
	}

	// Second pass: generate display names
	for i := range projects {
		parts := strings.Split(projects[i].path, "/")
		if len(parts) == 0 {
			continue
		}

		lastDir := parts[len(parts)-1]

		// If no duplicates, use only the last directory
		if lastDirCount[lastDir] == 1 {
			projects[i].display = lastDir
		} else {
			// For duplicates, find the minimum number of parent directories needed
			displayName := lastDir

			// Keep adding parent directories until we have a unique display name
			for j := len(parts) - 2; j >= 0; j-- {
				displayName = parts[j] + "/" + displayName

				// Check if this display name is unique among all project files
				isUnique := true
				for k, otherP := range projects {
					if k == i {
						continue
					}
					// Only check against other files with the same last directory
					if lastDirOnly[k] == lastDir {
						if strings.HasSuffix(otherP.path, displayName) {
							isUnique = false
							break
						}
					}
				}

				if isUnique {
					break
				}
			}

			projects[i].display = displayName
		}
	}
}

// generateProjectID generates a short unique ID from project path
func generateProjectID(path string) string {
	h := sha256.Sum256([]byte(path))
	return hex.EncodeToString(h[:])[:7] // Use first 7 chars of SHA256
}

// findProjectByID finds a project path by its ID prefix or shortened path
func findProjectByID(config ClaudeConfig, query string) (string, error) {
	var matches []string
	var pathMatches []string

	// First, create projectStat list to generate shortened names
	projects := make([]projectStat, 0, len(config.Projects))
	for path, info := range config.Projects {
		if len(info.History) > 0 {
			ps := projectStat{
				id:       generateProjectID(path),
				path:     path,
				display:  path,
				commands: len(info.History),
			}
			projects = append(projects, ps)
		}
	}

	// Generate shortened display names
	shortenProjectPaths(projects)

	// Search by both ID and shortened path
	for _, p := range projects {
		// Check ID prefix match
		if strings.HasPrefix(p.id, query) {
			matches = append(matches, p.path)
		}
		// Check shortened path match (exact or suffix)
		if p.display == query || strings.HasSuffix(p.path, "/"+query) {
			pathMatches = append(pathMatches, p.path)
		}
	}

	// Prefer path matches over ID matches
	if len(pathMatches) == 1 {
		return pathMatches[0], nil
	}
	if len(pathMatches) > 1 {
		return "", fmt.Errorf("ambiguous path '%s' matches multiple projects", query)
	}

	// Fall back to ID matches
	if len(matches) == 0 {
		return "", fmt.Errorf("no project found with ID or path: %s", query)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous ID prefix %s matches multiple projects", query)
	}

	return matches[0], nil
}

// showAllProjectsInfo shows information for all projects
func showAllProjectsInfo(config ClaudeConfig) {
	// Sort projects by command count (most active first)
	projects := make([]projectStat, 0, len(config.Projects))
	for path, info := range config.Projects {
		// Skip projects with no messages
		if len(info.History) == 0 {
			continue
		}
		ps := projectStat{
			id:       generateProjectID(path),
			path:     path,
			display:  path, // Will be shortened later
			commands: len(info.History),
			lastCmd:  info.History[0].Display, // First element is the newest
		}
		projects = append(projects, ps)
	}

	// Sort by path alphabetically
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].path < projects[j].path
	})

	// Generate shortened display names
	shortenProjectPaths(projects)

	// Find the maximum display name length for alignment
	maxDisplayLen := 0
	for _, p := range projects {
		if len(p.display) > maxDisplayLen {
			maxDisplayLen = len(p.display)
		}
	}

	// Display each project in compact format (one line per project)
	for _, p := range projects {
		// Add truncated last message if exists
		if p.lastCmd != "" && p.commands > 0 {
			// Truncate at newline or max length for single line display
			lastMsg := truncateAtNewline(p.lastCmd, 50)
			// Use padding to align messages
			fmt.Printf("%s\t%-*s\t%s\n", p.id, maxDisplayLen, p.display, lastMsg)
		} else {
			fmt.Printf("%s\t%s\n", p.id, p.display)
		}
	}
}

// findProjectFileForPath finds the project file for a given path
func findProjectFileForPath(projectPath string) string {
	encoded := encodeDirectoryPath(projectPath)
	configDir := getClaudeConfigDir()
	projectDir := filepath.Join(configDir, "projects", encoded)

	files, err := os.ReadDir(projectDir)
	if err != nil {
		return ""
	}

	// Find the most recent JSONL file
	var newestFile string
	var newestTime int64
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".jsonl") {
			info, err := file.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Unix() > newestTime {
				newestTime = info.ModTime().Unix()
				newestFile = filepath.Join(projectDir, file.Name())
			}
		}
	}

	return newestFile
}

// HistorySearchResult represents a message history search result
type HistorySearchResult struct {
	Timestamp time.Time
	Project   string
	Command   string
	Index     int
}

// searchHistory searches message history for matching patterns
func searchHistory(query string) {
	// Load global .claude.json
	configDir := getClaudeConfigDir()
	configPath := filepath.Join(configDir, ".claude.json")

	configData, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", configPath, err)
		return
	}

	var config ClaudeConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing .claude.json: %v\n", err)
		return
	}

	fmt.Printf("Searching for: %s\n", query)
	fmt.Println(strings.Repeat("=", 80))

	// Collect matching messages
	var results []HistorySearchResult

	for projectPath, info := range config.Projects {
		for i, cmd := range info.History {
			// Check if message matches the query (case-insensitive glob pattern)
			if matchGlobPattern(query, cmd.Display) {
				result := HistorySearchResult{
					Project: projectPath,
					Command: cmd.Display,
					Index:   i,
				}

				// Try to get timestamp from project file
				if projectFile := findProjectFileForPath(projectPath); projectFile != "" {
					if fileInfo, err := os.Stat(projectFile); err == nil {
						result.Timestamp = fileInfo.ModTime()
					}
				}

				results = append(results, result)
			}
		}
	}

	if len(results) == 0 {
		fmt.Println("\nNo matching messages found.")
		return
	}

	// Sort by project path for consistent output
	sort.Slice(results, func(i, j int) bool {
		if results[i].Project != results[j].Project {
			return results[i].Project < results[j].Project
		}
		return results[i].Index < results[j].Index
	})

	// Display results grouped by project
	currentProject := ""
	for _, result := range results {
		if result.Project != currentProject {
			if currentProject != "" {
				fmt.Println()
			}
			currentProject = result.Project

			// Shorten long paths
			displayPath := result.Project
			if len(displayPath) > 70 {
				displayPath = "..." + displayPath[len(displayPath)-67:]
			}
			fmt.Printf("Project: %s\n", displayPath)
		}

		// Display message with index
		fmt.Printf("  [%3d] %s\n", result.Index+1, result.Command)
	}

	fmt.Printf("\nTotal matches: %d\n", len(results))
}
