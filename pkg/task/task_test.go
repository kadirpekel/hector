package task_test

import (
	"testing"
	"time"

	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/task"
)

// =============================================================================
// State Tests
// =============================================================================

func TestState_IsTerminal(t *testing.T) {
	tests := []struct {
		state    task.State
		terminal bool
	}{
		{task.StateSubmitted, false},
		{task.StateWorking, false},
		{task.StateCompleted, true},
		{task.StateFailed, true},
		{task.StateCancelled, true},
		{task.StateRejected, true},
		{task.StateInputRequired, false},
		{task.StateAuthRequired, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsTerminal(); got != tt.terminal {
				t.Errorf("State(%q).IsTerminal() = %v, want %v", tt.state, got, tt.terminal)
			}
		})
	}
}

func TestState_IsPending(t *testing.T) {
	tests := []struct {
		state   task.State
		pending bool
	}{
		{task.StateSubmitted, false},
		{task.StateWorking, false},
		{task.StateCompleted, false},
		{task.StateFailed, false},
		{task.StateCancelled, false},
		{task.StateRejected, false},
		{task.StateInputRequired, true},
		{task.StateAuthRequired, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsPending(); got != tt.pending {
				t.Errorf("State(%q).IsPending() = %v, want %v", tt.state, got, tt.pending)
			}
		})
	}
}

// =============================================================================
// Task Creation Tests
// =============================================================================

func TestNew(t *testing.T) {
	tsk := task.New("context-123", "default")

	if tsk.ID == "" {
		t.Error("Task ID should be generated")
	}
	if tsk.ContextID != "context-123" {
		t.Errorf("ContextID = %q, want context-123", tsk.ContextID)
	}
	if tsk.Status.State != task.StateSubmitted {
		t.Errorf("Initial state = %v, want submitted", tsk.Status.State)
	}
	if tsk.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if tsk.History == nil {
		t.Error("History should be initialized")
	}
	if tsk.Artifacts == nil {
		t.Error("Artifacts should be initialized")
	}
	if tsk.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
}

func TestTask_GetID(t *testing.T) {
	tsk := task.New("ctx", "default")
	if tsk.GetID() != tsk.ID {
		t.Error("GetID() should return ID field")
	}
}

// =============================================================================
// Task Status Tests
// =============================================================================

func TestTask_SetStatus(t *testing.T) {
	tsk := task.New("ctx", "default")
	beforeUpdate := tsk.UpdatedAt

	time.Sleep(10 * time.Millisecond)
	tsk.SetStatus(task.StateWorking, nil, nil)

	if tsk.Status.State != task.StateWorking {
		t.Errorf("State = %v, want working", tsk.Status.State)
	}
	if !tsk.UpdatedAt.After(beforeUpdate) {
		t.Error("UpdatedAt should be updated")
	}
}

func TestTask_GetStatus(t *testing.T) {
	tsk := task.New("ctx", "default")
	tsk.SetStatus(task.StateCompleted, nil, nil)

	status := tsk.GetStatus()
	if status.State != task.StateCompleted {
		t.Errorf("GetStatus().State = %v, want completed", status.State)
	}
}

func TestTask_SetStatus_WithError(t *testing.T) {
	tsk := task.New("ctx", "default")
	err := task.ErrTaskNotFound

	tsk.SetStatus(task.StateFailed, nil, err)

	status := tsk.GetStatus()
	if status.State != task.StateFailed {
		t.Error("State should be failed")
	}
	if status.Error != err {
		t.Error("Error should be set")
	}
}

// =============================================================================
// HITL (Human-in-the-Loop) Tests
// =============================================================================

func TestTask_RequestInput(t *testing.T) {
	tsk := task.New("ctx", "default")

	req := &task.InputRequirement{
		Type:    task.InputTypeToolApproval,
		Timeout: 5 * time.Minute,
		Options: []task.InputOption{
			{ID: "approve", Label: "Approve"},
			{ID: "deny", Label: "Deny"},
		},
	}

	tsk.RequestInput(req)

	if tsk.Status.State != task.StateInputRequired {
		t.Errorf("State = %v, want input_required", tsk.Status.State)
	}
	if tsk.InputRequirement == nil {
		t.Fatal("InputRequirement should be set")
	}
	if tsk.InputRequirement.RequestedAt.IsZero() {
		t.Error("RequestedAt should be set")
	}
}

func TestTask_ProvideInput(t *testing.T) {
	tsk := task.New("ctx", "default")
	tsk.RequestInput(&task.InputRequirement{
		Type: task.InputTypeToolApproval,
		Options: []task.InputOption{
			{ID: "approve", Label: "Approve", Value: true},
			{ID: "deny", Label: "Deny", Value: false},
		},
	})

	selected := tsk.ProvideInput("approve")

	if selected == nil {
		t.Fatal("Should return selected option")
	}
	if selected.ID != "approve" {
		t.Errorf("selected.ID = %q, want approve", selected.ID)
	}
	if selected.Value != true {
		t.Error("selected.Value should be true")
	}
	if tsk.Status.State != task.StateWorking {
		t.Errorf("State = %v, want working (resumed)", tsk.Status.State)
	}
	if tsk.InputRequirement != nil {
		t.Error("InputRequirement should be cleared")
	}
}

