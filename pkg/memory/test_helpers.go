package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
)

// ============================================================================
// TEST HELPERS
// Shared helpers for all history strategy tests
// ============================================================================

// DeterministicSummarizer provides predictable summarization for testing
type DeterministicSummarizer struct {
	CallCount    int
	SummaryCalls [][]*pb.Message // Track what was summarized
}

// SummarizeConversation creates deterministic summaries
func (d *DeterministicSummarizer) SummarizeConversation(ctx context.Context, messages []*pb.Message) (string, error) {
	d.CallCount++
	d.SummaryCalls = append(d.SummaryCalls, messages)

	// Create deterministic summary
	var parts []string
	for _, msg := range messages {
		textContent := protocol.ExtractTextFromMessage(msg)
		contentPreview := textContent
		if len(textContent) > 10 {
			contentPreview = textContent[:10]
		}
		parts = append(parts, fmt.Sprintf("%s:%s", string(msg.Role), contentPreview))
	}
	return fmt.Sprintf("SUMMARY#%d[%s]", d.CallCount, strings.Join(parts, ",")), nil
}
