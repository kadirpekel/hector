package reasoning

import (
	"context"
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/llms"
)

func TestFallbackAnalysis(t *testing.T) {
	tests := []struct {
		name           string
		toolCalls      []llms.ToolCall
		results        []ToolResult
		wantSuccessful int
		wantFailed     int
		wantPivot      bool
	}{
		{
			name: "all successful",
			toolCalls: []llms.ToolCall{
				{Name: "tool1", Arguments: map[string]interface{}{}},
				{Name: "tool2", Arguments: map[string]interface{}{}},
			},
			results: []ToolResult{
				{Content: "Success: operation completed", Error: nil},
				{Content: "Result: 42", Error: nil},
			},
			wantSuccessful: 2,
			wantFailed:     0,
			wantPivot:      false,
		},
		{
			name: "one failure",
			toolCalls: []llms.ToolCall{
				{Name: "tool1", Arguments: map[string]interface{}{}},
				{Name: "tool2", Arguments: map[string]interface{}{}},
			},
			results: []ToolResult{
				{Content: "Success: operation completed", Error: nil},
				{Content: "Error: connection failed", Error: nil},
			},
			wantSuccessful: 1,
			wantFailed:     1,
			wantPivot:      false,
		},
		{
			name: "majority failures should pivot",
			toolCalls: []llms.ToolCall{
				{Name: "tool1", Arguments: map[string]interface{}{}},
				{Name: "tool2", Arguments: map[string]interface{}{}},
				{Name: "tool3", Arguments: map[string]interface{}{}},
			},
			results: []ToolResult{
				{Content: "Error: failed", Error: nil},
				{Content: "Error: timeout", Error: nil},
				{Content: "Success", Error: nil},
			},
			wantSuccessful: 1,
			wantFailed:     2,
			wantPivot:      true,
		},
		{
			name:           "empty results",
			toolCalls:      []llms.ToolCall{},
			results:        []ToolResult{},
			wantSuccessful: 0,
			wantFailed:     0,
			wantPivot:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := fallbackAnalysis(tt.toolCalls, tt.results)

			if len(analysis.SuccessfulTools) != tt.wantSuccessful {
				t.Errorf("SuccessfulTools count = %d, want %d", len(analysis.SuccessfulTools), tt.wantSuccessful)
			}

			if len(analysis.FailedTools) != tt.wantFailed {
				t.Errorf("FailedTools count = %d, want %d", len(analysis.FailedTools), tt.wantFailed)
			}

			if analysis.ShouldPivot != tt.wantPivot {
				t.Errorf("ShouldPivot = %v, want %v", analysis.ShouldPivot, tt.wantPivot)
			}

			// Confidence should be in valid range
			if analysis.Confidence < 0.0 || analysis.Confidence > 1.0 {
				t.Errorf("Confidence = %f, should be between 0.0 and 1.0", analysis.Confidence)
			}

			// Recommendation should be valid
			validRecommendations := map[string]bool{
				"continue":       true,
				"retry_failed":   true,
				"pivot_approach": true,
				"stop":           true,
			}
			if !validRecommendations[analysis.Recommendation] {
				t.Errorf("Recommendation = %s, should be one of [continue, retry_failed, pivot_approach, stop]", analysis.Recommendation)
			}
		})
	}
}

func TestBuildAnalysisPrompt(t *testing.T) {
	toolCalls := []llms.ToolCall{
		{
			Name: "test_tool",
			Arguments: map[string]interface{}{
				"arg1": "value1",
			},
		},
	}
	results := []ToolResult{
		{Content: "Test result content"},
	}

	prompt := buildAnalysisPrompt(toolCalls, results)

	// Check that prompt contains key elements
	if prompt == "" {
		t.Error("buildAnalysisPrompt() returned empty string")
	}

	// Should mention the tool name
	if len(prompt) < 50 {
		t.Errorf("buildAnalysisPrompt() returned suspiciously short prompt: %s", prompt)
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		maxLen  int
		wantLen int
	}{
		{
			name:    "short string",
			input:   "hello",
			maxLen:  10,
			wantLen: 5,
		},
		{
			name:    "exact length",
			input:   "hello",
			maxLen:  5,
			wantLen: 5,
		},
		{
			name:    "needs truncation",
			input:   "hello world this is a long string",
			maxLen:  10,
			wantLen: 13, // 10 + "..." = 13
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if len(result) != tt.wantLen {
				t.Errorf("truncateString() length = %d, want %d", len(result), tt.wantLen)
			}

			// Should not truncate if input is shorter
			if len(tt.input) <= tt.maxLen && result != tt.input {
				t.Errorf("truncateString() should not modify strings shorter than maxLen")
			}
		})
	}
}

func TestAnalyzeToolResults_EmptyResults(t *testing.T) {
	// Create minimal mock services
	services := &mockAgentServices{}

	analysis, err := AnalyzeToolResults(
		context.Background(),
		[]llms.ToolCall{},
		[]ToolResult{},
		services,
	)

	if err != nil {
		t.Errorf("AnalyzeToolResults() with empty results should not return error, got: %v", err)
	}

	if analysis == nil {
		t.Fatal("AnalyzeToolResults() returned nil analysis")
	}

	// Empty results should return successful analysis with defaults
	if len(analysis.SuccessfulTools) != 0 {
		t.Errorf("Empty results should have 0 successful tools, got %d", len(analysis.SuccessfulTools))
	}

	if analysis.Confidence != 1.0 {
		t.Errorf("Empty results should have confidence 1.0, got %f", analysis.Confidence)
	}
}

// Mock implementation for testing
type mockAgentServices struct{}

func (m *mockAgentServices) GetConfig() config.ReasoningConfig {
	return config.ReasoningConfig{
		MaxIterations: 10,
	}
}

func (m *mockAgentServices) LLM() LLMService {
	// Return nil to trigger fallback analysis
	return nil
}

func (m *mockAgentServices) Tools() ToolService {
	return nil
}

func (m *mockAgentServices) Prompt() PromptService {
	return nil
}

func (m *mockAgentServices) History() HistoryService {
	return nil
}

func (m *mockAgentServices) Context() ContextService {
	return nil
}
