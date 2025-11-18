package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
)

type GrepSearchTool struct {
	config *config.GrepSearchConfig
}

func NewGrepSearchTool(cfg *config.GrepSearchConfig) *GrepSearchTool {
	if cfg == nil {
		cfg = &config.GrepSearchConfig{
			MaxResults:       1000,
			MaxFileSize:      10485760, // 10MB
			WorkingDirectory: "./",
			ContextLines:     2,
		}
	}

	if cfg.MaxResults == 0 {
		cfg.MaxResults = 1000
	}
	if cfg.MaxFileSize == 0 {
		cfg.MaxFileSize = 10485760
	}
	if cfg.WorkingDirectory == "" {
		cfg.WorkingDirectory = "./"
	}

	return &GrepSearchTool{config: cfg}
}

func NewGrepSearchToolWithConfig(name string, toolConfig *config.ToolConfig) (*GrepSearchTool, error) {
	if toolConfig == nil {
		return nil, fmt.Errorf("tool config is required")
	}

	cfg := &config.GrepSearchConfig{
		MaxResults:       toolConfig.MaxResults,
		MaxFileSize:      int(toolConfig.MaxFileSize),
		WorkingDirectory: toolConfig.WorkingDirectory,
		ContextLines:     toolConfig.ContextLines,
	}

	cfg.SetDefaults()
	return NewGrepSearchTool(cfg), nil
}

func (t *GrepSearchTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name:        "grep_search",
		Description: "Search for patterns in files using regular expressions. Like Unix grep but with context lines. Use for finding exact strings, symbols, or regex patterns across files.",
		Parameters: []ToolParameter{
			{
				Name:        "pattern",
				Type:        "string",
				Description: "Regular expression pattern to search for (supports Go regex syntax)",
				Required:    true,
			},
			{
				Name:        "path",
				Type:        "string",
				Description: "File or directory path to search in (relative to working directory)",
				Required:    false,
				Default:     ".",
			},
			{
				Name:        "file_pattern",
				Type:        "string",
				Description: "File glob pattern to filter files (e.g., '*.go', '*.py')",
				Required:    false,
			},
			{
				Name:        "case_insensitive",
				Type:        "boolean",
				Description: "Perform case-insensitive search (default: false)",
				Required:    false,
				Default:     false,
			},
			{
				Name:        "context_lines",
				Type:        "number",
				Description: "Number of context lines to show before and after matches (default: 2)",
				Required:    false,
				Default:     2,
			},
			{
				Name:        "max_results",
				Type:        "number",
				Description: "Maximum number of matches to return (default: 100)",
				Required:    false,
				Default:     100,
			},
			{
				Name:        "recursive",
				Type:        "boolean",
				Description: "Search recursively in directories (default: true)",
				Required:    false,
				Default:     true,
			},
		},
		ServerURL: "local",
	}
}

func (t *GrepSearchTool) GetName() string {
	return "grep_search"
}

func (t *GrepSearchTool) GetDescription() string {
	return "Search for regex patterns in files with context lines"
}

