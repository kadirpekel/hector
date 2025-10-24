package memory

import (
	"fmt"
)

type WorkingMemoryConfig struct {
	Strategy   string
	WindowSize int
	Budget     int
	Threshold  float64
	Target     float64
	Model      string
	Summarizer SummarizationService
}

func NewWorkingMemoryStrategy(config WorkingMemoryConfig) (WorkingMemoryStrategy, error) {

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
			Summarizer: config.Summarizer,
		})

	default:
		return nil, fmt.Errorf("unknown working memory strategy: %s (supported: 'buffer_window', 'summary_buffer')", strategy)
	}
}
