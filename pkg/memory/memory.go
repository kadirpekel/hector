package memory

import (
	"fmt"
	"log"
	"sync"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// MemoryService manages conversation memory using SessionService
// Session lifecycle is delegated to SessionService (cleaner separation of concerns)
//
// Thread-Safety: MemoryService is safe for concurrent use.
// The pendingBatches map is protected by batchMu for concurrent long-term memory operations.
type MemoryService struct {
	agentID        string // Agent identifier (for long-term memory isolation)
	sessionService reasoning.SessionService
	workingMemory  WorkingMemoryStrategy
	longTermMemory LongTermMemoryStrategy // Optional (can be nil)

	// Long-term memory configuration
	longTermConfig LongTermConfig

	// Pending batches per session (for long-term memory batching)
	// Key: sessionID, Value: pending messages to be flushed
	// Protected by batchMu for concurrent access
	batchMu        sync.RWMutex // Protects pendingBatches map
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

// AddToHistory adds a single message to the session history
//
// ⚠️ DEPRECATED: Use AddBatchToHistory instead.
//
// This method is kept for backward compatibility but should NOT be used in production code.
// Limitations:
//   - No transaction support (partial save failures possible)
//   - No summarization checks (strategy state becomes stale)
//   - Inefficient (multiple DB calls vs single batch transaction)
//
// Migrate by using: AddBatchToHistory(sessionID, []*pb.Message{msg})
func (s *MemoryService) AddToHistory(sessionID string, msg *pb.Message) error {
	if sessionID == "" {
		sessionID = "default"
	}

	// Append message directly to session store (no batch, no transaction)
	if err := s.sessionService.AppendMessage(sessionID, msg); err != nil {
		log.Printf("⚠️  Failed to append message to session: %v", err)
		return fmt.Errorf("failed to append message: %w", err)
	}

	// Store in long-term memory (if enabled)
	s.addToLongTermBatch(sessionID, msg)

	// NOTE: No summarization check - that's why AddBatchToHistory should be used instead
	return nil
}

// AddBatchToHistory adds multiple messages atomically at a turn boundary
// This is the CORRECT way to save messages - checks summarization ONCE per turn
//
// Error Handling:
//   - Batch save failures are returned immediately (no partial saves)
//   - Summarization errors are logged but don't fail the operation (messages already saved)
//   - Strategy message save failures are logged and returned
func (s *MemoryService) AddBatchToHistory(sessionID string, messages []*pb.Message) error {
	if sessionID == "" {
		sessionID = "default"
	}

	if len(messages) == 0 {
		return nil
	}

	// 1. Append all messages to session store ATOMICALLY (using transaction)
	// This is MUCH safer than the old loop - either all messages save or none
	if err := s.sessionService.AppendMessages(sessionID, messages); err != nil {
		// CRITICAL: This is a real error, not a warning - return it
		return fmt.Errorf("failed to append messages (transaction rolled back): %w", err)
	}

	// Store in long-term memory (if enabled)
	// This is done AFTER successful save to ensure consistency
	for _, msg := range messages {
		s.addToLongTermBatch(sessionID, msg)
	}

	// 2. NOW check summarization ONCE for the entire batch
	// This prevents the infinite loop bug where each message triggers summarization
	// Use strategy loading (checkpoint-aware, efficient)
	history, err := s.workingMemory.LoadState(sessionID, s.sessionService)
	if err != nil {
		// Summarization is optional - log warning but don't fail
		// Messages are already saved successfully
		log.Printf("⚠️  Failed to load state for summarization check: %v", err)
		log.Printf("⚠️  Skipping summarization for this turn (messages saved successfully)")
		return nil // Don't fail the save operation
	}

	// Let working memory strategy decide if summarization is needed
	// This is called ONCE per turn, not once per message
	// Strategy has already loaded efficiently (e.g., from checkpoint)
	newMessages, err := s.workingMemory.CheckAndSummarize(history)
	if err != nil {
		// Summarization failure is not critical - messages are already saved
		log.Printf("⚠️  Summarization check failed: %v", err)
		log.Printf("⚠️  Continuing without summarization (messages saved successfully)")
		// Don't return error - messages are already saved
	}

	// Save any new messages created by strategy (e.g., summary message)
	// This is CRITICAL for checkpoint detection to work!
	if len(newMessages) > 0 {
		// Use batch append for strategy messages too (atomic)
		if err := s.sessionService.AppendMessages(sessionID, newMessages); err != nil {
			// This IS critical - checkpoint won't work without summary message
			return fmt.Errorf("failed to save strategy messages: %w", err)
		}
		log.Printf("💾 Saved %d strategy message(s)", len(newMessages))
	}

	return nil
}

// GetRecentHistory returns messages from memory
// Delegates to strategy to load and reconstruct conversation state
func (s *MemoryService) GetRecentHistory(sessionID string) ([]*pb.Message, error) {
	if sessionID == "" {
		sessionID = "default"
	}

	// 1. Let strategy load its state from persistent storage
	// Strategy decides HOW to load (e.g., from checkpoint, last N messages, etc.)
	history, err := s.workingMemory.LoadState(sessionID, s.sessionService)
	if err != nil {
		log.Printf("⚠️  Failed to load state from strategy: %v", err)
		// Fallback: Create empty history
		history, _ = hectorcontext.NewConversationHistory(sessionID)
	}

	// 2. Get messages from reconstructed history
	// Strategy has already applied its logic (checkpoint detection, windowing, etc.)
	filteredMessages, err := s.workingMemory.GetMessages(history)
	if err != nil {
		log.Printf("⚠️  Working memory strategy failed: %v", err)
		// Fallback to empty messages if strategy fails
		return []*pb.Message{}, nil
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

// ClearHistory clears memory for a session
func (s *MemoryService) ClearHistory(sessionID string) error {
	if sessionID == "" {
		sessionID = "default"
	}

	// Flush any pending long-term memory batch
	if s.longTermMemory != nil {
		// Check batch size with read lock
		s.batchMu.RLock()
		hasPending := len(s.pendingBatches[sessionID]) > 0
		s.batchMu.RUnlock()

		if hasPending {
			if err := s.flushLongTermBatch(sessionID); err != nil {
				log.Printf("⚠️  Failed to flush pending batch on clear: %v", err)
			}
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

// Shutdown gracefully shuts down the memory service
// It flushes all pending long-term memory batches to ensure no data loss
//
// This method should be called before application shutdown to ensure
// all pending messages are persisted to long-term memory.
//
// Thread-safe: Can be called concurrently with other operations
func (s *MemoryService) Shutdown() error {
	if s.longTermMemory == nil {
		return nil // Nothing to flush
	}

	// Get all session IDs that have pending batches
	s.batchMu.RLock()
	sessionIDs := make([]string, 0, len(s.pendingBatches))
	for sessionID := range s.pendingBatches {
		sessionIDs = append(sessionIDs, sessionID)
	}
	s.batchMu.RUnlock()

	// Flush each session's pending batch
	var firstError error
	for _, sessionID := range sessionIDs {
		if err := s.flushLongTermBatch(sessionID); err != nil {
			log.Printf("⚠️  Failed to flush batch for session %s during shutdown: %v", sessionID, err)
			if firstError == nil {
				firstError = err // Capture first error
			}
		}
	}

	if firstError != nil {
		return fmt.Errorf("failed to flush all batches during shutdown: %w", firstError)
	}

	log.Printf("✅ Memory service shutdown complete (flushed %d sessions)", len(sessionIDs))
	return nil
}

// addToLongTermBatch adds a message to the pending long-term memory batch
// This is a helper to avoid code duplication between AddToHistory and AddBatchToHistory
//
// Thread-safe: Uses batchMu to protect concurrent access to pendingBatches
func (s *MemoryService) addToLongTermBatch(sessionID string, msg *pb.Message) {
	if s.longTermMemory != nil && s.shouldStoreLongTerm(msg) {
		s.batchMu.Lock()

		// Add to pending batch
		if _, exists := s.pendingBatches[sessionID]; !exists {
			s.pendingBatches[sessionID] = make([]*pb.Message, 0, s.longTermConfig.BatchSize)
		}
		s.pendingBatches[sessionID] = append(s.pendingBatches[sessionID], msg)

		// Check if batch size reached (while holding lock)
		shouldFlush := len(s.pendingBatches[sessionID]) >= s.longTermConfig.BatchSize

		s.batchMu.Unlock()

		// Flush outside of lock to avoid holding mutex during I/O
		if shouldFlush {
			if err := s.flushLongTermBatch(sessionID); err != nil {
				log.Printf("⚠️  Long-term storage failed: %v", err)
			}
		}
	}
}

// flushLongTermBatch flushes pending messages to long-term memory
//
// Thread-safe: Uses batchMu to protect concurrent access to pendingBatches
func (s *MemoryService) flushLongTermBatch(sessionID string) error {
	// Get batch with lock
	s.batchMu.Lock()
	batch := s.pendingBatches[sessionID]
	if len(batch) == 0 {
		s.batchMu.Unlock()
		return nil
	}

	// Clear the pending batch immediately (before I/O)
	// This prevents other goroutines from adding to a batch being flushed
	delete(s.pendingBatches, sessionID)
	s.batchMu.Unlock()

	// Store the batch with agent isolation (outside of lock - I/O operation)
	if err := s.longTermMemory.Store(s.agentID, sessionID, batch); err != nil {
		// On error, we've already removed from pendingBatches
		// Log the error but don't try to restore (could cause duplicates)
		log.Printf("⚠️  Failed to store batch for session %s: %v", sessionID, err)
		return err
	}

	return nil
}
