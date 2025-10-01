package agent

import (
	"context"
	"strings"
	"time"

	"github.com/kadirpekel/hector/component"
	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/reasoning"
)

// Agent is now a minimal wrapper around ReasoningEngine
// No redundant wrapper methods - callers should use services directly
type Agent struct {
	name            string
	description     string
	config          *config.AgentConfig
	reasoningEngine reasoning.ReasoningEngine
}

// NewAgent creates a new agent (minimal wrapper)
func NewAgent(agentConfig *config.AgentConfig, componentManager *component.ComponentManager) (*Agent, error) {
	reasoningEngine, err := NewReasoningEngine(agentConfig, componentManager)
	if err != nil {
		return nil, err
	}

	return &Agent{
		name:            agentConfig.Name,
		description:     agentConfig.Description,
		config:          agentConfig,
		reasoningEngine: reasoningEngine,
	}, nil
}

// Query executes a query using the reasoning engine (non-streaming interface for compatibility)
func (a *Agent) Query(ctx context.Context, query string) (*reasoning.ReasoningResponse, error) {
	start := time.Now()

	// Use the unified streaming interface internally
	streamCh, err := a.reasoningEngine.Execute(ctx, query)
	if err != nil {
		return nil, err
	}

	// Collect all streaming output
	var fullResponse strings.Builder
	var tokensUsed int

	for chunk := range streamCh {
		fullResponse.WriteString(chunk)
		// Rough token estimation
		tokensUsed += len(strings.Fields(chunk))
	}

	return &reasoning.ReasoningResponse{
		Answer:     fullResponse.String(),
		TokensUsed: tokensUsed,
		Duration:   time.Since(start),
		Confidence: 0.8, // Good confidence for default approach
	}, nil
}

// QueryStreaming executes a query with streaming output through reasoning engine
func (a *Agent) QueryStreaming(ctx context.Context, query string) (<-chan string, error) {
	return a.reasoningEngine.Execute(ctx, query)
}

// GetName returns the agent's name
func (a *Agent) GetName() string {
	return a.name
}

// GetDescription returns the agent's description
func (a *Agent) GetDescription() string {
	return a.description
}

// GetConfig returns the agent's configuration
func (a *Agent) GetConfig() *config.AgentConfig {
	return a.config
}

// GetReasoningEngine returns the reasoning engine for direct access
func (a *Agent) GetReasoningEngine() reasoning.ReasoningEngine {
	return a.reasoningEngine
}
