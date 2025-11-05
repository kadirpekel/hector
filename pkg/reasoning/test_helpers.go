package reasoning

import "github.com/kadirpekel/hector/pkg/config"

// mockAgentServices is a minimal mock for testing
type mockAgentServices struct{}

func (m *mockAgentServices) GetConfig() config.ReasoningConfig {
	return config.ReasoningConfig{
		Engine:        "chain-of-thought",
		MaxIterations: 10,
	}
}

func (m *mockAgentServices) LLM() LLMService                { return nil }
func (m *mockAgentServices) Tools() ToolService             { return nil }
func (m *mockAgentServices) Context() ContextService        { return nil }
func (m *mockAgentServices) Prompt() PromptService          { return nil }
func (m *mockAgentServices) Session() SessionService        { return nil }
func (m *mockAgentServices) History() HistoryService        { return nil }
func (m *mockAgentServices) Registry() AgentRegistryService { return nil }
func (m *mockAgentServices) Task() TaskService              { return nil }
