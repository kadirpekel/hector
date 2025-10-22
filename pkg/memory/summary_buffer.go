package memory

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/reasoning"
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
// DEPRECATED: This is kept for compatibility but should not trigger summarization
// Use CheckAndSummarize at turn boundaries instead
func (s *SummaryBufferStrategy) AddMessage(session *hectorcontext.ConversationHistory, msg *pb.Message) error {
	// Just add the message, don't check summarization
	// Summarization is now handled at turn boundaries via CheckAndSummarize
	return session.AddMessage(msg)
}

// CheckAndSummarize checks if summarization is needed and performs it
// This should be called ONCE per turn, not per message
// Returns the summary message if summarization occurred (needs to be persisted)
func (s *SummaryBufferStrategy) CheckAndSummarize(session *hectorcontext.ConversationHistory) ([]*pb.Message, error) {
	if s.shouldSummarize(session) {
		summaryMsg, err := s.summarize(session)
		if err != nil {
			return nil, err
		}
		// Return summary message so it can be persisted
		return []*pb.Message{summaryMsg}, nil
	}
	return nil, nil
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
// Returns the summary message that needs to be persisted
func (s *SummaryBufferStrategy) summarize(session *hectorcontext.ConversationHistory) (*pb.Message, error) {
	// Calculate target tokens
	targetTokens := int(float64(s.tokenBudget) * s.target)

	// Get all messages
	allMessages := session.GetAllMessages()

	// Determine how many messages to keep recent
	recentMessages := s.selectRecentMessagesWithMinimum(allMessages, targetTokens)
	oldMessages := allMessages[:len(allMessages)-len(recentMessages)]

	if len(oldMessages) == 0 {
		return nil, nil // Nothing to summarize
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
		return nil, fmt.Errorf("summarization failed: %w", err)
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
		return nil, fmt.Errorf("failed to add summary: %w", err)
	}

	// Re-add recent messages
	for _, msg := range recentMessages {
		if err := session.AddMessage(msg); err != nil {
			log.Printf("⚠️  Failed to re-add message: %v", err)
		}
	}

	log.Printf("🎉 Summarization complete (kept %d recent messages)", len(recentMessages))

	// Return summary message so it can be persisted to database
	return summaryMsg, nil
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

// LoadState loads and reconstructs the strategy's state from persistent storage
// For summary_buffer, this detects the last summary message (checkpoint) and loads from there
func (s *SummaryBufferStrategy) LoadState(sessionID string, sessionService interface{}) (*hectorcontext.ConversationHistory, error) {
	// Type assert to get the session service
	sessService, ok := sessionService.(interface {
		GetMessagesWithOptions(sessionID string, opts reasoning.LoadOptions) ([]*pb.Message, error)
	})
	if !ok {
		return nil, fmt.Errorf("session service does not support GetMessagesWithOptions")
	}

	// Step 1: Load ALL messages from database
	allMessages, err := sessService.GetMessagesWithOptions(sessionID, reasoning.LoadOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}

	if len(allMessages) == 0 {
		// No messages, return empty session
		return hectorcontext.NewConversationHistory(sessionID)
	}

	// Step 2: Find last summary message (checkpoint detection)
	lastSummaryIdx := s.findLastSummaryIndex(allMessages)

	// Step 3: Load from checkpoint forward
	var messagesToLoad []*pb.Message
	if lastSummaryIdx >= 0 {
		// Found checkpoint: Load summary + everything after it
		messagesToLoad = allMessages[lastSummaryIdx:]
		log.Printf("📍 Checkpoint detected at message %d/%d, loading %d messages (%.1f%% reduction)",
			lastSummaryIdx+1, len(allMessages), len(messagesToLoad),
			float64(len(allMessages)-len(messagesToLoad))/float64(len(allMessages))*100)
	} else {
		// No checkpoint found: Load recent N messages to prevent overload
		maxRecent := 100 // Safety limit
		if len(allMessages) > maxRecent {
			messagesToLoad = allMessages[len(allMessages)-maxRecent:]
			log.Printf("⚠️  No checkpoint found, loading recent %d of %d messages", maxRecent, len(allMessages))
		} else {
			messagesToLoad = allMessages
			// Debug log - only visible in server logs, not CLI output
			// log.Printf("📥 Loading all %d messages (no checkpoint needed yet)", len(allMessages))
		}
	}

	// Step 4: Reconstruct in-memory session
	session, err := hectorcontext.NewConversationHistory(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	for _, msg := range messagesToLoad {
		if err := session.AddMessage(msg); err != nil {
			log.Printf("⚠️  Failed to add message to session: %v", err)
		}
	}

	// Debug log - only visible in server logs, not CLI output
	// log.Printf("✅ Loaded %d messages for session %s", len(messagesToLoad), sessionID)
	return session, nil
}

// findLastSummaryIndex finds the index of the last summary message
// Summary messages are detected by role and content pattern
func (s *SummaryBufferStrategy) findLastSummaryIndex(messages []*pb.Message) int {
	// Scan from end to beginning (most recent first)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]

		// Detect summary message by:
		// 1. Role is ROLE_UNSPECIFIED (system-like)
		// 2. Content starts with "Previous conversation summary:"
		if msg.Role == pb.Role_ROLE_UNSPECIFIED {
			text := protocol.ExtractTextFromMessage(msg)
			if len(text) > 0 && (strings.Contains(text, "Previous conversation summary:") ||
				strings.Contains(text, "Conversation summary:")) {
				return i
			}
		}
	}

	return -1 // No summary found
}
