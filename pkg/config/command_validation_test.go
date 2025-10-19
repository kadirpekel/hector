package config

import (
	"testing"
)

// TestCommandToolsConfig_ValidationWithSandboxing tests the new permissive-by-default logic
func TestCommandToolsConfig_ValidationWithSandboxing(t *testing.T) {
	tests := []struct {
		name          string
		config        CommandToolsConfig
		expectError   bool
		errorContains string
	}{
		{
			name: "sandboxing enabled + empty allowed_commands = valid (allow all)",
			config: CommandToolsConfig{
				EnableSandboxing: true,
				AllowedCommands:  []string{},
			},
			expectError: false,
		},
		{
			name: "sandboxing enabled + nil allowed_commands = valid (allow all)",
			config: CommandToolsConfig{
				EnableSandboxing: true,
				AllowedCommands:  nil,
			},
			expectError: false,
		},
		{
			name: "sandboxing enabled + explicit allowed_commands = valid",
			config: CommandToolsConfig{
				EnableSandboxing: true,
				AllowedCommands:  []string{"ls", "cat"},
			},
			expectError: false,
		},
		{
			name: "sandboxing disabled + empty allowed_commands = INVALID",
			config: CommandToolsConfig{
				EnableSandboxing: false,
				AllowedCommands:  []string{},
			},
			expectError:   true,
			errorContains: "allowed_commands is required when enable_sandboxing is false",
		},
		{
			name: "sandboxing disabled + nil allowed_commands = INVALID",
			config: CommandToolsConfig{
				EnableSandboxing: false,
				AllowedCommands:  nil,
			},
			expectError:   true,
			errorContains: "allowed_commands is required when enable_sandboxing is false",
		},
		{
			name: "sandboxing disabled + explicit allowed_commands = valid",
			config: CommandToolsConfig{
				EnableSandboxing: false,
				AllowedCommands:  []string{"ls", "cat", "grep"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorContains)
					return
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestCommandToolsConfig_SetDefaults tests default value assignment
func TestCommandToolsConfig_SetDefaults_Permissive(t *testing.T) {
	tests := []struct {
		name      string
		input     CommandToolsConfig
		checkFunc func(*testing.T, CommandToolsConfig)
	}{
		{
			name:  "empty config gets defaults (no allowed_commands)",
			input: CommandToolsConfig{},
			checkFunc: func(t *testing.T, c CommandToolsConfig) {
				if c.WorkingDirectory != "./" {
					t.Errorf("expected WorkingDirectory './', got %q", c.WorkingDirectory)
				}
				if c.MaxExecutionTime != 30_000_000_000 { // 30 seconds in nanoseconds
					t.Errorf("expected MaxExecutionTime 30s, got %v", c.MaxExecutionTime)
				}
				// AllowedCommands should remain nil/empty (permissive by default)
			},
		},
		{
			name: "existing allowed_commands preserved",
			input: CommandToolsConfig{
				AllowedCommands: []string{"custom", "commands"},
			},
			checkFunc: func(t *testing.T, c CommandToolsConfig) {
				if len(c.AllowedCommands) != 2 {
					t.Errorf("expected 2 allowed commands, got %d", len(c.AllowedCommands))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.input
			config.SetDefaults()
			tt.checkFunc(t, config)
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && s[0:len(substr)] == substr) ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
