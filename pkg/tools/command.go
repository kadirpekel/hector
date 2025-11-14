package tools

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
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
			EnableSandboxing: config.BoolPtr(true),
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

// validateAndPrepareArgs validates and extracts command arguments
// Returns command, workingDir, and error if validation fails
func (t *CommandTool) validateAndPrepareArgs(ctx context.Context, args map[string]interface{}) (command, workingDir string, newCtx context.Context, cancel context.CancelFunc, err error) {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return "", "", ctx, nil, fmt.Errorf("command parameter is required")
	}

	workingDir, _ = args["working_dir"].(string)
	if workingDir == "" {
		workingDir = t.config.WorkingDirectory
	}

	if err := t.validateCommand(command); err != nil {
		return "", "", ctx, nil, err
	}

	newCtx = ctx
	if t.config.MaxExecutionTime > 0 {
		newCtx, cancel = context.WithTimeout(ctx, t.config.MaxExecutionTime)
	}

	return command, workingDir, newCtx, cancel, nil
}

func (t *CommandTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	command, workingDir, execCtx, cancel, err := t.validateAndPrepareArgs(ctx, args)
	if cancel != nil {
		defer cancel()
	}
	if err != nil {
		return t.createErrorResult(err.Error(), err)
	}

	return t.executeCommand(execCtx, command, workingDir)
}

func (t *CommandTool) validateCommand(command string) error {

	if t.config.EnableSandboxing != nil && *t.config.EnableSandboxing && len(t.config.AllowedCommands) == 0 {
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

	result := t.buildCommandResult(string(output), err, nil, executionTime, command, workingDir)
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

// ExecuteStreaming implements StreamingTool interface for execute_command
// It streams command output incrementally as it's produced
func (t *CommandTool) ExecuteStreaming(ctx context.Context, args map[string]interface{}, resultCh chan<- string) (ToolResult, error) {
	command, workingDir, execCtx, cancel, err := t.validateAndPrepareArgs(ctx, args)
	if cancel != nil {
		defer cancel()
	}
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return t.executeCommandStreaming(execCtx, command, workingDir, resultCh)
}

// pipeStreamer handles streaming from a command pipe (stdout or stderr)
type pipeStreamer struct {
	ctx      context.Context
	resultCh chan<- string
	prefix   string
	builder  *strings.Builder
}

// stream reads from the pipe and streams lines to the result channel
func (s *pipeStreamer) stream(pipe *bufio.Scanner) error {
	for pipe.Scan() {
		line := pipe.Text()
		s.builder.WriteString(line)
		s.builder.WriteString("\n")
		// Send line to result channel with optional prefix
		select {
		case s.resultCh <- s.prefix + line + "\n":
		case <-s.ctx.Done():
			return s.ctx.Err()
		}
	}
	return pipe.Err()
}

func (t *CommandTool) executeCommandStreaming(ctx context.Context, command, workingDir string, resultCh chan<- string) (ToolResult, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = workingDir

	start := time.Now()

	// Create pipes for stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("failed to create stdout pipe: %v", err),
			ToolName: "execute_command",
		}, err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("failed to create stderr pipe: %v", err),
			ToolName: "execute_command",
		}, err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("failed to start command: %v", err),
			ToolName: "execute_command",
		}, err
	}

	var outputBuilder strings.Builder
	var errorBuilder strings.Builder
	var wg sync.WaitGroup
	var streamErr error

	// Stream stdout
	stdoutStreamer := &pipeStreamer{
		ctx:      ctx,
		resultCh: resultCh,
		prefix:   "",
		builder:  &outputBuilder,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		if err := stdoutStreamer.stream(scanner); err != nil {
			streamErr = err
		}
	}()

	// Stream stderr
	stderrStreamer := &pipeStreamer{
		ctx:      ctx,
		resultCh: resultCh,
		prefix:   "[stderr] ",
		builder:  &errorBuilder,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		if err := stderrStreamer.stream(scanner); err != nil && streamErr == nil {
			streamErr = err
		}
	}()

	// Wait for command to complete and streams to finish
	cmdErr := cmd.Wait()
	wg.Wait()

	executionTime := time.Since(start)
	output := outputBuilder.String()
	errorOutput := errorBuilder.String()

	// Combine stdout and stderr if both have content
	combinedOutput := output
	if errorOutput != "" {
		if combinedOutput != "" {
			combinedOutput += "\n" + errorOutput
		} else {
			combinedOutput = errorOutput
		}
	}

	result := t.buildCommandResult(combinedOutput, cmdErr, streamErr, executionTime, command, workingDir)

	// Send error to channel if present
	if cmdErr != nil {
		select {
		case resultCh <- fmt.Sprintf("\n[Error: %v]\n", cmdErr):
		default:
		}
	} else if streamErr != nil {
		select {
		case resultCh <- fmt.Sprintf("\n[Stream error: %v]\n", streamErr):
		default:
		}
	}

	return result, cmdErr
}

// buildCommandResult creates a ToolResult from command execution output
func (t *CommandTool) buildCommandResult(output string, cmdErr, streamErr error, executionTime time.Duration, command, workingDir string) ToolResult {
	result := ToolResult{
		Content:       output,
		Success:       cmdErr == nil && streamErr == nil,
		ToolName:      "execute_command",
		ExecutionTime: executionTime,
		Metadata: map[string]interface{}{
			"command":     command,
			"working_dir": workingDir,
		},
	}

	if cmdErr != nil {
		result.Error = cmdErr.Error()
		if exitError, ok := cmdErr.(*exec.ExitError); ok {
			result.Metadata["exit_code"] = exitError.ExitCode()
		}
	} else if streamErr != nil {
		result.Error = streamErr.Error()
	}

	return result
}
