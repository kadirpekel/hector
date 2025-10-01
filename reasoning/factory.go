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
	case "default":
		return NewDefaultReasoningEngine(services), nil
	case "chain-of-thought":
		// Get configuration for chain-of-thought engine
		config := services.GetConfig()
		maxIterations := config.MaxIterations
		loopDetection := true // Default, could be made configurable in the future

		return NewChainOfThoughtReasoningEngine(services, maxIterations, loopDetection), nil
	default:
		return nil, fmt.Errorf("unsupported reasoning engine type: %s", engineType)
	}
}

// ListAvailableEngines returns information about all available reasoning engines
func (f *DefaultReasoningEngineFactory) ListAvailableEngines() []ReasoningEngineInfo {
	return []ReasoningEngineInfo{
		{
			Name:        "default",
			Description: "Clean default reasoning engine using agent services for all operations",
			Features: []string{
				"Document search integration",
				"Conversation history",
				"Available tools listing",
				"Tool execution with display preferences",
				"Direct response generation",
				"Streaming support",
			},
			Parameters: []ReasoningParameter{
				{
					Name:        "max_iterations",
					Type:        "int",
					Description: "Maximum number of reasoning iterations (always 1 for default)",
					Required:    false,
					Default:     1,
				},
			},
			Examples: []ReasoningExample{
				{
					Name:        "Weather Query",
					Description: "Weather query with tool usage",
					Config: config.ReasoningConfig{
						Engine: "default",
					},
					Query: "how is the weather in Berlin today?",
				},
				{
					Name:        "File Operations",
					Description: "File system operations",
					Config: config.ReasoningConfig{
						Engine: "default",
					},
					Query: "how many files are in the current directory?",
				},
				{
					Name:        "Context Search",
					Description: "Search through documents",
					Config: config.ReasoningConfig{
						Engine: "default",
					},
					Query: "what is the main function in this codebase?",
				},
			},
		},
		{
			Name:        "chain-of-thought",
			Description: "Advanced reasoning engine that can recursively call itself to create chains of thought, enabling deep analysis and meta-cognitive reasoning",
			Features: []string{
				"Recursive self-calling capability",
				"Chain-of-thought reasoning",
				"Loop detection and prevention",
				"Meta-cognitive reasoning",
				"Deep problem decomposition",
				"Alternative approach exploration",
				"Reasoning verification",
				"Configurable iteration limits",
			},
			Parameters: []ReasoningParameter{
				{
					Name:        "max_iterations",
					Type:        "int",
					Description: "Maximum number of reasoning iterations to prevent infinite loops",
					Required:    false,
					Default:     5,
				},
				{
					Name:        "loop_detection",
					Type:        "bool",
					Description: "Enable loop detection to prevent repetitive reasoning",
					Required:    false,
					Default:     true,
				},
			},
			Examples: []ReasoningExample{
				{
					Name:        "Complex Problem Analysis",
					Description: "Breaking down complex problems into smaller parts",
					Config: config.ReasoningConfig{
						Engine:        "chain-of-thought",
						MaxIterations: 3,
					},
					Query: "What are the implications of implementing a new AI system in our organization?",
				},
				{
					Name:        "Meta-Cognitive Reasoning",
					Description: "Thinking about thinking and reasoning processes",
					Config: config.ReasoningConfig{
						Engine:        "chain-of-thought",
						MaxIterations: 4,
					},
					Query: "How can I improve my reasoning process for better decision making?",
				},
				{
					Name:        "Alternative Exploration",
					Description: "Exploring different approaches to a problem",
					Config: config.ReasoningConfig{
						Engine:        "chain-of-thought",
						MaxIterations: 6,
					},
					Query: "What are all the possible ways to solve this technical challenge?",
				},
			},
		},
		// Future engines can be added here
	}
}
