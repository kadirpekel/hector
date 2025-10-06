package agent

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a"
)

// ============================================================================
// SHARED A2A HELPERS
// ============================================================================

// extractInputText extracts text from A2A TaskInput
// Shared helper for A2AAdapter and A2AAgent
func extractInputText(input a2a.TaskInput) string {
	switch content := input.Content.(type) {
	case string:
		return content
	case map[string]interface{}:
		// Try to extract a "text" or "prompt" field
		if text, ok := content["text"].(string); ok {
			return text
		}
		if prompt, ok := content["prompt"].(string); ok {
			return prompt
		}
		// Fallback: convert to string
		return fmt.Sprintf("%v", content)
	default:
		return fmt.Sprintf("%v", content)
	}
}
