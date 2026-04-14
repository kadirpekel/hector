package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/verikod/hector/pkg/bootstrap"
	"github.com/verikod/hector/pkg/config"
)

// ServeCmd starts the Hector server with clean architecture.
type ServeCmd struct {
	// Database
	Database string `name:"database" env:"HECTOR_DATABASE" help:"Database DSN" default:"sqlite://.hector/hector.db"`

	// Network
	Host string `help:"Host to bind to" env:"HECTOR_HOST" default:"0.0.0.0"`
	Port int    `help:"Port to listen on" env:"HECTOR_PORT" default:"8080"`

	// Logging
	LogLevel  string `name:"log-level" env:"HECTOR_LOG_LEVEL" help:"Log level" default:"info"`
	LogFormat string `name:"log-format" env:"HECTOR_LOG_FORMAT" help:"Log format (json, text)" default:"text"`
	LogFile   string `name:"log-file" env:"HECTOR_LOG_FILE" help:"Log file path"`

	// Auth
	AuthSecret      string `name:"auth-secret" env:"HECTOR_AUTH_SECRET" help:"Admin API secret"`
	AuthJWKSURL     string `name:"auth-jwks-url" env:"HECTOR_AUTH_JWKS_URL" help:"JWKS URL for JWT auth"`
	AuthIssuer      string `name:"auth-issuer" env:"HECTOR_AUTH_ISSUER" help:"JWT issuer"`
	AuthAudience    string `name:"auth-audience" env:"HECTOR_AUTH_AUDIENCE" help:"JWT audience"`
	AuthClientID    string `name:"auth-client-id" env:"HECTOR_AUTH_CLIENT_ID" help:"Public client ID"`
	AuthSigningSeed string `name:"auth-signing-seed" env:"HECTOR_AUTH_SIGNING_SEED" help:"Deterministic seed for signing key"`

	// Queue
	QueueWorkers        int           `name:"queue-workers" env:"HECTOR_QUEUE_WORKERS" help:"Number of queue workers" default:"4"`
	QueueMaxRetries     int           `name:"queue-max-retries" env:"HECTOR_QUEUE_MAX_RETRIES" help:"Max retries for failed tasks" default:"3"`
	QueueInitialDelay   time.Duration `name:"queue-initial-delay" env:"HECTOR_QUEUE_INITIAL_DELAY" help:"Initial retry delay" default:"1s"`
	QueueMaxDelay       time.Duration `name:"queue-max-delay" env:"HECTOR_QUEUE_MAX_DELAY" help:"Max retry delay" default:"5m"`
	QueueStaleThreshold time.Duration `name:"queue-stale-threshold" env:"HECTOR_QUEUE_STALE_THRESHOLD" help:"Stale task recovery threshold" default:"5m"`

	// Observability
	MetricsEnabled  bool   `name:"metrics" help:"Enable Prometheus metrics"`
	TracingEndpoint string `name:"tracing-endpoint" env:"HECTOR_TRACING_ENDPOINT" help:"OTLP endpoint"`

	// Rate Limiting
	RateLimitEnabled   bool   `name:"rate-limit-enabled" env:"HECTOR_RATE_LIMIT_ENABLED" help:"Enable rate limiting"`
	RateLimitScope     string `name:"rate-limit-scope" env:"HECTOR_RATE_LIMIT_SCOPE" help:"Rate limit scope (session, user)" default:"session"`
	RateLimitLimits    string `name:"rate-limit-limits" env:"HECTOR_RATE_LIMIT_LIMITS" help:"Rate limits as JSON array"`
	RateLimitIPHeaders string `name:"rate-limit-ip-headers" env:"HECTOR_RATE_LIMIT_IP_HEADERS" help:"Comma-separated IP headers for client identification"`

	// Features
	Watch bool `help:"Watch config file for changes"`

	// App config
	Config string `short:"c" help:"Path to app config file" type:"path" default:".hector/config.yaml"`
}

func (c *ServeCmd) Run() error {
	// Build server config from CLI flags
	serverCfg := &config.ServerConfig{
		Database:  c.Database,
		Host:      c.Host,
		Port:      c.Port,
		LogLevel:  c.LogLevel,
		LogFormat: c.LogFormat,
		LogFile:   c.LogFile,

		Queue: &config.QueueConfig{
			Workers:        c.QueueWorkers,
			MaxRetries:     c.QueueMaxRetries,
			InitialDelay:   c.QueueInitialDelay,
			MaxDelay:       c.QueueMaxDelay,
			StaleThreshold: c.QueueStaleThreshold,
		},

		MetricsEnabled:  c.MetricsEnabled,
		TracingEndpoint: c.TracingEndpoint,
	}

	// Build auth config if provided
	if c.AuthSecret != "" || c.AuthJWKSURL != "" {
		serverCfg.Auth = &config.AuthConfig{
			Secret:      c.AuthSecret,
			JWKSURL:     c.AuthJWKSURL,
			Issuer:      c.AuthIssuer,
			Audience:    c.AuthAudience,
			ClientID:    c.AuthClientID,
			SigningSeed: c.AuthSigningSeed,
		}
	}

	// Build rate limit config if enabled
	if c.RateLimitEnabled {
		serverCfg.RateLimit = &config.RateLimitConfig{
			Enabled: config.BoolPtr(true),
			Scope:   c.RateLimitScope,
		}

		// Parse limits JSON if provided
		if c.RateLimitLimits != "" {
			var limits []config.RateLimitRule
			if err := json.Unmarshal([]byte(c.RateLimitLimits), &limits); err != nil {
				return fmt.Errorf("invalid rate-limit-limits JSON: %w", err)
			}
			serverCfg.RateLimit.Limits = limits
		}

		// Parse IP headers if provided
		if c.RateLimitIPHeaders != "" {
			serverCfg.RateLimit.IPHeaders = strings.Split(c.RateLimitIPHeaders, ",")
		}
	}

	// Start server using the reusable library API
	return bootstrap.Serve(context.Background(),
		bootstrap.WithServerConfig(serverCfg),
		bootstrap.WithConfigPath(c.Config),
		bootstrap.WithWatch(c.Watch),
	)
}
