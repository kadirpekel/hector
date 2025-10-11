package llms

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a"
)

func TestConvertToolInfoToDefinition(t *testing.T) {
	tests := []struct {
		name        string
		description string
		parameters  []interface{}
		expected    ToolDefinition
	}{
		{
			name:        "test_tool",
			description: "A test tool",
			parameters: []interface{}{
				map[string]interface{}{
					"name":        "param1",
					"type":        "string",
					"description": "First parameter",
					"required":    true,
				},
				map[string]interface{}{
					"name":        "param2",
					"type":        "number",
					"description": "Second parameter",
					"required":    false,
				},
			},
			expected: ToolDefinition{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{
							"type":        "string",
							"description": "First parameter",
						},
						"param2": map[string]interface{}{
							"type":        "number",
							"description": "Second parameter",
						},
					},
					"required": []string{"param1"},
				},
			},
		},
		{
			name:        "empty_tool",
			description: "Tool with no parameters",
			parameters:  []interface{}{},
			expected: ToolDefinition{
				Name:        "empty_tool",
				Description: "Tool with no parameters",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			name:        "optional_only_tool",
			description: "Tool with only optional parameters",
			parameters: []interface{}{
				map[string]interface{}{
					"name":        "optional_param",
					"type":        "string",
					"description": "Optional parameter",
					"required":    false,
				},
			},
			expected: ToolDefinition{
				Name:        "optional_only_tool",
				Description: "Tool with only optional parameters",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"optional_param": map[string]interface{}{
							"type":        "string",
							"description": "Optional parameter",
						},
					},
					"required": []string{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToolInfoToDefinition(tt.name, tt.description, tt.parameters)

			if result.Name != tt.expected.Name {
				t.Errorf("ConvertToolInfoToDefinition() name = %v, want %v", result.Name, tt.expected.Name)
			}

			if result.Description != tt.expected.Description {
				t.Errorf("ConvertToolInfoToDefinition() description = %v, want %v", result.Description, tt.expected.Description)
			}

			// Check parameters structure
			if result.Parameters["type"] != tt.expected.Parameters["type"] {
				t.Errorf("ConvertToolInfoToDefinition() parameters type = %v, want %v", result.Parameters["type"], tt.expected.Parameters["type"])
			}

			// Check properties
			expectedProps := tt.expected.Parameters["properties"].(map[string]interface{})
			resultProps := result.Parameters["properties"].(map[string]interface{})

			if len(resultProps) != len(expectedProps) {
				t.Errorf("ConvertToolInfoToDefinition() properties count = %v, want %v", len(resultProps), len(expectedProps))
			}

			for key, expectedProp := range expectedProps {
				if resultProp, exists := resultProps[key]; !exists {
					t.Errorf("ConvertToolInfoToDefinition() missing property %v", key)
				} else {
					expectedPropMap := expectedProp.(map[string]interface{})
					resultPropMap := resultProp.(map[string]interface{})

					if resultPropMap["type"] != expectedPropMap["type"] {
						t.Errorf("ConvertToolInfoToDefinition() property %v type = %v, want %v", key, resultPropMap["type"], expectedPropMap["type"])
					}
					if resultPropMap["description"] != expectedPropMap["description"] {
						t.Errorf("ConvertToolInfoToDefinition() property %v description = %v, want %v", key, resultPropMap["description"], expectedPropMap["description"])
					}
				}
			}

			// Check required fields
			expectedRequired := tt.expected.Parameters["required"].([]string)
			resultRequired := result.Parameters["required"].([]string)

			if len(resultRequired) != len(expectedRequired) {
				t.Errorf("ConvertToolInfoToDefinition() required count = %v, want %v", len(resultRequired), len(expectedRequired))
			}

			for i, expected := range expectedRequired {
				if i >= len(resultRequired) || resultRequired[i] != expected {
					t.Errorf("ConvertToolInfoToDefinition() required[%v] = %v, want %v", i, resultRequired[i], expected)
				}
			}
		})
	}
}

