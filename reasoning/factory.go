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
		// Future engines can be added here
	}
}
