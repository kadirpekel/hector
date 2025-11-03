package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
)

func TestNewFileWriterToolForTesting(t *testing.T) {
	tool := NewFileWriterToolForTesting()
	if tool == nil {
		t.Fatal("NewFileWriterToolForTesting() returned nil")
	}

	if tool.GetName() != "write_file" {
		t.Errorf("GetName() = %v, want 'write_file'", tool.GetName())
	}

	description := tool.GetDescription()
	if description == "" {
		t.Error("GetDescription() should not return empty string")
	}
}

func TestFileWriterTool_GetInfo(t *testing.T) {
	tool := NewFileWriterToolForTesting()
	info := tool.GetInfo()

	if info.Name == "" {
		t.Fatal("GetInfo() returned empty name")
	}

	if info.Description == "" {
		t.Error("Expected non-empty description")
	}
	if len(info.Parameters) == 0 {
		t.Error("Expected at least one parameter")
	}

	hasPathParam := false
	hasContentParam := false
	for _, param := range info.Parameters {
		if param.Name == "path" && param.Required {
			hasPathParam = true
		}
		if param.Name == "content" && param.Required {
			hasContentParam = true
		}
	}
	if !hasPathParam {
		t.Error("Expected 'path' parameter to be required")
	}
	if !hasContentParam {
		t.Error("Expected 'content' parameter to be required")
	}
}

func TestFileWriterTool_ValidatePath(t *testing.T) {
	tool := NewFileWriterToolForTesting()

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid relative path",
			path:    "test.txt",
			wantErr: false,
		},
		{
			name:    "valid nested path",
			path:    "subdir/test.txt",
			wantErr: false,
		},
		{
			name:    "absolute path not allowed",
			path:    "/absolute/path.txt",
			wantErr: true,
			errMsg:  "absolute paths not allowed",
		},
		{
			name:    "directory traversal not allowed",
			path:    "../outside.txt",
			wantErr: true,
			errMsg:  "directory traversal not allowed",
		},
		{
			name:    "double directory traversal",
			path:    "subdir/../../outside.txt",
			wantErr: true,
			errMsg:  "directory traversal not allowed",
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "path without extension",
			path:    "test",
			wantErr: true,
			errMsg:  "extensionless files not allowed",
		},
		{
			name:    "disallowed extension",
			path:    "test.exe",
			wantErr: true,
			errMsg:  "file extension .exe not allowed",
		},
		{
			name:    "allowed extension",
			path:    "test.txt",
			wantErr: false,
		},
		{
			name:    "allowed go extension",
			path:    "test.go",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tool.validatePath(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected validation error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestFileWriterTool_Execute_ValidationOnly(t *testing.T) {
	tool := NewFileWriterToolForTesting()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing path parameter",
			args: map[string]interface{}{
				"content": "test content",
			},
			wantErr: true,
			errMsg:  "path parameter is required",
		},
		{
			name: "missing content parameter",
			args: map[string]interface{}{
				"path": "test.txt",
			},
			wantErr: true,
			errMsg:  "content parameter is required",
		},
		{
			name: "empty path",
			args: map[string]interface{}{
				"path":    "",
				"content": "test content",
			},
			wantErr: true,
			errMsg:  "path parameter is required",
		},
		{
			name: "invalid path type",
			args: map[string]interface{}{
				"path":    123,
				"content": "test content",
			},
			wantErr: true,
			errMsg:  "path parameter is required",
		},
		{
			name: "invalid content type",
			args: map[string]interface{}{
				"path":    "test.txt",
				"content": 123,
			},
			wantErr: true,
			errMsg:  "content parameter is required",
		},
		{
			name: "content too large",
			args: map[string]interface{}{
				"path":    "test.txt",
				"content": strings.Repeat("a", 2000),
			},
			wantErr: true,
			errMsg:  "content exceeds max file size",
		},
		{
			name: "valid parameters structure",
			args: map[string]interface{}{
				"path":    "test.txt",
				"content": "test content",
			},
			wantErr: false,
		},
		{
			name: "backup parameter",
			args: map[string]interface{}{
				"path":    "test.txt",
				"content": "test content",
				"backup":  false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := tool.Execute(ctx, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected validation error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
				}
			} else {

				if err != nil && strings.Contains(err.Error(), "parameter is required") {
					t.Errorf("Expected file system error, got validation error: %v", err)
				}
			}
		})
	}
}

func TestFileWriterTool_ErrorResult(t *testing.T) {
	tool := NewFileWriterToolForTesting()

	result := tool.errorResult("test error message", time.Now())

	if result.Success {
		t.Error("Expected error result to have Success=false")
	}
	if result.Error != "test error message" {
		t.Errorf("Expected error message 'test error message', got: %s", result.Error)
	}
	if result.ToolName != "write_file" {
		t.Errorf("Expected tool name 'write_file', got: %s", result.ToolName)
	}
}

