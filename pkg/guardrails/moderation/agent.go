package moderation

import (
	"context"
	"fmt"

	"github.com/verikod/hector/pkg/agent"
)

// AgentConfig configures the agent-based moderation provider.
type AgentConfig struct {
	// Agent is the Hector agent to use for moderation.
	// The agent should output JSON with a safety field.
	Agent agent.Agent

	// SafeField is the JSON field to check for safety.
	// Default: "safe"
	SafeField string

	// Session is the session service for agent invocation.
	Session any // session.Service interface
}

// agentProvider implements Provider using a Hector agent.
type agentProvider struct {
	agent     agent.Agent
	safeField string
}

// NewAgent creates an agent-based moderation provider.
// This allows using any Hector agent (including workflows) as a moderator.
func NewAgent(cfg AgentConfig) Provider {
	safeField := cfg.SafeField
	if safeField == "" {
		safeField = "safe"
	}

	return &agentProvider{
		agent:     cfg.Agent,
		safeField: safeField,
	}
}

func (p *agentProvider) Name() string {
	if p.agent != nil {
		return "agent:" + p.agent.Name()
	}
	return "agent"
}

func (p *agentProvider) Moderate(ctx context.Context, content string) (*Result, error) {
	if p.agent == nil {
		return nil, fmt.Errorf("agent moderation: agent not configured")
	}

	// Note: Full agent invocation requires session/runner context.
	// This is a simplified implementation that shows the pattern.
	// In practice, this would need to be integrated with the runner.

	// For now, return a placeholder that indicates the agent strategy
	// requires integration with the runner/session layer.
	return nil, fmt.Errorf("agent moderation: requires runner integration (use conditional workflow agent instead)")
}
