package agent

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/utils"
)

// ============================================================================
// INTEGRATED CONTEXT MANAGER
// Brings together all memory improvements: token counting, summarization,
// smart selection, and compression
// ============================================================================

// ContextManager manages conversation context with all improvements
type ContextManager struct {
	tokenCounter    *utils.TokenCounter
	summarizer      *SummarizationService
	historySelector *HistorySelector
	config          *ContextManagerConfig
}

// ContextManagerConfig configures the context manager
type ContextManagerConfig struct {
	Model                string                 // Model for token counting
	MaxTokens            int                    // Maximum tokens for context
	SummarizationEnabled bool                   // Enable summarization
	SummarizeThreshold   float64                // Threshold to trigger summarization (0.8 = 80%)
	SelectionStrategy    SelectMessagesStrategy // Message selection strategy
	KeepRecentCount      int                    // Number of recent messages to always keep
	PreserveSystem       bool                   // Always preserve system messages
	PreserveErrors       bool                   // Always preserve error messages
	LLM                  llms.LLMProvider       // LLM for summarization (optional)
}

// NewContextManager creates a new context manager
func NewContextManager(config *ContextManagerConfig) (*ContextManager, error) {
	if config == nil {
		config = &ContextManagerConfig{
			Model:                "gpt-4o",
			MaxTokens:            2000,
			SummarizationEnabled: false,
			SummarizeThreshold:   0.8,
			SelectionStrategy:    StrategyRecent,
			KeepRecentCount:      5,
			PreserveSystem:       true,
			PreserveErrors:       true,
		}
	}

	// Set defaults
	if config.Model == "" {
		config.Model = "gpt-4o"
	}
	if config.MaxTokens <= 0 {
		config.MaxTokens = 2000
	}
	if config.SummarizeThreshold <= 0 || config.SummarizeThreshold > 1 {
		config.SummarizeThreshold = 0.8
	}
	if config.SelectionStrategy == "" {
		config.SelectionStrategy = StrategyRecent
	}
	if config.KeepRecentCount <= 0 {
		config.KeepRecentCount = 5
	}

	// Create token counter
	tokenCounter, err := utils.NewTokenCounter(config.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create token counter: %w", err)
	}

	// Create summarizer if enabled
	var summarizer *SummarizationService
	if config.SummarizationEnabled && config.LLM != nil {
		summarizer, err = NewSummarizationService(config.LLM, &SummarizationConfig{
			Model: config.Model,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create summarizer: %w", err)
		}
	}

	// Create smart selector
	historySelector, err := NewHistorySelector(&HistoryConfig{
		Model:                config.Model,
		EnableSummarization:  config.SummarizationEnabled,
		SummarizationService: summarizer,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create smart selector: %w", err)
	}

	return &ContextManager{
		tokenCounter:    tokenCounter,
		summarizer:      summarizer,
		historySelector: historySelector,
		config:          config,
	}, nil
}

// PrepareContext prepares messages for LLM context
// This is the main entry point that applies all improvements
func (cm *ContextManager) PrepareContext(ctx context.Context, messages []llms.Message) ([]llms.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	// Use smart selector to choose messages
	opts := &SelectionOptions{
		MaxTokens:          cm.config.MaxTokens,
		Strategy:           cm.config.SelectionStrategy,
		KeepRecentCount:    cm.config.KeepRecentCount,
		PreserveSystem:     cm.config.PreserveSystem,
		PreserveErrors:     cm.config.PreserveErrors,
		SummarizeThreshold: cm.config.SummarizeThreshold,
	}

	selected, err := cm.historySelector.SelectMessages(ctx, messages, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to select messages: %w", err)
	}

	return selected, nil
}

// GetContextStats returns statistics about the current context
func (cm *ContextManager) GetContextStats(messages []llms.Message) *ContextStats {
	if len(messages) == 0 {
		return &ContextStats{
			MaxTokens: cm.config.MaxTokens,
		}
	}

	// Convert to utils.Message for token counting
	utilsMessages := make([]utils.Message, len(messages))
	for i, msg := range messages {
		utilsMessages[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	currentTokens := cm.tokenCounter.CountMessages(utilsMessages)
	utilization := float64(currentTokens) / float64(cm.config.MaxTokens) * 100.0

	// Count important messages
	importantCount := 0
	opts := &SelectionOptions{
		PreserveSystem: cm.config.PreserveSystem,
		PreserveErrors: cm.config.PreserveErrors,
	}
	for _, msg := range messages {
		if cm.historySelector.isImportant(msg, opts) {
			importantCount++
		}
	}

	return &ContextStats{
		MessageCount:   len(messages),
		TokenCount:     currentTokens,
		MaxTokens:      cm.config.MaxTokens,
		Utilization:    utilization,
		ImportantCount: importantCount,
		NeedsReduction: currentTokens > cm.config.MaxTokens,
	}
}

// ShouldCompress determines if context should be compressed
func (cm *ContextManager) ShouldCompress(messages []llms.Message) bool {
	stats := cm.GetContextStats(messages)
	threshold := float64(cm.config.MaxTokens) * cm.config.SummarizeThreshold
	return float64(stats.TokenCount) >= threshold
}

// CompressContext compresses context using the configured strategy
func (cm *ContextManager) CompressContext(ctx context.Context, messages []llms.Message) ([]llms.Message, error) {
	if !cm.config.SummarizationEnabled || cm.summarizer == nil {
		// Fallback to simple truncation
		return cm.historySelector.selectRecent(messages, cm.config.MaxTokens), nil
	}

	// Use summarization
	summarizedHistory, err := cm.summarizer.SummarizeWithRecentContext(
		ctx,
		messages,
		cm.config.KeepRecentCount,
	)
	if err != nil {
		// Fallback on error
		return cm.historySelector.selectRecent(messages, cm.config.MaxTokens), nil
	}

	return summarizedHistory.ToMessages(), nil
}

// OptimizeContext finds the best way to fit messages in context
func (cm *ContextManager) OptimizeContext(ctx context.Context, messages []llms.Message) (*OptimizationResult, error) {
	if len(messages) == 0 {
		return &OptimizationResult{
			OriginalMessages:  messages,
			OptimizedMessages: messages,
			Strategy:          StrategyRecent,
		}, nil
	}

	originalStats := cm.GetContextStats(messages)

	// If it fits, no optimization needed
	if !originalStats.NeedsReduction {
		return &OptimizationResult{
			OriginalMessages:  messages,
			OptimizedMessages: messages,
			Strategy:          StrategyRecent,
			OriginalStats:     originalStats,
			OptimizedStats:    originalStats,
		}, nil
	}

	// Get recommended strategy
	opts := &SelectionOptions{
		PreserveSystem: cm.config.PreserveSystem,
		PreserveErrors: cm.config.PreserveErrors,
	}
	recommendedStrategy := cm.historySelector.GetRecommendedStrategy(messages, opts)

	// Apply optimization
	opts.MaxTokens = cm.config.MaxTokens
	opts.Strategy = recommendedStrategy
	opts.KeepRecentCount = cm.config.KeepRecentCount
	opts.SummarizeThreshold = cm.config.SummarizeThreshold

	optimized, err := cm.historySelector.SelectMessages(ctx, messages, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize context: %w", err)
	}

	optimizedStats := cm.GetContextStats(optimized)

	return &OptimizationResult{
		OriginalMessages:  messages,
		OptimizedMessages: optimized,
		Strategy:          recommendedStrategy,
		OriginalStats:     originalStats,
		OptimizedStats:    optimizedStats,
	}, nil
}

// ============================================================================
// DATA STRUCTURES
// ============================================================================

// ContextStats provides statistics about context usage
type ContextStats struct {
	MessageCount   int     // Number of messages
	TokenCount     int     // Total tokens used
	MaxTokens      int     // Maximum allowed tokens
	Utilization    float64 // Percentage of capacity used
	ImportantCount int     // Number of important messages
	NeedsReduction bool    // Whether reduction is needed
}

// OptimizationResult contains the results of context optimization
type OptimizationResult struct {
	OriginalMessages  []llms.Message         // Original messages
	OptimizedMessages []llms.Message         // Optimized messages
	Strategy          SelectMessagesStrategy // Strategy used
	OriginalStats     *ContextStats          // Stats before optimization
	OptimizedStats    *ContextStats          // Stats after optimization
}

// GetSavings returns token savings from optimization
func (or *OptimizationResult) GetSavings() int {
	if or.OriginalStats == nil || or.OptimizedStats == nil {
		return 0
	}
	savings := or.OriginalStats.TokenCount - or.OptimizedStats.TokenCount
	if savings < 0 {
		return 0
	}
	return savings
}

// GetReductionPercentage returns percentage of tokens reduced
func (or *OptimizationResult) GetReductionPercentage() float64 {
	if or.OriginalStats == nil || or.OriginalStats.TokenCount == 0 {
		return 0.0
	}
	savings := or.GetSavings()
	return float64(savings) / float64(or.OriginalStats.TokenCount) * 100.0
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// CompressMessages is a convenience function for quick context compression
func CompressMessages(ctx context.Context, messages []llms.Message, maxTokens int, llm llms.LLMProvider) ([]llms.Message, error) {
	config := &ContextManagerConfig{
		Model:                "gpt-4o",
		MaxTokens:            maxTokens,
		SummarizationEnabled: llm != nil,
		SummarizeThreshold:   0.8,
		SelectionStrategy:    StrategyBalanced,
		KeepRecentCount:      5,
		PreserveSystem:       true,
		PreserveErrors:       true,
		LLM:                  llm,
	}

	manager, err := NewContextManager(config)
	if err != nil {
		return nil, err
	}

	return manager.PrepareContext(ctx, messages)
}
