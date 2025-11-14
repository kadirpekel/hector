package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/registry"
	"google.golang.org/grpc/metadata"
)

type AgentEntry struct {
	Agent        pb.A2AServiceServer `json:"agent"`
	Config       *config.AgentConfig `json:"config"`
	Capabilities []string            `json:"capabilities"`
	AgentType    string              `json:"agent_type"`
	ID           string              `json:"id"` // Agent ID (config key, URL-safe)
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

func (r *AgentRegistry) RegisterAgent(agentID string, agent pb.A2AServiceServer, agentConfig *config.AgentConfig, capabilities []string) error {
	if agentID == "" {
		return NewAgentRegistryError("AgentRegistry", "RegisterAgent", "agent ID cannot be empty", nil)
	}
	if agent == nil {
		return NewAgentRegistryError("AgentRegistry", "RegisterAgent", "agent cannot be nil", nil)
	}
	if agentConfig == nil {
		return NewAgentRegistryError("AgentRegistry", "RegisterAgent", "agent config cannot be nil", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	agentType := r.extractAgentType(agentID)

	entry := AgentEntry{
		Agent:        agent,
		Config:       agentConfig,
		Capabilities: capabilities,
		AgentType:    agentType,
		ID:           agentID,
	}

	if err := r.Register(agentID, entry); err != nil {
		return NewAgentRegistryError("AgentRegistry", "RegisterAgent",
			fmt.Sprintf("failed to register agent %s", agentID), err)
	}

	if r.instances[agentType] == nil {
		r.instances[agentType] = make([]pb.A2AServiceServer, 0)
	}
	r.instances[agentType] = append(r.instances[agentType], agent)

	return nil
}

func (r *AgentRegistry) GetAgent(agentID string) (pb.A2AServiceServer, error) {
	entry, exists := r.Get(agentID)
	if !exists {

		allEntries := r.List()
		if len(allEntries) == 0 {
			return nil, NewAgentRegistryError("AgentRegistry", "GetAgent",
				fmt.Sprintf("agent '%s' not found: no agents defined", agentID), nil)
		}

		availableAgents := make([]string, 0, len(allEntries))
		for _, e := range allEntries {
			availableAgents = append(availableAgents, e.ID)
		}

		return nil, NewAgentRegistryError("AgentRegistry", "GetAgent",
			fmt.Sprintf("agent '%s' not found\n\nAvailable agents:\n  - %s",
				agentID, strings.Join(availableAgents, "\n  - ")), nil)
	}
	return entry.Agent, nil
}

// StreamMessage streams messages from a local agent for true streaming support.
// This enables agent_call to stream responses from local agents in real-time.
func (r *AgentRegistry) StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error) {
	entry, exists := r.Get(agentID)
	if !exists {
		return nil, NewAgentRegistryError("AgentRegistry", "StreamMessage",
			fmt.Sprintf("agent '%s' not found", agentID), nil)
	}

	req := &pb.SendMessageRequest{
		Request: message,
	}

	streamChan := make(chan *pb.StreamResponse, 10)

	// Create a streaming server wrapper similar to runtime/local.go
	stream := &localStreamWrapper{
		ctx:  ctx,
		send: streamChan,
	}

	go func() {
		defer close(streamChan)
		if err := entry.Agent.SendStreamingMessage(req, stream); err != nil {
			// Log error but don't return it - channel is already closed
			// The error will be visible in the stream response
			slog.Error("Failed to stream message from agent", "agent", agentID, "error", err)
		}
	}()

	return streamChan, nil
}

// localStreamWrapper wraps a channel to implement the gRPC streaming server interface
// Similar to runtime/local.go's localStream implementation
type localStreamWrapper struct {
	ctx  context.Context
	send chan<- *pb.StreamResponse
}

func (s *localStreamWrapper) Send(resp *pb.StreamResponse) error {
	select {
	case s.send <- resp:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *localStreamWrapper) Context() context.Context {
	return s.ctx
}

func (s *localStreamWrapper) SendMsg(m interface{}) error {
	return nil
}

func (s *localStreamWrapper) RecvMsg(m interface{}) error {
	return nil
}

func (s *localStreamWrapper) SendHeader(_ metadata.MD) error {
	return nil
}

func (s *localStreamWrapper) SetHeader(_ metadata.MD) error {
	return nil
}

func (s *localStreamWrapper) SetTrailer(_ metadata.MD) {
}

func (r *AgentRegistry) GetAllAgents() map[string]pb.A2AServiceServer {
	agents := make(map[string]pb.A2AServiceServer)

	for _, entry := range r.List() {
		agents[entry.ID] = entry.Agent
	}
	return agents
}

func (r *AgentRegistry) GetAgentConfig(agentID string) (*config.AgentConfig, error) {
	entry, exists := r.Get(agentID)
	if !exists {
		return nil, NewAgentRegistryError("AgentRegistry", "GetAgentConfig",
			fmt.Sprintf("agent config for %s not found", agentID), nil)
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

func (r *AgentRegistry) GetCapabilities(agentID string) ([]string, error) {
	entry, exists := r.Get(agentID)
	if !exists {
		return nil, NewAgentRegistryError("AgentRegistry", "GetCapabilities",
			fmt.Sprintf("capabilities for agent %s not found", agentID), nil)
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

	ids := make([]string, 0, len(r.List()))
	for _, entry := range r.List() {
		ids = append(ids, entry.ID)
	}

	return ids
}

func (r *AgentRegistry) extractAgentType(agentID string) string {
	if underscoreIndex := strings.LastIndex(agentID, "_"); underscoreIndex > 0 {
		return agentID[:underscoreIndex]
	}
	return agentID
}
