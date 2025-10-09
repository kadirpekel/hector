package history

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/llms"
)

// HistoryConfig contains configuration for creating a history strategy
type HistoryConfig struct {
	Strategy   string               // "buffer_window" or "summary_buffer" (default)
	WindowSize int                  // For buffer_window strategy
	Budget     int                  // For summary_buffer strategy
	Threshold  float64              // For summary_buffer strategy
	Target     float64              // For summary_buffer strategy
	Model      string               // Model for token counting
	LLM        llms.LLMProvider     // LLM for summarization
	Summarizer SummarizationService // Summarization service
}

// NewHistoryStrategy creates a new history strategy based on configuration
func NewHistoryStrategy(config HistoryConfig) (HistoryStrategy, error) {
	// Default to summary_buffer if not specified
	if config.Strategy == "" {
		config.Strategy = "summary_buffer"
	}

	switch config.Strategy {
	case "buffer_window":
		return NewBufferWindowStrategy(BufferWindowConfig{
			WindowSize: config.WindowSize,
		})

	case "summary_buffer":
		return NewSummaryBufferStrategy(SummaryBufferConfig{
			Budget:     config.Budget,
			Threshold:  config.Threshold,
			Target:     config.Target,
			Model:      config.Model,
			LLM:        config.LLM,
			Summarizer: config.Summarizer,
		})

	default:
		return nil, fmt.Errorf("unknown history strategy: %s (valid options: buffer_window, summary_buffer)", config.Strategy)
	}
}