func TestTask_ProvideInput_InvalidOption(t *testing.T) {
	tsk := task.New("ctx", "default")
	tsk.RequestInput(&task.InputRequirement{
		Type: task.InputTypeToolApproval,
		Options: []task.InputOption{
			{ID: "approve", Label: "Approve"},
		},
	})

	selected := tsk.ProvideInput("nonexistent")

	if selected != nil {
		t.Error("Should return nil for invalid option")
	}
	// Task should still resume
	if tsk.Status.State != task.StateWorking {
		t.Error("Task should resume even with invalid option")
	}
}

func TestTask_ProvideInput_NoRequirement(t *testing.T) {
	tsk := task.New("ctx", "default")

	selected := tsk.ProvideInput("anything")

	if selected != nil {
		t.Error("Should return nil when no input required")
	}
}

// =============================================================================
// Execution State Tests
// =============================================================================

func TestTask_SaveExecutionState(t *testing.T) {
	tsk := task.New("ctx", "default")

	state := &task.ExecutionState{
		Phase:     "tool_call",
		Iteration: 3,
		Custom:    map[string]any{"key": "value"},
	}

	tsk.SaveExecutionState(state)

	if tsk.ExecutionState == nil {
		t.Fatal("ExecutionState should be saved")
	}
	if tsk.ExecutionState.Phase != "tool_call" {
		t.Errorf("Phase = %q, want tool_call", tsk.ExecutionState.Phase)
	}
	if tsk.ExecutionState.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestTask_LoadExecutionState(t *testing.T) {
	tsk := task.New("ctx", "default")
	tsk.SaveExecutionState(&task.ExecutionState{Phase: "loop"})

	loaded := tsk.LoadExecutionState()

	if loaded == nil {
		t.Fatal("Should return saved state")
	}
	if loaded.Phase != "loop" {
		t.Errorf("Phase = %q, want loop", loaded.Phase)
	}
	if tsk.ExecutionState != nil {
		t.Error("ExecutionState should be cleared after load")
	}
}

func TestTask_LoadExecutionState_Empty(t *testing.T) {
	tsk := task.New("ctx", "default")

	loaded := tsk.LoadExecutionState()

	if loaded != nil {
		t.Error("Should return nil when no state saved")
	}
}

// =============================================================================
// History and Artifact Tests
// =============================================================================

func TestTask_AppendHistory(t *testing.T) {
	tsk := task.New("ctx", "default")

	tsk.AppendHistory(nil) // Should handle nil gracefully
	if len(tsk.History) != 1 {
		t.Errorf("History length = %d, want 1", len(tsk.History))
	}
}

func TestTask_SetMetadata(t *testing.T) {
	tsk := task.New("ctx", "default")

	tsk.SetMetadata("key1", "value1")
	tsk.SetMetadata("key2", 42)

	if tsk.Metadata["key1"] != "value1" {
		t.Errorf("Metadata[key1] = %v, want value1", tsk.Metadata["key1"])
	}
	if tsk.Metadata["key2"] != 42 {
		t.Errorf("Metadata[key2] = %v, want 42", tsk.Metadata["key2"])
	}
}

// =============================================================================
// Child Execution Tracking Tests
// =============================================================================

func TestTask_RegisterExecution(t *testing.T) {
	tsk := task.New("ctx", "default")

	exec := &agent.ChildExecution{
		CallID: "call-123",
		Name:   "test-tool",
	}

	tsk.RegisterExecution(exec)
	// Registration doesn't have a getter - tested via Cancel
}

func TestTask_RegisterExecution_NilOrEmpty(t *testing.T) {
	tsk := task.New("ctx", "default")

	// Should not panic
	tsk.RegisterExecution(nil)
	tsk.RegisterExecution(&agent.ChildExecution{CallID: ""})
}

func TestTask_CancelExecution(t *testing.T) {
	tsk := task.New("ctx", "default")

	cancelled := false
	exec := &agent.ChildExecution{
		CallID: "call-123",
		Cancel: func() bool {
			cancelled = true
			return true
		},
	}

	tsk.RegisterExecution(exec)
	result := tsk.CancelExecution("call-123")

	if !result {
		t.Error("CancelExecution should return true")
	}
	if !cancelled {
		t.Error("Cancel function should be called")
	}
}

func TestTask_CancelExecution_NotFound(t *testing.T) {
	tsk := task.New("ctx", "default")

	result := tsk.CancelExecution("nonexistent")

	if result {
		t.Error("CancelExecution should return false for unknown callID")
	}
}

func TestTask_CancelAllChildren(t *testing.T) {
	tsk := task.New("ctx", "default")

	cancelCount := 0
	for i := 0; i < 3; i++ {
		exec := &agent.ChildExecution{
			CallID: "call-" + string(rune('0'+i)),
			Cancel: func() bool {
				cancelCount++
				return true
			},
		}
		tsk.RegisterExecution(exec)
	}

	cancelled, failed := tsk.CancelAllChildren()

	if cancelled != 3 {
		t.Errorf("cancelled = %d, want 3", cancelled)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}
	if cancelCount != 3 {
		t.Errorf("cancelCount = %d, want 3", cancelCount)
	}
}

// =============================================================================
// InMemoryService Tests
// =============================================================================
