package utils

import (
	"fmt"
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

type TokenCounter struct {
	encoding *tiktoken.Tiktoken
	model    string
	mu       sync.RWMutex
}

type Message struct {
	Role    string
	Content string
}

var (
	encodingCache = make(map[string]*tiktoken.Tiktoken)
	cacheMu       sync.RWMutex
)

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

func (tc *TokenCounter) Count(text string) int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	tokens := tc.encoding.Encode(text, nil, nil)
	return len(tokens)
}

func (tc *TokenCounter) CountMessages(messages []Message) int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	tokensPerMessage := 3

	totalTokens := 0
	for _, msg := range messages {
		totalTokens += tokensPerMessage
		totalTokens += len(tc.encoding.Encode(msg.Role, nil, nil))
		totalTokens += len(tc.encoding.Encode(msg.Content, nil, nil))
	}

	totalTokens += 3

	return totalTokens
}

func (tc *TokenCounter) FitWithinLimit(messages []Message, maxTokens int) []Message {
	if len(messages) == 0 {
		return messages
	}

	fitted := []Message{}
	currentTokens := 0

	currentTokens += 3

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

func (tc *TokenCounter) EstimateTokensForText(text string) int {
	if tc == nil || tc.encoding == nil {

		return len(text) / 4
	}
	return tc.Count(text)
}

func (tc *TokenCounter) GetModel() string {
	return tc.model
}

func EstimateTokens(text string) int {

	return len(text) / 4
}

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

	for modelPrefix, encoding := range encodingMap {
		if len(model) >= len(modelPrefix) && model[:len(modelPrefix)] == modelPrefix {
			return encoding
		}
	}

	return "cl100k_base"
}
