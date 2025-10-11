package memory

import (
	"context"
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// VECTOR MEMORY STRATEGY UNIT TESTS
// ============================================================================

func TestVectorMemoryStrategy_NewVectorMemoryStrategy(t *testing.T) {
	db := NewMockDatabaseProvider()
	embedder := NewMockEmbedderProvider()

	t.Run("succeeds with valid inputs", func(t *testing.T) {
		strategy, err := NewVectorMemoryStrategy(db, embedder, "test_collection")
		require.NoError(t, err)
		assert.NotNil(t, strategy)
		assert.Equal(t, "vector_memory", strategy.Name())
	})

	t.Run("uses default collection name", func(t *testing.T) {
		strategy, err := NewVectorMemoryStrategy(db, embedder, "")
		require.NoError(t, err)
		assert.NotNil(t, strategy)
	})

	t.Run("fails without database", func(t *testing.T) {
		strategy, err := NewVectorMemoryStrategy(nil, embedder, "test")
		assert.Error(t, err)
		assert.Nil(t, strategy)
		assert.Contains(t, err.Error(), "database provider is required")
	})

	t.Run("fails without embedder", func(t *testing.T) {
		strategy, err := NewVectorMemoryStrategy(db, nil, "test")
		assert.Error(t, err)
		assert.Nil(t, strategy)
		assert.Contains(t, err.Error(), "embedder provider is required")
	})
}

func TestVectorMemoryStrategy_Store(t *testing.T) {
	db := NewMockDatabaseProvider()
	embedder := NewMockEmbedderProvider()
	strategy, err := NewVectorMemoryStrategy(db, embedder, "test_collection")
	require.NoError(t, err)

	t.Run("stores single message", func(t *testing.T) {
		messages := []a2a.Message{
			a2a.CreateUserMessage("Hello world"),
		}

		err := strategy.Store("session1", messages)
		assert.NoError(t, err)
		assert.Equal(t, 1, db.GetStoredCount("test_collection"))
	})

	t.Run("stores multiple messages", func(t *testing.T) {
		db := NewMockDatabaseProvider()
		embedder := NewMockEmbedderProvider()
		strategy, _ := NewVectorMemoryStrategy(db, embedder, "multi_test")

		messages := []a2a.Message{
			a2a.CreateUserMessage("First message"),
			a2a.CreateAssistantMessage("Second message"),
			a2a.CreateUserMessage("Third message"),
		}

		err := strategy.Store("session1", messages)
		assert.NoError(t, err)
		assert.Equal(t, 3, db.GetStoredCount("multi_test"))
	})

	t.Run("skips empty content", func(t *testing.T) {
		db := NewMockDatabaseProvider()
		embedder := NewMockEmbedderProvider()
		strategy, _ := NewVectorMemoryStrategy(db, embedder, "empty_test")

		messages := []a2a.Message{
			a2a.CreateUserMessage("Valid message"),
			a2a.CreateAssistantMessage(""),
			a2a.CreateUserMessage("Another valid message"),
		}

		err := strategy.Store("session1", messages)
		assert.NoError(t, err)
		assert.Equal(t, 2, db.GetStoredCount("empty_test"))
	})

	t.Run("handles empty message slice", func(t *testing.T) {
		db := NewMockDatabaseProvider()
		embedder := NewMockEmbedderProvider()
		strategy, _ := NewVectorMemoryStrategy(db, embedder, "empty_slice_test")

		err := strategy.Store("session1", []a2a.Message{})
		assert.NoError(t, err)
		assert.Equal(t, 0, db.GetStoredCount("empty_slice_test"))
	})

	t.Run("stores with correct metadata", func(t *testing.T) {
		db := NewMockDatabaseProvider()
		embedder := NewMockEmbedderProvider()
		strategy, _ := NewVectorMemoryStrategy(db, embedder, "metadata_test")

		messages := []a2a.Message{
			a2a.CreateUserMessage("Test message"),
		}

		err := strategy.Store("session123", messages)
		assert.NoError(t, err)

		// Verify metadata
		results, _ := db.Search(context.Background(), "metadata_test", []float32{}, 10)
		require.Len(t, results, 1)
		assert.Equal(t, "session123", results[0].Metadata["session_id"])
		assert.Equal(t, "user", results[0].Metadata["role"])
		assert.Equal(t, "Test message", results[0].Metadata["content"])
	})

	t.Run("handles embedder error", func(t *testing.T) {
		db := NewMockDatabaseProvider()
		embedder := NewMockEmbedderProvider()
		embedder.SetEmbedFunc(func(text string) ([]float32, error) {
			return nil, assert.AnError
		})
		strategy, _ := NewVectorMemoryStrategy(db, embedder, "error_test")

		messages := []a2a.Message{
			a2a.CreateUserMessage("Test"),
		}

		err := strategy.Store("session1", messages)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to embed")
	})
}

func TestVectorMemoryStrategy_Recall(t *testing.T) {
	db := NewMockDatabaseProvider()
	embedder := NewMockEmbedderProvider()
	strategy, _ := NewVectorMemoryStrategy(db, embedder, "recall_test")

	// Store some messages first
	messages := []a2a.Message{
		a2a.CreateUserMessage("I love programming"),
		a2a.CreateAssistantMessage("That's great! What languages?"),
		a2a.CreateUserMessage("Go and Python"),
	}
	strategy.Store("session1", messages)

	t.Run("recalls messages", func(t *testing.T) {
		recalled, err := strategy.Recall("session1", "programming languages", 10)
		assert.NoError(t, err)
		assert.NotEmpty(t, recalled)
	})

	t.Run("returns empty for empty query", func(t *testing.T) {
		recalled, err := strategy.Recall("session1", "", 10)
		assert.NoError(t, err)
		assert.Empty(t, recalled)
	})

	t.Run("filters by session ID", func(t *testing.T) {
		// Store messages for different sessions
		strategy.Store("session1", []a2a.Message{a2a.CreateUserMessage("Session 1 message")})
		strategy.Store("session2", []a2a.Message{a2a.CreateUserMessage("Session 2 message")})

		// Recall for session1 should only return session1 messages
		recalled, err := strategy.Recall("session1", "message", 10)
		assert.NoError(t, err)

		// All recalled messages should be from session1
		for _, msg := range recalled {
			// Messages from session1 should not contain "Session 2"
			textContent := a2a.ExtractTextFromMessage(msg)
			assert.NotContains(t, textContent, "Session 2")
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		db := NewMockDatabaseProvider()
		embedder := NewMockEmbedderProvider()
		strategy, _ := NewVectorMemoryStrategy(db, embedder, "limit_test")

		// Store many messages
		messages := []a2a.Message{
			a2a.CreateUserMessage("Message 1"),
			a2a.CreateUserMessage("Message 2"),
			a2a.CreateUserMessage("Message 3"),
			a2a.CreateUserMessage("Message 4"),
			a2a.CreateUserMessage("Message 5"),
		}
		strategy.Store("session1", messages)

		// Recall with limit
		recalled, err := strategy.Recall("session1", "message", 3)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(recalled), 3)
	})

	t.Run("extracts role correctly", func(t *testing.T) {
		db := NewMockDatabaseProvider()
		embedder := NewMockEmbedderProvider()
		strategy, _ := NewVectorMemoryStrategy(db, embedder, "role_test")

		messages := []a2a.Message{
			a2a.CreateUserMessage("User message"),
			a2a.CreateAssistantMessage("Assistant message"),
		}
		strategy.Store("session1", messages)

		recalled, err := strategy.Recall("session1", "message", 10)
		assert.NoError(t, err)
		require.NotEmpty(t, recalled)

		// Check that roles are preserved
		hasUser := false
		hasAssistant := false
		for _, msg := range recalled {
			if msg.Role == "user" {
				hasUser = true
			}
			if msg.Role == "assistant" {
				hasAssistant = true
			}
		}
		assert.True(t, hasUser || hasAssistant, "Should have at least one role")
	})
}

func TestVectorMemoryStrategy_Clear(t *testing.T) {
	db := NewMockDatabaseProvider()
	embedder := NewMockEmbedderProvider()
	strategy, _ := NewVectorMemoryStrategy(db, embedder, "clear_test")

	// Store messages for multiple sessions
	strategy.Store("session1", []a2a.Message{a2a.CreateUserMessage("Session 1")})
	strategy.Store("session2", []a2a.Message{a2a.CreateUserMessage("Session 2")})

	t.Run("clears session messages", func(t *testing.T) {
		// Clear session1
		err := strategy.Clear("session1")
		assert.NoError(t, err)

		// Session1 should have no messages
		recalled1, _ := strategy.Recall("session1", "session", 10)
		assert.Empty(t, recalled1)

		// Session2 should still have messages
		recalled2, _ := strategy.Recall("session2", "session", 10)
		assert.NotEmpty(t, recalled2)
	})

	t.Run("handles non-existent session", func(t *testing.T) {
		err := strategy.Clear("non_existent_session")
		assert.NoError(t, err)
	})
}

func TestVectorMemoryStrategy_SessionIsolation(t *testing.T) {
	db := NewMockDatabaseProvider()
	embedder := NewMockEmbedderProvider()
	strategy, _ := NewVectorMemoryStrategy(db, embedder, "isolation_test")

	// Store messages for different sessions
	session1Messages := []a2a.Message{
		a2a.CreateUserMessage("I am Alice and I love hiking"),
	}
	session2Messages := []a2a.Message{
		a2a.CreateUserMessage("I am Bob and I love cooking"),
	}

	strategy.Store("session1", session1Messages)
	strategy.Store("session2", session2Messages)

	t.Run("session1 only sees its messages", func(t *testing.T) {
		recalled, err := strategy.Recall("session1", "hobbies", 10)
		assert.NoError(t, err)

		// Should not see Bob's message
		for _, msg := range recalled {
			textContent := a2a.ExtractTextFromMessage(msg)
			assert.NotContains(t, textContent, "Bob")
			assert.NotContains(t, textContent, "cooking")
		}
	})

	t.Run("session2 only sees its messages", func(t *testing.T) {
		recalled, err := strategy.Recall("session2", "hobbies", 10)
		assert.NoError(t, err)

		// Should not see Alice's message
		for _, msg := range recalled {
			textContent := a2a.ExtractTextFromMessage(msg)
			assert.NotContains(t, textContent, "Alice")
			assert.NotContains(t, textContent, "hiking")
		}
	})

	t.Run("clearing one session doesn't affect another", func(t *testing.T) {
		err := strategy.Clear("session1")
		assert.NoError(t, err)

		// Session1 should be empty
		recalled1, _ := strategy.Recall("session1", "hobbies", 10)
		assert.Empty(t, recalled1)

		// Session2 should still have messages
		recalled2, _ := strategy.Recall("session2", "hobbies", 10)
		assert.NotEmpty(t, recalled2)
	})
}

func TestVectorMemoryStrategy_BatchStorage(t *testing.T) {
	db := NewMockDatabaseProvider()
	embedder := NewMockEmbedderProvider()
	strategy, _ := NewVectorMemoryStrategy(db, embedder, "batch_test")

	t.Run("stores messages in batch", func(t *testing.T) {
		messages := []a2a.Message{
			a2a.CreateUserMessage("Message 1"),
			a2a.CreateAssistantMessage("Message 2"),
			a2a.CreateUserMessage("Message 3"),
			a2a.CreateAssistantMessage("Message 4"),
			a2a.CreateUserMessage("Message 5"),
		}

		err := strategy.Store("session1", messages)
		assert.NoError(t, err)
		assert.Equal(t, 5, db.GetStoredCount("batch_test"))

		// All messages should be recallable
		recalled, err := strategy.Recall("session1", "message", 10)
		assert.NoError(t, err)
		assert.Len(t, recalled, 5)
	})
}

func TestVectorMemoryStrategy_ContentPreservation(t *testing.T) {
	t.Run("preserves exact content", func(t *testing.T) {
		db := NewMockDatabaseProvider()
		embedder := NewMockEmbedderProvider()
		strategy, _ := NewVectorMemoryStrategy(db, embedder, "content_test")

		originalContent := "This is a very specific message with unique content"
		messages := []a2a.Message{
			a2a.CreateUserMessage(originalContent),
		}

		strategy.Store("session1", messages)

		recalled, err := strategy.Recall("session1", "specific", 10)
		assert.NoError(t, err)
		require.Len(t, recalled, 1)
		textContent := a2a.ExtractTextFromMessage(recalled[0])
		assert.Equal(t, originalContent, textContent)
	})

	t.Run("preserves special characters", func(t *testing.T) {
		db := NewMockDatabaseProvider()
		embedder := NewMockEmbedderProvider()
		strategy, _ := NewVectorMemoryStrategy(db, embedder, "special_test")

		specialContent := "Special: @#$%^&*()_+-={}[]|\\:\";<>?,./"
		messages := []a2a.Message{
			a2a.CreateUserMessage(specialContent),
		}

		strategy.Store("session1", messages)

		recalled, err := strategy.Recall("session1", "special", 10)
		assert.NoError(t, err)
		require.Len(t, recalled, 1)
		textContent := a2a.ExtractTextFromMessage(recalled[0])
		assert.Equal(t, specialContent, textContent)
	})
}
