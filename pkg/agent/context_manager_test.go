package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/kadirpekel/hector/pkg/llms"
)

func TestNewContextManager(t *testing.T) {
	tests := []struct {
		name      string
		config    *ContextManagerConfig
		wantError bool
	}{
		{
			name:      "Nil config (uses defaults)",
			config:    nil,
			wantError: false,
		},
		{
			name: "Valid config without LLM",
			config: &ContextManagerConfig{
				Model:                "gpt-4o",
				MaxTokens:            2000,
				SummarizationEnabled: false,
			},
			wantError: false,
		},
		{
			name: "Valid config with LLM",
			config: &ContextManagerConfig{
				Model:                "gpt-4o",
				MaxTokens:            2000,
				SummarizationEnabled: true,
				LLM:                  &MockLLM{},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewContextManager(tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("NewContextManager() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && manager == nil {
				t.Error("NewContextManager() returned nil manager")
			}
		})
	}
}

func TestContextManager_PrepareContext(t *testing.T) {
	config := &ContextManagerConfig{
		Model:             "gpt-4o",
		MaxTokens:         50, // Small budget to force selection
		SelectionStrategy: StrategyRecent,
	}

	manager, err := NewContextManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create messages that will definitely exceed 50 tokens
	messages := []llms.Message{
		{Role: "user", Content: strings.Repeat("This is a longer message to ensure we exceed the token budget. ", 3)},
		{Role: "assistant", Content: strings.Repeat("This is a longer response to ensure we exceed the token budget. ", 3)},
		{Role: "user", Content: strings.Repeat("This is a longer message to ensure we exceed the token budget. ", 3)},
		{Role: "assistant", Content: strings.Repeat("This is a longer response to ensure we exceed the token budget. ", 3)},
		{Role: "user", Content: strings.Repeat("This is a longer message to ensure we exceed the token budget. ", 3)},
		{Role: "assistant", Content: strings.Repeat("This is a longer response to ensure we exceed the token budget. ", 3)},
	}

	ctx := context.Background()
	prepared, err := manager.PrepareContext(ctx, messages)
	if err != nil {
		t.Fatalf("PrepareContext() error = %v", err)
	}

	// Should reduce messages
	if len(prepared) >= len(messages) {
		t.Errorf("PrepareContext() should reduce messages with small budget, got %d/%d messages", len(prepared), len(messages))
	}

	// Should keep most recent
	if len(prepared) > 0 {
		lastPrepared := prepared[len(prepared)-1]
		lastOriginal := messages[len(messages)-1]
		if lastPrepared.Content != lastOriginal.Content {
			t.Error("PrepareContext() should preserve most recent messages")
		}
	}
}

func TestContextManager_GetContextStats(t *testing.T) {
	config := &ContextManagerConfig{
		Model:     "gpt-4o",
		MaxTokens: 1000,
	}

	manager, err := NewContextManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name     string
		messages []llms.Message
	}{
		{
			name:     "Empty messages",
			messages: []llms.Message{},
		},
		{
			name: "Simple messages",
			messages: []llms.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
		},
		{
			name: "With important messages",
			messages: []llms.Message{
				{Role: "system", Content: "You are helpful"},
				{Role: "user", Content: "Hello"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := manager.GetContextStats(tt.messages)
			if stats == nil {
				t.Error("GetContextStats() returned nil")
				return
			}

			if stats.MessageCount != len(tt.messages) {
				t.Errorf("MessageCount = %d, want %d", stats.MessageCount, len(tt.messages))
			}

			if stats.MaxTokens != config.MaxTokens {
				t.Errorf("MaxTokens = %d, want %d", stats.MaxTokens, config.MaxTokens)
			}

			if len(tt.messages) > 0 && stats.TokenCount <= 0 {
				t.Error("TokenCount should be > 0 for non-empty messages")
			}

			if stats.Utilization < 0 || stats.Utilization > 100 {
				t.Errorf("Utilization = %f, should be between 0 and 100", stats.Utilization)
			}
		})
	}
}

func TestContextManager_ShouldCompress(t *testing.T) {
	config := &ContextManagerConfig{
		Model:              "gpt-4o",
		MaxTokens:          100,
		SummarizeThreshold: 0.8,
	}

	manager, err := NewContextManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name     string
		messages []llms.Message
		want     bool
	}{
		{
			name:     "Empty messages",
			messages: []llms.Message{},
			want:     false,
		},
		{
			name: "Short messages",
			messages: []llms.Message{
				{Role: "user", Content: "Hi"},
			},
			want: false,
		},
		{
			name: "Long messages",
			messages: []llms.Message{
				{Role: "user", Content: strings.Repeat("This is a long message. ", 50)},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.ShouldCompress(tt.messages)
			if got != tt.want {
				t.Errorf("ShouldCompress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContextManager_CompressContext(t *testing.T) {
	config := &ContextManagerConfig{
		Model:                "gpt-4o",
		MaxTokens:            50,
		SummarizationEnabled: false, // Test without summarization first
		SelectionStrategy:    StrategyRecent,
	}

	manager, err := NewContextManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	messages := []llms.Message{
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Message 2"},
		{Role: "assistant", Content: "Response 2"},
		{Role: "user", Content: "Message 3"},
		{Role: "assistant", Content: "Response 3"},
	}

	ctx := context.Background()
	compressed, err := manager.CompressContext(ctx, messages)
	if err != nil {
		t.Fatalf("CompressContext() error = %v", err)
	}

	// Should reduce messages
	if len(compressed) >= len(messages) {
		t.Error("CompressContext() should reduce messages")
	}
}

func TestContextManager_OptimizeContext(t *testing.T) {
	config := &ContextManagerConfig{
		Model:             "gpt-4o",
		MaxTokens:         50,
		SelectionStrategy: StrategyRecent,
		PreserveSystem:    true,
	}

	manager, err := NewContextManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	messages := []llms.Message{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Message 2"},
		{Role: "assistant", Content: "Response 2"},
	}

	ctx := context.Background()
	result, err := manager.OptimizeContext(ctx, messages)
	if err != nil {
		t.Fatalf("OptimizeContext() error = %v", err)
	}

	if result == nil {
		t.Fatal("OptimizeContext() returned nil result")
	}

	if result.OriginalStats == nil || result.OptimizedStats == nil {
		t.Error("OptimizeContext() should provide stats")
	}

	if len(result.OptimizedMessages) > len(result.OriginalMessages) {
		t.Error("Optimized messages should not exceed original")
	}

	// Check savings
	savings := result.GetSavings()
	if savings < 0 {
		t.Error("Savings should not be negative")
	}

	reduction := result.GetReductionPercentage()
	if reduction < 0 || reduction > 100 {
		t.Errorf("Reduction percentage = %f, should be between 0 and 100", reduction)
	}
}

func TestContextManager_OptimizeContext_AllFit(t *testing.T) {
	config := &ContextManagerConfig{
		Model:     "gpt-4o",
		MaxTokens: 10000, // Large budget
	}

	manager, err := NewContextManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	messages := []llms.Message{
		{Role: "user", Content: "Hi"},
		{Role: "assistant", Content: "Hello"},
	}

	ctx := context.Background()
	result, err := manager.OptimizeContext(ctx, messages)
	if err != nil {
		t.Fatalf("OptimizeContext() error = %v", err)
	}

	// All messages should fit
	if len(result.OptimizedMessages) != len(result.OriginalMessages) {
		t.Error("All messages should fit with large budget")
	}

	if result.GetSavings() != 0 {
		t.Error("No savings expected when all messages fit")
	}
}

func TestCompressMessages(t *testing.T) {
	// Create messages that will definitely exceed 50 tokens
	messages := []llms.Message{
		{Role: "user", Content: strings.Repeat("This is a longer message to ensure we exceed the token budget. ", 3)},
		{Role: "assistant", Content: strings.Repeat("This is a longer response to ensure we exceed the token budget. ", 3)},
		{Role: "user", Content: strings.Repeat("This is a longer message to ensure we exceed the token budget. ", 3)},
		{Role: "assistant", Content: strings.Repeat("This is a longer response to ensure we exceed the token budget. ", 3)},
	}

	ctx := context.Background()
	compressed, err := CompressMessages(ctx, messages, 50, nil)
	if err != nil {
		t.Fatalf("CompressMessages() error = %v", err)
	}

	// Should reduce messages with small budget
	if len(compressed) >= len(messages) {
		t.Errorf("CompressMessages() should reduce messages, got %d/%d messages", len(compressed), len(messages))
	}
}

func TestOptimizationResult_GetSavings(t *testing.T) {
	result := &OptimizationResult{
		OriginalStats: &ContextStats{
			TokenCount: 100,
		},
		OptimizedStats: &ContextStats{
			TokenCount: 60,
		},
	}

	savings := result.GetSavings()
	if savings != 40 {
		t.Errorf("GetSavings() = %d, want 40", savings)
	}

	reduction := result.GetReductionPercentage()
	if reduction != 40.0 {
		t.Errorf("GetReductionPercentage() = %f, want 40.0", reduction)
	}
}

func TestOptimizationResult_GetSavings_NoIncrease(t *testing.T) {
	result := &OptimizationResult{
		OriginalStats: &ContextStats{
			TokenCount: 50,
		},
		OptimizedStats: &ContextStats{
			TokenCount: 60, // Shouldn't happen, but handle it
		},
	}

	savings := result.GetSavings()
	if savings != 0 {
		t.Errorf("GetSavings() = %d, want 0 (no negative savings)", savings)
	}
}

func BenchmarkContextManager_PrepareContext(b *testing.B) {
	config := &ContextManagerConfig{
		Model:             "gpt-4o",
		MaxTokens:         1000,
		SelectionStrategy: StrategyBalanced,
	}

	manager, err := NewContextManager(config)
	if err != nil {
		b.Fatalf("Failed to create manager: %v", err)
	}

	messages := make([]llms.Message, 50)
	for i := 0; i < 50; i++ {
		messages[i] = llms.Message{
			Role:    "user",
			Content: "This is a test message",
		}
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = manager.PrepareContext(ctx, messages)
	}
}

func BenchmarkContextManager_GetContextStats(b *testing.B) {
	config := &ContextManagerConfig{
		Model:     "gpt-4o",
		MaxTokens: 1000,
	}

	manager, err := NewContextManager(config)
	if err != nil {
		b.Fatalf("Failed to create manager: %v", err)
	}

	messages := make([]llms.Message, 50)
	for i := 0; i < 50; i++ {
		messages[i] = llms.Message{
			Role:    "user",
			Content: "This is a test message",
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = manager.GetContextStats(messages)
	}
}
