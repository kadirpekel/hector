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

	// Database drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// SQLTaskService implements reasoning.TaskService with SQL backend
// Supports PostgreSQL, MySQL, and SQLite via database/sql
type SQLTaskService struct {
	db            *sql.DB
	dialect       string // "postgres", "mysql", or "sqlite"
	subscribers   map[string][]chan *pb.StreamResponse
	subscribersMu sync.RWMutex
}

// taskRow represents the database schema for tasks
type taskRow struct {
	ID         string
	ContextID  string
	State      string
	StatusJSON string // JSON-encoded TaskStatus
	Artifacts  string // JSON-encoded []Artifact
	History    string // JSON-encoded []Message
	Metadata   string // JSON-encoded map[string]interface{}
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

const (
	// SQL schema (compatible with all three databases)
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

// NewSQLTaskService creates a new SQL-backed task service
func NewSQLTaskService(db *sql.DB, dialect string) (*SQLTaskService, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	// Validate dialect
	switch dialect {
	case "postgres", "mysql", "sqlite":
		// Valid
	default:
		return nil, fmt.Errorf("unsupported dialect: %s (supported: postgres, mysql, sqlite)", dialect)
	}

	s := &SQLTaskService{
		db:          db,
		dialect:     dialect,
		subscribers: make(map[string][]chan *pb.StreamResponse),
	}

	// Initialize schema
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

// NewSQLTaskServiceFromConfig creates a new SQL task service from configuration
func NewSQLTaskServiceFromConfig(cfg *config.TaskSQLConfig) (*SQLTaskService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("SQL configuration is required")
	}

	// Set defaults and validate
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Map driver name to actual SQL driver name
	// Config uses "sqlite" but the go-sqlite3 driver registers as "sqlite3"
	driverName := cfg.Driver
	if driverName == "sqlite" {
		driverName = "sqlite3"
	}

	// Open database connection
	db, err := sql.Open(driverName, cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create service
	return NewSQLTaskService(db, cfg.Driver)
}

// initSchema creates the tasks table if it doesn't exist
func (s *SQLTaskService) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// For SQLite, we need to adjust the create index syntax
	schemaSQL := createTableSQL
	// Note: All dialects use the same schema for now
	_ = s.dialect

	_, err := s.db.ExecContext(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// CreateTask implements reasoning.TaskService
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

	// Serialize to JSON
	row, err := s.taskToRow(task)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize task: %w", err)
	}

	// Insert into database
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

	// Notify subscribers
	s.notifySubscribers(taskID, &pb.StreamResponse{
		Payload: &pb.StreamResponse_Task{
			Task: task,
		},
	})

	return task, nil
}

// GetTask implements reasoning.TaskService
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

// UpdateTaskStatus implements reasoning.TaskService
func (s *SQLTaskService) UpdateTaskStatus(ctx context.Context, taskID string, state pb.TaskState, message *pb.Message) error {
	// First, get the current task
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	// Update status
	now := timestamppb.Now()
	task.Status = &pb.TaskStatus{
		State:     state,
		Update:    message,
		Timestamp: now,
	}

	// Serialize and update
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

	// Notify subscribers
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

// AddTaskArtifact implements reasoning.TaskService
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

	// Notify subscribers
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

// AddTaskMessage implements reasoning.TaskService
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

// CancelTask implements reasoning.TaskService
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

// ListTasksByContext implements reasoning.TaskService
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

// SubscribeToTask implements reasoning.TaskService
func (s *SQLTaskService) SubscribeToTask(ctx context.Context, taskID string) (<-chan *pb.StreamResponse, error) {
	// Verify task exists
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

// Close implements reasoning.TaskService
func (s *SQLTaskService) Close() error {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for taskID := range s.subscribers {
		s.closeTaskSubscribers(taskID)
	}

	return s.db.Close()
}

// notifySubscribers sends an event to all subscribers of a task
func (s *SQLTaskService) notifySubscribers(taskID string, event *pb.StreamResponse) {
	s.subscribersMu.RLock()
	subscribers := s.subscribers[taskID]
	s.subscribersMu.RUnlock()

	for _, ch := range subscribers {
		select {
		case ch <- event:
		default:
			// Skip if channel is full
		}
	}
}

// closeTaskSubscribers closes all subscriber channels for a task
func (s *SQLTaskService) closeTaskSubscribers(taskID string) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for _, ch := range s.subscribers[taskID] {
		close(ch)
	}
	delete(s.subscribers, taskID)
}

// taskToRow converts a pb.Task to a database row
func (s *SQLTaskService) taskToRow(task *pb.Task) (*taskRow, error) {
	now := time.Now()

	// Serialize status using protojson
	statusJSON, err := protojson.Marshal(task.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status: %w", err)
	}

	// Serialize artifacts using protojson (they are protobuf messages)
	var artifactsData []byte
	if len(task.Artifacts) > 0 {
		// Marshal each artifact to json.RawMessage and then marshal the array
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

	// Serialize history using protojson (they are protobuf messages)
	var historyData []byte
	if len(task.History) > 0 {
		// Marshal each message to json.RawMessage and then marshal the array
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

	// Serialize metadata
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

// rowToTask converts a database row to a pb.Task
func (s *SQLTaskService) rowToTask(row *taskRow) (*pb.Task, error) {
	task := &pb.Task{
		Id:        row.ID,
		ContextId: row.ContextID,
	}

	// Deserialize status
	if row.StatusJSON != "" {
		task.Status = &pb.TaskStatus{}
		err := protojson.Unmarshal([]byte(row.StatusJSON), task.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal status: %w", err)
		}
	}

	// Deserialize artifacts using protojson
	if row.Artifacts != "" && row.Artifacts != "[]" {
		// Parse JSON array manually to unmarshal each artifact
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

	// Deserialize history using protojson
	if row.History != "" && row.History != "[]" {
		// Parse JSON array manually to unmarshal each message
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

	// Deserialize metadata
	if row.Metadata != "" && row.Metadata != "{}" {
		err := json.Unmarshal([]byte(row.Metadata), &task.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return task, nil
}
