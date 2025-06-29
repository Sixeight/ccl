package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// projectFile represents a Claude project file
type projectFile struct {
	modTime time.Time
	path    string
	decoded string
	display string
	size    int64
	current bool
}

// getClaudeConfigDir returns the Claude configuration directory
// following the same logic as Claude Code:
// 1. CLAUDE_CONFIG_DIR environment variable
// 2. XDG_CONFIG_HOME/claude
// 3. ~/.claude (default)
func getClaudeConfigDir() string {
	// Check CLAUDE_CONFIG_DIR first
	if configDir := os.Getenv("CLAUDE_CONFIG_DIR"); configDir != "" {
		return configDir
	}

	// Check XDG_CONFIG_HOME
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "claude")
	}

	// Default to ~/.claude
	home := os.Getenv("HOME")
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".claude")
}

// Find project file in Claude Code config
func findProjectFile() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Encode the current directory path for matching
	encoded := encodeDirectoryPath(cwd)

	// Determine Claude config directory using the same logic as Claude Code
	configDir := getClaudeConfigDir()
	if configDir == "" {
		return ""
	}

	projectsDir := filepath.Join(configDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return ""
	}

	// Find matching directory
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() != encoded {
			continue
		}
		// Look for JSONL files in this directory
		projectDir := filepath.Join(projectsDir, entry.Name())
		files, err := os.ReadDir(projectDir)
		if err != nil {
			continue
		}

		// Find the most recent non-empty JSONL file
		var newestFile string
		var newestTime int64
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".jsonl") {
				continue
			}
			info, err := file.Info()
			if err != nil {
				continue
			}
			fullPath := filepath.Join(projectDir, file.Name())
			// Skip empty project files
			if isEmptyProjectFile(fullPath) {
				continue
			}
			if info.ModTime().Unix() > newestTime {
				newestTime = info.ModTime().Unix()
				newestFile = fullPath
			}
		}

		if newestFile != "" {
			return newestFile
		}
	}

	return ""
}

// Encode path for project directory name
func encodeDirectoryPath(path string) string {
	// Replace path separators and dots with dashes
	encoded := strings.ReplaceAll(path, "/", "-")
	encoded = strings.ReplaceAll(encoded, ".", "-")
	// Claude Code keeps the leading dash
	return encoded
}

// decodeDirectoryPath reverses the encoding to get original path
func decodeDirectoryPath(encoded string) string {
	// This is a simple approximation - we can't perfectly reverse it
	// but we can make it more readable

	// For common patterns, try to reconstruct the original path
	// Example: -Users-sixeight--config-claude -> /Users/sixeight/.config/claude
	decoded := encoded

	// Handle leading slash (paths usually start with /)
	if strings.HasPrefix(decoded, "-") {
		decoded = "/" + decoded[1:]
	}

	// Replace remaining dashes with slashes
	decoded = strings.ReplaceAll(decoded, "-", "/")

	// Try to fix common patterns like /.config
	decoded = strings.ReplaceAll(decoded, "//config", "/.config")
	decoded = strings.ReplaceAll(decoded, "//ssh", "/.ssh")
	decoded = strings.ReplaceAll(decoded, "//local", "/.local")
	decoded = strings.ReplaceAll(decoded, "//cache", "/.cache")

	return decoded
}

// isEmptyProjectFile checks if a project file contains no user/assistant messages
func isEmptyProjectFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return true // If we can't open it, treat as empty
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for {
		var entry map[string]interface{}
		if err := decoder.Decode(&entry); err != nil {
			break
		}
		// Check if this entry is a user or assistant message
		if entryType, ok := entry["type"].(string); ok && (entryType == "user" || entryType == "assistant") {
			return false // Found a message, not empty
		}
	}
	return true // No messages found
}

// listProjectFiles finds and displays all available project files
func listProjectFiles() {
	projectFiles := collectAllProjectFiles()
	if len(projectFiles) == 0 {
		fmt.Println("No project files found")
		return
	}

	// Sort by modification time (most recent first)
	sortProjectFilesByModTime(projectFiles)

	// Generate shortened display names
	shortenProjectNames(projectFiles)

	// Display project files
	displayProjectFiles(projectFiles)
}

// collectAllProjectFiles collects all project files from all project directories
func collectAllProjectFiles() []projectFile {
	configDir := getClaudeConfigDir()
	if configDir == "" {
		fmt.Fprintf(os.Stderr, "Error: Could not determine Claude config directory\n")
		return nil
	}

	projectsDir := filepath.Join(configDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "No projects directory found at %s\n", projectsDir)
		} else {
			fmt.Fprintf(os.Stderr, "Error reading projects directory: %v\n", err)
		}
		return nil
	}

	var projectFiles []projectFile

	// Get current working directory for comparison
	cwd, _ := os.Getwd()
	currentEncoded := encodeDirectoryPath(cwd)

	// Find all project files
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectDir := filepath.Join(projectsDir, entry.Name())
		files := collectProjectFilesFromDir(projectDir, entry.Name(), currentEncoded)
		projectFiles = append(projectFiles, files...)
	}

	return projectFiles
}

