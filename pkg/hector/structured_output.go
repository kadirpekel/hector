package hector

import (
	"github.com/kadirpekel/hector/pkg/config"
)

// StructuredOutputBuilder provides a fluent API for building structured output config
type StructuredOutputBuilder struct {
	config *config.StructuredOutputConfig
}

// NewStructuredOutput creates a new structured output builder
func NewStructuredOutput() *StructuredOutputBuilder {
	return &StructuredOutputBuilder{
		config: &config.StructuredOutputConfig{},
	}
}

// NewStructuredOutputWithConfig creates a builder with existing config
func NewStructuredOutputWithConfig(cfg *config.StructuredOutputConfig) *StructuredOutputBuilder {
	if cfg == nil {
		cfg = &config.StructuredOutputConfig{}
	}
	return &StructuredOutputBuilder{
		config: cfg,
	}
}

// Format sets the output format (e.g., "json", "yaml")
func (b *StructuredOutputBuilder) Format(format string) *StructuredOutputBuilder {
	b.config.Format = format
	return b
}

// Schema sets the JSON schema for structured output
func (b *StructuredOutputBuilder) Schema(schema map[string]interface{}) *StructuredOutputBuilder {
	b.config.Schema = schema
	return b
}

// Enum sets the enum values for constrained output
func (b *StructuredOutputBuilder) Enum(values []string) *StructuredOutputBuilder {
	b.config.Enum = values
	return b
}

// Prefill sets the prefill value for structured output
func (b *StructuredOutputBuilder) Prefill(value string) *StructuredOutputBuilder {
	b.config.Prefill = value
	return b
}

// PropertyOrdering sets the property ordering for structured output
func (b *StructuredOutputBuilder) PropertyOrdering(order []string) *StructuredOutputBuilder {
	b.config.PropertyOrdering = order
	return b
}

// Build returns the structured output config
func (b *StructuredOutputBuilder) Build() *config.StructuredOutputConfig {
	return b.config
}
