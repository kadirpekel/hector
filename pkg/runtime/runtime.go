package runtime

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/kadirpekel/hector/pkg/a2a/client"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
)

// Runtime manages the Hector runtime environment
// This is the CORE foundation used by both local mode and server mode
// Runtime implements the A2AClient interface for local mode usage
type Runtime struct {
	config     *config.Config
	components *component.ComponentManager
	registry   *agent.AgentRegistry
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
	AgentName  string // Agent name/ID for zero-config agent (default: "assistant")
}

// ToZeroConfigOptions converts runtime Options to config.ZeroConfigOptions
// This consolidates the mapping logic in one place to avoid duplication
func (o *Options) ToZeroConfigOptions() config.ZeroConfigOptions {
	return config.ZeroConfigOptions{
		Provider:    o.Provider,
		APIKey:      o.APIKey,
		BaseURL:     o.BaseURL,
		Model:       o.Model,
		EnableTools: o.Tools,
		MCPURL:      o.MCPURL,
		DocsFolder:  o.DocsFolder,
		AgentName:   o.AgentName,
	}
}

// New creates a new Runtime instance
func New(opts Options) (*Runtime, error) {
	cfg, err := loadOrCreateConfig(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return NewWithConfig(cfg)
}

// Registry returns the agent registry (for server mode)
func (r *Runtime) Registry() *agent.AgentRegistry {
	return r.registry
}

// Components returns the component manager (for server mode)
func (r *Runtime) Components() *component.ComponentManager {
	return r.components
}

// Config returns the configuration for this runtime
func (r *Runtime) Config() *config.Config {
	return r.config
}

// Close releases resources held by the runtime
func (r *Runtime) Close() error {
	var errors []error

	// Shutdown plugins (they may have external connections)
	if r.components != nil {
		ctx := context.Background()
		if err := r.components.ShutdownPlugins(ctx); err != nil {
			errors = append(errors, fmt.Errorf("plugin shutdown: %w", err))
			log.Printf("⚠️  Warning: Plugin shutdown error: %v", err)
		}
	}

	// Note: Registry and component registries don't need explicit cleanup
	// They don't hold external resources that require closing

	// Return first error if any occurred
	if len(errors) > 0 {
		return errors[0]
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

	// File doesn't exist, create zero-config using consolidated method
	cfg := config.CreateZeroConfig(opts.ToZeroConfigOptions())

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

// LoadConfigForValidation loads config without creating expensive client
// Used by CLI to validate agent exists before initialization
func LoadConfigForValidation(configFile string, opts Options) (*config.Config, error) {
	opts.ConfigFile = configFile
	return loadOrCreateConfig(opts)
}

// NewWithConfig creates a runtime with pre-loaded config
// This is the CORE initialization used by both local mode and server mode
func NewWithConfig(cfg *config.Config) (*Runtime, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Create agent registry first
	agentRegistry := agent.NewAgentRegistry()

	// Create component manager with agent registry for agent_call tool
	componentManager, err := component.NewComponentManagerWithAgentRegistry(cfg, agentRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	// Initialize and register all configured agents
	// Track failures for better error reporting
	var failures []string
	successCount := 0

	for agentID, agentCfg := range cfg.Agents {
		agentCfgCopy := agentCfg

		// Create agent based on type (native vs external)
		var agentInstance pb.A2AServiceServer
		var err error

		if agentCfgCopy.Type == "a2a" {
			// External A2A agent - create client proxy
			externalAgent, extErr := agent.NewExternalA2AAgent(&agentCfgCopy)
			if extErr != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", agentID, extErr))
				log.Printf("  ⚠️  Failed to create external agent '%s': %v", agentID, extErr)
				continue
			}
			agentInstance = externalAgent
		} else {
			// Native agent - create local instance
			agentInstance, err = agent.NewAgent(agentID, &agentCfgCopy, componentManager, agentRegistry)
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", agentID, err))
				log.Printf("  ⚠️  Failed to create native agent '%s': %v", agentID, err)
				continue
			}
		}

		// Register agent in registry
		if err := agentRegistry.RegisterAgent(agentID, agentInstance, &agentCfgCopy, nil); err != nil {
			failures = append(failures, fmt.Sprintf("%s (registration): %v", agentID, err))
			log.Printf("  ⚠️  Failed to register agent '%s': %v", agentID, err)
			continue
		}

		successCount++
	}

	// Return error if NO agents were initialized
	if successCount == 0 {
		if len(failures) > 0 {
			return nil, fmt.Errorf("failed to initialize any agents (attempted: %d, failures: %v)",
				len(cfg.Agents), failures)
		}
		return nil, fmt.Errorf("no agents configured")
	}

	// Log warning for partial failures
	if len(failures) > 0 {
		log.Printf("⚠️  Warning: %d/%d agents failed to initialize: %v",
			len(failures), len(cfg.Agents), failures)
	}

	return &Runtime{
		config:     cfg,
		components: componentManager,
		registry:   agentRegistry,
	}, nil
}
