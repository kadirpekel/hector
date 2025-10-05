// Package a2a implements the Agent-to-Agent (A2A) protocol
// Specification: https://a2a-protocol.org/
package a2a

import (
	"time"
)

// ============================================================================
// AGENT CARD - Agent Discovery & Capability Advertisement
// ============================================================================

// AgentCard represents an A2A agent's capabilities and metadata
// This is the agent's "business card" that other agents discover
type AgentCard struct {
	AgentID      string            `json:"agentId"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Version      string            `json:"version,omitempty"`
	Capabilities []string          `json:"capabilities"`
	Endpoints    AgentEndpoints    `json:"endpoints"`
	InputTypes   []string          `json:"inputTypes"`
	OutputTypes  []string          `json:"outputTypes"`
	Auth         AuthConfig        `json:"auth,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// AgentEndpoints defines the URLs for agent interaction
type AgentEndpoints struct {
	Task   string `json:"task"`             // POST endpoint for task execution
	Stream string `json:"stream,omitempty"` // WebSocket endpoint for streaming
	Status string `json:"status,omitempty"` // GET endpoint for task status
}

// AuthConfig describes authentication requirements
type AuthConfig struct {
	Type     string   `json:"type"`               // "bearer", "apiKey", "oauth2", "mtls"
	Schemes  []string `json:"schemes,omitempty"`  // Supported auth schemes
	TokenURL string   `json:"tokenUrl,omitempty"` // OAuth2 token URL
}

// ============================================================================
// TASK - Unit of Work in A2A Protocol
// ============================================================================

// TaskRequest represents a request to execute a task
type TaskRequest struct {
	TaskID     string                 `json:"taskId"`
	Input      TaskInput              `json:"input"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Context    *TaskContext           `json:"context,omitempty"`
}

// TaskInput represents the input data for a task
type TaskInput struct {
	Type    string      `json:"type"`    // MIME type: "text/plain", "application/json", etc.
	Content interface{} `json:"content"` // Actual input content
}

// TaskContext provides additional context for task execution
type TaskContext struct {
	SessionID      string            `json:"sessionId,omitempty"`
	ConversationID string            `json:"conversationId,omitempty"`
	UserID         string            `json:"userId,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// TaskResponse represents the response from a task execution
type TaskResponse struct {
	TaskID    string                 `json:"taskId"`
	Status    TaskStatus             `json:"status"`
	Output    *TaskOutput            `json:"output,omitempty"`
	Error     *TaskError             `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	StartedAt time.Time              `json:"startedAt,omitempty"`
	EndedAt   time.Time              `json:"endedAt,omitempty"`
}

// TaskOutput represents the output from a task
type TaskOutput struct {
	Type    string      `json:"type"`    // MIME type of output
	Content interface{} `json:"content"` // Actual output content
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskError represents an error during task execution
type TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ============================================================================
// STREAMING - Real-time Task Output
// ============================================================================

// StreamChunk represents a chunk of streaming output
type StreamChunk struct {
	TaskID    string      `json:"taskId"`
	ChunkType ChunkType   `json:"chunkType"`
	Content   interface{} `json:"content"`
	Timestamp time.Time   `json:"timestamp"`
	Final     bool        `json:"final"` // Indicates last chunk
}

// ChunkType represents the type of streaming chunk
type ChunkType string

const (
	ChunkTypeText     ChunkType = "text"
	ChunkTypeData     ChunkType = "data"
	ChunkTypeError    ChunkType = "error"
	ChunkTypeMetadata ChunkType = "metadata"
	ChunkTypeDone     ChunkType = "done"
)

// ============================================================================
// SESSION - Conversation State Management
// ============================================================================

// Session represents a conversation session with an agent
type Session struct {
	SessionID      string                 `json:"sessionId"`
	AgentID        string                 `json:"agentId"`
	CreatedAt      time.Time              `json:"createdAt"`
	LastActivityAt time.Time              `json:"lastActivityAt"`
	State          map[string]interface{} `json:"state,omitempty"`
	Metadata       map[string]string      `json:"metadata,omitempty"`
}

// SessionRequest represents a request to create or manage a session
type SessionRequest struct {
	AgentID  string            `json:"agentId"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ============================================================================
// AGENT DIRECTORY - Discovery Service
// ============================================================================

// AgentDirectory represents a collection of available agents
type AgentDirectory struct {
	Agents []AgentCard `json:"agents"`
	Total  int         `json:"total"`
}

// AgentQuery represents query parameters for agent discovery
type AgentQuery struct {
	Capabilities []string `json:"capabilities,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Search       string   `json:"search,omitempty"`
}

// ============================================================================
// ORCHESTRATION - Multi-Agent Coordination
// ============================================================================

// OrchestrationStep represents a step in a multi-agent workflow
type OrchestrationStep struct {
	Name       string                 `json:"name"`
	AgentID    string                 `json:"agentId"`
	Input      string                 `json:"input"` // Template with variables
	DependsOn  []string               `json:"dependsOn,omitempty"`
	OutputVar  string                 `json:"outputVar,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Retry      *RetryConfig           `json:"retry,omitempty"`
}

// RetryConfig defines retry behavior for tasks
type RetryConfig struct {
	MaxAttempts int           `json:"maxAttempts"`
	BackoffType string        `json:"backoffType"` // "fixed", "exponential"
	InitialWait time.Duration `json:"initialWait"`
}

// OrchestrationResult represents the result of an orchestrated workflow
type OrchestrationResult struct {
	WorkflowID  string                   `json:"workflowId"`
	Status      TaskStatus               `json:"status"`
	StepResults map[string]*TaskResponse `json:"stepResults"`
	FinalOutput *TaskOutput              `json:"finalOutput,omitempty"`
	Variables   map[string]interface{}   `json:"variables"`
	StartedAt   time.Time                `json:"startedAt"`
	CompletedAt time.Time                `json:"completedAt,omitempty"`
}
