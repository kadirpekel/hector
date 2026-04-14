package llmagent

import (
	"fmt"
	"strings"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/verikod/hector/pkg/agent"
)

// processMessageContent extracts text content from an a2a.Message.
func processMessageContent(msg *a2a.Message) string {
	if msg == nil {
		return ""
	}

	var text string
	for _, part := range msg.Parts {
		if tp, ok := part.(a2a.TextPart); ok {
			text += tp.Text
		}
	}
	return text
}

// buildMemoryContextMessage creates a context message from memory search results.
func buildMemoryContextMessage(results []agent.MemoryResult, limit int) *a2a.Message {
	if len(results) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("Retrieved relevant memories for context:\n")

	count := 0
	for _, res := range results {
		if count >= limit {
			break
		}
		// Format: [Score: 0.95] Memory content...
		sb.WriteString(fmt.Sprintf("\n[Similarity: %.2f] %s\n", res.Score, res.Content))
		count++
	}

	// Create a System message containing the memories
	// This helps distinguish it from the conversation history
	return a2a.NewMessage(
		a2a.MessageRoleUser,
		a2a.TextPart{Text: sb.String()},
	)
}
