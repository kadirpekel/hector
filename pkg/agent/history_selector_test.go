package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/kadirpekel/hector/pkg/llms"
)

func TestNewHistorySelector(t *testing.T) {
	tests := []struct {
		name      string
		config    *HistoryConfig
		wantError bool
	}{
		{
			name:      "Nil config",
			config:    nil,
			wantError: false,
		},
		{
			name: "Valid config",
			config: &HistoryConfig{
				Model:               "gpt-4o",
				EnableSummarization: false,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, err := NewHistorySelector(tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("NewHistorySelector() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && selector == nil {
				t.Error("NewHistorySelector() returned nil selector")
			}
		})
	}
}

func TestHistorySelector_SelectMessages_StrategyRecent(t *testing.T) {
	selector, err := NewHistorySelector(nil)
	if err != nil {
		t.Fatalf("Failed to create selector: %v", err)
	}

	messages := []llms.Message{
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Message 2"},
		{Role: "assistant", Content: "Response 2"},
		{Role: "user", Content: "Message 3"},
		{Role: "assistant", Content: "Response 3"},
	}

	opts := &SelectionOptions{
		MaxTokens: 50, // Small budget
		Strategy:  StrategyRecent,
	}

	ctx := context.Background()
	selected, err := selector.SelectMessages(ctx, messages, opts)
	if err != nil {
		t.Fatalf("SelectMessages() error = %v", err)
	}

	// Should select from the end (most recent)
	if len(selected) == 0 {
		t.Error("Expected some messages to be selected")
	}

	if len(selected) > 0 {
		lastSelected := selected[len(selected)-1]
		lastOriginal := messages[len(messages)-1]
		if lastSelected.Content != lastOriginal.Content {
			t.Error("Most recent message should be preserved")
		}
	}
}

func TestHistorySelector_SelectMessages_AllFit(t *testing.T) {
	selector, err := NewHistorySelector(nil)
	if err != nil {
		t.Fatalf("Failed to create selector: %v", err)
	}

	messages := []llms.Message{
		{Role: "user", Content: "Hi"},
		{Role: "assistant", Content: "Hello"},
	}

	opts := &SelectionOptions{
		MaxTokens: 10000, // Large budget
		Strategy:  StrategyRecent,
	}

	ctx := context.Background()
	selected, err := selector.SelectMessages(ctx, messages, opts)
	if err != nil {
		t.Fatalf("SelectMessages() error = %v", err)
	}

	// All messages should fit
	if len(selected) != len(messages) {
		t.Errorf("Expected all %d messages, got %d", len(messages), len(selected))
	}
}

func TestHistorySelector_SelectMessages_StrategyImportant(t *testing.T) {
	selector, err := NewHistorySelector(nil)
	if err != nil {
		t.Fatalf("Failed to create selector: %v", err)
	}

	messages := []llms.Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Regular message 1"},
		{Role: "assistant", Content: "Regular response 1"},
		{Role: "user", Content: "We decided to use approach A"},
		{Role: "assistant", Content: "Regular response 2"},
		{Role: "user", Content: "Error: Something failed"},
	}

	opts := &SelectionOptions{
		MaxTokens:      100,
		Strategy:       StrategyImportant,
		PreserveSystem: true,
		PreserveErrors: true,
	}

	ctx := context.Background()
	selected, err := selector.SelectMessages(ctx, messages, opts)
	if err != nil {
		t.Fatalf("SelectMessages() error = %v", err)
	}

	// Should include system message
	hasSystem := false
	hasError := false
	hasDecision := false

	for _, msg := range selected {
		if msg.Role == "system" {
			hasSystem = true
		}
		if strings.Contains(strings.ToLower(msg.Content), "error") {
			hasError = true
		}
		if strings.Contains(strings.ToLower(msg.Content), "decided") {
			hasDecision = true
		}
	}

	if !hasSystem {
		t.Error("Important strategy should preserve system messages")
	}
	if !hasError {
		t.Error("Important strategy should preserve error messages")
	}
	if !hasDecision {
		t.Error("Important strategy should preserve decision messages")
	}
}

