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

type FileWriterTool struct {
	config *config.FileWriterConfig
}

func NewFileWriterTool(cfg *config.FileWriterConfig) *FileWriterTool {
	if cfg == nil {
		cfg = &config.FileWriterConfig{
			MaxFileSize:       1048576,
			AllowedExtensions: nil,
			BackupOnOverwrite: true,
			WorkingDirectory:  "./",
		}
	}

	if cfg.MaxFileSize == 0 {
		cfg.MaxFileSize = 1048576
	}

	if cfg.WorkingDirectory == "" {
		cfg.WorkingDirectory = "./"
	}

	return &FileWriterTool{config: cfg}
}

func NewFileWriterToolWithConfig(name string, toolConfig *config.ToolConfig) (*FileWriterTool, error) {
	if toolConfig == nil {
		return nil, fmt.Errorf("tool config is required")
	}

	cfg := &config.FileWriterConfig{
		MaxFileSize:       int(toolConfig.MaxFileSize),
		AllowedExtensions: toolConfig.AllowedExtensions,
		DeniedExtensions:  toolConfig.DeniedExtensions,
		WorkingDirectory:  toolConfig.WorkingDirectory,
	}

	cfg.SetDefaults()
	return NewFileWriterTool(cfg), nil
}

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

func (t *FileWriterTool) GetName() string {
	return "write_file"
}

func (t *FileWriterTool) GetDescription() string {
	return "Create a new file or overwrite an existing file with content"
}

func (t *FileWriterTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

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

	backup := true
	if b, ok := args["backup"].(bool); ok {
		backup = b
	}

	if err := t.validatePath(path); err != nil {
		return t.errorResult(err.Error(), start), err
	}

	if len(content) > t.config.MaxFileSize {
		return t.errorResult(
				fmt.Sprintf("content too large: %d bytes (max: %d)",
					len(content), t.config.MaxFileSize),
				start),
			fmt.Errorf("content exceeds max file size")
	}

	fullPath := filepath.Join(t.config.WorkingDirectory, path)

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

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return t.errorResult(
			fmt.Sprintf("failed to create directory: %v", err),
			start), err
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return t.errorResult(
			fmt.Sprintf("failed to write file: %v", err),
			start), err
	}

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

func (t *FileWriterTool) validatePath(path string) error {

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

	ext := filepath.Ext(path)

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

	if len(t.config.AllowedExtensions) > 0 {
		allowed := false
		for _, allowedExt := range t.config.AllowedExtensions {

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

	return nil
}

func (t *FileWriterTool) errorResult(msg string, start time.Time) ToolResult {
	return ToolResult{
		Success:       false,
		Error:         msg,
		ToolName:      "write_file",
		ExecutionTime: time.Since(start),
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
