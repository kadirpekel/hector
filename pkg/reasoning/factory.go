package reasoning

import (
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/config"
)

func CreateStrategy(engineType string, config config.ReasoningConfig) (ReasoningStrategy, error) {
	// Normalize engine type to lowercase for case-insensitive comparison
	engineType = strings.ToLower(strings.TrimSpace(engineType))

	switch engineType {
	case "default", "", "chain-of-thought":
		return NewChainOfThoughtStrategy(), nil
	case "supervisor":
		return NewSupervisorStrategy(), nil
	default:
		return nil, fmt.Errorf("unsupported reasoning engine type: %s (supported: 'chain-of-thought', 'supervisor')", engineType)
	}
}

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
			},
		},
		{
			Name:        "supervisor",
			Description: "Multi-agent orchestration strategy optimized for delegation and coordination",
			Features: []string{
				"Specialized prompts for task decomposition",
				"Agent selection and delegation guidance",
				"Result synthesis and integration",
				"Works with agent_call tool",
				"Based on chain-of-thought with orchestration enhancements",
				"Systematic todo tracking for orchestration",
			},
			Parameters: []StrategyParameter{
				{
					Name:        "max_iterations",
					Type:        "int",
					Description: "Maximum reasoning iterations (default: 20 for multi-agent workflows)",
					Required:    false,
					Default:     20,
				},
				{
					Name:        "enable_streaming",
					Type:        "bool",
					Description: "Enable streaming output",
					Required:    false,
					Default:     true,
				},
			},
		},
	}
}

type StrategyInfo struct {
	Name        string
	Description string
	Features    []string
	Parameters  []StrategyParameter
}

type StrategyParameter struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Default     interface{}
}
