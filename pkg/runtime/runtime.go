// Package runtime provides runtime initialization and management for Hector
package runtime

import (
	"fmt"
	"os"

	"github.com/kadirpekel/hector/pkg/a2a/client"
	"github.com/kadirpekel/hector/pkg/config"
)

// Runtime manages the Hector runtime environment
type Runtime struct {
	config *config.Config
	client client.A2AClient
}

// Options holds options for runtime initialization
type Options struct {
	// Config file path
	ConfigFile string

	// Zero-config options (used if ConfigFile doesn't exist)
	Provider   string // LLM provider: "openai" (default), "anthropic", "gemini"
	APIKey     string
	BaseURL    string
	Model      string
	Tools      bool
	MCPURL     string
	DocsFolder string
}

// New creates a new Runtime instance
func New(opts Options) (*Runtime, error) {
	cfg, err := loadOrCreateConfig(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create local client
	a2aClient, err := client.NewLocalClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Runtime{
		config: cfg,
		client: a2aClient,
	}, nil
}

// Client returns the A2A client for this runtime
func (r *Runtime) Client() client.A2AClient {
	return r.client
}

// Config returns the configuration for this runtime
func (r *Runtime) Config() *config.Config {
	return r.config
}

// Close releases resources held by the runtime
func (r *Runtime) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// loadOrCreateConfig loads config from file or creates zero-config if file doesn't exist
func loadOrCreateConfig(opts Options) (*config.Config, error) {
	// Try to load from file first
	if _, err := os.Stat(opts.ConfigFile); err == nil {
		cfg, err := config.LoadConfig(opts.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}

		cfg.SetDefaults()
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}

		return cfg, nil
	}

	// File doesn't exist, create zero-config
	// Note: API key and MCP URL resolution from environment happens in CLI layer (parseArgs)
	cfg := config.CreateZeroConfig(config.ZeroConfigOptions{
		Provider:    opts.Provider, // Can be empty - defaults to "openai"
		APIKey:      opts.APIKey,   // Already resolved from flags or environment in CLI layer
		BaseURL:     opts.BaseURL,
		Model:       opts.Model,
		EnableTools: opts.Tools,
		MCPURL:      opts.MCPURL, // Already resolved from --mcp-url flag or MCP_URL env
		DocsFolder:  opts.DocsFolder,
	})

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid zero-config: %w", err)
	}

	return cfg, nil
}

// NewHTTPClient creates an HTTP-based A2A client (server mode)
func NewHTTPClient(serverURL, token string) client.A2AClient {
	return client.NewHTTPClient(serverURL, token)
}

// NewLocalClient creates a local (in-process) A2A client with custom config
func NewLocalClient(cfg *config.Config) (client.A2AClient, error) {
	return client.NewLocalClient(cfg)
}

// LoadConfigForValidation loads config without creating expensive client
// Used by CLI to validate agent exists before initialization
func LoadConfigForValidation(configFile string, opts Options) (*config.Config, error) {
	opts.ConfigFile = configFile
	return loadOrCreateConfig(opts)
}

// NewWithConfig creates a runtime with pre-loaded config
// Used after validation to avoid double-loading config
func NewWithConfig(cfg *config.Config) (*Runtime, error) {
	// Create local client with validated config
	a2aClient, err := client.NewLocalClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Runtime{
		config: cfg,
		client: a2aClient,
	}, nil
}
