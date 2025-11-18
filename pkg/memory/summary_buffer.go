package memory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/kadirpekel/hector/pkg/utils"
)

// Default summarization settings
const (
	defaultBudget                   = 8000 // Token budget before triggering summarization
	defaultThreshold                = 0.85 // Percentage of budget that triggers summarization
	defaultTarget                   = 0.7  // Target percentage of budget after summarization
	defaultMinMessagesBeforeSummary = 20   // Minimum messages before allowing summarization
	defaultMinMessagesToKeep        = 10   // Minimum recent messages to keep (5 turns)
	defaultRecentMessageBudget      = 0.8  // Percentage of target budget for recent messages
)

type SummarizationService interface {
	SummarizeConversation(ctx context.Context, messages []*pb.Message) (string, error)
}

type SummaryBufferStrategy struct {
	tokenBudget    int
	threshold      float64
	target         float64
	tokenCounter   *utils.TokenCounter
	summarizer     SummarizationService
	statusNotifier StatusNotifier
}

type SummaryBufferConfig struct {
	Budget     int
	Threshold  float64
	Target     float64
	Model      string
	Summarizer SummarizationService
}

func NewSummaryBufferStrategy(cfg SummaryBufferConfig) (*SummaryBufferStrategy, error) {

	if cfg.Budget <= 0 {
		cfg.Budget = defaultBudget
	}
	if cfg.Threshold <= 0 || cfg.Threshold > 1 {
		cfg.Threshold = defaultThreshold
	}
	if cfg.Target <= 0 || cfg.Target > 1 {
		cfg.Target = defaultTarget
	}

	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required for token counting")
	}

	tokenCounter, err := utils.NewTokenCounter(cfg.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create token counter: %w", err)
	}

	if cfg.Summarizer == nil {
		return nil, fmt.Errorf("summarization service is required")
	}

	return &SummaryBufferStrategy{
		tokenBudget:  cfg.Budget,
		threshold:    cfg.Threshold,
		target:       cfg.Target,
		tokenCounter: tokenCounter,
		summarizer:   cfg.Summarizer,
	}, nil
}

func (s *SummaryBufferStrategy) Name() string {
	return "summary_buffer"
}

func (s *SummaryBufferStrategy) SetStatusNotifier(notifier StatusNotifier) {
	s.statusNotifier = notifier
}

func (s *SummaryBufferStrategy) AddMessage(session *hectorcontext.ConversationHistory, msg *pb.Message) error {

	return session.AddMessage(msg)
}

func (s *SummaryBufferStrategy) CheckAndSummarize(session *hectorcontext.ConversationHistory) ([]*pb.Message, error) {
	if s.shouldSummarize(session) {
		summaryMsg, err := s.summarize(session)
		if err != nil {
			return nil, err
		}

		return []*pb.Message{summaryMsg}, nil
	}
	return nil, nil
}

func (s *SummaryBufferStrategy) GetMessages(session *hectorcontext.ConversationHistory) ([]*pb.Message, error) {
	return session.GetAllMessages(), nil
}

func (s *SummaryBufferStrategy) shouldSummarize(session *hectorcontext.ConversationHistory) bool {
	allMessages := session.GetAllMessages()
	if len(allMessages) < defaultMinMessagesBeforeSummary {
		return false
	}

	utilMessages := make([]utils.Message, len(allMessages))
	for i, msg := range allMessages {
		textContent := protocol.ExtractTextFromMessage(msg)
		utilMessages[i] = utils.Message{
			Role:    msg.Role.String(),
			Content: textContent,
		}
	}

	currentTokens := s.tokenCounter.CountMessages(utilMessages)
	thresholdTokens := int(float64(s.tokenBudget) * s.threshold)

	return currentTokens > thresholdTokens
}

