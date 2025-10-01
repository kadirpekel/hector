package reasoning

import (
	"fmt"

	"github.com/kadirpekel/hector/config"
)

// ============================================================================
// CONCRETE FACTORY IMPLEMENTATION
// ============================================================================

// DefaultReasoningEngineFactory is the default implementation of ReasoningEngineFactory
type DefaultReasoningEngineFactory struct{}

// NewReasoningEngineFactory creates a new reasoning engine factory
func NewReasoningEngineFactory() ReasoningEngineFactory {
	return &DefaultReasoningEngineFactory{}
}

// CreateEngine creates a reasoning engine of the specified type with injected services
func (f *DefaultReasoningEngineFactory) CreateEngine(engineType string, services AgentServices) (ReasoningEngine, error) {
	switch engineType {
	case "default", "":
		// Default now uses chain-of-thought (will use maxIterations from config, or 1 if not set)
		return NewChainOfThoughtReasoningEngine(services), nil
	case "chain-of-thought":
		return NewChainOfThoughtReasoningEngine(services), nil
	case "structured-reasoning":
		return NewStructuredReasoningEngine(services), nil
	default:
		return nil, fmt.Errorf("unsupported reasoning engine type: %s (available: default, chain-of-thought, structured-reasoning)", engineType)
	}
}

// ListAvailableEngines returns information about all available reasoning engines
func (f *DefaultReasoningEngineFactory) ListAvailableEngines() []ReasoningEngineInfo {
	return []ReasoningEngineInfo{
		{
			Name:        "chain-of-thought",
			Description: "Iterative reasoning engine with tool support. Use maxIterations to control: 1=simple (like old default), 3-5=standard, 5+=complex reasoning",
			Features: []string{
				"Iterative multi-pass reasoning",
				"Tool execution with automatic continuation",
				"Behavioral stopping signals",
				"Self-aware progress checking",
				"Conversation history management",
				"Document search integration",
				"Streaming support",
				"Configurable iteration depth",
			},
			Parameters: []ReasoningParameter{
				{
					Name:        "max_iterations",
					Type:        "int",
					Description: "Maximum reasoning iterations (1=simple, 3-5=standard, 5+=complex)",
					Required:    false,
					Default:     5,
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
			Examples: []ReasoningExample{
				{
					Name:        "Simple Query (like old default)",
					Description: "Single-pass reasoning",
					Config: config.ReasoningConfig{
						Engine:        "chain-of-thought",
						MaxIterations: 1,
						ShowDebugInfo: false,
					},
					Query: "What is the capital of France?",
				},
				{
					Name:        "Standard Reasoning",
					Description: "Multi-pass with tools",
					Config: config.ReasoningConfig{
						Engine:        "chain-of-thought",
						MaxIterations: 5,
						ShowDebugInfo: false,
					},
					Query: "Read the README and tell me what this project does",
				},
				{
					Name:        "Complex Analysis",
					Description: "Deep reasoning with iteration tracking",
					Config: config.ReasoningConfig{
						Engine:        "chain-of-thought",
						MaxIterations: 10,
						ShowDebugInfo: true,
					},
					Query: "Analyze the architecture and suggest improvements",
				},
			},
		},
		{
			Name:        "default",
			Description: "Alias for chain-of-thought (for backward compatibility)",
			Features:    []string{"Same as chain-of-thought"},
			Parameters:  []ReasoningParameter{},
			Examples:    []ReasoningExample{},
		},
		{
			Name:        "structured-reasoning",
			Description: "Goal-oriented reasoning with explicit planning, meta-cognition, and quality evaluation (matches Claude's actual reasoning process)",
			Features: []string{
				"Explicit goal extraction and tracking",
				"Meta-cognitive reflection after each tool use",
				"Progress tracking (accomplished vs pending goals)",
				"Quality-based stopping (confidence thresholds)",
				"Self-evaluation: 'Am I making progress?'",
				"Structured prompts with goal context",
				"Higher token usage but more thorough reasoning",
			},
			Parameters: []ReasoningParameter{
				{
					Name:        "max_iterations",
					Type:        "int",
					Description: "Maximum reasoning iterations (default: 10 for deeper analysis)",
					Required:    false,
					Default:     10,
				},
				{
					Name:        "show_debug_info",
					Type:        "bool",
					Description: "Show goals, reflection, and confidence scores",
					Required:    false,
					Default:     true,
				},
			},
			Examples: []ReasoningExample{
				{
					Name:        "Complex Analysis",
					Description: "Multi-step task with goal tracking",
					Config: config.ReasoningConfig{
						Engine:        "structured-reasoning",
						MaxIterations: 10,
						ShowDebugInfo: true,
					},
					Query: "Read files in this directory, identify 3 code quality issues, and suggest specific fixes",
				},
				{
					Name:        "Research Task",
					Description: "Deep analysis with reflection",
					Config: config.ReasoningConfig{
						Engine:        "structured-reasoning",
						MaxIterations: 15,
						ShowDebugInfo: true,
					},
					Query: "Analyze the architecture, compare it with best practices, and provide detailed recommendations",
				},
			},
		},
		// Future engines: ReAct, Tree-of-Thoughts, etc.
	}
}
