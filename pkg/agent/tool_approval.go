package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/protocol"
)

// ToolApprovalResult represents the result of tool approval check
type ToolApprovalResult struct {
	ApprovedCalls   []*protocol.ToolCall // Tools that were approved
	DeniedCalls     []*protocol.ToolCall // Tools that were denied by user
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
		DeniedCalls:    make([]*protocol.ToolCall, 0),
		NeedsUserInput: false,
	}

	// Safety check: if toolConfigs is nil, allow all tools (no approval configured)
	if toolConfigs == nil {
		slog.Info("toolConfigs is nil, allowing all tools without approval check")
		result.ApprovedCalls = toolCalls
		return result, nil
	}

	// Get taskID from context if available
	taskID := getTaskIDFromContext(ctx)

	// Check if we're resuming from INPUT_REQUIRED state with a user decision
	userDecision := getUserDecisionFromContext(ctx)

	for _, call := range toolCalls {
		// Get tool configuration
		toolConfig, exists := toolConfigs[call.Name]
		if !exists {
			// Tool not in config, allow by default
			slog.Info("Tool not found in config map, allowing without approval", "tool", call.Name, "available_keys", getConfigKeys(toolConfigs))
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
		slog.Debug("Tool requires approval", "tool", call.Name, "task", taskID)

		// If we have a user decision (resuming from INPUT_REQUIRED), apply it
		if userDecision != "" {
			switch userDecision {
			case DecisionApprove:
				slog.Debug("User approved tool", "tool", call.Name)
				result.ApprovedCalls = append(result.ApprovedCalls, call)

			case DecisionDeny:
				slog.Debug("User denied tool", "tool", call.Name)
				result.DeniedCalls = append(result.DeniedCalls, call)

			default:
				// Unknown decision, deny for safety
				slog.Warn("Unknown decision, denying tool", "decision", userDecision, "tool", call.Name)
				result.DeniedCalls = append(result.DeniedCalls, call)
			}

			continue // Move to next tool call
		}

		// No user decision yet - need to request approval
		// Currently handles one tool approval at a time for simplicity and clarity.
		// This ensures users can make informed decisions about each tool call individually.
		if taskID == "" {
			// Can't request approval without task ID (non-async mode)
			slog.Warn("Tool requires approval but no taskID available, denying", "tool", call.Name)
			result.DeniedCalls = append(result.DeniedCalls, call)
			continue
		}

		// Create approval request message (A2A compliant)
		customPrompt := getApprovalPrompt(toolConfig)
		// Convert Args to JSON string for display
		argsJSON, err := json.Marshal(call.Args)
		argsStr := string(argsJSON)
		if err != nil {
			argsStr = fmt.Sprintf("%v", call.Args)
		}
		interactionMsg := createToolApprovalMessage(call.Name, argsStr, customPrompt)

		result.NeedsUserInput = true
		result.InteractionMsg = interactionMsg
		result.PendingToolCall = call

		// Only handle one approval at a time (see NOTE above for batch enhancement plan)
		// Return immediately so task can transition to INPUT_REQUIRED
		return result, nil
	}

	return result, nil
}

// getConfigKeys returns a slice of all keys in the toolConfigs map for debugging
func getConfigKeys(toolConfigs map[string]*config.ToolConfig) []string {
	if toolConfigs == nil {
		return []string{}
	}
	keys := make([]string, 0, len(toolConfigs))
	for k := range toolConfigs {
		keys = append(keys, k)
	}
	return keys
}
