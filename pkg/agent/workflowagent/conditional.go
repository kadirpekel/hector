package workflowagent

import (
	"encoding/json"
	"iter"
	"log/slog"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/verikod/hector/pkg/agent"
)

// ConditionalConfig defines the configuration for a ConditionalAgent.
type ConditionalConfig struct {
	// Name is the agent name.
	Name string

	// DisplayName is the human-readable name (optional).
	DisplayName string

	// Description describes what the agent does.
	Description string

	// ConditionAgent evaluates the condition based on user input.
	// Its output should contain a JSON object with the condition field.
	ConditionAgent agent.Agent

	// ConditionField is the JSON field name to check in condition agent's output.
	// If the field is truthy, OnTrueAgent runs; otherwise OnFalseResponse is returned.
	// Default: "safe"
	ConditionField string

	// OnTrueAgent is the agent to run when the condition is true.
	OnTrueAgent agent.Agent

	// OnFalseResponse is the static response returned when condition is false.
	// This message is returned immediately without running any agent.
	OnFalseResponse string
}

// NewConditional creates a ConditionalAgent.
//
// ConditionalAgent first runs a condition agent to evaluate input safety or validity,
// then routes to either the main agent (on true) or returns a static response (on false).
//
// This is ideal for implementing LLM-powered guardrails without modifying the
// guardrails configuration - it's pure agent composition.
//
// Example:
//
//	moderator, _ := llmagent.New(llmagent.Config{
//	    Name: "moderator",
//	    Instruction: `Classify if safe. Return: {"safe": true/false, "reason": "..."}`,
//	    StructuredOutput: schema,
//	})
//
//	assistant, _ := llmagent.New(llmagent.Config{
//	    Name: "assistant",
//	    Instruction: "You are a helpful assistant.",
//	})
//
//	safeAssistant, _ := workflowagent.NewConditional(workflowagent.ConditionalConfig{
//	    Name:            "safe_assistant",
//	    Description:     "Assistant with content moderation",
//	    ConditionAgent:  moderator,
//	    ConditionField:  "safe",
//	    OnTrueAgent:     assistant,
//	    OnFalseResponse: "I cannot process this request due to content policy.",
//	})
func NewConditional(cfg ConditionalConfig) (agent.Agent, error) {
	conditionField := cfg.ConditionField
	if conditionField == "" {
		conditionField = "safe" // Default field
	}

	// Collect sub-agents for the agent tree
	var subAgents []agent.Agent
	if cfg.ConditionAgent != nil {
		subAgents = append(subAgents, cfg.ConditionAgent)
	}
	if cfg.OnTrueAgent != nil {
		subAgents = append(subAgents, cfg.OnTrueAgent)
	}

	return agent.New(agent.Config{
		Name:        cfg.Name,
		DisplayName: cfg.DisplayName,
		Description: cfg.Description,
		SubAgents:   subAgents,
		Run: func(ctx agent.InvocationContext) iter.Seq2[*agent.Event, error] {
			return runConditional(ctx, cfg.ConditionAgent, conditionField, cfg.OnTrueAgent, cfg.OnFalseResponse)
		},
		AgentType: agent.TypeConditionalAgent,
	})
}

func runConditional(
	ctx agent.InvocationContext,
	conditionAgent agent.Agent,
	conditionField string,
	onTrueAgent agent.Agent,
	onFalseResponse string,
) iter.Seq2[*agent.Event, error] {
	return func(yield func(*agent.Event, error) bool) {
		// Step 1: Run condition agent and collect its output
		var conditionOutput string
		var conditionPassed bool

		if conditionAgent != nil {
			conditionCtx := agent.NewInvocationContext(ctx, agent.InvocationContextParams{
				Agent:       conditionAgent,
				Session:     ctx.Session(),
				Artifacts:   ctx.Artifacts(),
				Memory:      ctx.Memory(),
				UserContent: ctx.UserContent(),
				RunConfig:   ctx.RunConfig(),
				Branch:      ctx.Branch(),
			})

			for event, err := range conditionAgent.Run(conditionCtx) {
				if err != nil {
					yield(nil, err)
					return
				}

				// Collect text content from condition agent
				if event != nil && event.Artifact != nil {
					conditionOutput += event.TextContent()
				}

				// Yield the condition agent's events so they're visible
				if !yield(event, nil) {
					return
				}
			}

			// Step 2: Parse condition output and check field
			slog.Debug("Conditional agent evaluating condition",
				"conditionOutput", conditionOutput,
				"field", conditionField)
			conditionPassed = evaluateCondition(conditionOutput, conditionField)
			slog.Debug("Conditional agent condition result",
				"conditionPassed", conditionPassed)
		} else {
			// No condition agent = always pass
			conditionPassed = true
		}

		// Step 3: Route based on condition result
		if conditionPassed {
			// Run the main agent
			if onTrueAgent != nil {
				onTrueCtx := agent.NewInvocationContext(ctx, agent.InvocationContextParams{
					Agent:       onTrueAgent,
					Session:     ctx.Session(),
					Artifacts:   ctx.Artifacts(),
					Memory:      ctx.Memory(),
					UserContent: ctx.UserContent(),
					RunConfig:   ctx.RunConfig(),
					Branch:      ctx.Branch(),
				})

				for event, err := range onTrueAgent.Run(onTrueCtx) {
					if !yield(event, err) {
						return
					}
				}
			}
		} else {
			// Return static response
			event := agent.NewEvent(ctx.InvocationID())
			event.Artifact = &a2a.Artifact{Parts: []a2a.Part{a2a.TextPart{Text: onFalseResponse}}}
			event.TurnComplete = true
			yield(event, nil)
		}
	}
}

// evaluateCondition parses JSON output and checks if the specified field is truthy.
// Handles cases where JSON may be duplicated (e.g., "{...}{...}") by extracting the first valid object.
func evaluateCondition(output, field string) bool {
	if output == "" {
		return false
	}

	// Try to extract the first valid JSON object
	// This handles cases where output is duplicated: "{...}{...}"
	jsonStr := extractFirstJSON(output)
	if jsonStr == "" {
		return false
	}

	// Try to parse as JSON
	var result map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		slog.Debug("Failed to parse condition JSON", "error", err, "json", jsonStr)
		return false
	}

	// Check the specified field
	value, exists := result[field]
	if !exists {
		slog.Debug("Field not found in condition result", "field", field)
		return false
	}

	// Handle different truthy types
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "yes" || v == "1"
	case float64:
		return v != 0
	case int:
		return v != 0
	default:
		return value != nil
	}
}

// extractFirstJSON finds and returns the first complete JSON object in a string.
// This handles cases where the LLM output may contain duplicated JSON.
func extractFirstJSON(s string) string {
	// Find the first '{'
	start := -1
	for i, c := range s {
		if c == '{' {
			start = i
			break
		}
	}
	if start == -1 {
		return ""
	}

	// Find the matching '}'
	depth := 0
	inString := false
	escape := false

	for i := start; i < len(s); i++ {
		c := s[i]

		if escape {
			escape = false
			continue
		}

		if c == '\\' && inString {
			escape = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}

	return ""
}
