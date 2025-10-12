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

	// Create direct client
	a2aClient, err := client.NewDirectClient(cfg)
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
	// Default to openai if no provider specified
	provider := opts.Provider
	if provider == "" {
		provider = "openai"
	}

	apiKey := opts.APIKey
	if apiKey == "" {
		// Try provider-specific env vars
		switch provider {
		case "anthropic":
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		case "gemini":
			apiKey = os.Getenv("GEMINI_API_KEY")
		default: // openai
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
	}
	if apiKey == "" {
		envVar := fmt.Sprintf("%s_API_KEY", provider)
		if provider == "openai" {
			envVar = "OPENAI_API_KEY"
		} else if provider == "anthropic" {
			envVar = "ANTHROPIC_API_KEY"
		} else if provider == "gemini" {
			envVar = "GEMINI_API_KEY"
		}
		return nil, fmt.Errorf("API key required for zero-config mode (use --api-key or set %s environment variable)", envVar)
	}

	zeroOpts := config.ZeroConfigOptions{
		Provider:    provider,
		APIKey:      apiKey,
		BaseURL:     opts.BaseURL,
		Model:       opts.Model,
		EnableTools: opts.Tools,
		MCPURL:      opts.MCPURL,
		DocsFolder:  opts.DocsFolder,
	}

	cfg := config.CreateZeroConfig(zeroOpts)
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid zero-config: %w", err)
	}

	return cfg, nil
}

// NewHTTPClient creates an HTTP-based A2A client (server mode)
func NewHTTPClient(serverURL, token string) client.A2AClient {
	return client.NewHTTPClient(serverURL, token)
}

// NewDirectClient creates a direct (in-process) A2A client with custom config
func NewDirectClient(cfg *config.Config) (client.A2AClient, error) {
	return client.NewDirectClient(cfg)
}