func TestHistorySelector_isImportant(t *testing.T) {
	selector, err := NewHistorySelector(nil)
	if err != nil {
		t.Fatalf("Failed to create selector: %v", err)
	}

	tests := []struct {
		name    string
		message llms.Message
		opts    *SelectionOptions
		want    bool
	}{
		{
			name:    "System message",
			message: llms.Message{Role: "system", Content: "You are helpful"},
			opts:    &SelectionOptions{PreserveSystem: true},
			want:    true,
		},
		{
			name:    "Error message",
			message: llms.Message{Role: "user", Content: "Error occurred"},
			opts:    &SelectionOptions{PreserveErrors: true},
			want:    true,
		},
		{
			name:    "Decision message",
			message: llms.Message{Role: "user", Content: "We decided to proceed"},
			opts:    &SelectionOptions{},
			want:    true,
		},
		{
			name:    "Tool call message",
			message: llms.Message{Role: "assistant", ToolCalls: []llms.ToolCall{{ID: "test", Name: "tool"}}},
			opts:    &SelectionOptions{},
			want:    true,
		},
		{
			name:    "Regular message",
			message: llms.Message{Role: "user", Content: "Hello there"},
			opts:    &SelectionOptions{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selector.isImportant(tt.message, tt.opts)
			if got != tt.want {
				t.Errorf("isImportant() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHistorySelector_EstimateReduction(t *testing.T) {
	selector, err := NewHistorySelector(nil)
	if err != nil {
		t.Fatalf("Failed to create selector: %v", err)
	}

	messages := []llms.Message{
		{Role: "user", Content: strings.Repeat("This is a long message. ", 20)},
		{Role: "assistant", Content: strings.Repeat("This is a long response. ", 20)},
	}

	// Estimate reduction needed to fit in small budget
	reduction := selector.EstimateReduction(messages, 50)

	if reduction <= 0 {
		t.Error("Expected positive reduction for long messages with small budget")
	}

	if reduction > 1.0 {
		t.Error("Reduction should not exceed 1.0")
	}

	// No reduction needed for large budget
	reduction2 := selector.EstimateReduction(messages, 10000)
	if reduction2 != 0.0 {
		t.Error("Expected zero reduction for large budget")
	}
}

func TestHistorySelector_GetRecommendedStrategy(t *testing.T) {
	selector, err := NewHistorySelector(nil)
	if err != nil {
		t.Fatalf("Failed to create selector: %v", err)
	}

	tests := []struct {
		name     string
		messages []llms.Message
		opts     *SelectionOptions
		want     SelectMessagesStrategy
	}{
		{
			name: "Many important messages",
			messages: []llms.Message{
				{Role: "system", Content: "System"},
				{Role: "user", Content: "We decided something"},
				{Role: "assistant", Content: "Error occurred"},
			},
			opts: &SelectionOptions{PreserveSystem: true, PreserveErrors: true},
			want: StrategyImportant,
		},
		{
			name: "Few messages",
			messages: []llms.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi"},
			},
			opts: &SelectionOptions{},
			want: StrategyRecent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selector.GetRecommendedStrategy(tt.messages, tt.opts)
			if got != tt.want {
				t.Errorf("GetRecommendedStrategy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHistorySelector_SelectMessages_EmptyMessages(t *testing.T) {
	selector, err := NewHistorySelector(nil)
	if err != nil {
		t.Fatalf("Failed to create selector: %v", err)
	}

	ctx := context.Background()
	selected, err := selector.SelectMessages(ctx, []llms.Message{}, nil)
	if err != nil {
		t.Fatalf("SelectMessages() error = %v", err)
	}

	if len(selected) != 0 {
		t.Error("Empty messages should return empty result")
	}
}

func TestHistorySelector_SelectMessages_DefaultOptions(t *testing.T) {
	selector, err := NewHistorySelector(nil)
	if err != nil {
		t.Fatalf("Failed to create selector: %v", err)
	}

	messages := []llms.Message{
		{Role: "user", Content: "Test"},
	}

	ctx := context.Background()
	selected, err := selector.SelectMessages(ctx, messages, nil)
	if err != nil {
		t.Fatalf("SelectMessages() error = %v", err)
	}

	// Should use default options
	if len(selected) != 1 {
		t.Error("Should handle nil options with defaults")
	}
}

func BenchmarkHistorySelector_SelectRecent(b *testing.B) {
	selector, err := NewHistorySelector(nil)
	if err != nil {
		b.Fatalf("Failed to create selector: %v", err)
	}

	messages := make([]llms.Message, 100)
	for i := 0; i < 100; i++ {
		messages[i] = llms.Message{
			Role:    "user",
			Content: "This is a test message",
		}
	}

	opts := &SelectionOptions{
		MaxTokens: 1000,
		Strategy:  StrategyRecent,
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = selector.SelectMessages(ctx, messages, opts)
	}
}

func BenchmarkHistorySelector_SelectImportant(b *testing.B) {
	selector, err := NewHistorySelector(nil)
	if err != nil {
		b.Fatalf("Failed to create selector: %v", err)
	}

	messages := make([]llms.Message, 100)
	for i := 0; i < 100; i++ {
		role := "user"
		content := "Regular message"
		if i%10 == 0 {
			role = "system"
			content = "Important system message"
		}
		messages[i] = llms.Message{
			Role:    role,
			Content: content,
		}
	}

	opts := &SelectionOptions{
		MaxTokens:      1000,
		Strategy:       StrategyImportant,
		PreserveSystem: true,
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = selector.SelectMessages(ctx, messages, opts)
	}
}
