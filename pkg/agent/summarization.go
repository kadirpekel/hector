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

type SummarizationService struct {
	llm          llms.LLMProvider
	tokenCounter *utils.TokenCounter
}

type SummarizationConfig struct {
	Model string
}

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

	tokenCounter, err := utils.NewTokenCounter(config.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create token counter: %w", err)
	}

	return &SummarizationService{
		llm:          llm,
		tokenCounter: tokenCounter,
	}, nil
}

func (s *SummarizationService) SummarizeConversation(ctx context.Context, messages []*pb.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	conversationText := s.formatConversation(messages)

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

	text, _, _, err := s.llm.Generate(ctx, []*pb.Message{
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

func (s *SummarizationService) SummarizeConversationChunked(ctx context.Context, messages []*pb.Message, chunkSize int) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	if chunkSize <= 0 {
		chunkSize = 20
	}

	if len(messages) <= chunkSize {
		return s.SummarizeConversation(ctx, messages)
	}

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

	if len(summaries) > 1 {
		combinedText := strings.Join(summaries, "\n\n---\n\n")

		systemPrompt := `You are a conversation summarization assistant. You will receive multiple summaries of different parts of a long conversation. Combine them into one coherent, comprehensive summary.

Preserve ALL key information from all summaries while eliminating redundancy.`

		userPrompt := fmt.Sprintf(`Please combine these conversation summaries into one comprehensive summary:

%s

Provide a unified summary:`, combinedText)

		text, _, _, err := s.llm.Generate(ctx, []*pb.Message{
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

func (s *SummarizationService) SummarizeWithRecentContext(
	ctx context.Context,
	messages []*pb.Message,
	keepRecentCount int,
) (*SummarizedHistory, error) {
	if len(messages) <= keepRecentCount {

		return &SummarizedHistory{
			Summary:        "",
			RecentMessages: messages,
		}, nil
	}

	oldMessages := messages[:len(messages)-keepRecentCount]
	recentMessages := messages[len(messages)-keepRecentCount:]

	summary, err := s.SummarizeConversation(ctx, oldMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to summarize old messages: %w", err)
	}

	return &SummarizedHistory{
		Summary:        summary,
		RecentMessages: recentMessages,
	}, nil
}

func (s *SummarizationService) EstimateTokenSavings(messages []*pb.Message, summary string) int {

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

func (s *SummarizationService) ShouldSummarize(messages []*pb.Message, maxTokens int, threshold float64) bool {
	if len(messages) == 0 {
		return false
	}

	if threshold <= 0 || threshold > 1 {
		threshold = 0.8
	}

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

func (s *SummarizationService) formatConversation(messages []*pb.Message) string {
	var sb strings.Builder

	for _, msg := range messages {

		role := string(msg.Role)
		if len(role) > 0 {
			role = strings.ToUpper(string(role[0])) + role[1:]
		}
		textContent := protocol.ExtractTextFromMessage(msg)
		sb.WriteString(fmt.Sprintf("%s: %s\n\n", role, textContent))
	}

	return sb.String()
}

type SummarizedHistory struct {
	Summary        string
	RecentMessages []*pb.Message
}

func (sh *SummarizedHistory) ToMessages() []*pb.Message {
	if sh.Summary == "" {
		return sh.RecentMessages
	}

	summaryMsg := protocol.CreateTextMessage(pb.Role_ROLE_UNSPECIFIED, fmt.Sprintf("Previous conversation summary:\n\n%s", sh.Summary))

	result := make([]*pb.Message, 0, 1+len(sh.RecentMessages))
	result = append(result, summaryMsg)
	result = append(result, sh.RecentMessages...)

	return result
}
