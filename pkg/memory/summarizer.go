package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/a2aproject/a2a-go/a2a"

	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/model"
)

// Default summarization prompt
const defaultSummarizationPrompt = `You are a conversation summarizer. Your task is to maintain a concise summary of the conversation history by integrating new dialogue into the existing summary.

Existing Summary:
%s

New Dialogue:
%s

Guidelines:
- Integrate the New Dialogue into the Existing Summary to create a single cohesive narrative.
- Preserve key facts, names, roles, and decisions from BOTH the Existing Summary and New Dialogue.
- Keep the summary concise but comprehensive.
- Write in a neutral, factual tone.
- Do not lose important details from the Existing Summary.`

// LLMSummarizer implements the Summarizer interface using an LLM.
type LLMSummarizer struct {
	llm    model.LLM
	prompt string
}

// LLMSummarizerConfig configures the LLM summarizer.
type LLMSummarizerConfig struct {
	// LLM is the language model to use for summarization.
	LLM model.LLM

	// Prompt is a custom summarization prompt template.
	// Use %s twice: 1st for Existing Summary, 2nd for New Dialogue.
	// If empty, uses the default prompt.
	Prompt string
}

// NewLLMSummarizer creates a new LLM-based summarizer.
func NewLLMSummarizer(cfg LLMSummarizerConfig) (*LLMSummarizer, error) {
	if cfg.LLM == nil {
		return nil, fmt.Errorf("LLM is required for summarization")
	}

	prompt := cfg.Prompt
	if prompt == "" {
		prompt = defaultSummarizationPrompt
	}

	return &LLMSummarizer{
		llm:    cfg.LLM,
		prompt: prompt,
	}, nil
}

// SummarizeConversation summarizes the given events into a concise summary.
func (s *LLMSummarizer) SummarizeConversation(ctx context.Context, events []*agent.Event) (string, error) {
	if len(events) == 0 {
		return "", nil
	}

	var existingSummary strings.Builder
	var newDialogue strings.Builder
	hasSummary := false

	for _, ev := range events {
		// Identify previous summary
		isSummary := false
		if ev.Author == agent.AuthorSystem {
			if typeVal, ok := ev.Metadata["type"].(string); ok && typeVal == agent.MetadataTypeSummary {
				isSummary = true
			}
		}

		text := ev.TextContent()
		if text == "" {
			continue
		}

		if isSummary {
			existingSummary.WriteString(text + "\n")
			hasSummary = true
		} else {
			role := ev.Author
			if role == "" {
				role = "unknown"
			}
			newDialogue.WriteString(fmt.Sprintf("[%s]: %s\n\n", role, text))
		}
	}

	// Calculate final prompt arguments
	summaryText := existingSummary.String()
	if !hasSummary {
		summaryText = "None"
	}
	dialogueText := newDialogue.String()
	if dialogueText == "" {
		// If only summary exists and no new dialogue, just return summary?
		// Or ask LLM to refine it?
		// Usually we have new dialogue. If not, return existing summary logic usually handled by caller.
		// But if we call LLM with empty new dialogue, it might hallucinate.
		if hasSummary {
			return strings.TrimSpace(summaryText), nil
		}
		return "", nil
	}

	// Build the summarization prompt
	fullPrompt := fmt.Sprintf(s.prompt, summaryText, dialogueText)

	// Create request for LLM
	req := &model.Request{
		Messages: []*a2a.Message{
			a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: fullPrompt}),
		},
	}

	// Call LLM (non-streaming)
	var summary string
	for resp, err := range s.llm.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", fmt.Errorf("summarization failed: %w", err)
		}
		if resp.Content != nil {
			for _, part := range resp.Content.Parts {
				if tp, ok := part.(a2a.TextPart); ok {
					summary += tp.Text
				}
			}
		}
	}

	return strings.TrimSpace(summary), nil
}

// Ensure LLMSummarizer implements Summarizer.
var _ Summarizer = (*LLMSummarizer)(nil)
