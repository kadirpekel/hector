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

type ApplyPatchTool struct {
	config *config.ApplyPatchConfig
}

func NewApplyPatchTool(cfg *config.ApplyPatchConfig) *ApplyPatchTool {
	if cfg == nil {
		cfg = &config.ApplyPatchConfig{
			MaxFileSize:      10485760, // 10MB
			CreateBackup:     config.BoolPtr(true),
			ContextLines:     3,
			WorkingDirectory: "./",
		}
	}

	if cfg.MaxFileSize == 0 {
		cfg.MaxFileSize = 10485760
	}
	if cfg.ContextLines == 0 {
		cfg.ContextLines = 3
	}
	if cfg.WorkingDirectory == "" {
		cfg.WorkingDirectory = "./"
	}

	return &ApplyPatchTool{config: cfg}
}

func NewApplyPatchToolWithConfig(name string, toolConfig *config.ToolConfig) (*ApplyPatchTool, error) {
	if toolConfig == nil {
		return nil, fmt.Errorf("tool config is required")
	}

	cfg := &config.ApplyPatchConfig{
		MaxFileSize:      int(toolConfig.MaxFileSize),
		WorkingDirectory: toolConfig.WorkingDirectory,
		ContextLines:     toolConfig.ContextLines,
	}

	cfg.SetDefaults()
	return NewApplyPatchTool(cfg), nil
}

func (t *ApplyPatchTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name:        "apply_patch",
		Description: "Apply a patch to a file by finding and replacing text with surrounding context. More robust than search_replace for code edits. Validates context before applying changes.",
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
				Description: "Text to find with sufficient surrounding context (3-5 lines before and after the change)",
				Required:    true,
			},
			{
				Name:        "new_string",
				Type:        "string",
				Description: "Replacement text (should include the same context as old_string)",
				Required:    true,
			},
			{
				Name:        "context_validation",
				Type:        "boolean",
				Description: "Validate that surrounding context matches (default: true, recommended for safety)",
				Required:    false,
				Default:     true,
			},
		},
		ServerURL: "local",
	}
}

func (t *ApplyPatchTool) GetName() string {
	return "apply_patch"
}

func (t *ApplyPatchTool) GetDescription() string {
	return "Apply a contextual patch to a file (safer than search_replace for code edits)"
}

func (t *ApplyPatchTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
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

	contextValidation := true
	if cv, ok := args["context_validation"].(bool); ok {
		contextValidation = cv
	}

	fullPath := filepath.Join(t.config.WorkingDirectory, path)
	if err := t.validatePath(path); err != nil {
		return t.errorResult(err.Error(), start), err
	}

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return t.errorResult(fmt.Sprintf("failed to stat file: %v", err), start), err
	}

	if fileInfo.Size() > int64(t.config.MaxFileSize) {
		return t.errorResult(
			fmt.Sprintf("file too large: %d bytes (max: %d)", fileInfo.Size(), t.config.MaxFileSize),
			start), fmt.Errorf("file exceeds max size")
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return t.errorResult(fmt.Sprintf("failed to read file: %v", err), start), err
	}

	originalContent := string(content)

	if !strings.Contains(originalContent, oldString) {
		return t.errorResult(
			"patch context not found in file. The old_string must match exactly including whitespace.",
			start), fmt.Errorf("patch not applicable")
	}

	count := strings.Count(originalContent, oldString)
	if count > 1 {
		return t.errorResult(
			fmt.Sprintf("ambiguous patch: old_string appears %d times. Add more context to make it unique.", count),
			start), fmt.Errorf("ambiguous patch location")
	}

	if contextValidation {
		if err := t.validateContextLines(oldString, newString); err != nil {
			return t.errorResult(fmt.Sprintf("context validation failed: %v", err), start), err
		}
	}

	newContent := strings.Replace(originalContent, oldString, newString, 1)

	if t.config.CreateBackup != nil && *t.config.CreateBackup {
		backupPath := fullPath + ".bak"
		if err := os.WriteFile(backupPath, content, 0644); err != nil {
			fmt.Printf("Warning: failed to create backup: %v\n", err)
		}
	}

	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return t.errorResult(fmt.Sprintf("failed to write file: %v", err), start), err
	}

	oldLines := strings.Split(oldString, "\n")
	newLines := strings.Split(newString, "\n")

	var response strings.Builder
	response.WriteString(fmt.Sprintf("SUCCESS: Patch applied successfully to %s\n", path))
	response.WriteString(fmt.Sprintf("CHANGED: Changed %d lines\n", len(oldLines)))
	response.WriteString("\n")
	response.WriteString(t.generateDiff(oldString, newString))

	if t.config.CreateBackup != nil && *t.config.CreateBackup {
		response.WriteString(fmt.Sprintf("\nBACKUP: Backup created: %s.bak", path))
	}

	return ToolResult{
		Success:       true,
		Content:       response.String(),
		ToolName:      "apply_patch",
		ExecutionTime: time.Since(start),
		Metadata: map[string]interface{}{
			"path":              path,
			"old_lines":         len(oldLines),
			"new_lines":         len(newLines),
			"size_change":       len(newContent) - len(originalContent),
			"backed_up":         config.BoolValue(t.config.CreateBackup, false),
			"context_validated": contextValidation,
		},
	}, nil
}

