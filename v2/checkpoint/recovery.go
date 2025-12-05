// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package checkpoint

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// RecoveryManager handles checkpoint recovery on startup and during runtime.
//
// Architecture (ported from legacy Hector):
//
//	On startup, RecoveryManager scans for pending checkpoints and:
//	  1. Validates checkpoint states (not expired, recoverable)
//	  2. For WORKING tasks: Auto-resumes if configured
//	  3. For INPUT_REQUIRED tasks: Waits for user action (unless AutoResumeHITL)
//	  4. For expired checkpoints: Marks tasks as FAILED
type RecoveryManager struct {
	config  *Config
	storage *Storage

	// resumeCallback is called to resume a task from checkpoint
	resumeCallback ResumeCallback

	mu sync.RWMutex
}

// ResumeCallback is called to resume execution from a checkpoint.
type ResumeCallback func(ctx context.Context, state *State) error

// NewRecoveryManager creates a new RecoveryManager.
func NewRecoveryManager(cfg *Config, storage *Storage) *RecoveryManager {
	return &RecoveryManager{
		config:  cfg,
		storage: storage,
	}
}

// SetResumeCallback sets the callback for resuming tasks.
func (m *RecoveryManager) SetResumeCallback(cb ResumeCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resumeCallback = cb
}

// RecoverPendingTasks recovers tasks with checkpoints on startup.
// This should be called during server initialization.
func (m *RecoveryManager) RecoverPendingTasks(ctx context.Context, appName string) error {
	if !m.config.ShouldAutoResume() {
		slog.Debug("Checkpoint recovery disabled", "app_name", appName)
		return nil
	}

	// Find all pending checkpoints
	states, err := m.storage.ListAllPending(ctx, appName)
	if err != nil {
		return fmt.Errorf("failed to list pending checkpoints: %w", err)
	}

	if len(states) == 0 {
		slog.Debug("No pending checkpoints to recover", "app_name", appName)
		return nil
	}

	slog.Info("Found pending checkpoints, starting recovery",
		"app_name", appName,
		"count", len(states))

	recoveredCount := 0
	failedCount := 0

	for _, state := range states {
		if err := m.recoverCheckpoint(ctx, state); err != nil {
			slog.Error("Failed to recover checkpoint",
				"task_id", state.TaskID,
				"session_id", state.SessionID,
				"error", err)
			failedCount++
			continue
		}
		recoveredCount++
	}

	slog.Info("Checkpoint recovery completed",
		"app_name", appName,
		"recovered", recoveredCount,
		"failed", failedCount)

	return nil
}

// recoverCheckpoint attempts to recover a single checkpoint.
func (m *RecoveryManager) recoverCheckpoint(ctx context.Context, state *State) error {
	// Check if checkpoint is recoverable
	if !state.IsRecoverable() {
		return fmt.Errorf("checkpoint not recoverable (phase=%s)", state.Phase)
	}

	// Check if checkpoint has expired
	timeout := m.config.GetRecoveryTimeout()
	if state.IsExpired(timeout) {
		slog.Warn("Checkpoint expired",
			"task_id", state.TaskID,
			"checkpoint_time", state.CheckpointTime,
			"timeout", timeout)
		// Clear the expired checkpoint
		if err := m.storage.Clear(ctx, state.AppName, state.UserID, state.SessionID, state.TaskID); err != nil {
			slog.Warn("Failed to clear expired checkpoint", "error", err)
		}
		return fmt.Errorf("checkpoint expired")
	}

	// Check if this is an INPUT_REQUIRED state that needs user action
	if state.NeedsUserInput() && !m.config.ShouldAutoResumeHITL() {
		slog.Info("Checkpoint awaiting user input (auto-resume HITL disabled)",
			"task_id", state.TaskID,
			"session_id", state.SessionID)
		return nil // Don't resume - wait for user
	}

	// Resume execution
	m.mu.RLock()
	callback := m.resumeCallback
	m.mu.RUnlock()

	if callback == nil {
		slog.Warn("No resume callback configured, checkpoint will be recovered on next access",
			"task_id", state.TaskID)
		return nil
	}

	slog.Info("Resuming task from checkpoint",
		"task_id", state.TaskID,
		"session_id", state.SessionID,
		"phase", state.Phase,
		"checkpoint_type", state.CheckpointType)

	// Run resume in background
	go func() {
		if err := callback(ctx, state); err != nil {
			slog.Error("Failed to resume task from checkpoint",
				"task_id", state.TaskID,
				"error", err)
		}
	}()

	return nil
}

