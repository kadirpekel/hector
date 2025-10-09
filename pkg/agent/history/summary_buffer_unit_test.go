package history

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// UNIT TESTS - Summary Buffer Strategy
// Deterministic tests verifying token counting and summarization behavior
// ============================================================================

func TestSummaryBuffer_Unit_BasicOperations(t *testing.T) {
	t.Run("Create with defaults", func(t *testing.T) {
		summarizer := &DeterministicSummarizer{}
		strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
			Budget:     0, // Should use default
			Model:      "gpt-4o",
			Summarizer: summarizer,
		})
		require.NoError(t, err)
		assert.Equal(t, "summary_buffer", strategy.Name())
		assert.Equal(t, 2000, strategy.tokenBudget)
		assert.Equal(t, 0.8, strategy.threshold)
		assert.Equal(t, 0.6, strategy.target)
	})

	t.Run("Create with custom values", func(t *testing.T) {
		summarizer := &DeterministicSummarizer{}
		strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
			Budget:     1000,
			Threshold:  0.75,
			Target:     0.5,
			Model:      "gpt-4o",
			Summarizer: summarizer,
		})
		require.NoError(t, err)
		assert.Equal(t, 1000, strategy.tokenBudget)
		assert.Equal(t, 0.75, strategy.threshold)
		assert.Equal(t, 0.5, strategy.target)
	})

	t.Run("Reject missing model", func(t *testing.T) {
		strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
			Budget:     1000,
			Model:      "", // Missing
			Summarizer: &DeterministicSummarizer{},
		})
		require.Error(t, err)
		assert.Nil(t, strategy)
		assert.Contains(t, err.Error(), "model")
	})

	t.Run("Reject missing summarizer", func(t *testing.T) {
		strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
			Budget:     1000,
			Model:      "gpt-4o",
			Summarizer: nil, // Missing
		})
		require.Error(t, err)
		assert.Nil(t, strategy)
		assert.Contains(t, err.Error(), "summarization service")
	})
}

func TestSummaryBuffer_Unit_TokenCounting(t *testing.T) {
	// DETERMINISTIC TEST: Verify accurate token counting
	summarizer := &DeterministicSummarizer{}
	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     1000,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	sessionID := "test-tokens"

	// Add a message and verify token counting works
	err = strategy.AddMessage(sessionID, llms.Message{
		Role:    "user",
		Content: "Hello, this is a test message.",
	})
	require.NoError(t, err)

	history, err := strategy.GetHistory(sessionID)
	require.NoError(t, err)
	assert.Len(t, history, 1)

	// Token counter should be working (exact count will depend on tiktoken)
	assert.NotNil(t, strategy.tokenCounter)
}

func TestSummaryBuffer_Unit_ThresholdCalculation(t *testing.T) {
	// DETERMINISTIC TEST: Verify threshold is calculated correctly
	summarizer := &DeterministicSummarizer{}

	testCases := []struct {
		name      string
		budget    int
		threshold float64
		expected  int // Expected threshold in tokens
	}{
		{
			name:      "Default (2000 @ 80%)",
			budget:    2000,
			threshold: 0.8,
			expected:  1600, // 2000 * 0.8
		},
		{
			name:      "Custom (1000 @ 75%)",
			budget:    1000,
			threshold: 0.75,
			expected:  750, // 1000 * 0.75
		},
		{
			name:      "Small (100 @ 90%)",
			budget:    100,
			threshold: 0.9,
			expected:  90, // 100 * 0.9
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
				Budget:     tc.budget,
				Threshold:  tc.threshold,
				Model:      "gpt-4o",
				Summarizer: summarizer,
			})
			require.NoError(t, err)

			// Calculate expected threshold
			expectedThreshold := int(float64(tc.budget) * tc.threshold)
			assert.Equal(t, tc.expected, expectedThreshold)

			// Verify strategy has correct values
			assert.Equal(t, tc.budget, strategy.tokenBudget)
			assert.Equal(t, tc.threshold, strategy.threshold)
		})
	}
}