// collectProjectFilesFromDir collects JSONL files from a single project directory
func collectProjectFilesFromDir(projectDir, encodedName, currentEncoded string) []projectFile {
	files, err := os.ReadDir(projectDir)
	if err != nil {
		return nil
	}

	projectFiles := make([]projectFile, 0, len(files))
	decoded := decodeDirectoryPath(encodedName)

	// Look for JSONL files
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".jsonl") {
			continue
		}
		info, err := file.Info()
		if err != nil {
			continue
		}

		fullPath := filepath.Join(projectDir, file.Name())
		// Skip empty project files
		if isEmptyProjectFile(fullPath) {
			continue
		}

		projectFiles = append(projectFiles, projectFile{
			path:    fullPath,
			decoded: decoded,
			modTime: info.ModTime(),
			size:    info.Size(),
			current: encodedName == currentEncoded,
		})
	}

	return projectFiles
}

// sortProjectFilesByModTime sorts project files by modification time (most recent first)
func sortProjectFilesByModTime(projectFiles []projectFile) {
	for i := 0; i < len(projectFiles); i++ {
		for j := i + 1; j < len(projectFiles); j++ {
			if projectFiles[j].modTime.After(projectFiles[i].modTime) {
				projectFiles[i], projectFiles[j] = projectFiles[j], projectFiles[i]
			}
		}
	}
}

// listCurrentProjectFiles finds and displays project files for current directory only
func listCurrentProjectFiles() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		return
	}

	configDir := getClaudeConfigDir()
	if configDir == "" {
		fmt.Fprintf(os.Stderr, "Error: Could not determine Claude config directory\n")
		return
	}

	// Encode current directory path
	encoded := encodeDirectoryPath(cwd)
	projectDir := filepath.Join(configDir, "projects", encoded)

	// Check if project directory exists
	if _, statErr := os.Stat(projectDir); os.IsNotExist(statErr) {
		// No project files for current directory
		return
	}

	// Read project directory
	files, err := os.ReadDir(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading project directory: %v\n", err)
		return
	}

	// Collect JSONL files
	projectFiles := make([]projectFile, 0, len(files))

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".jsonl") {
			continue
		}
		info, err := file.Info()
		if err != nil {
			continue
		}

		fullPath := filepath.Join(projectDir, file.Name())
		// Skip empty project files
		if isEmptyProjectFile(fullPath) {
			continue
		}
		projectFiles = append(projectFiles, projectFile{
			path:    fullPath,
			decoded: cwd,
			modTime: info.ModTime(),
			size:    info.Size(),
		})
	}

	// Sort by modification time (most recent first)
	sortProjectFilesByModTime(projectFiles)

	// Generate shortened display names
	shortenProjectNames(projectFiles)

	// Display paths
	displayProjectFiles(projectFiles)
}

// displayProjectFiles outputs the project files in the requested format
func displayProjectFiles(projectFiles []projectFile) {
	if cfg.OutputFormat == "json" {
		displayProjectFilesJSON(projectFiles)
	} else {
		displayProjectFilesText(projectFiles)
	}
}

// displayProjectFilesJSON outputs project files in JSON format
func displayProjectFilesJSON(projectFiles []projectFile) {
	output := make([]map[string]interface{}, 0, len(projectFiles))
	for _, pf := range projectFiles {
		entry := map[string]interface{}{
			"path": pf.path,
			"name": pf.display,
		}
		// Always include basic info
		entry["updated_at"] = pf.modTime.Format(time.RFC3339)
		entry["size"] = pf.size
		entry["size_human"] = formatFileSize(pf.size)
		entry["decoded_path"] = pf.decoded
		output = append(output, entry)
	}
	jsonData, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(jsonData))
}

// displayProjectFilesText outputs project files in text format
func displayProjectFilesText(projectFiles []projectFile) {
	for _, pf := range projectFiles {
		// Simple tab-separated output for cut compatibility
		fmt.Printf("%s\t%s\t%s\n",
			pf.path,
			pf.modTime.Format("2006-01-02 15:04:05"),
			formatFileSize(pf.size))
	}
}

// formatFileSize formats bytes into human-readable format like ls -l
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// shortenProjectNames takes a list of project files and generates shortened display names
// It shows only the last directory name, but includes parent directories when there are duplicates
func shortenProjectNames(projectFiles []projectFile) {
	// First pass: count occurrences of last directory names
	lastDirCount := make(map[string]int)
	lastDirOnly := make([]string, len(projectFiles))

	for i, pf := range projectFiles {
		parts := strings.Split(pf.decoded, "/")
		if len(parts) > 0 {
			lastDir := parts[len(parts)-1]
			lastDirOnly[i] = lastDir
			lastDirCount[lastDir]++
		}
	}

	// Second pass: generate display names
	for i, pf := range projectFiles {
		parts := strings.Split(pf.decoded, "/")
		if len(parts) == 0 {
			projectFiles[i].display = pf.decoded
			continue
		}

		lastDir := parts[len(parts)-1]

		// If no duplicates, use only the last directory
		if lastDirCount[lastDir] == 1 {
			projectFiles[i].display = lastDir
		} else {
			// For duplicates, find the minimum number of parent directories needed
			// to make each path unique within the duplicate set
			displayName := lastDir

			// Keep adding parent directories until we have a unique display name
			for j := len(parts) - 2; j >= 0; j-- {
				displayName = parts[j] + "/" + displayName

				// Check if this display name is unique among all project files
				isUnique := true
				for k, otherPF := range projectFiles {
					if k == i {
						continue
					}
					// Only check against other files with the same last directory
					if lastDirOnly[k] == lastDir {
						if strings.HasSuffix(otherPF.decoded, displayName) {
							isUnique = false
							break
						}
					}
				}

				if isUnique {
					break
				}
			}

			projectFiles[i].display = displayName
		}
	}
}
