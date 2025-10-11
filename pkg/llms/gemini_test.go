package llms

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a"

	"github.com/kadirpekel/hector/pkg/config"
)

// TestNewGeminiProviderFromConfig tests Gemini provider creation
func TestNewGeminiProviderFromConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.LLMProviderConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: &config.LLMProviderConfig{
				Type:        "gemini",
				Model:       "gemini-2.0-flash",
				Host:        "https://generativelanguage.googleapis.com",
				APIKey:      "test-api-key",
				Temperature: 0.7,
				MaxTokens:   2048,
				Timeout:     60,
				MaxRetries:  3,
				RetryDelay:  2,
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: &config.LLMProviderConfig{
				Type:   "gemini",
				Model:  "gemini-2.0-flash",
				Host:   "https://generativelanguage.googleapis.com",
				APIKey: "",
			},
			wantErr: true,
		},
		{
			name: "gemini-1.5-pro model",
			config: &config.LLMProviderConfig{
				Type:   "gemini",
				Model:  "gemini-1.5-pro",
				Host:   "https://generativelanguage.googleapis.com",
				APIKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "gemini-1.5-flash model",
			config: &config.LLMProviderConfig{
				Type:   "gemini",
				Model:  "gemini-1.5-flash",
				Host:   "https://generativelanguage.googleapis.com",
				APIKey: "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGeminiProviderFromConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGeminiProviderFromConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("Expected provider to be created, got nil")
			}
		})
	}
}

// TestGeminiProvider_GetModelName tests model name retrieval
func TestGeminiProvider_GetModelName(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		wantModel string
	}{
		{
			name:      "gemini-2.0-flash",
			model:     "gemini-2.0-flash",
			wantModel: "gemini-2.0-flash",
		},
		{
			name:      "gemini-1.5-pro",
			model:     "gemini-1.5-pro",
			wantModel: "gemini-1.5-pro",
		},
		{
			name:      "gemini-1.5-flash",
			model:     "gemini-1.5-flash",
			wantModel: "gemini-1.5-flash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.LLMProviderConfig{
				Type:   "gemini",
				Model:  tt.model,
				Host:   "https://generativelanguage.googleapis.com",
				APIKey: "test-key",
			}

			provider, err := NewGeminiProviderFromConfig(cfg)
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			if got := provider.GetModelName(); got != tt.wantModel {
				t.Errorf("GetModelName() = %v, want %v", got, tt.wantModel)
			}
		})
	}
}

// TestGeminiProvider_GetMaxTokens tests max tokens configuration
func TestGeminiProvider_GetMaxTokens(t *testing.T) {
	tests := []struct {
		name      string
		maxTokens int
	}{
		{"default tokens", 2048},
		{"high tokens", 8192},
		{"low tokens", 512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.LLMProviderConfig{
				Type:      "gemini",
				Model:     "gemini-2.0-flash",
				Host:      "https://generativelanguage.googleapis.com",
				APIKey:    "test-key",
				MaxTokens: tt.maxTokens,
			}

			provider, _ := NewGeminiProviderFromConfig(cfg)
			if got := provider.GetMaxTokens(); got != tt.maxTokens {
				t.Errorf("GetMaxTokens() = %v, want %v", got, tt.maxTokens)
			}
		})
	}
}

// TestGeminiProvider_GetTemperature tests temperature configuration
func TestGeminiProvider_GetTemperature(t *testing.T) {
	tests := []struct {
		name        string
		temperature float64
	}{
		{"low temperature", 0.0},
		{"medium temperature", 0.7},
		{"high temperature", 1.0},
		{"custom temperature", 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.LLMProviderConfig{
				Type:        "gemini",
				Model:       "gemini-2.0-flash",
				Host:        "https://generativelanguage.googleapis.com",
				APIKey:      "test-key",
				Temperature: tt.temperature,
			}

			provider, _ := NewGeminiProviderFromConfig(cfg)
			if got := provider.GetTemperature(); got != tt.temperature {
				t.Errorf("GetTemperature() = %v, want %v", got, tt.temperature)
			}
		})
	}
}

// TestGeminiProvider_SupportsStructuredOutput tests structured output capability
func TestGeminiProvider_SupportsStructuredOutput(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:   "gemini",
		Model:  "gemini-2.0-flash",
		Host:   "https://generativelanguage.googleapis.com",
		APIKey: "test-key",
	}

	provider, err := NewGeminiProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if !provider.SupportsStructuredOutput() {
		t.Error("Gemini provider should support structured output")
	}
}

