package agent

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecutionStateSerialization tests serialization and deserialization of execution state
func TestExecutionStateSerialization(t *testing.T) {
	execState := &ExecutionState{
		TaskID:    "test-task-123",
		ContextID: "test-context-456",
		Query:     "What is the weather?",
		ReasoningState: &ReasoningStateSnapshot{
			Iteration:         2,
			TotalTokens:       1500,
			Query:             "What is the weather?",
			AgentName:         "test-agent",
			AssistantResponse: "I'll check the weather for you.",
			ShowThinking:      true,
		},
		PendingToolCall: &protocol.ToolCall{
			Name: "get_weather",
			Args: map[string]interface{}{
				"location": "San Francisco",
			},
		},
	}

	// Serialize
	data, err := SerializeExecutionState(execState)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Deserialize
	restored, err := DeserializeExecutionState(data)
	require.NoError(t, err)
	require.NotNil(t, restored)

	// Verify all fields
	assert.Equal(t, execState.TaskID, restored.TaskID)
	assert.Equal(t, execState.ContextID, restored.ContextID)
	assert.Equal(t, execState.Query, restored.Query)
	assert.NotNil(t, restored.ReasoningState)
	assert.Equal(t, execState.ReasoningState.Iteration, restored.ReasoningState.Iteration)
	assert.Equal(t, execState.ReasoningState.TotalTokens, restored.ReasoningState.TotalTokens)
	assert.Equal(t, execState.ReasoningState.AssistantResponse, restored.ReasoningState.AssistantResponse)
	assert.Equal(t, execState.ReasoningState.AgentName, restored.ReasoningState.AgentName)
	assert.NotNil(t, restored.PendingToolCall)
	assert.Equal(t, execState.PendingToolCall.Name, restored.PendingToolCall.Name)
}

// TestCaptureExecutionState tests capturing execution state from reasoning state
func TestCaptureExecutionState(t *testing.T) {
	// Create a mock reasoning state
	services := createMockServices()
	outputCh := make(chan *pb.Part, 10)
	defer close(outputCh)

	state, err := reasoning.Builder().
		WithQuery("test query").
		WithAgentName("test-agent").
		WithOutputChannel(outputCh).
		WithServices(services).
		WithContext(context.Background()).
		Build()
	require.NoError(t, err)

	state.NextIteration()
	state.AddTokens(100)
	state.AppendResponse("test response")

	pendingToolCall := &protocol.ToolCall{
		Name: "test_tool",
		Args: map[string]interface{}{"key": "value"},
	}

	execState := CaptureExecutionState(
		"task-123",
		"context-456",
		"test query",
		state,
		pendingToolCall,
	)

	require.NotNil(t, execState)
	assert.Equal(t, "task-123", execState.TaskID)
	assert.Equal(t, "context-456", execState.ContextID)
	assert.Equal(t, "test query", execState.Query)
	assert.NotNil(t, execState.ReasoningState)
	assert.Equal(t, 1, execState.ReasoningState.Iteration)
	assert.Equal(t, 100, execState.ReasoningState.TotalTokens)
	assert.Equal(t, "test response", execState.ReasoningState.AssistantResponse)
	assert.Equal(t, pendingToolCall.Name, execState.PendingToolCall.Name)
}

// TestSessionExecutionStateHelpers tests save/load/clear execution state
func TestSessionExecutionStateHelpers(t *testing.T) {
	agent := createTestAgentWithSessionService(t)

	ctx := context.Background()
	sessionID := "test-session-123"
	taskID := "test-task-456"

	execState := &ExecutionState{
		TaskID:    taskID,
		ContextID: sessionID,
		Query:     "test query",
		ReasoningState: &ReasoningStateSnapshot{
			Iteration: 1,
			Query:     "test query",
		},
	}

	// Test SaveExecutionStateToSession
	err := agent.SaveExecutionStateToSession(ctx, sessionID, taskID, execState)
	require.NoError(t, err)

	// Test LoadExecutionStateFromSession
	loaded, err := agent.LoadExecutionStateFromSession(ctx, sessionID, taskID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, taskID, loaded.TaskID)
	assert.Equal(t, sessionID, loaded.ContextID)
	assert.Equal(t, "test query", loaded.Query)

	// Test ClearExecutionStateFromSession
	err = agent.ClearExecutionStateFromSession(ctx, sessionID, taskID)
	require.NoError(t, err)

	// Verify it's cleared
	_, err = agent.LoadExecutionStateFromSession(ctx, sessionID, taskID)
	assert.Error(t, err)
}

