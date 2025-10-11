package llms

import "github.com/kadirpekel/hector/pkg/a2a"

// ============================================================================
// COMMON FUNCTION CALLING TYPES
// Shared across OpenAI and Anthropic providers
// Uses A2A protocol Message types for true A2A-native architecture
// ============================================================================

// Message type removed - use a2a.Message directly for true A2A-native architecture

// ToolDefinition represents a tool/function that can be called
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// ToolCall type removed - use a2a.ToolCall directly for true A2A-native architecture

// StreamChunk represents a chunk of streaming response
type StreamChunk struct {
	Type     string        // "text", "tool_call", "done", "error"
	Text     string        // For text chunks
	ToolCall *a2a.ToolCall // For tool_call chunks
	Tokens   int           // For done chunks
	Error    error         // For error chunks
}

// ============================================================================
// STRUCTURED OUTPUT TYPES
// Provider-agnostic structured output configuration
// ============================================================================

// StructuredOutputConfig represents structured output configuration
// that works across all providers (Anthropic, OpenAI, Gemini)
type StructuredOutputConfig struct {
	// Format specifies the output format: "json", "xml", "enum"
	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	// Schema is the JSON schema for structured output (for format="json")
	// Can be provided as a JSON string or map
	Schema interface{} `json:"schema,omitempty" yaml:"schema,omitempty"`

	// Enum values (for format="enum")
	Enum []string `json:"enum,omitempty" yaml:"enum,omitempty"`

	// Prefill string for Anthropic (optional, Anthropic-specific optimization)
	Prefill string `json:"prefill,omitempty" yaml:"prefill,omitempty"`

	// PropertyOrdering for Gemini (optional, Gemini-specific optimization)
	PropertyOrdering []string `json:"property_ordering,omitempty" yaml:"property_ordering,omitempty"`
}

// JSONSchema represents a JSON Schema (simplified for common use)
type JSONSchema struct {
	Type                 string                `json:"type"`
	Properties           map[string]JSONSchema `json:"properties,omitempty"`
	Items                *JSONSchema           `json:"items,omitempty"`
	Required             []string              `json:"required,omitempty"`
	Enum                 []string              `json:"enum,omitempty"`
	Description          string                `json:"description,omitempty"`
	PropertyOrdering     []string              `json:"propertyOrdering,omitempty"`     // Gemini-specific
	AdditionalProperties *bool                 `json:"additionalProperties,omitempty"` // JSON Schema standard
}

// ConvertToolInfoToDefinition converts from tools package format
func ConvertToolInfoToDefinition(name, description string, parameters []interface{}) ToolDefinition {
	// Convert parameters to JSON Schema format
	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
		"required":   []string{},
	}

	properties := schema["properties"].(map[string]interface{})
	required := []string{}

	// Parse parameters (assuming they're in a specific format)
	for _, param := range parameters {
		if p, ok := param.(map[string]interface{}); ok {
			paramName := p["name"].(string)
			paramType := p["type"].(string)
			paramDesc := p["description"].(string)
			isRequired := p["required"].(bool)

			properties[paramName] = map[string]interface{}{
				"type":        paramType,
				"description": paramDesc,
			}

			if isRequired {
				required = append(required, paramName)
			}
		}
	}

	schema["required"] = required

	return ToolDefinition{
		Name:        name,
		Description: description,
		Parameters:  schema,
	}
}
