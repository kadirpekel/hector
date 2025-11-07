package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
)

func TestReadFileTool(t *testing.T) {
	// Create temp directory and test file
	tmpDir, err := os.MkdirTemp("", "read_file_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := `line 1
line 2
line 3
line 4
line 5`
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := &config.ReadFileConfig{
		WorkingDirectory: tmpDir,
		MaxFileSize:      1024,
		ShowLineNumbers:  config.BoolPtr(true),
	}
	tool := NewReadFileTool(cfg)

	t.Run("read entire file", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "test.txt",
		})
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if !result.Success {
			t.Errorf("Expected success, got error: %s", result.Error)
		}
		if !strings.Contains(result.Content, "line 1") {
			t.Errorf("Expected content to contain 'line 1', got: %s", result.Content)
		}
	})

	t.Run("read file with line range", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":       "test.txt",
			"start_line": float64(2),
			"end_line":   float64(4),
		})
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if !result.Success {
			t.Errorf("Expected success, got error: %s", result.Error)
		}
		if !strings.Contains(result.Content, "line 2") {
			t.Errorf("Expected content to contain 'line 2'")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "nonexistent.txt",
		})
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
		if result.Success {
			t.Error("Expected failure for nonexistent file")
		}
	})
}

func TestApplyPatchTool(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "apply_patch_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := &config.ApplyPatchConfig{
		WorkingDirectory: tmpDir,
		MaxFileSize:      1024,
		CreateBackup:     config.BoolPtr(true),
		ContextLines:     3,
	}
	tool := NewApplyPatchTool(cfg)

	t.Run("apply simple patch", func(t *testing.T) {
		oldString := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`
		newString := `package main

import "fmt"

func main() {
	fmt.Println("Hello, Hector!")
}`

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":               "test.go",
			"old_string":         oldString,
			"new_string":         newString,
			"context_validation": false, // Disable for simpler test
		})
		if err != nil {
			t.Fatalf("Failed to apply patch: %v", err)
		}
		if !result.Success {
			t.Errorf("Expected success, got error: %s", result.Error)
		}

		// Verify file was modified
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read modified file: %v", err)
		}
		if !strings.Contains(string(content), "Hello, Hector!") {
			t.Errorf("File was not modified correctly")
		}

		// Verify backup was created
		backupFile := testFile + ".bak"
		if _, err := os.Stat(backupFile); os.IsNotExist(err) {
			t.Error("Backup file was not created")
		}
	})
}

func TestGrepSearchTool(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_search_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	files := map[string]string{
		"test1.go": `package main
import "fmt"
func main() {
	fmt.Println("test")
}`,
		"test2.go": `package main
func helper() {
	// another test
}`,
		"test3.txt": `This is a test file
with multiple lines
containing test data`,
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", name, err)
		}
	}

	cfg := &config.GrepSearchConfig{
		WorkingDirectory: tmpDir,
		MaxFileSize:      1024,
		MaxResults:       100,
		ContextLines:     1,
	}
	tool := NewGrepSearchTool(cfg)

	t.Run("search for pattern", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"pattern": "test",
			"path":    ".",
		})
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
		if !result.Success {
			t.Errorf("Expected success, got error: %s", result.Error)
		}
		metadata := result.Metadata
		if totalMatches, ok := metadata["total_matches"].(int); !ok || totalMatches == 0 {
			t.Errorf("Expected to find matches, got: %v", totalMatches)
		}
	})

	t.Run("search with file pattern", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"pattern":      "test",
			"path":         ".",
			"file_pattern": "*.go",
		})
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
		if !result.Success {
			t.Errorf("Expected success, got error: %s", result.Error)
		}
		// Should only search .go files
		if !strings.Contains(result.Content, ".go") {
			t.Error("Expected results from .go files")
		}
	})

	t.Run("case insensitive search", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"pattern":          "TEST",
			"path":             ".",
			"case_insensitive": true,
		})
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
		if !result.Success {
			t.Errorf("Expected success, got error: %s", result.Error)
		}
		metadata := result.Metadata
		if totalMatches, ok := metadata["total_matches"].(int); !ok || totalMatches == 0 {
			t.Errorf("Expected to find matches with case insensitive search")
		}
	})
}