// TestShouldUseAsyncHITL tests the mode detection logic
func TestShouldUseAsyncHITL(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.AgentConfig
		hasSession     bool
		expectedResult bool
	}{
		{
			name:           "no config, no session - blocking",
			config:         nil,
			hasSession:     false,
			expectedResult: false,
		},
		{
			name:           "no config, has session - async",
			config:         nil,
			hasSession:     true,
			expectedResult: true,
		},
		{
			name: "explicit async mode",
			config: &config.AgentConfig{
				Task: &config.TaskConfig{
					HITL: &config.HITLConfig{
						Mode: "async",
					},
				},
			},
			hasSession:     true,
			expectedResult: true,
		},
		{
			name: "explicit blocking mode",
			config: &config.AgentConfig{
				Task: &config.TaskConfig{
					HITL: &config.HITLConfig{
						Mode: "blocking",
					},
				},
			},
			hasSession:     true,
			expectedResult: false,
		},
		{
			name: "auto mode with session - async",
			config: &config.AgentConfig{
				Task: &config.TaskConfig{
					HITL: &config.HITLConfig{
						Mode: "auto",
					},
				},
			},
			hasSession:     true,
			expectedResult: true,
		},
		{
			name: "auto mode without session - blocking",
			config: &config.AgentConfig{
				Task: &config.TaskConfig{
					HITL: &config.HITLConfig{
						Mode: "auto",
					},
				},
			},
			hasSession:     false,
			expectedResult: false,
		},
		{
			name: "no HITL config, has session - async (auto-detect)",
			config: &config.AgentConfig{
				Task: &config.TaskConfig{},
			},
			hasSession:     true,
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sessionService reasoning.SessionService
			if tt.hasSession {
				// For async HITL tests, use SQL session service (persistent)
				// In-memory session service should NOT enable async HITL
				// Create a minimal SQLite in-memory database for testing
				db, err := sql.Open("sqlite3", ":memory:")
				if err != nil {
					t.Fatalf("Failed to create test database: %v", err)
				}
				defer db.Close()

				// Initialize tables
				_, err = db.Exec(`
					CREATE TABLE IF NOT EXISTS sessions (
						id VARCHAR(255) NOT NULL,
						agent_id VARCHAR(255) NOT NULL,
						metadata TEXT,
						created_at TIMESTAMP NOT NULL,
						updated_at TIMESTAMP NOT NULL,
						PRIMARY KEY (id, agent_id)
					);
					CREATE TABLE IF NOT EXISTS session_messages (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						session_id VARCHAR(255) NOT NULL,
						message_id VARCHAR(255) NOT NULL,
						context_id VARCHAR(255),
						task_id VARCHAR(255),
						role VARCHAR(50) NOT NULL,
						message_json TEXT NOT NULL,
						sequence_num INTEGER NOT NULL,
						created_at TIMESTAMP NOT NULL
					);
				`)
				if err != nil {
					t.Fatalf("Failed to initialize test database: %v", err)
				}

				sessionService, err = memory.NewSQLSessionService(db, "sqlite", "test-agent")
				if err != nil {
					t.Fatalf("Failed to create SQL session service: %v", err)
				}
			}

			agent := &Agent{
				config: tt.config,
				services: &mockAgentServices{
					sessionService: sessionService,
				},
			}

			result := agent.shouldUseAsyncHITL()
			assert.Equal(t, tt.expectedResult, result, "shouldUseAsyncHITL() = %v, want %v", result, tt.expectedResult)
		})
	}
}

// TestErrInputRequired tests that ErrInputRequired is properly defined
func TestErrInputRequired(t *testing.T) {
	assert.NotNil(t, ErrInputRequired)
	assert.Equal(t, "input required - task paused for user approval", ErrInputRequired.Error())
}

