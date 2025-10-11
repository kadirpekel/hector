package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a"
	"github.com/kadirpekel/hector/pkg/llms"
)

// ============================================================================
// MOCK LLM FOR TESTING
// ============================================================================

type MockLLM struct {
	GenerateFunc func(messages []a2a.Message, tools []llms.ToolDefinition) (string, []a2a.ToolCall, int, error)
}

func (m *MockLLM) Generate(messages []a2a.Message, tools []llms.ToolDefinition) (string, []a2a.ToolCall, int, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(messages, tools)
	}
	return "This is a test summary of the conversation.", nil, 10, nil
}

func (m *MockLLM) GenerateStreaming(messages []a2a.Message, tools []llms.ToolDefinition) (<-chan llms.StreamChunk, error) {
	return nil, nil
}

func (m *MockLLM) GetModelName() string {
	return "mock-model"
}

func (m *MockLLM) GetMaxTokens() int {
	return 4096
}

func (m *MockLLM) GetTemperature() float64 {
	return 0.7
}

func (m *MockLLM) Close() error {
	return nil
}

// ============================================================================
// TESTS
// ============================================================================

func TestNewSummarizationService(t *testing.T) {
	mockLLM := &MockLLM{}

	tests := []struct {
		name      string
		llm       llms.LLMProvider
		config    *SummarizationConfig
		wantError bool
	}{
		{
			name:      "Valid service with nil config",
			llm:       mockLLM,
			config:    nil,
			wantError: false,
		},
		{
			name: "Valid service with config",
			llm:  mockLLM,
			config: &SummarizationConfig{
				Model: "gpt-4o",
			},
			wantError: false,
		},
		{
			name:      "Nil LLM",
			llm:       nil,
			config:    nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewSummarizationService(tt.llm, tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("NewSummarizationService() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && service == nil {
				t.Error("NewSummarizationService() returned nil service")
			}
		})
	}
}

