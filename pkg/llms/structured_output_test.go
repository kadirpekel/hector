package llms

import (
	"encoding/json"
	"testing"
)

func TestStructuredOutputConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *StructuredOutputConfig
	}{
		{
			name: "JSON with schema",
			config: &StructuredOutputConfig{
				Format: "json",
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"sentiment": map[string]interface{}{
							"type": "string",
							"enum": []string{"positive", "negative", "neutral"},
						},
						"score": map[string]interface{}{
							"type": "number",
						},
					},
					"required": []string{"sentiment", "score"},
				},
			},
		},
		{
			name: "Enum format",
			config: &StructuredOutputConfig{
				Format: "enum",
				Enum:   []string{"Percussion", "String", "Woodwind", "Brass", "Keyboard"},
			},
		},
		{
			name: "JSON with Anthropic prefill",
			config: &StructuredOutputConfig{
				Format:  "json",
				Schema:  map[string]interface{}{"type": "object"},
				Prefill: "{\"sentiment\":",
			},
		},
		{
			name: "JSON with Gemini property ordering",
			config: &StructuredOutputConfig{
				Format:           "json",
				Schema:           map[string]interface{}{"type": "object"},
				PropertyOrdering: []string{"name", "age", "city"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Failed to marshal config: %v", err)
			}

			var decoded StructuredOutputConfig
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal config: %v", err)
			}

			if decoded.Format != tt.config.Format {
				t.Errorf("Format mismatch: got %s, want %s", decoded.Format, tt.config.Format)
			}
		})
	}
}

func TestJSONSchemaValidation(t *testing.T) {
	schema := &JSONSchema{
		Type: "object",
		Properties: map[string]JSONSchema{
			"name": {
				Type:        "string",
				Description: "Person's name",
			},
			"age": {
				Type:        "number",
				Description: "Person's age",
			},
			"skills": {
				Type: "array",
				Items: &JSONSchema{
					Type: "string",
				},
			},
		},
		Required: []string{"name", "age"},
	}

	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	var decoded JSONSchema
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	if decoded.Type != "object" {
		t.Errorf("Type mismatch: got %s, want object", decoded.Type)
	}

	if len(decoded.Required) != 2 {
		t.Errorf("Required fields mismatch: got %d, want 2", len(decoded.Required))
	}
}

func TestStructuredOutputProviderInterface(t *testing.T) {

	var _ StructuredOutputProvider = (*OpenAIProvider)(nil)
	var _ StructuredOutputProvider = (*AnthropicProvider)(nil)
	var _ StructuredOutputProvider = (*GeminiProvider)(nil)
	var _ StructuredOutputProvider = (*OllamaProvider)(nil)
}

func TestProviderSupportsStructuredOutput(t *testing.T) {
	tests := []struct {
		name     string
		provider interface {
			SupportsStructuredOutput() bool
		}
		expected bool
	}{
		{
			name:     "OpenAI supports structured output",
			provider: &OpenAIProvider{},
			expected: true,
		},
		{
			name:     "Anthropic supports structured output",
			provider: &AnthropicProvider{},
			expected: true,
		},
		{
			name:     "Gemini supports structured output",
			provider: &GeminiProvider{},
			expected: true,
		},
		{
			name:     "Ollama supports structured output",
			provider: &OllamaProvider{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.provider.SupportsStructuredOutput(); got != tt.expected {
				t.Errorf("SupportsStructuredOutput() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func ExampleStructuredOutputConfig_sentiment() {
	config := &StructuredOutputConfig{
		Format: "json",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"sentiment": map[string]interface{}{
					"type": "string",
					"enum": []string{"positive", "negative", "neutral"},
				},
				"confidence": map[string]interface{}{
					"type":    "number",
					"minimum": 0,
					"maximum": 1,
				},
				"key_phrases": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"sentiment", "confidence"},
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	_ = data

}

func ExampleStructuredOutputConfig_enum() {
	config := &StructuredOutputConfig{
		Format: "enum",
		Enum:   []string{"Percussion", "String", "Woodwind", "Brass", "Keyboard"},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	_ = data

}

func ExampleStructuredOutputConfig_anthropicPrefill() {
	config := &StructuredOutputConfig{
		Format: "json",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"report_type": map[string]interface{}{"type": "string"},
				"summary":     map[string]interface{}{"type": "string"},
			},
		},
		Prefill: "{\"report_type\":",
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	_ = data

}

func ExampleStructuredOutputConfig_geminiPropertyOrdering() {
	config := &StructuredOutputConfig{
		Format: "json",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name":  map[string]interface{}{"type": "string"},
				"age":   map[string]interface{}{"type": "number"},
				"email": map[string]interface{}{"type": "string"},
			},
		},
		PropertyOrdering: []string{"name", "age", "email"},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	_ = data

}
