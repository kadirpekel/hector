package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// SessionServiceBuilder provides a fluent API for building session services
type SessionServiceBuilder struct {
	backend          string
	database         string // Reference to SQL database from databases section
	rateLimit        *config.RateLimitConfig
	agentID          string
	componentManager *component.ComponentManager // For getting SQL database connections
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

// Database sets the SQL database reference for SQL backend
func (b *SessionServiceBuilder) Database(dbName string) *SessionServiceBuilder {
	b.database = dbName
	return b
}

// WithComponentManager sets the component manager for getting SQL database connections
func (b *SessionServiceBuilder) WithComponentManager(cm *component.ComponentManager) *SessionServiceBuilder {
	b.componentManager = cm
	return b
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

		service, err = memory.NewSQLSessionService(db, driver, b.agentID)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQL session service: %w", err)
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