// ResumeTask manually resumes a task from checkpoint.
// This is used when a user explicitly requests to resume an INPUT_REQUIRED task.
func (m *RecoveryManager) ResumeTask(ctx context.Context, appName, userID, sessionID, taskID string, userInput string) error {
	// Load the checkpoint
	state, err := m.storage.Load(ctx, appName, userID, sessionID, taskID)
	if err != nil {
		return fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Validate state
	if !state.IsRecoverable() {
		return fmt.Errorf("checkpoint not recoverable")
	}

	// Check expiry
	if state.IsExpired(m.config.GetRecoveryTimeout()) {
		// Clear expired checkpoint
		_ = m.storage.Clear(ctx, appName, userID, sessionID, taskID)
		return fmt.Errorf("checkpoint expired")
	}

	// Get callback
	m.mu.RLock()
	callback := m.resumeCallback
	m.mu.RUnlock()

	if callback == nil {
		return fmt.Errorf("no resume callback configured")
	}

	// Add user input to state if provided
	if userInput != "" {
		if state.PendingToolCall != nil {
			// Store user decision in custom state
			if state.AgentState == nil {
				state.AgentState = &AgentStateSnapshot{}
			}
			if state.AgentState.Custom == nil {
				state.AgentState.Custom = make(map[string]any)
			}
			state.AgentState.Custom["user_input"] = userInput
		}
	}

	return callback(ctx, state)
}

// GetPendingCheckpoints returns all pending checkpoints for a user.
func (m *RecoveryManager) GetPendingCheckpoints(ctx context.Context, appName, userID string) ([]*State, error) {
	return m.storage.ListPending(ctx, appName, userID)
}

// GetCheckpoint returns a specific checkpoint.
func (m *RecoveryManager) GetCheckpoint(ctx context.Context, appName, userID, sessionID, taskID string) (*State, error) {
	return m.storage.Load(ctx, appName, userID, sessionID, taskID)
}

// CancelCheckpoint removes a checkpoint without resuming.
func (m *RecoveryManager) CancelCheckpoint(ctx context.Context, appName, userID, sessionID, taskID string) error {
	return m.storage.Clear(ctx, appName, userID, sessionID, taskID)
}

// CheckpointStats contains statistics about pending checkpoints.
type CheckpointStats struct {
	Total         int
	Working       int
	InputRequired int
	Expired       int
	OldestAge     time.Duration
	AverageAge    time.Duration
}

// GetStats returns statistics about pending checkpoints.
func (m *RecoveryManager) GetStats(ctx context.Context, appName string) (*CheckpointStats, error) {
	states, err := m.storage.ListAllPending(ctx, appName)
	if err != nil {
		return nil, err
	}

	stats := &CheckpointStats{
		Total: len(states),
	}

	if len(states) == 0 {
		return stats, nil
	}

	var totalAge time.Duration
	timeout := m.config.GetRecoveryTimeout()

	for _, state := range states {
		age := time.Since(state.CheckpointTime)
		totalAge += age

		if age > stats.OldestAge {
			stats.OldestAge = age
		}

		if state.IsExpired(timeout) {
			stats.Expired++
		} else if state.NeedsUserInput() {
			stats.InputRequired++
		} else {
			stats.Working++
		}
	}

	if len(states) > 0 {
		stats.AverageAge = totalAge / time.Duration(len(states))
	}

	return stats, nil
}
