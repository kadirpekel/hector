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
//
// Key Design: Passes the authoritative Error field from ToolResult to the LLM
// This eliminates guessing - the LLM knows definitively which tools succeeded/failed
// and can focus on analyzing WHY and WHAT happened, not WHETHER it happened
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
	useStructuredOutput := cfg.EnableStructuredReflection != nil && *cfg.EnableStructuredReflection

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

			// Determine actual execution status from the Error field
			executionStatus := "SUCCESS"
			if result.Error != nil {
				executionStatus = fmt.Sprintf("FAILED: %v", result.Error)
			}

			prompt.WriteString(fmt.Sprintf("Tool: %s\n", toolName))
			prompt.WriteString(fmt.Sprintf("Arguments: %v\n", toolCalls[i].Args))
			prompt.WriteString(fmt.Sprintf("Execution Status: %s\n", executionStatus))

			// Get truncation length from tool's metadata (if provided)
			// This allows tools to control their own reflection context size
			maxLen := getReflectionContextSize(&result)
			prompt.WriteString(fmt.Sprintf("Output: %s\n\n", truncateString(result.Content, maxLen)))
		}
	}

	prompt.WriteString(`Provide your analysis based on the EXECUTION STATUS of each tool:

- successful_tools: List tools where "Execution Status: SUCCESS"
- failed_tools: List tools where "Execution Status: FAILED"
- critical_errors: Brief descriptions of failures (from FAILED status messages)
- confidence: Your confidence (0.0-1.0) that the iteration made meaningful progress
- should_pivot: Whether the agent should fundamentally change its approach
- recommendation: One of ["continue", "retry_failed", "pivot_approach", "stop"]

IMPORTANT: 
- Respect the Execution Status field - it's the authoritative source of success/failure
- A search tool with "SUCCESS" status and 0 results is SUCCESSFUL (it answered: "nothing found")
- Only mark tools as failed if their Execution Status says "FAILED"
- Use the Output field to understand WHAT the tool did, but use Execution Status for WHETHER it succeeded`)

	return prompt.String()
}

// fallbackAnalysis provides heuristic-based analysis when structured output isn't available
// Uses the authoritative Error field from ToolResult to determine success/failure
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

		// Use the Error field as the authoritative source of success/failure
		// This is systematic: if Error is nil, the tool succeeded
		if result.Error != nil {
			// Tool failed
			analysis.FailedTools = append(analysis.FailedTools, toolName)
			errorMsg := result.Error.Error()
			if len(errorMsg) > 200 {
				errorMsg = errorMsg[:200] + "..."
			}
			analysis.CriticalErrors = append(analysis.CriticalErrors, errorMsg)
		} else {
			// Tool succeeded (even if output is empty or "0 results")
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

// getReflectionContextSize returns the appropriate truncation length for a tool result
// Tools can specify their preferred size via metadata["reflection_context_size"]
// Falls back to sensible defaults if not specified
func getReflectionContextSize(result *ToolResult) int {
	// Check if tool specified a preferred context size in metadata
	if result.Metadata != nil {
		if size, ok := result.Metadata["reflection_context_size"].(int); ok && size > 0 {
			return size
		}
		// Also support float64 (from JSON unmarshaling)
		if size, ok := result.Metadata["reflection_context_size"].(float64); ok && size > 0 {
			return int(size)
		}
	}

	// Default: 500 chars for most tools
	// Tools with large outputs should set their own metadata
	return 500
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
