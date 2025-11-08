package agent

import (
	"context"
	"fmt"
	"log"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/protocol"
)

// ToolApprovalResult represents the result of tool approval check
type ToolApprovalResult struct {
	ApprovedCalls   []*protocol.ToolCall // Tools that were approved
	NeedsUserInput  bool                 // If true, task should pause for INPUT_REQUIRED
	InteractionMsg  *pb.Message          // Message to send in INPUT_REQUIRED state
	PendingToolCall *protocol.ToolCall   // The tool call waiting for approval
}

// filterToolCallsWithApproval checks if any tools require approval
// Returns approved tools and whether the task needs to pause for user input
// A2A Protocol compliant: uses INPUT_REQUIRED state for human-in-the-loop
func (a *Agent) filterToolCallsWithApproval(
	ctx context.Context,
	toolCalls []*protocol.ToolCall,
	toolConfigs map[string]*config.ToolConfig,
) (*ToolApprovalResult, error) {

	result := &ToolApprovalResult{
		ApprovedCalls:  make([]*protocol.ToolCall, 0, len(toolCalls)),
		NeedsUserInput: false,
	}

	// Get taskID from context if available
	taskID := ""
	if taskIDValue := ctx.Value(taskIDContextKey); taskIDValue != nil {
		if tid, ok := taskIDValue.(string); ok {
			taskID = tid
		}
	}

	// Check if we're resuming from INPUT_REQUIRED state with a user decision
	userDecision := ""
	if decisionValue := ctx.Value(userDecisionContextKey); decisionValue != nil {
		if decision, ok := decisionValue.(string); ok {
			userDecision = decision
		}
	}

	for _, call := range toolCalls {
		// Get tool configuration
		toolConfig, exists := toolConfigs[call.Name]
		if !exists {
			// Tool not in config, allow by default
			result.ApprovedCalls = append(result.ApprovedCalls, call)
			continue
		}

		// Check if tool requires approval
		requiresApproval := isApprovalRequired(toolConfig)
		if !requiresApproval {
			// Tool doesn't need approval
			result.ApprovedCalls = append(result.ApprovedCalls, call)
			continue
		}

		// Tool requires approval
		log.Printf("[HITL] Tool %s requires approval (taskID: %s)", call.Name, taskID)

		// If we have a user decision (resuming from INPUT_REQUIRED), apply it
		if userDecision != "" {
			switch userDecision {
			case "approve":
				log.Printf("[HITL] User approved tool %s", call.Name)
				result.ApprovedCalls = append(result.ApprovedCalls, call)

			case "deny":
				log.Printf("[HITL] User denied tool %s", call.Name)
				// Skip this tool (don't add to approved list)

			default:
				// Unknown decision, deny for safety
				log.Printf("[HITL] Unknown decision '%s', denying tool %s", userDecision, call.Name)
			}

			continue // Move to next tool call
		}

		// No user decision yet - need to request approval
		// Only handle one tool approval at a time for simplicity
		if taskID == "" {
			// Can't request approval without task ID (non-async mode)
			log.Printf("[HITL] Warning: Tool %s requires approval but no taskID available, denying", call.Name)
			continue // Skip this tool
		}

		// Create approval request message (A2A compliant)
		customPrompt := getApprovalPrompt(toolConfig)
		// Convert Args to JSON string for display
		argsJSON := fmt.Sprintf("%v", call.Args)
		interactionMsg := createToolApprovalMessage(call.Name, argsJSON, customPrompt)

		result.NeedsUserInput = true
		result.InteractionMsg = interactionMsg
		result.PendingToolCall = call

		// Only handle one approval at a time
		// Return immediately so task can transition to INPUT_REQUIRED
		return result, nil
	}

	return result, nil
}
