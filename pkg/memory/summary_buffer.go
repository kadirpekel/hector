package memory

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/google/uuid"

	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/utils"
)

// Default summary buffer settings (from legacy pkg/memory/summary_buffer.go)
const (
	DefaultSummaryBudget    = 8000 // Token budget before triggering summarization
	DefaultSummaryThreshold = 0.85 // Percentage of budget that triggers summarization
	DefaultSummaryTarget    = 0.7  // Target percentage of budget after summarization
	// DefaultMinMessagesBeforeSummary   = 20   // (Deprecated) Minimum messages before allowing summarization
	DefaultMinMessagesToKeep          = 10  // Minimum recent messages to keep
	DefaultRecentMessageBudgetPercent = 0.8 // Percentage of target budget for recent messages
)

// Summarizer is the interface for conversation summarization.
// Implementations should use an LLM to summarize conversation history.
type Summarizer interface {
	// SummarizeConversation summarizes the given messages into a concise summary.
	SummarizeConversation(ctx context.Context, events []*agent.Event) (string, error)
}

// SummaryBufferStrategy implements token-based context window management
// with automatic summarization when the budget is exceeded.
//
// When the token count exceeds (budget * threshold), old messages are
// summarized and replaced with a summary message. Recent messages are
// preserved to maintain conversational continuity.
//
// SummaryBufferMemory keeps a rolling LLM-generated summary of conversation history.
type SummaryBufferStrategy struct {
	budget       int
	threshold    float64
	target       float64
	minMessages  int
	tokenCounter *utils.TokenCounter
	summarizer   Summarizer
	model        string
}

// SummaryBufferConfig holds configuration for the summary buffer strategy.
type SummaryBufferConfig struct {
	// Budget is the maximum number of tokens before summarization triggers.
	// Default: 8000
	Budget int

	// Threshold is the percentage of budget that triggers summarization.
	// When current tokens > budget * threshold, summarization occurs.
	// Default: 0.85 (85%)
	Threshold float64

	// Target is the percentage of budget to reduce to after summarization.
	// Recent messages are kept within budget * target.
	// Default: 0.7 (70%)
	Target float64

	// Model is the LLM model name for accurate token counting.
	// Required for accurate counting.
	Model string

	// Summarizer performs conversation summarization.
	// If nil, summarization is disabled (behaves like token_window).
	Summarizer Summarizer

	// MinMessages is a minimum message count for debug logging purposes.
	// Note: Token budget is the primary trigger for summarization, not message count.
	// This field is used in debug logs to help diagnose summarization behavior.
	// Default: 10
	MinMessages int
}

// NewSummaryBufferStrategy creates a new summary buffer strategy.
func NewSummaryBufferStrategy(cfg SummaryBufferConfig) (*SummaryBufferStrategy, error) {
	budget := cfg.Budget
	if budget <= 0 {
		budget = DefaultSummaryBudget
	}

	threshold := cfg.Threshold
	if threshold <= 0 || threshold > 1 {
		threshold = DefaultSummaryThreshold
	}

	target := cfg.Target
	if target <= 0 || target > 1 {
		target = DefaultSummaryTarget
	}

	// Default minMessages if not set
	minMessages := cfg.MinMessages
	if minMessages <= 0 {
		minMessages = DefaultMinMessagesToKeep
	}

	// Calculate capacity based on budget
	// Assume average message size is ~60 tokens (conservative average for chat)
	const avgTokensPerMessage = 60
	capacity := int(float64(budget) * target / float64(avgTokensPerMessage))
	if capacity < 2 {
		capacity = 2 // Keeping less than 2 messages breaks conversation flow
	}

	// Dynamic adjustment:
	// If the requested minMessages exceeds what the budget can realistically hold,
	// scale it down to the calculated capacity. This prevents summarization loops.
	if minMessages > capacity {
		slog.Warn("SummaryBufferStrategy: reducing minMessages to fit budget",
			"requested", minMessages,
			"adjusted", capacity,
			"reason", "budget_constraint")
		minMessages = capacity
	}

	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required for summary_buffer strategy")
	}

	tokenCounter, err := utils.NewTokenCounter(cfg.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create token counter: %w", err)
	}

	return &SummaryBufferStrategy{
		budget:       budget,
		threshold:    threshold,
		target:       target,
		minMessages:  minMessages,
		tokenCounter: tokenCounter,
		summarizer:   cfg.Summarizer,
		model:        cfg.Model,
	}, nil
}

