package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/reasoning"
)

// ReasoningBuilder provides a fluent API for building reasoning strategies
type ReasoningBuilder struct {
	strategyType   string
	maxIterations  int
	enableStreaming *bool
	showTools      *bool
	showThinking   *bool
}

// NewReasoning creates a new reasoning strategy builder
func NewReasoning(strategyType string) *ReasoningBuilder {
	if strategyType == "" {
		strategyType = "chain-of-thought"
	}
	return &ReasoningBuilder{
		strategyType:    strategyType,
		maxIterations:   100, // Default from ReasoningConfig
		enableStreaming: boolPtr(true),
		showTools:       boolPtr(true),
		showThinking:    boolPtr(true),
	}
}

// MaxIterations sets the maximum number of reasoning iterations
func (b *ReasoningBuilder) MaxIterations(max int) *ReasoningBuilder {
	if max <= 0 {
		panic("max iterations must be positive")
	}
	b.maxIterations = max
	return b
}

// EnableStreaming enables or disables streaming output
func (b *ReasoningBuilder) EnableStreaming(enable bool) *ReasoningBuilder {
	b.enableStreaming = &enable
	return b
}

// ShowTools enables or disables showing tool-related events
func (b *ReasoningBuilder) ShowTools(show bool) *ReasoningBuilder {
	b.showTools = &show
	return b
}

// ShowThinking enables or disables showing thinking-related content
func (b *ReasoningBuilder) ShowThinking(show bool) *ReasoningBuilder {
	b.showThinking = &show
	return b
}

// Build creates the reasoning strategy
func (b *ReasoningBuilder) Build() (reasoning.ReasoningStrategy, error) {
	switch b.strategyType {
	case "chain-of-thought", "default", "":
		return reasoning.NewChainOfThoughtStrategy(), nil
	case "supervisor":
		return reasoning.NewSupervisorStrategy(), nil
	default:
		return nil, fmt.Errorf("unknown reasoning strategy: %s (supported: 'chain-of-thought', 'supervisor')", b.strategyType)
	}
}

// GetConfig returns the reasoning configuration (for use in agent building)
func (b *ReasoningBuilder) GetConfig() ReasoningConfig {
	return ReasoningConfig{
		Engine:          b.strategyType,
		MaxIterations:   b.maxIterations,
		EnableStreaming: b.enableStreaming,
		ShowTools:       b.showTools,
		ShowThinking:    b.showThinking,
	}
}

// ReasoningConfig represents reasoning configuration
type ReasoningConfig struct {
	Engine          string
	MaxIterations   int
	EnableStreaming *bool
	ShowTools       *bool
	ShowThinking    *bool
}

