package memory

import (
	"fmt"
	"testing"

	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// BUFFER WINDOW TESTS
// ============================================================================

func TestMemoryService_BufferWindow_BasicOperations(t *testing.T) {
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 5})
	require.NoError(t, err)

	service := NewMemoryService(strategy)

	sessionID := "test-session"

	// Add messages
	messages := []llms.Message{
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Message 2"},
	}

	for _, msg := range messages {
		err := service.AddToHistory(sessionID, msg)
		require.NoError(t, err)
	}

	// Get history
	history, err := service.GetRecentHistory(sessionID)
	require.NoError(t, err)
	assert.Equal(t, 3, len(history))
}

func TestMemoryService_BufferWindow_WindowEnforcement(t *testing.T) {
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 3})
	require.NoError(t, err)

	service := NewMemoryService(strategy)
	sessionID := "test-fifo"

	// Add 5 messages (exceeds window of 3)
	for i := 1; i <= 5; i++ {
		err := service.AddToHistory(sessionID, llms.Message{
			Role:    "user",
			Content: "Message " + string(rune('0'+i)),
		})
		require.NoError(t, err)
	}

	// Should only have last 3
	history, err := service.GetRecentHistory(sessionID)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(history), 3)
}

func TestMemoryService_BufferWindow_SessionIsolation(t *testing.T) {
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 5})
	require.NoError(t, err)

	service := NewMemoryService(strategy)

	// Add to session 1
	service.AddToHistory("session1", llms.Message{Role: "user", Content: "S1-M1"})
	service.AddToHistory("session1", llms.Message{Role: "user", Content: "S1-M2"})

	// Add to session 2
	service.AddToHistory("session2", llms.Message{Role: "user", Content: "S2-M1"})

	// Verify isolation
	h1, _ := service.GetRecentHistory("session1")
	h2, _ := service.GetRecentHistory("session2")

	assert.Equal(t, 2, len(h1))
	assert.Equal(t, 1, len(h2))
}

func TestMemoryService_BufferWindow_Clear(t *testing.T) {
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 5})
	require.NoError(t, err)

	service := NewMemoryService(strategy)
	sessionID := "test-clear"

	// Add messages
	for i := 1; i <= 3; i++ {
		service.AddToHistory(sessionID, llms.Message{
			Role:    "user",
			Content: "Message " + string(rune('0'+i)),
		})
	}

	// Clear
	err = service.ClearHistory(sessionID)
	require.NoError(t, err)

	// Verify empty
	history, err := service.GetRecentHistory(sessionID)
	require.NoError(t, err)
	assert.Empty(t, history)
}

// ============================================================================
// SUMMARY BUFFER TESTS
// ============================================================================

func TestMemoryService_SummaryBuffer_BasicOperations(t *testing.T) {
	summarizer := &DeterministicSummarizer{}

	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     2000,
		Threshold:  0.8,
		Target:     0.6,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	service := NewMemoryService(strategy)
	sessionID := "test-session"

	// Add a message
	err = service.AddToHistory(sessionID, llms.Message{
		Role:    "user",
		Content: "Hello, this is a test message.",
	})
	require.NoError(t, err)

	// Get history
	history, err := service.GetRecentHistory(sessionID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(history))
}

func TestMemoryService_SummaryBuffer_Summarization(t *testing.T) {
	summarizer := &DeterministicSummarizer{}

	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     200, // Very small to trigger quickly
		Threshold:  0.8,
		Target:     0.6,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	service := NewMemoryService(strategy)
	sessionID := "test-trigger"

	// Add messages until we exceed threshold
	for i := 1; i <= 15; i++ {
		err := service.AddToHistory(sessionID, llms.Message{
			Role:    "user",
			Content: fmt.Sprintf("This is test message number %d with some content to increase token count", i),
		})
		require.NoError(t, err)
	}

	// Should have triggered summarization
	assert.Greater(t, summarizer.CallCount, 0, "Summarization should have been triggered")

	// History should have summary + recent messages
	history, err := service.GetRecentHistory(sessionID)
	require.NoError(t, err)
	assert.Greater(t, len(history), 0)
}

