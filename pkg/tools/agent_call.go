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

// StreamingAgentClient is an interface for agents that support streaming.
// This allows us to use true streaming for both local and external agents.
//
// Example usage:
//
//	resultCh := make(chan string, 10)
//	go func() {
//		for chunk := range resultCh {
//			fmt.Print(chunk) // Process chunks as they arrive
//		}
//	}()
//	streamChan, err := client.StreamMessage(ctx, agentID, message)
//	if err != nil {
//		return err
//	}
//	for resp := range streamChan {
//		// Process StreamResponse messages
//	}
type StreamingAgentClient interface {
	// StreamMessage streams messages from the agent in real-time.
	// Returns a channel of StreamResponse messages that will be closed when streaming completes.
	StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error)
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

// validateAndExtractArgs validates and extracts agent and task arguments
// Returns agentID, task, and error if validation fails
func (t *AgentCallTool) validateAndExtractArgs(args map[string]interface{}) (agentID, task string, err error) {
	agentID, ok := args["agent"].(string)
	if !ok {
		if agentID, ok = args["agent_name"].(string); !ok {
			return "", "", fmt.Errorf("missing or invalid 'agent' parameter")
		}
	}

	task, ok = args["task"].(string)
	if !ok {
		if task, ok = args["message"].(string); !ok {
			return "", "", fmt.Errorf("missing or invalid 'task' parameter")
		}
	}

	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return "", "", fmt.Errorf("agent ID cannot be empty")
	}

	task = strings.TrimSpace(task)
	if task == "" {
		return "", "", fmt.Errorf("task cannot be empty")
	}

	if t.registry == nil {
		return "", "", fmt.Errorf("agent registry not available")
	}

	return agentID, task, nil
}

// buildAgentNotFoundError creates a user-friendly error message when agent is not found
func (t *AgentCallTool) buildAgentNotFoundError(agentID string, err error) (ToolResult, error) {
	errStr := err.Error()
	var errorMsg string
	if strings.Contains(errStr, "Available agents:") {
		errorMsg = fmt.Sprintf("Agent '%s' was not found. The agent name you used does not exist.\n\n%s\n\nTo fix this:\n- Use one of the exact agent IDs listed above\n- Do not invent agent names - only use the IDs from the list above\n\nPlease retry the agent_call tool with the correct agent ID.", agentID, errStr)
	} else {
		errorMsg = fmt.Sprintf("Agent '%s' not found. %s\n\nPlease check the available agents list in the context and use the correct agent ID.", agentID, errStr)
	}

	return ToolResult{
		Success: false,
		Content: errorMsg,
		Error:   errorMsg,
	}, fmt.Errorf("agent '%s' not found: %v", agentID, err)
}

// buildAgentCallError creates a user-friendly error message for agent call failures
func (t *AgentCallTool) buildAgentCallError(agentID string, err error) (ToolResult, error) {
	errorMsg := fmt.Sprintf("Failed to call agent '%s': %v", agentID, err)
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
		Content: errorMsg,
		Error:   errorMsg,
	}, fmt.Errorf("failed to call agent '%s': %v", agentID, err)
}

