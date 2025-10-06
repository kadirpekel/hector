package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a"
	"github.com/kadirpekel/hector/pkg/tools"
)

// ============================================================================
// AGENT CALL TOOL - Enables orchestration via agent delegation
// Lives in agent package to avoid cyclic imports
// ============================================================================

// AgentCallTool allows an agent to delegate tasks to other agents
// Works transparently with both native Agent and A2AAgent via Executable interface
type AgentCallTool struct {
	registry *AgentRegistry
}

// NewAgentCallTool creates a new agent call tool
func NewAgentCallTool(registry *AgentRegistry) tools.Tool {
	return &AgentCallTool{
		registry: registry,
	}
}

// GetInfo implements tools.Tool.GetInfo
func (t *AgentCallTool) GetInfo() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        "agent_call",
		Description: "Call another agent to delegate a subtask. Use this for multi-agent orchestration.",
		Parameters: []tools.ToolParameter{
			{
				Name:        "agent",
				Type:        "string",
				Description: "The name of the agent to call",
				Required:    true,
			},
			{
				Name:        "task",
				Type:        "string",
				Description: "The task or prompt to send to the agent",
				Required:    true,
			},
		},
	}
}

// GetName implements tools.Tool.GetName
func (t *AgentCallTool) GetName() string {
	return "agent_call"
}

// GetDescription implements tools.Tool.GetDescription
func (t *AgentCallTool) GetDescription() string {
	return "Call another agent to delegate a subtask. Use this for multi-agent orchestration."
}

// Execute implements tools.Tool.Execute
func (t *AgentCallTool) Execute(ctx context.Context, args map[string]interface{}) (tools.ToolResult, error) {
	start := time.Now()

	// Extract arguments
	agentName, ok := args["agent"].(string)
	if !ok || agentName == "" {
		return tools.ToolResult{
			Success:       false,
			ToolName:      "agent_call",
			Error:         "Missing or invalid 'agent' parameter",
			ExecutionTime: time.Since(start),
		}, fmt.Errorf("agent name is required")
	}

	task, ok := args["task"].(string)
	if !ok || task == "" {
		return tools.ToolResult{
			Success:       false,
			ToolName:      "agent_call",
			Error:         "Missing or invalid 'task' parameter",
			ExecutionTime: time.Since(start),
		}, fmt.Errorf("task is required")
	}

	// Get agent from registry (pure a2a.Agent interface)
	targetAgent, err := t.registry.GetAgent(agentName)
	if err != nil {
		return tools.ToolResult{
			Success:       false,
			ToolName:      "agent_call",
			Error:         fmt.Sprintf("Agent '%s' not found: %v", agentName, err),
			ExecutionTime: time.Since(start),
		}, err
	}

	// Create A2A TaskRequest
	taskRequest := &a2a.TaskRequest{
		TaskID: fmt.Sprintf("task-%d", time.Now().UnixNano()),
		Input: a2a.TaskInput{
			Type:    "text/plain",
			Content: task,
		},
	}

	// Execute the agent using pure A2A protocol
	taskResponse, err := targetAgent.ExecuteTask(ctx, taskRequest)
	if err != nil {
		return tools.ToolResult{
			Success:       false,
			ToolName:      "agent_call",
			Error:         fmt.Sprintf("Agent '%s' execution failed: %v", agentName, err),
			Content:       fmt.Sprintf("Called agent '%s' but got error: %v", agentName, err),
			ExecutionTime: time.Since(start),
		}, err
	}

	// Check A2A task status
	if taskResponse.Status == a2a.TaskStatusFailed {
		errorMsg := "Task failed"
		if taskResponse.Error != nil {
			errorMsg = taskResponse.Error.Message
		}
		return tools.ToolResult{
			Success:       false,
			ToolName:      "agent_call",
			Error:         errorMsg,
			Content:       fmt.Sprintf("Agent '%s' task failed: %s", agentName, errorMsg),
			ExecutionTime: time.Since(start),
		}, fmt.Errorf("%s", errorMsg)
	}

	// Extract output from A2A response
	output := a2a.ExtractOutputText(taskResponse.Output)

	// Return successful result
	return tools.ToolResult{
		Success:       true,
		ToolName:      "agent_call",
		Content:       fmt.Sprintf("[Delegated to: %s]\n\n%s", agentName, output),
		ExecutionTime: time.Since(start),
		Metadata: map[string]interface{}{
			"agent_name":        agentName,
			"task":              task,
			"execution_time_ms": time.Since(start).Milliseconds(),
			"a2a_task_id":       taskResponse.TaskID,
			"a2a_status":        string(taskResponse.Status),
		},
	}, nil
}

// ============================================================================
// COMPILE-TIME CHECK
// ============================================================================

var _ tools.Tool = (*AgentCallTool)(nil)
