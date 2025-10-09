package agent

import (
	"fmt"
	"log"
	"sync"
	"time"

	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/kadirpekel/hector/pkg/utils"
)

// ============================================================================
// UNIFIED HISTORY SERVICE
// Supports both count-based and token-based memory management
// ============================================================================

// HistoryService implements reasoning.HistoryService with session awareness
// Behavior is determined by configuration:
// - tokenBudget = 0: Simple count-based (message limit)
// - tokenBudget > 0: Token-aware (accurate token counting)
// - enableSummarization: LLM-based summarization when approaching limit
type HistoryService struct {
	sessions map[string]*hectorcontext.ConversationHistory
	mu       sync.RWMutex

	// Count-based mode (always active as fallback)
	maxMessages int

	// Token-aware mode (optional)
	tokenBudget  int
	tokenCounter *utils.TokenCounter

	// Summarization mode (optional - not yet implemented)
	enableSummarization bool
	summarizer          *SummarizationService
	summarizeThreshold  float64
}

// HistoryConfig configures the history service
type HistoryConfig struct {
	// Count-based settings (fallback)
	MaxMessages int // Default: 10

	// Token-aware settings (optional)
	TokenBudget int    // Set > 0 to enable token-based memory
	Model       string // LLM model for token counting

	// Summarization settings (optional - not yet implemented)
	EnableSummarization bool             // Enable LLM-based summarization
	SummarizeThreshold  float64          // Trigger at this % of budget (default: 0.8)
	LLM                 llms.LLMProvider // LLM for summarization
}