// Name returns the strategy name.
func (s *SummaryBufferStrategy) Name() string {
	return "summary_buffer"
}

// FilterEvents returns events that fit within the target token budget.
// It looks for existing summaries (checkpoint) and loads from there,
// or applies token-based filtering.
func (s *SummaryBufferStrategy) FilterEvents(events []*agent.Event) []*agent.Event {
	if len(events) == 0 {
		return events
	}

	// Look for existing summary (checkpoint) - start from there
	summaryIdx := s.findLastSummaryIndex(events)
	if summaryIdx >= 0 {
		// Start from summary
		events = events[summaryIdx:]
		slog.Debug("SummaryBufferStrategy: loading from checkpoint",
			"checkpoint_idx", summaryIdx,
			"events_after", len(events))
	}

	// Apply token-based filtering within target budget
	targetBudget := int(float64(s.budget) * s.target)
	return s.filterEventsWithinBudget(events, targetBudget)
}

// CheckAndSummarize checks if summarization should occur and performs it.
// Returns the new working memory state [Summary + Recent] to REPLACE the session,
// or nil if no summarization is needed.
func (s *SummaryBufferStrategy) CheckAndSummarize(ctx context.Context, events []*agent.Event, onProgress func(status, message string)) ([]*agent.Event, error) {
	if s.summarizer == nil {
		slog.Debug("SummaryBufferStrategy: summarizer not configured, skipping")
		return nil, nil // Summarization disabled
	}

	// Find last summary and only consider events from it onwards
	// IMPORTANT: We INCLUDE the summary event so the LLM can create a CUMULATIVE summary
	// that incorporates the previous summary's content
	relevantEvents := events
	summaryIdx := s.findLastSummaryIndex(events)
	if summaryIdx >= 0 {
		// Include the summary event itself so its content can be incorporated
		relevantEvents = events[summaryIdx:]
	}

	should := s.shouldSummarize(relevantEvents)
	if !should {
		slog.Debug("SummaryBufferStrategy: conditions not met for summarization",
			"event_count", len(relevantEvents),
			"min_messages", s.minMessages,
			"current_tokens", s.countEventsTokens(relevantEvents),
			"threshold_tokens", int(float64(s.budget)*s.threshold))
		return nil, nil
	}

	// Report progress (UI feedback)
	if onProgress != nil {
		onProgress(agent.ProgressStatusActive, "Compacting session...")
	}

	// Summarize and get the new working memory state
	return s.summarize(ctx, relevantEvents)
}

// shouldSummarize checks if summarization should be triggered.
func (s *SummaryBufferStrategy) shouldSummarize(events []*agent.Event) bool {
	currentTokens := s.countEventsTokens(events)
	thresholdTokens := int(float64(s.budget) * s.threshold)

	// Check minimum messages first
	if len(events) < s.minMessages {
		return false
	}

	// If tokens exceed threshold, we MUST summarize regardless of message count
	// This prevents catastrophic memory loss when budget is small
	if currentTokens > thresholdTokens {
		return true
	}

	// If under threshold, don't summarize (even if we have many messages)
	// This simplifies logic: token budget is the primary trigger
	return false
}

// summarize performs summarization and returns the new working memory state.
// Returns [SummaryEvent] + [RecentEvents] to replace the session.
func (s *SummaryBufferStrategy) summarize(ctx context.Context, events []*agent.Event) ([]*agent.Event, error) {
	targetTokens := int(float64(s.budget) * s.target)

	// Select recent messages to keep (verbatim)
	recentEvents := s.selectRecentEventsWithMinimum(events, targetTokens)
	oldEvents := events[:len(events)-len(recentEvents)]

	if len(oldEvents) == 0 {
		return nil, nil // Nothing to summarize
	}

	slog.Info("Summarizing events",
		"total", len(events),
		"old", len(oldEvents),
		"keeping_recent", len(recentEvents))

	// Summarizer receives old events including previous summary)
	summary, err := s.summarizer.SummarizeConversation(ctx, oldEvents)
	if err != nil {
		return nil, fmt.Errorf("summarization failed: %w", err)
	}

	// Create summary event
	summaryEvent := &agent.Event{
		ID:     uuid.NewString(),
		Author: agent.AuthorSystem, // Use system author
		Artifact: &a2a.Artifact{
			Parts: []a2a.Part{a2a.TextPart{Text: summary}},
		},
		Metadata: map[string]any{
			"type": agent.MetadataTypeSummary,
		},
	}

	slog.Info("Summarization complete",
		"summarized_events", len(oldEvents),
		"summary_length", len(summary),
		"summary_content", summary)

	// Build new working memory state: [Summary] + [Recent Events]
	newState := make([]*agent.Event, 0, 1+len(recentEvents))
	newState = append(newState, summaryEvent)
	newState = append(newState, recentEvents...)

	return newState, nil
}

