package config

import (
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid_minimal_config",
			config: &Config{
				Agents: map[string]*AgentConfig{
					"test-agent": {
						Name: "Test Agent",
						LLM:  "test-llm",
					},
				},
				LLMs: map[string]*LLMProviderConfig{
					"test-llm": {
						Type:   "openai",
						Model:  DefaultOpenAIModel,
						Host:   "https://api.openai.com/v1",
						APIKey: "sk-test-key",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_complete_config",
			config: &Config{
				Agents: map[string]*AgentConfig{
					"test-agent": {
						Name: "Test Agent",
						LLM:  "test-llm",
					},
				},
				LLMs: map[string]*LLMProviderConfig{
					"test-llm": {
						Type:   "openai",
						Model:  DefaultOpenAIModel,
						Host:   "https://api.openai.com/v1",
						APIKey: "sk-test-key",
					},
				},
				Global: GlobalSettings{
					Performance: PerformanceConfig{
						MaxConcurrency: 4,
					},
					A2AServer: A2AServerConfig{
						Host: "0.0.0.0",
						Port: 8080,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid_agent_config",
			config: &Config{
				Agents: map[string]*AgentConfig{
					"test-agent": {
						Name: "",
						LLM:  "test-llm",
					},
				},
				LLMs: map[string]*LLMProviderConfig{
					"test-llm": {
						Type:  "openai",
						Model: DefaultOpenAIModel,
						Host:  "https://api.openai.com/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_llm_config",
			config: &Config{
				Agents: map[string]*AgentConfig{
					"test-agent": {
						Name: "Test Agent",
						LLM:  "test-llm",
					},
				},
				LLMs: map[string]*LLMProviderConfig{
					"test-llm": {
						Type:  "",
						Model: DefaultOpenAIModel,
						Host:  "https://api.openai.com/v1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_global_settings",
			config: &Config{
				Agents: map[string]*AgentConfig{
					"test-agent": {
						Name: "Test Agent",
						LLM:  "test-llm",
					},
				},
				LLMs: map[string]*LLMProviderConfig{
					"test-llm": {
						Type:  "openai",
						Model: DefaultOpenAIModel,
						Host:  "https://api.openai.com/v1",
					},
				},
				Global: GlobalSettings{
					Performance: PerformanceConfig{
						MaxConcurrency: -1, // Invalid
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "empty_config",
			config:  &Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if !tt.wantErr {
				tt.config.SetDefaults()
			}

			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		validateConfig func(t *testing.T, config *Config)
	}{
		{
			name:   "empty_config",
			config: &Config{},
			validateConfig: func(t *testing.T, config *Config) {

				if len(config.LLMs) == 0 {
					t.Error("SetDefaults() should create default LLM")
				}
				if len(config.VectorStores) == 0 {
					t.Error("SetDefaults() should create default vector store")
				}
				if len(config.Embedders) == 0 {
					t.Error("SetDefaults() should create default embedder")
				}
				if len(config.Agents) == 0 {
					t.Error("SetDefaults() should create default agent")
				}

				if llm, exists := config.LLMs["default-llm"]; exists {
					if llm.Type != "openai" {
						t.Errorf("Default LLM type = %v, want %v", llm.Type, "openai")
					}
					if llm.Model != DefaultOpenAIModel {
						t.Errorf("Default LLM model = %v, want %v", llm.Model, DefaultOpenAIModel)
					}
				} else {
					t.Error("Default LLM 'default-llm' should exist")
				}

				if agent, exists := config.Agents["default-agent"]; exists {
					// Name is now set in Validate() to use agent ID, not in SetDefaults()
					if agent.Name != "" {
						t.Errorf("SetDefaults should leave name empty (set in Validate), got = %v", agent.Name)
					}
					if agent.LLM != "default-llm" {
						t.Errorf("Default agent LLM = %v, want %v", agent.LLM, "default-llm")
					}
				} else {
					t.Error("Default agent 'default-agent' should exist")
				}
			},
		},
		{
			name: "config_with_existing_services",
			config: &Config{
				LLMs: map[string]*LLMProviderConfig{
					"custom-llm": {
						Type:  "anthropic",
						Model: "claude-3-5-sonnet",
					},
				},
				Agents: map[string]*AgentConfig{
					"custom-agent": {
						Name: "Custom Agent",
						LLM:  "custom-llm",
					},
				},
			},
			validateConfig: func(t *testing.T, config *Config) {

				if len(config.LLMs) != 1 {
					t.Errorf("Should have 1 LLM, got %d", len(config.LLMs))
				}
				if len(config.Agents) != 1 {
					t.Errorf("Should have 1 agent, got %d", len(config.Agents))
				}

				if len(config.VectorStores) == 0 {
					t.Error("SetDefaults() should create default vector store")
				}
				if len(config.Embedders) == 0 {
					t.Error("SetDefaults() should create default embedder")
				}
			},
		},
		{
			name: "config_with_partial_llm",
			config: &Config{
				LLMs: map[string]*LLMProviderConfig{
					"partial-llm": {
						Type: "openai",
					},
				},
			},
			validateConfig: func(t *testing.T, config *Config) {
				if llm, exists := config.LLMs["partial-llm"]; exists {
					if llm.Model == "" {
						t.Error("SetDefaults() should set default model for LLM")
					}
					if llm.Host == "" {
						t.Error("SetDefaults() should set default host for LLM")
					}
					if llm.Temperature == nil || *llm.Temperature == 0 {
						t.Error("SetDefaults() should set default temperature for LLM")
					}
				} else {
					t.Error("Partial LLM should still exist after SetDefaults")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.SetDefaults()
			tt.validateConfig(t, tt.config)
		})
	}
}

func TestConfig_HelperMethods(t *testing.T) {
	config := &Config{
		Agents: map[string]*AgentConfig{
			"agent1": {Name: "Agent 1", LLM: "llm1"},
			"agent2": {Name: "Agent 2", LLM: "llm2"},
		},
		DocumentStores: map[string]*DocumentStoreConfig{
			"store1": {Source: "directory", Path: "./docs"},
			"store2": {Source: "directory", Path: "./data"},
		},
	}

	t.Run("GetAgent", func(t *testing.T) {

		agent, exists := config.GetAgent("agent1")
		if !exists {
			t.Error("GetAgent() should return true for existing agent")
		}
		if agent.Name != "Agent 1" {
			t.Errorf("GetAgent() name = %v, want %v", agent.Name, "Agent 1")
		}

		_, exists = config.GetAgent("non-existing")
		if exists {
			t.Error("GetAgent() should return false for non-existing agent")
		}
	})

	t.Run("GetDocumentStore", func(t *testing.T) {

		store, exists := config.GetDocumentStore("store1")
		if !exists {
			t.Error("GetDocumentStore() should return true for existing store")
		}
		if store == nil {
			t.Error("GetDocumentStore() store should not be nil")
		}

		_, exists = config.GetDocumentStore("non-existing")
		if exists {
			t.Error("GetDocumentStore() should return false for non-existing store")
		}
	})

	t.Run("ListAgents", func(t *testing.T) {
		agents := config.ListAgents()
		if len(agents) != 2 {
			t.Errorf("ListAgents() length = %v, want %v", len(agents), 2)
		}

		agentMap := make(map[string]bool)
		for _, agent := range agents {
			agentMap[agent] = true
		}
		if !agentMap["agent1"] || !agentMap["agent2"] {
			t.Error("ListAgents() should contain both agents")
		}
	})

	t.Run("ListDocumentStores", func(t *testing.T) {
		stores := config.ListDocumentStores()
		if len(stores) != 2 {
			t.Errorf("ListDocumentStores() length = %v, want %v", len(stores), 2)
		}

		storeMap := make(map[string]bool)
		for _, store := range stores {
			storeMap[store] = true
		}
		if !storeMap["store1"] || !storeMap["store2"] {
			t.Error("ListDocumentStores() should contain both stores")
		}
	})
}

func TestConfig_EmptyMaps(t *testing.T) {
	config := &Config{
		Agents:         make(map[string]*AgentConfig),
		DocumentStores: make(map[string]*DocumentStoreConfig),
	}

	t.Run("EmptyAgents", func(t *testing.T) {
		agents := config.ListAgents()
		if len(agents) != 0 {
			t.Errorf("ListAgents() length = %v, want %v", len(agents), 0)
		}

		_, exists := config.GetAgent("non-existing")
		if exists {
			t.Error("GetAgent() should return false for empty map")
		}
	})

	t.Run("EmptyDocumentStores", func(t *testing.T) {
		stores := config.ListDocumentStores()
		if len(stores) != 0 {
			t.Errorf("ListDocumentStores() length = %v, want %v", len(stores), 0)
		}

		_, exists := config.GetDocumentStore("non-existing")
		if exists {
			t.Error("GetDocumentStore() should return false for empty map")
		}
	})
}

func TestConfig_NilMaps(t *testing.T) {
	config := &Config{
		Agents:         nil,
		DocumentStores: nil,
	}

	t.Run("NilAgents", func(t *testing.T) {
		agents := config.ListAgents()
		if len(agents) != 0 {
			t.Errorf("ListAgents() length = %v, want %v", len(agents), 0)
		}

		_, exists := config.GetAgent("non-existing")
		if exists {
			t.Error("GetAgent() should return false for nil map")
		}
	})

	t.Run("NilDocumentStores", func(t *testing.T) {
		stores := config.ListDocumentStores()
		if len(stores) != 0 {
			t.Errorf("ListDocumentStores() length = %v, want %v", len(stores), 0)
		}

		_, exists := config.GetDocumentStore("non-existing")
		if exists {
			t.Error("GetDocumentStore() should return false for nil map")
		}
	})
}