func (s *SummaryBufferStrategy) summarize(session *hectorcontext.ConversationHistory) (*pb.Message, error) {

	targetTokens := int(float64(s.tokenBudget) * s.target)

	allMessages := session.GetAllMessages()

	recentMessages := s.selectRecentMessagesWithMinimum(allMessages, targetTokens)
	oldMessages := allMessages[:len(allMessages)-len(recentMessages)]

	if len(oldMessages) == 0 {
		return nil, nil
	}

	slog.Info("Summarizing messages", "total", len(oldMessages), "keeping_recent", len(recentMessages))

	if s.statusNotifier != nil {
			s.statusNotifier("THINKING: Summarizing conversation history...")
	}

	summary, err := s.summarizer.SummarizeConversation(context.Background(), oldMessages)
	if err != nil {
		slog.Warn("Summarization failed", "error", err)
		if s.statusNotifier != nil {
			s.statusNotifier("Warning: Summarization failed, continuing with full history")
		}
		return nil, fmt.Errorf("summarization failed: %w", err)
	}

	slog.Info("Summarized messages", "count", len(oldMessages), "summary_tokens", len(summary))

	session.Clear()

	summaryMsg := &pb.Message{
		Role: pb.Role_ROLE_UNSPECIFIED,
		Parts: []*pb.Part{
			{Part: &pb.Part_Text{Text: fmt.Sprintf("Previous conversation summary: %s", summary)}},
		},
	}
	if err := session.AddMessage(summaryMsg); err != nil {
		return nil, fmt.Errorf("failed to add summary: %w", err)
	}

	for _, msg := range recentMessages {
		if err := session.AddMessage(msg); err != nil {
			slog.Warn("Failed to re-add message", "error", err)
		}
	}

	slog.Info("Summarization complete", "kept_recent", len(recentMessages))

	return summaryMsg, nil
}

func (s *SummaryBufferStrategy) selectRecentMessagesWithMinimum(messages []*pb.Message, targetTokens int) []*pb.Message {
	if len(messages) == 0 {
		return []*pb.Message{}
	}

	minMessages := defaultMinMessagesToKeep
	if len(messages) < minMessages {
		return messages
	}

	recentTokenBudget := int(float64(targetTokens) * defaultRecentMessageBudget)

	recentMessages := s.selectRecentMessages(messages, recentTokenBudget)

	if len(recentMessages) < minMessages {
		startIdx := len(messages) - minMessages
		if startIdx < 0 {
			startIdx = 0
		}
		return messages[startIdx:]
	}

	return recentMessages
}

func (s *SummaryBufferStrategy) selectRecentMessages(messages []*pb.Message, tokenBudget int) []*pb.Message {
	if len(messages) == 0 {
		return []*pb.Message{}
	}

	var selected []*pb.Message
	currentTokens := 0

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

func (s *SummaryBufferStrategy) LoadState(sessionID string, sessionService interface{}) (*hectorcontext.ConversationHistory, error) {

	sessService, ok := sessionService.(interface {
		GetMessagesWithOptions(sessionID string, opts reasoning.LoadOptions) ([]*pb.Message, error)
	})
	if !ok {
		return nil, fmt.Errorf("session service does not support GetMessagesWithOptions")
	}

	allMessages, err := sessService.GetMessagesWithOptions(sessionID, reasoning.LoadOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}

	if len(allMessages) == 0 {

		return hectorcontext.NewConversationHistory(sessionID)
	}

	lastSummaryIdx := s.findLastSummaryIndex(allMessages)

	var messagesToLoad []*pb.Message
	if lastSummaryIdx >= 0 {

		messagesToLoad = allMessages[lastSummaryIdx:]
		reduction := float64(len(allMessages)-len(messagesToLoad)) / float64(len(allMessages)) * 100
		slog.Info("Checkpoint detected, loading messages", "checkpoint_idx", lastSummaryIdx+1, "total", len(allMessages), "loading", len(messagesToLoad), "reduction_pct", reduction)
	} else {

		maxRecent := 100
		if len(allMessages) > maxRecent {
			messagesToLoad = allMessages[len(allMessages)-maxRecent:]
			slog.Warn("No checkpoint found, loading recent messages", "loading", maxRecent, "total", len(allMessages))
		} else {
			messagesToLoad = allMessages
		}
	}

	session, err := hectorcontext.NewConversationHistory(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	for _, msg := range messagesToLoad {
		if err := session.AddMessage(msg); err != nil {
			slog.Warn("Failed to add message to session", "error", err)
		}
	}

	return session, nil
}

func (s *SummaryBufferStrategy) findLastSummaryIndex(messages []*pb.Message) int {

	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]

		if msg.Role == pb.Role_ROLE_UNSPECIFIED {
			text := protocol.ExtractTextFromMessage(msg)
			if len(text) > 0 && (strings.Contains(text, "Previous conversation summary:") ||
				strings.Contains(text, "Conversation summary:")) {
				return i
			}
		}
	}

	return -1
}
