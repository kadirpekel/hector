package llmagent

import (
	"context"
	"iter"
	"testing"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/verikod/hector/pkg/agent"
)

// MockMemory implements agent.Memory
type MockMemory struct {
	SearchResults []agent.MemoryResult
	SearchQuery   string
	SearchLimit   int
}

func (m *MockMemory) AddSession(ctx context.Context, session agent.Session) error {
	return nil
}

func (m *MockMemory) Search(ctx context.Context, query string) (*agent.MemorySearchResponse, error) {
	m.SearchQuery = query
	// Note: Generic interface doesn't pass limit to Search currently?
	// Checking agent.Memory interface in context.go: Search(ctx, query)
	return &agent.MemorySearchResponse{Results: m.SearchResults}, nil
}

// MockEvents implements agent.Events
type MockEvents struct {
	events []*agent.Event
}

func (e *MockEvents) All() iter.Seq[*agent.Event] {
	return func(yield func(*agent.Event) bool) {
		for _, evt := range e.events {
			if !yield(evt) {
				return
			}
		}
	}
}

func (e *MockEvents) Len() int {
	return len(e.events)
}

func (e *MockEvents) At(i int) *agent.Event {
	return e.events[i]
}

// MockSession implements agent.Session
type MockSession struct {
	events *MockEvents
}

func (s *MockSession) ID() string      { return "test-session" }
func (s *MockSession) AppName() string { return "test-app" }
func (s *MockSession) UserID() string  { return "test-user" }
func (s *MockSession) State() agent.State {
	return nil // Not needed for this test
}
func (s *MockSession) Events() agent.Events {
	return s.events
}

// MockAgent implements agent.Agent
type MockAgent struct {
	name string
}

func (a *MockAgent) Name() string                                               { return a.name }
func (a *MockAgent) DisplayName() string                                        { return a.name }
func (a *MockAgent) Description() string                                        { return "" }
func (a *MockAgent) Run(agent.InvocationContext) iter.Seq2[*agent.Event, error] { return nil }
func (a *MockAgent) SubAgents() []agent.Agent                                   { return nil }
func (a *MockAgent) Type() agent.AgentType                                      { return agent.TypeLLMAgent }

func TestAutoRecall(t *testing.T) {
	// Setup
	mockMemory := &MockMemory{
		SearchResults: []agent.MemoryResult{
			{Content: "User likes cats", Score: 0.9},
			{Content: "User name is Alice", Score: 0.8},
		},
	}

	// Create user message event
	evt := agent.NewEvent("test-invocation")
	evt.Artifact = &a2a.Artifact{Parts: []a2a.Part{a2a.TextPart{Text: "What do I like?"}}}
	evt.Author = agent.AuthorUser
	evt.Timestamp = time.Now()

	mockEvents := &MockEvents{
		events: []*agent.Event{evt},
	}

	mockSession := &MockSession{events: mockEvents}

	// Create context
	ctx := agent.NewInvocationContext(context.Background(), agent.InvocationContextParams{
		Session: mockSession,
		Memory:  mockMemory,
		Branch:  "",
	})

	// Create agent with auto-recall enabled
	a := &llmAgent{
		Agent:           &MockAgent{name: "test-agent"},
		autoRecall:      true,
		autoRecallLimit: 5,
		// Needs workingMemory or it might panic if logic depends on it?
		// Logic: "if a.workingMemory != nil". If nil, it uses raw events.
	}

	// 2. Filter Events logic in buildMessages accesses a.includeContents
	// a.includeContents default is 0 (IncludeContentsAll)? No, need to check default.
	// But lines 440+ check includeContents.

	// Run buildMessages
	messages := a.buildMessages(ctx)

	// Verify
	// Expectation:
	// 1. Memory search was triggered
	if mockMemory.SearchQuery != "What do I like?" {
		t.Errorf("Expected search query 'What do I like?', got '%s'", mockMemory.SearchQuery)
	}

	// 2. Messages should contain memory context (User role) + User message
	// The memory context is PREPENDED.
	if len(messages) < 2 {
		t.Fatalf("Expected at least 2 messages, got %d", len(messages))
	}

	firstMsg := messages[0]
	// Check role
	if firstMsg.Role != a2a.MessageRoleUser {
		t.Errorf("Expected first message role User, got %s", firstMsg.Role)
	}

	// Check content
	content := processMessageContent(firstMsg)
	if content == "" {
		t.Error("First message content is empty")
	}
	if !contains(content, "Retrieved relevant memories") {
		t.Errorf("Expected memory header in content, got: %s", content)
	}
	if !contains(content, "User likes cats") {
		t.Errorf("Expected recalled memory 'User likes cats', got: %s", content)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr ||
		len(s) > len(substr) && contains(s[1:], substr)
}
