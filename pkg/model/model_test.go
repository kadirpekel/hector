package model_test

import (
	"testing"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/verikod/hector/pkg/model"
	"github.com/verikod/hector/pkg/tool"
)

// =============================================================================
// Provider Constants Tests
// =============================================================================

func TestProviderConstants(t *testing.T) {
	tests := []struct {
		provider model.Provider
		expected string
	}{
		{model.ProviderOpenAI, "openai"},
		{model.ProviderAnthropic, "anthropic"},
		{model.ProviderGemini, "gemini"},
		{model.ProviderOllama, "ollama"},
		{model.ProviderUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.provider) != tt.expected {
				t.Errorf("Provider = %q, want %q", tt.provider, tt.expected)
			}
		})
	}
}

// =============================================================================
// GenerateConfig Tests
// =============================================================================

func TestGenerateConfig_Clone_Nil(t *testing.T) {
	var cfg *model.GenerateConfig
	cloned := cfg.Clone()
	if cloned != nil {
		t.Error("Clone of nil should return nil")
	}
}

func TestGenerateConfig_Clone_Empty(t *testing.T) {
	cfg := &model.GenerateConfig{}
	cloned := cfg.Clone()

	if cloned == nil {
		t.Fatal("Clone should not return nil for empty config")
	}
	if cloned == cfg {
		t.Error("Clone should return a new instance")
	}
}

func TestGenerateConfig_Clone_WithValues(t *testing.T) {
	temp := 0.7
	maxTokens := 1024
	topP := 0.9

	cfg := &model.GenerateConfig{
		Temperature:      &temp,
		MaxTokens:        &maxTokens,
		TopP:             &topP,
		StopSequences:    []string{"stop1", "stop2"},
		ResponseMIMEType: "application/json",
		EnableThinking:   true,
		ThinkingBudget:   5000,
	}

	cloned := cfg.Clone()

	// Verify values are copied
	if cloned.Temperature == nil || *cloned.Temperature != 0.7 {
		t.Error("Temperature not cloned correctly")
	}
	if cloned.MaxTokens == nil || *cloned.MaxTokens != 1024 {
		t.Error("MaxTokens not cloned correctly")
	}
	if !cloned.EnableThinking {
		t.Error("EnableThinking not cloned correctly")
	}
	if cloned.ThinkingBudget != 5000 {
		t.Error("ThinkingBudget not cloned correctly")
	}

	// Verify deep copy - modifications shouldn't affect original
	*cloned.Temperature = 0.5
	if *cfg.Temperature != 0.7 {
		t.Error("Clone should be a deep copy - original was modified")
	}
}

func TestGenerateConfig_Clone_StopSequences(t *testing.T) {
	cfg := &model.GenerateConfig{
		StopSequences: []string{"a", "b"},
	}

	cloned := cfg.Clone()

	if len(cloned.StopSequences) != 2 {
		t.Error("StopSequences not cloned correctly")
	}

	// Modify cloned slice - shouldn't affect original
	cloned.StopSequences[0] = "modified"
	if cfg.StopSequences[0] != "a" {
		t.Error("Clone should be a deep copy")
	}
}

func TestGenerateConfig_Clone_ResponseSchema(t *testing.T) {
	cfg := &model.GenerateConfig{
		ResponseSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
			},
		},
	}

	cloned := cfg.Clone()

	if cloned.ResponseSchema == nil {
		t.Error("ResponseSchema should be cloned")
	}
	if cloned.ResponseSchema["type"] != "object" {
		t.Error("ResponseSchema values not cloned")
	}
}

// =============================================================================
// Request Tests
// =============================================================================

func TestRequest_Fields(t *testing.T) {
	req := model.Request{
		Messages: []*a2a.Message{
			a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "Hello"}),
		},
		SystemInstruction: "You are helpful",
		Config: &model.GenerateConfig{
			EnableThinking: true,
		},
	}

	if len(req.Messages) != 1 {
		t.Errorf("Messages length = %d, want 1", len(req.Messages))
	}
	if req.SystemInstruction != "You are helpful" {
		t.Errorf("SystemInstruction = %q", req.SystemInstruction)
	}
	if req.Config == nil || !req.Config.EnableThinking {
		t.Error("Config not set correctly")
	}
}

// =============================================================================
// Response Tests
// =============================================================================

func TestResponse_TextContent_Empty(t *testing.T) {
	resp := &model.Response{}
	if resp.TextContent() != "" {
		t.Error("Empty response should return empty text")
	}
}

func TestResponse_TextContent_NilContent(t *testing.T) {
	resp := &model.Response{Content: nil}
	if resp.TextContent() != "" {
		t.Error("Nil content should return empty text")
	}
}

func TestResponse_TextContent_WithText(t *testing.T) {
	resp := &model.Response{
		Content: &model.Content{
			Parts: []a2a.Part{
				a2a.TextPart{Text: "Hello, world!"},
			},
		},
	}

	if resp.TextContent() != "Hello, world!" {
		t.Errorf("TextContent() = %q, want 'Hello, world!'", resp.TextContent())
	}
}

func TestResponse_HasToolCalls(t *testing.T) {
	t.Run("no tool calls", func(t *testing.T) {
		resp := &model.Response{}
		if resp.HasToolCalls() {
			t.Error("Empty response should not have tool calls")
		}
	})

	t.Run("with tool calls", func(t *testing.T) {
		resp := &model.Response{
			ToolCalls: []tool.ToolCall{
				{ID: "call-1", Name: "test"},
			},
		}
		if !resp.HasToolCalls() {
			t.Error("Response with ToolCalls should have tool calls")
		}
	})
}

func TestResponse_Partial(t *testing.T) {
	partial := &model.Response{Partial: true}
	complete := &model.Response{Partial: false}

	if !partial.Partial {
		t.Error("Partial response should be marked as partial")
	}
	if complete.Partial {
		t.Error("Complete response should not be marked as partial")
	}
}

// =============================================================================
// Content Tests
// =============================================================================

func TestContent_Fields(t *testing.T) {
	content := &model.Content{
		Parts: []a2a.Part{
			a2a.TextPart{Text: "Hello"},
		},
		Role: a2a.MessageRoleAgent,
	}

	if len(content.Parts) != 1 {
		t.Error("Parts not set correctly")
	}
	if content.Role != a2a.MessageRoleAgent {
		t.Error("Role not set correctly")
	}
}

// =============================================================================
// Usage Tests
// =============================================================================

func TestUsage_Fields(t *testing.T) {
	usage := &model.Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		ThinkingTokens:   20,
	}

	if usage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 50 {
		t.Errorf("CompletionTokens = %d", usage.CompletionTokens)
	}
	if usage.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d", usage.TotalTokens)
	}
	if usage.ThinkingTokens != 20 {
		t.Errorf("ThinkingTokens = %d", usage.ThinkingTokens)
	}
}

// =============================================================================
// ThinkingBlock Tests
// =============================================================================

func TestThinkingBlock_Fields(t *testing.T) {
	block := &model.ThinkingBlock{
		ID:        "thinking-1",
		Content:   "Let me think about this...",
		Signature: "sig-abc",
	}

	if block.ID != "thinking-1" {
		t.Errorf("ID = %q", block.ID)
	}
	if block.Content != "Let me think about this..." {
		t.Errorf("Content = %q", block.Content)
	}
	if block.Signature != "sig-abc" {
		t.Errorf("Signature = %q", block.Signature)
	}
}
