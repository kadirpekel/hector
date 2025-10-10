package memory

import (
	"log"
	"sync"
	"time"

	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/llms"
)

// MemoryService manages conversation memory across sessions
// It orchestrates working memory (recent context) and long-term memory (semantic recall)
type MemoryService struct {
	workingMemory  WorkingMemoryStrategy
	longTermMemory LongTermMemoryStrategy // Optional (can be nil)
	sessions       map[string]*hectorcontext.ConversationHistory

	// Long-term memory batching
	pendingBatch map[string][]llms.Message // sessionID -> messages
	batchSize    int                       // Default: 1 (immediate storage)
	storageScope StorageScope              // What messages to store
	autoRecall   bool                      // Auto-inject memories before LLM calls
	recallLimit  int                       // Max memories to recall

	mu sync.RWMutex
}

// NewMemoryService creates a new memory service
// working: Working memory strategy (required)
// longTerm: Long-term memory strategy (optional, can be nil)
// longTermConfig: Configuration for long-term memory
func NewMemoryService(
	working WorkingMemoryStrategy,
	longTerm LongTermMemoryStrategy,
	longTermConfig LongTermConfig,
) *MemoryService {
	// Apply defaults
	longTermConfig.SetDefaults()

	// Handle AutoRecall default (since bool zero value is false)
	autoRecall := true // Default to true
	if longTerm == nil {
		autoRecall = false // Disable if no long-term memory
	}

	return &MemoryService{
		workingMemory:  working,
		longTermMemory: longTerm,
		sessions:       make(map[string]*hectorcontext.ConversationHistory),
		pendingBatch:   make(map[string][]llms.Message),
		batchSize:      longTermConfig.BatchSize,
		storageScope:   longTermConfig.StorageScope,
		autoRecall:     autoRecall,
		recallLimit:    longTermConfig.RecallLimit,
	}
}

// AddToHistory adds a message to memory (orchestrates working + long-term)
func (s *MemoryService) AddToHistory(sessionID string, msg llms.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	session := s.getOrCreateSession(sessionID)

	// 1. Accumulate for long-term batch (if enabled)
	if s.longTermMemory != nil && s.shouldStoreLongTerm(msg) {
		s.pendingBatch[sessionID] = append(s.pendingBatch[sessionID], msg)

		// Flush when batch size reached
		// With batchSize=1, this flushes every message (immediate)
		if len(s.pendingBatch[sessionID]) >= s.batchSize {
			if err := s.flushBatch(sessionID); err != nil {
				log.Printf("⚠️  Long-term storage failed: %v", err)
			}
		}
	}

	// 2. Add to working memory (may trigger internal summarization)
	return s.workingMemory.AddMessage(session, msg)
}

// GetRecentHistory returns messages (orchestrates working + long-term recall)
func (s *MemoryService) GetRecentHistory(sessionID string) ([]llms.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if sessionID == "" {
		sessionID = "default"
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		return []llms.Message{}, nil
	}

	// 1. Get working memory (recent context)
	messages, err := s.workingMemory.GetMessages(session)
	if err != nil {
		return nil, err
	}

	// 2. Auto-recall from long-term (if enabled)
	if s.longTermMemory != nil && s.autoRecall && len(messages) > 0 {
		// Use last user message as query
		query := s.getLastUserMessage(messages)
		if query != "" {
			recalled, err := s.longTermMemory.Recall(sessionID, query, s.recallLimit)
			if err != nil {
				log.Printf("⚠️  Long-term recall failed: %v", err)
			} else if len(recalled) > 0 {
				// Prepend recalled memories (older context first)
				messages = append(recalled, messages...)
			}
		}
	}

	return messages, nil
}

// ClearHistory clears memory for a session (orchestrates both)
func (s *MemoryService) ClearHistory(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	// 1. Flush any pending batch
	if err := s.flushBatch(sessionID); err != nil {
		log.Printf("⚠️  Failed to flush batch on clear: %v", err)
	}

	// 2. Clear working memory
	delete(s.sessions, sessionID)

	// Note: We do NOT clear long-term memory here
	// Long-term memories persist across sessions for future recall
	// If you need to clear long-term memory, call longTermMemory.Clear() directly

	return nil
}

// GetSessionCount returns the number of active sessions
func (s *MemoryService) GetSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.sessions)
}

// SetStatusNotifier sets a status notifier on the working memory strategy
func (s *MemoryService) SetStatusNotifier(notifier StatusNotifier) {
	s.workingMemory.SetStatusNotifier(notifier)
}

// flushBatch flushes pending batch to long-term memory
// Must be called with lock held
func (s *MemoryService) flushBatch(sessionID string) error {
	if s.longTermMemory == nil || len(s.pendingBatch[sessionID]) == 0 {
		return nil
	}

	err := s.longTermMemory.Store(sessionID, s.pendingBatch[sessionID])
	if err != nil {
		return err
	}

	// Clear batch after successful storage
	s.pendingBatch[sessionID] = nil
	return nil
}

// shouldStoreLongTerm decides if a message should be stored in long-term memory
func (s *MemoryService) shouldStoreLongTerm(msg llms.Message) bool {
	switch s.storageScope {
	case StorageScopeAll:
		return true
	case StorageScopeConversational:
		return msg.Role == "user" || msg.Role == "assistant"
	case StorageScopeSummariesOnly:
		// Check if message has is_summary metadata (not currently supported in llms.Message)
		// For now, treat as conversational
		return msg.Role == "user" || msg.Role == "assistant"
	default:
		return msg.Role == "user" || msg.Role == "assistant"
	}
}

// getLastUserMessage extracts the last user message as a query for recall
func (s *MemoryService) getLastUserMessage(messages []llms.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	return ""
}

// getOrCreateSession gets an existing session or creates a new one
// Must be called with lock held
func (s *MemoryService) getOrCreateSession(sessionID string) *hectorcontext.ConversationHistory {
	session, exists := s.sessions[sessionID]
	if !exists {
		now := time.Now()
		session = &hectorcontext.ConversationHistory{
			SessionID:   sessionID,
			Messages:    make([]hectorcontext.Message, 0),
			Context:     make(map[string]interface{}),
			LastUpdated: now,
			MaxMessages: 1000, // Max 1000 messages per session
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		s.sessions[sessionID] = session
	}
	return session
}
