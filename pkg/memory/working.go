package memory

import (
	"context"

	"github.com/verikod/hector/pkg/agent"
)

// WorkingMemoryStrategy defines the interface for context window management.
// Different strategies can implement different approaches:
//   - buffer_window: Keep last N messages (simple, fast)
//   - token_window: Keep messages within token budget (accurate)
//   - summary_buffer: Summarize old messages when exceeding budget (compact)
//
// Working memory tracks the current conversation context.
//
// NOTE: Future optimization opportunity - session loading could be checkpoint-aware
// to avoid loading all events for strategies like summary_buffer. The session.GetRequest
// already supports NumRecentEvents for this purpose. See pkg/memory/summary_buffer.go
// LoadState for the legacy approach.
type WorkingMemoryStrategy interface {
	// Name returns the strategy identifier.
	Name() string

	// FilterEvents applies the strategy to filter/truncate events for context window.
	// This is called before building messages for the LLM.
	// Returns the filtered events that should be included in the context.
	FilterEvents(events []*agent.Event) []*agent.Event

	// CheckAndSummarize checks if summarization should occur and performs it if needed.
	// This is called after a turn completes (when events are persisted).
	//
	// The onProgress callback allows the strategy to report progress (e.g. "Summarizing...")
	// before the potentially long-running operation completes.
	//
	// Returns:
	//   - nil, nil: No summarization needed, session unchanged
	//   - []*agent.Event, nil: New working memory state to REPLACE the session events
	//
	// The returned events represent the compacted working memory:
	//   [SummaryEvent] + [RecentEvents]
	//
	// This is optional - strategies like buffer_window return nil.
	CheckAndSummarize(ctx context.Context, events []*agent.Event, onProgress func(status, message string)) ([]*agent.Event, error)
}

// NilWorkingMemory is a no-op strategy that returns all events unchanged.
// Used when no working memory strategy is configured.
type NilWorkingMemory struct{}

// Name returns the strategy name.
func (NilWorkingMemory) Name() string {
	return "none"
}

// FilterEvents returns all events unchanged.
func (NilWorkingMemory) FilterEvents(events []*agent.Event) []*agent.Event {
	return events
}

// CheckAndSummarize always returns nil (no summarization).
func (NilWorkingMemory) CheckAndSummarize(ctx context.Context, events []*agent.Event, onProgress func(status, message string)) ([]*agent.Event, error) {
	return nil, nil
}

// Ensure NilWorkingMemory implements WorkingMemoryStrategy.
var _ WorkingMemoryStrategy = NilWorkingMemory{}

// EnsureUserFirst adjusts startIdx to ensure history starts with a User message.
// This satisfies strict turn-based API constraints (e.g. Gemini) that require
// conversations to begin with a User message.
//
// It scans backward from startIdx until it finds a User message.
// If no User message is found before index 0, returns 0.
func EnsureUserFirst(events []*agent.Event, startIdx int) int {
	for startIdx > 0 && events[startIdx].Author != agent.AuthorUser {
		startIdx--
	}
	return startIdx
}

// WorkingMemoryProvider is implemented by agents that have a working memory strategy.
// This allows the runner to access the strategy for post-turn summarization.
type WorkingMemoryProvider interface {
	// WorkingMemory returns the agent's working memory strategy.
	// Returns nil if no strategy is configured.
	WorkingMemory() WorkingMemoryStrategy
}
