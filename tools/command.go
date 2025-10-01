package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kadirpekel/hector/config"
)

// ============================================================================
// COMMAND EXECUTOR - SECURE SHELL COMMAND EXECUTION
// ============================================================================

// CommandTool handles secure command execution
type CommandTool struct {
	config *config.CommandToolsConfig
}

// NewCommandTool creates a new command tool with secure defaults
func NewCommandTool(commandConfig *config.CommandToolsConfig) *CommandTool {
	if commandConfig == nil {
		commandConfig = &config.CommandToolsConfig{
			AllowedCommands: []string{
				"cat", "head", "tail", "ls", "find", "grep", "wc", "pwd",
				"git", "npm", "go", "curl", "wget", "echo", "date",
			},
			WorkingDirectory: "./",
			MaxExecutionTime: 30 * time.Second,
			EnableSandboxing: true,
		}
	}

	// Apply defaults if not set
	if len(commandConfig.AllowedCommands) == 0 {
		commandConfig.AllowedCommands = []string{
			"cat", "head", "tail", "ls", "find", "grep", "wc", "pwd",
			"git", "npm", "go", "curl", "wget", "echo", "date",
		}
	}
	if commandConfig.WorkingDirectory == "" {
		commandConfig.WorkingDirectory = "./"
	}
	if commandConfig.MaxExecutionTime == 0 {
		commandConfig.MaxExecutionTime = 30 * time.Second
	}

	return &CommandTool{config: commandConfig}
}

// NewCommandToolWithConfig creates a command tool from a ToolDefinition configuration
func NewCommandToolWithConfig(toolDef config.ToolDefinition) (*CommandTool, error) {
	// Convert the generic config map to CommandToolsConfig
	var commandConfig *config.CommandToolsConfig
	if toolDef.Config != nil {
		commandConfig = &config.CommandToolsConfig{}
		// Map the common fields from map[string]interface{}
		if allowedCommands, ok := toolDef.Config["allowed_commands"].([]interface{}); ok {
			commands := make([]string, len(allowedCommands))
			for i, cmd := range allowedCommands {
				if cmdStr, ok := cmd.(string); ok {
					commands[i] = cmdStr
				}
			}
			commandConfig.AllowedCommands = commands
		}
		if workDir, ok := toolDef.Config["working_directory"].(string); ok {
			commandConfig.WorkingDirectory = workDir
		}
		if enableSandbox, ok := toolDef.Config["enable_sandboxing"].(bool); ok {
			commandConfig.EnableSandboxing = enableSandbox
		}
	}

	// Apply defaults if config is nil
	if commandConfig == nil {
		commandConfig = &config.CommandToolsConfig{}
	}
	commandConfig.SetDefaults()

	return NewCommandTool(commandConfig), nil
}

// Execute runs a command with security checks and timeout protection
func (t *CommandTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	// Extract parameters - support both "command" and "input" for flexibility
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return t.createErrorResult("command parameter is required", fmt.Errorf("command parameter is required"))
	}

	workingDir, _ := args["working_dir"].(string)

	// Set working directory
	if workingDir == "" {
		workingDir = t.config.WorkingDirectory
	}

	// Security validation
	if err := t.validateCommand(command); err != nil {
		return t.createErrorResult(err.Error(), err)
	}

	// Apply timeout
	if t.config.MaxExecutionTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.config.MaxExecutionTime)
		defer cancel()
	}

	// Execute command through shell for consistent behavior
	return t.executeCommand(ctx, command, workingDir)
}

// validateCommand performs security validation on the command
func (t *CommandTool) validateCommand(command string) error {
	if !t.config.EnableSandboxing {
		return nil
	}

	baseCmd := t.extractBaseCommand(command)
	if !t.isCommandAllowed(baseCmd) {
		return fmt.Errorf("command not allowed: %s", baseCmd)
	}

	return nil
}

// executeCommand executes the validated command
func (t *CommandTool) executeCommand(ctx context.Context, command, workingDir string) (ToolResult, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = workingDir

	start := time.Now()
	output, err := cmd.CombinedOutput()
	executionTime := time.Since(start)

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

// createErrorResult creates a standardized error result
func (t *CommandTool) createErrorResult(message string, err error) (ToolResult, error) {
	return ToolResult{
		Success:  false,
		Error:    message,
		ToolName: "execute_command",
	}, err
}

// extractBaseCommand gets the first command from a complex shell command
func (t *CommandTool) extractBaseCommand(command string) string {
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

// isCommandAllowed checks if a command is in the allowed list
func (t *CommandTool) isCommandAllowed(command string) bool {
	for _, allowed := range t.config.AllowedCommands {
		if command == allowed {
			return true
		}
	}
	return false
}

// GetInfo returns tool information for the Tool interface
func (t *CommandTool) GetInfo() ToolInfo {
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
		ServerURL: "local",
	}
}

// GetName returns the tool name
func (t *CommandTool) GetName() string {
	return "execute_command"
}

// GetDescription returns the tool description
func (t *CommandTool) GetDescription() string {
	return "Execute shell commands for file operations, system tasks, and development workflows"
}
