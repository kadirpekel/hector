package agent

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/utils"
)

// ============================================================================
// SMART HISTORY SELECTION
// Intelligently selects which messages to include in context
// ============================================================================

// HistorySelector handles intelligent message selection
type HistorySelector struct {
	tokenCounter    *utils.TokenCounter
	summarizer      *SummarizationService
	enableSummarize bool
}

// HistoryConfig configures smart history selection
type HistoryConfig struct {
	Model                string                // Model for token counting
	EnableSummarization  bool                  // Enable automatic summarization
	SummarizationService *SummarizationService // Optional pre-configured summarizer
}

// NewHistorySelector creates a new smart history selector
func NewHistorySelector(config *HistoryConfig) (*HistorySelector, error) {
	if config == nil {
		config = &HistoryConfig{
			Model:               "gpt-4o",
			EnableSummarization: false,
		}
	}

	if config.Model == "" {
		config.Model = "gpt-4o"
	}

	// Create token counter
	tokenCounter, err := utils.NewTokenCounter(config.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create token counter: %w", err)
	}

	return &HistorySelector{
		tokenCounter:    tokenCounter,
		summarizer:      config.SummarizationService,
		enableSummarize: config.EnableSummarization && config.SummarizationService != nil,
	}, nil
}

// SelectMessagesStrategy defines how messages are selected
type SelectMessagesStrategy string

const (
	// StrategyRecent - Keep most recent messages (default)
	StrategyRecent SelectMessagesStrategy = "recent"

	// StrategyImportant - Keep important messages (system, errors, decisions)
	StrategyImportant SelectMessagesStrategy = "important"

	// StrategyBalanced - Balance between recent and important
	StrategyBalanced SelectMessagesStrategy = "balanced"

	// StrategySummarize - Summarize old, keep recent
	StrategySummarize SelectMessagesStrategy = "summarize"
)

// SelectionOptions configures message selection
type SelectionOptions struct {
	MaxTokens          int                    // Token budget
	Strategy           SelectMessagesStrategy // Selection strategy
	KeepRecentCount    int                    // Number of recent messages to always keep (for summarize strategy)
	PreserveSystem     bool                   // Always preserve system messages
	PreserveErrors     bool                   // Always preserve error messages
	SummarizeThreshold float64                // Percentage threshold to trigger summarization (0.8 = 80%)
}

