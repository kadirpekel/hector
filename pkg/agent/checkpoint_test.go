package agent

import (
	"testing"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
)

func TestExecutionPhaseConstants(t *testing.T) {
	tests := []struct {
		name     string
		phase    ExecutionPhase
		expected string
	}{
		{"Initialized", PhaseInitialized, "initialized"},
		{"PreLLM", PhasePreLLM, "pre_llm"},
		{"PostLLM", PhasePostLLM, "post_llm"},
		{"ToolExecution", PhaseToolExecution, "tool_execution"},
		{"PostTool", PhasePostTool, "post_tool"},
		{"IterationEnd", PhaseIterationEnd, "iteration_end"},
		{"ToolApproval", PhaseToolApproval, "tool_approval"},
		{"Error", PhaseError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.phase) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.phase))
			}
		})
	}
}

func TestCheckpointTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		cpType   CheckpointType
		expected string
	}{
		{"Event", CheckpointTypeEvent, "event"},
		{"Interval", CheckpointTypeInterval, "interval"},
		{"Manual", CheckpointTypeManual, "manual"},
		{"Error", CheckpointTypeError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.cpType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.cpType))
			}
		})
	}
}

func TestExecutionStateWithCheckpointMetadata(t *testing.T) {
	// Test that ExecutionState can be serialized with checkpoint metadata
	execState := &ExecutionState{
		TaskID:          "task-123",
		ContextID:       "ctx-456",
		Query:           "test query",
		ReasoningState:  &ReasoningStateSnapshot{Iteration: 1},
		PendingToolCall: nil,
		Phase:           PhaseIterationEnd,
		CheckpointType:  CheckpointTypeInterval,
		CheckpointTime:  time.Now(),
	}

	// Serialize
	data, err := SerializeExecutionState(execState)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Deserialize
	restored, err := DeserializeExecutionState(data)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// Verify checkpoint metadata
	if restored.Phase != PhaseIterationEnd {
		t.Errorf("Expected phase %s, got %s", PhaseIterationEnd, restored.Phase)
	}
	if restored.CheckpointType != CheckpointTypeInterval {
		t.Errorf("Expected checkpoint type %s, got %s", CheckpointTypeInterval, restored.CheckpointType)
	}
	if restored.CheckpointTime.IsZero() {
		t.Error("CheckpointTime should not be zero")
	}
}

func TestExecutionStateBackwardCompatibility(t *testing.T) {
	// Test that ExecutionState without checkpoint metadata still works
	execState := &ExecutionState{
		TaskID:          "task-123",
		ContextID:       "ctx-456",
		Query:           "test query",
		ReasoningState:  &ReasoningStateSnapshot{Iteration: 1},
		PendingToolCall: nil,
		// No Phase, CheckpointType, or CheckpointTime (backward compatibility)
	}

	// Serialize
	data, err := SerializeExecutionState(execState)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Deserialize
	restored, err := DeserializeExecutionState(data)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// Verify backward compatibility - empty phase/type should be OK
	if restored.Phase != "" {
		t.Errorf("Expected empty phase for backward compatibility, got %s", restored.Phase)
	}
	if restored.CheckpointType != "" {
		t.Errorf("Expected empty checkpoint type for backward compatibility, got %s", restored.CheckpointType)
	}
	if !restored.CheckpointTime.IsZero() {
		t.Error("CheckpointTime should be zero for backward compatibility")
	}
}

func TestShouldCheckpointInterval(t *testing.T) {
	tests := []struct {
		name           string
		iteration      int
		intervalEveryN int
		expected       bool
	}{
		{"Disabled (0)", 5, 0, false},
		{"Disabled (negative)", 5, -1, false},
		{"First iteration", 1, 5, false}, // First iteration never checkpoints
		{"Not divisible", 3, 5, false},
		{"Divisible (5)", 5, 5, true},
		{"Divisible (10)", 10, 5, true},
		{"Divisible (15)", 15, 5, true},
		{"Zero iteration", 0, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal agent for testing
			agent := &Agent{}
			result := agent.shouldCheckpointInterval(tt.iteration, tt.intervalEveryN)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetCheckpointInterval(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.AgentConfig
		expected    int
		description string
	}{
		{
			name:        "No config",
			config:      nil,
			expected:    0,
			description: "Should return 0 when no config",
		},
		{
			name: "Checkpoint disabled",
			config: &config.AgentConfig{
				Task: &config.TaskConfig{
					EnableCheckpointing: config.BoolPtr(false),
				},
			},
			expected:    0,
			description: "Should return 0 when checkpoint disabled",
		},
		{
			name: "Event strategy",
			config: &config.AgentConfig{
				Task: &config.TaskConfig{
					EnableCheckpointing: config.BoolPtr(true),
					CheckpointStrategy:  "event",
					CheckpointInterval:  5,
				},
			},
			expected:    0,
			description: "Should return 0 for event strategy",
		},
		{
			name: "Interval strategy",
			config: &config.AgentConfig{
				Task: &config.TaskConfig{
					EnableCheckpointing: config.BoolPtr(true),
					CheckpointStrategy:  "interval",
					CheckpointInterval:  5,
				},
			},
			expected:    5,
			description: "Should return interval for interval strategy",
		},
		{
			name: "Hybrid strategy",
			config: &config.AgentConfig{
				Task: &config.TaskConfig{
					EnableCheckpointing: config.BoolPtr(true),
					CheckpointStrategy:  "hybrid",
					CheckpointInterval:  10,
				},
			},
			expected:    10,
			description: "Should return interval for hybrid strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{
				config: tt.config,
			}
			result := agent.getCheckpointInterval()
			if result != tt.expected {
				t.Errorf("%s: Expected %d, got %d", tt.description, tt.expected, result)
			}
		})
	}
}

func TestIsCheckpointEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.AgentConfig
		expected bool
	}{
		{"No config", nil, false},
		{"No task config", &config.AgentConfig{}, false},
		{"No checkpoint config", &config.AgentConfig{Task: &config.TaskConfig{}}, false},
		{"Checkpoint disabled", &config.AgentConfig{
			Task: &config.TaskConfig{
				EnableCheckpointing: config.BoolPtr(false),
			},
		}, false},
		{"Checkpoint enabled", &config.AgentConfig{
			Task: &config.TaskConfig{
				EnableCheckpointing: config.BoolPtr(true),
			},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{config: tt.config}
			result := agent.isCheckpointEnabled()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsRecoveryEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.AgentConfig
		expected bool
	}{
		{"No config", nil, false},
		{"No task config", &config.AgentConfig{}, false},
		{"No checkpoint config", &config.AgentConfig{Task: &config.TaskConfig{}}, false},
		{"Checkpoint disabled", &config.AgentConfig{
			Task: &config.TaskConfig{
				EnableCheckpointing: config.BoolPtr(false),
			},
		}, false},
		{"No recovery config", &config.AgentConfig{
			Task: &config.TaskConfig{
				EnableCheckpointing: config.BoolPtr(true),
			},
		}, false},
		{"Recovery disabled", &config.AgentConfig{
			Task: &config.TaskConfig{
				EnableCheckpointing: config.BoolPtr(true),
				AutoResume:          config.BoolPtr(false),
			},
		}, false},
		{"Recovery enabled", &config.AgentConfig{
			Task: &config.TaskConfig{
				EnableCheckpointing: config.BoolPtr(true),
				AutoResume:          config.BoolPtr(true),
			},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{config: tt.config}
			result := agent.isRecoveryEnabled()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsCheckpointExpired(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name           string
		checkpointTime time.Time
		timeout        int
		expected       bool
	}{
		{"No timestamp (backward compat)", time.Time{}, 3600, false},
		{"No timeout configured", now.Add(-2 * time.Hour), 0, false},
		{"Not expired (1 hour old, 2 hour timeout)", now.Add(-1 * time.Hour), 7200, false},
		{"Expired (2 hours old, 1 hour timeout)", now.Add(-2 * time.Hour), 3600, true},
		{"Just expired", now.Add(-3601 * time.Second), 3600, true},
		{"Not expired (just under)", now.Add(-3599 * time.Second), 3600, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execState := &ExecutionState{
				CheckpointTime: tt.checkpointTime,
			}
			agent := &Agent{
				config: &config.AgentConfig{
					Task: &config.TaskConfig{
						ResumeTimeout: tt.timeout,
					},
				},
			}
			result := agent.isCheckpointExpired(execState)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetRecoveryTimeout(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.AgentConfig
		expected int
	}{
		{"No config", nil, 0},
		{"No task config", &config.AgentConfig{}, 0},
		{"No checkpoint config", &config.AgentConfig{Task: &config.TaskConfig{}}, 0},
		{"No recovery config", &config.AgentConfig{
			Task: &config.TaskConfig{
				EnableCheckpointing: config.BoolPtr(true),
			},
		}, 0},
		{"Default timeout", &config.AgentConfig{
			Task: &config.TaskConfig{
				ResumeTimeout: 0,
			},
		}, 0},
		{"Custom timeout", &config.AgentConfig{
			Task: &config.TaskConfig{
				ResumeTimeout: 7200,
			},
		}, 7200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{config: tt.config}
			result := agent.getRecoveryTimeout()
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestShouldAutoResumeHITL(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.AgentConfig
		expected bool
	}{
		{"No config", nil, false},
		{"No task config", &config.AgentConfig{}, false},
		{"No checkpoint config", &config.AgentConfig{Task: &config.TaskConfig{}}, false},
		{"No recovery config", &config.AgentConfig{
			Task: &config.TaskConfig{
				EnableCheckpointing: config.BoolPtr(true),
			},
		}, false},
		{"AutoResumeHITL disabled", &config.AgentConfig{
			Task: &config.TaskConfig{
				AutoResumeHITL: config.BoolPtr(false),
			},
		}, false},
		{"AutoResumeHITL enabled", &config.AgentConfig{
			Task: &config.TaskConfig{
				AutoResumeHITL: config.BoolPtr(true),
			},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{config: tt.config}
			result := agent.shouldAutoResumeHITL()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
