package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/memory"
)

// MemoryBuilder provides a fluent API for building memory strategies
type MemoryBuilder struct {
	workingStrategy  memory.WorkingMemoryStrategy
	longTermStrategy memory.LongTermMemoryStrategy
	longTermConfig   memory.LongTermConfig
}

// NewMemory creates a new memory builder
func NewMemory() *MemoryBuilder {
	return &MemoryBuilder{
		longTermConfig: memory.LongTermConfig{
			Enabled: false,
		},
	}
}

// Working sets the working memory strategy
func (b *MemoryBuilder) Working(strategy memory.WorkingMemoryStrategy) *MemoryBuilder {
	b.workingStrategy = strategy
	return b
}

// LongTerm sets the long-term memory strategy
func (b *MemoryBuilder) LongTerm(strategy memory.LongTermMemoryStrategy, config memory.LongTermConfig) *MemoryBuilder {
	b.longTermStrategy = strategy
	b.longTermConfig = config
	return b
}

// Build returns the memory strategies (used by agent builder)
func (b *MemoryBuilder) Build() (memory.WorkingMemoryStrategy, memory.LongTermMemoryStrategy, memory.LongTermConfig) {
	return b.workingStrategy, b.longTermStrategy, b.longTermConfig
}

// WorkingMemoryBuilder provides builders for working memory strategies
type WorkingMemoryBuilder struct {
	strategyType string
	windowSize   int
	budget       int
	threshold    float64
	target       float64
	llmProvider  llms.LLMProvider
}

// NewWorkingMemory creates a builder for working memory strategies
func NewWorkingMemory(strategyType string) *WorkingMemoryBuilder {
	return &WorkingMemoryBuilder{
		strategyType: strategyType,
		budget:       2000,
		threshold:    0.8,
		target:       0.6,
		windowSize:   20,
	}
}

// WindowSize sets the window size for buffer_window strategy
func (b *WorkingMemoryBuilder) WindowSize(size int) *WorkingMemoryBuilder {
	b.windowSize = size
	return b
}

// Budget sets the token budget for summary_buffer strategy
func (b *WorkingMemoryBuilder) Budget(budget int) *WorkingMemoryBuilder {
	b.budget = budget
	return b
}

// Threshold sets the threshold for summary_buffer strategy
func (b *WorkingMemoryBuilder) Threshold(threshold float64) *WorkingMemoryBuilder {
	b.threshold = threshold
	return b
}

// Target sets the target for summary_buffer strategy
func (b *WorkingMemoryBuilder) Target(target float64) *WorkingMemoryBuilder {
	b.target = target
	return b
}

// WithLLMProvider sets the LLM provider (required for summary_buffer)
func (b *WorkingMemoryBuilder) WithLLMProvider(provider llms.LLMProvider) *WorkingMemoryBuilder {
	b.llmProvider = provider
	return b
}

// Build creates the working memory strategy
func (b *WorkingMemoryBuilder) Build() (memory.WorkingMemoryStrategy, error) {
	switch b.strategyType {
	case "buffer_window":
		return memory.NewBufferWindowStrategy(memory.BufferWindowConfig{
			WindowSize: b.windowSize,
		})

	case "summary_buffer", "":
		if b.llmProvider == nil {
			return nil, fmt.Errorf("LLM provider is required for summary_buffer strategy")
		}

		// Create summarization service
		summarizer, err := createSummarizationService(b.llmProvider)
		if err != nil {
			return nil, fmt.Errorf("failed to create summarization service: %w", err)
		}

		return memory.NewSummaryBufferStrategy(memory.SummaryBufferConfig{
			Budget:     b.budget,
			Threshold:  b.threshold,
			Target:     b.target,
			Model:      b.llmProvider.GetModelName(),
			Summarizer: summarizer,
		})

	default:
		return nil, fmt.Errorf("unknown working memory strategy: %s", b.strategyType)
	}
}

// createSummarizationService creates a summarization service from LLM provider
func createSummarizationService(llmProvider llms.LLMProvider) (memory.SummarizationService, error) {
	summarizer, err := agent.NewSummarizationService(llmProvider, &agent.SummarizationConfig{
		Model: llmProvider.GetModelName(),
	})
	if err != nil {
		return nil, err
	}
	// agent.SummarizationService implements memory.SummarizationService
	return summarizer, nil
}

// LongTermMemoryBuilder provides a builder for long-term memory strategies
type LongTermMemoryBuilder struct {
	db         databases.DatabaseProvider
	embedder   embedders.EmbedderProvider
	collection string
	enabled    bool
	config     memory.LongTermConfig
}

// NewLongTermMemory creates a builder for long-term memory
func NewLongTermMemory() *LongTermMemoryBuilder {
	return &LongTermMemoryBuilder{
		collection: "hector_session_memory",
		enabled:    false,
		config: memory.LongTermConfig{
			Enabled:      false,
			StorageScope: memory.StorageScopeAll,
			BatchSize:    1,
			AutoRecall:   false,
			RecallLimit:  5,
		},
	}
}

// WithDatabase sets the database provider
func (b *LongTermMemoryBuilder) WithDatabase(db databases.DatabaseProvider) *LongTermMemoryBuilder {
	b.db = db
	return b
}

// WithEmbedder sets the embedder provider
func (b *LongTermMemoryBuilder) WithEmbedder(embedder embedders.EmbedderProvider) *LongTermMemoryBuilder {
	b.embedder = embedder
	return b
}

// Collection sets the collection name
func (b *LongTermMemoryBuilder) Collection(name string) *LongTermMemoryBuilder {
	b.collection = name
	b.config.Collection = name
	return b
}

// Enabled enables long-term memory
func (b *LongTermMemoryBuilder) Enabled(enabled bool) *LongTermMemoryBuilder {
	b.enabled = enabled
	b.config.Enabled = enabled
	return b
}

// StorageScope sets the storage scope
func (b *LongTermMemoryBuilder) StorageScope(scope memory.StorageScope) *LongTermMemoryBuilder {
	b.config.StorageScope = scope
	return b
}

// BatchSize sets the batch size
func (b *LongTermMemoryBuilder) BatchSize(size int) *LongTermMemoryBuilder {
	b.config.BatchSize = size
	return b
}

// AutoRecall enables automatic recall
func (b *LongTermMemoryBuilder) AutoRecall(enabled bool) *LongTermMemoryBuilder {
	b.config.AutoRecall = enabled
	return b
}

// RecallLimit sets the recall limit
func (b *LongTermMemoryBuilder) RecallLimit(limit int) *LongTermMemoryBuilder {
	b.config.RecallLimit = limit
	return b
}

// Build creates the long-term memory strategy
func (b *LongTermMemoryBuilder) Build() (memory.LongTermMemoryStrategy, memory.LongTermConfig, error) {
	if !b.enabled {
		return nil, b.config, nil
	}

	if b.db == nil {
		return nil, b.config, fmt.Errorf("database provider is required for long-term memory")
	}
	if b.embedder == nil {
		return nil, b.config, fmt.Errorf("embedder provider is required for long-term memory")
	}

	strategy, err := memory.NewVectorMemoryStrategy(b.db, b.embedder, b.collection)
	if err != nil {
		return nil, b.config, fmt.Errorf("failed to create vector memory strategy: %w", err)
	}

	return strategy, b.config, nil
}
