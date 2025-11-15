package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// TaskServiceBuilder provides a fluent API for building task services
type TaskServiceBuilder struct {
	backend          string
	workerPool       int
	database         string // Reference to SQL database from databases section
	inputTimeout     int
	timeout          int
	hitlConfig       *config.HITLConfig
	checkpointConfig *config.CheckpointConfig
	componentManager *component.ComponentManager // For getting SQL database connections
}

// NewTaskService creates a new task service builder
func NewTaskService() *TaskServiceBuilder {
	return &TaskServiceBuilder{
		backend:      "memory",
		workerPool:   100,
		inputTimeout: 600,  // 10 minutes default
		timeout:      3600, // 1 hour default
	}
}

// Backend sets the task backend ("memory" or "sql")
func (b *TaskServiceBuilder) Backend(backend string) *TaskServiceBuilder {
	if backend != "memory" && backend != "sql" {
		panic(fmt.Sprintf("invalid backend: %s (must be 'memory' or 'sql')", backend))
	}
	b.backend = backend
	return b
}

// WorkerPool sets the worker pool size
func (b *TaskServiceBuilder) WorkerPool(size int) *TaskServiceBuilder {
	if size < 0 {
		panic("worker pool size must be non-negative")
	}
	b.workerPool = size
	return b
}

// Database sets the SQL database reference for SQL backend
func (b *TaskServiceBuilder) Database(dbName string) *TaskServiceBuilder {
	b.database = dbName
	return b
}

// WithComponentManager sets the component manager for getting SQL database connections
func (b *TaskServiceBuilder) WithComponentManager(cm *component.ComponentManager) *TaskServiceBuilder {
	b.componentManager = cm
	return b
}

// InputTimeout sets the timeout for INPUT_REQUIRED state (seconds)
func (b *TaskServiceBuilder) InputTimeout(seconds int) *TaskServiceBuilder {
	if seconds < 0 {
		panic("input timeout must be non-negative")
	}
	b.inputTimeout = seconds
	return b
}

// Timeout sets the timeout for async task execution (seconds)
func (b *TaskServiceBuilder) Timeout(seconds int) *TaskServiceBuilder {
	if seconds < 0 {
		panic("timeout must be non-negative")
	}
	b.timeout = seconds
	return b
}

// WithHITL sets the HITL configuration
func (b *TaskServiceBuilder) WithHITL(cfg *config.HITLConfig) *TaskServiceBuilder {
	b.hitlConfig = cfg
	return b
}

// HITL creates an HITL config builder
func (b *TaskServiceBuilder) HITL() *HITLConfigBuilder {
	if b.hitlConfig == nil {
		b.hitlConfig = &config.HITLConfig{}
	}
	return NewHITLConfigBuilder(b.hitlConfig)
}

// WithCheckpoint sets the checkpoint configuration
func (b *TaskServiceBuilder) WithCheckpoint(cfg *config.CheckpointConfig) *TaskServiceBuilder {
	b.checkpointConfig = cfg
	return b
}

// Checkpoint creates a checkpoint config builder
func (b *TaskServiceBuilder) Checkpoint() *CheckpointConfigBuilder {
	if b.checkpointConfig == nil {
		b.checkpointConfig = &config.CheckpointConfig{}
	}
	return NewCheckpointConfigBuilder(b.checkpointConfig)
}

// Build creates the task service
func (b *TaskServiceBuilder) Build() (reasoning.TaskService, error) {
	switch b.backend {
	case "memory":
		return agent.NewInMemoryTaskService(), nil

	case "sql":
		if b.database == "" {
			return nil, fmt.Errorf("SQL backend requires database reference (use Database() method)")
		}
		if b.componentManager == nil {
			return nil, fmt.Errorf("component manager is required when using database reference")
		}

		db, driver, err := b.componentManager.GetSQLDatabase(b.database)
		if err != nil {
			return nil, fmt.Errorf("failed to get SQL database '%s': %w", b.database, err)
		}

		return agent.NewSQLTaskService(db, driver)

	default:
		return nil, fmt.Errorf("unsupported backend: %s", b.backend)
	}
}

// GetConfig returns the task configuration
func (b *TaskServiceBuilder) GetConfig() TaskConfig {
	cfg := TaskConfig{
		Backend:      b.backend,
		WorkerPool:   b.workerPool,
		InputTimeout: b.inputTimeout,
		Timeout:      b.timeout,
		HITLConfig:   b.hitlConfig,
	}

	// Convert CheckpointConfig to flattened fields
	if b.checkpointConfig != nil {
		cfg.EnableCheckpointing = b.checkpointConfig.Enabled
		cfg.CheckpointStrategy = b.checkpointConfig.Strategy
		if b.checkpointConfig.Interval != nil {
			cfg.CheckpointInterval = b.checkpointConfig.Interval.EveryNIterations
			cfg.CheckpointAfterTools = b.checkpointConfig.Interval.AfterToolCalls
			cfg.CheckpointBeforeLLM = b.checkpointConfig.Interval.BeforeLLMCalls
		}
		if b.checkpointConfig.Recovery != nil {
			cfg.AutoResume = b.checkpointConfig.Recovery.AutoResume
			cfg.AutoResumeHITL = b.checkpointConfig.Recovery.AutoResumeHITL
			cfg.ResumeTimeout = b.checkpointConfig.Recovery.ResumeTimeout
		}
	}

	return cfg
}

// GetTaskConfig returns the config.TaskConfig (for agent config)
func (b *TaskServiceBuilder) GetTaskConfig() *config.TaskConfig {
	if b.backend == "" && b.workerPool == 0 && b.database == "" {
		return nil // Task not enabled
	}
	cfg := &config.TaskConfig{
		Backend:      b.backend,
		WorkerPool:   b.workerPool,
		SQLDatabase:  b.database,
		InputTimeout: b.inputTimeout,
		Timeout:      b.timeout,
		HITL:         b.hitlConfig,
	}

	// Convert CheckpointConfig to flattened fields
	if b.checkpointConfig != nil {
		cfg.EnableCheckpointing = b.checkpointConfig.Enabled
		cfg.CheckpointStrategy = b.checkpointConfig.Strategy
		if b.checkpointConfig.Interval != nil {
			cfg.CheckpointInterval = b.checkpointConfig.Interval.EveryNIterations
			cfg.CheckpointAfterTools = b.checkpointConfig.Interval.AfterToolCalls
			cfg.CheckpointBeforeLLM = b.checkpointConfig.Interval.BeforeLLMCalls
		}
		if b.checkpointConfig.Recovery != nil {
			cfg.AutoResume = b.checkpointConfig.Recovery.AutoResume
			cfg.AutoResumeHITL = b.checkpointConfig.Recovery.AutoResumeHITL
			cfg.ResumeTimeout = b.checkpointConfig.Recovery.ResumeTimeout
		}
	}

	return cfg
}

// TaskConfig represents task configuration
type TaskConfig struct {
	Backend              string
	WorkerPool           int
	InputTimeout         int
	Timeout              int
	HITLConfig           *config.HITLConfig
	EnableCheckpointing  *bool
	CheckpointStrategy   string
	CheckpointInterval   int
	CheckpointAfterTools *bool
	CheckpointBeforeLLM  *bool
	AutoResume           *bool
	AutoResumeHITL       *bool
	ResumeTimeout        int
}

// TaskSQLConfigBuilder provides a fluent API for building SQL task config
