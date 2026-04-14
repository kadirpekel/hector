package guardrails

import (
	"github.com/a2aproject/a2a-go/a2a"

	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/agent/llmagent"
	"github.com/verikod/hector/pkg/model"
	"github.com/verikod/hector/pkg/tool"
)

// ToBeforeAgentCallback converts an input guardrail chain to BeforeAgentCallback.
// If the chain blocks, returns a message with the block reason and sets intervention metadata.
func ToBeforeAgentCallback(chain *InputChain) agent.BeforeAgentCallback {
	return func(ctx agent.CallbackContext) (*a2a.Message, error) {
		// Get user input from context
		userContent := ctx.UserContent()
		if userContent == nil {
			return nil, nil
		}

		// Extract text from user content (a2a.Message)
		var inputText string
		for _, part := range userContent.Parts {
			if tp, ok := part.(a2a.TextPart); ok {
				inputText += tp.Text
			}
		}

		if inputText == "" {
			return nil, nil
		}

		// Run the guardrail chain
		result, err := chain.Check(ctx, inputText)
		if err != nil {
			return nil, err
		}

		if result.IsBlocking() {
			// Build intervention metadata for X-ray traceability
			extras := map[string]any{
				"guardrail": result.GuardrailName,
			}
			if result.Severity != "" {
				extras["severity"] = string(result.Severity)
			}
			if result.Details != nil {
				for k, v := range result.Details {
					extras[k] = v
				}
			}
			ctx.SetMetadata(agent.BuildInterventionMetadata(
				agent.MetadataTypeGuardrail,
				agent.InterventionActionBlock,
				agent.InterventionSourceInputGuardrail,
				result.Reason,
				extras,
			))

			// Return a message to short-circuit the agent
			return a2a.NewMessage(a2a.MessageRoleAgent,
				a2a.TextPart{Text: "I cannot process this request. " + result.Reason},
			), nil
		}

		// If input was modified, we'd need to modify the context
		// For now, modifications are logged but the original is used
		// TODO: Support input modification via context

		return nil, nil
	}
}

// ToAfterModelCallback converts an output guardrail chain to AfterModelCallback.
// If the chain modifies output, the modified response is returned with intervention metadata.
func ToAfterModelCallback(chain *OutputChain) llmagent.AfterModelCallback {
	return func(ctx agent.CallbackContext, resp *model.Response, respErr error) (*model.Response, error) {
		// Pass through errors
		if respErr != nil {
			return resp, respErr
		}

		// Skip if no response or no content
		if resp == nil || resp.Content == nil {
			return resp, nil
		}

		// Extract text content
		var outputText string
		for _, part := range resp.Content.Parts {
			if tp, ok := part.(a2a.TextPart); ok {
				outputText += tp.Text
			}
		}

		if outputText == "" {
			return resp, nil
		}

		// Run the guardrail chain
		result, err := chain.Check(ctx, outputText)
		if err != nil {
			return nil, err
		}

		if result.IsBlocking() {
			// Build intervention metadata for X-ray traceability
			extras := map[string]any{
				"guardrail": result.GuardrailName,
			}
			if result.Severity != "" {
				extras["severity"] = string(result.Severity)
			}
			if result.Details != nil {
				for k, v := range result.Details {
					extras[k] = v
				}
			}

			// Return a safe response with intervention metadata
			return &model.Response{
				Content: &model.Content{
					Parts: []a2a.Part{a2a.TextPart{Text: "I cannot provide this response. " + result.Reason}},
					Role:  a2a.MessageRoleAgent,
				},
				Metadata: agent.BuildInterventionMetadata(
					agent.MetadataTypeGuardrail,
					agent.InterventionActionBlock,
					agent.InterventionSourceOutputGuardrail,
					result.Reason,
					extras,
				),
			}, nil
		}

		// Handle modified output
		if result.Action == ActionModify {
			if modified, ok := result.Modified.(string); ok {
				// Build intervention metadata for modification
				extras := map[string]any{
					"guardrail": result.GuardrailName,
				}
				if result.Details != nil {
					for k, v := range result.Details {
						extras[k] = v
					}
				}

				// Create a new response with modified text and metadata
				newResponse := *resp
				newResponse.Content = &model.Content{
					Parts: []a2a.Part{a2a.TextPart{Text: modified}},
					Role:  resp.Content.Role,
				}
				newResponse.Metadata = agent.BuildInterventionMetadata(
					agent.MetadataTypeGuardrail,
					agent.InterventionActionModify,
					agent.InterventionSourceOutputGuardrail,
					result.Reason,
					extras,
				)
				return &newResponse, nil
			}
		}

		return resp, nil
	}
}

// ToBeforeToolCallback converts a tool guardrail chain to BeforeToolCallback.
// If the chain blocks, returns an error result.
func ToBeforeToolCallback(chain *ToolChain) llmagent.BeforeToolCallback {
	return func(ctx tool.Context, t tool.Tool, args map[string]any) (map[string]any, error) {
		// Run the guardrail chain
		result, err := chain.Check(ctx, t.Name(), args)
		if err != nil {
			return nil, err
		}

		if result.IsBlocking() {
			// Build intervention metadata for X-ray traceability
			extras := map[string]any{
				"guardrail": result.GuardrailName,
			}
			if result.Severity != "" {
				extras["severity"] = string(result.Severity)
			}
			if result.Details != nil {
				for k, v := range result.Details {
					extras[k] = v
				}
			}
			ctx.SetMetadata(agent.BuildInterventionMetadata(
				agent.MetadataTypeGuardrail,
				agent.InterventionActionBlock,
				agent.InterventionSourceToolGuardrail,
				result.Reason,
				extras,
			))

			// Return error in the result
			return map[string]any{
				"error": result.Reason,
			}, nil
		}

		// Handle modified args
		if result.Action == ActionModify {
			if modified, ok := result.Modified.(map[string]any); ok {
				return modified, nil
			}
		}

		// Continue with original args
		return nil, nil
	}
}
