package runtime

import (
	"fmt"
	"log"

	"github.com/kadirpekel/hector/pkg/a2a/client"
	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/hector"
)

type Runtime struct {
	config     *config.Config
	components *component.ComponentManager
	registry   *agent.AgentRegistry
}

type Options struct {
	ConfigFile string

	Provider   string
	APIKey     string
	BaseURL    string
	Model      string
	Tools      bool
	MCPURL     string
	DocsFolder string
	AgentName  string
}

func (r *Runtime) Registry() *agent.AgentRegistry {
	return r.registry
}

func (r *Runtime) Components() *component.ComponentManager {
	return r.components
}

func (r *Runtime) Config() *config.Config {
	return r.config
}

// GetAgentID returns empty string for Runtime (not applicable for multi-agent local runtime)
func (r *Runtime) GetAgentID() string {
	return ""
}

func (r *Runtime) Close() error {
	var errors []error

	if r.components != nil {
		if err := r.components.Close(); err != nil {
			errors = append(errors, fmt.Errorf("component manager cleanup: %w", err))
			log.Printf("Warning: Component manager cleanup error: %v", err)
		}
	}

	if len(errors) > 0 {
		return errors[0]
	}
	return nil
}

func NewHTTPClient(serverURL, token string) client.A2AClient {
	return client.NewHTTPClient(serverURL, token)
}

func NewWithConfig(cfg *config.Config) (*Runtime, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Use programmatic API builder internally (foundation pattern)
	configBuilder, err := hector.NewConfigAgentBuilder(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create config builder: %w", err)
	}

	// Build all agents using programmatic API
	agents, err := configBuilder.BuildAllAgents()
	if err != nil {
		return nil, fmt.Errorf("failed to build agents: %w", err)
	}

	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents configured")
	}

	// Get component manager and registry from config builder
	componentManager := configBuilder.ComponentManager()
	agentRegistry := configBuilder.AgentRegistry()

	return &Runtime{
		config:     cfg,
		components: componentManager,
		registry:   agentRegistry,
	}, nil
}

// NewRuntimeBuilder creates a new runtime builder (programmatic API)
func NewRuntimeBuilder() *RuntimeBuilder {
	return &RuntimeBuilder{
		agents: make(map[string]*agent.Agent),
	}
}

// RuntimeBuilder provides a fluent API for building runtime programmatically
type RuntimeBuilder struct {
	agents map[string]*agent.Agent
}

// WithAgent adds an agent to the runtime
func (b *RuntimeBuilder) WithAgent(agent *agent.Agent) *RuntimeBuilder {
	if agent == nil {
		panic("agent cannot be nil")
	}
	b.agents[agent.GetID()] = agent
	return b
}

// WithAgents adds multiple agents to the runtime
func (b *RuntimeBuilder) WithAgents(agents map[string]*agent.Agent) *RuntimeBuilder {
	for id, agent := range agents {
		if agent == nil {
			panic(fmt.Sprintf("agent %s cannot be nil", id))
		}
		b.agents[id] = agent
	}
	return b
}

// Start creates and starts the runtime
func (b *RuntimeBuilder) Start() (*Runtime, error) {
	if len(b.agents) == 0 {
		return nil, fmt.Errorf("at least one agent is required")
	}

	// Create agent registry
	registry := agent.NewAgentRegistry()

	// Register all agents
	for id, agentInstance := range b.agents {
		if err := registry.RegisterAgent(id, agentInstance, nil, nil); err != nil {
			return nil, fmt.Errorf("failed to register agent %s: %w", id, err)
		}
	}

	// Create runtime with agents
	return NewRuntimeWithAgents(registry)
}

// NewRuntimeWithAgents creates a runtime directly from an agent registry
func NewRuntimeWithAgents(registry *agent.AgentRegistry) (*Runtime, error) {
	if registry == nil {
		return nil, fmt.Errorf("agent registry is required")
	}

	return &Runtime{
		config:     nil, // No config for programmatic runtime
		components: nil, // No components for programmatic runtime
		registry:   registry,
	}, nil
}
