package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

const (
	pendingExecutionsKey = "pending_executions"
)

// ErrInputRequired is returned when task needs to pause for user input (async HITL)
var ErrInputRequired = fmt.Errorf("input required - task paused for user approval")

// SaveExecutionStateToSession saves execution state to session metadata
func (a *Agent) SaveExecutionStateToSession(
	ctx context.Context,
	sessionID string,
	taskID string,
	execState *ExecutionState,
) error {
	sessionService := a.services.Session()
	if sessionService == nil {
		return fmt.Errorf("session service not available")
	}

	// Get current session metadata
	metadata, err := sessionService.GetOrCreateSessionMetadata(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session metadata: %w", err)
	}

	// Initialize pending_executions if needed
	if metadata.Metadata == nil {
		metadata.Metadata = make(map[string]interface{})
	}

	pendingExecutions, exists := metadata.Metadata[pendingExecutionsKey]
	if !exists {
		pendingExecutions = make(map[string]interface{})
		metadata.Metadata[pendingExecutionsKey] = pendingExecutions
	}

	// Convert to map for manipulation
	pendingMap, ok := pendingExecutions.(map[string]interface{})
	if !ok {
		pendingMap = make(map[string]interface{})
		metadata.Metadata[pendingExecutionsKey] = pendingMap
	}

	// Serialize execution state
	stateJSON, err := SerializeExecutionState(execState)
	if err != nil {
		return fmt.Errorf("failed to serialize execution state: %w", err)
	}

	// Store as JSON map (metadata values must be JSON-serializable)
	var stateMap map[string]interface{}
	if err := json.Unmarshal(stateJSON, &stateMap); err != nil {
		return fmt.Errorf("failed to unmarshal execution state: %w", err)
	}

	pendingMap[taskID] = stateMap

	// Update session metadata
	if err := sessionService.UpdateSessionMetadata(sessionID, metadata.Metadata); err != nil {
		return fmt.Errorf("failed to update session metadata: %w", err)
	}

	// Log checkpoint info (handle backward compatibility - old checkpoints may not have phase/type)
	if execState.Phase != "" || execState.CheckpointType != "" {
		log.Printf("[Agent:%s] [Checkpoint] Saved execution state for task %s in session %s (phase: %s, type: %s)",
			a.id, taskID, sessionID, execState.Phase, execState.CheckpointType)
	} else {
		log.Printf("[Agent:%s] [HITL] Saved execution state for task %s in session %s", a.id, taskID, sessionID)
	}
	return nil
}

// LoadExecutionStateFromSession loads execution state from session metadata
func (a *Agent) LoadExecutionStateFromSession(
	ctx context.Context,
	sessionID string,
	taskID string,
) (*ExecutionState, error) {
	sessionService := a.services.Session()
	if sessionService == nil {
		return nil, fmt.Errorf("session service not available")
	}

	metadata, err := sessionService.GetOrCreateSessionMetadata(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session metadata: %w", err)
	}

	if metadata.Metadata == nil {
		return nil, fmt.Errorf("no execution state found for task %s", taskID)
	}

	pendingExecutions, exists := metadata.Metadata[pendingExecutionsKey]
	if !exists {
		return nil, fmt.Errorf("no pending executions in session")
	}

	pendingMap, ok := pendingExecutions.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid pending_executions format")
	}

	taskState, exists := pendingMap[taskID]
	if !exists {
		return nil, fmt.Errorf("execution state not found for task %s", taskID)
	}

	// Serialize back to JSON for deserialization
	stateJSON, err := json.Marshal(taskState)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task state: %w", err)
	}

	return DeserializeExecutionState(stateJSON)
}

// ClearExecutionStateFromSession removes execution state after resuming
func (a *Agent) ClearExecutionStateFromSession(
	ctx context.Context,
	sessionID string,
	taskID string,
) error {
	sessionService := a.services.Session()
	if sessionService == nil {
		return fmt.Errorf("session service not available")
	}

	metadata, err := sessionService.GetOrCreateSessionMetadata(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session metadata: %w", err)
	}

	if metadata.Metadata == nil {
		return nil // Nothing to clear
	}

	pendingExecutions, exists := metadata.Metadata[pendingExecutionsKey]
	if !exists {
		return nil // Nothing to clear
	}

	pendingMap, ok := pendingExecutions.(map[string]interface{})
	if !ok {
		return nil
	}

	delete(pendingMap, taskID)

	// If no more pending executions, remove the key
	if len(pendingMap) == 0 {
		delete(metadata.Metadata, pendingExecutionsKey)
	}

	// Update session metadata
	if err := sessionService.UpdateSessionMetadata(sessionID, metadata.Metadata); err != nil {
		return fmt.Errorf("failed to update session metadata: %w", err)
	}

	log.Printf("[Agent:%s] [HITL] Cleared execution state for task %s from session %s", a.id, taskID, sessionID)
	return nil
}

// shouldUseAsyncHITL determines if async HITL should be used based on configuration
func (a *Agent) shouldUseAsyncHITL() bool {
	if a.config == nil {
		// Programmatic API: auto-detect based on session service
		return a.services.Session() != nil
	}

	taskCfg := a.config.Task
	mode := "auto" // Default
	if taskCfg != nil && taskCfg.HITL != nil {
		mode = taskCfg.HITL.Mode
		if mode == "" {
			mode = "auto"
		}
	}

	hasSessionStore := a.services.Session() != nil

	switch mode {
	case "async":
		return true // Explicit async
	case "blocking":
		return false // Explicit blocking
	case "auto":
		return hasSessionStore // Auto-detect
	default:
		return hasSessionStore // Fallback to auto-detect
	}
}
