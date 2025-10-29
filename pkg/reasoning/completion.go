package reasoning

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/llms"
)

type CompletionAssessment struct {
	IsComplete     bool     `json:"is_complete"`
	Confidence     float64  `json:"confidence"`
	MissingActions []string `json:"missing_actions"`
	Quality        string   `json:"quality"`
	Recommendation string   `json:"recommendation"`
	Reasoning      string   `json:"reasoning"`
}

func AssessTaskCompletion(
	ctx context.Context,
	originalQuery string,
	assistantResponse string,
	services AgentServices,
) (*CompletionAssessment, error) {

	prompt := buildCompletionPrompt(originalQuery, assistantResponse)

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"is_complete": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the user's request has been fully addressed",
			},
			"confidence": map[string]interface{}{
				"type":        "number",
				"minimum":     0.0,
				"maximum":     1.0,
				"description": "Confidence in the completion assessment (0.0-1.0)",
			},
			"missing_actions": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "List of actions that still need to be taken (empty if complete)",
			},
			"quality": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"excellent", "good", "needs_improvement"},
				"description": "Quality assessment of the response",
			},
			"recommendation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"stop", "continue", "clarify"},
				"description": "What the agent should do next",
			},
			"reasoning": map[string]interface{}{
				"type":        "string",
				"description": "Brief explanation of the assessment",
			},
		},
		"required": []string{
			"is_complete",
			"confidence",
			"missing_actions",
			"quality",
			"recommendation",
			"reasoning",
		},
	}

	config := &llms.StructuredOutputConfig{
		Format: "json",
		Schema: schema,
	}

	llmService := services.LLM()

	if llmService == nil || !llmService.SupportsStructuredOutput() {

		return &CompletionAssessment{
			IsComplete:     true,
			Confidence:     0.7,
			MissingActions: []string{},
			Quality:        "good",
			Recommendation: "stop",
			Reasoning:      "Provider doesn't support structured output; assuming complete",
		}, nil
	}

	messages := []*pb.Message{
		{Role: pb.Role_ROLE_USER, Parts: []*pb.Part{{Part: &pb.Part_Text{Text: prompt}}}},
	}

	text, _, _, err := llmService.GenerateStructured(messages, nil, config)
	if err != nil {

		return &CompletionAssessment{
			IsComplete:     true,
			Confidence:     0.7,
			MissingActions: []string{},
			Quality:        "good",
			Recommendation: "stop",
			Reasoning:      "Error during assessment; assuming complete",
		}, nil
	}

	var assessment CompletionAssessment
	if err := json.Unmarshal([]byte(text), &assessment); err != nil {

		return &CompletionAssessment{
			IsComplete:     true,
			Confidence:     0.7,
			MissingActions: []string{},
			Quality:        "good",
			Recommendation: "stop",
			Reasoning:      "Failed to parse assessment; assuming complete",
		}, nil
	}

	return &assessment, nil
}

func buildCompletionPrompt(originalQuery string, assistantResponse string) string {
	return fmt.Sprintf(`You are evaluating whether an AI agent has fully completed a user's request.

**Original User Request:**
%s

**Agent's Response:**
%s

**Your Task:**
Assess whether the agent has fully addressed the user's request. Consider:
1. Did the agent perform all requested actions?
2. Are there any obvious missing steps or information?
3. Is the response quality sufficient for the request?
4. Does the user need to provide clarification?

Be strict but fair. Mark as complete only if the request is truly satisfied.
Mark as needs clarification if the original request was ambiguous.

Provide your assessment in JSON format with:
- is_complete: Whether the task is fully complete
- confidence: Your confidence level (0.0-1.0)
- missing_actions: List of actions still needed (empty if complete)
- quality: "excellent", "good", or "needs_improvement"
- recommendation: "stop" (task done), "continue" (more work needed), or "clarify" (need user input)
- reasoning: Brief explanation of your assessment`, originalQuery, assistantResponse)
}
