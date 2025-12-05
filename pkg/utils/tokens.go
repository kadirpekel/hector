// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel
//
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package utils provides utility functions for v2.
package utils

import (
	"fmt"
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

// TokenCounter provides token counting functionality using tiktoken.
// Ported from pkg/utils/tokens.go for use in v2.
type TokenCounter struct {
	encoding *tiktoken.Tiktoken
	model    string
	mu       sync.RWMutex
}

// Message represents a chat message for token counting.
type Message struct {
	Role    string
	Content string
}

var (
	encodingCache = make(map[string]*tiktoken.Tiktoken)
	cacheMu       sync.RWMutex
)

// NewTokenCounter creates a new token counter for the given model.
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

	encoding, err := tiktoken.EncodingForModel(model)
	if err != nil {
		// Fallback to cl100k_base for unknown models
		encoding, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			return nil, fmt.Errorf("failed to get encoding: %w", err)
		}
	}

	cacheMu.Lock()
	encodingCache[model] = encoding
	cacheMu.Unlock()

	return &TokenCounter{
		encoding: encoding,
		model:    model,
	}, nil
}

// Count returns the number of tokens in the given text.
func (tc *TokenCounter) Count(text string) int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	tokens := tc.encoding.Encode(text, nil, nil)
	return len(tokens)
}

// CountMessages returns the approximate token count for a list of messages.
// Uses the ChatML format estimation (3 tokens per message + role + content).
func (tc *TokenCounter) CountMessages(messages []Message) int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	// Tokens per message overhead (ChatML format)
	tokensPerMessage := 3

	totalTokens := 0
	for _, msg := range messages {
		totalTokens += tokensPerMessage
		totalTokens += len(tc.encoding.Encode(msg.Role, nil, nil))
		totalTokens += len(tc.encoding.Encode(msg.Content, nil, nil))
	}

	// Add 3 tokens for assistant reply priming
	totalTokens += 3

	return totalTokens
}

// FitWithinLimit returns messages that fit within the token limit.
// It works backwards from the most recent message, keeping as many as fit.
// This preserves the most recent context which is typically most relevant.
func (tc *TokenCounter) FitWithinLimit(messages []Message, maxTokens int) []Message {
	if len(messages) == 0 {
		return messages
	}

	fitted := []Message{}
	currentTokens := 0

	// Reserve tokens for reply priming
	currentTokens += 3

	// Work backwards from most recent
	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := tc.CountMessages([]Message{messages[i]})

		if currentTokens+msgTokens > maxTokens {
			break
		}

		fitted = append([]Message{messages[i]}, fitted...)
		currentTokens += msgTokens
	}

	return fitted
}

// EstimateTokensForText returns an estimate of tokens for text.
// If the counter is not initialized, falls back to character-based estimation.
func (tc *TokenCounter) EstimateTokensForText(text string) int {
	if tc == nil || tc.encoding == nil {
		// Fallback: ~4 characters per token
		return len(text) / 4
	}
	return tc.Count(text)
}

// GetModel returns the model name used by this counter.
func (tc *TokenCounter) GetModel() string {
	return tc.model
}

// EstimateTokens provides a quick token estimate without a counter.
// Uses ~4 characters per token as a rough approximation.
func EstimateTokens(text string) int {
	return len(text) / 4
}

// GetEncodingForModel returns the appropriate encoding name for a model.
func GetEncodingForModel(model string) string {
	encodingMap := map[string]string{
		"gpt-4":                "cl100k_base",
		"gpt-4-turbo":          "cl100k_base",
		"gpt-4o":               "o200k_base",
		"gpt-4o-mini":          "o200k_base",
		"gpt-3.5-turbo":        "cl100k_base",
		"text-embedding-ada":   "cl100k_base",
		"claude":               "cl100k_base",
		"claude-3":             "cl100k_base",
		"claude-3-opus":        "cl100k_base",
		"claude-3-5-sonnet":    "cl100k_base",
		"gemini":               "cl100k_base",
		"gemini-pro":           "cl100k_base",
		"gemini-1.5-pro":       "cl100k_base",
		"gemini-2.0-flash-exp": "cl100k_base",
	}

	if encoding, exists := encodingMap[model]; exists {
		return encoding
	}

	// Try prefix matching
	for modelPrefix, encoding := range encodingMap {
		if len(model) >= len(modelPrefix) && model[:len(modelPrefix)] == modelPrefix {
			return encoding
		}
	}

	return "cl100k_base"
}
