package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
)

type SearchReplaceTool struct {
	config *config.SearchReplaceConfig
}

func NewSearchReplaceTool(cfg *config.SearchReplaceConfig) *SearchReplaceTool {
	if cfg == nil {
		cfg = &config.SearchReplaceConfig{
			MaxReplacements:  100,
			ShowDiff:         config.BoolPtr(true),
			CreateBackup:     config.BoolPtr(true),
			WorkingDirectory: "./",
		}
	}

	if cfg.MaxReplacements == 0 {
		cfg.MaxReplacements = 100
	}
	if cfg.WorkingDirectory == "" {
		cfg.WorkingDirectory = "./"
	}

	return &SearchReplaceTool{config: cfg}
}

func NewSearchReplaceToolWithConfig(name string, toolConfig *config.ToolConfig) (*SearchReplaceTool, error) {
	if toolConfig == nil {
		return nil, fmt.Errorf("tool config is required")
	}

	cfg := &config.SearchReplaceConfig{
		MaxReplacements:  toolConfig.MaxReplacements,
		WorkingDirectory: toolConfig.WorkingDirectory,
	}

	cfg.SetDefaults()
	return NewSearchReplaceTool(cfg), nil
}

func (t *SearchReplaceTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name:        "search_replace",
		Description: "Replace exact text in a file. Preserves formatting and indentation. Use for precise edits.",
		Parameters: []ToolParameter{
			{
				Name:        "path",
				Type:        "string",
				Description: "File path to edit (relative to working directory)",
				Required:    true,
			},
			{
				Name:        "old_string",
				Type:        "string",
				Description: "Exact text to find (must be unique unless replace_all=true)",
				Required:    true,
			},
			{
				Name:        "new_string",
				Type:        "string",
				Description: "Replacement text",
				Required:    true,
			},
			{
				Name:        "replace_all",
				Type:        "boolean",
				Description: "Replace all occurrences (default: false, requires unique match)",
				Required:    false,
				Default:     false,
			},
		},
		ServerURL: "local",
	}
}

func (t *SearchReplaceTool) GetName() string {
	return "search_replace"
}

func (t *SearchReplaceTool) GetDescription() string {
	return "Replace exact text in a file (preserves formatting)"
}

func (t *SearchReplaceTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	path, ok := args["path"].(string)
	if !ok || path == "" {
		return t.errorResult("path parameter is required", start),
			fmt.Errorf("path parameter is required")
	}

	oldString, ok := args["old_string"].(string)
	if !ok || oldString == "" {
		return t.errorResult("old_string parameter is required", start),
			fmt.Errorf("old_string parameter is required")
	}

	newString, ok := args["new_string"].(string)
	if !ok {
		return t.errorResult("new_string parameter is required", start),
			fmt.Errorf("new_string parameter is required")
	}

	replaceAll := false
	if ra, ok := args["replace_all"].(bool); ok {
		replaceAll = ra
	}

	fullPath := filepath.Join(t.config.WorkingDirectory, path)
	if err := t.validatePath(path); err != nil {
		return t.errorResult(err.Error(), start), err
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return t.errorResult(fmt.Sprintf("failed to read file: %v", err), start), err
	}

	originalContent := string(content)

	if !strings.Contains(originalContent, oldString) {
		return t.errorResult(
				fmt.Sprintf("old_string not found in file: '%s'", truncateString(oldString, 50)),
				start),
			fmt.Errorf("old_string not found")
	}

	count := strings.Count(originalContent, oldString)
	if !replaceAll && count > 1 {
		return t.errorResult(
				fmt.Sprintf("old_string appears %d times - must be unique or use replace_all=true", count),
				start),
			fmt.Errorf("ambiguous replacement: %d occurrences", count)
	}

	if count > t.config.MaxReplacements {
		return t.errorResult(
				fmt.Sprintf("too many replacements: %d (max: %d)", count, t.config.MaxReplacements),
				start),
			fmt.Errorf("exceeds max replacements")
	}

	var newContent string
	replacementCount := 0
	if replaceAll {
		newContent = strings.ReplaceAll(originalContent, oldString, newString)
		replacementCount = count
	} else {
		newContent = strings.Replace(originalContent, oldString, newString, 1)
		replacementCount = 1
	}

	if t.config.CreateBackup != nil && *t.config.CreateBackup {
		backupPath := fullPath + ".bak"
		if err := os.WriteFile(backupPath, content, 0644); err != nil {

			fmt.Printf("Warning: failed to create backup: %v\n", err)
		}
	}

	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return t.errorResult(fmt.Sprintf("failed to write file: %v", err), start), err
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("SUCCESS: Replaced %d occurrence(s) in %s\n", replacementCount, path))

	if t.config.ShowDiff != nil && *t.config.ShowDiff {
		diff := t.generateDiff(oldString, newString)
		response.WriteString(fmt.Sprintf("\n%s\n", diff))
	}

	if t.config.CreateBackup != nil && *t.config.CreateBackup {
		response.WriteString(fmt.Sprintf("\nBACKUP: Backup created: %s.bak", path))
	}

	return ToolResult{
		Success:       true,
		Content:       response.String(),
		ToolName:      "search_replace",
		ExecutionTime: time.Since(start),
		Metadata: map[string]interface{}{
			"path":         path,
			"replacements": replacementCount,
			"replace_all":  replaceAll,
			"backed_up":    config.BoolValue(t.config.CreateBackup, false),
			"old_length":   len(oldString),
			"new_length":   len(newString),
			"size_change":  len(newContent) - len(originalContent),
		},
	}, nil
}

func (t *SearchReplaceTool) validatePath(path string) error {
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed")
	}

	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("directory traversal not allowed")
	}

	fullPath := filepath.Join(t.config.WorkingDirectory, path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}

	return nil
}

func (t *SearchReplaceTool) generateDiff(oldStr, newStr string) string {
	var diff strings.Builder

	diff.WriteString("CHANGES:\n")
	diff.WriteString(strings.Repeat("-", 60) + "\n")

	oldLines := strings.Split(oldStr, "\n")
	for _, line := range oldLines {
		if line != "" {
			diff.WriteString(fmt.Sprintf("- %s\n", line))
		}
	}

	newLines := strings.Split(newStr, "\n")
	for _, line := range newLines {
		if line != "" {
			diff.WriteString(fmt.Sprintf("+ %s\n", line))
		}
	}

	diff.WriteString(strings.Repeat("-", 60))

	return diff.String()
}

func (t *SearchReplaceTool) errorResult(msg string, start time.Time) ToolResult {
	return ToolResult{
		Success:       false,
		Error:         msg,
		ToolName:      "search_replace",
		ExecutionTime: time.Since(start),
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
