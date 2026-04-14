// Package moderation provides content moderation through various providers.
//
// This package implements the adapter pattern for LLM-powered content moderation,
// supporting multiple backends:
//
//   - OpenAI Moderation API (free with OpenAI usage)
//   - Lakera Guard API (specialized AI security)
//   - Prompt-based (any LLM with custom prompt)
//   - Agent-based (use any Hector agent as moderator)
//
// Example usage:
//
//	provider := moderation.NewOpenAI(moderation.OpenAIConfig{
//	    APIKey:    os.Getenv("OPENAI_API_KEY"),
//	    Threshold: 0.8,
//	})
//
//	result, err := provider.Moderate(ctx, userInput)
//	if err != nil {
//	    return err
//	}
//	if !result.Safe {
//	    return fmt.Errorf("content blocked: %s", result.Reason)
//	}
package moderation
