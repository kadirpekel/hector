package workflow

import (
	"context"
	"time"

	"github.com/kadirpekel/hector/config"
)

// ============================================================================
// WORKFLOW EXECUTION STATUS TYPES
// ============================================================================

// WorkflowStatus represents the current state of workflow execution
type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)

// ExecutionStatus represents internal execution status
type ExecutionStatus string

const (
	StatusInitializing ExecutionStatus = "initializing"
	StatusPlanning     ExecutionStatus = "planning"
	StatusExecuting    ExecutionStatus = "executing"
	StatusCompleted    ExecutionStatus = "completed"
	StatusFailed       ExecutionStatus = "failed"
	StatusCancelled    ExecutionStatus = "cancelled"
	StatusRetrying     ExecutionStatus = "retrying"
)

// StepStatus represents the execution status of a workflow step
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusReady     StepStatus = "ready"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// ============================================================================
// WORKFLOW REQUEST AND CONTEXT TYPES
// ============================================================================

// WorkflowRequest contains all information needed to execute a workflow
type WorkflowRequest struct {
	Workflow      *config.WorkflowConfig
	AgentServices AgentServices // Abstract agent services - NO CONCRETE TYPES!
	Input         string
	Context       WorkflowContext
}

// AgentServices provides abstract access to agent capabilities for workflow executors
type AgentServices interface {
	// ExecuteAgent executes an agent with the given input and returns the result
	ExecuteAgent(ctx context.Context, agentName string, input string) (*AgentResult, error)

	// GetAvailableAgents returns the list of available agent names
	GetAvailableAgents() []string

	// GetAgentCapabilities returns the capabilities of a specific agent
	GetAgentCapabilities(agentName string) ([]string, error)

	// IsAgentAvailable checks if an agent is available for execution
	IsAgentAvailable(agentName string) bool
}

// WorkflowContext holds typed context data (no interface{})
type WorkflowContext struct {
	Variables map[string]string   // workflow variables
	Metadata  map[string]string   // metadata
	Artifacts map[string]Artifact // artifacts
}

// Artifact represents a workflow artifact
type Artifact struct {
	Type     string
	Content  []byte
	MimeType string
	Size     int64
}

// ============================================================================
// WORKFLOW RESULT TYPES
// ============================================================================

// WorkflowResult contains the complete workflow execution result
type WorkflowResult struct {
	WorkflowName  string                  `json:"workflow_name"`
	Status        WorkflowStatus          `json:"status"`
	Success       bool                    `json:"success"`
	Error         string                  `json:"error,omitempty"`
	Results       map[string]*AgentResult `json:"results"`
	FinalOutput   string                  `json:"final_output"`
	ExecutionTime time.Duration           `json:"execution_time"`
	TotalTokens   int                     `json:"total_tokens"`
	SharedContext WorkflowContext         `json:"shared_context"`
	StepsExecuted int                     `json:"steps_executed"`
	AgentsUsed    []string                `json:"agents_used"`
	Metadata      map[string]string       `json:"metadata,omitempty"`
}

// AgentResult represents individual agent execution result
type AgentResult struct {
	AgentName  string              `json:"agent_name"`
	StepName   string              `json:"step_name"`
	Result     string              `json:"result"`
	Success    bool                `json:"success"`
	Error      string              `json:"error,omitempty"`
	Duration   time.Duration       `json:"duration"`
	TokensUsed int                 `json:"tokens_used"`
	Artifacts  map[string]Artifact `json:"artifacts,omitempty"`
	Metadata   map[string]string   `json:"metadata,omitempty"`
	Timestamp  time.Time           `json:"timestamp"`
	Confidence float64             `json:"confidence"`
}
