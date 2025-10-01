package reasoning

import (
	"context"
	"fmt"
)

// ============================================================================
// CHAIN OF THOUGHT EXTENSION - CHAIN-OF-THOUGHT REASONING CAPABILITY
// ============================================================================

// ChainOfThoughtExtension provides the ability for the LLM to call the chain-of-thought reasoning engine
type ChainOfThoughtExtension struct {
	reasoningEngine  ReasoningEngine
	services         AgentServices
	extensionService ExtensionService
}

// NewChainOfThoughtExtension creates a new chain-of-thought extension
func NewChainOfThoughtExtension(reasoningEngine ReasoningEngine, services AgentServices) *ChainOfThoughtExtension {
	return &ChainOfThoughtExtension{
		reasoningEngine:  reasoningEngine,
		services:         services,
		extensionService: services.Extensions(),
	}
}

// CreateExtension creates the chain-of-thought extension definition
func (re *ChainOfThoughtExtension) CreateExtension() ExtensionDefinition {
	return ExtensionDefinition{
		Name:        "chain-of-thought",
		Description: "Call the chain-of-thought reasoning engine to continue thinking about specific aspects of a problem",
		OpenTag:     "REASONING_CALL:",
		CloseTag:    "",
		Processor:   re.processReasoningCall,
		Executor:    re.executeReasoningCall,
		PromptFormat: `## Chain-of-Thought Extension

You can call the chain-of-thought reasoning engine to think deeper about specific aspects of a problem. This enables recursive reasoning and chain-of-thought analysis.

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
func (re *ChainOfThoughtExtension) processReasoningCall(content string) (string, string) {
	// Extract a user-friendly display message using extension service
	if purpose := re.extensionService.ExtractField(content, "purpose"); purpose != "" {
		return fmt.Sprintf("\n🤔 **Deepening Reasoning:** %s", purpose), content
	}

	return "\n🤔 **Continuing Reasoning Chain...**", content
}

// executeReasoningCall processes the reasoning call data and returns it for the reasoning engine to handle
func (re *ChainOfThoughtExtension) executeReasoningCall(ctx context.Context, rawData string) (ExtensionResult, error) {
	// Parse the reasoning call using extension service
	var reasoningCall ReasoningCall
	if err := re.extensionService.ParseJSON(rawData, &reasoningCall); err != nil {
		return ExtensionResult{
			Name:    "chain-of-thought",
			Success: false,
			Error:   fmt.Sprintf("Failed to parse reasoning call: %v", err),
		}, nil
	}

	// Validate the reasoning call using extension service
	if err := re.extensionService.ValidateRequiredFields(map[string]interface{}{
		"query": reasoningCall.Query,
	}, []string{"query"}); err != nil {
		return ExtensionResult{
			Name:    "chain-of-thought",
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Return the parsed reasoning call data for the reasoning engine to process
	// The reasoning engine will decide what to do with this data
	return ExtensionResult{
		Name:    "chain-of-thought",
		Success: true,
		Content: "", // No content - just metadata for the reasoning engine
		Metadata: map[string]interface{}{
			"original_query": reasoningCall.Query,
			"purpose":        reasoningCall.Purpose,
			"context":        reasoningCall.Context,
			"reasoning_call": true,
		},
	}, nil
}
