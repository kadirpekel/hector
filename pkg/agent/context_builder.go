package agent

import (
	"context"

	"github.com/kadirpekel/hector/pkg/protocol"
)

// AgentContextBuilder helps build consistent agent execution contexts
// with all required values (taskID, sessionID) properly set.
type AgentContextBuilder struct {
	ctx context.Context
}

// NewAgentContext creates a new context builder from a base context
func NewAgentContext(baseCtx context.Context) *AgentContextBuilder {
	return &AgentContextBuilder{ctx: baseCtx}
}

// WithTaskID adds taskID to the context
func (b *AgentContextBuilder) WithTaskID(taskID string) *AgentContextBuilder {
	if taskID != "" {
		b.ctx = context.WithValue(b.ctx, taskIDContextKey, taskID)
	}
	return b
}

// WithSessionID adds sessionID to the context
func (b *AgentContextBuilder) WithSessionID(sessionID string) *AgentContextBuilder {
	if sessionID != "" {
		b.ctx = context.WithValue(b.ctx, protocol.SessionIDKey, sessionID)
	}
	return b
}

// Build returns the final context with all values set
func (b *AgentContextBuilder) Build() context.Context {
	return b.ctx
}

// EnsureAgentContext ensures a context has required agent values.
// If values are missing, it adds them from the provided parameters.
// This helps maintain consistency across the codebase.
func EnsureAgentContext(ctx context.Context, taskID, sessionID string) context.Context {
	// Check if values already exist
	hasTaskID := ctx.Value(taskIDContextKey) != nil
	hasSessionID := ctx.Value(protocol.SessionIDKey) != nil

	// Only add missing values
	if !hasTaskID && taskID != "" {
		ctx = context.WithValue(ctx, taskIDContextKey, taskID)
	}
	if !hasSessionID && sessionID != "" {
		ctx = context.WithValue(ctx, protocol.SessionIDKey, sessionID)
	}

	return ctx
}

