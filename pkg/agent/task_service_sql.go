package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type SQLTaskService struct {
	db            *sql.DB
	dialect       string
	subscribers   map[string][]chan *pb.StreamResponse
	subscribersMu sync.RWMutex
}

type taskRow struct {
	ID         string
	ContextID  string
	State      string
	StatusJSON string
	Artifacts  string
	History    string
	Metadata   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

const (
	createTableSQL = `
CREATE TABLE IF NOT EXISTS tasks (
    id VARCHAR(255) PRIMARY KEY,
    context_id VARCHAR(255) NOT NULL,
    state VARCHAR(50) NOT NULL,
    status_json TEXT,
    artifacts TEXT,
    history TEXT,
    metadata TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_context_id ON tasks(context_id);
CREATE INDEX IF NOT EXISTS idx_tasks_state ON tasks(state);
CREATE INDEX IF NOT EXISTS idx_tasks_updated_at ON tasks(updated_at);
`
)

func NewSQLTaskService(db *sql.DB, dialect string) (*SQLTaskService, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	switch dialect {
	case "postgres", "mysql", "sqlite":

	default:
		return nil, fmt.Errorf("unsupported dialect: %s (supported: postgres, mysql, sqlite)", dialect)
	}

	s := &SQLTaskService{
		db:          db,
		dialect:     dialect,
		subscribers: make(map[string][]chan *pb.StreamResponse),
	}

	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

func NewSQLTaskServiceFromConfig(cfg *config.TaskSQLConfig) (*SQLTaskService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("SQL configuration is required")
	}

	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	driverName := cfg.Driver
	if driverName == "sqlite" {
		driverName = "sqlite3"
	}

	db, err := sql.Open(driverName, cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return NewSQLTaskService(db, cfg.Driver)
}

func (s *SQLTaskService) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schemaSQL := createTableSQL

	_ = s.dialect

	_, err := s.db.ExecContext(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

func (s *SQLTaskService) CreateTask(ctx context.Context, contextID string, initialMessage *pb.Message) (*pb.Task, error) {
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

	row, err := s.taskToRow(task)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize task: %w", err)
	}

	query := `
INSERT INTO tasks (id, context_id, state, status_json, artifacts, history, metadata, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	if s.dialect == "postgres" {
		query = `
INSERT INTO tasks (id, context_id, state, status_json, artifacts, history, metadata, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`
	}

	_, err = s.db.ExecContext(ctx, query,
		row.ID, row.ContextID, row.State, row.StatusJSON,
		row.Artifacts, row.History, row.Metadata,
		row.CreatedAt, row.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert task: %w", err)
	}

	s.notifySubscribers(taskID, &pb.StreamResponse{
		Payload: &pb.StreamResponse_Task{
			Task: task,
		},
	})

	return task, nil
}

func (s *SQLTaskService) GetTask(ctx context.Context, taskID string) (*pb.Task, error) {
	query := `
SELECT id, context_id, state, status_json, artifacts, history, metadata, created_at, updated_at
FROM tasks
WHERE id = ?
`
	if s.dialect == "postgres" {
		query = `
SELECT id, context_id, state, status_json, artifacts, history, metadata, created_at, updated_at
FROM tasks
WHERE id = $1
`
	}

	var row taskRow
	err := s.db.QueryRowContext(ctx, query, taskID).Scan(
		&row.ID, &row.ContextID, &row.State, &row.StatusJSON,
		&row.Artifacts, &row.History, &row.Metadata,
		&row.CreatedAt, &row.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query task: %w", err)
	}

	return s.rowToTask(&row)
}

func (s *SQLTaskService) UpdateTaskStatus(ctx context.Context, taskID string, state pb.TaskState, message *pb.Message) error {

	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	now := timestamppb.Now()
	task.Status = &pb.TaskStatus{
		State:     state,
		Update:    message,
		Timestamp: now,
	}

	row, err := s.taskToRow(task)
	if err != nil {
		return fmt.Errorf("failed to serialize task: %w", err)
	}

	query := `
UPDATE tasks
SET state = ?, status_json = ?, updated_at = ?
WHERE id = ?
`
	if s.dialect == "postgres" {
		query = `
UPDATE tasks
SET state = $1, status_json = $2, updated_at = $3
WHERE id = $4
`
	}

	_, err = s.db.ExecContext(ctx, query, row.State, row.StatusJSON, row.UpdatedAt, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
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

func (s *SQLTaskService) AddTaskArtifact(ctx context.Context, taskID string, artifact *pb.Artifact) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if artifact.ArtifactId == "" {
		artifact.ArtifactId = generateArtifactID()
	}

	task.Artifacts = append(task.Artifacts, artifact)

	row, err := s.taskToRow(task)
	if err != nil {
		return fmt.Errorf("failed to serialize task: %w", err)
	}

	query := `
UPDATE tasks
SET artifacts = ?, updated_at = ?
WHERE id = ?
`
	if s.dialect == "postgres" {
		query = `
UPDATE tasks
SET artifacts = $1, updated_at = $2
WHERE id = $3
`
	}

	_, err = s.db.ExecContext(ctx, query, row.Artifacts, row.UpdatedAt, taskID)
	if err != nil {
		return fmt.Errorf("failed to add artifact: %w", err)
	}

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

func (s *SQLTaskService) AddTaskMessage(ctx context.Context, taskID string, message *pb.Message) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if message.ContextId == "" {
		message.ContextId = task.ContextId
	}
	if message.TaskId == "" {
		message.TaskId = taskID
	}

	task.History = append(task.History, message)

	row, err := s.taskToRow(task)
	if err != nil {
		return fmt.Errorf("failed to serialize task: %w", err)
	}

	query := `
UPDATE tasks
SET history = ?, updated_at = ?
WHERE id = ?
`
	if s.dialect == "postgres" {
		query = `
UPDATE tasks
SET history = $1, updated_at = $2
WHERE id = $3
`
	}

	_, err = s.db.ExecContext(ctx, query, row.History, row.UpdatedAt, taskID)
	if err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	return nil
}

func (s *SQLTaskService) CancelTask(ctx context.Context, taskID string) (*pb.Task, error) {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if isTerminalState(task.Status.State) {
		return nil, fmt.Errorf("cannot cancel task in terminal state: %s", task.Status.State)
	}

	err = s.UpdateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_CANCELLED, nil)
	if err != nil {
		return nil, err
	}

	return s.GetTask(ctx, taskID)
}

func (s *SQLTaskService) ListTasksByContext(ctx context.Context, contextID string) ([]*pb.Task, error) {
	query := `
SELECT id, context_id, state, status_json, artifacts, history, metadata, created_at, updated_at
FROM tasks
WHERE context_id = ?
ORDER BY created_at DESC
`
	if s.dialect == "postgres" {
		query = `
SELECT id, context_id, state, status_json, artifacts, history, metadata, created_at, updated_at
FROM tasks
WHERE context_id = $1
ORDER BY created_at DESC
`
	}

	rows, err := s.db.QueryContext(ctx, query, contextID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*pb.Task
	for rows.Next() {
		var row taskRow
		err := rows.Scan(
			&row.ID, &row.ContextID, &row.State, &row.StatusJSON,
			&row.Artifacts, &row.History, &row.Metadata,
			&row.CreatedAt, &row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		task, err := s.rowToTask(&row)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize task: %w", err)
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *SQLTaskService) SubscribeToTask(ctx context.Context, taskID string) (<-chan *pb.StreamResponse, error) {

	_, err := s.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	ch := make(chan *pb.StreamResponse, 10)

	s.subscribersMu.Lock()
	s.subscribers[taskID] = append(s.subscribers[taskID], ch)
	s.subscribersMu.Unlock()

	return ch, nil
}

func (s *SQLTaskService) Close() error {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for taskID := range s.subscribers {
		s.closeTaskSubscribers(taskID)
	}

	return s.db.Close()
}

func (s *SQLTaskService) notifySubscribers(taskID string, event *pb.StreamResponse) {
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

func (s *SQLTaskService) closeTaskSubscribers(taskID string) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for _, ch := range s.subscribers[taskID] {
		close(ch)
	}
	delete(s.subscribers, taskID)
}

func (s *SQLTaskService) taskToRow(task *pb.Task) (*taskRow, error) {
	now := time.Now()

	statusJSON, err := protojson.Marshal(task.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status: %w", err)
	}

	var artifactsData []byte
	if len(task.Artifacts) > 0 {

		rawArtifacts := make([]json.RawMessage, len(task.Artifacts))
		for i, artifact := range task.Artifacts {
			artJSON, err := protojson.Marshal(artifact)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal artifact %d: %w", i, err)
			}
			rawArtifacts[i] = json.RawMessage(artJSON)
		}
		var err error
		artifactsData, err = json.Marshal(rawArtifacts)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal artifacts array: %w", err)
		}
	} else {
		artifactsData = []byte("[]")
	}

	var historyData []byte
	if len(task.History) > 0 {

		rawMessages := make([]json.RawMessage, len(task.History))
		for i, msg := range task.History {
			msgJSON, err := protojson.Marshal(msg)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal message %d: %w", i, err)
			}
			rawMessages[i] = json.RawMessage(msgJSON)
		}
		var err error
		historyData, err = json.Marshal(rawMessages)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal history array: %w", err)
		}
	} else {
		historyData = []byte("[]")
	}

	metadataData, err := json.Marshal(task.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return &taskRow{
		ID:         task.Id,
		ContextID:  task.ContextId,
		State:      task.Status.State.String(),
		StatusJSON: string(statusJSON),
		Artifacts:  string(artifactsData),
		History:    string(historyData),
		Metadata:   string(metadataData),
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (s *SQLTaskService) rowToTask(row *taskRow) (*pb.Task, error) {
	task := &pb.Task{
		Id:        row.ID,
		ContextId: row.ContextID,
	}

	if row.StatusJSON != "" {
		task.Status = &pb.TaskStatus{}
		err := protojson.Unmarshal([]byte(row.StatusJSON), task.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal status: %w", err)
		}
	}

	if row.Artifacts != "" && row.Artifacts != "[]" {

		var rawArtifacts []json.RawMessage
		err := json.Unmarshal([]byte(row.Artifacts), &rawArtifacts)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal artifacts array: %w", err)
		}

		task.Artifacts = make([]*pb.Artifact, len(rawArtifacts))
		for i, raw := range rawArtifacts {
			artifact := &pb.Artifact{}
			err := protojson.Unmarshal(raw, artifact)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal artifact %d: %w", i, err)
			}
			task.Artifacts[i] = artifact
		}
	} else {
		task.Artifacts = make([]*pb.Artifact, 0)
	}

	if row.History != "" && row.History != "[]" {

		var rawMessages []json.RawMessage
		err := json.Unmarshal([]byte(row.History), &rawMessages)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal history array: %w", err)
		}

		task.History = make([]*pb.Message, len(rawMessages))
		for i, raw := range rawMessages {
			message := &pb.Message{}
			err := protojson.Unmarshal(raw, message)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal message %d: %w", i, err)
			}
			task.History[i] = message
		}
	} else {
		task.History = make([]*pb.Message, 0)
	}

	if row.Metadata != "" && row.Metadata != "{}" {
		err := json.Unmarshal([]byte(row.Metadata), &task.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return task, nil
}
