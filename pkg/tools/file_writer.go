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

// ============================================================================
// FILE WRITER - CREATE AND MODIFY FILES
// ============================================================================

// FileWriterTool handles secure file creation and modification
type FileWriterTool struct {
	config *config.FileWriterConfig
}

// NewFileWriterTool creates a new file writer tool with secure defaults
func NewFileWriterTool(cfg *config.FileWriterConfig) *FileWriterTool {
	if cfg == nil {
		cfg = &config.FileWriterConfig{
			MaxFileSize:       1048576, // 1MB default
			AllowedExtensions: nil,     // nil = allow all (default behavior)
			BackupOnOverwrite: true,
			WorkingDirectory:  "./",
		}
	}

	// Apply defaults if not set
	if cfg.MaxFileSize == 0 {
		cfg.MaxFileSize = 1048576
	}
	// Note: Empty AllowedExtensions means "allow all" (not restricted)
	// To restrict, user must explicitly set allowed_extensions in config
	if cfg.WorkingDirectory == "" {
		cfg.WorkingDirectory = "./"
	}

	return &FileWriterTool{config: cfg}
}

// NewFileWriterToolWithConfig creates a file writer tool from a ToolConfig configuration
func NewFileWriterToolWithConfig(name string, toolConfig config.ToolConfig) (*FileWriterTool, error) {
	cfg := &config.FileWriterConfig{
		MaxFileSize:       int(toolConfig.MaxFileSize),
		AllowedExtensions: toolConfig.AllowedExtensions,
		DeniedExtensions:  toolConfig.DeniedExtensions,
		WorkingDirectory:  toolConfig.WorkingDirectory,
	}

	cfg.SetDefaults()
	return NewFileWriterTool(cfg), nil
}

// GetInfo returns tool metadata for the Tool interface
func (t *FileWriterTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name:        "write_file",
		Description: "Create a new file or overwrite an existing file with content. Supports backups and safety checks.",
		Parameters: []ToolParameter{
			{
				Name:        "path",
				Type:        "string",
				Description: "File path relative to working directory",
				Required:    true,
			},
			{
				Name:        "content",
				Type:        "string",
				Description: "Content to write to the file",
				Required:    true,
			},
			{
				Name:        "backup",
				Type:        "boolean",
				Description: "Create .bak backup if file exists (default: true)",
				Required:    false,
				Default:     true,
			},
		},
		ServerURL: "local",
	}
}

// GetName returns the tool name
func (t *FileWriterTool) GetName() string {
	return "write_file"
}

// GetDescription returns the tool description
func (t *FileWriterTool) GetDescription() string {
	return "Create a new file or overwrite an existing file with content"
}

// Execute writes the file with safety checks
func (t *FileWriterTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	// Extract parameters
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return t.errorResult("path parameter is required", start),
			fmt.Errorf("path parameter is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return t.errorResult("content parameter is required", start),
			fmt.Errorf("content parameter is required")
	}

	// Default backup to true
	backup := true
	if b, ok := args["backup"].(bool); ok {
		backup = b
	}

	// Validate path
	if err := t.validatePath(path); err != nil {
		return t.errorResult(err.Error(), start), err
	}

	// Validate content size
	if len(content) > t.config.MaxFileSize {
		return t.errorResult(
				fmt.Sprintf("content too large: %d bytes (max: %d)",
					len(content), t.config.MaxFileSize),
				start),
			fmt.Errorf("content exceeds max file size")
	}

	// Full path
	fullPath := filepath.Join(t.config.WorkingDirectory, path)

	// Create backup if file exists and backup is enabled
	fileExisted := false
	if backup && t.config.BackupOnOverwrite {
		if _, err := os.Stat(fullPath); err == nil {
			fileExisted = true
			backupPath := fullPath + ".bak"
			if err := copyFile(fullPath, backupPath); err != nil {
				return t.errorResult(
					fmt.Sprintf("failed to create backup: %v", err),
					start), err
			}
		}
	}

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return t.errorResult(
			fmt.Sprintf("failed to create directory: %v", err),
			start), err
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return t.errorResult(
			fmt.Sprintf("failed to write file: %v", err),
			start), err
	}

	// Success message
	action := "created"
	if fileExisted {
		action = "overwritten"
	}
	message := fmt.Sprintf("File %s successfully: %s (%d bytes)", action, path, len(content))
	if fileExisted && backup {
		message += fmt.Sprintf("\nBackup created: %s.bak", path)
	}

	return ToolResult{
		Success:       true,
		Content:       message,
		ToolName:      "write_file",
		ExecutionTime: time.Since(start),
		Metadata: map[string]interface{}{
			"path":         path,
			"size":         len(content),
			"backed_up":    fileExisted && backup,
			"file_existed": fileExisted,
			"action":       action,
		},
	}, nil
}

// validatePath checks if the path is safe to write to
func (t *FileWriterTool) validatePath(path string) error {
	// Prevent absolute paths
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed, use relative paths")
	}

	// Clean the path and check for directory traversal
	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("directory traversal not allowed (..)")
	}

	// Check if path tries to escape working directory
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

	// Extension validation logic:
	// DEFAULT: Allow ALL extensions (most permissive, zero-config friendly)
	// - If denied_extensions set: Block those extensions (blacklist)
	// - If allowed_extensions set: ONLY allow those extensions (whitelist)
	// - Whitelist takes precedence over default permissive behavior

	ext := filepath.Ext(path)

	// Check denied_extensions first (blacklist)
	if len(t.config.DeniedExtensions) > 0 {
		for _, deniedExt := range t.config.DeniedExtensions {
			if ext == deniedExt || (ext == "" && deniedExt == "") {
				if ext == "" {
					return fmt.Errorf("extensionless files are explicitly denied")
				}
				return fmt.Errorf("file extension %s is explicitly denied", ext)
			}
		}
	}

	// If allowed_extensions is configured, enforce whitelist (overrides default permissive behavior)
	if len(t.config.AllowedExtensions) > 0 {
		allowed := false
		for _, allowedExt := range t.config.AllowedExtensions {
			// Check for exact match (including extensionless files with "")
			if ext == allowedExt {
				allowed = true
				break
			}
		}
		if !allowed {
			if ext == "" {
				return fmt.Errorf("extensionless files not allowed (add '' to allowed_extensions to allow Makefile, Dockerfile, etc.)")
			}
			return fmt.Errorf("file extension %s not allowed (allowed: %v)", ext, t.config.AllowedExtensions)
		}
	}

	// DEFAULT: If neither allowed_extensions nor denied_extensions configured,
	// allow all extensions (zero-config, permissive by default)

	return nil
}

// errorResult creates a standardized error result
func (t *FileWriterTool) errorResult(msg string, start time.Time) ToolResult {
	return ToolResult{
		Success:       false,
		Error:         msg,
		ToolName:      "write_file",
		ExecutionTime: time.Since(start),
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
