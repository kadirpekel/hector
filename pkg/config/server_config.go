package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// ServerConfig holds server-level operational settings.
// These are passed via CLI flags and environment variables ONLY.
// They control HOW the server runs, not WHAT it does.
type ServerConfig struct {
	// Database
	Database       string          // Raw DSN (e.g., "postgres://...", "sqlite://...")
	DatabaseConfig *DatabaseConfig // Parsed database configuration

	// Network
	Host string // Host to bind to (e.g., "0.0.0.0", "127.0.0.1")
	Port int    // Port to listen on (e.g., 8080)

	// Logging
	LogLevel  string // Log level: debug, info, warn, error
	LogFormat string // Log format: json, text
	LogFile   string // Log file path (empty = stderr)

	// Authentication
	Auth *AuthConfig // Authentication configuration

	// Queue configuration for durable task execution
	Queue *QueueConfig

	// Observability
	MetricsEnabled  bool   // Enable Prometheus metrics endpoint
	TracingEndpoint string // OTLP tracing endpoint (e.g., "localhost:4317")

	// Rate Limiting (operational/infrastructure)
	RateLimit *RateLimitConfig `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`

	// GitAllowedRepos is an optional allowlist of repository URL prefixes
	// permitted for git-backed document sources.
	// When empty, all repositories are allowed.
	GitAllowedRepos []string
}

// QueueConfig configures the task queue for durable execution.
type QueueConfig struct {
	// Workers is the number of concurrent worker goroutines.
	// Default: 4
	Workers int `yaml:"workers" json:"workers"`

	// MaxRetries is the maximum number of retry attempts for failed tasks.
	// Default: 3
	MaxRetries int `yaml:"max_retries" json:"max_retries"`

	// InitialDelay is the delay before the first retry.
	// Default: 1s
	InitialDelay time.Duration `yaml:"initial_delay" json:"initial_delay"`

	// MaxDelay is the maximum delay between retries.
	// Default: 5m
	MaxDelay time.Duration `yaml:"max_delay" json:"max_delay"`

	// BackoffFactor is the multiplier for exponential backoff.
	// Default: 2.0
	BackoffFactor float64 `yaml:"backoff_factor" json:"backoff_factor"`

	// StaleThreshold is how old a "processing" item must be to be considered stale.
	// Default: 5m
	StaleThreshold time.Duration `yaml:"stale_threshold" json:"stale_threshold"`
}

// SetDefaults applies default values to queue config.
func (q *QueueConfig) SetDefaults() {
	if q.Workers <= 0 {
		q.Workers = 4
	}
	if q.MaxRetries <= 0 {
		q.MaxRetries = 3
	}
	if q.InitialDelay <= 0 {
		q.InitialDelay = 1 * time.Second
	}
	if q.MaxDelay <= 0 {
		q.MaxDelay = 5 * time.Minute
	}
	if q.BackoffFactor <= 0 {
		q.BackoffFactor = 2.0
	}
	if q.StaleThreshold <= 0 {
		q.StaleThreshold = 5 * time.Minute
	}
}

// AuthConfig configures authentication for the server.
type AuthConfig struct {
	// Secret token for admin API authentication
	Secret string

	// JWT authentication (optional)
	JWKSURL         string // JWKS URL for JWT validation
	Issuer          string // Expected JWT issuer
	Audience        string // Expected JWT audience
	ClientID        string // Public client ID for frontend
	RefreshInterval int    // JWKS refresh interval in seconds

	// SigningSeed is a deterministic seed for generating the signing key (Ed25519 or RSA).
	// If provided, the server will generate the same key on every restart.
	SigningSeed string

	// Runtime state
	Enabled bool // Computed: true if Secret or JWKS is configured
}

// IsEnabled returns true if authentication is configured.
func (a *AuthConfig) IsEnabled() bool {
	if a == nil {
		return false
	}
	// Auth is enabled if either secret or JWKS is configured
	return a.Secret != "" || (a.JWKSURL != "" && a.Issuer != "" && a.Audience != "")
}

// SetDefaults applies default values to auth config.
func (a *AuthConfig) SetDefaults() {
	if a == nil {
		return
	}
	// Default refresh interval for JWKS (15 minutes)
	if a.RefreshInterval == 0 && a.JWKSURL != "" {
		a.RefreshInterval = 900
	}
}

// Validate validates the auth configuration.
func (a *AuthConfig) Validate() error {
	if a == nil || !a.Enabled {
		return nil
	}

	// If using JWKS, all three fields are required
	if a.JWKSURL != "" || a.Issuer != "" || a.Audience != "" {
		if a.JWKSURL == "" {
			return fmt.Errorf("auth.jwks_url is required when using JWT authentication")
		}
		if a.Issuer == "" {
			return fmt.Errorf("auth.issuer is required when using JWT authentication")
		}
		if a.Audience == "" {
			return fmt.Errorf("auth.audience is required when using JWT authentication")
		}
	}

	return nil
}