func (t *ApplyPatchTool) validateContextLines(oldString, newString string) error {
	oldLines := strings.Split(oldString, "\n")
	newLines := strings.Split(newString, "\n")

	minContextLines := t.config.ContextLines
	if len(oldLines) < minContextLines*2+1 {
		return fmt.Errorf("insufficient context: provide at least %d lines before and after the change", minContextLines)
	}

	contextMatches := 0
	for i := 0; i < minContextLines && i < len(oldLines) && i < len(newLines); i++ {
		if oldLines[i] == newLines[i] {
			contextMatches++
		}
	}

	for i := 1; i <= minContextLines && i <= len(oldLines) && i <= len(newLines); i++ {
		oldIdx := len(oldLines) - i
		newIdx := len(newLines) - i
		if oldIdx >= 0 && newIdx >= 0 && oldLines[oldIdx] == newLines[newIdx] {
			contextMatches++
		}
	}

	if contextMatches < minContextLines {
		return fmt.Errorf("context mismatch: ensure old_string and new_string have matching surrounding lines")
	}

	return nil
}

func (t *ApplyPatchTool) validatePath(path string) error {
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

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}

	return nil
}

func (t *ApplyPatchTool) generateDiff(oldStr, newStr string) string {
	var diff strings.Builder

	diff.WriteString("Changes:\n")
	diff.WriteString(strings.Repeat("-", 60) + "\n")

	oldLines := strings.Split(oldStr, "\n")
	newLines := strings.Split(newStr, "\n")

	maxLines := len(oldLines)
	if len(newLines) > maxLines {
		maxLines = len(newLines)
	}

	for i := 0; i < maxLines; i++ {
		if i < len(oldLines) && i < len(newLines) {
			if oldLines[i] != newLines[i] {
				diff.WriteString(fmt.Sprintf("- %s\n", oldLines[i]))
				diff.WriteString(fmt.Sprintf("+ %s\n", newLines[i]))
			} else {
				diff.WriteString(fmt.Sprintf("  %s\n", oldLines[i]))
			}
		} else if i < len(oldLines) {
			diff.WriteString(fmt.Sprintf("- %s\n", oldLines[i]))
		} else if i < len(newLines) {
			diff.WriteString(fmt.Sprintf("+ %s\n", newLines[i]))
		}
	}

	diff.WriteString(strings.Repeat("-", 60))

	return diff.String()
}

func (t *ApplyPatchTool) errorResult(msg string, start time.Time) ToolResult {
	return ToolResult{
		Success:       false,
		Error:         msg,
		ToolName:      "apply_patch",
		ExecutionTime: time.Since(start),
	}
}
