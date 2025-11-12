package config

import (
	"testing"
)

func TestLLMProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  LLMProviderConfig
		wantErr bool
	}{
		{
			name: "valid_openai_config",
			config: LLMProviderConfig{
				Type:        "openai",
				Model:       DefaultOpenAIModel,
				APIKey:      "sk-test-key",
				Host:        "https://api.openai.com/v1",
				Temperature: 0.7,
				MaxTokens:   4000,

				MaxRetries: 5,
				RetryDelay: 2,
			},
			wantErr: false,
		},
		{
			name: "valid_anthropic_config",
			config: LLMProviderConfig{
				Type:        "anthropic",
				Model:       "claude-3-5-sonnet",
				APIKey:      "sk-ant-test-key",
				Host:        "https://api.anthropic.com",
				Temperature: 0.8,
				MaxTokens:   8000,

				MaxRetries: 3,
				RetryDelay: 1,
			},
			wantErr: false,
		},
		{
			name: "valid_ollama_config",
			config: LLMProviderConfig{
				Type:        "ollama",
				Model:       "llama3",
				Host:        "http://localhost:11434",
				Temperature: 0.5,
				MaxTokens:   2000,
			},
			wantErr: false,
		},
		{
			name: "missing_type",
			config: LLMProviderConfig{
				Model: DefaultOpenAIModel,
				Host:  "https://api.openai.com/v1",
			},
			wantErr: true,
		},
		{
			name: "missing_model",
			config: LLMProviderConfig{
				Type: "openai",
				Host: "https://api.openai.com/v1",
			},
			wantErr: true,
		},
		{
			name: "missing_host",
			config: LLMProviderConfig{
				Type:  "openai",
				Model: DefaultOpenAIModel,
			},
			wantErr: true,
		},
		{
			name: "missing_api_key_for_openai",
			config: LLMProviderConfig{
				Type:  "openai",
				Model: DefaultOpenAIModel,
				Host:  "https://api.openai.com/v1",
			},
			wantErr: true,
		},
		{
			name: "invalid_temperature_too_low",
			config: LLMProviderConfig{
				Type:        "openai",
				Model:       DefaultOpenAIModel,
				Host:        "https://api.openai.com/v1",
				Temperature: -0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid_temperature_too_high",
			config: LLMProviderConfig{
				Type:        "openai",
				Model:       DefaultOpenAIModel,
				Host:        "https://api.openai.com/v1",
				Temperature: 2.1,
			},
			wantErr: true,
		},
		{
			name: "negative_max_tokens",
			config: LLMProviderConfig{
				Type:      "openai",
				Model:     DefaultOpenAIModel,
				Host:      "https://api.openai.com/v1",
				MaxTokens: -1,
			},
			wantErr: true,
		},
		{
			wantErr: true,
		},
		{
			name: "negative_max_retries",
			config: LLMProviderConfig{
				Type:       "openai",
				Model:      DefaultOpenAIModel,
				Host:       "https://api.openai.com/v1",
				MaxRetries: -1,
			},
			wantErr: true,
		},
		{
			name: "negative_retry_delay",
			config: LLMProviderConfig{
				Type:       "openai",
				Model:      DefaultOpenAIModel,
				Host:       "https://api.openai.com/v1",
				RetryDelay: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("LLMProviderConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDatabaseProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  DatabaseProviderConfig
		wantErr bool
	}{
		{
			name: "valid_qdrant_config",
			config: DatabaseProviderConfig{
				Type:   "qdrant",
				Host:   "localhost",
				Port:   6334,
				APIKey: "test-key",

				UseTLS: BoolPtr(true),
			},
			wantErr: false,
		},
		{
			name: "valid_qdrant_config_without_tls",
			config: DatabaseProviderConfig{
				Type: "qdrant",
				Host: "localhost",
				Port: 6334,

				UseTLS: BoolPtr(false),
			},
			wantErr: false,
		},
		{
			name: "missing_type",
			config: DatabaseProviderConfig{
				Host: "localhost",
				Port: 6334,
			},
			wantErr: true,
		},
		{
			name: "missing_host",
			config: DatabaseProviderConfig{
				Type: "qdrant",
				Port: 6334,
			},
			wantErr: true,
		},
		{
			name: "invalid_port_zero",
			config: DatabaseProviderConfig{
				Type: "qdrant",
				Host: "localhost",
				Port: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid_port_negative",
			config: DatabaseProviderConfig{
				Type: "qdrant",
				Host: "localhost",
				Port: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid_port_too_high",
			config: DatabaseProviderConfig{
				Type: "qdrant",
				Host: "localhost",
				Port: 70000,
			},
			wantErr: false,
		},
		{
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DatabaseProviderConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmbedderProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  EmbedderProviderConfig
		wantErr bool
	}{
		{
			name: "valid_ollama_config",
			config: EmbedderProviderConfig{
				Type:      "ollama",
				Model:     "nomic-embed-text",
				Host:      "http://localhost:11434",
				Dimension: 768,

				MaxRetries: 3,
			},
			wantErr: false,
		},
		{
			name: "missing_type",
			config: EmbedderProviderConfig{
				Model:     "nomic-embed-text",
				Host:      "http://localhost:11434",
				Dimension: 768,
			},
			wantErr: true,
		},
		{
			name: "missing_model",
			config: EmbedderProviderConfig{
				Type:      "ollama",
				Host:      "http://localhost:11434",
				Dimension: 768,
			},
			wantErr: true,
		},
		{
			name: "missing_host",
			config: EmbedderProviderConfig{
				Type:      "ollama",
				Model:     "nomic-embed-text",
				Dimension: 768,
			},
			wantErr: true,
		},
		{
			name: "invalid_dimension_zero",
			config: EmbedderProviderConfig{
				Type:      "ollama",
				Model:     "nomic-embed-text",
				Host:      "http://localhost:11434",
				Dimension: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid_dimension_negative",
			config: EmbedderProviderConfig{
				Type:      "ollama",
				Model:     "nomic-embed-text",
				Host:      "http://localhost:11434",
				Dimension: -1,
			},
			wantErr: true,
		},
		{
			wantErr: true,
		},
		{
			name: "negative_max_retries",
			config: EmbedderProviderConfig{
				Type:       "ollama",
				Model:      "nomic-embed-text",
				Host:       "http://localhost:11434",
				Dimension:  768,
				MaxRetries: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EmbedderProviderConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  AgentConfig
		wantErr bool
	}{
		{
			name: "valid_native_agent",
			config: AgentConfig{
				Type:        "native",
				Name:        "Test Agent",
				Description: "A test agent",
				LLM:         "test-llm",
				Visibility:  "public",
				Reasoning: ReasoningConfig{
					Engine:        "chain-of-thought",
					MaxIterations: 10,
				},
				Search: SearchConfig{
					Models: []SearchModel{
						{
							Name:        "documents",
							Collection:  "docs",
							DefaultTopK: 5,
							MaxTopK:     20,
						},
					},
					TopK:      5,
					Threshold: 0.7,
				},
			},
			wantErr: false,
		},
		{
			name: "valid_a2a_agent",
			config: AgentConfig{
				Type:        "a2a",
				Name:        "External Agent",
				Description: "An external A2A agent",
				URL:         "https://example.com/agents/test",
				Visibility:  "public",
			},
			wantErr: false,
		},
		{
			name: "valid_native_agent_with_document_stores",
			config: AgentConfig{
				Type:           "native",
				Name:           "Test Agent",
				LLM:            "test-llm",
				Database:       "test-db",
				Embedder:       "test-embedder",
				DocumentStores: []string{"test-store"},
				Reasoning: ReasoningConfig{
					Engine:        "chain-of-thought",
					MaxIterations: 10,
				},
				Search: SearchConfig{
					Models: []SearchModel{
						{
							Name:        "documents",
							Collection:  "docs",
							DefaultTopK: 5,
							MaxTopK:     20,
						},
					},
					TopK:      5,
					Threshold: 0.7,
				},
			},
			wantErr: false,
		},
		{
			name: "missing_name",
			config: AgentConfig{
				Type: "native",
				LLM:  "test-llm",
			},
			wantErr: true,
		},
		{
			name: "invalid_visibility",
			config: AgentConfig{
				Type:       "native",
				Name:       "Test Agent",
				LLM:        "test-llm",
				Visibility: "invalid",
			},
			wantErr: true,
		},
		{
			name: "a2a_agent_missing_url",
			config: AgentConfig{
				Type: "a2a",
				Name: "External Agent",
			},
			wantErr: true,
		},
		{
			name: "a2a_agent_with_llm",
			config: AgentConfig{
				Type: "a2a",
				Name: "External Agent",
				URL:  "https://example.com/agents/test",
				LLM:  "test-llm",
			},
			wantErr: true,
		},
		{
			name: "native_agent_missing_llm",
			config: AgentConfig{
				Type: "native",
				Name: "Test Agent",
			},
			wantErr: true,
		},
		{
			name: "native_agent_with_document_stores_missing_database",
			config: AgentConfig{
				Type:           "native",
				Name:           "Test Agent",
				LLM:            "test-llm",
				DocumentStores: []string{"test-store"},
			},
			wantErr: true,
		},
		{
			name: "native_agent_with_document_stores_missing_embedder",
			config: AgentConfig{
				Type:           "native",
				Name:           "Test Agent",
				LLM:            "test-llm",
				Database:       "test-db",
				DocumentStores: []string{"test-store"},
			},
			wantErr: true,
		},
		{
			name: "invalid_agent_type",
			config: AgentConfig{
				Type: "invalid",
				Name: "Test Agent",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AgentConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGlobalSettings_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  GlobalSettings
		wantErr bool
	}{
		{
			name: "valid_global_settings",
			config: GlobalSettings{
				Performance: PerformanceConfig{
					MaxConcurrency: 4,
				},
				A2AServer: A2AServerConfig{
					Host: "0.0.0.0",
					Port: 8080,
				},
				Auth: AuthConfig{
					JWKSURL:  "https://auth.example.com/.well-known/jwks.json",
					Issuer:   "https://auth.example.com/",
					Audience: "hector-api",
				},
			},
			wantErr: false,
		},
		{
			name: "valid_global_settings_disabled_auth",
			config: GlobalSettings{
				Performance: PerformanceConfig{
					MaxConcurrency: 4,
				},
				A2AServer: A2AServerConfig{},
				Auth:      AuthConfig{},
			},
			wantErr: false,
		},
		{
			name:    "invalid_logging_config",
			config:  GlobalSettings{},
			wantErr: true,
		},
		{
			name: "invalid_performance_config",
			config: GlobalSettings{
				Performance: PerformanceConfig{
					MaxConcurrency: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_a2a_server_config",
			config: GlobalSettings{
				A2AServer: A2AServerConfig{
					Port: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_auth_config_missing_jwks_url",
			config: GlobalSettings{
				Auth: AuthConfig{
					Issuer:   "https://auth.example.com/",
					Audience: "hector-api",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("GlobalSettings.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPerformanceConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  PerformanceConfig
		wantErr bool
	}{
		{
			name: "valid_performance_config",
			config: PerformanceConfig{
				MaxConcurrency: 4,
			},
			wantErr: false,
		},
		{
			name: "valid_high_concurrency",
			config: PerformanceConfig{
				MaxConcurrency: 16,
			},
			wantErr: false,
		},
		{
			name: "invalid_zero_concurrency",
			config: PerformanceConfig{
				MaxConcurrency: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid_negative_concurrency",
			config: PerformanceConfig{
				MaxConcurrency: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("PerformanceConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestA2AServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  A2AServerConfig
		wantErr bool
	}{
		{
			name: "valid_enabled_server",
			config: A2AServerConfig{
				Host: "0.0.0.0",
				Port: 8080,
			},
			wantErr: false,
		},
		{
			name:    "valid_disabled_server",
			config:  A2AServerConfig{},
			wantErr: false,
		},
		{
			name: "valid_custom_port",
			config: A2AServerConfig{
				Host: "localhost",
				Port: 3000,
			},
			wantErr: false,
		},
		{
			name: "invalid_port_zero",
			config: A2AServerConfig{
				Port: 0,
			},
			wantErr: false,
		},
		{
			name: "invalid_port_negative",
			config: A2AServerConfig{
				Port: -1,
			},
			wantErr: false,
		},
		{
			name: "invalid_port_too_high",
			config: A2AServerConfig{
				Port: 65536,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("A2AServerConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  AuthConfig
		wantErr bool
	}{
		{
			name: "valid_enabled_auth",
			config: AuthConfig{
				JWKSURL:  "https://auth.example.com/.well-known/jwks.json",
				Issuer:   "https://auth.example.com/",
				Audience: "hector-api",
			},
			wantErr: false,
		},
		{
			name:    "valid_disabled_auth",
			config:  AuthConfig{},
			wantErr: false,
		},
		{
			name: "missing_jwks_url",
			config: AuthConfig{
				Issuer:   "https://auth.example.com/",
				Audience: "hector-api",
			},
			wantErr: false,
		},
		{
			name: "missing_issuer",
			config: AuthConfig{
				JWKSURL:  "https://auth.example.com/.well-known/jwks.json",
				Audience: "hector-api",
			},
			wantErr: false,
		},
		{
			name: "missing_audience",
			config: AuthConfig{
				JWKSURL: "https://auth.example.com/.well-known/jwks.json",
				Issuer:  "https://auth.example.com/",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
