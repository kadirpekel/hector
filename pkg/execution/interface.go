package execution

import (
	"context"
)

// Request represents a request to execute an agent task.
type Request struct {
	// ID is the unique identifier for this execution request.
	// This usually maps to the Task ID.
	ID string

	// AppID identifies the tenant/app.
	AppID string

	// AgentName is the name of the agent to invoke.
	AgentName string

	// Input is the user input or prompt.
	Input string

	// ContextID is the session/conversation ID.
	ContextID string

	// UserID is the user triggering the task.
	UserID string

	// Metadata contains arbitrary execution metadata.
	Metadata map[string]any

	// Priority affects execution order (higher = sooner).
	Priority int
}

// Provider defines the interface for task execution backends.
// Implementations can be local (Native) or remote (Temporal, etc).
type Provider interface {
	// Submit enqueues a task for execution.
	Submit(ctx context.Context, req *Request) error

	// Start starts the execution consumer (worker/poller).
	Start(ctx context.Context) error

	// Stop stops the execution consumer.
	Stop() error
}
