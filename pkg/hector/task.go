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
	database         string                // Reference to SQL database from databases section
	sqlConfig        *config.TaskSQLConfig // Deprecated: use database reference instead
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

// WithSQLConfig sets the SQL configuration for SQL backend (deprecated: use Database instead)
func (b *TaskServiceBuilder) WithSQLConfig(cfg *config.TaskSQLConfig) *TaskServiceBuilder {
	b.sqlConfig = cfg
	return b
}

// SQLConfig creates a SQL config builder
func (b *TaskServiceBuilder) SQLConfig() *TaskSQLConfigBuilder {
	if b.sqlConfig == nil {
		b.sqlConfig = &config.TaskSQLConfig{}
	}
	return NewTaskSQLConfigBuilder(b.sqlConfig)
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
		// Check if database reference is provided (new way)
		if b.database != "" {
			if b.componentManager == nil {
				return nil, fmt.Errorf("component manager is required when using database reference")
			}

			db, driver, err := b.componentManager.GetSQLDatabase(b.database)
			if err != nil {
				return nil, fmt.Errorf("failed to get SQL database '%s': %w", b.database, err)
			}

			return agent.NewSQLTaskService(db, driver)
		} else if b.sqlConfig != nil {
			// Fallback to inline SQL config (deprecated but supported)
			b.sqlConfig.SetDefaults()
			if err := b.sqlConfig.Validate(); err != nil {
				return nil, fmt.Errorf("invalid SQL configuration: %w", err)
			}
			return agent.NewSQLTaskServiceFromConfig(b.sqlConfig)
		} else {
			return nil, fmt.Errorf("SQL backend requires either 'database' reference or 'sql' configuration")
		}

	default:
		return nil, fmt.Errorf("unsupported backend: %s", b.backend)
	}
}

// GetConfig returns the task configuration
func (b *TaskServiceBuilder) GetConfig() TaskConfig {
	return TaskConfig{
		Backend:          b.backend,
		WorkerPool:       b.workerPool,
		SQLConfig:        b.sqlConfig,
		InputTimeout:     b.inputTimeout,
		Timeout:          b.timeout,
		HITLConfig:       b.hitlConfig,
		CheckpointConfig: b.checkpointConfig,
	}
}

// GetTaskConfig returns the config.TaskConfig (for agent config)
func (b *TaskServiceBuilder) GetTaskConfig() *config.TaskConfig {
	if b.backend == "" && b.workerPool == 0 && b.sqlConfig == nil {
		return nil // Task not enabled
	}
	return &config.TaskConfig{
		Backend:      b.backend,
		WorkerPool:   b.workerPool,
		SQL:          b.sqlConfig,
		InputTimeout: b.inputTimeout,
		Timeout:      b.timeout,
		HITL:         b.hitlConfig,
		Checkpoint:   b.checkpointConfig,
	}
}

// TaskConfig represents task configuration
type TaskConfig struct {
	Backend          string
	WorkerPool       int
	SQLConfig        *config.TaskSQLConfig
	InputTimeout     int
	Timeout          int
	HITLConfig       *config.HITLConfig
	CheckpointConfig *config.CheckpointConfig
}

// TaskSQLConfigBuilder provides a fluent API for building SQL task config
type TaskSQLConfigBuilder struct {
	config *config.TaskSQLConfig
}

// NewTaskSQLConfigBuilder creates a new SQL task config builder
func NewTaskSQLConfigBuilder(cfg *config.TaskSQLConfig) *TaskSQLConfigBuilder {
	if cfg == nil {
		cfg = &config.TaskSQLConfig{}
	}
	return &TaskSQLConfigBuilder{
		config: cfg,
	}
}

// Driver sets the database driver ("postgres", "mysql", or "sqlite")
func (b *TaskSQLConfigBuilder) Driver(driver string) *TaskSQLConfigBuilder {
	b.config.Driver = driver
	return b
}

// Host sets the database host
func (b *TaskSQLConfigBuilder) Host(host string) *TaskSQLConfigBuilder {
	b.config.Host = host
	return b
}

// Port sets the database port
func (b *TaskSQLConfigBuilder) Port(port int) *TaskSQLConfigBuilder {
	b.config.Port = port
	return b
}

// Database sets the database name
func (b *TaskSQLConfigBuilder) Database(db string) *TaskSQLConfigBuilder {
	b.config.Database = db
	return b
}

// Username sets the database username
func (b *TaskSQLConfigBuilder) Username(user string) *TaskSQLConfigBuilder {
	b.config.Username = user
	return b
}

// Password sets the database password
func (b *TaskSQLConfigBuilder) Password(pass string) *TaskSQLConfigBuilder {
	b.config.Password = pass
	return b
}

// SSLMode sets the SSL mode (for PostgreSQL)
func (b *TaskSQLConfigBuilder) SSLMode(mode string) *TaskSQLConfigBuilder {
	b.config.SSLMode = mode
	return b
}

// MaxConns sets the maximum connections
func (b *TaskSQLConfigBuilder) MaxConns(max int) *TaskSQLConfigBuilder {
	b.config.MaxConns = max
	return b
}

// MaxIdle sets the maximum idle connections
func (b *TaskSQLConfigBuilder) MaxIdle(max int) *TaskSQLConfigBuilder {
	b.config.MaxIdle = max
	return b
}

// Build returns the SQL config
func (b *TaskSQLConfigBuilder) Build() *config.TaskSQLConfig {
	return b.config
}
