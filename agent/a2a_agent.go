package agent

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/a2a"
)

// ============================================================================
// A2A AGENT - Remote agent accessed via A2A protocol
// Implements pure a2a.Agent interface using a2a.Client
// ============================================================================

// A2AAgent represents a remote agent accessed via the A2A protocol
// It implements the pure a2a.Agent interface by delegating to an A2A client
type A2AAgent struct {
	agentCard *a2a.AgentCard
	client    *a2a.Client
}

// NewA2AAgent creates a new A2A agent from a discovered agent card
func NewA2AAgent(agentCard *a2a.AgentCard, client *a2a.Client) *A2AAgent {
	return &A2AAgent{
		agentCard: agentCard,
		client:    client,
	}
}

// NewA2AAgentFromURL creates a new A2A agent by discovering it from a URL
func NewA2AAgentFromURL(ctx context.Context, agentURL string, client *a2a.Client) (*A2AAgent, error) {
	// Discover the agent
	agentCard, err := client.DiscoverAgent(ctx, agentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover A2A agent at %s: %w", agentURL, err)
	}

	return NewA2AAgent(agentCard, client), nil
}

// ============================================================================
// PURE A2A INTERFACE IMPLEMENTATION
// ============================================================================

// GetAgentCard implements a2a.Agent.GetAgentCard
func (a *A2AAgent) GetAgentCard() *a2a.AgentCard {
	return a.agentCard
}

// ExecuteTask implements a2a.Agent.ExecuteTask
// Pure A2A protocol: TaskRequest → TaskResponse
func (a *A2AAgent) ExecuteTask(ctx context.Context, request *a2a.TaskRequest) (*a2a.TaskResponse, error) {
	// Execute task via A2A client (pure protocol pass-through)
	result, err := a.client.ExecuteTaskRequest(ctx, a.agentCard, request)
	if err != nil {
		return nil, fmt.Errorf("A2A agent %s execution failed: %w", a.agentCard.Name, err)
	}

	return result, nil
}

// ExecuteTaskStreaming implements a2a.Agent.ExecuteTaskStreaming
func (a *A2AAgent) ExecuteTaskStreaming(ctx context.Context, request *a2a.TaskRequest) (<-chan *a2a.StreamChunk, error) {
	// TODO: Implement true A2A streaming when a2a.Client supports it
	// For now, fall back to non-streaming
	streamCh := make(chan *a2a.StreamChunk, 1)

	go func() {
		defer close(streamCh)

		result, err := a.ExecuteTask(ctx, request)
		if err != nil {
			streamCh <- &a2a.StreamChunk{
				TaskID:    request.TaskID,
				ChunkType: a2a.ChunkTypeText,
				Content:   fmt.Sprintf("Error: %v", err),
				Final:     true,
			}
			return
		}

		// Send the result as a single chunk
		if result.Output != nil {
			streamCh <- &a2a.StreamChunk{
				TaskID:    request.TaskID,
				ChunkType: a2a.ChunkTypeText,
				Content:   a2a.ExtractOutputText(result.Output),
				Final:     true,
			}
		}
	}()

	return streamCh, nil
}

// Note: Shared helpers (extractInputText) are in a2a_helpers.go

// ============================================================================
// COMPILE-TIME CHECK
// ============================================================================

// Ensure A2AAgent implements pure a2a.Agent interface
var _ a2a.Agent = (*A2AAgent)(nil)
