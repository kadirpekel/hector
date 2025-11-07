package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// TaskAwaiter handles paused tasks waiting for user input
// This implements A2A Protocol Section 6.3 - INPUT_REQUIRED state
type TaskAwaiter struct {
	mu sync.RWMutex

	// Paused tasks waiting for input: taskID → input channel
	waiting map[string]chan *pb.Message

	// Timeout for waiting tasks
	defaultTimeout time.Duration
}

// NewTaskAwaiter creates a new task awaiter with specified default timeout
func NewTaskAwaiter(timeout time.Duration) *TaskAwaiter {
	if timeout == 0 {
		timeout = 10 * time.Minute // Default 10 minutes
	}

	return &TaskAwaiter{
		waiting:        make(map[string]chan *pb.Message),
		defaultTimeout: timeout,
	}
}

// WaitForInput pauses task execution and waits for user input
// This is called when a task transitions to INPUT_REQUIRED state
// Returns the user's message or error if timeout/cancelled
func (a *TaskAwaiter) WaitForInput(
	ctx context.Context,
	taskID string,
	timeout time.Duration,
) (*pb.Message, error) {

	if timeout == 0 {
		timeout = a.defaultTimeout
	}

	// Create channel for this task
	inputCh := make(chan *pb.Message, 1)

	a.mu.Lock()
	a.waiting[taskID] = inputCh
	a.mu.Unlock()

	// Cleanup on exit
	defer func() {
		a.mu.Lock()
		delete(a.waiting, taskID)
		a.mu.Unlock()
		close(inputCh)
	}()

	// Wait for input, timeout, or cancellation
	select {
	case msg := <-inputCh:
		return msg, nil

	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for user input")

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ProvideInput delivers user input to a waiting task
// This is called when a client sends a message with an existing taskId
func (a *TaskAwaiter) ProvideInput(taskID string, message *pb.Message) error {
	a.mu.RLock()
	ch, exists := a.waiting[taskID]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task not waiting for input: %s", taskID)
	}

	// Deliver message (non-blocking to prevent deadlock)
	select {
	case ch <- message:
		return nil
	default:
		return fmt.Errorf("task already received input: %s", taskID)
	}
}

// IsWaiting checks if a task is currently waiting for input
func (a *TaskAwaiter) IsWaiting(taskID string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, exists := a.waiting[taskID]
	return exists
}

// CancelWaiting cancels a waiting task (e.g., when task is cancelled)
func (a *TaskAwaiter) CancelWaiting(taskID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if ch, exists := a.waiting[taskID]; exists {
		close(ch)
		delete(a.waiting, taskID)
	}
}

// GetWaitingTasks returns list of task IDs currently waiting for input
func (a *TaskAwaiter) GetWaitingTasks() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	tasks := make([]string, 0, len(a.waiting))
	for taskID := range a.waiting {
		tasks = append(tasks, taskID)
	}
	return tasks
}
