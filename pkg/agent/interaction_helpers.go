package agent

import (
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/protocol"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	// Decision constants for human-in-the-loop tool approval
	DecisionApprove = "approve"
	DecisionDeny    = "deny"
)

// parseUserDecision extracts the user's decision from a message
// Checks DataPart first (structured), then falls back to TextPart
// Returns: DecisionApprove or DecisionDeny
func parseUserDecision(msg *pb.Message) string {
	if msg == nil {
		return DecisionDeny // Safe default
	}

	// Check DataPart first (structured response)
	for _, part := range msg.Parts {
		if dataPart := part.GetData(); dataPart != nil {
			// Validate DataPart structure before accessing
			if dataPart.Data != nil && dataPart.Data.Fields != nil {
				if decision, ok := dataPart.Data.Fields["decision"]; ok && decision != nil {
					decisionStr := strings.ToLower(strings.TrimSpace(decision.GetStringValue()))
					// Only accept valid decisions
					if decisionStr == DecisionApprove || decisionStr == DecisionDeny {
						return decisionStr
					}
				}
			}
		}
	}

	// Fallback to TextPart (natural language)
	text := strings.ToLower(strings.TrimSpace(protocol.ExtractTextFromMessage(msg)))

	// Check for explicit keywords
	if strings.Contains(text, DecisionApprove) || text == "yes" || text == "y" {
		return DecisionApprove
	}
	if strings.Contains(text, DecisionDeny) || text == "no" || text == "n" {
		return DecisionDeny
	}

	return DecisionDeny // Safe default
}

// createInteractionMessage creates an A2A-compliant message for INPUT_REQUIRED state
// This message will appear in TaskStatus.update field
func createInteractionMessage(
	interactionType string,
	toolName string,
	toolInput string,
	prompt string,
	options []string,
) *pb.Message {
	// Build structured data for interaction
	fields := map[string]*structpb.Value{
		"interaction_type": structpb.NewStringValue(interactionType),
	}

	if toolName != "" {
		fields["tool_name"] = structpb.NewStringValue(toolName)
	}
	if toolInput != "" {
		fields["tool_input"] = structpb.NewStringValue(toolInput)
	}
	if len(options) > 0 {
		optionValues := make([]*structpb.Value, len(options))
		for i, opt := range options {
			optionValues[i] = structpb.NewStringValue(opt)
		}
		fields["options"] = structpb.NewListValue(&structpb.ListValue{Values: optionValues})
	}

	return &pb.Message{
		Role: pb.Role_ROLE_AGENT,
		Parts: []*pb.Part{
			// TextPart: Human-readable prompt
			{
				Part: &pb.Part_Text{
					Text: prompt,
				},
			},
			// DataPart: Structured metadata for programmatic parsing
			{
				Part: &pb.Part_Data{
					Data: &pb.DataPart{
						Data: &structpb.Struct{
							Fields: fields,
						},
					},
				},
			},
		},
	}
}

// createToolApprovalMessage creates a message requesting tool approval
// A2A Protocol compliant: uses TextPart + DataPart structure
func createToolApprovalMessage(toolName string, toolInput string, customPrompt string) *pb.Message {
	// Safety: ensure toolName is not empty
	if toolName == "" {
		toolName = "unknown_tool"
	}
	// Safety: ensure toolInput is not empty for display
	if toolInput == "" {
		toolInput = "{}"
	}

	// Use custom prompt if provided, otherwise generate default
	prompt := customPrompt
	if prompt == "" {
		prompt = fmt.Sprintf(
			"🔐 Tool Approval Required\n\nTool: %s\nInput: %s\n\nPlease respond with: approve or deny",
			toolName,
			toolInput,
		)
	} else {
		// Interpolate variables in custom prompt
		prompt = strings.ReplaceAll(prompt, "{tool}", toolName)
		prompt = strings.ReplaceAll(prompt, "{input}", toolInput)
	}

	return createInteractionMessage(
		"tool_approval",
		toolName,
		toolInput,
		prompt,
		[]string{DecisionApprove, DecisionDeny},
	)
}

// isApprovalRequired checks if a tool requires approval based on configuration
func isApprovalRequired(toolConfig *config.ToolConfig) bool {
	if toolConfig == nil {
		return false
	}

	if toolConfig.RequiresApproval != nil {
		return *toolConfig.RequiresApproval
	}

	// Default: no approval required
	return false
}

// getApprovalPrompt gets the custom approval prompt or returns empty string
func getApprovalPrompt(toolConfig *config.ToolConfig) string {
	if toolConfig == nil {
		return ""
	}
	return toolConfig.ApprovalPrompt
}
