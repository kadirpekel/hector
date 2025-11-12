package config

import (
	"os"
	"testing"
)

func TestLLMProviderConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		config         LLMProviderConfig
		envVars        map[string]string
		validateConfig func(t *testing.T, config LLMProviderConfig)
	}{
		{
			name:   "empty_config_openai_defaults",
			config: LLMProviderConfig{},
			validateConfig: func(t *testing.T, config LLMProviderConfig) {
				if config.Type != "openai" {
					t.Errorf("Default type = %v, want %v", config.Type, "openai")
				}
				if config.Model != DefaultOpenAIModel {
					t.Errorf("Default model = %v, want %v", config.Model, DefaultOpenAIModel)
				}
				if config.Host != "https://api.openai.com/v1" {
					t.Errorf("Default host = %v, want %v", config.Host, "https://api.openai.com/v1")
				}
				if config.Temperature != 0.7 {
					t.Errorf("Default temperature = %v, want %v", config.Temperature, 0.7)
				}
				if config.MaxTokens != 8000 {
					t.Errorf("Default max_tokens = %v, want %v", config.MaxTokens, 8000)
				}
				// Timeout field was removed
				// if config.Timeout != 60 {
				// 	t.Errorf("Default timeout = %v, want %v", config.Timeout, 60)
				// }
				if config.MaxRetries != 5 {
					t.Errorf("Default max_retries = %v, want %v", config.MaxRetries, 5)
				}
				if config.RetryDelay != 2 {
					t.Errorf("Default retry_delay = %v, want %v", config.RetryDelay, 2)
				}
			},
		},
		{
			name: "partial_config_preserves_values",
			config: LLMProviderConfig{
				Type:  "anthropic",
				Model: "claude-3-5-sonnet",
			},
			validateConfig: func(t *testing.T, config LLMProviderConfig) {
				if config.Type != "anthropic" {
					t.Errorf("Type should be preserved: %v", config.Type)
				}
				if config.Model != "claude-3-5-sonnet" {
					t.Errorf("Model should be preserved: %v", config.Model)
				}
				if config.Host != "https://api.anthropic.com" {
					t.Errorf("Default host for anthropic = %v, want %v", config.Host, "https://api.anthropic.com")
				}
				if config.Temperature != 0.7 {
					t.Errorf("Default temperature = %v, want %v", config.Temperature, 0.7)
				}
			},
		},
		{
			name: "anthropic_type_defaults",
			config: LLMProviderConfig{
				Type: "anthropic",
			},
			validateConfig: func(t *testing.T, config LLMProviderConfig) {
				if config.Model != "claude-3-7-sonnet-latest" {
					t.Errorf("Default anthropic model = %v, want %v", config.Model, "claude-3-7-sonnet-latest")
				}
				if config.Host != "https://api.anthropic.com" {
					t.Errorf("Default anthropic host = %v, want %v", config.Host, "https://api.anthropic.com")
				}
			},
		},
		{
			name: "api_key_from_environment_openai",
			config: LLMProviderConfig{
				Type: "openai",
			},
			envVars: map[string]string{
				"OPENAI_API_KEY": "sk-test-key-123",
			},
			validateConfig: func(t *testing.T, config LLMProviderConfig) {
				if config.APIKey != "sk-test-key-123" {
					t.Errorf("API key from env = %v, want %v", config.APIKey, "sk-test-key-123")
				}
			},
		},
		{
			name: "api_key_from_environment_anthropic",
			config: LLMProviderConfig{
				Type: "anthropic",
			},
			envVars: map[string]string{
				"ANTHROPIC_API_KEY": "sk-ant-test-key-456",
			},
			validateConfig: func(t *testing.T, config LLMProviderConfig) {
				if config.APIKey != "sk-ant-test-key-456" {
					t.Errorf("API key from env = %v, want %v", config.APIKey, "sk-ant-test-key-456")
				}
			},
		},
		{
			name: "zero_values_set_defaults",
			config: LLMProviderConfig{
				Type:        "openai",
				Model:       DefaultOpenAIModel,
				Host:        "https://api.openai.com/v1",
				Temperature: 0,
				MaxTokens:   0,
				Timeout:     0,
			},
			validateConfig: func(t *testing.T, config LLMProviderConfig) {
				if config.Temperature != 0.7 {
					t.Errorf("Zero temperature should be set to default: %v", config.Temperature)
				}
				if config.MaxTokens != 8000 {
					t.Errorf("Zero max_tokens should be set to default: %v", config.MaxTokens)
				}
				// Timeout field was removed
				// if config.Timeout != 60 {
				// 	t.Errorf("Zero timeout should be set to default: %v", config.Timeout)
				// }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			tt.config.SetDefaults()
			tt.validateConfig(t, tt.config)
		})
	}
}

func TestDatabaseProviderConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		config         DatabaseProviderConfig
		validateConfig func(t *testing.T, config DatabaseProviderConfig)
	}{
		{
			name:   "empty_config_qdrant_defaults",
			config: DatabaseProviderConfig{},
			validateConfig: func(t *testing.T, config DatabaseProviderConfig) {
				if config.Type != "qdrant" {
					t.Errorf("Default type = %v, want %v", config.Type, "qdrant")
				}
				if config.Host != "localhost" {
					t.Errorf("Default host = %v, want %v", config.Host, "localhost")
				}
				if config.Port != 6334 {
					t.Errorf("Default port = %v, want %v", config.Port, 6334)
				}
				// Timeout field was removed from DatabaseProviderConfig
				// if config.Timeout != 30 {
				// 	t.Errorf("Default timeout = %v, want %v", config.Timeout, 30)
				// }
			},
		},
		{
			name: "partial_config_preserves_values",
			config: DatabaseProviderConfig{
				Type: "custom",
				Host: "custom-host",
			},
			validateConfig: func(t *testing.T, config DatabaseProviderConfig) {
				if config.Type != "custom" {
					t.Errorf("Type should be preserved: %v", config.Type)
				}
				if config.Host != "custom-host" {
					t.Errorf("Host should be preserved: %v", config.Host)
				}
				if config.Port != 6334 {
					t.Errorf("Default port = %v, want %v", config.Port, 6334)
				}
				// Timeout field was removed
				// if config.Timeout != 30 {
				// 	t.Errorf("Default timeout = %v, want %v", config.Timeout, 30)
				// }
			},
		},
		{
			name: "zero_values_set_defaults",
			config: DatabaseProviderConfig{
				Type: "qdrant",
				Host: "localhost",
				Port: 0,
				// Timeout field was removed from DatabaseProviderConfig
			},
			validateConfig: func(t *testing.T, config DatabaseProviderConfig) {
				if config.Port != 6334 {
					t.Errorf("Zero port should be set to default: %v", config.Port)
				}
				// Timeout field was removed
				// if config.Timeout != 30 {
				// 	t.Errorf("Zero timeout should be set to default: %v", config.Timeout)
				// }
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

func TestEmbedderProviderConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		config         EmbedderProviderConfig
		validateConfig func(t *testing.T, config EmbedderProviderConfig)
	}{
		{
			name:   "empty_config_ollama_defaults",
			config: EmbedderProviderConfig{},
			validateConfig: func(t *testing.T, config EmbedderProviderConfig) {
				if config.Type != "ollama" {
					t.Errorf("Default type = %v, want %v", config.Type, "ollama")
				}
				if config.Model != "nomic-embed-text" {
					t.Errorf("Default model = %v, want %v", config.Model, "nomic-embed-text")
				}
				if config.Host != "http://localhost:11434" {
					t.Errorf("Default host = %v, want %v", config.Host, "http://localhost:11434")
				}
				if config.Dimension != 768 {
					t.Errorf("Default dimension = %v, want %v", config.Dimension, 768)
				}
				// Timeout field was removed from EmbedderProviderConfig
				// if config.Timeout != 30 {
				// 	t.Errorf("Default timeout = %v, want %v", config.Timeout, 30)
				// }
				if config.MaxRetries != 3 {
					t.Errorf("Default max_retries = %v, want %v", config.MaxRetries, 3)
				}
			},
		},
		{
			name: "partial_config_preserves_values",
			config: EmbedderProviderConfig{
				Type:  "custom",
				Model: "custom-model",
			},
			validateConfig: func(t *testing.T, config EmbedderProviderConfig) {
				if config.Type != "custom" {
					t.Errorf("Type should be preserved: %v", config.Type)
				}
				if config.Model != "custom-model" {
					t.Errorf("Model should be preserved: %v", config.Model)
				}
				if config.Host != "http://localhost:11434" {
					t.Errorf("Default host = %v, want %v", config.Host, "http://localhost:11434")
				}
				if config.Dimension != 768 {
					t.Errorf("Default dimension = %v, want %v", config.Dimension, 768)
				}
			},
		},
		{
			name: "zero_values_set_defaults",
			config: EmbedderProviderConfig{
				Type:      "ollama",
				Model:     "nomic-embed-text",
				Host:      "http://localhost:11434",
				Dimension: 0,
				// Timeout field was removed from EmbedderProviderConfig
				MaxRetries: 0,
			},
			validateConfig: func(t *testing.T, config EmbedderProviderConfig) {
				if config.Dimension != 768 {
					t.Errorf("Zero dimension should be set to default: %v", config.Dimension)
				}
				// Timeout field was removed
				// if config.Timeout != 30 {
				// 	t.Errorf("Zero timeout should be set to default: %v", config.Timeout)
				// }
				if config.MaxRetries != 3 {
					t.Errorf("Zero max_retries should be set to default: %v", config.MaxRetries)
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

func TestAgentConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		config         AgentConfig
		validateConfig func(t *testing.T, config AgentConfig)
	}{
		{
			name:   "empty_config_native_defaults",
			config: AgentConfig{},
			validateConfig: func(t *testing.T, config AgentConfig) {
				if config.Type != "native" {
					t.Errorf("Default type = %v, want %v", config.Type, "native")
				}
				if config.Visibility != "public" {
					t.Errorf("Default visibility = %v, want %v", config.Visibility, "public")
				}
				// Name is now set in Validate() to use agent ID, not in SetDefaults()
				if config.Name != "" {
					t.Errorf("SetDefaults should leave name empty (set in Validate), got = %v", config.Name)
				}
				if config.Description != "AI assistant with local tools and knowledge" {
					t.Errorf("Default description = %v, want %v", config.Description, "AI assistant with local tools and knowledge")
				}
				if config.LLM != "default-llm" {
					t.Errorf("Default LLM = %v, want %v", config.LLM, "default-llm")
				}
			},
		},
		{
			name: "a2a_agent_defaults",
			config: AgentConfig{
				Type: "a2a",
			},
			validateConfig: func(t *testing.T, config AgentConfig) {
				if config.Type != "a2a" {
					t.Errorf("Type should be preserved: %v", config.Type)
				}
				if config.Name != "External Agent" {
					t.Errorf("Default A2A name = %v, want %v", config.Name, "External Agent")
				}
				if config.Description != "External A2A-compliant agent" {
					t.Errorf("Default A2A description = %v, want %v", config.Description, "External A2A-compliant agent")
				}
				if config.LLM != "" {
					t.Errorf("A2A agent should not have LLM: %v", config.LLM)
				}
			},
		},
		{
			name: "partial_config_preserves_values",
			config: AgentConfig{
				Type:        "native",
				Name:        "Custom Agent",
				Description: "Custom description",
				LLM:         "custom-llm",
			},
			validateConfig: func(t *testing.T, config AgentConfig) {
				if config.Type != "native" {
					t.Errorf("Type should be preserved: %v", config.Type)
				}
				if config.Name != "Custom Agent" {
					t.Errorf("Name should be preserved: %v", config.Name)
				}
				if config.Description != "Custom description" {
					t.Errorf("Description should be preserved: %v", config.Description)
				}
				if config.LLM != "custom-llm" {
					t.Errorf("LLM should be preserved: %v", config.LLM)
				}
				if config.Visibility != "public" {
					t.Errorf("Default visibility = %v, want %v", config.Visibility, "public")
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

func TestGlobalSettings_SetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		config         GlobalSettings
		validateConfig func(t *testing.T, config GlobalSettings)
	}{
		{
			name:   "empty_config_defaults",
			config: GlobalSettings{},
			validateConfig: func(t *testing.T, config GlobalSettings) {

				if config.Performance.MaxConcurrency != 4 {
					t.Errorf("Default max concurrency = %v, want %v", config.Performance.MaxConcurrency, 4)
				}

				if config.A2AServer.Host != "0.0.0.0" {
					t.Errorf("Default A2A host = %v, want %v", config.A2AServer.Host, "0.0.0.0")
				}
				if config.A2AServer.Port != 8080 {
					t.Errorf("Default A2A port = %v, want %v", config.A2AServer.Port, 8080)
				}
			},
		},
		{
			name: "partial_config_preserves_values",
			config: GlobalSettings{
				Performance: PerformanceConfig{
					MaxConcurrency: 8,
				},
			},
			validateConfig: func(t *testing.T, config GlobalSettings) {
				if config.Performance.MaxConcurrency != 8 {
					t.Errorf("Max concurrency should be preserved: %v", config.Performance.MaxConcurrency)
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

func TestPerformanceConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		config         PerformanceConfig
		validateConfig func(t *testing.T, config PerformanceConfig)
	}{
		{
			name:   "empty_config_defaults",
			config: PerformanceConfig{},
			validateConfig: func(t *testing.T, config PerformanceConfig) {
				if config.MaxConcurrency != 4 {
					t.Errorf("Default max concurrency = %v, want %v", config.MaxConcurrency, 4)
				}
			},
		},
		{
			name: "partial_config_preserves_values",
			config: PerformanceConfig{
				MaxConcurrency: 8,
			},
			validateConfig: func(t *testing.T, config PerformanceConfig) {
				if config.MaxConcurrency != 8 {
					t.Errorf("Max concurrency should be preserved: %v", config.MaxConcurrency)
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

func TestA2AServerConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		config         A2AServerConfig
		validateConfig func(t *testing.T, config A2AServerConfig)
	}{
		{
			name:   "empty_config_defaults",
			config: A2AServerConfig{},
			validateConfig: func(t *testing.T, config A2AServerConfig) {
				if config.Host != "0.0.0.0" {
					t.Errorf("Default host = %v, want %v", config.Host, "0.0.0.0")
				}
				if config.Port != 8080 {
					t.Errorf("Default port = %v, want %v", config.Port, 8080)
				}
			},
		},
		{
			name: "partial_config_preserves_values",
			config: A2AServerConfig{
				Host: "localhost",
			},
			validateConfig: func(t *testing.T, config A2AServerConfig) {
				if config.Host != "localhost" {
					t.Errorf("Host should be preserved: %v", config.Host)
				}
				if config.Port != 8080 {
					t.Errorf("Default port = %v, want %v", config.Port, 8080)
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

func TestAuthConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		config         AuthConfig
		validateConfig func(t *testing.T, config AuthConfig)
	}{
		{
			name:   "empty_config_no_defaults",
			config: AuthConfig{},
			validateConfig: func(t *testing.T, config AuthConfig) {

				if config.JWKSURL != "" {
					t.Errorf("JWKS URL should remain empty: %v", config.JWKSURL)
				}
				if config.Issuer != "" {
					t.Errorf("Issuer should remain empty: %v", config.Issuer)
				}
				if config.Audience != "" {
					t.Errorf("Audience should remain empty: %v", config.Audience)
				}
			},
		},
		{
			name: "partial_config_preserves_values",
			config: AuthConfig{
				JWKSURL: "https://example.com/.well-known/jwks.json",
			},
			validateConfig: func(t *testing.T, config AuthConfig) {
				if config.JWKSURL != "https://example.com/.well-known/jwks.json" {
					t.Errorf("JWKS URL should be preserved: %v", config.JWKSURL)
				}
				if config.Issuer != "" {
					t.Errorf("Issuer should remain empty: %v", config.Issuer)
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
