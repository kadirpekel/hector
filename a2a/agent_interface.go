package a2a

import (
	"context"
)

// ============================================================================
// PURE A2A PROTOCOL INTERFACE
// Based on https://a2a-protocol.org/latest/
// ============================================================================

// Agent represents any A2A-compliant agent
// This is the pure A2A protocol interface - no abstractions, no convenience wrappers
type Agent interface {
	// GetAgentCard returns the agent's capability card for discovery
	// This is how agents advertise their capabilities, endpoints, and metadata
	GetAgentCard() *AgentCard

	// ExecuteTask executes an A2A task
	// Takes a TaskRequest (with structured input, parameters, context)
	// Returns a TaskResponse (with structured output, status, metadata)
	ExecuteTask(ctx context.Context, request *TaskRequest) (*TaskResponse, error)

	// ExecuteTaskStreaming executes an A2A task with streaming output (optional)
	// Returns a channel of StreamChunks for real-time output
	ExecuteTaskStreaming(ctx context.Context, request *TaskRequest) (<-chan *StreamChunk, error)
}

// ============================================================================
// PURE A2A PROTOCOL - NO HECTOR-SPECIFIC CODE
// The a2a package should be completely standalone and reusable
// ============================================================================
