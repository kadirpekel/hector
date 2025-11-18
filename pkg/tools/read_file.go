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

type ReadFileTool struct {
	config *config.ReadFileConfig
}

func NewReadFileTool(cfg *config.ReadFileConfig) *ReadFileTool {
	if cfg == nil {
		cfg = &config.ReadFileConfig{
			MaxFileSize:      10485760, // 10MB default
			WorkingDirectory: "./",
			ShowLineNumbers:  config.BoolPtr(true),
		}
	}

	if cfg.MaxFileSize == 0 {
		cfg.MaxFileSize = 10485760
	}
	if cfg.WorkingDirectory == "" {
		cfg.WorkingDirectory = "./"
	}

	return &ReadFileTool{config: cfg}
}

func NewReadFileToolWithConfig(name string, toolConfig *config.ToolConfig) (*ReadFileTool, error) {
	if toolConfig == nil {
		return nil, fmt.Errorf("tool config is required")
	}

	cfg := &config.ReadFileConfig{
		MaxFileSize:      int(toolConfig.MaxFileSize),
		WorkingDirectory: toolConfig.WorkingDirectory,
	}

	cfg.SetDefaults()
	return NewReadFileTool(cfg), nil
}

func (t *ReadFileTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name:        "read_file",
		Description: "Read the contents of a file with optional line numbers and range selection. Use to understand code structure and context before making edits.",
		Parameters: []ToolParameter{
			{
				Name:        "path",
				Type:        "string",
				Description: "File path to read (relative to working directory)",
				Required:    true,
			},
			{
				Name:        "start_line",
				Type:        "number",
				Description: "Starting line number (1-indexed, optional)",
				Required:    false,
			},
			{
				Name:        "end_line",
				Type:        "number",
				Description: "Ending line number (inclusive, optional)",
				Required:    false,
			},
			{
				Name:        "line_numbers",
				Type:        "boolean",
				Description: "Include line numbers in output (default: true)",
				Required:    false,
				Default:     true,
			},
		},
		ServerURL: "local",
	}
}

func (t *ReadFileTool) GetName() string {
	return "read_file"
}

func (t *ReadFileTool) GetDescription() string {
	return "Read file contents with optional line numbers and range selection"
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	path, ok := args["path"].(string)
	if !ok || path == "" {
		return t.errorResult("path parameter is required", start),
			fmt.Errorf("path parameter is required")
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

	showLineNumbers := true
	if ln, ok := args["line_numbers"].(bool); ok {
		showLineNumbers = ln
	}

	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	startLine := 1
	if sl, ok := args["start_line"].(float64); ok {
		startLine = int(sl)
		if startLine < 1 {
			startLine = 1
		}
	}

	endLine := totalLines
	if el, ok := args["end_line"].(float64); ok {
		endLine = int(el)
		if endLine > totalLines {
			endLine = totalLines
		}
	}

	if startLine > endLine {
		return t.errorResult(
			fmt.Sprintf("invalid range: start_line (%d) > end_line (%d)", startLine, endLine),
			start), fmt.Errorf("invalid line range")
	}

	if startLine > totalLines {
		return t.errorResult(
			fmt.Sprintf("start_line (%d) exceeds file length (%d lines)", startLine, totalLines),
			start), fmt.Errorf("start_line out of range")
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("FILE: %s\n", path))
	output.WriteString(fmt.Sprintf("STATS: Total lines: %d", totalLines))

	if startLine != 1 || endLine != totalLines {
		output.WriteString(fmt.Sprintf(" | Showing lines %d-%d", startLine, endLine))
	}
	output.WriteString("\n")
	output.WriteString(strings.Repeat("─", 60) + "\n")

	for i := startLine - 1; i < endLine && i < len(lines); i++ {
		if showLineNumbers {
			output.WriteString(fmt.Sprintf("%6d| %s\n", i+1, lines[i]))
		} else {
			output.WriteString(fmt.Sprintf("%s\n", lines[i]))
		}
	}

	output.WriteString(strings.Repeat("─", 60))

	return ToolResult{
		Success:       true,
		Content:       output.String(),
		ToolName:      "read_file",
		ExecutionTime: time.Since(start),
		Metadata: map[string]interface{}{
			"path":         path,
			"total_lines":  totalLines,
			"start_line":   startLine,
			"end_line":     endLine,
			"lines_shown":  endLine - startLine + 1,
			"file_size":    fileInfo.Size(),
			"line_numbers": showLineNumbers,
		},
	}, nil
}

func (t *ReadFileTool) validatePath(path string) error {
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

func (t *ReadFileTool) errorResult(msg string, start time.Time) ToolResult {
	return ToolResult{
		Success:       false,
		Error:         msg,
		ToolName:      "read_file",
		ExecutionTime: time.Since(start),
	}
}
