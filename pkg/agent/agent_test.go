package agent

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
)

func TestNewAgent(t *testing.T) {
	tests := []struct {
		name         string
		agentConfig  *config.AgentConfig
		componentMgr interface{}
		wantErr      bool
	}{
		{
			name:         "nil agent config",
			agentConfig:  nil,
			componentMgr: &MockComponentManager{},
			wantErr:      true,
		},
		{
			name: "nil component manager",
			agentConfig: &config.AgentConfig{
				Name: "Test Agent",
				LLM:  "test-llm",
			},
			componentMgr: nil,
			wantErr:      true,
		},
		{
			name: "invalid component manager type",
			agentConfig: &config.AgentConfig{
				Name: "Test Agent",
				LLM:  "test-llm",
			},
			componentMgr: "invalid",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewAgent("test-agent", tt.agentConfig, tt.componentMgr, nil, "http://localhost:8080")
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && agent == nil {
				t.Error("NewAgent() returned nil agent without error")
			}
		})
	}
}

func TestAgent_GetAgentCard(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

func TestAgent_ExecuteTask(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

func TestAgent_ExecuteTaskStreaming(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

func TestAgent_ClearHistory(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

func TestAgent_GetName(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

func TestAgent_GetDescription(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

func TestAgent_GetConfig(t *testing.T) {

	t.Skip("Skipping test that requires full component manager setup")
}

type MockComponentManager struct{}