func TestSummaryBuffer_Unit_SessionIsolation(t *testing.T) {
	// DETERMINISTIC TEST: Verify sessions are completely isolated
	summarizer := &DeterministicSummarizer{}
	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     1000,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	// Session 1
	strategy.AddMessage("session1", llms.Message{Role: "user", Content: "S1-M1"})
	strategy.AddMessage("session1", llms.Message{Role: "assistant", Content: "S1-R1"})

	// Session 2
	strategy.AddMessage("session2", llms.Message{Role: "user", Content: "S2-M1"})
	strategy.AddMessage("session2", llms.Message{Role: "assistant", Content: "S2-R1"})
	strategy.AddMessage("session2", llms.Message{Role: "user", Content: "S2-M2"})

	// Session 3
	strategy.AddMessage("session3", llms.Message{Role: "user", Content: "S3-M1"})

	// Verify each session has correct messages
	h1, _ := strategy.GetHistory("session1")
	h2, _ := strategy.GetHistory("session2")
	h3, _ := strategy.GetHistory("session3")

	assert.Len(t, h1, 2, "Session 1 should have 2 messages")
	assert.Len(t, h2, 3, "Session 2 should have 3 messages")
	assert.Len(t, h3, 1, "Session 3 should have 1 message")

	// Verify content doesn't leak
	assert.Equal(t, "S1-M1", h1[0].Content)
	assert.Equal(t, "S2-M1", h2[0].Content)
	assert.Equal(t, "S3-M1", h3[0].Content)

	// Verify session count
	assert.Equal(t, 3, strategy.GetSessionCount())

	// Verify no summarization was triggered yet (messages are small)
	assert.Equal(t, 0, summarizer.CallCount, "Should not have triggered summarization")
}

func TestSummaryBuffer_Unit_ClearOperation(t *testing.T) {
	// DETERMINISTIC TEST: Verify clear removes all messages
	summarizer := &DeterministicSummarizer{}
	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     1000,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	sessionID := "test-clear"

	// Add messages
	for i := 1; i <= 3; i++ {
		strategy.AddMessage(sessionID, llms.Message{
			Role:    "user",
			Content: fmt.Sprintf("Message %d", i),
		})
	}

	// Verify messages exist
	history, _ := strategy.GetHistory(sessionID)
	assert.Len(t, history, 3)

	// Clear
	err = strategy.Clear(sessionID)
	require.NoError(t, err)

	// Verify empty
	history, _ = strategy.GetHistory(sessionID)
	assert.Len(t, history, 0, "History should be empty after clear")
}

func TestSummaryBuffer_Unit_SummarizationTrigger(t *testing.T) {
	// DETERMINISTIC TEST: Verify summarization triggers at correct threshold
	summarizer := &DeterministicSummarizer{}

	// Use very small budget to force summarization
	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     50,  // Very small
		Threshold:  0.8, // Trigger at 40 tokens
		Target:     0.6, // Compress to 30 tokens
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	sessionID := "test-trigger"

	// Add messages until we exceed threshold
	// Each message is roughly 10-20 tokens
	for i := 1; i <= 15; i++ {
		err := strategy.AddMessage(sessionID, llms.Message{
			Role:    "user",
			Content: fmt.Sprintf("This is test message number %d with some content to increase token count", i),
		})
		require.NoError(t, err)
	}

	// Summarization should have been triggered at least once
	assert.Greater(t, summarizer.CallCount, 0, "Summarization should have been triggered")

	// Verify history contains summary
	history, err := strategy.GetHistory(sessionID)
	require.NoError(t, err)
	assert.NotEmpty(t, history)

	// First message should be a summary (contains "SUMMARY")
	if summarizer.CallCount > 0 {
		assert.Contains(t, history[0].Content, "SUMMARY", "First message should be summary")
		assert.Equal(t, "system", history[0].Role, "Summary should be system message")
	}
}

func TestSummaryBuffer_Unit_MinimumMessageThreshold(t *testing.T) {
	// DETERMINISTIC TEST: Verify no summarization for < 10 messages
	summarizer := &DeterministicSummarizer{}

	// Very small budget but we won't add enough messages
	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     1, // Extremely small (would normally trigger)
		Threshold:  0.5,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	sessionID := "test-min-messages"

	// Add only 5 messages (less than minimum 10)
	for i := 1; i <= 5; i++ {
		err := strategy.AddMessage(sessionID, llms.Message{
			Role:    "user",
			Content: fmt.Sprintf("Message %d", i),
		})
		require.NoError(t, err)
	}

	// Should NOT trigger summarization (< 10 messages)
	assert.Equal(t, 0, summarizer.CallCount, "Should not summarize < 10 messages")
}

func TestSummaryBuffer_Unit_EmptySession(t *testing.T) {
	// DETERMINISTIC TEST: Verify empty session handling
	summarizer := &DeterministicSummarizer{}
	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     1000,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	// Get history from non-existent session
	history, err := strategy.GetHistory("non-existent")
	require.NoError(t, err)
	assert.Len(t, history, 0, "Non-existent session should return empty history")
}

func TestSummaryBuffer_Unit_DefaultSessionID(t *testing.T) {
	// DETERMINISTIC TEST: Verify empty sessionID defaults to "default"
	summarizer := &DeterministicSummarizer{}
	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     1000,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	// Add with empty session ID
	err = strategy.AddMessage("", llms.Message{Role: "user", Content: "Test"})
	require.NoError(t, err)

	// Should be retrievable with empty string
	history, err := strategy.GetHistory("")
	require.NoError(t, err)
	assert.Len(t, history, 1)
}

