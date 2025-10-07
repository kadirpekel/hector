// Package a2a implements the Agent-to-Agent (A2A) Protocol
// Specification: https://a2a-protocol.org/latest/specification/
// This is a genuine implementation of A2A HTTP+JSON transport (Section 3.2.3)
package a2a

import (
	"context"
	"time"
)

// ============================================================================
// A2A PROTOCOL VERSION
// Spec Section 1
// ============================================================================

const (
	ProtocolVersion = "1.0" // A2A Protocol version
)

// ============================================================================
// CORE AGENT INTERFACE
// Agents implement this interface to be A2A-compliant
// ============================================================================

// Agent represents any A2A-compliant agent
type Agent interface {
	// GetAgentCard returns the agent's capability card for discovery (Section 5)
	GetAgentCard() *AgentCard

	// ExecuteTask executes a task and returns the complete response
	ExecuteTask(ctx context.Context, task *Task) (*Task, error)

	// ExecuteTaskStreaming executes a task with real-time streaming output
	ExecuteTaskStreaming(ctx context.Context, task *Task) (<-chan StreamEvent, error)
}

// ============================================================================
// AGENT CARD - Agent Discovery & Capability Advertisement
// Spec Section 5.5: AgentCard Object Structure
// ============================================================================

// AgentCard represents an A2A agent's capabilities and metadata
type AgentCard struct {
	// Core identity
	Name        string `json:"name"`        // Human-readable agent name
	URL         string `json:"url"`         // Base URL where agent is accessible
	Version     string `json:"version"`     // Agent version
	Description string `json:"description"` // What the agent does

	// Provider information (optional)
	Provider *AgentProvider `json:"provider,omitempty"`

	// Transport configuration
	PreferredTransport   string           `json:"preferredTransport"` // "http+json", "json-rpc", "grpc"
	AdditionalInterfaces []AgentInterface `json:"additionalInterfaces,omitempty"`

	// Capabilities
	Capabilities AgentCapabilities `json:"capabilities"`

	// Skills (optional)
	Skills []AgentSkill `json:"skills,omitempty"`

	// Security schemes (optional)
	SecuritySchemes []SecurityScheme `json:"securitySchemes,omitempty"`
}

// AgentProvider describes the provider of an agent (Section 5.5.1)
type AgentProvider struct {
	Name         string `json:"name"`
	Organization string `json:"organization,omitempty"`
	URL          string `json:"url,omitempty"`
}

// AgentInterface defines an additional transport interface (Section 5.5.5)
type AgentInterface struct {
	Transport string `json:"transport"` // "http+json", "json-rpc", "grpc"
	URL       string `json:"url"`       // URL for this transport
}

// AgentCapabilities describes what an agent can do (Section 5.5.2)
type AgentCapabilities struct {
	Streaming         bool             `json:"streaming"`            // Supports real-time streaming
	MultiTurn         bool             `json:"multiTurn"`            // Supports multi-turn conversations
	PushNotifications bool             `json:"pushNotifications"`    // Supports push notifications
	Extensions        []AgentExtension `json:"extensions,omitempty"` // Custom extensions
}

// AgentExtension represents a custom capability extension (Section 5.5.2.1)
type AgentExtension struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version,omitempty"`
}

// AgentSkill describes a specific skill the agent possesses (Section 5.5.4)
type AgentSkill struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// SecurityScheme describes authentication requirements (Section 5.5.3)
type SecurityScheme struct {
	Type   string `json:"type"`           // "bearer", "apiKey", "oauth2"
	Scheme string `json:"scheme"`         // "Bearer", etc.
	In     string `json:"in,omitempty"`   // "header", "query"
	Name   string `json:"name,omitempty"` // Header/query param name
}

// ============================================================================
// TASK - Unit of Work in A2A Protocol
// Spec Section 6.1: Task Object
// ============================================================================

// Task represents a unit of work in the A2A protocol
type Task struct {
	ID        string       `json:"id"`                  // Unique task identifier
	Status    TaskStatus   `json:"status"`              // Current task status
	Messages  []Message    `json:"messages"`            // Conversation messages
	Artifacts []Artifact   `json:"artifacts,omitempty"` // Output artifacts
	Error     *TaskError   `json:"error,omitempty"`     // Error if failed
	Metadata  TaskMetadata `json:"metadata,omitempty"`  // Additional metadata
}

// TaskStatus represents the status of a task (Section 6.2)
type TaskStatus struct {
	State     TaskState `json:"state"`            // Current state
	CreatedAt time.Time `json:"createdAt"`        // When task was created
	UpdatedAt time.Time `json:"updatedAt"`        // Last update time
	Reason    string    `json:"reason,omitempty"` // Additional status info
}

// TaskState represents the state of a task (Section 6.3)
type TaskState string

const (
	TaskStateSubmitted     TaskState = "submitted"
	TaskStateWorking       TaskState = "working"
	TaskStateInputRequired TaskState = "input_required"
	TaskStateCompleted     TaskState = "completed"
	TaskStateFailed        TaskState = "failed"
	TaskStateCanceled      TaskState = "canceled"
)

// TaskMetadata contains additional task information
type TaskMetadata map[string]interface{}

// TaskError represents an error during task execution
type TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ============================================================================
// MESSAGE - Conversation Messages
// Spec Section 6.4: Message Object
// ============================================================================

// Message represents a message in a conversation
type Message struct {
	Role  MessageRole `json:"role"`  // "user" or "assistant"
	Parts []Part      `json:"parts"` // Message content parts
}

