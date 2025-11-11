package agent

import (
	"context"
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateSessionMetadata tests the UpdateSessionMetadata implementation
func TestUpdateSessionMetadata(t *testing.T) {
	sessionService := memory.NewInMemorySessionService()
	sessionID := "test-session-123"

	// Create session metadata
	metadata, err := sessionService.GetOrCreateSessionMetadata(sessionID)
	require.NoError(t, err)
	assert.NotNil(t, metadata)

	// Update metadata
	newMetadata := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": map[string]interface{}{
			"nested": "value",
		},
	}

	err = sessionService.UpdateSessionMetadata(sessionID, newMetadata)
	require.NoError(t, err)

	// Verify update
	updated, err := sessionService.GetOrCreateSessionMetadata(sessionID)
	require.NoError(t, err)
	assert.Equal(t, "value1", updated.Metadata["key1"])
	assert.Equal(t, 42, updated.Metadata["key2"])
}

// TestHITLConfigValidation tests HITL configuration validation
func TestHITLConfigValidation(t *testing.T) {
	tests := []struct {
		name          string
		hitlConfig    *config.HITLConfig
		expectedError bool
		errorContains string
	}{
		{
			name:          "valid auto mode",
			hitlConfig:    &config.HITLConfig{Mode: "auto"},
			expectedError: false,
		},
		{
			name:          "valid blocking mode",
			hitlConfig:    &config.HITLConfig{Mode: "blocking"},
			expectedError: false,
		},
		{
			name:          "valid async mode",
			hitlConfig:    &config.HITLConfig{Mode: "async"},
			expectedError: false,
		},
		{
			name:          "empty mode (defaults to auto)",
			hitlConfig:    &config.HITLConfig{Mode: ""},
			expectedError: false,
		},
		{
			name:          "invalid mode",
			hitlConfig:    &config.HITLConfig{Mode: "invalid"},
			expectedError: true,
			errorContains: "invalid hitl.mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.hitlConfig.Validate()
			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestExecutionStateRoundTrip tests complete round-trip: save -> load -> clear
func TestExecutionStateRoundTrip(t *testing.T) {
	agent := createTestAgentWithSessionService(t)
	ctx := context.Background()
	sessionID := "roundtrip-session"
	taskID := "roundtrip-task"

	// Create execution state
	execState := &ExecutionState{
		TaskID:    taskID,
		ContextID: sessionID,
		Query:     "roundtrip test query",
		ReasoningState: &ReasoningStateSnapshot{
			Iteration:         3,
			TotalTokens:       2500,
			Query:             "roundtrip test query",
			AgentName:         "test-agent",
			AssistantResponse: "test response",
		},
	}

	// Save
	err := agent.SaveExecutionStateToSession(ctx, sessionID, taskID, execState)
	require.NoError(t, err)

	// Load
	loaded, err := agent.LoadExecutionStateFromSession(ctx, sessionID, taskID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Verify all fields match
	assert.Equal(t, execState.TaskID, loaded.TaskID)
	assert.Equal(t, execState.ContextID, loaded.ContextID)
	assert.Equal(t, execState.Query, loaded.Query)
	assert.NotNil(t, loaded.ReasoningState)
	assert.Equal(t, execState.ReasoningState.Iteration, loaded.ReasoningState.Iteration)
	assert.Equal(t, execState.ReasoningState.TotalTokens, loaded.ReasoningState.TotalTokens)
	assert.Equal(t, execState.ReasoningState.AssistantResponse, loaded.ReasoningState.AssistantResponse)
	assert.Equal(t, execState.ReasoningState.AgentName, loaded.ReasoningState.AgentName)

	// Clear
	err = agent.ClearExecutionStateFromSession(ctx, sessionID, taskID)
	require.NoError(t, err)

	// Verify cleared
	_, err = agent.LoadExecutionStateFromSession(ctx, sessionID, taskID)
	assert.Error(t, err)
}

// TestLoadNonExistentExecutionState tests loading non-existent execution state
func TestLoadNonExistentExecutionState(t *testing.T) {
	agent := createTestAgentWithSessionService(t)
	ctx := context.Background()

	// Try to load non-existent state
	_, err := agent.LoadExecutionStateFromSession(ctx, "non-existent-session", "non-existent-task")
	assert.Error(t, err)
	// Error message indicates state doesn't exist (either "not found" or "no pending executions")
	assert.True(t,
		contains(err.Error(), "not found") || contains(err.Error(), "no pending executions"),
		"error should indicate state doesn't exist, got: %s", err.Error())
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestClearNonExistentExecutionState tests clearing non-existent execution state (should not error)
func TestClearNonExistentExecutionState(t *testing.T) {
	agent := createTestAgentWithSessionService(t)
	ctx := context.Background()

	// Clear non-existent state (should not error)
	err := agent.ClearExecutionStateFromSession(ctx, "non-existent-session", "non-existent-task")
	assert.NoError(t, err) // Clearing non-existent state is idempotent
}

// TestSaveLoadMultipleSessions tests saving/loading execution states across multiple sessions
func TestSaveLoadMultipleSessions(t *testing.T) {
	agent := createTestAgentWithSessionService(t)
	ctx := context.Background()

	// Save states in different sessions
	sessions := []string{"session-1", "session-2", "session-3"}
	for i, sessionID := range sessions {
		taskID := "task-" + sessionID
		execState := &ExecutionState{
			TaskID:    taskID,
			ContextID: sessionID,
			Query:     "query for " + sessionID,
			ReasoningState: &ReasoningStateSnapshot{
				Iteration: i + 1,
			},
		}
		err := agent.SaveExecutionStateToSession(ctx, sessionID, taskID, execState)
		require.NoError(t, err)
	}

	// Load each one
	for i, sessionID := range sessions {
		taskID := "task-" + sessionID
		loaded, err := agent.LoadExecutionStateFromSession(ctx, sessionID, taskID)
		require.NoError(t, err)
		assert.Equal(t, taskID, loaded.TaskID)
		assert.Equal(t, sessionID, loaded.ContextID)
		assert.Equal(t, i+1, loaded.ReasoningState.Iteration)
	}
}

// TestExecutionStateEmptyFields tests handling of empty/zero values
func TestExecutionStateEmptyFields(t *testing.T) {
	execState := &ExecutionState{
		TaskID:    "",
		ContextID: "",
		Query:     "",
		ReasoningState: &ReasoningStateSnapshot{
			Iteration: 0,
		},
		PendingToolCall: nil,
	}

	// Should still serialize/deserialize
	data, err := SerializeExecutionState(execState)
	require.NoError(t, err)

	restored, err := DeserializeExecutionState(data)
	require.NoError(t, err)

	assert.Equal(t, "", restored.TaskID)
	assert.Equal(t, "", restored.ContextID)
	assert.NotNil(t, restored.ReasoningState)
	assert.Equal(t, 0, restored.ReasoningState.Iteration)
	assert.Nil(t, restored.PendingToolCall)
}

// TestShouldUseAsyncHITLWithNilServices tests edge case with nil services
func TestShouldUseAsyncHITLWithNilServices(t *testing.T) {
	agent := &Agent{
		config: nil,
		services: &mockAgentServices{
			sessionService: nil, // No session service
		},
	}

	result := agent.shouldUseAsyncHITL()
	assert.False(t, result, "should return false when no session service")
}

// TestExecutionStateNilReasoningState tests handling nil reasoning state snapshot
func TestExecutionStateNilReasoningState(t *testing.T) {
	execState := &ExecutionState{
		TaskID:         "test-task",
		ContextID:      "test-context",
		Query:          "test query",
		ReasoningState: nil, // Nil reasoning state
	}

	// Should serialize (nil is valid)
	data, err := SerializeExecutionState(execState)
	require.NoError(t, err)

	// Should deserialize
	restored, err := DeserializeExecutionState(data)
	require.NoError(t, err)
	assert.Nil(t, restored.ReasoningState)
}