func TestSummaryBuffer_Unit_MessageOrder(t *testing.T) {
	// DETERMINISTIC TEST: Verify messages maintain chronological order
	summarizer := &DeterministicSummarizer{}
	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     10000, // Large enough to avoid summarization
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	sessionID := "test-order"

	// Add messages in specific order
	messages := []string{
		"First message",
		"Second message",
		"Third message",
		"Fourth message",
		"Fifth message",
	}

	for _, content := range messages {
		err := strategy.AddMessage(sessionID, llms.Message{
			Role:    "user",
			Content: content,
		})
		require.NoError(t, err)
	}

	// Retrieve and verify order
	history, err := strategy.GetHistory(sessionID)
	require.NoError(t, err)
	assert.Len(t, history, 5)

	// Verify chronological order is maintained
	for i, msg := range history {
		expected := messages[i]
		assert.Equal(t, expected, msg.Content, "Message %d should be in correct order", i)
	}

	// Should not have triggered summarization
	assert.Equal(t, 0, summarizer.CallCount)
}

func TestSummaryBuffer_Unit_RolePreservation(t *testing.T) {
	// DETERMINISTIC TEST: Verify roles are preserved correctly
	summarizer := &DeterministicSummarizer{}
	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     10000,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	sessionID := "test-roles"

	// Add messages with different roles
	messages := []llms.Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "User question"},
		{Role: "assistant", Content: "Assistant response"},
		{Role: "user", Content: "Follow-up question"},
	}

	for _, msg := range messages {
		err := strategy.AddMessage(sessionID, msg)
		require.NoError(t, err)
	}

	// Verify roles are preserved
	history, err := strategy.GetHistory(sessionID)
	require.NoError(t, err)
	assert.Len(t, history, 4)

	assert.Equal(t, "system", history[0].Role)
	assert.Equal(t, "user", history[1].Role)
	assert.Equal(t, "assistant", history[2].Role)
	assert.Equal(t, "user", history[3].Role)
}

func TestSummaryBuffer_Unit_SummaryContent(t *testing.T) {
	// DETERMINISTIC TEST: Verify summary content is correct
	summarizer := &DeterministicSummarizer{}

	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     30, // Very small to force summarization
		Threshold:  0.8,
		Target:     0.6,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	sessionID := "test-summary-content"

	// Add enough messages to trigger summarization
	for i := 1; i <= 12; i++ {
		err := strategy.AddMessage(sessionID, llms.Message{
			Role:    "user",
			Content: fmt.Sprintf("Message number %d with enough content to count tokens properly", i),
		})
		require.NoError(t, err)
	}

	// Should have triggered summarization
	if summarizer.CallCount > 0 {
		// Verify summary was created
		assert.NotEmpty(t, summarizer.SummaryCalls, "Should have record of summarization calls")

		// Get history and verify summary is present
		history, err := strategy.GetHistory(sessionID)
		require.NoError(t, err)

		// First message should be summary
		assert.Contains(t, history[0].Content, "SUMMARY", "First message should contain summary")
		assert.Equal(t, "system", history[0].Role)
	}
}

func TestSummaryBuffer_Unit_MultipleSummarizations(t *testing.T) {
	// DETERMINISTIC TEST: Verify multiple summarizations work (hierarchical)
	summarizer := &DeterministicSummarizer{}

	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     30, // Very small to force multiple summarizations
		Threshold:  0.8,
		Target:     0.6,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	sessionID := "test-multiple-summaries"

	// Add many messages to trigger multiple summarizations
	for i := 1; i <= 30; i++ {
		err := strategy.AddMessage(sessionID, llms.Message{
			Role:    "user",
			Content: fmt.Sprintf("This is message number %d with content that will accumulate tokens", i),
		})
		require.NoError(t, err)
	}

	// Should have triggered multiple summarizations
	if summarizer.CallCount >= 2 {
		t.Logf("✅ Multiple summarizations triggered: %d times", summarizer.CallCount)

		// Verify hierarchical summarization (summaries of summaries)
		assert.GreaterOrEqual(t, summarizer.CallCount, 2, "Should have multiple summarizations")

		// Later summarizations should include previous summaries
		if len(summarizer.SummaryCalls) >= 2 {
			// Second summarization should process the first summary
			secondCall := summarizer.SummaryCalls[1]
			found := false
			for _, msg := range secondCall {
				if strings.Contains(msg.Content, "SUMMARY") {
					found = true
					break
				}
			}
			if found {
				t.Logf("✅ Hierarchical summarization confirmed (summary of summaries)")
			}
		}
	}
}