func TestMemoryService_SummaryBuffer_SessionIsolation(t *testing.T) {
	summarizer := &DeterministicSummarizer{}

	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     1000,
		Threshold:  0.8,
		Target:     0.6,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	service := NewMemoryService(strategy)

	// Add to different sessions
	service.AddToHistory("session1", llms.Message{Role: "user", Content: "S1-M1"})
	service.AddToHistory("session1", llms.Message{Role: "assistant", Content: "S1-R1"})

	service.AddToHistory("session2", llms.Message{Role: "user", Content: "S2-M1"})
	service.AddToHistory("session2", llms.Message{Role: "assistant", Content: "S2-R1"})
	service.AddToHistory("session2", llms.Message{Role: "user", Content: "S2-M2"})

	// Verify each session has correct messages
	h1, _ := service.GetRecentHistory("session1")
	h2, _ := service.GetRecentHistory("session2")

	assert.Equal(t, 2, len(h1))
	assert.Equal(t, 3, len(h2))
}

func TestMemoryService_SummaryBuffer_MinimumMessages(t *testing.T) {
	summarizer := &DeterministicSummarizer{}

	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     200,
		Threshold:  0.8,
		Target:     0.6,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	service := NewMemoryService(strategy)
	sessionID := "test-min"

	// Add enough messages to trigger summarization
	for i := 1; i <= 12; i++ {
		err := service.AddToHistory(sessionID, llms.Message{
			Role:    "user",
			Content: fmt.Sprintf("Message number %d with content for token counting", i),
		})
		require.NoError(t, err)
	}

	// Verify summarization was triggered and minimum messages kept
	history, err := service.GetRecentHistory(sessionID)
	require.NoError(t, err)

	// Should have at least 3 recent messages + 1 summary
	assert.GreaterOrEqual(t, len(history), 3, "Should keep at least 3 recent messages")
}

// ============================================================================
// GENERAL SERVICE TESTS
// ============================================================================

func TestMemoryService_GetSessionCount(t *testing.T) {
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 5})
	require.NoError(t, err)

	service := NewMemoryService(strategy)

	// Initially 0
	assert.Equal(t, 0, service.GetSessionCount())

	// Add to different sessions
	service.AddToHistory("A", llms.Message{Role: "user", Content: "Test"})
	service.AddToHistory("B", llms.Message{Role: "user", Content: "Test"})
	service.AddToHistory("A", llms.Message{Role: "user", Content: "Test"})

	// Should have 2 sessions
	assert.Equal(t, 2, service.GetSessionCount())
}

func TestMemoryService_DefaultSessionID(t *testing.T) {
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 5})
	require.NoError(t, err)

	service := NewMemoryService(strategy)

	// Empty string should default to "default"
	err = service.AddToHistory("", llms.Message{Role: "user", Content: "Test"})
	require.NoError(t, err)

	history, err := service.GetRecentHistory("")
	require.NoError(t, err)
	assert.Equal(t, 1, len(history))
}

func TestMemoryService_StatusNotifier(t *testing.T) {
	summarizer := &DeterministicSummarizer{}

	strategy, err := NewSummaryBufferStrategy(SummaryBufferConfig{
		Budget:     200,
		Threshold:  0.8,
		Target:     0.6,
		Model:      "gpt-4o",
		Summarizer: summarizer,
	})
	require.NoError(t, err)

	service := NewMemoryService(strategy)

	// Set up notifier
	notified := false
	service.SetStatusNotifier(func(msg string) {
		if msg != "" {
			notified = true
		}
	})

	// Add messages to trigger summarization
	for i := 1; i <= 15; i++ {
		service.AddToHistory("test", llms.Message{
			Role:    "user",
			Content: fmt.Sprintf("Message %d with content to trigger summarization", i),
		})
	}

	// Should have been notified
	assert.True(t, notified, "Status notifier should have been called")
}