// MessageRole represents the role of a message sender
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
)

// ============================================================================
// PART - Message Content Parts
// Spec Section 6.5: Part Union Type
// ============================================================================

// Part represents a part of a message (union type)
type Part struct {
	// Type discriminator
	Type PartType `json:"type"` // "text", "file", "data"

	// Text part (Section 6.5.1)
	Text string `json:"text,omitempty"`

	// File part (Section 6.5.2)
	File *FilePart `json:"file,omitempty"`

	// Data part (Section 6.5.3)
	Data     interface{} `json:"data,omitempty"`
	DataType string      `json:"dataType,omitempty"` // MIME type for data
}

// PartType represents the type of message part
type PartType string

const (
	PartTypeText PartType = "text"
	PartTypeFile PartType = "file"
	PartTypeData PartType = "data"
)

// FilePart represents a file in a message (Section 6.6)
type FilePart struct {
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`

	// Either bytes or URI (FileWithBytes or FileWithUri)
	Bytes []byte `json:"bytes,omitempty"` // Inline file content
	URI   string `json:"uri,omitempty"`   // Reference to file
	Size  int64  `json:"size,omitempty"`  // File size in bytes
}

// ============================================================================
// ARTIFACT - Task Output Artifacts
// Spec Section 6.7: Artifact Object
// ============================================================================

// Artifact represents an output artifact from a task
type Artifact struct {
	ID          string      `json:"id"`                    // Unique artifact identifier
	Name        string      `json:"name"`                  // Human-readable name
	Description string      `json:"description,omitempty"` // What the artifact is
	Parts       []Part      `json:"parts"`                 // Artifact content
	Metadata    interface{} `json:"metadata,omitempty"`    // Additional metadata
}

// ============================================================================
// STREAMING - Real-time Task Updates
// ============================================================================

// StreamEvent represents a streaming event
type StreamEvent struct {
	Type      StreamEventType `json:"type"`
	TaskID    string          `json:"taskId"`
	Message   *Message        `json:"message,omitempty"`
	Artifact  *Artifact       `json:"artifact,omitempty"`
	Status    *TaskStatus     `json:"status,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// StreamEventType represents the type of streaming event
type StreamEventType string

const (
	StreamEventTypeMessage  StreamEventType = "message"
	StreamEventTypeArtifact StreamEventType = "artifact"
	StreamEventTypeStatus   StreamEventType = "status"
)

// ============================================================================
// RPC METHOD PARAMETERS
// Spec Section 7: Protocol RPC Methods
// ============================================================================

// MessageSendParams represents parameters for message/send (Section 7.1.1)
type MessageSendParams struct {
	Message       Message               `json:"message"`
	TaskID        string                `json:"taskId,omitempty"` // Continue existing task
	Configuration *MessageConfiguration `json:"configuration,omitempty"`
}

// MessageConfiguration provides execution configuration (Section 7.1.2)
type MessageConfiguration struct {
	Temperature    *float64               `json:"temperature,omitempty"`
	MaxTokens      *int                   `json:"maxTokens,omitempty"`
	TopP           *float64               `json:"topP,omitempty"`
	StopSequences  []string               `json:"stopSequences,omitempty"`
	CustomSettings map[string]interface{} `json:"customSettings,omitempty"`
}

// TaskQueryParams represents parameters for tasks/get (Section 7.3.1)
type TaskQueryParams struct {
	TaskID string `json:"taskId"`
}

// TaskCancelParams represents parameters for tasks/cancel (Section 7.4.1)
type TaskCancelParams struct {
	TaskID string `json:"taskId"`
	Reason string `json:"reason,omitempty"`
}

// ============================================================================
// SSE STREAMING - Server-Sent Events (Spec Section 7.2)
// ============================================================================

// SendStreamingMessageResponse represents an SSE event (Section 7.2.1)
type SendStreamingMessageResponse struct {
	Event string      `json:"event"` // "message", "status", or "artifact"
	Data  interface{} `json:"data"`  // The actual event data
}

// TaskStatusUpdateEvent represents a status update event (Section 7.2.2)
type TaskStatusUpdateEvent struct {
	TaskID string     `json:"taskId"`
	Status TaskStatus `json:"status"`
}

// TaskArtifactUpdateEvent represents an artifact update event (Section 7.2.3)
type TaskArtifactUpdateEvent struct {
	TaskID   string   `json:"taskId"`
	Artifact Artifact `json:"artifact"`
}

// TaskMessageEvent represents a message event in SSE streaming
type TaskMessageEvent struct {
	TaskID  string  `json:"taskId"`
	Message Message `json:"message"`
}

// TaskResubscribeParams represents parameters for tasks/resubscribe (Section 7.9)
type TaskResubscribeParams struct {
	LastEventID string `json:"lastEventId,omitempty"` // Resume from this event
}

// ============================================================================
// SESSION - Multi-turn Conversation State
// Hector extension (not in core A2A spec, but compatible)
// ============================================================================

// Session represents a conversation session
type Session struct {
	ID             string                 `json:"id"`
	AgentName      string                 `json:"agentName"`
	Tasks          []string               `json:"tasks"` // Task IDs in this session
	CreatedAt      time.Time              `json:"createdAt"`
	LastActivityAt time.Time              `json:"lastActivityAt"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ============================================================================
// AGENT DIRECTORY - Discovery
// ============================================================================

// AgentDirectory represents a collection of available agents
type AgentDirectory struct {
	Agents []AgentCard `json:"agents"`
	Total  int         `json:"total"`
}
