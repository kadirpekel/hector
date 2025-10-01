package agent

import (
	"fmt"
	"strings"
	"sync"

	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/registry"
)

// AgentEntry represents a complete agent entry with all metadata
type AgentEntry struct {
	Agent        *Agent              `json:"agent"`
	Config       *config.AgentConfig `json:"config"`
	Capabilities []string            `json:"capabilities"`
	AgentType    string              `json:"agent_type"`
	Name         string              `json:"name"`
}

// AgentRegistry manages agents with single source of truth
type AgentRegistry struct {
	*registry.BaseRegistry[AgentEntry]
	mu        sync.RWMutex
	instances map[string][]*Agent // agent_type -> instance pool
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		BaseRegistry: registry.NewBaseRegistry[AgentEntry](),
		instances:    make(map[string][]*Agent),
	}
}

// AgentRegistryError represents an agent registry error
type AgentRegistryError struct {
	Component string
	Action    string
	Message   string
	Err       error
}

func (e *AgentRegistryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Component, e.Action, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Component, e.Action, e.Message)
}

func NewAgentRegistryError(component, action, message string, err error) *AgentRegistryError {
	return &AgentRegistryError{
		Component: component,
		Action:    action,
		Message:   message,
		Err:       err,
	}
}

// RegisterAgent registers an agent with the registry
func (r *AgentRegistry) RegisterAgent(name string, agent *Agent, agentConfig *config.AgentConfig, capabilities []string) error {
	if name == "" {
		return NewAgentRegistryError("AgentRegistry", "RegisterAgent", "agent name cannot be empty", nil)
	}
	if agent == nil {
		return NewAgentRegistryError("AgentRegistry", "RegisterAgent", "agent cannot be nil", nil)
	}
	if agentConfig == nil {
		return NewAgentRegistryError("AgentRegistry", "RegisterAgent", "agent config cannot be nil", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	agentType := r.extractAgentType(name)

	entry := AgentEntry{
		Agent:        agent,
		Config:       agentConfig,
		Capabilities: capabilities,
		AgentType:    agentType,
		Name:         name,
	}

	// Register with agent name as key - single source of truth
	if err := r.Register(name, entry); err != nil {
		return NewAgentRegistryError("AgentRegistry", "RegisterAgent",
			fmt.Sprintf("failed to register agent %s", name), err)
	}

	// Add to instance pool by agent type
	if r.instances[agentType] == nil {
		r.instances[agentType] = make([]*Agent, 0)
	}
	r.instances[agentType] = append(r.instances[agentType], agent)

	return nil
}

// GetAgent retrieves a specific agent by name
func (r *AgentRegistry) GetAgent(name string) (*Agent, error) {
	entry, exists := r.Get(name)
	if !exists {
		return nil, NewAgentRegistryError("AgentRegistry", "GetAgent",
			fmt.Sprintf("agent %s not found", name), nil)
	}
	return entry.Agent, nil
}

// GetAllAgents returns all registered agents
func (r *AgentRegistry) GetAllAgents() map[string]*Agent {
	agents := make(map[string]*Agent)

	for _, entry := range r.List() {
		agents[entry.Name] = entry.Agent
	}
	return agents
}

// GetAgentConfig retrieves agent configuration
func (r *AgentRegistry) GetAgentConfig(name string) (*config.AgentConfig, error) {
	entry, exists := r.Get(name)
	if !exists {
		return nil, NewAgentRegistryError("AgentRegistry", "GetAgentConfig",
			fmt.Sprintf("agent config for %s not found", name), nil)
	}
	return entry.Config, nil
}

// GetAgentsByType returns agents of a specific type
func (r *AgentRegistry) GetAgentsByType(agentType string) ([]*Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances, exists := r.instances[agentType]
	if !exists {
		return []*Agent{}, nil
	}

	result := make([]*Agent, len(instances))
	copy(result, instances)
	return result, nil
}

// GetCapabilities returns capabilities for an agent
func (r *AgentRegistry) GetCapabilities(name string) ([]string, error) {
	entry, exists := r.Get(name)
	if !exists {
		return nil, NewAgentRegistryError("AgentRegistry", "GetCapabilities",
			fmt.Sprintf("capabilities for agent %s not found", name), nil)
	}
	return entry.Capabilities, nil
}

// GetAgentsByCapability retrieves agents that have a specific capability
func (r *AgentRegistry) GetAgentsByCapability(capability string) ([]*Agent, error) {
	if capability == "" {
		return nil, NewAgentRegistryError("AgentRegistry", "GetAgentsByCapability", "capability cannot be empty", nil)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []*Agent
	for _, entry := range r.List() {
		for _, cap := range entry.Capabilities {
			if cap == capability {
				agents = append(agents, entry.Agent)
				break
			}
		}
	}

	return agents, nil
}

// ListAgents returns all registered agent names
func (r *AgentRegistry) ListAgents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.List()))
	for _, entry := range r.List() {
		names = append(names, entry.Name)
	}

	return names
}

// extractAgentType extracts the agent type from the agent name
func (r *AgentRegistry) extractAgentType(name string) string {
	if underscoreIndex := strings.LastIndex(name, "_"); underscoreIndex > 0 {
		return name[:underscoreIndex]
	}
	return name
}