// SetDefaults applies default values to server config.
func (c *ServerConfig) SetDefaults() {
	if c.Database == "" {
		c.Database = DefaultDSN()
	}

	if c.Host == "" {
		c.Host = "0.0.0.0"
	}

	if c.Port == 0 {
		c.Port = 8080
	}

	if c.LogLevel == "" {
		c.LogLevel = "info"
	}

	if c.LogFormat == "" {
		c.LogFormat = "text"
	}

	// Parse database DSN if not already parsed
	if c.DatabaseConfig == nil && c.Database != "" {
		dbCfg, err := ParseDSN(c.Database)
		if err != nil {
			slog.Warn("Failed to parse database DSN", "dsn", c.Database, "error", err)
		} else {
			c.DatabaseConfig = dbCfg
		}
	}

	// Set auth enabled flag
	if c.Auth != nil {
		c.Auth.Enabled = c.Auth.IsEnabled()
	}

	// Load rate limit config from environment variables
	if err := c.loadRateLimitFromEnv(); err != nil {
		// Log error but don't fail - validation will catch invalid config later
		slog.Warn("Failed to load rate limit config from environment", "error", err)
	}

	// Load git allowlist from environment variables.
	c.loadGitAllowlistFromEnv()

	// Set rate limit defaults
	if c.RateLimit != nil {
		c.RateLimit.SetDefaults()
	}

	// Initialize queue config with defaults
	if c.Queue == nil {
		c.Queue = &QueueConfig{}
	}
	c.Queue.SetDefaults()
}

// loadRateLimitFromEnv loads rate limiting configuration from environment variables.
//
// Example:
//
//	export HECTOR_RATE_LIMIT_ENABLED=true
//	export HECTOR_RATE_LIMIT_SCOPE=session
//	export HECTOR_RATE_LIMIT_LIMITS='[{"type":"token","window":"day","limit":100000},{"type":"count","window":"minute","limit":60}]'
//	export HECTOR_RATE_LIMIT_IP_HEADERS="CF-Connecting-IP,X-Forwarded-For,X-Real-IP"  # Cloudflare example
//
// Returns an error if HECTOR_RATE_LIMIT_LIMITS contains invalid JSON.
func (c *ServerConfig) loadRateLimitFromEnv() error {
	// Check if rate limiting is enabled
	if enabled := os.Getenv("HECTOR_RATE_LIMIT_ENABLED"); enabled == "true" {
		if c.RateLimit == nil {
			c.RateLimit = &RateLimitConfig{}
		}
		c.RateLimit.Enabled = BoolPtr(true)

		// Load scope
		if scope := os.Getenv("HECTOR_RATE_LIMIT_SCOPE"); scope != "" {
			c.RateLimit.Scope = scope
		}

		// Load limits from JSON (if provided)
		if limitsJSON := os.Getenv("HECTOR_RATE_LIMIT_LIMITS"); limitsJSON != "" {
			var limits []RateLimitRule
			if err := json.Unmarshal([]byte(limitsJSON), &limits); err != nil {
				return fmt.Errorf("invalid HECTOR_RATE_LIMIT_LIMITS JSON: %w", err)
			}
			c.RateLimit.Limits = limits
		}

		// Load IP headers (comma-separated list)
		if ipHeadersEnv := os.Getenv("HECTOR_RATE_LIMIT_IP_HEADERS"); ipHeadersEnv != "" {
			headers := strings.Split(ipHeadersEnv, ",")
			// Trim whitespace from each header name
			for i, h := range headers {
				headers[i] = strings.TrimSpace(h)
			}
			c.RateLimit.IPHeaders = headers
		}
	}
	return nil
}

// loadGitAllowlistFromEnv loads git repository allowlist from environment.
//
// Example:
//
//	export HECTOR_GIT_ALLOWED_REPOS="https://github.com/verikod/hector,https://github.com/verikod/"
func (c *ServerConfig) loadGitAllowlistFromEnv() {
	val := os.Getenv("HECTOR_GIT_ALLOWED_REPOS")
	if strings.TrimSpace(val) == "" {
		return
	}

	parts := strings.Split(val, ",")
	allow := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			allow = append(allow, trimmed)
		}
	}

	if len(allow) > 0 {
		c.GitAllowedRepos = allow
	}
}

// Validate validates the server configuration.
func (c *ServerConfig) Validate() error {
	// Validate database DSN
	if c.Database != "" {
		if _, err := ParseDSN(c.Database); err != nil {
			return fmt.Errorf("invalid database DSN: %w", err)
		}
	}

	// Validate port
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Port)
	}

	// Validate log level
	validLevels := []string{"debug", "info", "warn", "error"}
	levelValid := false
	for _, level := range validLevels {
		if strings.ToLower(c.LogLevel) == level {
			levelValid = true
			break
		}
	}
	if !levelValid {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}

	// Validate log format
	if c.LogFormat != "json" && c.LogFormat != "text" {
		return fmt.Errorf("invalid log format: %s (must be json or text)", c.LogFormat)
	}

	// Validate auth config
	if c.Auth != nil {
		if err := c.Auth.Validate(); err != nil {
			return fmt.Errorf("invalid auth configuration: %w", err)
		}
	}

	// Validate rate limit config
	if c.RateLimit != nil {
		if err := c.RateLimit.Validate(); err != nil {
			return fmt.Errorf("invalid rate limit configuration: %w", err)
		}
	}

	for _, r := range c.GitAllowedRepos {
		if strings.TrimSpace(r) == "" {
			return fmt.Errorf("invalid git allowlist entry: empty value")
		}
	}

	return nil
}

// Address returns the server address as host:port.
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GetDatabaseConfig returns the parsed database configuration.
func (c *ServerConfig) GetDatabaseConfig() (*DatabaseConfig, error) {
	if c.DatabaseConfig != nil {
		return c.DatabaseConfig, nil
	}

	if c.Database == "" {
		return nil, fmt.Errorf("database not configured")
	}

	dbCfg, err := ParseDSN(c.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database DSN: %w", err)
	}

	c.DatabaseConfig = dbCfg
	return c.DatabaseConfig, nil
}
