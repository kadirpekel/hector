package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/kadirpekel/hector/pkg/reasoning"
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
		log.Printf("[Agent:%s] [HITL] No metadata found for session %s (task %s)", a.id, sessionID, taskID)
		return nil, fmt.Errorf("no execution state found for task %s", taskID)
	}

	pendingExecutions, exists := metadata.Metadata[pendingExecutionsKey]
	if !exists {
		log.Printf("[Agent:%s] [HITL] No pending_executions key in session %s metadata (task %s). Available keys: %v", a.id, sessionID, taskID, getMetadataKeys(metadata.Metadata))
		return nil, fmt.Errorf("no pending executions in session")
	}

	pendingMap, ok := pendingExecutions.(map[string]interface{})
	if !ok {
		log.Printf("[Agent:%s] [HITL] Invalid pending_executions format in session %s (task %s): expected map, got %T", a.id, sessionID, taskID, pendingExecutions)
		return nil, fmt.Errorf("invalid pending_executions format")
	}

	taskState, exists := pendingMap[taskID]
	if !exists {
		availableTasks := make([]string, 0, len(pendingMap))
		for k := range pendingMap {
			availableTasks = append(availableTasks, k)
		}
		log.Printf("[Agent:%s] [HITL] Task %s not found in pending executions for session %s. Available tasks: %v", a.id, taskID, sessionID, availableTasks)
		return nil, fmt.Errorf("execution state not found for task %s", taskID)
	}

	// Serialize back to JSON for deserialization
	stateJSON, err := json.Marshal(taskState)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task state: %w", err)
	}

	execState, err := DeserializeExecutionState(stateJSON)
	if err != nil {
		log.Printf("[Agent:%s] [HITL] Failed to deserialize execution state for task %s in session %s: %v", a.id, taskID, sessionID, err)
		return nil, fmt.Errorf("failed to deserialize execution state: %w", err)
	}
	log.Printf("[Agent:%s] [HITL] Successfully loaded execution state for task %s from session %s", a.id, taskID, sessionID)
	return execState, nil
}

// getMetadataKeys returns a slice of all keys in the metadata map for debugging
func getMetadataKeys(metadata map[string]interface{}) []string {
	if metadata == nil {
		return []string{}
	}
	keys := make([]string, 0, len(metadata))
	for k := range metadata {
		keys = append(keys, k)
	}
	return keys
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
		// Programmatic API: auto-detect based on session service type
		// Only use async HITL if session service is SQL/persistent (not in-memory)
		sessionService := a.services.Session()
		if sessionService == nil {
			return false
		}
		// Check if it's a SQL session service (has GetDB method or similar indicator)
		// For now, we'll check if it's NOT an in-memory service by checking for a method
		// that only SQL services have. If we can't determine, default to blocking for safety.
		return isPersistentSessionService(sessionService)
	}

	taskCfg := a.config.Task
	mode := "auto" // Default
	if taskCfg != nil && taskCfg.HITL != nil {
		mode = taskCfg.HITL.Mode
		if mode == "" {
			mode = "auto"
		}
	}

	// Check if agent has a configured session_store (from config, not just any session service)
	// This is more reliable than checking if session service exists, since in-memory
	// session service is always created even without session_store config
	hasPersistentSessionStore := a.hasConfiguredSessionStore()

	switch mode {
	case "async":
		if !hasPersistentSessionStore {
			log.Printf("[Agent:%s] [HITL] Warning: async HITL mode requested but no persistent session_store configured, falling back to blocking mode", a.id)
			return false
		}
		return true // Explicit async (requires persistent store)
	case "blocking":
		return false // Explicit blocking
	case "auto":
		return hasPersistentSessionStore // Auto-detect: async only if persistent store configured
	default:
		return hasPersistentSessionStore // Fallback to auto-detect
	}
}

// hasConfiguredSessionStore checks if agent has a configured session_store in config
// (not just an in-memory session service which is always created)
func (a *Agent) hasConfiguredSessionStore() bool {
	// For config-based agents, check if SessionStore field is set in agent config
	// This is the most reliable way to determine if a persistent session store was configured
	if a.config != nil && a.config.SessionStore != "" {
		// SessionStore is configured in agent config - verify it's a persistent store
		// by checking if the session service is SQL-based
		return isPersistentSessionService(a.services.Session())
	}

	// For programmatic API or agents without SessionStore config, check if session service is persistent
	return isPersistentSessionService(a.services.Session())
}

// persistentSessionService is a minimal interface to check if a session service is persistent
// SQLSessionService implements Close() method, InMemorySessionService does not
type persistentSessionService interface {
	Close() error
}

// isPersistentSessionService checks if a session service is persistent (SQL) vs in-memory
// It uses type assertion to check if the service implements Close() method,
// which only SQLSessionService has (InMemorySessionService doesn't implement it)
func isPersistentSessionService(service reasoning.SessionService) bool {
	if service == nil {
		return false
	}

	// Use type assertion to check if service implements Close() method
	// This is more reliable than string matching on type names
	_, isPersistent := service.(persistentSessionService)
	return isPersistent
}
