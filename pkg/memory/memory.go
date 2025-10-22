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
	agentID        string // Agent identifier (for long-term memory isolation)
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
	agentID string,
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
		agentID:        agentID,
		sessionService: sessionService,
		workingMemory:  working,
		longTermMemory: longTerm,
		longTermConfig: longTermConfig,
		pendingBatches: make(map[string][]*pb.Message),
	}
}

// AddToHistory adds a single message to memory
// DEPRECATED: Use AddBatchToHistory for better performance and correct turn boundaries
// This method is kept for backward compatibility but skips summarization checks
func (s *MemoryService) AddToHistory(sessionID string, msg *pb.Message) error {
	if sessionID == "" {
		sessionID = "default"
	}

	// 1. Append message directly to session store
	if err := s.sessionService.AppendMessage(sessionID, msg); err != nil {
		log.Printf("⚠️  Failed to append message to session: %v", err)
		return fmt.Errorf("failed to append message: %w", err)
	}

	// 2. Store in long-term memory (if enabled and should store)
	s.addToLongTermBatch(sessionID, msg)

	// NOTE: No summarization check here - that's handled by AddBatchToHistory
	// at turn boundaries to prevent infinite loops

	return nil
}

// AddBatchToHistory adds multiple messages atomically at a turn boundary
// This is the CORRECT way to save messages - checks summarization ONCE per turn
func (s *MemoryService) AddBatchToHistory(sessionID string, messages []*pb.Message) error {
	if sessionID == "" {
		sessionID = "default"
	}

	if len(messages) == 0 {
		return nil
	}

	// 1. Append all messages to session store first
	for _, msg := range messages {
		if err := s.sessionService.AppendMessage(sessionID, msg); err != nil {
			log.Printf("⚠️  Failed to append message to session: %v", err)
			return fmt.Errorf("failed to append message: %w", err)
		}

		// Store in long-term memory (if enabled)
		s.addToLongTermBatch(sessionID, msg)
	}

	// 2. NOW check summarization ONCE for the entire batch
	// This prevents the infinite loop bug where each message triggers summarization
	allMessages, err := s.sessionService.GetMessages(sessionID, 0)
	if err != nil {
		log.Printf("⚠️  Failed to get messages for summarization check: %v", err)
		return nil // Don't fail the save operation
	}

	history := s.messagesToConversationHistory(sessionID, allMessages)

	// Let working memory strategy decide if summarization is needed
	// This is called ONCE per turn, not once per message
	if err := s.workingMemory.CheckAndSummarize(history); err != nil {
		log.Printf("⚠️  Summarization check failed: %v", err)
		// Don't fail - messages are already saved
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
			recalled, err := s.longTermMemory.Recall(s.agentID, sessionID, query, s.longTermConfig.RecallLimit)
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

// addToLongTermBatch adds a message to the pending long-term memory batch
// This is a helper to avoid code duplication between AddToHistory and AddBatchToHistory
func (s *MemoryService) addToLongTermBatch(sessionID string, msg *pb.Message) {
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
}

// flushLongTermBatch flushes pending messages to long-term memory
func (s *MemoryService) flushLongTermBatch(sessionID string) error {
	batch := s.pendingBatches[sessionID]
	if len(batch) == 0 {
		return nil
	}

	// Store the batch with agent isolation
	if err := s.longTermMemory.Store(s.agentID, sessionID, batch); err != nil {
		return err
	}

	// Clear the pending batch
	delete(s.pendingBatches, sessionID)
	return nil
}
