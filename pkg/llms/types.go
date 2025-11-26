package llms

import (
	"github.com/kadirpekel/hector/pkg/protocol"
)

type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type StreamChunk struct {
	Type      string
	Text      string
	ToolCall  *protocol.ToolCall
	Tokens    int
	Error     error
	Signature string // For Anthropic thinking blocks - signature for verification
}

type StructuredOutputConfig struct {
	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	Schema interface{} `json:"schema,omitempty" yaml:"schema,omitempty"`

	Enum []string `json:"enum,omitempty" yaml:"enum,omitempty"`

	Prefill string `json:"prefill,omitempty" yaml:"prefill,omitempty"`

	PropertyOrdering []string `json:"property_ordering,omitempty" yaml:"property_ordering,omitempty"`
}

type JSONSchema struct {
	Type                 string                `json:"type"`
	Properties           map[string]JSONSchema `json:"properties,omitempty"`
	Items                *JSONSchema           `json:"items,omitempty"`
	Required             []string              `json:"required,omitempty"`
	Enum                 []string              `json:"enum,omitempty"`
	Description          string                `json:"description,omitempty"`
	PropertyOrdering     []string              `json:"propertyOrdering,omitempty"`
	AdditionalProperties *bool                 `json:"additionalProperties,omitempty"`
}

func ConvertToolInfoToDefinition(name, description string, parameters []interface{}) ToolDefinition {

	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
		"required":   []string{},
	}

	properties := schema["properties"].(map[string]interface{})
	required := []string{}

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