// TestMultiplePendingExecutions tests handling multiple pending executions in same session
func TestMultiplePendingExecutions(t *testing.T) {
	agent := createTestAgentWithSessionService(t)

	ctx := context.Background()
	sessionID := "test-session-123"

	// Save multiple execution states
	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("test-task-%d", i)
		execState := &ExecutionState{
			TaskID:    taskID,
			ContextID: sessionID,
			Query:     fmt.Sprintf("test query %d", i),
		}
		err := agent.SaveExecutionStateToSession(ctx, sessionID, taskID, execState)
		require.NoError(t, err)
	}

	// Load each one
	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("test-task-%d", i)
		loaded, err := agent.LoadExecutionStateFromSession(ctx, sessionID, taskID)
		require.NoError(t, err)
		assert.Equal(t, taskID, loaded.TaskID)
	}

	// Clear one
	err := agent.ClearExecutionStateFromSession(ctx, sessionID, "test-task-0")
	require.NoError(t, err)

	// Verify it's cleared but others remain
	_, err = agent.LoadExecutionStateFromSession(ctx, sessionID, "test-task-0")
	assert.Error(t, err)

	loaded, err := agent.LoadExecutionStateFromSession(ctx, sessionID, "test-task-1")
	require.NoError(t, err)
	assert.Equal(t, "test-task-1", loaded.TaskID)
}

// TestExecutionStateWithComplexData tests serialization with complex nested data
func TestExecutionStateWithComplexData(t *testing.T) {
	execState := &ExecutionState{
		TaskID:    "task-123",
		ContextID: "context-456",
		Query:     "complex query",
		ReasoningState: &ReasoningStateSnapshot{
			Iteration: 5,
			// Note: History and CurrentTurn contain protobuf messages which
			// can't be directly JSON serialized in tests, but work fine in runtime
			// due to protobuf's JSON marshaling support
		},
		PendingToolCall: &protocol.ToolCall{
			Name: "complex_tool",
			Args: map[string]interface{}{
				"nested": map[string]interface{}{
					"key":    "value",
					"number": 42,
				},
				"array": []interface{}{1, 2, 3},
			},
		},
	}

	// Serialize and deserialize
	data, err := SerializeExecutionState(execState)
	require.NoError(t, err)

	restored, err := DeserializeExecutionState(data)
	require.NoError(t, err)

	// Verify complex nested data
	assert.Equal(t, execState.PendingToolCall.Name, restored.PendingToolCall.Name)
	nested := restored.PendingToolCall.Args["nested"].(map[string]interface{})
	assert.Equal(t, "value", nested["key"])
	assert.Equal(t, float64(42), nested["number"]) // JSON numbers become float64
	array := restored.PendingToolCall.Args["array"].([]interface{})
	assert.Len(t, array, 3)
	assert.Equal(t, float64(1), array[0])
}

// Helper functions

func createTestAgentWithSessionService(t *testing.T) *Agent {
	sessionService := memory.NewInMemorySessionService()
	services := &mockAgentServices{
		sessionService: sessionService,
	}

	return &Agent{
		id:       "test-agent",
		name:     "Test Agent",
		services: services,
		config:   nil,
	}
}

func createMockServices() reasoning.AgentServices {
	return &mockAgentServices{
		sessionService: memory.NewInMemorySessionService(),
	}
}

type mockAgentServices struct {
	sessionService reasoning.SessionService
}

func (m *mockAgentServices) GetConfig() config.ReasoningConfig {
	return config.ReasoningConfig{}
}

func (m *mockAgentServices) LLM() reasoning.LLMService {
	return nil
}

func (m *mockAgentServices) Tools() reasoning.ToolService {
	return nil
}

func (m *mockAgentServices) Context() reasoning.ContextService {
	return nil
}

func (m *mockAgentServices) Prompt() reasoning.PromptService {
	return nil
}

func (m *mockAgentServices) Session() reasoning.SessionService {
	return m.sessionService
}

func (m *mockAgentServices) History() reasoning.HistoryService {
	return nil
}

func (m *mockAgentServices) Registry() reasoning.AgentRegistryService {
	return nil
}

func (m *mockAgentServices) Task() reasoning.TaskService {
	return nil
}
