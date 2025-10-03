package reasoning

import (
	"fmt"

	"github.com/kadirpekel/hector/config"
)

// ============================================================================
// STRATEGY FACTORY
// Creates reasoning strategies based on configuration
// ============================================================================

// CreateStrategy creates a reasoning strategy based on engine type
// Currently only supports chain-of-thought (single-agent reasoning)
func CreateStrategy(engineType string, config config.ReasoningConfig) (ReasoningStrategy, error) {
	switch engineType {
	case "default", "", "chain-of-thought":
		return NewChainOfThoughtStrategy(), nil
	default:
		return nil, fmt.Errorf("unsupported reasoning engine type: %s (only 'chain-of-thought' is supported for single-agent reasoning)", engineType)
	}
}

// ListAvailableStrategies returns information about all available reasoning strategies
func ListAvailableStrategies() []StrategyInfo {
	return []StrategyInfo{
		{
			Name:        "chain-of-thought",
			Description: "Single-agent iterative reasoning with native function calling (matches Cursor/Claude approach)",
			Features: []string{
				"One LLM call per iteration",
				"Implicit planning and completion detection",
				"Tool execution with automatic continuation",
				"Conversation history management",
				"Streaming support",
				"Fast and cost-effective",
			},
			Parameters: []StrategyParameter{
				{
					Name:        "max_iterations",
					Type:        "int",
					Description: "Maximum reasoning iterations (default: 10)",
					Required:    false,
					Default:     10,
				},
				{
					Name:        "enable_streaming",
					Type:        "bool",
					Description: "Enable streaming output",
					Required:    false,
					Default:     true,
				},
				{
					Name:        "show_debug_info",
					Type:        "bool",
					Description: "Show iteration counts and reasoning summary",
					Required:    false,
					Default:     false,
				},
			},
		},
	}
}

// StrategyInfo describes a reasoning strategy
type StrategyInfo struct {
	Name        string
	Description string
	Features    []string
	Parameters  []StrategyParameter
}

// StrategyParameter describes a strategy parameter
type StrategyParameter struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Default     interface{}
}