// TestGeminiProvider_ConvertMessages tests message conversion
func TestGeminiProvider_ConvertMessages(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:   "gemini",
		Model:  "gemini-2.0-flash",
		Host:   "https://generativelanguage.googleapis.com",
		APIKey: "test-key",
	}

	provider, _ := NewGeminiProviderFromConfig(cfg)

	tests := []struct {
		name     string
		messages []a2a.Message
		wantLen  int
	}{
		{
			name: "single user message",
			messages: []a2a.Message{
				a2a.CreateUserMessage("Hello"),
			},
			wantLen: 1,
		},
		{
			name: "user and assistant messages",
			messages: []a2a.Message{
				a2a.CreateUserMessage("Hello"),
				a2a.CreateAssistantMessage("Hi there!"),
			},
			wantLen: 2,
		},
		{
			name: "system message converts to user",
			messages: []a2a.Message{
				a2a.CreateTextMessage(a2a.MessageRoleSystem, "You are helpful"),
				a2a.CreateUserMessage("Hello"),
			},
			wantLen: 2,
		},
		{
			name: "with tool calls",
			messages: []a2a.Message{
				{
					Role: "assistant",
					ToolCalls: []a2a.ToolCall{
						{Name: "get_weather", Arguments: map[string]interface{}{"city": "NYC"}},
					},
				},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converted := provider.convertMessages(tt.messages)
			if len(converted) != tt.wantLen {
				t.Errorf("convertMessages() returned %d contents, want %d", len(converted), tt.wantLen)
			}

			// Verify system messages are converted to user role
			for i, msg := range tt.messages {
				if msg.Role == "system" && converted[i].Role != "user" {
					t.Errorf("System message not converted to user role")
				}
			}

			// Verify assistant role is converted to model
			for i, msg := range tt.messages {
				if msg.Role == "assistant" && converted[i].Role != "model" {
					t.Errorf("Assistant role not converted to model role")
				}
			}
		})
	}
}

// TestGeminiProvider_ConvertTools tests tool conversion
func TestGeminiProvider_ConvertTools(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:   "gemini",
		Model:  "gemini-2.0-flash",
		Host:   "https://generativelanguage.googleapis.com",
		APIKey: "test-key",
	}

	provider, _ := NewGeminiProviderFromConfig(cfg)

	tests := []struct {
		name    string
		tools   []ToolDefinition
		wantLen int
	}{
		{
			name:    "no tools",
			tools:   []ToolDefinition{},
			wantLen: 0,
		},
		{
			name: "single tool",
			tools: []ToolDefinition{
				{
					Name:        "get_weather",
					Description: "Get weather for a city",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"city": map[string]interface{}{"type": "string"},
						},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "multiple tools",
			tools: []ToolDefinition{
				{Name: "tool1", Description: "First tool"},
				{Name: "tool2", Description: "Second tool"},
				{Name: "tool3", Description: "Third tool"},
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converted := provider.convertTools(tt.tools)
			if len(converted) != tt.wantLen {
				t.Errorf("convertTools() returned %d functions, want %d", len(converted), tt.wantLen)
			}

			// Verify function declarations have required fields
			for i, fn := range converted {
				if fn.Name != tt.tools[i].Name {
					t.Errorf("Function name = %v, want %v", fn.Name, tt.tools[i].Name)
				}
				if fn.Description != tt.tools[i].Description {
					t.Errorf("Function description = %v, want %v", fn.Description, tt.tools[i].Description)
				}
			}
		})
	}
}

// TestGeminiProvider_BuildGenerationConfig tests generation config building
func TestGeminiProvider_BuildGenerationConfig(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:        "gemini",
		Model:       "gemini-2.0-flash",
		Host:        "https://generativelanguage.googleapis.com",
		APIKey:      "test-key",
		Temperature: 0.7,
		MaxTokens:   2048,
	}

	provider, _ := NewGeminiProviderFromConfig(cfg)

	tests := []struct {
		name         string
		structConfig *StructuredOutputConfig
		wantMimeType string
	}{
		{
			name:         "no structured output",
			structConfig: nil,
			wantMimeType: "",
		},
		{
			name: "JSON structured output",
			structConfig: &StructuredOutputConfig{
				Format: "json",
				Schema: map[string]interface{}{
					"type": "object",
				},
			},
			wantMimeType: "application/json",
		},
		{
			name: "enum structured output",
			structConfig: &StructuredOutputConfig{
				Format: "enum",
			},
			wantMimeType: "text/x.enum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := provider.buildGenerationConfig(tt.structConfig)

			if config.MaxOutputTokens != 2048 {
				t.Errorf("MaxOutputTokens = %v, want 2048", config.MaxOutputTokens)
			}

			if tt.structConfig != nil {
				if config.ResponseMimeType != tt.wantMimeType {
					t.Errorf("ResponseMimeType = %v, want %v", config.ResponseMimeType, tt.wantMimeType)
				}
			}
		})
	}
}

