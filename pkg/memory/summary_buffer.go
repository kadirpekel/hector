package memory

import (
	"context"
	"fmt"
	"log"
	"time"

	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/utils"
)

// SummarizationService interface for summarizing conversations
// This avoids circular dependency with pkg/agent
type SummarizationService interface {
	SummarizeConversation(ctx context.Context, messages []llms.Message) (string, error)
}

// SummaryBufferStrategy implements token-based memory with threshold-triggered summarization
// This is the DEFAULT and RECOMMENDED strategy for working memory
type SummaryBufferStrategy struct {
	tokenBudget    int     // Default: 2000
	threshold      float64 // Default: 0.8 (trigger at 80%)
	target         float64 // Default: 0.6 (compress to 60%)
	tokenCounter   *utils.TokenCounter
	summarizer     SummarizationService
	statusNotifier StatusNotifier // Optional callback for status updates
}

// SummaryBufferConfig configures the summary buffer strategy
type SummaryBufferConfig struct {
	Budget     int                  // Token budget (default: 2000)
	Threshold  float64              // Trigger at % of budget (default: 0.8)
	Target     float64              // Compress to % of budget (default: 0.6)
	Model      string               // Model for token counting
	LLM        llms.LLMProvider     // LLM for summarization
	Summarizer SummarizationService // Summarization service
}

// NewSummaryBufferStrategy creates a new summary buffer strategy
func NewSummaryBufferStrategy(config SummaryBufferConfig) (*SummaryBufferStrategy, error) {
	// Apply sensible defaults
	if config.Budget <= 0 {
		config.Budget = 2000 // ~50 messages
	}
	if config.Threshold <= 0 || config.Threshold > 1 {
		config.Threshold = 0.8 // Trigger at 80%
	}
	if config.Target <= 0 || config.Target > 1 {
		config.Target = 0.6 // Compress to 60%
	}

	// Initialize token counter
	if config.Model == "" {
		return nil, fmt.Errorf("model is required for token counting")
	}

	tokenCounter, err := utils.NewTokenCounter(config.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create token counter: %w", err)
	}

	// Summarizer is required
	if config.Summarizer == nil {
		return nil, fmt.Errorf("summarization service is required")
	}

	log.Printf("✅ Summary buffer strategy initialized (budget: %d, threshold: %.0f%%, target: %.0f%%)",
		config.Budget, config.Threshold*100, config.Target*100)

	return &SummaryBufferStrategy{
		tokenBudget:  config.Budget,
		threshold:    config.Threshold,
		target:       config.Target,
		tokenCounter: tokenCounter,
		summarizer:   config.Summarizer,
	}, nil
}

// Name returns the strategy identifier
func (s *SummaryBufferStrategy) Name() string {
	return "summary_buffer"
}

// SetStatusNotifier sets a callback for status notifications
func (s *SummaryBufferStrategy) SetStatusNotifier(notifier StatusNotifier) {
	s.statusNotifier = notifier
}

// AddMessage adds a message to the session's memory
// May trigger summarization if threshold is exceeded (blocking operation)
func (s *SummaryBufferStrategy) AddMessage(session *hectorcontext.ConversationHistory, msg llms.Message) error {
	// Add message to session
	_, err := session.AddMessage(msg.Role, msg.Content, nil)
	if err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	// Check if we need to summarize
	if s.shouldSummarize(session) {
		return s.summarize(session)
	}

	return nil
}

// GetMessages returns messages from the session within token budget
func (s *SummaryBufferStrategy) GetMessages(session *hectorcontext.ConversationHistory) ([]llms.Message, error) {
	allMessages := s.getAllMessages(session)
	return allMessages, nil
}

