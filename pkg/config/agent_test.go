package config

import (
	"testing"
)

func TestStructuredOutputConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  StructuredOutputConfig
		wantErr bool
	}{
		{
			name: "valid strict schema",
			config: StructuredOutputConfig{
				Strict: BoolPtr(true),
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"foo": map[string]interface{}{"type": "string"},
					},
					"required":             []interface{}{"foo"},
					"additionalProperties": false,
				},
			},
			wantErr: false,
		},
		{
			name: "missing additionalProperties: false",
			config: StructuredOutputConfig{
				Strict: BoolPtr(true),
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"foo": map[string]interface{}{"type": "string"},
					},
					"required": []interface{}{"foo"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing required field",
			config: StructuredOutputConfig{
				Strict: BoolPtr(true),
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"foo": map[string]interface{}{"type": "string"},
						"bar": map[string]interface{}{"type": "string"},
					},
					"required":             []interface{}{"foo"},
					"additionalProperties": false,
				},
			},
			wantErr: true,
		},
		{
			name: "nested valid strict",
			config: StructuredOutputConfig{
				Strict: BoolPtr(true),
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"wrapper": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"inner": map[string]interface{}{"type": "string"},
							},
							"required":             []interface{}{"inner"},
							"additionalProperties": false,
						},
					},
					"required":             []interface{}{"wrapper"},
					"additionalProperties": false,
				},
			},
			wantErr: false,
		},
		{
			name: "nested missing required",
			config: StructuredOutputConfig{
				Strict: BoolPtr(true),
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"wrapper": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"inner": map[string]interface{}{"type": "string"},
								"other": map[string]interface{}{"type": "string"},
							},
							"required":             []interface{}{"inner"}, // missing 'other'
							"additionalProperties": false,
						},
					},
					"required":             []interface{}{"wrapper"},
					"additionalProperties": false,
				},
			},
			wantErr: true,
		},
		{
			name: "nested missing additionalProperties",
			config: StructuredOutputConfig{
				Strict: BoolPtr(true),
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"wrapper": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"inner": map[string]interface{}{"type": "string"},
							},
							"required": []interface{}{"inner"},
							// missing additionalProperties: false
						},
					},
					"required":             []interface{}{"wrapper"},
					"additionalProperties": false,
				},
			},
			wantErr: true,
		},
		{
			name: "strict disabled allows loose schema",
			config: StructuredOutputConfig{
				Strict: BoolPtr(false),
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"foo": map[string]interface{}{"type": "string"},
					},
					// missing required, missing additionalProperties
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("StructuredOutputConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helpers needed if not in same package?
// No, config package likely has BoolPtr helper or I can redefine or use literal pointers.
// agent.go uses `BoolPtr`. It might be in `types.go` or `utils.go`.
// If it's private or not exported, I'll need to duplicate it or check.
// agent.go lines 365+ use BoolPtr.
// I'll check if BoolPtr is exported or check `types.go`.
