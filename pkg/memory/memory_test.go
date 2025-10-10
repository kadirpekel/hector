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

	service := NewMemoryService(strategy, nil, LongTermConfig{})

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

	service := NewMemoryService(strategy, nil, LongTermConfig{})
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

	service := NewMemoryService(strategy, nil, LongTermConfig{})

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

	service := NewMemoryService(strategy, nil, LongTermConfig{})
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

	service := NewMemoryService(strategy, nil, LongTermConfig{})
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

	service := NewMemoryService(strategy, nil, LongTermConfig{})
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

	service := NewMemoryService(strategy, nil, LongTermConfig{})

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

	service := NewMemoryService(strategy, nil, LongTermConfig{})
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

	service := NewMemoryService(strategy, nil, LongTermConfig{})

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

	service := NewMemoryService(strategy, nil, LongTermConfig{})

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

	service := NewMemoryService(strategy, nil, LongTermConfig{})

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

// ============================================================================
// LONG-TERM MEMORY INTEGRATION TESTS
// ============================================================================

func TestMemoryService_LongTermMemory_BasicStorage(t *testing.T) {
	workingStrategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 5})
	require.NoError(t, err)

	db := NewMockDatabaseProvider()
	embedder := NewMockEmbedderProvider()
	longTermStrategy, err := NewVectorMemoryStrategy(db, embedder, "test_collection")
	require.NoError(t, err)

	service := NewMemoryService(
		workingStrategy,
		longTermStrategy,
		LongTermConfig{
			Enabled:      true,
			BatchSize:    1, // Immediate storage
			StorageScope: StorageScopeAll,
			AutoRecall:   false, // Disable for this test
			RecallLimit:  5,
			Collection:   "test_collection",
		},
	)

	sessionID := "test-session"

	// Add messages
	service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "Test message 1"})
	service.AddToHistory(sessionID, llms.Message{Role: "assistant", Content: "Response 1"})

	// Verify stored in long-term memory
	assert.Equal(t, 2, db.GetStoredCount("test_collection"))
}

func TestMemoryService_LongTermMemory_Batching(t *testing.T) {
	workingStrategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 10})
	require.NoError(t, err)

	db := NewMockDatabaseProvider()
	embedder := NewMockEmbedderProvider()
	longTermStrategy, err := NewVectorMemoryStrategy(db, embedder, "batch_collection")
	require.NoError(t, err)

	service := NewMemoryService(
		workingStrategy,
		longTermStrategy,
		LongTermConfig{
			Enabled:      true,
			BatchSize:    3, // Batch every 3 messages
			StorageScope: StorageScopeAll,
			AutoRecall:   false,
			RecallLimit:  5,
			Collection:   "batch_collection",
		},
	)

	sessionID := "batch-session"

	// Add 2 messages (should not flush yet)
	service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "Message 1"})
	service.AddToHistory(sessionID, llms.Message{Role: "assistant", Content: "Response 1"})
	assert.Equal(t, 0, db.GetStoredCount("batch_collection"), "Should not flush before batch size")

	// Add 3rd message (should flush)
	service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "Message 2"})
	assert.Equal(t, 3, db.GetStoredCount("batch_collection"), "Should flush at batch size")

	// Add 2 more (should not flush)
	service.AddToHistory(sessionID, llms.Message{Role: "assistant", Content: "Response 2"})
	service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "Message 3"})
	assert.Equal(t, 3, db.GetStoredCount("batch_collection"), "Should not flush before next batch")

	// Add 3rd message (should flush again)
	service.AddToHistory(sessionID, llms.Message{Role: "assistant", Content: "Response 3"})
	assert.Equal(t, 6, db.GetStoredCount("batch_collection"), "Should flush second batch")
}

func TestMemoryService_LongTermMemory_FlushOnClear(t *testing.T) {
	workingStrategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 5})
	require.NoError(t, err)

	db := NewMockDatabaseProvider()
	embedder := NewMockEmbedderProvider()
	longTermStrategy, err := NewVectorMemoryStrategy(db, embedder, "flush_collection")
	require.NoError(t, err)

	service := NewMemoryService(
		workingStrategy,
		longTermStrategy,
		LongTermConfig{
			Enabled:      true,
			BatchSize:    5, // Large batch
			StorageScope: StorageScopeAll,
			AutoRecall:   false,
			RecallLimit:  5,
			Collection:   "flush_collection",
		},
	)

	sessionID := "flush-session"

	// Add messages (not enough to trigger flush)
	err = service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "Message 1"})
	require.NoError(t, err)
	err = service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "Message 2"})
	require.NoError(t, err)

	count := db.GetStoredCount("flush_collection")
	assert.Equal(t, 0, count, "Should not flush yet (got %d messages)", count)

	// Clear should flush pending batch
	err = service.ClearHistory(sessionID)
	require.NoError(t, err)

	count = db.GetStoredCount("flush_collection")
	assert.Equal(t, 2, count, "Should flush pending batch on clear (got %d messages)", count)
}

