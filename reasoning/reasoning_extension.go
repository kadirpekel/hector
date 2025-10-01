package reasoning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ============================================================================
// REASONING EXTENSION - SELF-RECURSIVE REASONING CAPABILITY
// ============================================================================

// ReasoningExtension provides the ability for the LLM to call the reasoning engine itself
type ReasoningExtension struct {
	reasoningEngine ReasoningEngine
	services        AgentServices
}

// NewReasoningExtension creates a new reasoning extension
func NewReasoningExtension(reasoningEngine ReasoningEngine, services AgentServices) *ReasoningExtension {
	return &ReasoningExtension{
		reasoningEngine: reasoningEngine,
		services:        services,
	}
}

// CreateExtension creates the reasoning extension definition
func (re *ReasoningExtension) CreateExtension() ExtensionDefinition {
	return ExtensionDefinition{
		Name:        "reasoning",
		Description: "Call the reasoning engine itself to continue thinking about specific aspects of a problem",
		OpenTag:     "REASONING_CALL:",
		CloseTag:    "",
		Processor:   re.processReasoningCall,
		Executor:    re.executeReasoningCall,
		PromptFormat: `## Reasoning Extension

You can call the reasoning engine itself to think deeper about specific aspects of a problem. This enables recursive reasoning and chain-of-thought analysis.

**Format:**
REASONING_CALL:
{
  "query": "The specific question or aspect you want to reason about",
  "purpose": "Why you need to reason about this (optional)",
  "context": "Additional context for this reasoning step (optional)"
}

**When to use:**
- When you need to think deeper about a specific aspect
- To break down complex problems into smaller parts
- To explore alternative approaches or perspectives
- To verify your conclusions or reasoning
- To gather more information before finalizing an answer
- To think about your own thinking process (meta-cognition)

**Examples:**

1. **Breaking down complex problems:**
   REASONING_CALL:
   {
     "query": "What are the potential risks and benefits of implementing this solution?",
     "purpose": "Need to analyze the trade-offs before making a recommendation",
     "context": "This is part of a larger decision-making process"
   }

2. **Exploring alternatives:**
   REASONING_CALL:
   {
     "query": "What alternative approaches could solve this problem?",
     "purpose": "Want to ensure I'm considering all viable options"
   }

3. **Meta-cognitive reasoning:**
   REASONING_CALL:
   {
     "query": "Am I making any assumptions that I should question?",
     "purpose": "Self-reflection to improve reasoning quality"
   }

4. **Verification:**
   REASONING_CALL:
   {
     "query": "What evidence supports or contradicts my current conclusion?",
     "purpose": "Need to verify the soundness of my reasoning"
   }

**Important:**
- Be specific about what you want to reason about
- Don't use this for simple questions you can answer directly
- Each call should focus on a specific aspect or sub-problem
- The reasoning engine will continue the chain of thought
- Use this to create deeper, more thorough analysis

`,
	}
}

// processReasoningCall processes reasoning call content
func (re *ReasoningExtension) processReasoningCall(content string) (string, string) {
	// Extract a user-friendly display message
	if purpose := re.extractPurpose(content); purpose != "" {
		return fmt.Sprintf("\n🤔 **Deepening Reasoning:** %s", purpose), content
	}

	return "\n🤔 **Continuing Reasoning Chain...**", content
}

// executeReasoningCall executes the reasoning call by calling the reasoning engine itself
func (re *ReasoningExtension) executeReasoningCall(ctx context.Context, rawData string) (ExtensionResult, error) {
	// Parse the reasoning call
	reasoningCall, err := re.parseReasoningCall(rawData)
	if err != nil {
		return ExtensionResult{
			Name:    "reasoning",
			Success: false,
			Error:   fmt.Sprintf("Failed to parse reasoning call: %v", err),
		}, nil
	}

	// Validate the reasoning call
	if reasoningCall.Query == "" {
		return ExtensionResult{
			Name:    "reasoning",
			Success: false,
			Error:   "Reasoning call must include a 'query' field",
		}, nil
	}

	// Execute the reasoning engine with the new query
	// This creates the recursive call - the reasoning engine calls itself
	outputCh, err := re.reasoningEngine.Execute(ctx, reasoningCall.Query)
	if err != nil {
		return ExtensionResult{
			Name:    "reasoning",
			Success: false,
			Error:   fmt.Sprintf("Failed to execute reasoning: %v", err),
		}, nil
	}

	// Collect the reasoning output
	var reasoningOutput strings.Builder
	for chunk := range outputCh {
		reasoningOutput.WriteString(chunk)
	}

	// Return the reasoning result
	return ExtensionResult{
		Name:    "reasoning",
		Success: true,
		Content: reasoningOutput.String(),
		Metadata: map[string]interface{}{
			"original_query": reasoningCall.Query,
			"purpose":        reasoningCall.Purpose,
			"context":        reasoningCall.Context,
			"recursive_call": true,
		},
	}, nil
}

// parseReasoningCall parses the reasoning call JSON
func (re *ReasoningExtension) parseReasoningCall(rawData string) (*ReasoningCall, error) {
	// Try to parse the entire raw data as JSON first
	rawData = strings.TrimSpace(rawData)
	if strings.HasPrefix(rawData, "{") {
		var parsed ReasoningCall
		if err := json.Unmarshal([]byte(rawData), &parsed); err == nil && parsed.Query != "" {
			return &parsed, nil
		}
	}

	// Fallback: try line by line
	lines := strings.Split(rawData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}

		var parsed ReasoningCall
		if err := json.Unmarshal([]byte(line), &parsed); err == nil && parsed.Query != "" {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("no valid reasoning call JSON found")
}

// extractPurpose extracts the purpose from reasoning call JSON
func (re *ReasoningExtension) extractPurpose(content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}

		var parsed struct {
			Purpose string `json:"purpose,omitempty"`
		}
		if err := json.Unmarshal([]byte(line), &parsed); err == nil && parsed.Purpose != "" {
			return parsed.Purpose
		}
	}
	return ""
}

// ReasoningCall represents a parsed reasoning call
type ReasoningCall struct {
	Query   string `json:"query"`
	Purpose string `json:"purpose,omitempty"`
	Context string `json:"context,omitempty"`
}
