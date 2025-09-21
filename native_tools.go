package hector

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SecurityConfig defines security boundaries for command execution
type SecurityConfig struct {
	AllowedCommands  []string `yaml:"allowed_commands" json:"allowed_commands"`
	WorkingDirectory string   `yaml:"working_directory" json:"working_directory"`
	MaxExecutionTime int      `yaml:"max_execution_time_seconds" json:"max_execution_time_seconds"`
	EnableSandboxing bool     `yaml:"enable_sandboxing" json:"enable_sandboxing"`
}

// CommandExecutor handles secure command execution
type CommandExecutor struct {
	config *SecurityConfig
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(config *SecurityConfig) *CommandExecutor {
	if config == nil {
		config = &SecurityConfig{
			AllowedCommands: []string{
				"cat", "head", "tail", "ls", "find", "grep", "wc", "pwd",
				"git", "npm", "go", "curl", "wget",
			},
			WorkingDirectory: "./",
			MaxExecutionTime: 30,
			EnableSandboxing: true,
		}
	}
	return &CommandExecutor{config: config}
}

// Execute runs a command with security checks
func (e *CommandExecutor) Execute(ctx context.Context, command string, workingDir string) (ToolResult, error) {
	if command == "" {
		return ToolResult{Success: false, Error: "empty command"}, fmt.Errorf("empty command")
	}

	// Set working directory
	if workingDir == "" {
		workingDir = e.config.WorkingDirectory
	}

	// Security check - extract base command for validation
	baseCmd := e.extractBaseCommand(command)
	if e.config.EnableSandboxing && !e.isCommandAllowed(baseCmd) {
		return ToolResult{Success: false, Error: "command not allowed"},
			fmt.Errorf("command not allowed: %s", baseCmd)
	}

	// Add timeout if configured
	if e.config.MaxExecutionTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(e.config.MaxExecutionTime)*time.Second)
		defer cancel()
	}

	// Always use shell for consistent behavior
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = workingDir

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	executionTime := time.Since(startTime).Milliseconds()

	result := ToolResult{
		Content:       string(output),
		Success:       err == nil,
		ToolName:      "execute_command",
		ExecutionTime: executionTime,
		Metadata: map[string]interface{}{
			"command":     command,
			"working_dir": workingDir,
		},
	}

	if err != nil {
		result.Error = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			result.Metadata["exit_code"] = exitError.ExitCode()
		}
	}

	return result, err
}

// extractBaseCommand gets the first command from a complex shell command
func (e *CommandExecutor) extractBaseCommand(command string) string {
	// Handle pipes, redirects, etc. - get the first command
	parts := strings.FieldsFunc(command, func(r rune) bool {
		return r == '|' || r == '>' || r == '<' || r == ';'
	})

	if len(parts) == 0 {
		return ""
	}

	// Get first word of first command
	firstCmd := strings.TrimSpace(parts[0])
	cmdParts := strings.Fields(firstCmd)
	if len(cmdParts) == 0 {
		return ""
	}

	return cmdParts[0]
}

// isCommandAllowed checks if a command is allowed
func (e *CommandExecutor) isCommandAllowed(command string) bool {
	if !e.config.EnableSandboxing {
		return true
	}

	for _, allowed := range e.config.AllowedCommands {
		if command == allowed {
			return true
		}
	}
	return false
}

// GetToolInfo returns tool information for the LLM
func (e *CommandExecutor) GetToolInfo() ToolInfo {
	return ToolInfo{
		Name:        "execute_command",
		Description: "Execute shell commands for file operations, system tasks, and development workflows",
		Parameters: []ToolParameter{
			{
				Name:        "command",
				Type:        "string",
				Description: "Shell command to execute (supports pipes, redirects, etc.)",
				Required:    true,
			},
			{
				Name:        "working_dir",
				Type:        "string",
				Description: "Working directory (optional)",
				Required:    false,
			},
		},
		ServerURL: "command",
	}
}
