package hector

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ============================================================================
// COMMAND-LINE TOOL INFRASTRUCTURE
// ============================================================================

// CommandTool represents a command-line tool that can be executed directly
type CommandTool interface {
	Name() string
	Description() string
	Parameters() []ToolParameter
	Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error)
}

// CommandToolRegistry manages command-line tools
type CommandToolRegistry struct {
	tools       map[string]CommandTool
	permissions *SecurityConfig
}

// SecurityConfig defines security boundaries for command-line tools
type SecurityConfig struct {
	AllowedCommands  []string `yaml:"allowed_commands" json:"allowed_commands"`
	WorkingDirectory string   `yaml:"working_directory" json:"working_directory"`
	MaxExecutionTime int      `yaml:"max_execution_time_seconds" json:"max_execution_time_seconds"`
	EnableSandboxing bool     `yaml:"enable_sandboxing" json:"enable_sandboxing"`
}

// NewCommandToolRegistry creates a new command-line tool registry
func NewCommandToolRegistry(permissions *SecurityConfig) *CommandToolRegistry {
	if permissions == nil {
		permissions = &SecurityConfig{
			AllowedCommands: []string{
				// File operations
				"cat", "head", "tail", "less", "more",
				"ls", "dir", "find", "locate", "which", "whereis",
				"cp", "mv", "rm", "mkdir", "rmdir", "touch",
				"chmod", "chown", "stat", "file", "du", "df",
				// Text processing
				"grep", "awk", "sed", "sort", "uniq", "cut", "paste",
				"wc", "tr", "diff", "patch",
				// System info
				"pwd", "whoami", "id", "uname", "uptime", "ps", "top",
				"free", "df", "mount", "env", "printenv",
				// Development tools
				"git", "npm", "node", "python", "go", "gcc", "make",
				"curl", "wget", "ssh", "scp", "rsync",
			},
			WorkingDirectory: "./",
			MaxExecutionTime: 30,
			EnableSandboxing: true,
		}
	}

	registry := &CommandToolRegistry{
		tools:       make(map[string]CommandTool),
		permissions: permissions,
	}

	// Register default tools
	registry.registerDefaultTools()

	return registry
}

// registerDefaultTools registers the default set of command-line tools
func (r *CommandToolRegistry) registerDefaultTools() {
	// Single unified command execution tool
	r.RegisterTool(&ExecuteCommandTool{permissions: r.permissions})
}

// RegisterTool registers a command-line tool
func (r *CommandToolRegistry) RegisterTool(tool CommandTool) {
	r.tools[tool.Name()] = tool
}

// GetTool retrieves a command-line tool by name
func (r *CommandToolRegistry) GetTool(name string) (CommandTool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// ListTools returns all registered command-line tools
func (r *CommandToolRegistry) ListTools() []ToolInfo {
	var tools []ToolInfo
	for _, tool := range r.tools {
		tools = append(tools, ToolInfo{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
			ServerURL:   "command", // Mark as command-line tool
		})
	}
	return tools
}

// ExecuteTool executes a command-line tool
func (r *CommandToolRegistry) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (ToolResult, error) {
	tool, exists := r.GetTool(toolName)
	if !exists {
		return ToolResult{
			Content:  "",
			Success:  false,
			Error:    fmt.Sprintf("command tool %s not found", toolName),
			ToolName: toolName,
		}, fmt.Errorf("command tool %s not found", toolName)
	}

	// Add timeout context if configured
	if r.permissions.MaxExecutionTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(r.permissions.MaxExecutionTime)*time.Second)
		defer cancel()
	}

	startTime := time.Now()
	result, err := tool.Execute(ctx, args)
	executionTime := time.Since(startTime).Milliseconds()

	if err != nil {
		return ToolResult{
			Content:       "",
			Success:       false,
			Error:         err.Error(),
			ToolName:      toolName,
			ExecutionTime: executionTime,
		}, err
	}

	result.ExecutionTime = executionTime
	return result, nil
}

// ============================================================================
// COMMAND LINE TOOLS
// ============================================================================

// ExecuteCommandTool executes shell commands - unified tool for all command-line operations
type ExecuteCommandTool struct {
	permissions *SecurityConfig
}

func (e *ExecuteCommandTool) Name() string {
	return "execute_command"
}

func (e *ExecuteCommandTool) Description() string {
	return "Execute shell commands for file operations, system tasks, and development workflows. Supports all standard Unix commands like ls, cat, grep, find, git, npm, etc."
}

func (e *ExecuteCommandTool) Parameters() []ToolParameter {
	return []ToolParameter{
		{
			Name:        "command",
			Type:        "string",
			Description: "Full command to execute (e.g., 'ls -la', 'cat file.txt', 'git status')",
			Required:    true,
		},
		{
			Name:        "working_dir",
			Type:        "string",
			Description: "Working directory for the command (optional, defaults to configured directory)",
			Required:    false,
		},
	}
}

func (e *ExecuteCommandTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	command, ok := args["command"].(string)
	if !ok {
		return ToolResult{Success: false, Error: "command parameter is required"}, fmt.Errorf("command parameter is required")
	}

	// Parse command into parts
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ToolResult{Success: false, Error: "empty command"}, fmt.Errorf("empty command")
	}

	baseCommand := parts[0]
	cmdArgs := parts[1:]

	// Check if command is allowed
	if !e.permissions.isCommandAllowed(baseCommand) {
		return ToolResult{Success: false, Error: "command not allowed"}, fmt.Errorf("command not allowed: %s", baseCommand)
	}

	// Set working directory
	workingDir := e.permissions.WorkingDirectory
	if wdArg, ok := args["working_dir"].(string); ok && wdArg != "" {
		workingDir = wdArg
	}

	// Create command
	cmd := exec.CommandContext(ctx, baseCommand, cmdArgs...)
	cmd.Dir = workingDir

	// Execute command
	output, err := cmd.CombinedOutput()

	// Create result
	result := ToolResult{
		Content:  string(output),
		Success:  err == nil,
		ToolName: e.Name(),
		Metadata: map[string]interface{}{
			"command":      command,
			"base_command": baseCommand,
			"args":         cmdArgs,
			"working_dir":  workingDir,
		},
	}

	if err != nil {
		result.Error = err.Error()
		// Try to get exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			result.Metadata["exit_code"] = exitError.ExitCode()
		}
		return result, err
	}

	// Success case
	if cmd.ProcessState != nil {
		result.Metadata["exit_code"] = cmd.ProcessState.ExitCode()
	}

	return result, nil
}

// ============================================================================
// SECURITY HELPER METHODS
// ============================================================================

// isCommandAllowed checks if a command is allowed based on security configuration
func (s *SecurityConfig) isCommandAllowed(command string) bool {
	if !s.EnableSandboxing {
		return true
	}

	// Check against allowed commands
	for _, allowedCmd := range s.AllowedCommands {
		if command == allowedCmd {
			return true
		}
	}

	return false
}
