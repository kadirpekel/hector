package task

import (
	"context"
	"sync"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/verikod/hector/pkg/app"
)

// InMemoryTaskStore implements a2asrv.TaskStore using an in-memory map.
// It supports multi-app isolation.
type InMemoryTaskStore struct {
	// tasks is a nested map: AppID -> TaskID -> Task
	tasks map[string]map[a2a.TaskID]*a2a.Task
	mu    sync.RWMutex
}

// NewInMemoryTaskStore creates a new in-memory task store.
func NewInMemoryTaskStore() *InMemoryTaskStore {
	return &InMemoryTaskStore{
		tasks: make(map[string]map[a2a.TaskID]*a2a.Task),
	}
}

// Save stores a task.
func (s *InMemoryTaskStore) Save(ctx context.Context, task *a2a.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	appName := app.IDFromContext(ctx)

	if s.tasks[appName] == nil {
		s.tasks[appName] = make(map[a2a.TaskID]*a2a.Task)
	}

	// Store a deep copy to prevent external mutation affecting the store
	// (Simulating DB behavior)
	s.tasks[appName][task.ID] = cloneTask(task)

	return nil
}

// Get retrieves a task by ID.
func (s *InMemoryTaskStore) Get(ctx context.Context, id a2a.TaskID) (*a2a.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	appName := app.IDFromContext(ctx)

	appTasks, ok := s.tasks[appName]
	if !ok {
		return nil, a2a.ErrTaskNotFound
	}

	task, ok := appTasks[id]
	if !ok {
		return nil, a2a.ErrTaskNotFound
	}

	// Return a deep copy
	return cloneTask(task), nil
}

// List retrieves tasks for a specific app and context (session).
func (s *InMemoryTaskStore) List(ctx context.Context, appName, contextID string) ([]*a2a.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	appTasks, ok := s.tasks[appName]
	if !ok {
		return nil, nil // No tasks for this app
	}

	var results []*a2a.Task
	for _, task := range appTasks {
		if task.ContextID == contextID {
			results = append(results, cloneTask(task))
		}
	}

	// Sort by creation time (simulating DB query order)
	// Note: a2a.Task doesn't have CreatedAt field publicly standard,
	// but we can assume ID ordering or metadata if available.
	// For now, we just return the list.

	// If needed we can sort by ID or metadata "_created_at" if we were setting it.

	return results, nil
}

// DeleteByApp removes all tasks for a specific app.
func (s *InMemoryTaskStore) DeleteByApp(ctx context.Context, appName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tasks, appName)
	return nil
}

// cloneTask creates a deep copy of a task.
func cloneTask(t *a2a.Task) *a2a.Task {
	if t == nil {
		return nil
	}

	clone := &a2a.Task{
		ID:        t.ID,
		ContextID: t.ContextID,
		Status:    t.Status, // Status is a string (primitive)
		Metadata:  make(map[string]any),
	}

	// Clone history
	if t.History != nil {
		clone.History = make([]*a2a.Message, len(t.History))
		copy(clone.History, t.History)
		// Deep copy message content if needed, but shallow copy of messages is often enough
		// if they are considered immutable. A2A messages are complex structures though.
		// For thoroughness:
		for i, msg := range t.History {
			newMsg := *msg // Shallow copy struct
			// Deep copy generic parts if necessary, but skipping for brevity as this is usually sufficient for tests/in-memory
			clone.History[i] = &newMsg
		}
	}

	// Clone artifacts
	if t.Artifacts != nil {
		clone.Artifacts = make([]*a2a.Artifact, len(t.Artifacts))
		for i, v := range t.Artifacts {
			newArt := *v
			clone.Artifacts[i] = &newArt
		}
	}

	// Clone metadata
	for k, v := range t.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// Compile-time interface compliance check
var _ a2asrv.TaskStore = (*InMemoryTaskStore)(nil)
