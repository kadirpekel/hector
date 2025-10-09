package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/llms"
)

// ============================================================================
// TEST HELPERS
// Shared helpers for all history strategy tests
// ============================================================================

// DeterministicSummarizer provides predictable summarization for testing
type DeterministicSummarizer struct {
	CallCount    int
	SummaryCalls [][]llms.Message // Track what was summarized
}

// SummarizeConversation creates deterministic summaries
func (d *DeterministicSummarizer) SummarizeConversation(ctx context.Context, messages []llms.Message) (string, error) {
	d.CallCount++
	d.SummaryCalls = append(d.SummaryCalls, messages)

	// Create deterministic summary
	var parts []string
	for _, msg := range messages {
		parts = append(parts, fmt.Sprintf("%s:%s", msg.Role, msg.Content[:min(10, len(msg.Content))]))
	}
	return fmt.Sprintf("SUMMARY#%d[%s]", d.CallCount, strings.Join(parts, ",")), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