// SelectMessages intelligently selects messages that fit within token budget
func (s *HistorySelector) SelectMessages(ctx context.Context, messages []llms.Message, opts *SelectionOptions) ([]llms.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	if opts == nil {
		opts = &SelectionOptions{
			MaxTokens:          2000,
			Strategy:           StrategyRecent,
			KeepRecentCount:    5,
			PreserveSystem:     true,
			PreserveErrors:     false,
			SummarizeThreshold: 0.8,
		}
	}

	// Set defaults
	if opts.MaxTokens <= 0 {
		opts.MaxTokens = 2000
	}
	if opts.Strategy == "" {
		opts.Strategy = StrategyRecent
	}
	if opts.KeepRecentCount <= 0 {
		opts.KeepRecentCount = 5
	}
	if opts.SummarizeThreshold <= 0 || opts.SummarizeThreshold > 1 {
		opts.SummarizeThreshold = 0.8
	}

	// Convert to utils.Message for token counting
	utilsMessages := make([]utils.Message, len(messages))
	for i, msg := range messages {
		utilsMessages[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Check if we need to reduce messages
	totalTokens := s.tokenCounter.CountMessages(utilsMessages)
	if totalTokens <= opts.MaxTokens {
		return messages, nil // All messages fit
	}

	// Select based on strategy
	switch opts.Strategy {
	case StrategyRecent:
		return s.selectRecent(messages, opts.MaxTokens), nil

	case StrategyImportant:
		return s.selectImportant(messages, opts), nil

	case StrategyBalanced:
		return s.selectBalanced(messages, opts), nil

	case StrategySummarize:
		if s.enableSummarize && s.summarizer != nil {
			return s.selectWithSummarization(ctx, messages, opts)
		}
		// Fallback to recent if summarization not available
		return s.selectRecent(messages, opts.MaxTokens), nil

	default:
		return s.selectRecent(messages, opts.MaxTokens), nil
	}
}

// selectRecent keeps the most recent messages
func (s *HistorySelector) selectRecent(messages []llms.Message, maxTokens int) []llms.Message {
	utilsMessages := make([]utils.Message, len(messages))
	for i, msg := range messages {
		utilsMessages[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	fitted := s.tokenCounter.FitWithinLimit(utilsMessages, maxTokens)

	// Convert back
	result := make([]llms.Message, len(fitted))
	for i, msg := range fitted {
		result[i] = llms.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return result
}

// selectImportant keeps important messages (system, errors, tool calls)
func (s *HistorySelector) selectImportant(messages []llms.Message, opts *SelectionOptions) []llms.Message {
	// First, collect important messages
	important := []llms.Message{}
	regular := []llms.Message{}

	for _, msg := range messages {
		if s.isImportant(msg, opts) {
			important = append(important, msg)
		} else {
			regular = append(regular, msg)
		}
	}

	// Calculate tokens for important messages
	importantUtils := make([]utils.Message, len(important))
	for i, msg := range important {
		importantUtils[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	importantTokens := s.tokenCounter.CountMessages(importantUtils)

	// If important messages alone exceed budget, trim them
	if importantTokens > opts.MaxTokens {
		return s.selectRecent(important, opts.MaxTokens)
	}

	// Fill remaining budget with recent regular messages
	remainingBudget := opts.MaxTokens - importantTokens
	recentRegular := s.selectRecent(regular, remainingBudget)

	// Merge: important first, then recent regular
	result := append([]llms.Message{}, important...)
	result = append(result, recentRegular...)

	return result
}

// selectBalanced balances between important and recent messages
func (s *HistorySelector) selectBalanced(messages []llms.Message, opts *SelectionOptions) []llms.Message {
	// Split budget: 40% important, 60% recent
	importantBudget := int(float64(opts.MaxTokens) * 0.4)
	recentBudget := int(float64(opts.MaxTokens) * 0.6)

	// Get important messages
	importantOpts := &SelectionOptions{
		MaxTokens:      importantBudget,
		PreserveSystem: opts.PreserveSystem,
		PreserveErrors: opts.PreserveErrors,
	}
	important := s.selectImportant(messages, importantOpts)

	// Calculate actual tokens used by important
	importantUtils := make([]utils.Message, len(important))
	for i, msg := range important {
		importantUtils[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	actualImportantTokens := s.tokenCounter.CountMessages(importantUtils)

	// Adjust recent budget based on actual important tokens
	adjustedRecentBudget := opts.MaxTokens - actualImportantTokens
	if adjustedRecentBudget < recentBudget {
		adjustedRecentBudget = recentBudget
	}

	// Get recent messages (excluding important)
	regularMessages := []llms.Message{}
	importantSet := make(map[string]bool)
	for _, msg := range important {
		importantSet[msg.Role+":"+msg.Content] = true
	}

	for _, msg := range messages {
		key := msg.Role + ":" + msg.Content
		if !importantSet[key] {
			regularMessages = append(regularMessages, msg)
		}
	}

	recent := s.selectRecent(regularMessages, adjustedRecentBudget)

	// Merge results
	result := append([]llms.Message{}, important...)
	result = append(result, recent...)

	return result
}

// selectWithSummarization summarizes old messages and keeps recent ones
func (s *HistorySelector) selectWithSummarization(ctx context.Context, messages []llms.Message, opts *SelectionOptions) ([]llms.Message, error) {
	if s.summarizer == nil {
		return s.selectRecent(messages, opts.MaxTokens), nil
	}

	// Check if we need to summarize
	utilsMessages := make([]utils.Message, len(messages))
	for i, msg := range messages {
		utilsMessages[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	totalTokens := s.tokenCounter.CountMessages(utilsMessages)

	thresholdTokens := int(float64(opts.MaxTokens) * opts.SummarizeThreshold)
	if totalTokens < thresholdTokens {
		// Below threshold, no need to summarize
		return messages, nil
	}

	// Summarize old messages, keep recent
	summarizedHistory, err := s.summarizer.SummarizeWithRecentContext(ctx, messages, opts.KeepRecentCount)
	if err != nil {
		// Fallback to recent on error
		return s.selectRecent(messages, opts.MaxTokens), nil
	}

	// Convert back to messages
	result := summarizedHistory.ToMessages()

	// Verify it fits
	resultUtils := make([]utils.Message, len(result))
	for i, msg := range result {
		resultUtils[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	resultTokens := s.tokenCounter.CountMessages(resultUtils)

	if resultTokens <= opts.MaxTokens {
		return result, nil
	}

	// If still too large, trim recent messages
	return s.selectRecent(result, opts.MaxTokens), nil
}

// isImportant determines if a message is important
func (s *HistorySelector) isImportant(msg llms.Message, opts *SelectionOptions) bool {
	// System messages
	if opts.PreserveSystem && msg.Role == "system" {
		return true
	}

	// Error messages
	if opts.PreserveErrors {
		content := strings.ToLower(msg.Content)
		if strings.Contains(content, "error") ||
			strings.Contains(content, "failed") ||
			strings.Contains(content, "exception") {
			return true
		}
	}

	// Tool calls and responses
	if len(msg.ToolCalls) > 0 || msg.ToolCallID != "" {
		return true
	}

	// Decision points (contains keywords)
	content := strings.ToLower(msg.Content)
	decisionKeywords := []string{
		"decided", "choose", "selected", "opted",
		"concluded", "determined", "resolved",
	}
	for _, keyword := range decisionKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	return false
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// EstimateReduction estimates how much reduction is needed
func (s *HistorySelector) EstimateReduction(messages []llms.Message, targetTokens int) float64 {
	utilsMessages := make([]utils.Message, len(messages))
	for i, msg := range messages {
		utilsMessages[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	currentTokens := s.tokenCounter.CountMessages(utilsMessages)
	if currentTokens <= targetTokens {
		return 0.0
	}

	reduction := float64(currentTokens-targetTokens) / float64(currentTokens)
	return math.Min(reduction, 1.0)
}

// GetRecommendedStrategy recommends a selection strategy based on message characteristics
func (s *HistorySelector) GetRecommendedStrategy(messages []llms.Message, opts *SelectionOptions) SelectMessagesStrategy {
	if len(messages) == 0 {
		return StrategyRecent
	}

	// Count important messages
	importantCount := 0
	for _, msg := range messages {
		if s.isImportant(msg, opts) {
			importantCount++
		}
	}

	importantRatio := float64(importantCount) / float64(len(messages))

	// High ratio of important messages -> use important strategy
	if importantRatio > 0.5 {
		return StrategyImportant
	}

	// Many messages with summarization enabled -> use summarize
	if len(messages) > 20 && s.enableSummarize {
		return StrategySummarize
	}

	// Medium mix -> use balanced
	if importantRatio > 0.2 {
		return StrategyBalanced
	}

	// Default to recent
	return StrategyRecent
}
