package agent

import "time"

// ============================================================================
// ORCHESTRATION - Multi-Agent Coordination (Hector-specific)
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
	WorkflowID  string                 `json:"workflowId"`
	Status      string                 `json:"status"` // Use string instead of a2a.TaskStatus
	StepResults map[string]interface{} `json:"stepResults"`
	FinalOutput interface{}            `json:"finalOutput,omitempty"`
	Variables   map[string]interface{} `json:"variables"`
	StartedAt   time.Time              `json:"startedAt"`
	CompletedAt time.Time              `json:"completedAt,omitempty"`
}