func TestFileWriterTool_DefaultAllowAll(t *testing.T) {

	tempDir, err := os.MkdirTemp("", "filewriter_default_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tool := NewFileWriterTool(&config.FileWriterConfig{
		MaxFileSize:       1024,
		AllowedExtensions: nil,
		BackupOnOverwrite: config.BoolPtr(false),
		WorkingDirectory:  tempDir,
	})

	tests := []struct {
		name        string
		path        string
		content     string
		wantSuccess bool
	}{
		{
			name:        "python file allowed by default",
			path:        "test.py",
			content:     "print('hello')",
			wantSuccess: true,
		},
		{
			name:        "toml file allowed by default",
			path:        "pyproject.toml",
			content:     "[tool.poetry]\nname = \"test\"",
			wantSuccess: true,
		},
		{
			name:        "extensionless file (Makefile) allowed by default",
			path:        "Makefile",
			content:     "all:\n\techo test",
			wantSuccess: true,
		},
		{
			name:        "dockerfile allowed by default",
			path:        "Dockerfile",
			content:     "FROM alpine",
			wantSuccess: true,
		},
		{
			name:        ".mod file allowed by default",
			path:        "go.mod",
			content:     "module test",
			wantSuccess: true,
		},
		{
			name:        ".lock file allowed by default",
			path:        "Cargo.lock",
			content:     "# cargo lock",
			wantSuccess: true,
		},
		{
			name:        ".ini file allowed by default",
			path:        "setup.cfg",
			content:     "[metadata]\nname = test",
			wantSuccess: true,
		},
		{
			name:        ".exe file allowed by default",
			path:        "test.exe",
			content:     "binary",
			wantSuccess: true,
		},
		{
			name:        ".bat file allowed by default",
			path:        "script.bat",
			content:     "@echo off",
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			args := map[string]interface{}{
				"path":    tt.path,
				"content": tt.content,
				"backup":  false,
			}

			result, err := tool.Execute(ctx, args)

			if tt.wantSuccess {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
					return
				}
				if !result.Success {
					t.Errorf("Expected success=true, got: %v", result.Success)
				}

				filePath := filepath.Join(tempDir, tt.path)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("Expected file %s to be created", tt.path)
				}
			} else {
				if err == nil {
					t.Error("Execute() expected error, got nil")
				}
			}
		})
	}
}

func TestFileWriterTool_WhitelistRestrictions(t *testing.T) {

	tempDir, err := os.MkdirTemp("", "filewriter_whitelist_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tool := NewFileWriterTool(&config.FileWriterConfig{
		MaxFileSize:       1024,
		AllowedExtensions: []string{".txt", ".md", ""},
		BackupOnOverwrite: config.BoolPtr(false),
		WorkingDirectory:  tempDir,
	})

	tests := []struct {
		name        string
		path        string
		wantSuccess bool
		wantError   string
	}{
		{
			name:        "allowed .txt file (in whitelist)",
			path:        "test.txt",
			wantSuccess: true,
		},
		{
			name:        "allowed .md file (in whitelist)",
			path:        "README.md",
			wantSuccess: true,
		},
		{
			name:        "allowed extensionless file (in whitelist)",
			path:        "Makefile",
			wantSuccess: true,
		},
		{
			name:        "disallowed .py file (not in whitelist)",
			path:        "test.py",
			wantSuccess: false,
			wantError:   "file extension .py not allowed",
		},
		{
			name:        "disallowed .go file (not in whitelist)",
			path:        "main.go",
			wantSuccess: false,
			wantError:   "file extension .go not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			args := map[string]interface{}{
				"path":    tt.path,
				"content": "test content",
				"backup":  false,
			}

			result, err := tool.Execute(ctx, args)

			if tt.wantSuccess {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
					return
				}
				if !result.Success {
					t.Errorf("Expected success=true, got: %v (error: %s)", result.Success, result.Error)
				}
			} else {
				if err == nil {
					t.Error("Execute() expected error, got nil")
					return
				}
				if tt.wantError != "" && !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("Expected error containing %q, got: %v", tt.wantError, err)
				}
			}
		})
	}
}

func TestFileWriterTool_WithTempDir(t *testing.T) {

	tempDir, err := os.MkdirTemp("", "filewriter_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tool := NewFileWriterTool(&config.FileWriterConfig{
		MaxFileSize:       1024,
		AllowedExtensions: []string{".txt", ".md"},
		BackupOnOverwrite: config.BoolPtr(false),
		WorkingDirectory:  tempDir,
	})

	tests := []struct {
		name        string
		path        string
		content     string
		backup      bool
		wantSuccess bool
		validate    func(t *testing.T, result ToolResult, tempDir string)
	}{
		{
			name:        "create new file",
			path:        "test.txt",
			content:     "Hello, World!",
			backup:      false,
			wantSuccess: true,
			validate: func(t *testing.T, result ToolResult, tempDir string) {
				if !result.Success {
					t.Error("Expected success=true")
				}
				if !strings.Contains(result.Content, "created") {
					t.Error("Expected result to mention 'created'")
				}

				filePath := filepath.Join(tempDir, "test.txt")
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Error("Expected file to be created")
				}
			},
		},
		{
			name:        "overwrite existing file",
			path:        "test.txt",
			content:     "Updated content",
			backup:      false,
			wantSuccess: true,
			validate: func(t *testing.T, result ToolResult, tempDir string) {
				if !result.Success {
					t.Error("Expected success=true")
				}
				if !strings.Contains(result.Content, "overwritten") && !strings.Contains(result.Content, "created") {
					t.Error("Expected result to mention 'overwritten' or 'created'")
				}

				filePath := filepath.Join(tempDir, "test.txt")
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}
				if string(content) != "Updated content" {
					t.Errorf("Expected file content 'Updated content', got: %s", string(content))
				}
			},
		},
		{
			name:        "create file in subdirectory",
			path:        "subdir/nested.txt",
			content:     "Nested content",
			backup:      false,
			wantSuccess: true,
			validate: func(t *testing.T, result ToolResult, tempDir string) {
				if !result.Success {
					t.Error("Expected success=true")
				}

				filePath := filepath.Join(tempDir, "subdir", "nested.txt")
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Error("Expected nested file to be created")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			args := map[string]interface{}{
				"path":    tt.path,
				"content": tt.content,
				"backup":  tt.backup,
			}

			result, err := tool.Execute(ctx, args)

			if tt.wantSuccess {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
					return
				}
				tt.validate(t, result, tempDir)
			} else {
				if err == nil {
					t.Error("Execute() expected error, got nil")
				}
			}
		})
	}
}