func TestSummarizationService_SummarizeConversation(t *testing.T) {
	mockLLM := &MockLLM{
		GenerateFunc: func(messages []a2a.Message, tools []llms.ToolDefinition) (string, []a2a.ToolCall, int, error) {
			// Check that summarization prompt is used
			if len(messages) < 2 {
				t.Error("Expected system and user messages")
			}

			return "Summary: User asked about AI, assistant explained it's Artificial Intelligence.", nil, 10, nil
		},
	}

	service, err := NewSummarizationService(mockLLM, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	messages := []a2a.Message{
		a2a.CreateUserMessage("What is AI?"),
		a2a.CreateAssistantMessage("AI stands for Artificial Intelligence."),
		a2a.CreateUserMessage("Tell me more."),
		a2a.CreateAssistantMessage("AI involves creating intelligent machines."),
	}

	ctx := context.Background()
	summary, err := service.SummarizeConversation(ctx, messages)
	if err != nil {
		t.Fatalf("SummarizeConversation() error = %v", err)
	}

	if summary == "" {
		t.Error("SummarizeConversation() returned empty summary")
	}

	// Check that summary contains key information
	if !strings.Contains(strings.ToLower(summary), "ai") {
		t.Error("Summary should mention AI")
	}
}

func TestSummarizationService_SummarizeConversation_EmptyMessages(t *testing.T) {
	mockLLM := &MockLLM{}
	service, err := NewSummarizationService(mockLLM, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	summary, err := service.SummarizeConversation(ctx, []a2a.Message{})
	if err != nil {
		t.Errorf("SummarizeConversation() error = %v, want nil", err)
	}

	if summary != "" {
		t.Error("SummarizeConversation() should return empty string for empty messages")
	}
}

func TestSummarizationService_SummarizeWithRecentContext(t *testing.T) {
	mockLLM := &MockLLM{}
	service, err := NewSummarizationService(mockLLM, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	messages := []a2a.Message{
		a2a.CreateUserMessage("Message 1"),
		a2a.CreateAssistantMessage("Response 1"),
		a2a.CreateUserMessage("Message 2"),
		a2a.CreateAssistantMessage("Response 2"),
		a2a.CreateUserMessage("Message 3"),
		a2a.CreateAssistantMessage("Response 3"),
	}

	ctx := context.Background()
	result, err := service.SummarizeWithRecentContext(ctx, messages, 2)
	if err != nil {
		t.Fatalf("SummarizeWithRecentContext() error = %v", err)
	}

	// Should have summary and 2 recent messages
	if result.Summary == "" {
		t.Error("Expected non-empty summary")
	}

	if len(result.RecentMessages) != 2 {
		t.Errorf("Expected 2 recent messages, got %d", len(result.RecentMessages))
	}

	// Recent messages should be the last 2
	textContent := a2a.ExtractTextFromMessage(result.RecentMessages[0])
	if textContent != "Message 3" {
		t.Error("Recent messages should be the last ones")
	}
}

func TestSummarizationService_SummarizeWithRecentContext_AllRecent(t *testing.T) {
	mockLLM := &MockLLM{}
	service, err := NewSummarizationService(mockLLM, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	messages := []a2a.Message{
		a2a.CreateUserMessage("Message 1"),
		a2a.CreateAssistantMessage("Response 1"),
	}

	ctx := context.Background()
	result, err := service.SummarizeWithRecentContext(ctx, messages, 5)
	if err != nil {
		t.Fatalf("SummarizeWithRecentContext() error = %v", err)
	}

	// All messages are recent, no summary needed
	if result.Summary != "" {
		t.Error("Expected empty summary when all messages are recent")
	}

	if len(result.RecentMessages) != len(messages) {
		t.Error("All messages should be in recent when keep count >= total")
	}
}

func TestSummarizationService_ShouldSummarize(t *testing.T) {
	mockLLM := &MockLLM{}
	service, err := NewSummarizationService(mockLLM, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	tests := []struct {
		name      string
		messages  []a2a.Message
		maxTokens int
		threshold float64
		want      bool
	}{
		{
			name:      "Empty messages",
			messages:  []a2a.Message{},
			maxTokens: 1000,
			threshold: 0.8,
			want:      false,
		},
		{
			name: "Below threshold",
			messages: []a2a.Message{
				a2a.CreateUserMessage("Hi"),
			},
			maxTokens: 1000,
			threshold: 0.8,
			want:      false,
		},
		{
			name: "Above threshold",
			messages: []a2a.Message{
				a2a.CreateUserMessage(strings.Repeat("This is a long message. ", 100)),
			},
			maxTokens: 100,
			threshold: 0.8,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.ShouldSummarize(tt.messages, tt.maxTokens, tt.threshold)
			if got != tt.want {
				t.Errorf("ShouldSummarize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSummarizedHistory_ToMessages(t *testing.T) {
	tests := []struct {
		name    string
		history *SummarizedHistory
		want    int // Expected number of messages
	}{
		{
			name: "With summary",
			history: &SummarizedHistory{
				Summary: "Previous conversation about AI",
				RecentMessages: []a2a.Message{
					a2a.CreateUserMessage("Tell me more"),
				},
			},
			want: 2, // Summary message + 1 recent
		},
		{
			name: "Without summary",
			history: &SummarizedHistory{
				Summary: "",
				RecentMessages: []a2a.Message{
					a2a.CreateUserMessage("Hello"),
				},
			},
			want: 1, // Just recent messages
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.history.ToMessages()
			if len(got) != tt.want {
				t.Errorf("ToMessages() returned %d messages, want %d", len(got), tt.want)
			}

			// If there's a summary, first message should be system
			if tt.history.Summary != "" {
				if got[0].Role != "system" {
					t.Error("First message should be system when summary exists")
				}
				textContent := a2a.ExtractTextFromMessage(got[0])
				if !strings.Contains(textContent, tt.history.Summary) {
					t.Error("System message should contain summary")
				}
			}
		})
	}
}

func TestSummarizationService_EstimateTokenSavings(t *testing.T) {
	mockLLM := &MockLLM{}
	service, err := NewSummarizationService(mockLLM, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Long messages that would benefit from summarization
	messages := []a2a.Message{
		a2a.CreateUserMessage(strings.Repeat("This is a long message. ", 20)),
		a2a.CreateAssistantMessage(strings.Repeat("This is a long response. ", 20)),
	}

	summary := "User asked a question, assistant provided an answer."

	savings := service.EstimateTokenSavings(messages, summary)

	// Savings should be positive for this case
	if savings <= 0 {
		t.Error("Expected positive token savings for long messages with short summary")
	}
}

func TestSummarizationService_formatConversation(t *testing.T) {
	mockLLM := &MockLLM{}
	service, err := NewSummarizationService(mockLLM, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	messages := []a2a.Message{
		a2a.CreateUserMessage("Hello"),
		a2a.CreateAssistantMessage("Hi there!"),
	}

	formatted := service.formatConversation(messages)

	// Check that both messages are included
	if !strings.Contains(formatted, "Hello") {
		t.Error("Formatted conversation should contain user message")
	}
	if !strings.Contains(formatted, "Hi there!") {
		t.Error("Formatted conversation should contain assistant message")
	}

	// Check that roles are capitalized
	if !strings.Contains(formatted, "User:") && !strings.Contains(formatted, "USER:") {
		t.Error("Formatted conversation should have readable role labels")
	}
}

func TestCreateSummarizationPrompt(t *testing.T) {
	conversationText := "User: Hi\nAssistant: Hello"
	customInstructions := "Focus on key decisions"

	prompt := CreateSummarizationPrompt(conversationText, customInstructions)

	if !strings.Contains(prompt, conversationText) {
		t.Error("Prompt should contain conversation text")
	}

	if !strings.Contains(prompt, customInstructions) {
		t.Error("Prompt should contain custom instructions")
	}

	// Test without custom instructions
	prompt2 := CreateSummarizationPrompt(conversationText, "")
	if !strings.Contains(prompt2, conversationText) {
		t.Error("Prompt should contain conversation text even without custom instructions")
	}
}
