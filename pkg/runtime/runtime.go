package runtime

import (
	"fmt"
	"log"

	"github.com/kadirpekel/hector/pkg/a2a/client"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
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

	agentRegistry := agent.NewAgentRegistry()

	componentManager, err := component.NewComponentManagerWithAgentRegistry(cfg, agentRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	// Resolve the base URL from A2A server configuration
	baseURL := resolveBaseURL(cfg)

	// Get preferred transport from config (global default)
	preferredTransport := cfg.Global.A2AServer.PreferredTransport
	if preferredTransport == "" {
		preferredTransport = "json-rpc" // Default
	}

	var failures []string
	successCount := 0

	var cleanupOnError = func() {
		if err := componentManager.Close(); err != nil {
			log.Printf("⚠️  Warning: Failed to cleanup component manager on error: %v", err)
		}
	}

	for agentID, agentCfg := range cfg.Agents {
		agentCfgCopy := agentCfg

		var agentInstance pb.A2AServiceServer
		var err error

		if agentCfgCopy.Type == "a2a" {

			externalAgent, extErr := agent.NewExternalA2AAgent(agentID, agentCfgCopy)
			if extErr != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", agentID, extErr))
				log.Printf("  Warning: Failed to create external agent '%s': %v", agentID, extErr)
				continue
			}
			agentInstance = externalAgent
		} else {

			agentInstance, err = agent.NewAgent(agentID, agentCfgCopy, componentManager, agentRegistry, baseURL, preferredTransport)
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", agentID, err))
				log.Printf("  Warning: Failed to create native agent '%s': %v", agentID, err)
				continue
			}
		}

		if err := agentRegistry.RegisterAgent(agentID, agentInstance, agentCfgCopy, nil); err != nil {
			failures = append(failures, fmt.Sprintf("%s (registration): %v", agentID, err))
			log.Printf("  Warning: Failed to register agent '%s': %v", agentID, err)
			continue
		}

		successCount++
	}

	if successCount == 0 {
		cleanupOnError()
		if len(failures) > 0 {
			return nil, fmt.Errorf("failed to initialize any agents (attempted: %d, failures: %v)",
				len(cfg.Agents), failures)
		}
		return nil, fmt.Errorf("no agents configured")
	}

	if len(failures) > 0 {
		log.Printf("Warning: %d/%d agents failed to initialize: %v",
			len(failures), len(cfg.Agents), failures)
	}

	return &Runtime{
		config:     cfg,
		components: componentManager,
		registry:   agentRegistry,
	}, nil
}

// resolveBaseURL constructs the base URL from the A2A server configuration
func resolveBaseURL(cfg *config.Config) string {
	// If base_url is explicitly set, use it
	if cfg.Global.A2AServer.BaseURL != "" {
		return cfg.Global.A2AServer.BaseURL
	}

	// Otherwise construct from host and port
	host := cfg.Global.A2AServer.Host
	if host == "" || host == "0.0.0.0" {
		host = "localhost"
	}

	port := cfg.Global.A2AServer.Port
	if port == 0 {
		port = 8080
	}

	return fmt.Sprintf("http://%s:%d", host, port)
}
