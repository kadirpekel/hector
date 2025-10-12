package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/utils"
)

// ============================================================================
// CONVERSATION SUMMARIZATION SERVICE
// Provides LLM-based summarization of conversation history
// ============================================================================

// SummarizationService handles conversation summarization
type SummarizationService struct {
	llm          llms.LLMProvider
	tokenCounter *utils.TokenCounter
}

// SummarizationConfig configures the summarization service
type SummarizationConfig struct {
	Model string // LLM model for token counting
}

// NewSummarizationService creates a new summarization service
func NewSummarizationService(llm llms.LLMProvider, config *SummarizationConfig) (*SummarizationService, error) {
	if llm == nil {
		return nil, fmt.Errorf("llm is required")
	}

	if config == nil {
		config = &SummarizationConfig{
			Model: "gpt-4o",
		}
	}

	if config.Model == "" {
		config.Model = "gpt-4o"
	}

	// Create token counter
	tokenCounter, err := utils.NewTokenCounter(config.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create token counter: %w", err)
	}

	return &SummarizationService{
		llm:          llm,
		tokenCounter: tokenCounter,
	}, nil
}

// SummarizeConversation summarizes a conversation while preserving key information
// The summary is designed to be used in place of old messages in the context window
func (s *SummarizationService) SummarizeConversation(ctx context.Context, messages []*pb.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	// Build conversation text
	conversationText := s.formatConversation(messages)

	// Create summarization prompt
	systemPrompt := `You are a conversation summarization assistant. Your task is to create a concise, accurate summary of the conversation below.

REQUIREMENTS:
1. Preserve ALL key facts, decisions, and action items
2. Maintain the logical flow and context
3. Include important user preferences or requirements mentioned
4. Keep technical details that might be referenced later
5. Note any unresolved questions or pending tasks
6. Use clear, direct language
7. Aim for 30-50% of original length while keeping all essential information

Format your summary as a coherent narrative, not bullet points unless the conversation naturally requires it.`

	userPrompt := fmt.Sprintf(`Please summarize this conversation:

%s

Provide a comprehensive summary that preserves all important context:`, conversationText)

	// Generate summary
	text, _, _, err := s.llm.Generate([]*pb.Message{
		protocol.CreateTextMessage(pb.Role_ROLE_UNSPECIFIED, systemPrompt),
		protocol.CreateUserMessage(userPrompt),
	}, []llms.ToolDefinition{})

	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	summary := strings.TrimSpace(text)
	if summary == "" {
		return "", fmt.Errorf("empty summary generated")
	}

	return summary, nil
}

// SummarizeConversationChunked summarizes a long conversation in chunks
// This is useful for very long conversations that exceed the LLM's context window
func (s *SummarizationService) SummarizeConversationChunked(ctx context.Context, messages []*pb.Message, chunkSize int) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	if chunkSize <= 0 {
		chunkSize = 20 // Default chunk size
	}

	// If messages fit in a single chunk, use regular summarization
	if len(messages) <= chunkSize {
		return s.SummarizeConversation(ctx, messages)
	}

	// Split into chunks and summarize each
	summaries := []string{}
	for i := 0; i < len(messages); i += chunkSize {
		end := i + chunkSize
		if end > len(messages) {
			end = len(messages)
		}

		chunk := messages[i:end]
		summary, err := s.SummarizeConversation(ctx, chunk)
		if err != nil {
			return "", fmt.Errorf("failed to summarize chunk %d: %w", i/chunkSize, err)
		}

		summaries = append(summaries, summary)
	}

	// If we have multiple summaries, combine them into one
	if len(summaries) > 1 {
		combinedText := strings.Join(summaries, "\n\n---\n\n")

		// Summarize the summaries
		systemPrompt := `You are a conversation summarization assistant. You will receive multiple summaries of different parts of a long conversation. Combine them into one coherent, comprehensive summary.

Preserve ALL key information from all summaries while eliminating redundancy.`

		userPrompt := fmt.Sprintf(`Please combine these conversation summaries into one comprehensive summary:

%s

Provide a unified summary:`, combinedText)

		text, _, _, err := s.llm.Generate([]*pb.Message{
			protocol.CreateTextMessage(pb.Role_ROLE_UNSPECIFIED, systemPrompt),
			protocol.CreateUserMessage(userPrompt),
		}, []llms.ToolDefinition{})

		if err != nil {
			return "", fmt.Errorf("failed to generate combined summary: %w", err)
		}

		return strings.TrimSpace(text), nil
	}

	return summaries[0], nil
}

