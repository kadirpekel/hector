package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// TaskServiceBuilder provides a fluent API for building task services
type TaskServiceBuilder struct {
	backend      string
	workerPool   int
	sqlConfig    *config.TaskSQLConfig
	inputTimeout int
	timeout      int
}

// NewTaskService creates a new task service builder
func NewTaskService() *TaskServiceBuilder {
	return &TaskServiceBuilder{
		backend:      "memory",
		workerPool:   100,
		inputTimeout: 600, // 10 minutes default
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

// WithSQLConfig sets the SQL configuration for SQL backend
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

// Build creates the task service
func (b *TaskServiceBuilder) Build() (reasoning.TaskService, error) {
	switch b.backend {
	case "memory":
		return agent.NewInMemoryTaskService(), nil

	case "sql":
		if b.sqlConfig == nil {
			return nil, fmt.Errorf("SQL configuration is required for SQL backend")
		}
		b.sqlConfig.SetDefaults()
		if err := b.sqlConfig.Validate(); err != nil {
			return nil, fmt.Errorf("invalid SQL configuration: %w", err)
		}
		return agent.NewSQLTaskServiceFromConfig(b.sqlConfig)

	default:
		return nil, fmt.Errorf("unsupported backend: %s", b.backend)
	}
}

// GetConfig returns the task configuration
func (b *TaskServiceBuilder) GetConfig() TaskConfig {
	return TaskConfig{
		Backend:      b.backend,
		WorkerPool:   b.workerPool,
		SQLConfig:    b.sqlConfig,
		InputTimeout: b.inputTimeout,
		Timeout:      b.timeout,
	}
}

// TaskConfig represents task configuration
type TaskConfig struct {
	Backend      string
	WorkerPool   int
	SQLConfig    *config.TaskSQLConfig
	InputTimeout int
	Timeout      int
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