// shouldSummarize checks if summarization should be triggered
func (s *SummaryBufferStrategy) shouldSummarize(session *hectorcontext.ConversationHistory) bool {
	allMessages := s.getAllMessages(session)
	if len(allMessages) < 10 {
		// Need at least 10 messages for summarization to be worthwhile
		return false
	}

	// Convert to utils.Message
	utilMessages := make([]utils.Message, len(allMessages))
	for i, msg := range allMessages {
		utilMessages[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Count current tokens
	currentTokens := s.tokenCounter.CountMessages(utilMessages)
	thresholdTokens := int(float64(s.tokenBudget) * s.threshold)

	return currentTokens > thresholdTokens
}

// summarize performs blocking summarization and updates the session
func (s *SummaryBufferStrategy) summarize(session *hectorcontext.ConversationHistory) error {
	// Calculate target tokens
	targetTokens := int(float64(s.tokenBudget) * s.target)

	// Get all messages
	allMessages := s.getAllMessages(session)

	// Determine how many messages to keep recent
	// Strategy: Keep recent messages that fit within token budget
	// If budget is very small, keep at least the last 2-3 messages
	recentMessages := s.selectRecentMessagesWithMinimum(allMessages, targetTokens)
	oldMessages := allMessages[:len(allMessages)-len(recentMessages)]

	if len(oldMessages) == 0 {
		return nil // Nothing to summarize
	}

	log.Printf("🧠 Summarizing %d messages (keeping %d recent)...",
		len(oldMessages), len(recentMessages))

	// Notify user if callback is set
	if s.statusNotifier != nil {
		s.statusNotifier("💭 Summarizing conversation history...")
	}

	// BLOCKING CALL: Summarize old messages (takes 2-5 seconds)
	// User is already waiting for response, so this is acceptable
	summary, err := s.summarizer.SummarizeConversation(context.Background(), oldMessages)
	if err != nil {
		log.Printf("⚠️  Summarization failed: %v", err)
		// Notify user of failure
		if s.statusNotifier != nil {
			s.statusNotifier("⚠️  Summarization failed, continuing with full history")
		}
		return fmt.Errorf("summarization failed: %w", err)
	}

	log.Printf("✅ Summarized %d messages into %d tokens",
		len(oldMessages), len(summary))

	// Reconstruct session: summary + recent messages
	session.Clear()

	// Add summary as system message
	_, err = session.AddMessage("system", fmt.Sprintf("Previous conversation summary: %s", summary), map[string]interface{}{
		"is_summary":    true,
		"summarized_at": time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("failed to add summary: %w", err)
	}

	// Re-add recent messages
	for _, msg := range recentMessages {
		_, err := session.AddMessage(msg.Role, msg.Content, nil)
		if err != nil {
			log.Printf("⚠️  Failed to re-add message: %v", err)
		}
	}

	log.Printf("🎉 Summarization complete (kept %d recent messages)", len(recentMessages))

	return nil
}

// selectRecentMessagesWithMinimum selects recent messages with smart allocation
// Guarantees keeping at least a minimum number of recent messages for context
func (s *SummaryBufferStrategy) selectRecentMessagesWithMinimum(messages []llms.Message, targetTokens int) []llms.Message {
	if len(messages) == 0 {
		return []llms.Message{}
	}

	// Always keep at least the last 3 messages (or all if fewer)
	// This ensures users don't lose immediate context
	minMessages := 3
	if len(messages) < minMessages {
		return messages // Keep all if we have fewer than minimum
	}

	// Allocate 60% of target budget for recent messages
	// Remaining 40% is for summary overhead
	recentTokenBudget := int(float64(targetTokens) * 0.6)

	// Try to fit messages within budget
	recentMessages := s.selectRecentMessages(messages, recentTokenBudget)

	// If we got fewer than minimum, take the last N messages regardless of tokens
	if len(recentMessages) < minMessages {
		startIdx := len(messages) - minMessages
		if startIdx < 0 {
			startIdx = 0
		}
		return messages[startIdx:]
	}

	return recentMessages
}

// selectRecentMessages selects recent messages that fit within the token budget
func (s *SummaryBufferStrategy) selectRecentMessages(messages []llms.Message, tokenBudget int) []llms.Message {
	if len(messages) == 0 {
		return []llms.Message{}
	}

	// Convert to utils.Message
	utilMessages := make([]utils.Message, len(messages))
	for i, msg := range messages {
		utilMessages[i] = utils.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Use token counter to fit within budget
	fitted := s.tokenCounter.FitWithinLimit(utilMessages, tokenBudget)

	// Convert back to llms.Message
	result := make([]llms.Message, len(fitted))
	for i, msg := range fitted {
		result[i] = llms.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return result
}

// getAllMessages retrieves all messages from a session
func (s *SummaryBufferStrategy) getAllMessages(session *hectorcontext.ConversationHistory) []llms.Message {
	contextMessages := session.GetRecentMessages(100000) // Get all messages

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