// NewHistoryService creates a unified history service
// Behavior is determined by config:
// - No config / TokenBudget=0: count-based (simple)
// - TokenBudget>0: token-based (smart)
// - EnableSummarization: adds summarization (not yet implemented)
func NewHistoryService(config *HistoryConfig) (reasoning.HistoryService, error) {
	if config == nil {
		config = &HistoryConfig{
			MaxMessages: 10,
		}
	}

	// Set defaults
	if config.MaxMessages <= 0 {
		config.MaxMessages = 10
	}
	if config.SummarizeThreshold <= 0 || config.SummarizeThreshold > 1 {
		config.SummarizeThreshold = 0.8
	}

	service := &HistoryService{
		sessions:            make(map[string]*hectorcontext.ConversationHistory),
		maxMessages:         config.MaxMessages,
		tokenBudget:         config.TokenBudget,
		enableSummarization: config.EnableSummarization,
		summarizeThreshold:  config.SummarizeThreshold,
	}

	// Initialize token counting if budget is set
	if config.TokenBudget > 0 {
		if config.Model == "" {
			return nil, fmt.Errorf("model is required when token budget is set")
		}

		tokenCounter, err := utils.NewTokenCounter(config.Model)
		if err != nil {
			return nil, fmt.Errorf("failed to create token counter: %w", err)
		}
		service.tokenCounter = tokenCounter
		log.Printf("✅ Token-aware memory enabled (budget: %d tokens, model: %s)", config.TokenBudget, config.Model)
	} else {
		log.Printf("📝 Count-based memory enabled (max: %d messages)", config.MaxMessages)
	}

	// Initialize summarization if enabled (not yet fully implemented)
	if config.EnableSummarization {
		if config.LLM == nil {
			return nil, fmt.Errorf("LLM is required when summarization is enabled")
		}
		summarizer, err := NewSummarizationService(config.LLM, &SummarizationConfig{
			Model: config.Model,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create summarization service: %w", err)
		}
		service.summarizer = summarizer
		log.Printf("⚠️  Summarization initialized but not yet fully implemented (threshold: %.0f%%)", config.SummarizeThreshold*100)
	}

	return service, nil
}

// getOrCreateSession gets an existing session or creates a new one
// Must be called with lock held
func (s *HistoryService) getOrCreateSession(sessionID string) *hectorcontext.ConversationHistory {
	if sessionID == "" {
		sessionID = "default"
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		var err error
		session, err = hectorcontext.NewConversationHistoryWithMax(sessionID, s.maxMessages)
		if err != nil || session == nil {
			// Fallback to creating minimal session
			log.Printf("⚠️  Failed to create session %s: %v, creating minimal session", sessionID, err)
			now := time.Now()
			session = &hectorcontext.ConversationHistory{
				SessionID:   sessionID,
				Messages:    make([]hectorcontext.Message, 0),
				Context:     make(map[string]interface{}),
				LastUpdated: now,
				MaxMessages: s.maxMessages,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
		}
		s.sessions[sessionID] = session
	}
	return session
}

// getAllMessages retrieves all messages from a session
func (s *HistoryService) getAllMessages(session *hectorcontext.ConversationHistory) []llms.Message {
	// Get all messages (GetRecentMessages with large N returns all)
	contextMessages := session.GetRecentMessages(100000) // Large enough to get all

	// Convert to llms.Message format
	messages := make([]llms.Message, len(contextMessages))
	for i, msg := range contextMessages {
		messages[i] = llms.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return messages
}

// selectMessagesByCount returns the last N messages
func (s *HistoryService) selectMessagesByCount(messages []llms.Message, count int) []llms.Message {
	if count <= 0 {
		count = s.maxMessages
	}
	if count >= len(messages) {
		return messages
	}
	return messages[len(messages)-count:]
}

// selectMessagesByTokens returns messages that fit within token budget
func (s *HistoryService) selectMessagesByTokens(messages []llms.Message) []llms.Message {
	if s.tokenCounter == nil {
		return messages
	}

	// Convert to utils.Message for token counting
	utilMessages := make([]utils.Message, len(messages))
	for i, msg := range messages {
		utilMessages[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Fit within budget (FitWithinLimit handles conversion)
	fittedMessages := s.tokenCounter.FitWithinLimit(utilMessages, s.tokenBudget)

	// Convert back to llms.Message
	selectedMessages := make([]llms.Message, len(fittedMessages))
	for i, msg := range fittedMessages {
		selectedMessages[i] = llms.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return selectedMessages
}

// selectMessagesByTokensWithSummarization returns messages within budget,
// with optional summarization of old messages
func (s *HistoryService) selectMessagesByTokensWithSummarization(messages []llms.Message) []llms.Message {
	if !s.enableSummarization || s.summarizer == nil {
		// No summarization - just fit within budget
		return s.selectMessagesByTokens(messages)
	}

	// Convert to utils.Message for token counting
	utilMessages := make([]utils.Message, len(messages))
	for i, msg := range messages {
		utilMessages[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	currentTokens := s.tokenCounter.CountMessages(utilMessages)
	thresholdTokens := int(float64(s.tokenBudget) * s.summarizeThreshold)

	// Check if we should summarize
	if currentTokens > thresholdTokens {
		// Trigger summarization: keep recent messages, summarize old ones
		keepRecentCount := 5 // Keep last 5 messages intact
		if len(messages) <= keepRecentCount {
			// Not enough messages to summarize
			return s.selectMessagesByTokens(messages)
		}

		// Split into old and recent
		oldMessages := messages[:len(messages)-keepRecentCount]
		recentMessages := messages[len(messages)-keepRecentCount:]

		// Summarize old messages (this is a blocking call)
		// In production, you might want to make this async or cached
		log.Printf("🧠 Summarization triggered: %d old messages, keeping %d recent", len(oldMessages), len(recentMessages))

		// For now, just fit within budget without actual summarization
		// to avoid blocking on LLM calls during GetRecentHistory
		// TODO: Implement proper async summarization
		return s.selectMessagesByTokens(messages)
	}

	// Below threshold - just fit within budget
	return s.selectMessagesByTokens(messages)
}

// GetRecentHistory implements reasoning.HistoryService
// Returns messages based on configuration:
// - Token budget set: Returns messages within token limit
// - Token budget not set: Returns last N messages
// - Summarization enabled: Summarizes old messages when threshold exceeded
func (s *HistoryService) GetRecentHistory(sessionID string, count int) []llms.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if sessionID == "" {
		sessionID = "default"
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		return []llms.Message{}
	}

	// Get all messages from session
	allMessages := s.getAllMessages(session)
	if len(allMessages) == 0 {
		return []llms.Message{}
	}

	// Select messages based on configuration
	var selectedMessages []llms.Message
	if s.tokenBudget > 0 && s.tokenCounter != nil {
		// Token-aware mode with optional summarization
		selectedMessages = s.selectMessagesByTokensWithSummarization(allMessages)
	} else {
		// Count-based mode
		selectedMessages = s.selectMessagesByCount(allMessages, count)
	}

	return selectedMessages
}

// AddToHistory implements reasoning.HistoryService
func (s *HistoryService) AddToHistory(sessionID string, msg llms.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	session := s.getOrCreateSession(sessionID)
	_, err := session.AddMessage(msg.Role, msg.Content, nil)
	if err != nil {
		log.Printf("⚠️  Failed to add message to session %s: %v", sessionID, err)
	}
}

// ClearHistory implements reasoning.HistoryService
func (s *HistoryService) ClearHistory(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	if session, exists := s.sessions[sessionID]; exists {
		session.Clear()
	}
}

// ClearAllSessions clears all sessions (useful for testing/cleanup)
func (s *HistoryService) ClearAllSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions = make(map[string]*hectorcontext.ConversationHistory)
}

// GetSessionCount returns the number of active sessions
func (s *HistoryService) GetSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.sessions)
}

// ============================================================================
// COMPILE-TIME CHECK
// ============================================================================

var _ reasoning.HistoryService = (*HistoryService)(nil)
