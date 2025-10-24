package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
)

type CommandTool struct {
	config *config.CommandToolsConfig
}

func NewCommandTool(commandConfig *config.CommandToolsConfig) *CommandTool {
	if commandConfig == nil {
		commandConfig = &config.CommandToolsConfig{
			AllowedCommands:  nil,
			WorkingDirectory: "./",
			MaxExecutionTime: 30 * time.Second,
			EnableSandboxing: true,
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

func NewCommandToolWithConfig(name string, toolConfig *config.ToolConfig) (*CommandTool, error) {
	if toolConfig == nil {
		return nil, fmt.Errorf("tool config is required")
	}

	commandConfig := &config.CommandToolsConfig{
		AllowedCommands:  toolConfig.AllowedCommands,
		WorkingDirectory: toolConfig.WorkingDirectory,
		EnableSandboxing: toolConfig.EnableSandboxing,
	}

	if toolConfig.MaxExecutionTime != "" {
		duration, err := time.ParseDuration(toolConfig.MaxExecutionTime)
		if err != nil {
			return nil, fmt.Errorf("invalid max_execution_time: %w", err)
		}
		commandConfig.MaxExecutionTime = duration
	}

	commandConfig.SetDefaults()

	return NewCommandTool(commandConfig), nil
}

func (t *CommandTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {

	command, ok := args["command"].(string)
	if !ok || command == "" {
		return t.createErrorResult("command parameter is required", fmt.Errorf("command parameter is required"))
	}

	workingDir, _ := args["working_dir"].(string)

	if workingDir == "" {
		workingDir = t.config.WorkingDirectory
	}

	if err := t.validateCommand(command); err != nil {
		return t.createErrorResult(err.Error(), err)
	}

	if t.config.MaxExecutionTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.config.MaxExecutionTime)
		defer cancel()
	}

	return t.executeCommand(ctx, command, workingDir)
}

func (t *CommandTool) validateCommand(command string) error {

	if t.config.EnableSandboxing && len(t.config.AllowedCommands) == 0 {
		return nil
	}

	baseCmd := t.extractBaseCommand(command)
	if !t.isCommandAllowed(baseCmd) {
		return fmt.Errorf("command not allowed: %s (allowed: %v)", baseCmd, t.config.AllowedCommands)
	}

	return nil
}

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

func (t *CommandTool) createErrorResult(message string, err error) (ToolResult, error) {
	return ToolResult{
		Success:  false,
		Error:    message,
		ToolName: "execute_command",
	}, err
}

func (t *CommandTool) extractBaseCommand(command string) string {

	parts := strings.FieldsFunc(command, func(r rune) bool {
		return r == '|' || r == '>' || r == '<' || r == ';'
	})

	if len(parts) == 0 {
		return ""
	}

	firstCmd := strings.TrimSpace(parts[0])
	cmdParts := strings.Fields(firstCmd)
	if len(cmdParts) == 0 {
		return ""
	}

	return cmdParts[0]
}

func (t *CommandTool) isCommandAllowed(command string) bool {

	if len(t.config.AllowedCommands) == 0 {
		return true
	}

	for _, allowed := range t.config.AllowedCommands {
		if command == allowed {
			return true
		}
	}
	return false
}

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

func (t *CommandTool) GetName() string {
	return "execute_command"
}

func (t *CommandTool) GetDescription() string {
	return "Execute shell commands for file operations, system tasks, and development workflows. Use 'sed -n \"START,ENDp\" FILE' to read specific line ranges."
}
