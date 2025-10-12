package memory

import (
	"context"
	"fmt"
	"log"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/utils"
)

// SummarizationService interface for summarizing conversations
// This avoids circular dependency with pkg/agent
type SummarizationService interface {
	SummarizeConversation(ctx context.Context, messages []*pb.Message) (string, error)
}

// SummaryBufferStrategy implements token-based memory with threshold-triggered summarization
// Uses NATIVE pb.Message storage (no conversion)
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

	// Debug logging removed for cleaner output
	// log.Printf("✅ Summary buffer strategy initialized (budget: %d, threshold: %.0f%%, target: %.0f%%)",
	// 	config.Budget, config.Threshold*100, config.Target*100)

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
// NO CONVERSION - stores pb.Message directly
func (s *SummaryBufferStrategy) AddMessage(session *hectorcontext.ConversationHistory, msg *pb.Message) error {
	// Add message to session
	if err := session.AddMessage(msg); err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	// Check if we need to summarize
	if s.shouldSummarize(session) {
		return s.summarize(session)
	}

	return nil
}

// GetMessages returns messages from the session within token budget
// NO CONVERSION - returns pb.Message directly
func (s *SummaryBufferStrategy) GetMessages(session *hectorcontext.ConversationHistory) ([]*pb.Message, error) {
	return session.GetAllMessages(), nil
}

// shouldSummarize checks if summarization should be triggered
func (s *SummaryBufferStrategy) shouldSummarize(session *hectorcontext.ConversationHistory) bool {
	allMessages := session.GetAllMessages()
	if len(allMessages) < 10 {
		// Need at least 10 messages for summarization to be worthwhile
		return false
	}

	// Convert to utils.Message for token counting
	utilMessages := make([]utils.Message, len(allMessages))
	for i, msg := range allMessages {
		textContent := protocol.ExtractTextFromMessage(msg)
		utilMessages[i] = utils.Message{
			Role:    msg.Role.String(),
			Content: textContent,
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
	allMessages := session.GetAllMessages()

	// Determine how many messages to keep recent
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
	summary, err := s.summarizer.SummarizeConversation(context.Background(), oldMessages)
	if err != nil {
		log.Printf("⚠️  Summarization failed: %v", err)
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
	summaryMsg := &pb.Message{
		Role: pb.Role_ROLE_UNSPECIFIED, // System-like
		Content: []*pb.Part{
			{Part: &pb.Part_Text{Text: fmt.Sprintf("Previous conversation summary: %s", summary)}},
		},
	}
	if err := session.AddMessage(summaryMsg); err != nil {
		return fmt.Errorf("failed to add summary: %w", err)
	}

	// Re-add recent messages
	for _, msg := range recentMessages {
		if err := session.AddMessage(msg); err != nil {
			log.Printf("⚠️  Failed to re-add message: %v", err)
		}
	}

	log.Printf("🎉 Summarization complete (kept %d recent messages)", len(recentMessages))

	return nil
}

// selectRecentMessagesWithMinimum selects recent messages with smart allocation
func (s *SummaryBufferStrategy) selectRecentMessagesWithMinimum(messages []*pb.Message, targetTokens int) []*pb.Message {
	if len(messages) == 0 {
		return []*pb.Message{}
	}

	// Always keep at least the last 3 messages
	minMessages := 3
	if len(messages) < minMessages {
		return messages
	}

	// Allocate 60% of target budget for recent messages
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
func (s *SummaryBufferStrategy) selectRecentMessages(messages []*pb.Message, tokenBudget int) []*pb.Message {
	if len(messages) == 0 {
		return []*pb.Message{}
	}

	// Convert to utils.Message for token counting
	var selected []*pb.Message
	currentTokens := 0

	// Work backwards from most recent
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		textContent := protocol.ExtractTextFromMessage(msg)
		msgTokens := s.tokenCounter.CountMessages([]utils.Message{
			{Role: msg.Role.String(), Content: textContent},
		})

		if currentTokens+msgTokens <= tokenBudget {
			selected = append([]*pb.Message{msg}, selected...)
			currentTokens += msgTokens
		} else {
			break
		}
	}

	return selected
}
