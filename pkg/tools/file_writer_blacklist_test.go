package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
)

func TestFileWriterTool_BlacklistDenied(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "filewriter_blacklist_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create tool with BLACKLIST (deny specific extensions)
	tool := NewFileWriterTool(&config.FileWriterConfig{
		MaxFileSize:       1024,
		DeniedExtensions:  []string{".exe", ".bat", ".sh"}, // Block executables
		BackupOnOverwrite: false,
		WorkingDirectory:  tempDir,
	})

	tests := []struct {
		name        string
		path        string
		wantSuccess bool
		wantError   string
	}{
		{
			name:        "allowed .py file (not in blacklist)",
			path:        "test.py",
			wantSuccess: true,
		},
		{
			name:        "allowed .go file (not in blacklist)",
			path:        "main.go",
			wantSuccess: true,
		},
		{
			name:        "allowed Makefile (not in blacklist)",
			path:        "Makefile",
			wantSuccess: true,
		},
		{
			name:        "denied .exe file (in blacklist)",
			path:        "virus.exe",
			wantSuccess: false,
			wantError:   "is explicitly denied",
		},
		{
			name:        "denied .bat file (in blacklist)",
			path:        "script.bat",
			wantSuccess: false,
			wantError:   "is explicitly denied",
		},
		{
			name:        "denied .sh file (in blacklist)",
			path:        "install.sh",
			wantSuccess: false,
			wantError:   "is explicitly denied",
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

				// Verify file was actually created
				filePath := filepath.Join(tempDir, tt.path)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("Expected file %s to be created", tt.path)
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

func TestFileWriterTool_WhitelistAndBlacklist(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "filewriter_both_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create tool with BOTH whitelist and blacklist
	// Blacklist takes precedence
	tool := NewFileWriterTool(&config.FileWriterConfig{
		MaxFileSize:       1024,
		AllowedExtensions: []string{".py", ".go", ".sh"}, // Whitelist
		DeniedExtensions:  []string{".sh"},               // But deny .sh
		BackupOnOverwrite: false,
		WorkingDirectory:  tempDir,
	})

	tests := []struct {
		name        string
		path        string
		wantSuccess bool
		wantError   string
	}{
		{
			name:        "allowed .py (in whitelist, not in blacklist)",
			path:        "test.py",
			wantSuccess: true,
		},
		{
			name:        "allowed .go (in whitelist, not in blacklist)",
			path:        "main.go",
			wantSuccess: true,
		},
		{
			name:        "denied .sh (in whitelist BUT in blacklist - blacklist wins)",
			path:        "install.sh",
			wantSuccess: false,
			wantError:   "is explicitly denied",
		},
		{
			name:        "denied .txt (not in whitelist)",
			path:        "readme.txt",
			wantSuccess: false,
			wantError:   "not allowed",
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

			_, err := tool.Execute(ctx, args)

			if tt.wantSuccess {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
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