// SummarizeWithRecentContext creates a summary of old messages while keeping recent ones intact
// This is the recommended approach for maintaining conversation quality
func (s *SummarizationService) SummarizeWithRecentContext(
	ctx context.Context,
	messages []*pb.Message,
	keepRecentCount int,
) (*SummarizedHistory, error) {
	if len(messages) <= keepRecentCount {
		// All messages are recent, no need to summarize
		return &SummarizedHistory{
			Summary:        "",
			RecentMessages: messages,
		}, nil
	}

	// Split into old and recent
	oldMessages := messages[:len(messages)-keepRecentCount]
	recentMessages := messages[len(messages)-keepRecentCount:]

	// Summarize old messages
	summary, err := s.SummarizeConversation(ctx, oldMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to summarize old messages: %w", err)
	}

	return &SummarizedHistory{
		Summary:        summary,
		RecentMessages: recentMessages,
	}, nil
}

// EstimateTokenSavings estimates how many tokens would be saved by summarizing
func (s *SummarizationService) EstimateTokenSavings(messages []*pb.Message, summary string) int {
	// Convert to utils.Message format
	utilsMessages := make([]utils.Message, len(messages))
	for i, msg := range messages {
		textContent := protocol.ExtractTextFromMessage(msg)
		utilsMessages[i] = utils.Message{
			Role:    string(msg.Role),
			Content: textContent,
		}
	}

	originalTokens := s.tokenCounter.CountMessages(utilsMessages)
	summaryTokens := s.tokenCounter.Count(summary)

	savings := originalTokens - summaryTokens
	if savings < 0 {
		return 0
	}
	return savings
}

// ShouldSummarize determines if messages should be summarized based on token budget
// Returns true if summarization would be beneficial
func (s *SummarizationService) ShouldSummarize(messages []*pb.Message, maxTokens int, threshold float64) bool {
	if len(messages) == 0 {
		return false
	}

	if threshold <= 0 || threshold > 1 {
		threshold = 0.8 // Default to 80%
	}

	// Convert to utils.Message
	utilsMessages := make([]utils.Message, len(messages))
	for i, msg := range messages {
		textContent := protocol.ExtractTextFromMessage(msg)
		utilsMessages[i] = utils.Message{
			Role:    string(msg.Role),
			Content: textContent,
		}
	}

	currentTokens := s.tokenCounter.CountMessages(utilsMessages)
	thresholdTokens := int(float64(maxTokens) * threshold)

	return currentTokens >= thresholdTokens
}

// formatConversation formats messages into a readable conversation text
func (s *SummarizationService) formatConversation(messages []*pb.Message) string {
	var sb strings.Builder

	for _, msg := range messages {
		// Capitalize role for readability
		role := string(msg.Role)
		if len(role) > 0 {
			role = strings.ToUpper(string(role[0])) + role[1:]
		}
		textContent := protocol.ExtractTextFromMessage(msg)
		sb.WriteString(fmt.Sprintf("%s: %s\n\n", role, textContent))
	}

	return sb.String()
}

// ============================================================================
// DATA STRUCTURES
// ============================================================================

// SummarizedHistory represents a conversation with a summary and recent messages
type SummarizedHistory struct {
	Summary        string        // Summary of old messages
	RecentMessages []*pb.Message // Recent messages (kept intact)
}

// ToMessages converts a summarized history back to message list format
// The summary is included as a system message at the beginning
func (sh *SummarizedHistory) ToMessages() []*pb.Message {
	if sh.Summary == "" {
		return sh.RecentMessages
	}

	// Create summary message
	summaryMsg := protocol.CreateTextMessage(pb.Role_ROLE_UNSPECIFIED, fmt.Sprintf("Previous conversation summary:\n\n%s", sh.Summary))

	// Combine summary with recent messages
	result := make([]*pb.Message, 0, 1+len(sh.RecentMessages))
	result = append(result, summaryMsg)
	result = append(result, sh.RecentMessages...)

	return result
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// CreateSummarizationPrompt creates a custom summarization prompt
// This can be used for specialized summarization needs
func CreateSummarizationPrompt(conversationText string, customInstructions string) string {
	basePrompt := fmt.Sprintf(`Please summarize this conversation:

%s`, conversationText)

	if customInstructions != "" {
		basePrompt += fmt.Sprintf("\n\nAdditional instructions:\n%s", customInstructions)
	}

	return basePrompt
}
