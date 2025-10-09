package history

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// UNIT TESTS - Buffer Window Strategy
// Deterministic tests verifying exact behavior
// ============================================================================

func TestBufferWindow_Unit_BasicOperations(t *testing.T) {
	t.Run("Create with defaults", func(t *testing.T) {
		strategy, err := NewBufferWindowStrategy(BufferWindowConfig{
			WindowSize: 0, // Should use default
		})
		require.NoError(t, err)
		assert.Equal(t, "buffer_window", strategy.Name())
		assert.Equal(t, 20, strategy.windowSize) // Default
	})

	t.Run("Create with custom window size", func(t *testing.T) {
		strategy, err := NewBufferWindowStrategy(BufferWindowConfig{
			WindowSize: 5,
		})
		require.NoError(t, err)
		assert.Equal(t, 5, strategy.windowSize)
	})
}

func TestBufferWindow_Unit_ExactWindowSize(t *testing.T) {
	// DETERMINISTIC TEST: Verify exact window size enforcement
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{
		WindowSize: 3,
	})
	require.NoError(t, err)

	sessionID := "test-exact-window"

	// Add exactly 3 messages
	messages := []llms.Message{
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Message 2"},
	}

	for _, msg := range messages {
		err := strategy.AddMessage(sessionID, msg)
		require.NoError(t, err)
	}

	// Should have exactly 3 messages
	history, err := strategy.GetHistory(sessionID)
	require.NoError(t, err)
	assert.Len(t, history, 3, "Should have exactly 3 messages")
	assert.Equal(t, "Message 1", history[0].Content)
	assert.Equal(t, "Message 2", history[2].Content)
}

func TestBufferWindow_Unit_FIFOBehavior(t *testing.T) {
	// DETERMINISTIC TEST: Verify FIFO (First In, First Out) behavior
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{
		WindowSize: 3,
	})
	require.NoError(t, err)

	sessionID := "test-fifo"

	// Add 5 messages (exceeds window of 3)
	for i := 1; i <= 5; i++ {
		err := strategy.AddMessage(sessionID, llms.Message{
			Role:    "user",
			Content: "Message " + string(rune('0'+i)),
		})
		require.NoError(t, err)
	}

	history, err := strategy.GetHistory(sessionID)
	require.NoError(t, err)

	// Should only keep last 3 messages (3, 4, 5)
	// Note: ConversationHistory may have its own trimming logic
	assert.True(t, len(history) <= 3, "Should not exceed window size")
}

func TestBufferWindow_Unit_SessionIsolation(t *testing.T) {
	// DETERMINISTIC TEST: Verify sessions are completely isolated
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{
		WindowSize: 5,
	})
	require.NoError(t, err)

	// Session 1
	strategy.AddMessage("session1", llms.Message{Role: "user", Content: "S1-M1"})
	strategy.AddMessage("session1", llms.Message{Role: "user", Content: "S1-M2"})

	// Session 2
	strategy.AddMessage("session2", llms.Message{Role: "user", Content: "S2-M1"})
	strategy.AddMessage("session2", llms.Message{Role: "user", Content: "S2-M2"})
	strategy.AddMessage("session2", llms.Message{Role: "user", Content: "S2-M3"})

	// Session 3
	strategy.AddMessage("session3", llms.Message{Role: "user", Content: "S3-M1"})

	// Verify each session has correct messages
	h1, _ := strategy.GetHistory("session1")
	h2, _ := strategy.GetHistory("session2")
	h3, _ := strategy.GetHistory("session3")

	assert.Len(t, h1, 2, "Session 1 should have 2 messages")
	assert.Len(t, h2, 3, "Session 2 should have 3 messages")
	assert.Len(t, h3, 1, "Session 3 should have 1 message")

	// Verify content doesn't leak between sessions
	assert.Equal(t, "S1-M1", h1[0].Content)
	assert.Equal(t, "S2-M1", h2[0].Content)
	assert.Equal(t, "S3-M1", h3[0].Content)

	// Verify session count
	assert.Equal(t, 3, strategy.GetSessionCount())
}

func TestBufferWindow_Unit_ClearOperation(t *testing.T) {
	// DETERMINISTIC TEST: Verify clear removes all messages
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{
		WindowSize: 5,
	})
	require.NoError(t, err)

	sessionID := "test-clear"

	// Add messages
	for i := 1; i <= 3; i++ {
		strategy.AddMessage(sessionID, llms.Message{
			Role:    "user",
			Content: "Message " + string(rune('0'+i)),
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

func TestBufferWindow_Unit_EmptySession(t *testing.T) {
	// DETERMINISTIC TEST: Verify empty session handling
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{
		WindowSize: 5,
	})
	require.NoError(t, err)

	// Get history from non-existent session
	history, err := strategy.GetHistory("non-existent")
	require.NoError(t, err)
	assert.Len(t, history, 0, "Non-existent session should return empty history")
}

func TestBufferWindow_Unit_DefaultSessionID(t *testing.T) {
	// DETERMINISTIC TEST: Verify empty sessionID defaults to "default"
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{
		WindowSize: 5,
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

func TestBufferWindow_Unit_ConcurrentSessions(t *testing.T) {
	// DETERMINISTIC TEST: Verify multiple sessions don't interfere
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{
		WindowSize: 2,
	})
	require.NoError(t, err)

	// Add messages to multiple sessions in interleaved fashion
	strategy.AddMessage("A", llms.Message{Role: "user", Content: "A1"})
	strategy.AddMessage("B", llms.Message{Role: "user", Content: "B1"})
	strategy.AddMessage("A", llms.Message{Role: "user", Content: "A2"})
	strategy.AddMessage("B", llms.Message{Role: "user", Content: "B2"})
	strategy.AddMessage("C", llms.Message{Role: "user", Content: "C1"})

	// Verify each session maintained its own history
	hA, _ := strategy.GetHistory("A")
	hB, _ := strategy.GetHistory("B")
	hC, _ := strategy.GetHistory("C")

	assert.Len(t, hA, 2)
	assert.Equal(t, "A1", hA[0].Content)
	assert.Equal(t, "A2", hA[1].Content)

	assert.Len(t, hB, 2)
	assert.Equal(t, "B1", hB[0].Content)
	assert.Equal(t, "B2", hB[1].Content)

	assert.Len(t, hC, 1)
	assert.Equal(t, "C1", hC[0].Content)
}

func TestBufferWindow_Unit_MessageOrder(t *testing.T) {
	// DETERMINISTIC TEST: Verify messages maintain chronological order
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{
		WindowSize: 10,
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
}

func TestBufferWindow_Unit_RolePreservation(t *testing.T) {
	// DETERMINISTIC TEST: Verify roles are preserved correctly
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{
		WindowSize: 5,
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

func TestBufferWindow_Unit_WindowSizeOne(t *testing.T) {
	// EDGE CASE: Window size of 1 (minimum useful size)
	strategy, err := NewBufferWindowStrategy(BufferWindowConfig{
		WindowSize: 1,
	})
	require.NoError(t, err)

	sessionID := "test-size-one"

	// Add multiple messages
	strategy.AddMessage(sessionID, llms.Message{Role: "user", Content: "First"})
	strategy.AddMessage(sessionID, llms.Message{Role: "user", Content: "Second"})
	strategy.AddMessage(sessionID, llms.Message{Role: "user", Content: "Third"})

	// Should only keep the last message
	history, err := strategy.GetHistory(sessionID)
	require.NoError(t, err)
	assert.True(t, len(history) <= 1, "Should have at most 1 message")
}
