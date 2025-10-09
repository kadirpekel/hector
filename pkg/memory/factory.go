package memory

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/llms"
)

// WorkingMemoryConfig contains configuration for creating a working memory strategy
type WorkingMemoryConfig struct {
	Strategy   string
	WindowSize int
	Budget     int
	Threshold  float64
	Target     float64
	Model      string
	LLM        llms.LLMProvider
	Summarizer SummarizationService
}

// NewWorkingMemoryStrategy creates a new working memory strategy based on configuration
func NewWorkingMemoryStrategy(config WorkingMemoryConfig) (WorkingMemoryStrategy, error) {
	// Default to summary_buffer if not specified
	strategy := config.Strategy
	if strategy == "" {
		strategy = "summary_buffer"
	}

	switch strategy {
	case "buffer_window":
		return NewBufferWindowStrategy(BufferWindowConfig{
			WindowSize: config.WindowSize,
		})

	case "summary_buffer":
		// Validate required fields for summary_buffer
		if config.Model == "" {
			return nil, fmt.Errorf("model is required for summary_buffer strategy")
		}
		if config.Summarizer == nil {
			return nil, fmt.Errorf("summarizer is required for summary_buffer strategy")
		}

		return NewSummaryBufferStrategy(SummaryBufferConfig{
			Budget:     config.Budget,
			Threshold:  config.Threshold,
			Target:     config.Target,
			Model:      config.Model,
			LLM:        config.LLM,
			Summarizer: config.Summarizer,
		})

	default:
		return nil, fmt.Errorf("unknown working memory strategy: %s (supported: 'buffer_window', 'summary_buffer')", strategy)
	}
}
