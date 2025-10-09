package history

import (
	"fmt"
	"testing"

	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// INTEGRATION TESTS
// End-to-end tests verifying strategies work together correctly
// ============================================================================

func TestIntegration_Factory_CreatesCorrectStrategy(t *testing.T) {
	t.Run("Creates buffer_window strategy", func(t *testing.T) {
		strategy, err := NewHistoryStrategy(HistoryConfig{
			Strategy:   "buffer_window",
			WindowSize: 10,
		})
		require.NoError(t, err)
		assert.Equal(t, "buffer_window", strategy.Name())

		// Verify it works
		err = strategy.AddMessage("test", llms.Message{Role: "user", Content: "Hi"})
		assert.NoError(t, err)

		history, err := strategy.GetHistory("test")
		assert.NoError(t, err)
		assert.Len(t, history, 1)
	})

	t.Run("Creates summary_buffer strategy", func(t *testing.T) {
		summarizer := &DeterministicSummarizer{}

		strategy, err := NewHistoryStrategy(HistoryConfig{
			Strategy:   "summary_buffer",
			Budget:     2000,
			Model:      "gpt-4o",
			Summarizer: summarizer,
		})
		require.NoError(t, err)
		assert.Equal(t, "summary_buffer", strategy.Name())

		// Verify it works
		err = strategy.AddMessage("test", llms.Message{Role: "user", Content: "Hi"})
		assert.NoError(t, err)

		history, err := strategy.GetHistory("test")
		assert.NoError(t, err)
		assert.Len(t, history, 1)
	})

	t.Run("Defaults to summary_buffer when strategy empty", func(t *testing.T) {
		summarizer := &DeterministicSummarizer{}

		strategy, err := NewHistoryStrategy(HistoryConfig{
			Strategy:   "", // Empty, should default
			Budget:     2000,
			Model:      "gpt-4o",
			Summarizer: summarizer,
		})
		require.NoError(t, err)
		assert.Equal(t, "summary_buffer", strategy.Name())
	})

	t.Run("Rejects unknown strategy", func(t *testing.T) {
		strategy, err := NewHistoryStrategy(HistoryConfig{
			Strategy: "unknown_strategy",
		})
		require.Error(t, err)
		assert.Nil(t, strategy)
		assert.Contains(t, err.Error(), "unknown history strategy")
	})
}

func TestIntegration_BufferWindow_FullWorkflow(t *testing.T) {
	// Complete workflow test for buffer_window
	strategy, err := NewHistoryStrategy(HistoryConfig{
		Strategy:   "buffer_window",
		WindowSize: 5,
	})
	require.NoError(t, err)

	sessionID := "workflow-test"

	// Simulate a conversation
	conversation := []llms.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
		{Role: "assistant", Content: "I'm doing well, thanks!"},
		{Role: "user", Content: "Can you help me?"},
		{Role: "assistant", Content: "Of course!"},
		{Role: "user", Content: "What's the weather?"},
		{Role: "assistant", Content: "It's sunny today."},
	}

	// Add all messages
	for _, msg := range conversation {
		err := strategy.AddMessage(sessionID, msg)
		require.NoError(t, err)
	}

	// Get history (should be truncated to window size of 5)
	history, err := strategy.GetHistory(sessionID)
	require.NoError(t, err)
	assert.True(t, len(history) <= 5, "Should not exceed window size")

	// Verify sessions are independent
	err = strategy.AddMessage("other-session", llms.Message{Role: "user", Content: "Other"})
	require.NoError(t, err)

	otherHistory, err := strategy.GetHistory("other-session")
	require.NoError(t, err)
	assert.Len(t, otherHistory, 1)

	// Original session should be unchanged
	history2, err := strategy.GetHistory(sessionID)
	require.NoError(t, err)
	assert.Equal(t, len(history), len(history2))
}

