// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package builder

import (
	"fmt"

	"github.com/kadirpekel/hector/v2/memory"
	"github.com/kadirpekel/hector/v2/model"
)

// WorkingMemoryBuilder provides a fluent API for building working memory strategies.
//
// Example:
//
//	strategy, err := builder.NewWorkingMemory("summary_buffer").
//	    Budget(8000).
//	    Threshold(0.85).
//	    WithLLM(llm).
//	    Build()
type WorkingMemoryBuilder struct {
	strategyType string
	windowSize   int
	budget       int
	threshold    float64
	target       float64
	llm          model.LLM
}

// NewWorkingMemory creates a new working memory builder.
//
// Supported strategies:
//   - "buffer_window": Simple sliding window of recent messages
//   - "token_window": Token-based window management
//   - "summary_buffer": Summarization-based memory (requires LLM)
//
// Example:
//
//	strategy, err := builder.NewWorkingMemory("buffer_window").
//	    WindowSize(20).
//	    Build()
func NewWorkingMemory(strategyType string) *WorkingMemoryBuilder {
	return &WorkingMemoryBuilder{
		strategyType: strategyType,
		windowSize:   20,
		budget:       8000,
		threshold:    0.85,
		target:       0.6,
	}
}

// WindowSize sets the window size for buffer_window strategy.
//
// Example:
//
//	builder.NewWorkingMemory("buffer_window").WindowSize(30)
func (b *WorkingMemoryBuilder) WindowSize(size int) *WorkingMemoryBuilder {
	if size <= 0 {
		panic("window size must be positive")
	}
	b.windowSize = size
	return b
}

// Budget sets the token budget for summary_buffer strategy.
//
// Example:
//
//	builder.NewWorkingMemory("summary_buffer").Budget(8000)
func (b *WorkingMemoryBuilder) Budget(budget int) *WorkingMemoryBuilder {
	if budget <= 0 {
		panic("budget must be positive")
	}
	b.budget = budget
	return b
}

// Threshold sets the threshold for triggering summarization.
// When token usage exceeds this percentage of budget, summarization is triggered.
//
// Example:
//
//	builder.NewWorkingMemory("summary_buffer").Threshold(0.85)
func (b *WorkingMemoryBuilder) Threshold(threshold float64) *WorkingMemoryBuilder {
	if threshold <= 0 || threshold > 1 {
		panic("threshold must be between 0 and 1")
	}
	b.threshold = threshold
	return b
}

// Target sets the target percentage after summarization.
//
// Example:
//
//	builder.NewWorkingMemory("summary_buffer").Target(0.6)
func (b *WorkingMemoryBuilder) Target(target float64) *WorkingMemoryBuilder {
	if target <= 0 || target > 1 {
		panic("target must be between 0 and 1")
	}
	b.target = target
	return b
}

// WithLLM sets the LLM for summarization (required for summary_buffer).
//
// Example:
//
//	builder.NewWorkingMemory("summary_buffer").WithLLM(llm)
func (b *WorkingMemoryBuilder) WithLLM(llm model.LLM) *WorkingMemoryBuilder {
	b.llm = llm
	return b
}

// Build creates the working memory strategy.
//
// Returns an error if required parameters are missing.
func (b *WorkingMemoryBuilder) Build() (memory.WorkingMemoryStrategy, error) {
	switch b.strategyType {
	case "buffer_window":
		return memory.NewBufferWindowStrategy(memory.BufferWindowConfig{
			WindowSize: b.windowSize,
		}), nil

	case "token_window":
		return memory.NewTokenWindowStrategy(memory.TokenWindowConfig{
			Budget: b.budget,
		})

	case "summary_buffer", "":
		if b.llm == nil {
			return nil, fmt.Errorf("LLM is required for summary_buffer strategy")
		}
		summarizer, err := memory.NewLLMSummarizer(memory.LLMSummarizerConfig{
			LLM: b.llm,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create summarizer: %w", err)
		}
		return memory.NewSummaryBufferStrategy(memory.SummaryBufferConfig{
			Budget:     b.budget,
			Threshold:  b.threshold,
			Target:     b.target,
			Model:      b.llm.Name(),
			Summarizer: summarizer,
		})

	default:
		return nil, fmt.Errorf("unknown working memory strategy: %s (supported: buffer_window, token_window, summary_buffer)", b.strategyType)
	}
}

// MustBuild creates the working memory strategy or panics on error.
func (b *WorkingMemoryBuilder) MustBuild() memory.WorkingMemoryStrategy {
	strategy, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build working memory: %v", err))
	}
	return strategy
}

