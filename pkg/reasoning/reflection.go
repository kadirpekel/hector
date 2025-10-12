package reasoning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/protocol"
)

// ============================================================================
// STRUCTURED REFLECTION
// Uses structured output to analyze tool execution results reliably
// ============================================================================

// ReflectionAnalysis represents structured analysis of iteration results
type ReflectionAnalysis struct {
	SuccessfulTools []string `json:"successful_tools"`
	FailedTools     []string `json:"failed_tools"`
	CriticalErrors  []string `json:"critical_errors"`
	Confidence      float64  `json:"confidence"`
	ShouldPivot     bool     `json:"should_pivot"`
	Recommendation  string   `json:"recommendation"`
}

// AnalyzeToolResults uses structured output to analyze tool execution results
// This improves reliability over simple string matching heuristics
// When EnableStructuredReflection is true, uses LLM-based analysis; otherwise uses heuristics
func AnalyzeToolResults(
	ctx context.Context,
	toolCalls []*protocol.ToolCall,
	results []ToolResult,
	services AgentServices,
) (*ReflectionAnalysis, error) {
	if len(results) == 0 {
		return &ReflectionAnalysis{
			SuccessfulTools: []string{},
			FailedTools:     []string{},
			CriticalErrors:  []string{},
			Confidence:      1.0,
			ShouldPivot:     false,
			Recommendation:  "continue",
		}, nil
	}

	// Check if structured reflection is enabled via config
	cfg := services.GetConfig()
	useStructuredOutput := cfg.EnableStructuredReflection

	// Define structured output schema
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"successful_tools": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"failed_tools": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"critical_errors": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"confidence": map[string]interface{}{
				"type":    "number",
				"minimum": 0.0,
				"maximum": 1.0,
			},
			"should_pivot": map[string]interface{}{"type": "boolean"},
			"recommendation": map[string]interface{}{
				"type": "string",
				"enum": []string{"continue", "retry_failed", "pivot_approach", "stop"},
			},
		},
		"required": []string{
			"successful_tools",
			"failed_tools",
			"confidence",
			"should_pivot",
			"recommendation",
		},
	}

	// Get LLM service
	llmService := services.LLM()

	// Try structured output if enabled and supported
	if useStructuredOutput && llmService.SupportsStructuredOutput() {
		// Build analysis prompt
		prompt := buildAnalysisPrompt(toolCalls, results)

		// Create structured output config
		config := &llms.StructuredOutputConfig{
			Format: "json",
			Schema: schema,
		}

		// Make structured LLM call
		messages := []*pb.Message{
			{Role: pb.Role_ROLE_USER, Content: []*pb.Part{{Part: &pb.Part_Text{Text: prompt}}}},
		}

		text, _, _, err := llmService.GenerateStructured(messages, nil, config)
		if err != nil {
			// Fallback to heuristics on error
			return fallbackAnalysis(toolCalls, results), nil
		}

		// Parse response
		var analysis ReflectionAnalysis
		if err := parseJSON(text, &analysis); err != nil {
			// Fallback to heuristics on parse error
			return fallbackAnalysis(toolCalls, results), nil
		}

		return &analysis, nil
	}

	// Use heuristic analysis (default or when structured output not available)
	// This provides:
	// - Better error detection heuristics (multiple keywords)
	// - Confidence scoring based on failure rate
	// - Intelligent pivot recommendations
	return fallbackAnalysis(toolCalls, results), nil
}

// buildAnalysisPrompt creates the prompt for tool result analysis
func buildAnalysisPrompt(toolCalls []*protocol.ToolCall, results []ToolResult) string {
	var prompt strings.Builder

	prompt.WriteString("Analyze the following tool execution results and provide a structured assessment:\n\n")

	for i, result := range results {
		if i < len(toolCalls) {
			toolName := toolCalls[i].Name
			prompt.WriteString(fmt.Sprintf("Tool: %s\n", toolName))
			prompt.WriteString(fmt.Sprintf("Arguments: %v\n", toolCalls[i].Args))
			prompt.WriteString(fmt.Sprintf("Result: %s\n\n", truncateString(result.Content, 500)))
		}
	}

	prompt.WriteString(`Provide your analysis:
- successful_tools: List of tool names that executed successfully
- failed_tools: List of tool names that encountered errors
- critical_errors: Brief descriptions of critical errors (empty if none)
- confidence: Your confidence (0.0-1.0) that the iteration made meaningful progress
- should_pivot: Whether the agent should fundamentally change its approach
- recommendation: One of ["continue", "retry_failed", "pivot_approach", "stop"]

Be strict: only mark tools as failed if they clearly indicate errors, not just empty or unexpected results.`)

	return prompt.String()
}

// fallbackAnalysis provides heuristic-based analysis when structured output isn't available
func fallbackAnalysis(toolCalls []*protocol.ToolCall, results []ToolResult) *ReflectionAnalysis {
	analysis := &ReflectionAnalysis{
		SuccessfulTools: make([]string, 0),
		FailedTools:     make([]string, 0),
		CriticalErrors:  make([]string, 0),
		Confidence:      0.7, // Default moderate confidence
		ShouldPivot:     false,
		Recommendation:  "continue",
	}

	for i, result := range results {
		if i >= len(toolCalls) {
			continue
		}

		toolName := toolCalls[i].Name
		content := strings.ToLower(result.Content)

		// Improved heuristics for error detection
		isError := strings.Contains(content, "error:") ||
			strings.Contains(content, "failed:") ||
			strings.Contains(content, "exception:") ||
			strings.Contains(content, "fatal:") ||
			(result.Error != nil && result.Error.Error() != "")

		if isError {
			analysis.FailedTools = append(analysis.FailedTools, toolName)
			errorMsg := result.Content
			if len(errorMsg) > 200 {
				errorMsg = errorMsg[:200] + "..."
			}
			analysis.CriticalErrors = append(analysis.CriticalErrors, errorMsg)
		} else {
			analysis.SuccessfulTools = append(analysis.SuccessfulTools, toolName)
		}
	}

	// Adjust confidence based on failure rate
	if len(analysis.FailedTools) > 0 {
		failureRate := float64(len(analysis.FailedTools)) / float64(len(results))
		analysis.Confidence = 1.0 - (failureRate * 0.5) // Partial credit for attempts
		if failureRate > 0.5 {
			analysis.ShouldPivot = true
			analysis.Recommendation = "pivot_approach"
		} else if failureRate > 0 {
			analysis.Recommendation = "retry_failed"
		}
	}

	return analysis
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// parseJSON safely parses JSON with error handling
func parseJSON(text string, v interface{}) error {
	return json.Unmarshal([]byte(text), v)
}
