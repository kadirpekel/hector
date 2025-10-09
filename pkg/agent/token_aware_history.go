package agent

import (
	"fmt"
	"sync"
	"time"

	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/kadirpekel/hector/pkg/utils"
)

// ============================================================================
// TOKEN-AWARE HISTORY SERVICE
// Manages conversation history with accurate token counting
// ============================================================================

// TokenAwareHistoryService implements reasoning.HistoryService with token awareness
// This service accurately tracks token usage and ensures context limits are never exceeded
type TokenAwareHistoryService struct {
	sessions     map[string]*hectorcontext.ConversationHistory
	maxSize      int // Max messages per session (fallback)
	maxTokens    int // Max tokens per session
	tokenCounter *utils.TokenCounter
	mu           sync.RWMutex
}

// TokenAwareHistoryConfig configures the token-aware history service
type TokenAwareHistoryConfig struct {
	MaxMessages int    // Fallback: max message count
	MaxTokens   int    // Primary: max token count
	Model       string // LLM model for token counting
}

// NewTokenAwareHistoryService creates a new token-aware history service
func NewTokenAwareHistoryService(config *TokenAwareHistoryConfig) (reasoning.HistoryService, error) {
	if config == nil {
		config = &TokenAwareHistoryConfig{
			MaxMessages: 10,
			MaxTokens:   2000,
			Model:       "gpt-4o",
		}
	}

	// Set defaults
	if config.MaxMessages <= 0 {
		config.MaxMessages = 10
	}
	if config.MaxTokens <= 0 {
		config.MaxTokens = 2000
	}
	if config.Model == "" {
		config.Model = "gpt-4o"
	}

	// Create token counter
	tokenCounter, err := utils.NewTokenCounter(config.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create token counter: %w", err)
	}

	return &TokenAwareHistoryService{
		sessions:     make(map[string]*hectorcontext.ConversationHistory),
		maxSize:      config.MaxMessages,
		maxTokens:    config.MaxTokens,
		tokenCounter: tokenCounter,
	}, nil
}

// getOrCreateSession gets an existing session or creates a new one
func (s *TokenAwareHistoryService) getOrCreateSession(sessionID string) *hectorcontext.ConversationHistory {
	if sessionID == "" {
		sessionID = "default"
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		var err error
		session, err = hectorcontext.NewConversationHistoryWithMax(sessionID, s.maxSize)
		if err != nil || session == nil {
			session, err = hectorcontext.NewConversationHistory(sessionID)
			if err != nil || session == nil {
				// Last resort fallback
				now := time.Now()
				session = &hectorcontext.ConversationHistory{
					SessionID:   sessionID,
					Messages:    make([]hectorcontext.Message, 0),
					Context:     make(map[string]interface{}),
					LastUpdated: now,
					MaxMessages: s.maxSize,
					CreatedAt:   now,
					UpdatedAt:   now,
				}
			}
		}
		s.sessions[sessionID] = session
	}
	return session
}

