package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a"
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
// Returns error if discovery fails - caller must handle
func NewA2AAgentFromURL(ctx context.Context, agentURL string, client *a2a.Client) (*A2AAgent, error) {
	// Discover the agent immediately
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
// Converts Task to message, calls remote agent, returns updated Task
func (a *A2AAgent) ExecuteTask(ctx context.Context, task *a2a.Task) (*a2a.Task, error) {
	// Get the last user message
	if len(task.Messages) == 0 {
		return nil, fmt.Errorf("task has no messages")
	}

	lastMessage := task.Messages[len(task.Messages)-1]

	// Send message to remote agent
	resultTask, err := a.client.SendMessage(ctx, a.agentCard.URL, lastMessage, nil)
	if err != nil {
		return nil, fmt.Errorf("A2A agent %s execution failed: %w", a.agentCard.Name, err)
	}

	return resultTask, nil
}

// ExecuteTaskStreaming implements a2a.Agent.ExecuteTaskStreaming
func (a *A2AAgent) ExecuteTaskStreaming(ctx context.Context, task *a2a.Task) (<-chan a2a.StreamEvent, error) {
	// Get the last user message
	if len(task.Messages) == 0 {
		return nil, fmt.Errorf("task has no messages")
	}

	lastMessage := task.Messages[len(task.Messages)-1]

	// Use client's streaming method
	eventCh, err := a.client.SendMessageStreaming(ctx, a.agentCard.URL, lastMessage)
	if err != nil {
		return nil, fmt.Errorf("A2A agent %s streaming failed: %w", a.agentCard.Name, err)
	}

	return eventCh, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// ExtractTextFromTask extracts text content from a task (for backward compat)
func ExtractTextFromTask(task *a2a.Task) string {
	if task == nil {
		return ""
	}

	var texts []string

	// Extract from assistant messages
	for _, msg := range task.Messages {
		if msg.Role == a2a.MessageRoleAssistant {
			for _, part := range msg.Parts {
				if part.Type == a2a.PartTypeText {
					texts = append(texts, part.Text)
				}
			}
		}
	}

	// Extract from artifacts
	for _, artifact := range task.Artifacts {
		for _, part := range artifact.Parts {
			if part.Type == a2a.PartTypeText {
				texts = append(texts, part.Text)
			}
		}
	}

	return strings.Join(texts, "\n")
}

// ============================================================================
// COMPILE-TIME CHECK
// ============================================================================

// Ensure A2AAgent implements pure a2a.Agent interface
var _ a2a.Agent = (*A2AAgent)(nil)
