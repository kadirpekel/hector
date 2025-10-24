package agent

import "time"

type OrchestrationStep struct {
	Name       string                 `json:"name"`
	AgentID    string                 `json:"agentId"`
	Input      string                 `json:"input"`
	DependsOn  []string               `json:"dependsOn,omitempty"`
	OutputVar  string                 `json:"outputVar,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Retry      *RetryConfig           `json:"retry,omitempty"`
}

type RetryConfig struct {
	MaxAttempts int           `json:"maxAttempts"`
	BackoffType string        `json:"backoffType"`
	InitialWait time.Duration `json:"initialWait"`
}

type OrchestrationResult struct {
	WorkflowID  string                 `json:"workflowId"`
	Status      string                 `json:"status"`
	StepResults map[string]interface{} `json:"stepResults"`
	FinalOutput interface{}            `json:"finalOutput,omitempty"`
	Variables   map[string]interface{} `json:"variables"`
	StartedAt   time.Time              `json:"startedAt"`
	CompletedAt time.Time              `json:"completedAt,omitempty"`
}
