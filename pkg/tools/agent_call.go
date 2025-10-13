package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// AgentCallTool enables agents to call other agents in a multi-agent system
type AgentCallTool struct {
	name        string
	description string
	registry    AgentRegistry
}

// AgentRegistry interface to avoid import cycle
type AgentRegistry interface {
	GetAgent(name string) (pb.A2AServiceServer, error)
}

// NewAgentCallTool creates a new agent call tool with the given registry
func NewAgentCallTool(registry AgentRegistry) *AgentCallTool {
	return &AgentCallTool{
		name:        "agent_call",
		description: "Call another agent to delegate a task or get specialized assistance",
		registry:    registry,
	}
}

// GetInfo returns metadata about the agent call tool
func (t *AgentCallTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name:        t.name,
		Description: t.description,
		Parameters: []ToolParameter{
			{
				Name:        "agent",
				Type:        "string",
				Description: "Name of the agent to call (e.g., 'weather_expert', 'travel_advisor')",
				Required:    true,
			},
			{
				Name:        "task",
				Type:        "string",
				Description: "Task or message to send to the agent",
				Required:    true,
			},
		},
	}
}

// GetName returns the tool name
func (t *AgentCallTool) GetName() string {
	return t.name
}

// GetDescription returns the tool description
func (t *AgentCallTool) GetDescription() string {
	return t.description
}

// Execute calls another agent with the given message
func (t *AgentCallTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	// Extract arguments - use "agent" and "task" to match legacy interface
	agentName, ok := args["agent"].(string)
	if !ok {
		// Try "agent_name" as fallback
		if agentName, ok = args["agent_name"].(string); !ok {
			return ToolResult{
				Success: false,
				Error:   "Missing or invalid 'agent' parameter",
			}, nil
		}
	}

	task, ok := args["task"].(string)
	if !ok {
		// Try "message" as fallback
		if task, ok = args["message"].(string); !ok {
			return ToolResult{
				Success: false,
				Error:   "Missing or invalid 'task' parameter",
			}, nil
		}
	}

	// Validate agent name
	agentName = strings.TrimSpace(agentName)
	if agentName == "" {
		return ToolResult{
			Success: false,
			Error:   "agent name cannot be empty",
		}, nil
	}

	// Validate task
	task = strings.TrimSpace(task)
	if task == "" {
		return ToolResult{
			Success: false,
			Error:   "task cannot be empty",
		}, nil
	}

	// Check if registry is available
	if t.registry == nil {
		return ToolResult{
			Success: false,
			Error:   "agent registry not available",
		}, nil
	}

	// Get the target agent
	targetAgent, err := t.registry.GetAgent(agentName)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Agent '%s' not found: %v", agentName, err),
		}, fmt.Errorf("agent '%s' not found: %v", agentName, err)
	}

	// Create the A2A request using current protobuf format
	request := &pb.SendMessageRequest{
		Request: &pb.Message{
			MessageId: fmt.Sprintf("agent_call_%s_%d", agentName, time.Now().UnixNano()),
			ContextId: fmt.Sprintf("%s:agent_call_session", agentName), // Set context_id for routing
			Content: []*pb.Part{
				{
					Part: &pb.Part_Text{Text: task},
				},
			},
		},
	}

	// Call the target agent
	response, err := targetAgent.SendMessage(ctx, request)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to call agent '%s': %v", agentName, err),
		}, fmt.Errorf("failed to call agent '%s': %v", agentName, err)
	}

	// Extract response content
	var responseText string
	if response.Payload != nil {
		switch payload := response.Payload.(type) {
		case *pb.SendMessageResponse_Msg:
			if payload.Msg != nil && len(payload.Msg.Content) > 0 {
				// Extract text from the first part
				for _, part := range payload.Msg.Content {
					if textPart := part.GetText(); textPart != "" {
						responseText = textPart
						break
					}
				}
			}
		case *pb.SendMessageResponse_Task:
			if payload.Task != nil {
				responseText = fmt.Sprintf("Task created: %s (status: %s)", payload.Task.Id, payload.Task.Status.String())
			}
		}
	}

	if responseText == "" {
		responseText = "No response content"
	}

	return ToolResult{
		Success: true,
		Content: fmt.Sprintf("[Delegated to: %s]\n\n%s", agentName, responseText),
		Metadata: map[string]interface{}{
			"agent_name":        agentName,
			"task":              task,
			"execution_time_ms": time.Since(start).Milliseconds(),
		},
	}, nil
}
