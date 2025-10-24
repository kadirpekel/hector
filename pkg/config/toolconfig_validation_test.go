package config

import (
	"testing"
)

func TestToolConfig_CommandValidation(t *testing.T) {
	tests := []struct {
		name          string
		tool          ToolConfig
		expectError   bool
		errorContains string
	}{
		{
			name: "command tool with sandboxing enabled + no allowed_commands = valid (permissive default)",
			tool: ToolConfig{
				Type:             "command",
				EnableSandboxing: true,
				AllowedCommands:  []string{},
			},
			expectError: false,
		},
		{
			name: "command tool with sandboxing enabled + nil allowed_commands = valid (permissive default)",
			tool: ToolConfig{
				Type:             "command",
				EnableSandboxing: true,
				AllowedCommands:  nil,
			},
			expectError: false,
		},
		{
			name: "command tool with sandboxing enabled + explicit allowed_commands = valid",
			tool: ToolConfig{
				Type:             "command",
				EnableSandboxing: true,
				AllowedCommands:  []string{"ls", "cat"},
			},
			expectError: false,
		},
		{
			name: "command tool with sandboxing disabled + no allowed_commands = INVALID (security)",
			tool: ToolConfig{
				Type:             "command",
				EnableSandboxing: false,
				AllowedCommands:  []string{},
			},
			expectError:   true,
			errorContains: "allowed_commands is required when enable_sandboxing is false",
		},
		{
			name: "command tool with sandboxing disabled + nil allowed_commands = INVALID (security)",
			tool: ToolConfig{
				Type:             "command",
				EnableSandboxing: false,
				AllowedCommands:  nil,
			},
			expectError:   true,
			errorContains: "allowed_commands is required when enable_sandboxing is false",
		},
		{
			name: "command tool with sandboxing disabled + explicit allowed_commands = valid",
			tool: ToolConfig{
				Type:             "command",
				EnableSandboxing: false,
				AllowedCommands:  []string{"ls"},
			},
			expectError: false,
		},
		{
			name: "command tool with default (zero-value) sandboxing + no allowed_commands = valid (default is true)",
			tool: ToolConfig{
				Type:            "command",
				AllowedCommands: []string{},
			},
			expectError: false, // Now valid because SetDefaults() sets EnableSandboxing to true
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldCallSetDefaults := tt.name != "command tool with sandboxing disabled + no allowed_commands = INVALID (security)" &&
				tt.name != "command tool with sandboxing disabled + nil allowed_commands = INVALID (security)"

			if shouldCallSetDefaults {
				tt.tool.SetDefaults()
			}

			err := tt.tool.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %s", err.Error())
				}
			}
		})
	}
}
