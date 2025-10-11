package agent

import (
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// ============================================================================
// AGENT REGISTRY SERVICE IMPLEMENTATION
// Implements reasoning.AgentRegistryService interface
// Returns A2A protocol AgentCard types for true A2A-native architecture
// ============================================================================

// RegistryService wraps AgentRegistry to implement the service interface
// This provides a clean abstraction for reasoning strategies
type RegistryService struct {
	registry *AgentRegistry
}

// NewRegistryService creates a new agent registry service
func NewRegistryService(registry *AgentRegistry) reasoning.AgentRegistryService {
	return &RegistryService{
		registry: registry,
	}
}

// ListAgents returns all available agents with their A2A AgentCards
func (s *RegistryService) ListAgents() []reasoning.AgentRegistryEntry {
	if s.registry == nil {
		return nil
	}

	entries := s.registry.List()
	result := make([]reasoning.AgentRegistryEntry, 0, len(entries))

	for _, entry := range entries {
		// Get A2A AgentCard from the agent
		card := entry.Agent.GetAgentCard()

		result = append(result, reasoning.AgentRegistryEntry{
			ID:         entry.Name,
			Card:       card, // A2A protocol AgentCard
			Visibility: entry.Config.Visibility,
		})
	}

	return result
}

// GetAgent returns agent entry for a specific agent
func (s *RegistryService) GetAgent(id string) (reasoning.AgentRegistryEntry, bool) {
	if s.registry == nil {
		return reasoning.AgentRegistryEntry{}, false
	}

	entry, exists := s.registry.Get(id)
	if !exists {
		return reasoning.AgentRegistryEntry{}, false
	}

	// Get A2A AgentCard from the agent
	card := entry.Agent.GetAgentCard()

	return reasoning.AgentRegistryEntry{
		ID:         entry.Name,
		Card:       card, // A2A protocol AgentCard
		Visibility: entry.Config.Visibility,
	}, true
}

// FilterAgents returns agents matching the given IDs
// If ids is empty, returns all agents
func (s *RegistryService) FilterAgents(ids []string) []reasoning.AgentRegistryEntry {
	if s.registry == nil {
		return nil
	}

	// If no filter, return all
	if len(ids) == 0 {
		return s.ListAgents()
	}

	// Filter to specified IDs
	agents := make([]reasoning.AgentRegistryEntry, 0, len(ids))
	for _, id := range ids {
		if entry, exists := s.GetAgent(id); exists {
			agents = append(agents, entry)
		}
	}

	return agents
}

// NoOpRegistryService is a nil-safe implementation for single-agent mode
type NoOpRegistryService struct{}

// NewNoOpRegistryService creates a registry service that returns empty results
func NewNoOpRegistryService() reasoning.AgentRegistryService {
	return &NoOpRegistryService{}
}

func (s *NoOpRegistryService) ListAgents() []reasoning.AgentRegistryEntry {
	return nil
}

func (s *NoOpRegistryService) GetAgent(id string) (reasoning.AgentRegistryEntry, bool) {
	return reasoning.AgentRegistryEntry{}, false
}

func (s *NoOpRegistryService) FilterAgents(ids []string) []reasoning.AgentRegistryEntry {
	return nil
}