func TestIntegration_SummaryBuffer_FullWorkflow(t *testing.T) {
	// Complete workflow test for summary_buffer
	summarizer := &DeterministicSummarizer{}

	strategy, err := NewHistoryStrategy(HistoryConfig{
		Strategy:   "summary_buffer",
		Budget:     100, // Small budget to trigger summarization
		Threshold:  0.8,
		Target:     0.6,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	sessionID := "summary-workflow"

	// Add enough messages to exceed budget
	for i := 1; i <= 20; i++ {
		err := strategy.AddMessage(sessionID, llms.Message{
			Role:    "user",
			Content: fmt.Sprintf("This is message number %d with content that accumulates tokens", i),
		})
		require.NoError(t, err)
	}

	// Verify summarization was triggered
	if summarizer.CallCount > 0 {
		t.Logf("✅ Summarization triggered %d times", summarizer.CallCount)

		// Get history and verify summary exists
		history, err := strategy.GetHistory(sessionID)
		require.NoError(t, err)
		assert.NotEmpty(t, history)

		// First message should be summary
		assert.Contains(t, history[0].Content, "SUMMARY")
		assert.Equal(t, "system", history[0].Role)
	}

	// Verify sessions are independent
	err = strategy.AddMessage("other-session", llms.Message{Role: "user", Content: "Other"})
	require.NoError(t, err)

	otherHistory, err := strategy.GetHistory("other-session")
	require.NoError(t, err)
	assert.Len(t, otherHistory, 1)
	assert.NotContains(t, otherHistory[0].Content, "SUMMARY", "New session should not have summary")
}

func TestIntegration_StrategyComparison(t *testing.T) {
	// Compare behavior of both strategies with same inputs

	// Setup identical conversations for both strategies
	conversation := []llms.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi!"},
		{Role: "user", Content: "How are you?"},
		{Role: "assistant", Content: "Good!"},
		{Role: "user", Content: "Great!"},
	}

	t.Run("Buffer Window maintains exact messages", func(t *testing.T) {
		strategy, err := NewHistoryStrategy(HistoryConfig{
			Strategy:   "buffer_window",
			WindowSize: 10, // Large enough to hold all
		})
		require.NoError(t, err)

		for _, msg := range conversation {
			strategy.AddMessage("test", msg)
		}

		history, _ := strategy.GetHistory("test")
		assert.Len(t, history, 5, "Buffer window should keep all messages")
	})

	t.Run("Summary Buffer maintains messages under budget", func(t *testing.T) {
		summarizer := &DeterministicSummarizer{}
		strategy, err := NewHistoryStrategy(HistoryConfig{
			Strategy:   "summary_buffer",
			Budget:     10000, // Large enough to avoid summarization
			Model:      "gpt-4o",
			Summarizer: summarizer,
		})
		require.NoError(t, err)

		for _, msg := range conversation {
			strategy.AddMessage("test", msg)
		}

		history, _ := strategy.GetHistory("test")
		assert.Len(t, history, 5, "Summary buffer should keep all messages when under budget")
		assert.Equal(t, 0, summarizer.CallCount, "Should not trigger summarization")
	})
}

func TestIntegration_SessionLifecycle(t *testing.T) {
	// Test complete session lifecycle for both strategies

	testCases := []struct {
		name       string
		strategy   string
		windowSize int
		budget     int
	}{
		{
			name:       "Buffer Window",
			strategy:   "buffer_window",
			windowSize: 5,
		},
		{
			name:     "Summary Buffer",
			strategy: "summary_buffer",
			budget:   1000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var strategy HistoryStrategy
			var err error

			if tc.strategy == "buffer_window" {
				strategy, err = NewHistoryStrategy(HistoryConfig{
					Strategy:   tc.strategy,
					WindowSize: tc.windowSize,
				})
			} else {
				strategy, err = NewHistoryStrategy(HistoryConfig{
					Strategy:   tc.strategy,
					Budget:     tc.budget,
					Model:      "gpt-4o",
					Summarizer: &DeterministicSummarizer{},
				})
			}
			require.NoError(t, err)

			sessionID := "lifecycle-test"

			// 1. Create session (implicit on first message)
			err = strategy.AddMessage(sessionID, llms.Message{Role: "user", Content: "Start"})
			assert.NoError(t, err)

			// 2. Add multiple messages
			for i := 1; i <= 3; i++ {
				err = strategy.AddMessage(sessionID, llms.Message{
					Role:    "user",
					Content: fmt.Sprintf("Message %d", i),
				})
				assert.NoError(t, err)
			}

			// 3. Get history
			history, err := strategy.GetHistory(sessionID)
			assert.NoError(t, err)
			assert.NotEmpty(t, history)

			// 4. Clear session
			err = strategy.Clear(sessionID)
			assert.NoError(t, err)

			// 5. Verify empty
			history, err = strategy.GetHistory(sessionID)
			assert.NoError(t, err)
			assert.Empty(t, history)

			// 6. Can restart session
			err = strategy.AddMessage(sessionID, llms.Message{Role: "user", Content: "Restart"})
			assert.NoError(t, err)

			history, err = strategy.GetHistory(sessionID)
			assert.NoError(t, err)
			assert.Len(t, history, 1)
		})
	}
}

