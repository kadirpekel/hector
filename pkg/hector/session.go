package hector

import (
	"database/sql"
	"fmt"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// SessionServiceBuilder provides a fluent API for building session services
type SessionServiceBuilder struct {
	backend    string
	sqlConfig  *config.SessionSQLConfig
	rateLimit  *config.RateLimitConfig
	agentID    string
}

// NewSessionService creates a new session service builder
func NewSessionService(agentID string) *SessionServiceBuilder {
	if agentID == "" {
		panic("agent ID is required for session service")
	}
	return &SessionServiceBuilder{
		backend: "memory",
		agentID: agentID,
	}
}

// Backend sets the session backend ("memory" or "sql")
func (b *SessionServiceBuilder) Backend(backend string) *SessionServiceBuilder {
	if backend != "memory" && backend != "sql" {
		panic(fmt.Sprintf("invalid backend: %s (must be 'memory' or 'sql')", backend))
	}
	b.backend = backend
	return b
}

// WithSQLConfig sets the SQL configuration for SQL backend
func (b *SessionServiceBuilder) WithSQLConfig(cfg *config.SessionSQLConfig) *SessionServiceBuilder {
	b.sqlConfig = cfg
	return b
}

// SQLConfig creates a SQL config builder
func (b *SessionServiceBuilder) SQLConfig() *SessionSQLConfigBuilder {
	if b.sqlConfig == nil {
		b.sqlConfig = &config.SessionSQLConfig{}
	}
	return NewSessionSQLConfigBuilder(b.sqlConfig)
}

// WithRateLimit sets the rate limit configuration
func (b *SessionServiceBuilder) WithRateLimit(cfg *config.RateLimitConfig) *SessionServiceBuilder {
	b.rateLimit = cfg
	return b
}

// RateLimit creates a rate limit config builder
func (b *SessionServiceBuilder) RateLimit() *RateLimitConfigBuilder {
	if b.rateLimit == nil {
		b.rateLimit = &config.RateLimitConfig{}
	}
	return NewRateLimitConfigBuilder(b.rateLimit)
}

// Build creates the session service
func (b *SessionServiceBuilder) Build() (reasoning.SessionService, error) {
	var service reasoning.SessionService

	switch b.backend {
	case "memory":
		service = memory.NewInMemorySessionService()

	case "sql":
		if b.sqlConfig == nil {
			return nil, fmt.Errorf("SQL configuration is required for SQL backend")
		}
		b.sqlConfig.SetDefaults()
		if err := b.sqlConfig.Validate(); err != nil {
			return nil, fmt.Errorf("invalid SQL configuration: %w", err)
		}

		// Create database connection
		driverName := b.sqlConfig.Driver
		if driverName == "sqlite" {
			driverName = "sqlite3"
		}

		db, err := sql.Open(driverName, b.sqlConfig.ConnectionString())
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}

		db.SetMaxOpenConns(b.sqlConfig.MaxConns)
		db.SetMaxIdleConns(b.sqlConfig.MaxIdle)

		var err2 error
		service, err2 = memory.NewSQLSessionService(db, b.sqlConfig.Driver, b.agentID)
		if err2 != nil {
			db.Close()
			return nil, fmt.Errorf("failed to create SQL session service: %w", err2)
		}

	default:
		return nil, fmt.Errorf("unsupported backend: %s", b.backend)
	}

	// Note: Rate limiting is typically handled at a higher level (transport/server)
	// The session service itself doesn't wrap with rate limiting
	// Rate limit config is stored for reference but not applied here

	return service, nil
}

// GetRateLimitConfig returns the rate limit configuration
func (b *SessionServiceBuilder) GetRateLimitConfig() *config.RateLimitConfig {
	return b.rateLimit
}

// SessionSQLConfigBuilder provides a fluent API for building SQL session config
type SessionSQLConfigBuilder struct {
	config *config.SessionSQLConfig
}

