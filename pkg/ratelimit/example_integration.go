package ratelimit

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// RateLimitedSessionService wraps a SessionService with rate limiting
type RateLimitedSessionService struct {
	base    reasoning.SessionService
	limiter RateLimiter
	scope   Scope // Whether to limit by session or user
}

// NewRateLimitedSessionService creates a new rate-limited session service
func NewRateLimitedSessionService(base reasoning.SessionService, limiter RateLimiter, scope Scope) *RateLimitedSessionService {
	return &RateLimitedSessionService{
		base:    base,
		limiter: limiter,
		scope:   scope,
	}
}

// AppendMessage appends a message with rate limit checking
func (s *RateLimitedSessionService) AppendMessage(sessionID string, message *pb.Message) error {
	// Estimate token count from message (simple approximation)
	tokenCount := estimateTokenCount(message)

	// Check and record rate limit (1 request, N tokens)
	result, err := s.limiter.CheckAndRecord(context.Background(), s.scope, sessionID, tokenCount, 1)
	if err != nil {
		return fmt.Errorf("rate limit check failed: %w", err)
	}

	if !result.Allowed {
		return &RateLimitError{
			Message: result.Reason,
			Result:  result,
		}
	}

	// Proceed with actual append
	return s.base.AppendMessage(sessionID, message)
}

// AppendMessages appends multiple messages with rate limit checking
func (s *RateLimitedSessionService) AppendMessages(sessionID string, messages []*pb.Message) error {
	// Estimate total token count
	var totalTokens int64
	for _, msg := range messages {
		totalTokens += estimateTokenCount(msg)
	}

	// Check and record rate limit (N requests, M tokens)
	result, err := s.limiter.CheckAndRecord(context.Background(), s.scope, sessionID, totalTokens, int64(len(messages)))
	if err != nil {
		return fmt.Errorf("rate limit check failed: %w", err)
	}

	if !result.Allowed {
		return &RateLimitError{
			Message: result.Reason,
			Result:  result,
		}
	}

	// Proceed with actual append
	return s.base.AppendMessages(sessionID, messages)
}

// GetMessages delegates to base service (no rate limiting on reads)
func (s *RateLimitedSessionService) GetMessages(sessionID string, limit int) ([]*pb.Message, error) {
	return s.base.GetMessages(sessionID, limit)
}

// GetMessagesWithOptions delegates to base service
func (s *RateLimitedSessionService) GetMessagesWithOptions(sessionID string, opts reasoning.LoadOptions) ([]*pb.Message, error) {
	return s.base.GetMessagesWithOptions(sessionID, opts)
}

// GetMessageCount delegates to base service
func (s *RateLimitedSessionService) GetMessageCount(sessionID string) (int, error) {
	return s.base.GetMessageCount(sessionID)
}

// GetOrCreateSessionMetadata delegates to base service
func (s *RateLimitedSessionService) GetOrCreateSessionMetadata(sessionID string) (*reasoning.SessionMetadata, error) {
	return s.base.GetOrCreateSessionMetadata(sessionID)
}

// DeleteSession delegates to base service and also resets rate limits
func (s *RateLimitedSessionService) DeleteSession(sessionID string) error {
	// Reset rate limits for this session
	_ = s.limiter.Reset(context.Background(), s.scope, sessionID)

	// Delete the session
	return s.base.DeleteSession(sessionID)
}

// SessionCount delegates to base service
func (s *RateLimitedSessionService) SessionCount() int {
	return s.base.SessionCount()
}

// GetRateLimitUsage returns current rate limit usage for a session
func (s *RateLimitedSessionService) GetRateLimitUsage(sessionID string) ([]Usage, error) {
	return s.limiter.GetUsage(context.Background(), s.scope, sessionID)
}

// RateLimitError represents a rate limit error
type RateLimitError struct {
	Message string
	Result  *CheckResult
}

func (e *RateLimitError) Error() string {
	return e.Message
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	_, ok := err.(*RateLimitError)
	return ok
}

// GetRateLimitResult extracts the CheckResult from a rate limit error
func GetRateLimitResult(err error) *CheckResult {
	if rle, ok := err.(*RateLimitError); ok {
		return rle.Result
	}
	return nil
}

// estimateTokenCount estimates token count from a message (simple approximation)
// In production, you would use a proper tokenizer
func estimateTokenCount(message *pb.Message) int64 {
	if message == nil {
		return 0
	}

	// Simple approximation: ~4 characters per token
	var totalChars int64

	// Count content from all parts
	for _, part := range message.Parts {
		// Count text parts
		if part.GetText() != "" {
			totalChars += int64(len(part.GetText()))
		}
		// Count file parts (name + mime type)
		if part.GetFile() != nil {
			totalChars += int64(len(part.GetFile().GetName()))
			totalChars += int64(len(part.GetFile().GetMimeType()))
		}
		// Count data parts (use a fixed estimate)
		if part.GetData() != nil {
			// Data parts typically contain JSON, estimate ~100 chars per data part
			totalChars += 100
		}
	}

	// Estimate tokens (4 chars per token is rough average for English)
	tokens := totalChars / 4
	if tokens < 1 && totalChars > 0 {
		tokens = 1
	}

	return tokens
}