// buildAgentRequest creates a SendMessageRequest for calling another agent
func (t *AgentCallTool) buildAgentRequest(agentID, task string) *pb.SendMessageRequest {
	return &pb.SendMessageRequest{
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
}

// extractResponseText extracts response text from SendMessageResponse (non-streaming version)
func (t *AgentCallTool) extractResponseText(response *pb.SendMessageResponse) string {
	var responseText string
	if response.Payload != nil {
		switch payload := response.Payload.(type) {
		case *pb.SendMessageResponse_Msg:
			if payload.Msg != nil {
				responseText = protocol.ExtractAllTextFromMessage(payload.Msg)
			}
		case *pb.SendMessageResponse_Task:
			if payload.Task != nil {
				if payload.Task.Status != nil {
					state := payload.Task.Status.State
					if state == pb.TaskState_TASK_STATE_COMPLETED ||
						state == pb.TaskState_TASK_STATE_FAILED ||
						state == pb.TaskState_TASK_STATE_CANCELLED ||
						state == pb.TaskState_TASK_STATE_REJECTED {
						if taskText := protocol.ExtractTextFromTask(payload.Task); taskText != "" {
							responseText = taskText
						} else if payload.Task.Status.Update != nil {
							if statusText := protocol.ExtractAllTextFromMessage(payload.Task.Status.Update); statusText != "" {
								responseText = statusText
							}
						} else {
							// Try extracting from the last agent message in history
							if len(payload.Task.History) > 0 {
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
						statusStr := state.String()
						responseText = fmt.Sprintf("Task %s is still %s (expected terminal state with blocking=true)", payload.Task.Id, statusStr)
					}
				} else {
					responseText = fmt.Sprintf("Task %s has no status information", payload.Task.Id)
				}
			}
		}
	}

	if responseText == "" {
		responseText = "No response content"
	}

	return responseText
}

// processAgentResponse extracts response text from SendMessageResponse and streams it
func (t *AgentCallTool) processAgentResponse(response *pb.SendMessageResponse, agentID string, resultCh chan<- string) string {
	responseText := t.extractResponseText(response)
	if responseText != "" {
		resultCh <- fmt.Sprintf("[Delegated to: %s]\n\n", agentID)
		resultCh <- responseText
	} else {
		responseText = "No response content"
		resultCh <- fmt.Sprintf("[Delegated to: %s]\n\n%s", agentID, responseText)
	}
	return responseText
}

// extractTextFromStreamMsg extracts text from a StreamResponse message
func (t *AgentCallTool) extractTextFromStreamMsg(msg *pb.Message) string {
	return protocol.ExtractAllTextFromMessage(msg)
}

// extractTextFromStreamTask extracts text from a StreamResponse task
func (t *AgentCallTool) extractTextFromStreamTask(task *pb.Task) string {
	if task.Status != nil {
		state := task.Status.State
		// Check if task is in terminal state
		if state == pb.TaskState_TASK_STATE_COMPLETED ||
			state == pb.TaskState_TASK_STATE_FAILED ||
			state == pb.TaskState_TASK_STATE_CANCELLED ||
			state == pb.TaskState_TASK_STATE_REJECTED {
			// Extract final text from task
			if taskText := protocol.ExtractTextFromTask(task); taskText != "" {
				return taskText
			} else if task.Status.Update != nil {
				return protocol.ExtractAllTextFromMessage(task.Status.Update)
			}
		}
	}
	return ""
}

// executeStreamingMessage handles true real-time streaming from an agent
func (t *AgentCallTool) executeStreamingMessage(
	ctx context.Context,
	agentID, task string,
	streamingClient StreamingAgentClient,
	message *pb.Message,
	resultCh chan<- string,
	start time.Time,
) (ToolResult, error) {
	// Stream messages in real-time
	streamChan, err := streamingClient.StreamMessage(ctx, agentID, message)
	if err != nil {
		errorResult, callErr := t.buildAgentCallError(agentID, err)
		resultCh <- errorResult.Content
		close(resultCh)
		return errorResult, callErr
	}

	var responseText strings.Builder
	prefixSent := false

	// Stream responses in real-time
	for streamResp := range streamChan {
		if streamResp == nil {
			continue
		}

		// Send delegation prefix on first chunk
		if !prefixSent {
			resultCh <- fmt.Sprintf("[Delegated to: %s]\n\n", agentID)
			prefixSent = true
		}

		// Handle different payload types
		switch payload := streamResp.Payload.(type) {
		case *pb.StreamResponse_Msg:
			if payload.Msg != nil {
				text := t.extractTextFromStreamMsg(payload.Msg)
				if text != "" {
					responseText.WriteString(text)
					// Stream the text chunk immediately
					resultCh <- text
				}
			}
		case *pb.StreamResponse_Task:
			if payload.Task != nil {
				text := t.extractTextFromStreamTask(payload.Task)
				if text != "" {
					responseText.WriteString(text)
					resultCh <- text
				}
			}
		}
	}

	close(resultCh)

	finalText := responseText.String()
	if finalText == "" {
		finalText = "No response content"
	}

	return ToolResult{
		Success: true,
		Content: fmt.Sprintf("[Delegated to: %s]\n\n%s", agentID, finalText),
		Metadata: map[string]interface{}{
			"agent_id":          agentID,
			"task":              task,
			"execution_time_ms": time.Since(start).Milliseconds(),
			"streaming":         true, // Indicate that true streaming was used
		},
	}, nil
}

func (t *AgentCallTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	agentID, task, err := t.validateAndExtractArgs(args)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	targetAgent, err := t.registry.GetAgent(agentID)
	if err != nil {
		return t.buildAgentNotFoundError(agentID, err)
	}

	request := t.buildAgentRequest(agentID, task)
	response, err := targetAgent.SendMessage(ctx, request)
	if err != nil {
		return t.buildAgentCallError(agentID, err)
	}

	// Reuse response extraction helper
	responseText := t.extractResponseText(response)

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

// ExecuteStreaming implements StreamingTool interface for agent_call
// It streams responses from the called agent incrementally
// Note: Currently uses SendMessage (non-streaming) and streams the result chunks as they're extracted.
// TODO: Enhance AgentRegistry to support streaming clients for true real-time streaming support.
func (t *AgentCallTool) ExecuteStreaming(ctx context.Context, args map[string]interface{}, resultCh chan<- string) (ToolResult, error) {
	start := time.Now()

	// Reuse validation helper
	agentID, task, err := t.validateAndExtractArgs(args)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Reuse agent lookup logic
	targetAgent, err := t.registry.GetAgent(agentID)
	if err != nil {
		return t.buildAgentNotFoundError(agentID, err)
	}

	// Reuse request building helper
	request := t.buildAgentRequest(agentID, task)

	// Try to use true streaming if the agent supports it
	// Check if agent implements StreamingAgentClient (external agents)
	if streamingClient, ok := targetAgent.(StreamingAgentClient); ok {
		return t.executeStreamingMessage(ctx, agentID, task, streamingClient, request.Request, resultCh, start)
	}

	// Fallback to non-streaming SendMessage (for local agents or agents without streaming support)
	// Note: Local agents would require creating a streaming server wrapper, which is complex
	// For now, we use SendMessage and stream the result chunks as they're extracted
	response, err := targetAgent.SendMessage(ctx, request)
	if err != nil {
		errorResult, callErr := t.buildAgentCallError(agentID, err)
		// Send error as chunk
		resultCh <- errorResult.Content
		close(resultCh)
		return errorResult, callErr
	}

	// Reuse response processing helper
	responseText := t.processAgentResponse(response, agentID, resultCh)
	close(resultCh)

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
