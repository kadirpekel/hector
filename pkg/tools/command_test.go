package tools

import (
	"context"
	"strings"
	"testing"
)

func TestNewCommandToolForTesting(t *testing.T) {
	tool := NewCommandToolForTesting()
	if tool == nil {
		t.Fatal("NewCommandToolForTesting() returned nil")
	}

	if tool.GetName() != "execute_command" {
		t.Errorf("GetName() = %v, want 'execute_command'", tool.GetName())
	}

	description := tool.GetDescription()
	if description == "" {
		t.Error("GetDescription() should not return empty string")
	}
}

func TestCommandTool_GetInfo(t *testing.T) {
	tool := NewCommandToolForTesting()
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

	hasCommandParam := false
	for _, param := range info.Parameters {
		if param.Name == "command" && param.Required {
			hasCommandParam = true
			break
		}
	}
	if !hasCommandParam {
		t.Error("Expected 'command' parameter to be required")
	}
}

func TestCommandTool_ValidateCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		allowedCmds []string
		wantErr     bool
	}{
		{
			name:        "allowed command",
			command:     "echo hello",
			allowedCmds: []string{"echo", "pwd"},
			wantErr:     false,
		},
		{
			name:        "disallowed command",
			command:     "rm -rf /",
			allowedCmds: []string{"echo", "pwd"},
			wantErr:     true,
		},
		{
			name:        "command with pipes",
			command:     "echo hello | grep hello",
			allowedCmds: []string{"echo", "grep"},
			wantErr:     false,
		},
		{
			name:        "disallowed command in pipe",
			command:     "rm -rf / | echo hello",
			allowedCmds: []string{"echo", "grep"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewCommandToolForTestingWithCommands(tt.allowedCmds)
			err := tool.validateCommand(tt.command)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommandTool_ExtractBaseCommand(t *testing.T) {
	tool := NewCommandToolForTesting()

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "simple command",
			command:  "echo hello",
			expected: "echo",
		},
		{
			name:     "command with pipes",
			command:  "echo hello | grep hello",
			expected: "echo",
		},
		{
			name:     "command with redirects",
			command:  "ls -la > output.txt",
			expected: "ls",
		},
		{
			name:     "command with semicolon",
			command:  "echo hello; echo world",
			expected: "echo",
		},
		{
			name:     "complex command",
			command:  "find . -name '*.go' | grep test | head -10",
			expected: "find",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.extractBaseCommand(tt.command)
			if result != tt.expected {
				t.Errorf("extractBaseCommand(%q) = %q, want %q", tt.command, result, tt.expected)
			}
		})
	}
}

func TestCommandTool_IsCommandAllowed(t *testing.T) {
	tool := NewCommandToolForTestingWithCommands([]string{"echo", "pwd", "ls"})

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "allowed command",
			command:  "echo",
			expected: true,
		},
		{
			name:     "disallowed command",
			command:  "rm",
			expected: false,
		},
		{
			name:     "another allowed command",
			command:  "pwd",
			expected: true,
		},
		{
			name:     "case sensitive",
			command:  "ECHO",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.isCommandAllowed(tt.command)
			if result != tt.expected {
				t.Errorf("isCommandAllowed(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestCommandTool_Execute_ValidationOnly(t *testing.T) {
	tool := NewCommandToolForTesting()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "missing command",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "empty command",
			args: map[string]interface{}{
				"command": "",
			},
			wantErr: true,
		},
		{
			name: "valid command structure",
			args: map[string]interface{}{
				"command": "echo hello",
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
				} else if !strings.Contains(err.Error(), "command") {
					t.Errorf("Expected command-related error, got: %v", err)
				}
			}
		})
	}
}