func (t *GrepSearchTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return t.errorResult("pattern parameter is required", start),
			fmt.Errorf("pattern parameter is required")
	}

	searchPath := "."
	if p, ok := args["path"].(string); ok && p != "" {
		searchPath = p
	}

	caseInsensitive := false
	if ci, ok := args["case_insensitive"].(bool); ok {
		caseInsensitive = ci
	}

	contextLines := t.config.ContextLines
	if cl, ok := args["context_lines"].(float64); ok {
		contextLines = int(cl)
	}

	maxResults := 100
	if mr, ok := args["max_results"].(float64); ok {
		maxResults = int(mr)
	}
	if maxResults > t.config.MaxResults {
		maxResults = t.config.MaxResults
	}

	recursive := true
	if r, ok := args["recursive"].(bool); ok {
		recursive = r
	}

	filePattern := ""
	if fp, ok := args["file_pattern"].(string); ok {
		filePattern = fp
	}

	if caseInsensitive {
		pattern = "(?i)" + pattern
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return t.errorResult(fmt.Sprintf("invalid regex pattern: %v", err), start), err
	}

	fullPath := filepath.Join(t.config.WorkingDirectory, searchPath)
	if err := t.validatePath(searchPath); err != nil {
		return t.errorResult(err.Error(), start), err
	}

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return t.errorResult(fmt.Sprintf("failed to stat path: %v", err), start), err
	}

	var filesToSearch []string
	if fileInfo.IsDir() {
		if recursive {
			_ = filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // skip errors
				}
				if !info.IsDir() && info.Size() <= int64(t.config.MaxFileSize) {
					if filePattern == "" || t.matchesPattern(filepath.Base(path), filePattern) {
						relPath, _ := filepath.Rel(t.config.WorkingDirectory, path)
						filesToSearch = append(filesToSearch, relPath)
					}
				}
				return nil
			})
		} else {
			entries, err := os.ReadDir(fullPath)
			if err == nil {
				for _, entry := range entries {
					if !entry.IsDir() {
						if info, err := entry.Info(); err == nil && info.Size() <= int64(t.config.MaxFileSize) {
							fileName := entry.Name()
							if filePattern == "" || t.matchesPattern(fileName, filePattern) {
								relPath := filepath.Join(searchPath, fileName)
								filesToSearch = append(filesToSearch, relPath)
							}
						}
					}
				}
			}
		}
	} else {
		filesToSearch = append(filesToSearch, searchPath)
	}

	results := []map[string]interface{}{}
	totalMatches := 0

	for _, filePath := range filesToSearch {
		if totalMatches >= maxResults {
			break
		}

		matches, err := t.searchFile(filePath, regex, contextLines)
		if err != nil {
			continue // skip files with errors
		}

		for _, match := range matches {
			if totalMatches >= maxResults {
				break
			}
			match["file"] = filePath
			results = append(results, match)
			totalMatches++
		}
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("PATTERN: %s\n", pattern))
	output.WriteString(fmt.Sprintf("SEARCH_PATH: %s\n", searchPath))
	output.WriteString(fmt.Sprintf("STATS: Found %d matches in %d files\n", totalMatches, len(results)))
	output.WriteString(strings.Repeat("─", 60) + "\n")

	if len(results) == 0 {
		output.WriteString("\nNo matches found.\n")
	} else {
		currentFile := ""
		for _, result := range results {
			file := result["file"].(string)
			lineNum := result["line"].(int)
			line := result["content"].(string)
			context := result["context"].([]string)

			if file != currentFile {
				if currentFile != "" {
					output.WriteString("\n")
				}
				output.WriteString(fmt.Sprintf("\nFILE: %s\n", file))
				currentFile = file
			}

			if len(context) > 0 {
				for _, ctx := range context {
					output.WriteString(fmt.Sprintf("  %s\n", ctx))
				}
			}

			output.WriteString(fmt.Sprintf("→ %d: %s\n", lineNum, line))
		}
	}

	if totalMatches >= maxResults {
		output.WriteString(fmt.Sprintf("\nWARN: Results limited to %d matches\n", maxResults))
	}

	return ToolResult{
		Success:       true,
		Content:       output.String(),
		ToolName:      "grep_search",
		ExecutionTime: time.Since(start),
		Metadata: map[string]interface{}{
			"pattern":          pattern,
			"path":             searchPath,
			"total_matches":    totalMatches,
			"files_searched":   len(filesToSearch),
			"case_insensitive": caseInsensitive,
			"recursive":        recursive,
			"truncated":        totalMatches >= maxResults,
		},
		Output: results,
	}, nil
}

func (t *GrepSearchTool) searchFile(filePath string, regex *regexp.Regexp, contextLines int) ([]map[string]interface{}, error) {
	fullPath := filepath.Join(t.config.WorkingDirectory, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	results := []map[string]interface{}{}

	for i, line := range lines {
		if regex.MatchString(line) {
			context := []string{}

			// Add context before
			for j := contextLines; j > 0; j-- {
				if i-j >= 0 {
					context = append(context, fmt.Sprintf("%6d  %s", i-j+1, lines[i-j]))
				}
			}

			results = append(results, map[string]interface{}{
				"line":    i + 1,
				"content": line,
				"context": context,
			})
		}
	}

	return results, nil
}

func (t *GrepSearchTool) matchesPattern(filename, pattern string) bool {
	matched, err := filepath.Match(pattern, filename)
	if err != nil {
		return false
	}
	return matched
}

func (t *GrepSearchTool) validatePath(path string) error {
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed, use relative paths")
	}

	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("directory traversal not allowed (..)")
	}

	absPath, err := filepath.Abs(filepath.Join(t.config.WorkingDirectory, cleaned))
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	absWorkDir, err := filepath.Abs(t.config.WorkingDirectory)
	if err != nil {
		return fmt.Errorf("invalid working directory: %w", err)
	}

	if !strings.HasPrefix(absPath, absWorkDir) {
		return fmt.Errorf("path escapes working directory")
	}

	return nil
}

func (t *GrepSearchTool) errorResult(msg string, start time.Time) ToolResult {
	return ToolResult{
		Success:       false,
		Error:         msg,
		ToolName:      "grep_search",
		ExecutionTime: time.Since(start),
	}
}
