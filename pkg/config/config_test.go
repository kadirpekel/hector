package config

import (
	"testing"
)

func TestConfig_AgentAccess(t *testing.T) {
	config := &Config{
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name: "Test Agent",
				LLM:  "test-llm",
			},
		},
	}

	if agent, exists := config.Agents["test-agent"]; !exists {
		t.Error("Expected agent 'test-agent' to exist")
	} else if agent.Name != "Test Agent" {
		t.Errorf("Agent name = %v, want %v", agent.Name, "Test Agent")
	}

	if _, exists := config.Agents["non-existing"]; exists {
		t.Error("Expected agent 'non-existing' to not exist")
	}
}

func TestConfig_LLMAccess(t *testing.T) {
	config := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:  "openai",
				Model: "gpt-4o-mini",
			},
		},
	}

	if llm, exists := config.LLMs["test-llm"]; !exists {
		t.Error("Expected LLM 'test-llm' to exist")
	} else if llm.Type != "openai" {
		t.Errorf("LLM type = %v, want %v", llm.Type, "openai")
	}

	if _, exists := config.LLMs["non-existing"]; exists {
		t.Error("Expected LLM 'non-existing' to not exist")
	}
}

func TestConfig_AgentCount(t *testing.T) {
	config := &Config{
		Agents: map[string]*AgentConfig{
			"agent1": {Name: "Agent 1", LLM: "llm1"},
			"agent2": {Name: "Agent 2", LLM: "llm2"},
			"agent3": {Name: "Agent 3", LLM: "llm3"},
		},
	}

	if got := len(config.Agents); got != 3 {
		t.Errorf("Config.Agents length = %v, want %v", got, 3)
	}
}

func TestConfig_LLMCount(t *testing.T) {
	config := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"llm1": {Type: "openai", Model: "gpt-4o-mini"},
			"llm2": {Type: "anthropic", Model: "claude-3-5-sonnet"},
		},
	}

	if got := len(config.LLMs); got != 2 {
		t.Errorf("Config.LLMs length = %v, want %v", got, 2)
	}
}

func TestConfig_EmptyConfig(t *testing.T) {
	config := &Config{}

	if len(config.Agents) != 0 {
		t.Errorf("Empty config should have 0 agents, got %v", len(config.Agents))
	}

	if len(config.LLMs) != 0 {
		t.Errorf("Empty config should have 0 LLMs, got %v", len(config.LLMs))
	}
}
