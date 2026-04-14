package memory

import (
	"context"

	"github.com/verikod/hector/pkg/agent"
)

// DefaultBufferWindowSize is the default number of events to keep.
const DefaultBufferWindowSize = 20

// BufferWindowStrategy implements a simple sliding window that keeps
// the last N events. This is the simplest and fastest strategy.
//
// BufferWindowMemory keeps the last N conversation turns as context.
type BufferWindowStrategy struct {
	windowSize int
}

// BufferWindowConfig holds configuration for the buffer window strategy.
type BufferWindowConfig struct {
	// WindowSize is the maximum number of events to keep.
	// Default: 20
	WindowSize int
}

// NewBufferWindowStrategy creates a new buffer window strategy.
func NewBufferWindowStrategy(cfg BufferWindowConfig) *BufferWindowStrategy {
	windowSize := cfg.WindowSize
	if windowSize <= 0 {
		windowSize = DefaultBufferWindowSize
	}

	return &BufferWindowStrategy{
		windowSize: windowSize,
	}
}

// Name returns the strategy name.
func (s *BufferWindowStrategy) Name() string {
	return "buffer_window"
}

// FilterEvents applies a sliding window that GUARANTEES the history starts with a valid
// User message (Author == "user"), satisfying strict turn-based API constraints (e.g. Gemini).
//
// It uses a "Backward Scan / Soft Window" approach:
//  1. Calculates the ideal window cut point (len - N).
//  2. If the cut point lands in the middle of a turn (e.g. Model response),
//     it scans BACKWARD until it finds the initiating User message.
//  3. This means the returned history may be slightly larger than windowSize to preserve integrity.
func (s *BufferWindowStrategy) FilterEvents(events []*agent.Event) []*agent.Event {
	if len(events) <= s.windowSize {
		return events
	}

	start := len(events) - s.windowSize
	if start < 0 {
		start = 0
	}

	// Turn-Awareness: Ensure we start with a User message.
	// Uses shared helper to prevent "orphaned" Model responses at the start of history.
	start = EnsureUserFirst(events, start)

	return events[start:]
}

// CheckAndSummarize always returns nil (buffer window doesn't summarize).
func (s *BufferWindowStrategy) CheckAndSummarize(ctx context.Context, events []*agent.Event, onProgress func(status, message string)) ([]*agent.Event, error) {
	return nil, nil
}

// WindowSize returns the configured window size.
func (s *BufferWindowStrategy) WindowSize() int {
	return s.windowSize
}

// Ensure BufferWindowStrategy implements WorkingMemoryStrategy.
var _ WorkingMemoryStrategy = (*BufferWindowStrategy)(nil)