// GetRecentHistory implements reasoning.HistoryService with token awareness
// Returns messages that fit within the token budget
func (s *TokenAwareHistoryService) GetRecentHistory(sessionID string, count int) []llms.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if sessionID == "" {
		sessionID = "default"
	}

	// Get session's conversation history
	session, exists := s.sessions[sessionID]
	if !exists {
		return []llms.Message{}
	}

	// Get all recent messages
	allMessages := session.GetRecentMessages(1000) // Get all messages

	if len(allMessages) == 0 {
		return []llms.Message{}
	}

	// Convert to utils.Message format for token counting
	utilsMessages := make([]utils.Message, len(allMessages))
	for i, msg := range allMessages {
		utilsMessages[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Use token counter to fit within budget
	// Use maxTokens as the budget, or count-based limit as fallback
	tokenBudget := s.maxTokens
	if count > 0 && count < len(allMessages) {
		// If count is specified and smaller, use it as additional constraint
		utilsMessages = utilsMessages[len(utilsMessages)-count:]
	}

	// Fit within token budget
	fittedMessages := s.tokenCounter.FitWithinLimit(utilsMessages, tokenBudget)

	// Convert back to llms.Message
	result := make([]llms.Message, len(fittedMessages))
	for i, msg := range fittedMessages {
		result[i] = llms.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return result
}

// GetRecentHistoryWithTokenLimit returns messages within a specific token limit
// This is useful for dynamic token allocation
func (s *TokenAwareHistoryService) GetRecentHistoryWithTokenLimit(sessionID string, maxTokens int) []llms.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if sessionID == "" {
		sessionID = "default"
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		return []llms.Message{}
	}

	// Get all messages
	allMessages := session.GetRecentMessages(1000)
	if len(allMessages) == 0 {
		return []llms.Message{}
	}

	// Convert to utils.Message
	utilsMessages := make([]utils.Message, len(allMessages))
	for i, msg := range allMessages {
		utilsMessages[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Fit within the specified token limit
	fittedMessages := s.tokenCounter.FitWithinLimit(utilsMessages, maxTokens)

	// Convert back
	result := make([]llms.Message, len(fittedMessages))
	for i, msg := range fittedMessages {
		result[i] = llms.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return result
}

// AddToHistory implements reasoning.HistoryService
func (s *TokenAwareHistoryService) AddToHistory(sessionID string, msg llms.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	// Get or create session
	session := s.getOrCreateSession(sessionID)

	// Add message to session
	_, err := session.AddMessage(msg.Role, msg.Content, nil)
	if err != nil {
		fmt.Printf("⚠️  Failed to add message to session %s: %v\n", sessionID, err)
	}

	// Optional: Trim old messages if total tokens exceed a higher threshold
	// This prevents unlimited memory growth
	s.trimIfNeeded(session)
}

// trimIfNeeded removes old messages if session exceeds token threshold
func (s *TokenAwareHistoryService) trimIfNeeded(session *hectorcontext.ConversationHistory) {
	allMessages := session.GetRecentMessages(1000)
	if len(allMessages) == 0 {
		return
	}

	// Convert to utils.Message
	utilsMessages := make([]utils.Message, len(allMessages))
	for i, msg := range allMessages {
		utilsMessages[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Count total tokens
	totalTokens := s.tokenCounter.CountMessages(utilsMessages)

	// If exceeding 2x the max tokens, trim to max tokens
	// This gives us a buffer before trimming
	threshold := s.maxTokens * 2
	if totalTokens > threshold {
		// Keep only messages that fit within maxTokens
		fitted := s.tokenCounter.FitWithinLimit(utilsMessages, s.maxTokens)

		// Update session with trimmed messages
		// Note: This is a simple approach; for production, you'd want to
		// preserve important messages or summarize old ones
		session.Messages = session.Messages[len(session.Messages)-len(fitted):]
	}
}

// ClearHistory implements reasoning.HistoryService
func (s *TokenAwareHistoryService) ClearHistory(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	if session, exists := s.sessions[sessionID]; exists {
		session.Clear()
	}
}

// GetTokenCount returns the current token count for a session
func (s *TokenAwareHistoryService) GetTokenCount(sessionID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if sessionID == "" {
		sessionID = "default"
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		return 0
	}

	allMessages := session.GetRecentMessages(1000)
	if len(allMessages) == 0 {
		return 0
	}

	// Convert and count
	utilsMessages := make([]utils.Message, len(allMessages))
	for i, msg := range allMessages {
		utilsMessages[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return s.tokenCounter.CountMessages(utilsMessages)
}

// GetSessionStats returns statistics for a session
type SessionStats struct {
	MessageCount int
	TokenCount   int
	MaxTokens    int
	Utilization  float64 // Percentage of token budget used
}

func (s *TokenAwareHistoryService) GetSessionStats(sessionID string) *SessionStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if sessionID == "" {
		sessionID = "default"
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		return &SessionStats{
			MaxTokens: s.maxTokens,
		}
	}

	messageCount := session.GetMessageCount()
	tokenCount := 0

	if messageCount > 0 {
		allMessages := session.GetRecentMessages(1000)
		utilsMessages := make([]utils.Message, len(allMessages))
		for i, msg := range allMessages {
			utilsMessages[i] = utils.Message{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}
		tokenCount = s.tokenCounter.CountMessages(utilsMessages)
	}

	utilization := 0.0
	if s.maxTokens > 0 {
		utilization = float64(tokenCount) / float64(s.maxTokens) * 100.0
	}

	return &SessionStats{
		MessageCount: messageCount,
		TokenCount:   tokenCount,
		MaxTokens:    s.maxTokens,
		Utilization:  utilization,
	}
}

// ClearAllSessions clears all sessions
func (s *TokenAwareHistoryService) ClearAllSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions = make(map[string]*hectorcontext.ConversationHistory)
}

// GetSessionCount returns the number of active sessions
func (s *TokenAwareHistoryService) GetSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.sessions)
}

// ============================================================================
// COMPILE-TIME CHECK
// ============================================================================

var _ reasoning.HistoryService = (*TokenAwareHistoryService)(nil)