func TestConvertToolInfoToDefinition_InvalidParameters(t *testing.T) {
	tests := []struct {
		name        string
		description string
		parameters  []interface{}
		shouldPanic bool
	}{
		{
			name:        "invalid_param_structure",
			description: "Tool with invalid parameter structure",
			parameters: []interface{}{
				"invalid_string_param",
				map[string]interface{}{
					"name":        "valid_param",
					"type":        "string",
					"description": "Valid parameter",
					"required":    true,
				},
			},
			shouldPanic: false, // Should handle gracefully
		},
		{
			name:        "missing_param_fields",
			description: "Tool with parameters missing required fields",
			parameters: []interface{}{
				map[string]interface{}{
					"name": "param1",
					// Missing type, description, required
				},
			},
			shouldPanic: true, // This will panic due to nil interface conversion
		},
		{
			name:        "nil_parameters",
			description: "Tool with nil parameters",
			parameters:  nil,
			shouldPanic: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				// This will panic, so we expect it to panic
				defer func() {
					if r := recover(); r == nil {
						t.Error("ConvertToolInfoToDefinition() expected panic with invalid parameters")
					}
				}()
				ConvertToolInfoToDefinition(tt.name, tt.description, tt.parameters)
				return
			}

			// Should not panic and should handle gracefully
			result := ConvertToolInfoToDefinition(tt.name, tt.description, tt.parameters)

			if result.Name != tt.name {
				t.Errorf("ConvertToolInfoToDefinition() name = %v, want %v", result.Name, tt.name)
			}

			if result.Description != tt.description {
				t.Errorf("ConvertToolInfoToDefinition() description = %v, want %v", result.Description, tt.description)
			}

			// Should still have valid parameters structure
			if result.Parameters["type"] != "object" {
				t.Errorf("ConvertToolInfoToDefinition() parameters type = %v, want object", result.Parameters["type"])
			}

			properties := result.Parameters["properties"].(map[string]interface{})
			if properties == nil {
				t.Error("ConvertToolInfoToDefinition() properties should not be nil")
			}

			required := result.Parameters["required"].([]string)
			if required == nil {
				t.Error("ConvertToolInfoToDefinition() required should not be nil")
			}
		})
	}
}

func TestMessage_Structure(t *testing.T) {
	// Test A2A Message struct creation and field access
	msg := a2a.CreateUserMessage("Hello, world!")

	if msg.Role != a2a.MessageRoleUser {
		t.Errorf("Message.Role = %v, want user", msg.Role)
	}

	textContent := a2a.ExtractTextFromMessage(msg)
	if textContent != "Hello, world!" {
		t.Errorf("Message.Content = %v, want Hello, world!", textContent)
	}

	if len(msg.ToolCalls) != 0 {
		t.Errorf("Message.ToolCalls length = %v, want 0", len(msg.ToolCalls))
	}
}

func TestToolDefinition_Structure(t *testing.T) {
	// Test ToolDefinition struct creation and field access
	toolDef := ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	if toolDef.Name != "test_tool" {
		t.Errorf("ToolDefinition.Name = %v, want test_tool", toolDef.Name)
	}

	if toolDef.Description != "A test tool" {
		t.Errorf("ToolDefinition.Description = %v, want A test tool", toolDef.Description)
	}

	if toolDef.Parameters["type"] != "object" {
		t.Errorf("ToolDefinition.Parameters[type] = %v, want object", toolDef.Parameters["type"])
	}
}

func TestToolCall_Structure(t *testing.T) {
	// Test A2A ToolCall struct creation and field access
	toolCall := a2a.ToolCall{
		ID:        "call_123",
		Name:      "test_tool",
		Arguments: map[string]interface{}{"param1": "value1"},
		RawArgs:   `{"param1": "value1"}`,
	}

	if toolCall.ID != "call_123" {
		t.Errorf("ToolCall.ID = %v, want call_123", toolCall.ID)
	}

	if toolCall.Name != "test_tool" {
		t.Errorf("ToolCall.Name = %v, want test_tool", toolCall.Name)
	}

	if toolCall.Arguments["param1"] != "value1" {
		t.Errorf("ToolCall.Arguments[param1] = %v, want value1", toolCall.Arguments["param1"])
	}

	if toolCall.RawArgs != `{"param1": "value1"}` {
		t.Errorf("ToolCall.RawArgs = %v, want {\"param1\": \"value1\"}", toolCall.RawArgs)
	}
}

func TestStreamChunk_Structure(t *testing.T) {
	// Test StreamChunk struct creation and field access
	chunk := StreamChunk{
		Type:     "text",
		Text:     "Hello",
		ToolCall: nil,
		Tokens:   5,
		Error:    nil,
	}

	if chunk.Type != "text" {
		t.Errorf("StreamChunk.Type = %v, want text", chunk.Type)
	}

	if chunk.Text != "Hello" {
		t.Errorf("StreamChunk.Text = %v, want Hello", chunk.Text)
	}

	if chunk.Tokens != 5 {
		t.Errorf("StreamChunk.Tokens = %v, want 5", chunk.Tokens)
	}

	if chunk.ToolCall != nil {
		t.Error("StreamChunk.ToolCall should be nil")
	}

	if chunk.Error != nil {
		t.Error("StreamChunk.Error should be nil")
	}
}
