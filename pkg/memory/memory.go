package memory

import (
	"fmt"
	"log"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// MemoryService manages conversation memory using SessionService
// Session lifecycle is delegated to SessionService (cleaner separation of concerns)
type MemoryService struct {
	sessionService reasoning.SessionService
	workingMemory  WorkingMemoryStrategy
	longTermMemory LongTermMemoryStrategy // Optional (can be nil)

	// Long-term memory configuration
	longTermConfig LongTermConfig

	// Pending batches per session (for long-term memory batching)
	// Key: sessionID, Value: pending messages to be flushed
	pendingBatches map[string][]*pb.Message
}

// NewMemoryService creates a new memory service using SessionService
func NewMemoryService(
	sessionService reasoning.SessionService,
	working WorkingMemoryStrategy,
	longTerm LongTermMemoryStrategy,
	longTermConfig LongTermConfig,
) *MemoryService {
	// Apply defaults
	longTermConfig.SetDefaults()

	// Handle AutoRecall default
	if longTerm == nil {
		longTermConfig.AutoRecall = false
	} else if longTermConfig.AutoRecall {
		longTermConfig.AutoRecall = true // Default to true when long-term is enabled
	}

	return &MemoryService{
		sessionService: sessionService,
		workingMemory:  working,
		longTermMemory: longTerm,
		longTermConfig: longTermConfig,
		pendingBatches: make(map[string][]*pb.Message),
	}
}

// AddToHistory adds a message to memory
func (s *MemoryService) AddToHistory(sessionID string, msg *pb.Message) error {
	if sessionID == "" {
		sessionID = "default"
	}

	// 1. Append message directly to session store (efficient!)
	// This replaces the old bulk update approach
	if err := s.sessionService.AppendMessage(sessionID, msg); err != nil {
		log.Printf("⚠️  Failed to append message to session: %v", err)
		return fmt.Errorf("failed to append message: %w", err)
	}

	// 2. Let working memory strategy process the message (for summarization, etc.)
	// Get current history state and let strategy process it
	allMessages, err := s.sessionService.GetMessages(sessionID, 0)
	if err == nil {
		history := s.messagesToConversationHistory(sessionID, allMessages)
		if err := s.workingMemory.AddMessage(history, msg); err != nil {
			log.Printf("⚠️  Working memory strategy AddMessage failed: %v", err)
		}
	}

	// 3. Store in long-term memory (if enabled and should store)
	if s.longTermMemory != nil && s.shouldStoreLongTerm(msg) {
		// Add to pending batch
		if _, exists := s.pendingBatches[sessionID]; !exists {
			s.pendingBatches[sessionID] = make([]*pb.Message, 0, s.longTermConfig.BatchSize)
		}
		s.pendingBatches[sessionID] = append(s.pendingBatches[sessionID], msg)

		// Flush if batch size reached
		if len(s.pendingBatches[sessionID]) >= s.longTermConfig.BatchSize {
			if err := s.flushLongTermBatch(sessionID); err != nil {
				log.Printf("⚠️  Long-term storage failed: %v", err)
			}
		}
	}

	return nil
}

// GetRecentHistory returns messages from memory
// Applies working memory strategy to filter/transform messages
func (s *MemoryService) GetRecentHistory(sessionID string) ([]*pb.Message, error) {
	if sessionID == "" {
		sessionID = "default"
	}

	// 1. Get all messages from session store
	// limit=0 means get all messages
	allMessages, err := s.sessionService.GetMessages(sessionID, 0)
	if err != nil {
		log.Printf("⚠️  Failed to get messages from session: %v", err)
		return []*pb.Message{}, nil
	}

	// 2. Apply working memory strategy to filter/transform messages
	// Convert to ConversationHistory for strategy compatibility
	history := s.messagesToConversationHistory(sessionID, allMessages)

	// Let the strategy decide which messages to return
	filteredMessages, err := s.workingMemory.GetMessages(history)
	if err != nil {
		log.Printf("⚠️  Working memory strategy failed: %v", err)
		// Fallback to all messages if strategy fails
		filteredMessages = allMessages
	}

	// 3. Auto-recall from long-term (if enabled)
	if s.longTermMemory != nil && s.longTermConfig.AutoRecall && len(filteredMessages) > 0 {
		query := s.getLastUserMessage(filteredMessages)
		if query != "" {
			recalled, err := s.longTermMemory.Recall(sessionID, query, s.longTermConfig.RecallLimit)
			if err != nil {
				log.Printf("⚠️  Long-term recall failed: %v", err)
			} else if len(recalled) > 0 {
				// Prepend recalled memories
				filteredMessages = append(recalled, filteredMessages...)
			}
		}
	}

	return filteredMessages, nil
}

// messagesToConversationHistory converts messages to ConversationHistory for strategy compatibility
func (s *MemoryService) messagesToConversationHistory(sessionID string, messages []*pb.Message) *hectorcontext.ConversationHistory {
	history, err := hectorcontext.NewConversationHistory(sessionID)
	if err != nil {
		log.Printf("⚠️  Failed to create conversation history: %v", err)
		// Return empty history if creation fails
		history, _ = hectorcontext.NewConversationHistory(sessionID)
		return history
	}

	// Add native pb.Messages directly (no conversion needed)
	for _, msg := range messages {
		if err := history.AddMessage(msg); err != nil {
			log.Printf("⚠️  Failed to add message to conversation history: %v", err)
		}
	}

	return history
}

// ClearHistory clears memory for a session
func (s *MemoryService) ClearHistory(sessionID string) error {
	if sessionID == "" {
		sessionID = "default"
	}

	// Flush any pending long-term memory batch
	if s.longTermMemory != nil && len(s.pendingBatches[sessionID]) > 0 {
		if err := s.flushLongTermBatch(sessionID); err != nil {
			log.Printf("⚠️  Failed to flush pending batch on clear: %v", err)
		}
	}

	// Delete session via SessionService
	// This automatically handles cleanup
	return s.sessionService.DeleteSession(sessionID)
}

// Removed getConversationHistory - no longer needed with message-level architecture
// Messages are stored/retrieved directly via SessionStore, not bulk session state

// shouldStoreLongTerm decides if a message should be stored in long-term memory
func (s *MemoryService) shouldStoreLongTerm(msg *pb.Message) bool {
	switch s.longTermConfig.StorageScope {
	case StorageScopeAll:
		return true
	case StorageScopeConversational:
		return msg.Role == pb.Role_ROLE_USER || msg.Role == pb.Role_ROLE_AGENT
	case StorageScopeSummariesOnly:
		return msg.Role == pb.Role_ROLE_USER || msg.Role == pb.Role_ROLE_AGENT
	default:
		return msg.Role == pb.Role_ROLE_USER || msg.Role == pb.Role_ROLE_AGENT
	}
}

// getLastUserMessage extracts the last user message as a query for recall
func (s *MemoryService) getLastUserMessage(messages []*pb.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == pb.Role_ROLE_USER {
			return protocol.ExtractTextFromMessage(messages[i])
		}
	}
	return ""
}

// SetStatusNotifier sets a status notifier on the working memory strategy
func (s *MemoryService) SetStatusNotifier(notifier StatusNotifier) {
	s.workingMemory.SetStatusNotifier(notifier)
}

// flushLongTermBatch flushes pending messages to long-term memory
func (s *MemoryService) flushLongTermBatch(sessionID string) error {
	batch := s.pendingBatches[sessionID]
	if len(batch) == 0 {
		return nil
	}

	// Store the batch
	if err := s.longTermMemory.Store(sessionID, batch); err != nil {
		return err
	}

	// Clear the pending batch
	delete(s.pendingBatches, sessionID)
	return nil
}
