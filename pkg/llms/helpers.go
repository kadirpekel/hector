package llms

import (
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
)

// ============================================================================
// A2A MESSAGE HELPER FUNCTIONS
// Shared utilities for working with A2A protocol messages in LLM providers
// ============================================================================

// These helper functions are deprecated - use protocol.ExtractTextFromMessage and protocol.GetToolCallsFromMessage instead
// Keeping them for backward compatibility during migration

// ExtractTextFromMessage is deprecated - use protocol.ExtractTextFromMessage instead
func ExtractTextFromMessage(msg *pb.Message) string {
	return protocol.ExtractTextFromMessage(msg)
}

// ExtractToolCallsFromMessage is deprecated - use protocol.GetToolCallsFromMessage instead
func ExtractToolCallsFromMessage(msg *pb.Message) []*protocol.ToolCall {
	return protocol.GetToolCallsFromMessage(msg)
}

// ExtractToolCallIDFromMessage is deprecated
func ExtractToolCallIDFromMessage(msg *pb.Message) string {
	return msg.MessageId
}
