package agent

import (
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

type RegistryService struct {
	registry *AgentRegistry
}

func (s *RegistryService) GetRegistry() *AgentRegistry {
	return s.registry
}

func NewRegistryService(registry *AgentRegistry) reasoning.AgentRegistryService {
	return &RegistryService{
		registry: registry,
	}
}

func (s *RegistryService) ListAgents() []reasoning.AgentRegistryEntry {
	if s.registry == nil {
		return nil
	}

	entries := s.registry.List()
	result := make([]reasoning.AgentRegistryEntry, 0, len(entries))

	for _, entry := range entries {

		var card *pb.AgentCard
		if agent, ok := entry.Agent.(*Agent); ok {
			card = agent.GetAgentCardSimple()
		} else {

			card = &pb.AgentCard{Name: entry.ID}
		}

		result = append(result, reasoning.AgentRegistryEntry{
			ID:         entry.ID,
			Card:       card,
			Visibility: entry.Config.Visibility,
		})
	}

	return result
}

func (s *RegistryService) GetAgent(id string) (reasoning.AgentRegistryEntry, bool) {
	if s.registry == nil {
		return reasoning.AgentRegistryEntry{}, false
	}

	entry, exists := s.registry.Get(id)
	if !exists {
		return reasoning.AgentRegistryEntry{}, false
	}

	var card *pb.AgentCard
	if agent, ok := entry.Agent.(*Agent); ok {
		card = agent.GetAgentCardSimple()
	} else {

		card = &pb.AgentCard{Name: entry.ID}
	}

	return reasoning.AgentRegistryEntry{
		ID:         entry.ID,
		Card:       card,
		Visibility: entry.Config.Visibility,
	}, true
}

func (s *RegistryService) FilterAgents(ids []string) []reasoning.AgentRegistryEntry {
	if s.registry == nil {
		return nil
	}

	if len(ids) == 0 {
		return s.ListAgents()
	}

	agents := make([]reasoning.AgentRegistryEntry, 0, len(ids))
	for _, id := range ids {
		if entry, exists := s.GetAgent(id); exists {
			agents = append(agents, entry)
		}
	}

	return agents
}

type NoOpRegistryService struct{}

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
