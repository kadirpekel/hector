package llms

import "github.com/kadirpekel/hector/pkg/a2a"

// ============================================================================
// A2A MESSAGE HELPER FUNCTIONS
// Shared utilities for working with A2A protocol messages in LLM providers
// ============================================================================

// ExtractTextFromMessage extracts text content from an A2A message
func ExtractTextFromMessage(msg a2a.Message) string {
	for _, part := range msg.Parts {
		if part.Type == a2a.PartTypeText {
			return part.Text
		}
	}
	return ""
}

// ExtractToolCallsFromMessage extracts tool calls from an A2A message
func ExtractToolCallsFromMessage(msg a2a.Message) []a2a.ToolCall {
	return msg.ToolCalls
}

// ExtractToolCallIDFromMessage extracts tool call ID from an A2A message
func ExtractToolCallIDFromMessage(msg a2a.Message) string {
	return msg.ToolCallID
}