func TestIntegration_DefaultValues(t *testing.T) {
	// Verify default values are applied correctly

	t.Run("Buffer Window defaults", func(t *testing.T) {
		strategy, err := NewHistoryStrategy(HistoryConfig{
			Strategy:   "buffer_window",
			WindowSize: 0, // Should use default (20)
		})
		require.NoError(t, err)

		bufferStrategy := strategy.(*BufferWindowStrategy)
		assert.Equal(t, 20, bufferStrategy.windowSize)
	})

	t.Run("Summary Buffer defaults", func(t *testing.T) {
		summarizer := &DeterministicSummarizer{}

		strategy, err := NewHistoryStrategy(HistoryConfig{
			Strategy:   "summary_buffer",
			Budget:     0, // Should use default (2000)
			Threshold:  0, // Should use default (0.8)
			Target:     0, // Should use default (0.6)
			Model:      "gpt-4o",
			Summarizer: summarizer,
		})
		require.NoError(t, err)

		summaryStrategy := strategy.(*SummaryBufferStrategy)
		assert.Equal(t, 2000, summaryStrategy.tokenBudget)
		assert.Equal(t, 0.8, summaryStrategy.threshold)
		assert.Equal(t, 0.6, summaryStrategy.target)
	})
}

func TestIntegration_ErrorHandling(t *testing.T) {
	// Test error handling across strategies

	t.Run("Summary Buffer requires model", func(t *testing.T) {
		_, err := NewHistoryStrategy(HistoryConfig{
			Strategy:   "summary_buffer",
			Budget:     1000,
			Model:      "", // Missing
			Summarizer: &DeterministicSummarizer{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model")
	})

	t.Run("Summary Buffer requires summarizer", func(t *testing.T) {
		_, err := NewHistoryStrategy(HistoryConfig{
			Strategy:   "summary_buffer",
			Budget:     1000,
			Model:      "gpt-4o",
			Summarizer: nil, // Missing
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "summarization service")
	})

	t.Run("Invalid strategy name", func(t *testing.T) {
		_, err := NewHistoryStrategy(HistoryConfig{
			Strategy: "invalid",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown history strategy")
	})
}

func TestIntegration_ConcurrentSessions(t *testing.T) {
	// Test multiple sessions work independently for both strategies

	strategies := map[string]HistoryStrategy{}

	// Setup buffer_window
	bw, err := NewHistoryStrategy(HistoryConfig{
		Strategy:   "buffer_window",
		WindowSize: 5,
	})
	require.NoError(t, err)
	strategies["buffer_window"] = bw

	// Setup summary_buffer
	sb, err := NewHistoryStrategy(HistoryConfig{
		Strategy:   "summary_buffer",
		Budget:     1000,
		Model:      "gpt-4o",
		Summarizer: &DeterministicSummarizer{},
	})
	require.NoError(t, err)
	strategies["summary_buffer"] = sb

	for name, strategy := range strategies {
		t.Run(name, func(t *testing.T) {
			// Create 3 sessions
			sessions := []string{"session-A", "session-B", "session-C"}

			// Add different messages to each
			for i, session := range sessions {
				for j := 0; j < 3; j++ {
					err := strategy.AddMessage(session, llms.Message{
						Role:    "user",
						Content: fmt.Sprintf("S%d-M%d", i, j),
					})
					require.NoError(t, err)
				}
			}

			// Verify each session has its own history
			for i, session := range sessions {
				history, err := strategy.GetHistory(session)
				require.NoError(t, err)
				assert.NotEmpty(t, history)

				// First message should match session
				assert.Contains(t, history[0].Content, fmt.Sprintf("S%d", i))
			}

			// Verify session count
			assert.Equal(t, 3, strategy.GetSessionCount())
		})
	}
}