// NewSessionSQLConfigBuilder creates a new SQL session config builder
func NewSessionSQLConfigBuilder(cfg *config.SessionSQLConfig) *SessionSQLConfigBuilder {
	if cfg == nil {
		cfg = &config.SessionSQLConfig{}
	}
	return &SessionSQLConfigBuilder{
		config: cfg,
	}
}

// Driver sets the database driver ("postgres", "mysql", or "sqlite")
func (b *SessionSQLConfigBuilder) Driver(driver string) *SessionSQLConfigBuilder {
	b.config.Driver = driver
	return b
}

// Host sets the database host
func (b *SessionSQLConfigBuilder) Host(host string) *SessionSQLConfigBuilder {
	b.config.Host = host
	return b
}

// Port sets the database port
func (b *SessionSQLConfigBuilder) Port(port int) *SessionSQLConfigBuilder {
	b.config.Port = port
	return b
}

// Database sets the database name
func (b *SessionSQLConfigBuilder) Database(db string) *SessionSQLConfigBuilder {
	b.config.Database = db
	return b
}

// Username sets the database username
func (b *SessionSQLConfigBuilder) Username(user string) *SessionSQLConfigBuilder {
	b.config.Username = user
	return b
}

// Password sets the database password
func (b *SessionSQLConfigBuilder) Password(pass string) *SessionSQLConfigBuilder {
	b.config.Password = pass
	return b
}

// SSLMode sets the SSL mode (for PostgreSQL)
func (b *SessionSQLConfigBuilder) SSLMode(mode string) *SessionSQLConfigBuilder {
	b.config.SSLMode = mode
	return b
}

// MaxConns sets the maximum connections
func (b *SessionSQLConfigBuilder) MaxConns(max int) *SessionSQLConfigBuilder {
	b.config.MaxConns = max
	return b
}

// MaxIdle sets the maximum idle connections
func (b *SessionSQLConfigBuilder) MaxIdle(max int) *SessionSQLConfigBuilder {
	b.config.MaxIdle = max
	return b
}

// Build returns the SQL config
func (b *SessionSQLConfigBuilder) Build() *config.SessionSQLConfig {
	return b.config
}

// RateLimitConfigBuilder provides a fluent API for building rate limit config
type RateLimitConfigBuilder struct {
	config *config.RateLimitConfig
}

// NewRateLimitConfigBuilder creates a new rate limit config builder
func NewRateLimitConfigBuilder(cfg *config.RateLimitConfig) *RateLimitConfigBuilder {
	if cfg == nil {
		cfg = &config.RateLimitConfig{}
	}
	return &RateLimitConfigBuilder{
		config: cfg,
	}
}

// Enabled enables or disables rate limiting
func (b *RateLimitConfigBuilder) Enabled(enabled bool) *RateLimitConfigBuilder {
	b.config.Enabled = &enabled
	return b
}

// Scope sets the rate limit scope ("session" or "user")
func (b *RateLimitConfigBuilder) Scope(scope string) *RateLimitConfigBuilder {
	b.config.Scope = scope
	return b
}

// Backend sets the rate limit backend ("memory" or "sql")
func (b *RateLimitConfigBuilder) Backend(backend string) *RateLimitConfigBuilder {
	b.config.Backend = backend
	return b
}

// WithLimit adds a rate limit rule
func (b *RateLimitConfigBuilder) WithLimit(ruleType string, window string, limit int64) *RateLimitConfigBuilder {
	if b.config.Limits == nil {
		b.config.Limits = make([]config.RateLimitRule, 0)
	}
	b.config.Limits = append(b.config.Limits, config.RateLimitRule{
		Type:   ruleType,
		Window: window,
		Limit:  limit,
	})
	return b
}

// Build returns the rate limit config
func (b *RateLimitConfigBuilder) Build() *config.RateLimitConfig {
	b.config.SetDefaults()
	return b.config
}

