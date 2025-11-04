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

func TestNewSearchReplaceToolForTesting(t *testing.T) {
	tool := NewSearchReplaceToolForTesting()
	if tool == nil {
		t.Fatal("NewSearchReplaceToolForTesting() returned nil")
	}

	if tool.GetName() != "search_replace" {
		t.Errorf("GetName() = %v, want 'search_replace'", tool.GetName())
	}

	description := tool.GetDescription()
	if description == "" {
		t.Error("GetDescription() should not return empty string")
	}
}

func TestSearchReplaceTool_GetInfo(t *testing.T) {
	tool := NewSearchReplaceToolForTesting()
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
	hasOldStringParam := false
	hasNewStringParam := false
	for _, param := range info.Parameters {
		if param.Name == "path" && param.Required {
			hasPathParam = true
		}
		if param.Name == "old_string" && param.Required {
			hasOldStringParam = true
		}
		if param.Name == "new_string" && param.Required {
			hasNewStringParam = true
		}
	}
	if !hasPathParam {
		t.Error("Expected 'path' parameter to be required")
	}
	if !hasOldStringParam {
		t.Error("Expected 'old_string' parameter to be required")
	}
	if !hasNewStringParam {
		t.Error("Expected 'new_string' parameter to be required")
	}
}

func TestSearchReplaceTool_ValidatePath(t *testing.T) {
	tool := NewSearchReplaceToolForTesting()

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
			wantErr: true,
			errMsg:  "file does not exist",
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
			name:    "non-existent file",
			path:    "nonexistent.txt",
			wantErr: true,
			errMsg:  "file does not exist",
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

func TestSearchReplaceTool_Execute_ValidationOnly(t *testing.T) {
	tool := NewSearchReplaceToolForTesting()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing path parameter",
			args: map[string]interface{}{
				"old_string": "old",
				"new_string": "new",
			},
			wantErr: true,
			errMsg:  "path parameter is required",
		},
		{
			name: "missing old_string parameter",
			args: map[string]interface{}{
				"path":       "test.txt",
				"new_string": "new",
			},
			wantErr: true,
			errMsg:  "old_string parameter is required",
		},
		{
			name: "missing new_string parameter",
			args: map[string]interface{}{
				"path":       "test.txt",
				"old_string": "old",
			},
			wantErr: true,
			errMsg:  "new_string parameter is required",
		},
		{
			name: "empty path",
			args: map[string]interface{}{
				"path":       "",
				"old_string": "old",
				"new_string": "new",
			},
			wantErr: true,
			errMsg:  "path parameter is required",
		},
		{
			name: "empty old_string",
			args: map[string]interface{}{
				"path":       "test.txt",
				"old_string": "",
				"new_string": "new",
			},
			wantErr: true,
			errMsg:  "old_string parameter is required",
		},
		{
			name: "invalid path type",
			args: map[string]interface{}{
				"path":       123,
				"old_string": "old",
				"new_string": "new",
			},
			wantErr: true,
			errMsg:  "path parameter is required",
		},
		{
			name: "invalid old_string type",
			args: map[string]interface{}{
				"path":       "test.txt",
				"old_string": 123,
				"new_string": "new",
			},
			wantErr: true,
			errMsg:  "old_string parameter is required",
		},
		{
			name: "invalid new_string type",
			args: map[string]interface{}{
				"path":       "test.txt",
				"old_string": "old",
				"new_string": 123,
			},
			wantErr: true,
			errMsg:  "new_string parameter is required",
		},
		{
			name: "replace_all parameter",
			args: map[string]interface{}{
				"path":        "test.txt",
				"old_string":  "old",
				"new_string":  "new",
				"replace_all": true,
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

func TestSearchReplaceTool_GenerateDiff(t *testing.T) {
	tool := NewSearchReplaceToolForTesting()

	tests := []struct {
		name     string
		oldStr   string
		newStr   string
		expected string
	}{
		{
			name:   "simple replacement",
			oldStr: "old text",
			newStr: "new text",
			expected: "📝 Changes:\n" +
				"------------------------------------------------------------\n" +
				"- old text\n" +
				"+ new text\n" +
				"------------------------------------------------------------",
		},
		{
			name:   "multiline replacement",
			oldStr: "line 1\nline 2",
			newStr: "updated line 1\nupdated line 2",
			expected: "📝 Changes:\n" +
				"------------------------------------------------------------\n" +
				"- line 1\n" +
				"- line 2\n" +
				"+ updated line 1\n" +
				"+ updated line 2\n" +
				"------------------------------------------------------------",
		},
		{
			name:   "empty lines ignored",
			oldStr: "text\n\nmore text",
			newStr: "new text\n\nnew more text",
			expected: "📝 Changes:\n" +
				"------------------------------------------------------------\n" +
				"- text\n" +
				"- more text\n" +
				"+ new text\n" +
				"+ new more text\n" +
				"------------------------------------------------------------",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.generateDiff(tt.oldStr, tt.newStr)
			if result != tt.expected {
				t.Errorf("generateDiff() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSearchReplaceTool_TruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "string shorter than max",
			input:  "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "string exactly max length",
			input:  "exactlyten",
			maxLen: 10,
			want:   "exactlyten",
		},
		{
			name:   "string longer than max",
			input:  "this is a very long string",
			maxLen: 10,
			want:   "this is...",
		},
		{
			name:   "string much longer than max",
			input:  "this is a very very very long string",
			maxLen: 5,
			want:   "th...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.want)
			}
		})
	}
}

func TestSearchReplaceTool_ErrorResult(t *testing.T) {
	tool := NewSearchReplaceToolForTesting()

	result := tool.errorResult("test error message", time.Now())

	if result.Success {
		t.Error("Expected error result to have Success=false")
	}
	if result.Error != "test error message" {
		t.Errorf("Expected error message 'test error message', got: %s", result.Error)
	}
	if result.ToolName != "search_replace" {
		t.Errorf("Expected tool name 'search_replace', got: %s", result.ToolName)
	}
}

func TestSearchReplaceTool_WithTempFile(t *testing.T) {

	tempDir, err := os.MkdirTemp("", "searchreplace_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!\nThis is a test file.\nHello again!"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewSearchReplaceTool(&config.SearchReplaceConfig{
		MaxReplacements:  10,
		ShowDiff:         config.BoolPtr(true),
		CreateBackup:     config.BoolPtr(false),
		WorkingDirectory: tempDir,
	})

	tests := []struct {
		name        string
		path        string
		oldString   string
		newString   string
		replaceAll  bool
		wantSuccess bool
		validate    func(t *testing.T, result ToolResult, tempDir string)
	}{
		{
			name:        "single replacement",
			path:        "test.txt",
			oldString:   "Hello, World!",
			newString:   "Hi, World!",
			replaceAll:  false,
			wantSuccess: true,
			validate: func(t *testing.T, result ToolResult, tempDir string) {
				if !result.Success {
					t.Error("Expected success=true")
				}
				if !strings.Contains(result.Content, "Replaced 1 occurrence") {
					t.Error("Expected result to mention 'Replaced 1 occurrence'")
				}

				filePath := filepath.Join(tempDir, "test.txt")
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}
				if !strings.Contains(string(content), "Hi, World!") {
					t.Error("Expected file to contain 'Hi, World!'")
				}
			},
		},
		{
			name:        "replace all occurrences",
			path:        "test.txt",
			oldString:   "Hello",
			newString:   "Hi",
			replaceAll:  true,
			wantSuccess: true,
			validate: func(t *testing.T, result ToolResult, tempDir string) {
				if !result.Success {
					t.Error("Expected success=true")
				}
				if !strings.Contains(result.Content, "Replaced 2 occurrence") {
					t.Error("Expected result to mention 'Replaced 2 occurrence'")
				}

				filePath := filepath.Join(tempDir, "test.txt")
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}
				contentStr := string(content)
				if strings.Count(contentStr, "Hi") != 2 {
					t.Errorf("Expected 2 occurrences of 'Hi', found: %d", strings.Count(contentStr, "Hi"))
				}
			},
		},
		{
			name:        "string not found",
			path:        "test.txt",
			oldString:   "Not found",
			newString:   "Replacement",
			replaceAll:  false,
			wantSuccess: false,
		},
		{
			name:        "ambiguous replacement without replace_all",
			path:        "test.txt",
			oldString:   "Hello",
			newString:   "Hi",
			replaceAll:  false,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err = os.WriteFile(testFile, []byte(testContent), 0644)
			if err != nil {
				t.Fatalf("Failed to reset test file: %v", err)
			}

			ctx := context.Background()
			args := map[string]interface{}{
				"path":        tt.path,
				"old_string":  tt.oldString,
				"new_string":  tt.newString,
				"replace_all": tt.replaceAll,
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