func TestMemoryService_LongTermMemory_AutoRecall(t *testing.T) {
	workingStrategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 3})
	require.NoError(t, err)

	db := NewMockDatabaseProvider()
	embedder := NewMockEmbedderProvider()
	longTermStrategy, err := NewVectorMemoryStrategy(db, embedder, "recall_collection")
	require.NoError(t, err)

	service := NewMemoryService(
		workingStrategy,
		longTermStrategy,
		LongTermConfig{
			Enabled:      true,
			BatchSize:    1,
			StorageScope: StorageScopeAll,
			AutoRecall:   true, // Enable auto-recall
			RecallLimit:  2,
			Collection:   "recall_collection",
		},
	)

	sessionID := "recall-session"

	// Add messages
	service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "What is Go programming language?"})
	service.AddToHistory(sessionID, llms.Message{Role: "assistant", Content: "Go is a statically typed compiled language."})
	service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "Tell me more about Go"})

	// Get history (should include auto-recalled messages)
	history, err := service.GetRecentHistory(sessionID)
	require.NoError(t, err)

	// Should have working memory (3) + potentially recalled messages
	assert.GreaterOrEqual(t, len(history), 3, "Should have at least working memory messages")
}

func TestMemoryService_LongTermMemory_StorageScope_Conversational(t *testing.T) {
	workingStrategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 5})
	require.NoError(t, err)

	db := NewMockDatabaseProvider()
	embedder := NewMockEmbedderProvider()
	longTermStrategy, err := NewVectorMemoryStrategy(db, embedder, "scope_collection")
	require.NoError(t, err)

	service := NewMemoryService(
		workingStrategy,
		longTermStrategy,
		LongTermConfig{
			Enabled:      true,
			BatchSize:    1,
			StorageScope: StorageScopeConversational, // Only user/assistant
			AutoRecall:   false,
			RecallLimit:  5,
			Collection:   "scope_collection",
		},
	)

	sessionID := "scope-session"

	// Add different types of messages
	service.AddToHistory(sessionID, llms.Message{Role: "system", Content: "System message"})
	service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "User message"})
	service.AddToHistory(sessionID, llms.Message{Role: "assistant", Content: "Assistant message"})
	service.AddToHistory(sessionID, llms.Message{Role: "tool", Content: "Tool output"})

	// Should only store user and assistant messages (2 messages)
	assert.Equal(t, 2, db.GetStoredCount("scope_collection"), "Should only store conversational messages")
}

func TestMemoryService_LongTermMemory_SessionIsolation(t *testing.T) {
	workingStrategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 5})
	require.NoError(t, err)

	db := NewMockDatabaseProvider()
	embedder := NewMockEmbedderProvider()
	longTermStrategy, err := NewVectorMemoryStrategy(db, embedder, "isolation_collection")
	require.NoError(t, err)

	service := NewMemoryService(
		workingStrategy,
		longTermStrategy,
		LongTermConfig{
			Enabled:      true,
			BatchSize:    1,
			StorageScope: StorageScopeAll,
			AutoRecall:   false,
			RecallLimit:  5,
			Collection:   "isolation_collection",
		},
	)

	// Add to different sessions
	service.AddToHistory("session1", llms.Message{Role: "user", Content: "Session 1 message"})
	service.AddToHistory("session2", llms.Message{Role: "user", Content: "Session 2 message"})

	// Verify both sessions have their messages in long-term memory
	recalled1, _ := longTermStrategy.Recall("session1", "message", 10)
	recalled2, _ := longTermStrategy.Recall("session2", "message", 10)

	assert.NotEmpty(t, recalled1, "Session 1 should have messages")
	assert.NotEmpty(t, recalled2, "Session 2 should have messages")

	// Clear session1's long-term memory directly (not via ClearHistory)
	err = longTermStrategy.Clear("session1")
	require.NoError(t, err)

	// Verify session1 is cleared but session2 is intact
	recalled1, _ = longTermStrategy.Recall("session1", "message", 10)
	recalled2, _ = longTermStrategy.Recall("session2", "message", 10)

	assert.Empty(t, recalled1, "Session 1 should be cleared after explicit Clear")
	assert.NotEmpty(t, recalled2, "Session 2 should still have messages")
}

func TestMemoryService_LongTermMemory_Disabled(t *testing.T) {
	workingStrategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 5})
	require.NoError(t, err)

	// Create service without long-term memory
	service := NewMemoryService(
		workingStrategy,
		nil, // No long-term strategy
		LongTermConfig{
			Enabled: false,
		},
	)

	sessionID := "no-longterm"

	// Add messages (should not crash)
	err = service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "Test"})
	require.NoError(t, err)

	// Get history (should work normally)
	history, err := service.GetRecentHistory(sessionID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(history))
}

func TestMemoryService_LongTermMemory_EmptyContent(t *testing.T) {
	workingStrategy, err := NewBufferWindowStrategy(BufferWindowConfig{WindowSize: 5})
	require.NoError(t, err)

	db := NewMockDatabaseProvider()
	embedder := NewMockEmbedderProvider()
	longTermStrategy, err := NewVectorMemoryStrategy(db, embedder, "empty_collection")
	require.NoError(t, err)

	service := NewMemoryService(
		workingStrategy,
		longTermStrategy,
		LongTermConfig{
			Enabled:      true,
			BatchSize:    1,
			StorageScope: StorageScopeAll,
			AutoRecall:   false,
			RecallLimit:  5,
			Collection:   "empty_collection",
		},
	)

	sessionID := "empty-session"

	// Add messages with empty content
	service.AddToHistory(sessionID, llms.Message{Role: "user", Content: ""})
	service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "Valid message"})

	// Should only store non-empty message
	assert.Equal(t, 1, db.GetStoredCount("empty_collection"), "Should skip empty messages")
}
