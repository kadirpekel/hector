package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
)

type AgentCallTool struct {
	name        string
	description string
	registry    AgentRegistry
}

type AgentRegistry interface {
	GetAgent(name string) (pb.A2AServiceServer, error)
}

func NewAgentCallTool(registry AgentRegistry) *AgentCallTool {
	return &AgentCallTool{
		name:        "agent_call",
		description: "Call another agent to delegate a task or get specialized assistance. Use this tool when you need information or capabilities that another agent provides. You MUST use the exact agent ID from the available agents list - do not invent or abbreviate agent names.",
		registry:    registry,
	}
}

func (t *AgentCallTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name:        t.name,
		Description: t.description,
		Parameters: []ToolParameter{
			{
				Name:        "agent",
				Type:        "string",
				Description: "The exact agent ID to call (must match one of the available agents listed in the context). Use the full agent ID exactly as shown - do not abbreviate or invent names.",
				Required:    true,
			},
			{
				Name:        "task",
				Type:        "string",
				Description: "The task, question, or request to send to the agent. Be clear and specific about what information or action you need.",
				Required:    true,
			},
		},
	}
}

func (t *AgentCallTool) GetName() string {
	return t.name
}

func (t *AgentCallTool) GetDescription() string {
	return t.description
}

func (t *AgentCallTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	agentID, ok := args["agent"].(string)
	if !ok {

		if agentID, ok = args["agent_name"].(string); !ok {
			return ToolResult{
				Success: false,
				Error:   "Missing or invalid 'agent' parameter",
			}, nil
		}
	}

	task, ok := args["task"].(string)
	if !ok {

		if task, ok = args["message"].(string); !ok {
			return ToolResult{
				Success: false,
				Error:   "Missing or invalid 'task' parameter",
			}, nil
		}
	}

	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return ToolResult{
			Success: false,
			Error:   "agent ID cannot be empty",
		}, nil
	}

	task = strings.TrimSpace(task)
	if task == "" {
		return ToolResult{
			Success: false,
			Error:   "task cannot be empty",
		}, nil
	}

	if t.registry == nil {
		return ToolResult{
			Success: false,
			Error:   "agent registry not available",
		}, nil
	}

	targetAgent, err := t.registry.GetAgent(agentID)
	if err != nil {
		// Extract available agents from error message if present
		errStr := err.Error()

		// Make the error message more actionable for the LLM
		// The registry error already includes available agents, but we can enhance it
		var errorMsg string
		if strings.Contains(errStr, "Available agents:") {
			// Error already has available agents list - enhance it with explicit instructions
			// Make it clear and actionable for the LLM without being too harsh
			errorMsg = fmt.Sprintf("Agent '%s' was not found. The agent name you used does not exist.\n\n%s\n\nTo fix this:\n- Use one of the exact agent IDs listed above\n- Do not invent agent names - only use the IDs from the list above\n\nPlease retry the agent_call tool with the correct agent ID.", agentID, errStr)
		} else {
			// Try to get available agents from registry if possible
			// Note: We can't easily access ListAgents() from the interface, so rely on error message
			errorMsg = fmt.Sprintf("Agent '%s' not found. %s\n\nPlease check the available agents list in the context and use the correct agent ID.", agentID, errStr)
		}

		return ToolResult{
			Success: false,
			Content: errorMsg, // Put error message in Content so LLM sees it clearly
			Error:   errorMsg,
		}, fmt.Errorf("agent '%s' not found: %v", agentID, err)
	}

	request := &pb.SendMessageRequest{
		Request: &pb.Message{
			MessageId: fmt.Sprintf("agent_call_%s_%d", agentID, time.Now().UnixNano()),
			ContextId: fmt.Sprintf("%s:agent_call_session", agentID),
			Parts: []*pb.Part{
				{
					Part: &pb.Part_Text{Text: task},
				},
			},
		},
	}

	response, err := targetAgent.SendMessage(ctx, request)
	if err != nil {
		// Provide more descriptive error messages for common failure scenarios
		errorMsg := fmt.Sprintf("Failed to call agent '%s': %v", agentID, err)

		// Check for common error patterns and provide helpful context
		errStr := err.Error()
		if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") {
			errorMsg = fmt.Sprintf("Agent '%s' is not reachable at its configured URL. The agent service may be down or the URL is incorrect. Error: %v", agentID, err)
		} else if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
			errorMsg = fmt.Sprintf("Agent '%s' did not respond within the timeout period. The agent may be overloaded or slow. Error: %v", agentID, err)
		} else if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
			errorMsg = fmt.Sprintf("Agent '%s' is rate limiting requests. Please wait and try again later. Error: %v", agentID, err)
		} else if strings.Contains(errStr, "not found") || strings.Contains(errStr, "404") {
			errorMsg = fmt.Sprintf("Agent '%s' was not found. The agent may not be registered or the agent ID is incorrect. Error: %v", agentID, err)
		}

		return ToolResult{
			Success: false,
			Content: errorMsg, // Put error message in Content so LLM sees it clearly
			Error:   errorMsg,
		}, fmt.Errorf("failed to call agent '%s': %v", agentID, err)
	}

	var responseText string
	if response.Payload != nil {
		switch payload := response.Payload.(type) {
		case *pb.SendMessageResponse_Msg:
			if payload.Msg != nil {
				// Extract all text parts (not just the first one)
				responseText = protocol.ExtractAllTextFromMessage(payload.Msg)
			}
		case *pb.SendMessageResponse_Task:
			if payload.Task != nil {
				// For task responses, extract text from the task history if available
				// With blocking=true (default), task should be in a terminal state (COMPLETED, FAILED, etc.)
				if payload.Task.Status != nil {
					state := payload.Task.Status.State

					// Handle terminal states (COMPLETED, FAILED, CANCELLED, REJECTED)
					if state == pb.TaskState_TASK_STATE_COMPLETED ||
						state == pb.TaskState_TASK_STATE_FAILED ||
						state == pb.TaskState_TASK_STATE_CANCELLED ||
						state == pb.TaskState_TASK_STATE_REJECTED {

						// Try to extract text from task history first
						if taskText := protocol.ExtractTextFromTask(payload.Task); taskText != "" {
							responseText = taskText
						} else {
							// Task in terminal state but no text found - try extracting from message parts
							// Check status update message first (may contain error info for FAILED)
							if payload.Task.Status.Update != nil {
								if statusText := protocol.ExtractAllTextFromMessage(payload.Task.Status.Update); statusText != "" {
									responseText = statusText
								}
							}

							// If still no text, try extracting from the last agent message in history
							if responseText == "" && len(payload.Task.History) > 0 {
								for i := len(payload.Task.History) - 1; i >= 0; i-- {
									if payload.Task.History[i].Role == pb.Role_ROLE_AGENT {
										if msgText := protocol.ExtractAllTextFromMessage(payload.Task.History[i]); msgText != "" {
											responseText = msgText
											break
										}
									}
								}
							}

							// If still no text, provide informative message based on state
							if responseText == "" {
								switch state {
								case pb.TaskState_TASK_STATE_FAILED:
									responseText = fmt.Sprintf("Task %s failed but no error message was provided", payload.Task.Id)
								case pb.TaskState_TASK_STATE_CANCELLED:
									responseText = fmt.Sprintf("Task %s was cancelled", payload.Task.Id)
								case pb.TaskState_TASK_STATE_REJECTED:
									responseText = fmt.Sprintf("Task %s was rejected by the agent", payload.Task.Id)
								default:
									responseText = fmt.Sprintf("Task %s completed but no response content found", payload.Task.Id)
								}
							}
						}
					} else {
						// Task not in terminal state (shouldn't happen with blocking=true, but handle gracefully)
						// This could happen if task is INPUT_REQUIRED, WORKING, etc.
						statusStr := state.String()
						responseText = fmt.Sprintf("Task %s is still %s (expected terminal state with blocking=true)", payload.Task.Id, statusStr)
					}
				} else {
					// Task has no status (shouldn't happen, but handle gracefully)
					responseText = fmt.Sprintf("Task %s has no status information", payload.Task.Id)
				}
			}
		}
	}

	if responseText == "" {
		responseText = "No response content"
	}

	return ToolResult{
		Success: true,
		Content: fmt.Sprintf("[Delegated to: %s]\n\n%s", agentID, responseText),
		Metadata: map[string]interface{}{
			"agent_id":          agentID,
			"task":              task,
			"execution_time_ms": time.Since(start).Milliseconds(),
		},
	}, nil
}
