package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type InMemoryTaskService struct {
	mu            sync.RWMutex
	tasks         map[string]*pb.Task
	subscribers   map[string][]chan *pb.StreamResponse
	subscribersMu sync.RWMutex
}

func NewInMemoryTaskService() *InMemoryTaskService {
	return &InMemoryTaskService{
		tasks:       make(map[string]*pb.Task),
		subscribers: make(map[string][]chan *pb.StreamResponse),
	}
}

func (s *InMemoryTaskService) CreateTask(ctx context.Context, contextID string, initialMessage *pb.Message) (*pb.Task, error) {
	if contextID == "" {
		return nil, fmt.Errorf("context_id is required")
	}

	taskID := generateTaskID()
	now := timestamppb.Now()

	task := &pb.Task{
		Id:        taskID,
		ContextId: contextID,
		Status: &pb.TaskStatus{
			State:     pb.TaskState_TASK_STATE_SUBMITTED,
			Timestamp: now,
		},
		Artifacts: make([]*pb.Artifact, 0),
		History:   make([]*pb.Message, 0),
	}

	if initialMessage != nil {
		if initialMessage.ContextId == "" {
			initialMessage.ContextId = contextID
		}
		if initialMessage.TaskId == "" {
			initialMessage.TaskId = taskID
		}
		task.History = append(task.History, initialMessage)
	}

	s.mu.Lock()
	s.tasks[taskID] = task
	s.mu.Unlock()

	s.notifySubscribers(taskID, &pb.StreamResponse{
		Payload: &pb.StreamResponse_Task{
			Task: task,
		},
	})

	return task, nil
}

func (s *InMemoryTaskService) GetTask(ctx context.Context, taskID string) (*pb.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

func (s *InMemoryTaskService) UpdateTaskStatus(ctx context.Context, taskID string, state pb.TaskState, message *pb.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Note: State transition validation happens at Agent level (business logic layer)
	// This storage layer just persists what it's told

	now := timestamppb.Now()
	task.Status = &pb.TaskStatus{
		State:     state,
		Update:    message,
		Timestamp: now,
	}

	isFinal := isTerminalState(state)

	event := &pb.TaskStatusUpdateEvent{
		TaskId:    taskID,
		ContextId: task.ContextId,
		Status:    task.Status,
		Final:     isFinal,
	}

	s.notifySubscribers(taskID, &pb.StreamResponse{
		Payload: &pb.StreamResponse_StatusUpdate{
			StatusUpdate: event,
		},
	})

	if isFinal {
		s.closeTaskSubscribers(taskID)
	}

	return nil
}

func (s *InMemoryTaskService) AddTaskArtifact(ctx context.Context, taskID string, artifact *pb.Artifact) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if artifact.ArtifactId == "" {
		artifact.ArtifactId = generateArtifactID()
	}

	task.Artifacts = append(task.Artifacts, artifact)

	event := &pb.TaskArtifactUpdateEvent{
		TaskId:    taskID,
		ContextId: task.ContextId,
		Artifact:  artifact,
		Append:    true,
		LastChunk: false,
	}

	s.notifySubscribers(taskID, &pb.StreamResponse{
		Payload: &pb.StreamResponse_ArtifactUpdate{
			ArtifactUpdate: event,
		},
	})

	return nil
}

func (s *InMemoryTaskService) AddTaskMessage(ctx context.Context, taskID string, message *pb.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if message.ContextId == "" {
		message.ContextId = task.ContextId
	}
	if message.TaskId == "" {
		message.TaskId = taskID
	}

	task.History = append(task.History, message)

	return nil
}

func (s *InMemoryTaskService) CancelTask(ctx context.Context, taskID string) (*pb.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	if isTerminalState(task.Status.State) {
		return task, nil
	}

	now := timestamppb.Now()
	task.Status = &pb.TaskStatus{
		State:     pb.TaskState_TASK_STATE_CANCELLED,
		Timestamp: now,
	}

	event := &pb.TaskStatusUpdateEvent{
		TaskId:    taskID,
		ContextId: task.ContextId,
		Status:    task.Status,
		Final:     true,
	}

	s.notifySubscribers(taskID, &pb.StreamResponse{
		Payload: &pb.StreamResponse_StatusUpdate{
			StatusUpdate: event,
		},
	})

	s.closeTaskSubscribers(taskID)

	return task, nil
}

