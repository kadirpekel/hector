package agent

import (
	"fmt"
	"strings"
	"sync"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/registry"
)

type AgentEntry struct {
	Agent        pb.A2AServiceServer `json:"agent"`
	Config       *config.AgentConfig `json:"config"`
	Capabilities []string            `json:"capabilities"`
	AgentType    string              `json:"agent_type"`
	Name         string              `json:"name"`
}

type AgentRegistry struct {
	*registry.BaseRegistry[AgentEntry]
	mu        sync.RWMutex
	instances map[string][]pb.A2AServiceServer
}

func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		BaseRegistry: registry.NewBaseRegistry[AgentEntry](),
		instances:    make(map[string][]pb.A2AServiceServer),
	}
}

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

func (r *AgentRegistry) RegisterAgent(name string, agent pb.A2AServiceServer, agentConfig *config.AgentConfig, capabilities []string) error {
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

	if err := r.Register(name, entry); err != nil {
		return NewAgentRegistryError("AgentRegistry", "RegisterAgent",
			fmt.Sprintf("failed to register agent %s", name), err)
	}

	if r.instances[agentType] == nil {
		r.instances[agentType] = make([]pb.A2AServiceServer, 0)
	}
	r.instances[agentType] = append(r.instances[agentType], agent)

	return nil
}

func (r *AgentRegistry) GetAgent(name string) (pb.A2AServiceServer, error) {
	entry, exists := r.Get(name)
	if !exists {

		allEntries := r.List()
		if len(allEntries) == 0 {
			return nil, NewAgentRegistryError("AgentRegistry", "GetAgent",
				fmt.Sprintf("agent '%s' not found: no agents defined", name), nil)
		}

		availableAgents := make([]string, 0, len(allEntries))
		for _, e := range allEntries {
			availableAgents = append(availableAgents, e.Name)
		}

		return nil, NewAgentRegistryError("AgentRegistry", "GetAgent",
			fmt.Sprintf("agent '%s' not found\n\nAvailable agents:\n  - %s",
				name, strings.Join(availableAgents, "\n  - ")), nil)
	}
	return entry.Agent, nil
}

func (r *AgentRegistry) GetAllAgents() map[string]pb.A2AServiceServer {
	agents := make(map[string]pb.A2AServiceServer)

	for _, entry := range r.List() {
		agents[entry.Name] = entry.Agent
	}
	return agents
}

func (r *AgentRegistry) GetAgentConfig(name string) (*config.AgentConfig, error) {
	entry, exists := r.Get(name)
	if !exists {
		return nil, NewAgentRegistryError("AgentRegistry", "GetAgentConfig",
			fmt.Sprintf("agent config for %s not found", name), nil)
	}
	return entry.Config, nil
}

func (r *AgentRegistry) GetAgentsByType(agentType string) ([]pb.A2AServiceServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances, exists := r.instances[agentType]
	if !exists {
		return []pb.A2AServiceServer{}, nil
	}

	result := make([]pb.A2AServiceServer, len(instances))
	copy(result, instances)
	return result, nil
}

func (r *AgentRegistry) GetCapabilities(name string) ([]string, error) {
	entry, exists := r.Get(name)
	if !exists {
		return nil, NewAgentRegistryError("AgentRegistry", "GetCapabilities",
			fmt.Sprintf("capabilities for agent %s not found", name), nil)
	}
	return entry.Capabilities, nil
}

func (r *AgentRegistry) GetAgentsByCapability(capability string) ([]pb.A2AServiceServer, error) {
	if capability == "" {
		return nil, NewAgentRegistryError("AgentRegistry", "GetAgentsByCapability", "capability cannot be empty", nil)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []pb.A2AServiceServer
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

func (r *AgentRegistry) ListAgents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.List()))
	for _, entry := range r.List() {
		names = append(names, entry.Name)
	}

	return names
}

func (r *AgentRegistry) extractAgentType(name string) string {
	if underscoreIndex := strings.LastIndex(name, "_"); underscoreIndex > 0 {
		return name[:underscoreIndex]
	}
	return name
}
