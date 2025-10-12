package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InMemoryTaskService implements reasoning.TaskService
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

	go func() {
		<-ctx.Done()
		s.unsubscribe(taskID, ch)
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