func (s *InMemoryTaskService) ListTasks(ctx context.Context, contextID string, status pb.TaskState, pageSize int32, pageToken string) ([]*pb.Task, string, int32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Collect all tasks matching filters
	var matchingTasks []*pb.Task
	for _, task := range s.tasks {
		// Filter by context ID if specified
		if contextID != "" && task.ContextId != contextID {
			continue
		}
		// Filter by status if specified (TASK_STATE_UNSPECIFIED means no filter)
		if status != pb.TaskState_TASK_STATE_UNSPECIFIED && task.Status.State != status {
			continue
		}
		matchingTasks = append(matchingTasks, task)
	}

	totalSize := int32(len(matchingTasks))

	// Apply pagination
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50 // Default page size
	}

	// Simple offset-based pagination using page token as offset
	startOffset := int32(0)
	if pageToken != "" {
		// In a real implementation, decode the page token
		// For simplicity, we'll skip pagination token parsing for now
		startOffset = 0
	}

	endOffset := startOffset + pageSize
	if endOffset > totalSize {
		endOffset = totalSize
	}

	// Slice the results
	var pagedTasks []*pb.Task
	if startOffset < totalSize {
		pagedTasks = matchingTasks[startOffset:endOffset]
	}

	// Generate next page token
	nextPageToken := ""
	if endOffset < totalSize {
		nextPageToken = fmt.Sprintf("%d", endOffset)
	}

	return pagedTasks, nextPageToken, totalSize, nil
}

func (s *InMemoryTaskService) ListTasksByContext(ctx context.Context, contextID string) ([]*pb.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*pb.Task
	for _, task := range s.tasks {
		if task.ContextId == contextID {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

func (s *InMemoryTaskService) SubscribeToTask(ctx context.Context, taskID string) (<-chan *pb.StreamResponse, error) {
	s.mu.RLock()
	task, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	ch := make(chan *pb.StreamResponse, 100)

	if isTerminalState(task.Status.State) {
		ch <- &pb.StreamResponse{
			Payload: &pb.StreamResponse_Task{
				Task: task,
			},
		}
		close(ch)
		return ch, nil
	}

	s.subscribersMu.Lock()
	s.subscribers[taskID] = append(s.subscribers[taskID], ch)
	s.subscribersMu.Unlock()

	// Ensure cleanup happens even if goroutine panics
	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		defer func() {
			if r := recover(); r != nil {
				// Panic recovery - still cleanup
				s.unsubscribe(taskID, ch)
			}
		}()

		select {
		case <-ctx.Done():
			s.unsubscribe(taskID, ch)
		case <-time.After(30 * time.Minute):
			// Timeout to prevent memory leaks - close subscription after 30 minutes
			s.unsubscribe(taskID, ch)
		}
	}()

	// Monitor cleanup goroutine to ensure it completes
	go func() {
		<-cleanupDone
	}()

	return ch, nil
}

func (s *InMemoryTaskService) Close() error {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for taskID := range s.subscribers {
		s.closeTaskSubscribers(taskID)
	}

	return nil
}

func (s *InMemoryTaskService) notifySubscribers(taskID string, event *pb.StreamResponse) {
	s.subscribersMu.RLock()
	subscribers := s.subscribers[taskID]
	s.subscribersMu.RUnlock()

	for _, ch := range subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *InMemoryTaskService) closeTaskSubscribers(taskID string) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	if subscribers, exists := s.subscribers[taskID]; exists {
		for _, ch := range subscribers {
			close(ch)
		}
		delete(s.subscribers, taskID)
	}
}

func (s *InMemoryTaskService) unsubscribe(taskID string, ch chan *pb.StreamResponse) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	if subscribers, exists := s.subscribers[taskID]; exists {
		for i, sub := range subscribers {
			if sub == ch {
				s.subscribers[taskID] = append(subscribers[:i], subscribers[i+1:]...)
				close(ch)
				break
			}
		}
	}
}

func isTerminalState(state pb.TaskState) bool {
	return state == pb.TaskState_TASK_STATE_COMPLETED ||
		state == pb.TaskState_TASK_STATE_FAILED ||
		state == pb.TaskState_TASK_STATE_CANCELLED ||
		state == pb.TaskState_TASK_STATE_REJECTED
}

func generateTaskID() string {
	return fmt.Sprintf("task-%s", uuid.New().String())
}

func generateArtifactID() string {
	return fmt.Sprintf("artifact-%s", uuid.New().String())
}