// TestGeminiProvider_ConvertSchemaToGemini tests schema conversion with property ordering
func TestGeminiProvider_ConvertSchemaToGemini(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:   "gemini",
		Model:  "gemini-2.0-flash",
		Host:   "https://generativelanguage.googleapis.com",
		APIKey: "test-key",
	}

	provider, _ := NewGeminiProviderFromConfig(cfg)

	tests := []struct {
		name             string
		schema           interface{}
		propertyOrdering []string
		wantOrdering     bool
	}{
		{
			name: "schema without ordering",
			schema: map[string]interface{}{
				"type": "object",
			},
			propertyOrdering: nil,
			wantOrdering:     false,
		},
		{
			name: "schema with property ordering",
			schema: map[string]interface{}{
				"type": "object",
			},
			propertyOrdering: []string{"name", "age", "email"},
			wantOrdering:     true,
		},
		{
			name:             "invalid schema type",
			schema:           "not a map",
			propertyOrdering: []string{"field1"},
			wantOrdering:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.convertSchemaToGemini(tt.schema, tt.propertyOrdering)

			if tt.wantOrdering {
				if result == nil {
					t.Error("Expected schema result, got nil")
					return
				}
				if ordering, ok := result["propertyOrdering"]; ok {
					orderList, ok := ordering.([]string)
					if !ok || len(orderList) != len(tt.propertyOrdering) {
						t.Errorf("Property ordering not properly set")
					}
				} else {
					t.Error("Property ordering not found in schema")
				}
			}
		})
	}
}

// TestGeminiProvider_Close tests provider cleanup
func TestGeminiProvider_Close(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:   "gemini",
		Model:  "gemini-2.0-flash",
		Host:   "https://generativelanguage.googleapis.com",
		APIKey: "test-key",
	}

	provider, err := NewGeminiProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if err := provider.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// TestGeminiProvider_InterfaceCompliance ensures Gemini implements required interfaces
func TestGeminiProvider_InterfaceCompliance(t *testing.T) {
	// Compile-time checks
	var _ LLMProvider = (*GeminiProvider)(nil)
	var _ StructuredOutputProvider = (*GeminiProvider)(nil)
}

// TestGeminiProvider_ParseResponse tests response parsing
func TestGeminiProvider_ParseResponse(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:   "gemini",
		Model:  "gemini-2.0-flash",
		Host:   "https://generativelanguage.googleapis.com",
		APIKey: "test-key",
	}

	provider, _ := NewGeminiProviderFromConfig(cfg)

	tests := []struct {
		name          string
		response      *GeminiResponse
		wantText      bool
		wantToolCalls bool
		wantErr       bool
	}{
		{
			name: "text only response",
			response: &GeminiResponse{
				Candidates: []GeminiCandidate{
					{
						Content: GeminiContent{
							Parts: []GeminiPart{
								{"text": "Hello, world!"},
							},
						},
					},
				},
				UsageMetadata: &GeminiUsageMetadata{TotalTokenCount: 10},
			},
			wantText:      true,
			wantToolCalls: false,
			wantErr:       false,
		},
		{
			name: "function call response",
			response: &GeminiResponse{
				Candidates: []GeminiCandidate{
					{
						Content: GeminiContent{
							Parts: []GeminiPart{
								{
									"functionCall": map[string]interface{}{
										"name": "get_weather",
										"args": map[string]interface{}{
											"city": "NYC",
										},
									},
								},
							},
						},
					},
				},
			},
			wantText:      false,
			wantToolCalls: true,
			wantErr:       false,
		},
		{
			name: "empty response",
			response: &GeminiResponse{
				Candidates: []GeminiCandidate{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, toolCalls, tokens, err := provider.parseResponse(tt.response)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if tt.wantText && text == "" {
				t.Error("Expected text content, got empty string")
			}

			if tt.wantToolCalls && len(toolCalls) == 0 {
				t.Error("Expected tool calls, got none")
			}

			if !tt.wantToolCalls && len(toolCalls) > 0 {
				t.Error("Unexpected tool calls in response")
			}

			if tt.response.UsageMetadata != nil && tokens != tt.response.UsageMetadata.TotalTokenCount {
				t.Errorf("Tokens = %d, want %d", tokens, tt.response.UsageMetadata.TotalTokenCount)
			}
		})
	}
}
