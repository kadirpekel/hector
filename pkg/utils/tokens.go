// Package utils provides utility functions for the Hector framework.
package utils

import (
	"fmt"
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

// ============================================================================
// TOKEN COUNTING - ACCURATE IMPLEMENTATION
// ============================================================================

// TokenCounter handles accurate token counting per model
type TokenCounter struct {
	encoding *tiktoken.Tiktoken
	model    string
	mu       sync.RWMutex
}

// Message represents a message for token counting
type Message struct {
	Role    string
	Content string
}

var (
	// Cache encodings to avoid repeated initialization
	encodingCache = make(map[string]*tiktoken.Tiktoken)
	cacheMu       sync.RWMutex
)

// NewTokenCounter creates a counter for specific model
func NewTokenCounter(model string) (*TokenCounter, error) {
	cacheMu.RLock()
	cached, exists := encodingCache[model]
	cacheMu.RUnlock()

	if exists {
		return &TokenCounter{
			encoding: cached,
			model:    model,
		}, nil
	}

	// Try to get encoding for the model
	encoding, err := tiktoken.EncodingForModel(model)
	if err != nil {
		// Fallback to cl100k_base (GPT-4, GPT-3.5-turbo, text-embedding-ada-002)
		encoding, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			return nil, fmt.Errorf("failed to get encoding: %w", err)
		}
	}

	// Cache the encoding
	cacheMu.Lock()
	encodingCache[model] = encoding
	cacheMu.Unlock()

	return &TokenCounter{
		encoding: encoding,
		model:    model,
	}, nil
}

// Count returns accurate token count for text
func (tc *TokenCounter) Count(text string) int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	tokens := tc.encoding.Encode(text, nil, nil)
	return len(tokens)
}

// CountMessages counts tokens in message list (includes role overhead)
// Based on OpenAI's token counting format:
// https://github.com/openai/openai-cookbook/blob/main/examples/How_to_count_tokens_with_tiktoken.ipynb
func (tc *TokenCounter) CountMessages(messages []Message) int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	// Message format overhead varies by model
	tokensPerMessage := 3 // <|start|>role|message<|end|>

	totalTokens := 0
	for _, msg := range messages {
		totalTokens += tokensPerMessage
		totalTokens += len(tc.encoding.Encode(msg.Role, nil, nil))
		totalTokens += len(tc.encoding.Encode(msg.Content, nil, nil))
	}

	// Every reply is primed with <|start|>assistant<|message|>
	totalTokens += 3

	return totalTokens
}

// FitWithinLimit returns messages that fit within token budget
// Messages are selected from most recent backwards
func (tc *TokenCounter) FitWithinLimit(messages []Message, maxTokens int) []Message {
	if len(messages) == 0 {
		return messages
	}

	// Start from most recent, work backwards
	fitted := []Message{}
	currentTokens := 0

	// Reserve tokens for the reply priming
	currentTokens += 3

	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := tc.CountMessages([]Message{messages[i]})

		if currentTokens+msgTokens > maxTokens {
			break // Can't fit more
		}

		fitted = append([]Message{messages[i]}, fitted...)
		currentTokens += msgTokens
	}

	return fitted
}

// EstimateTokensForText provides a rough estimate (for when TokenCounter isn't available)
// This is kept for backward compatibility but should be replaced with TokenCounter
func (tc *TokenCounter) EstimateTokensForText(text string) int {
	if tc == nil || tc.encoding == nil {
		// Fallback to rough estimation
		return len(text) / 4
	}
	return tc.Count(text)
}

// GetModel returns the model name this counter is configured for
func (tc *TokenCounter) GetModel() string {
	return tc.model
}

// ============================================================================
// LEGACY FUNCTION - KEPT FOR BACKWARD COMPATIBILITY
// ============================================================================

// EstimateTokens provides a rough token estimation
// DEPRECATED: Use TokenCounter.Count() for accurate counting
func EstimateTokens(text string) int {
	// Rough estimation: 4 characters per token
	// This is kept for backward compatibility but should be replaced
	return len(text) / 4
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// GetEncodingForModel returns the appropriate encoding name for a model
func GetEncodingForModel(model string) string {
	// Map common models to their encodings
	encodingMap := map[string]string{
		"gpt-4":                "cl100k_base",
		"gpt-4-turbo":          "cl100k_base",
		"gpt-4o":               "o200k_base", // GPT-4o uses o200k_base
		"gpt-4o-mini":          "o200k_base",
		"gpt-3.5-turbo":        "cl100k_base",
		"text-embedding-ada":   "cl100k_base",
		"claude":               "cl100k_base", // Approximate with OpenAI encoding
		"claude-3":             "cl100k_base",
		"claude-3-opus":        "cl100k_base",
		"claude-3-5-sonnet":    "cl100k_base",
		"gemini":               "cl100k_base", // Approximate with OpenAI encoding
		"gemini-pro":           "cl100k_base",
		"gemini-1.5-pro":       "cl100k_base",
		"gemini-2.0-flash-exp": "cl100k_base",
	}

	// Check for exact match
	if encoding, exists := encodingMap[model]; exists {
		return encoding
	}

	// Check for prefix match
	for modelPrefix, encoding := range encodingMap {
		if len(model) >= len(modelPrefix) && model[:len(modelPrefix)] == modelPrefix {
			return encoding
		}
	}

	// Default to cl100k_base
	return "cl100k_base"
}