// filterEventsWithinBudget returns events that fit within the token budget.
func (s *SummaryBufferStrategy) filterEventsWithinBudget(events []*agent.Event, budget int) []*agent.Event {
	if len(events) == 0 {
		return events
	}

	// Work backwards from most recent
	var selected []*agent.Event
	currentTokens := 0
	startIdx := len(events) // Will be adjusted

	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		evTokens := s.countEventTokens(ev)

		if currentTokens+evTokens <= budget {
			selected = append([]*agent.Event{ev}, selected...)
			currentTokens += evTokens
			startIdx = i
		} else {
			break
		}
	}

	// Ensure minimum messages
	minKeep := s.minMessages
	if len(events) < minKeep {
		return events
	}
	if len(selected) < minKeep {
		startIdx = len(events) - minKeep
		selected = events[startIdx:]
	}

	// Turn-Awareness: Ensure we start with a User message
	if len(selected) > 0 && startIdx > 0 {
		adjustedStart := EnsureUserFirst(events, startIdx)
		if adjustedStart < startIdx {
			// Include additional events to start with User
			selected = events[adjustedStart:]
		}
	}

	return selected
}

// selectRecentEventsWithMinimum selects recent events within budget, with minimum guarantee.
func (s *SummaryBufferStrategy) selectRecentEventsWithMinimum(events []*agent.Event, targetTokens int) []*agent.Event {
	if len(events) == 0 {
		return events
	}

	minEvents := s.minMessages
	if len(events) < minEvents {
		return events
	}

	recentBudget := int(float64(targetTokens) * DefaultRecentMessageBudgetPercent)
	recentEvents := s.filterEventsWithinBudget(events, recentBudget)

	if len(recentEvents) < minEvents {
		startIdx := len(events) - minEvents
		if startIdx < 0 {
			startIdx = 0
		}
		return events[startIdx:]
	}

	return recentEvents
}

// findLastSummaryIndex finds the index of the last summary event.
func (s *SummaryBufferStrategy) findLastSummaryIndex(events []*agent.Event) int {
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]

		// Check by metadata
		if ev.Author == agent.AuthorSystem {
			if typeVal, ok := ev.Metadata["type"].(string); ok && typeVal == agent.MetadataTypeSummary {
				return i
			}
		}
	}
	return -1
}

// countEventsTokens counts total tokens for all events.
func (s *SummaryBufferStrategy) countEventsTokens(events []*agent.Event) int {
	total := 0
	for _, ev := range events {
		total += s.countEventTokens(ev)
	}
	return total
}

// countEventTokens counts tokens for a single event.
func (s *SummaryBufferStrategy) countEventTokens(ev *agent.Event) int {
	if ev == nil || ev.Artifact == nil {
		return 0
	}

	text := ev.TextContent()
	messages := []utils.Message{{Role: ev.Author, Content: text}}
	return s.tokenCounter.CountMessages(messages)
}

// Budget returns the configured token budget.
func (s *SummaryBufferStrategy) Budget() int {
	return s.budget
}

// Threshold returns the configured threshold.
func (s *SummaryBufferStrategy) Threshold() float64 {
	return s.threshold
}

// Target returns the configured target.
func (s *SummaryBufferStrategy) Target() float64 {
	return s.target
}

// Ensure SummaryBufferStrategy implements WorkingMemoryStrategy.
var _ WorkingMemoryStrategy = (*SummaryBufferStrategy)(nil)
