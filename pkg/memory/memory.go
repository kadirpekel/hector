package memory

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

type MemoryService struct {
	agentID        string
	sessionService reasoning.SessionService
	workingMemory  WorkingMemoryStrategy
	longTermMemory LongTermMemoryStrategy

	longTermConfig LongTermConfig

	batchMu        sync.RWMutex
	pendingBatches map[string][]*pb.Message
}

func NewMemoryService(
	agentID string,
	sessionService reasoning.SessionService,
	working WorkingMemoryStrategy,
	longTerm LongTermMemoryStrategy,
	longTermConfig LongTermConfig,
) *MemoryService {

	longTermConfig.SetDefaults()

	if longTerm == nil {
		longTermConfig.AutoRecall = false
	} else if longTermConfig.AutoRecall {
		longTermConfig.AutoRecall = true
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

func (s *MemoryService) AddToHistory(sessionID string, msg *pb.Message) error {

	if sessionID == "" {
		return nil
	}

	if err := s.sessionService.AppendMessage(sessionID, msg); err != nil {
		slog.Warn("Failed to append message to session", "error", err)
		return fmt.Errorf("failed to append message: %w", err)
	}

	s.addToLongTermBatch(sessionID, msg)

	return nil
}

func (s *MemoryService) AddBatchToHistory(sessionID string, messages []*pb.Message) error {

	if sessionID == "" {
		return nil
	}

	if len(messages) == 0 {
		return nil
	}

	if err := s.sessionService.AppendMessages(sessionID, messages); err != nil {

		return fmt.Errorf("failed to append messages (transaction rolled back): %w", err)
	}

	for _, msg := range messages {
		s.addToLongTermBatch(sessionID, msg)
	}

	history, err := s.workingMemory.LoadState(sessionID, s.sessionService)
	if err != nil {

		slog.Warn("Failed to load state for summarization check", "error", err)
		slog.Warn("Skipping summarization for this turn (messages saved successfully)")
		return nil
	}

	newMessages, err := s.workingMemory.CheckAndSummarize(history)
	if err != nil {

		slog.Warn("Summarization check failed", "error", err)
		slog.Warn("Continuing without summarization (messages saved successfully)")

	}

	if len(newMessages) > 0 {

		if err := s.sessionService.AppendMessages(sessionID, newMessages); err != nil {

			return fmt.Errorf("failed to save strategy messages: %w", err)
		}
		slog.Info("Saved strategy messages", "count", len(newMessages))
	}

	return nil
}

func (s *MemoryService) GetRecentHistory(sessionID string) ([]*pb.Message, error) {

	if sessionID == "" {
		return []*pb.Message{}, nil
	}

	history, err := s.workingMemory.LoadState(sessionID, s.sessionService)
	if err != nil {
		slog.Warn("Failed to load state from strategy", "error", err)

		history, _ = hectorcontext.NewConversationHistory(sessionID)
	}

	filteredMessages, err := s.workingMemory.GetMessages(history)
	if err != nil {
		slog.Warn("Working memory strategy failed", "error", err)

		return []*pb.Message{}, nil
	}

	if s.longTermMemory != nil && s.longTermConfig.AutoRecall && len(filteredMessages) > 0 {
		query := s.getLastUserMessage(filteredMessages)
		if query != "" {
			recalled, err := s.longTermMemory.Recall(s.agentID, sessionID, query, s.longTermConfig.RecallLimit)
			if err != nil {
				slog.Warn("Long-term recall failed", "error", err)
			} else if len(recalled) > 0 {

				filteredMessages = append(recalled, filteredMessages...)
			}
		}
	}

	return filteredMessages, nil
}

func (s *MemoryService) ClearHistory(sessionID string) error {

	if sessionID == "" {
		return nil
	}

	if s.longTermMemory != nil {

		s.batchMu.RLock()
		hasPending := len(s.pendingBatches[sessionID]) > 0
		s.batchMu.RUnlock()

		if hasPending {
			if err := s.flushLongTermBatch(sessionID); err != nil {
				slog.Warn("Failed to flush pending batch on clear", "error", err)
			}
		}
	}

	return s.sessionService.DeleteSession(sessionID)
}

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

func (s *MemoryService) getLastUserMessage(messages []*pb.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == pb.Role_ROLE_USER {
			return protocol.ExtractTextFromMessage(messages[i])
		}
	}
	return ""
}

func (s *MemoryService) SetStatusNotifier(notifier StatusNotifier) {
	s.workingMemory.SetStatusNotifier(notifier)
}

func (s *MemoryService) Shutdown() error {
	if s.longTermMemory == nil {
		return nil
	}

	s.batchMu.RLock()
	sessionIDs := make([]string, 0, len(s.pendingBatches))
	for sessionID := range s.pendingBatches {
		sessionIDs = append(sessionIDs, sessionID)
	}
	s.batchMu.RUnlock()

	var firstError error
	for _, sessionID := range sessionIDs {
		if err := s.flushLongTermBatch(sessionID); err != nil {
			slog.Warn("Failed to flush batch for session during shutdown", "session", sessionID, "error", err)
			if firstError == nil {
				firstError = err
			}
		}
	}

	if firstError != nil {
		return fmt.Errorf("failed to flush all batches during shutdown: %w", firstError)
	}

	slog.Info("Memory service shutdown complete", "flushed_sessions", len(sessionIDs))
	return nil
}

func (s *MemoryService) addToLongTermBatch(sessionID string, msg *pb.Message) {
	if s.longTermMemory != nil && s.shouldStoreLongTerm(msg) {
		s.batchMu.Lock()

		if _, exists := s.pendingBatches[sessionID]; !exists {
			s.pendingBatches[sessionID] = make([]*pb.Message, 0, s.longTermConfig.BatchSize)
		}
		s.pendingBatches[sessionID] = append(s.pendingBatches[sessionID], msg)

		shouldFlush := len(s.pendingBatches[sessionID]) >= s.longTermConfig.BatchSize

		s.batchMu.Unlock()

		if shouldFlush {
			if err := s.flushLongTermBatch(sessionID); err != nil {
				slog.Warn("Long-term storage failed", "error", err)
			}
		}
	}
}

func (s *MemoryService) flushLongTermBatch(sessionID string) error {

	s.batchMu.Lock()
	batch := s.pendingBatches[sessionID]
	if len(batch) == 0 {
		s.batchMu.Unlock()
		return nil
	}

	delete(s.pendingBatches, sessionID)
	s.batchMu.Unlock()

	if err := s.longTermMemory.Store(s.agentID, sessionID, batch); err != nil {

		slog.Warn("Failed to store batch for session", "session", sessionID, "error", err)
		return err
	}

	return nil
}
