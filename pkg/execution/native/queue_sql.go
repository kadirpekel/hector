package native

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// SQLQueue implements Queue using SQL database.
//
// It uses SELECT FOR UPDATE SKIP LOCKED (PostgreSQL) or
// optimistic locking (SQLite/MySQL) for atomic dequeue operations.
type SQLQueue struct {
	db           *sql.DB
	dialect      string
	retryPolicy  *RetryPolicy
	pollInterval time.Duration
	workerID     string

	stopCh  chan struct{}
	stopped atomic.Bool
	mu      sync.Mutex
}

// SQLQueueOption configures SQLQueue.
type SQLQueueOption func(*SQLQueue)

// WithPollInterval sets how often to poll for new items.
func WithPollInterval(d time.Duration) SQLQueueOption {
	return func(q *SQLQueue) {
		q.pollInterval = d
	}
}

// WithWorkerID sets the worker ID for this queue instance.
func WithWorkerID(id string) SQLQueueOption {
	return func(q *SQLQueue) {
		q.workerID = id
	}
}

// WithRetryPolicy sets the retry policy.
func WithRetryPolicy(p *RetryPolicy) SQLQueueOption {
	return func(q *SQLQueue) {
		q.retryPolicy = p
	}
}

// NewSQLQueue creates a new SQL-backed queue.
func NewSQLQueue(db *sql.DB, dialect string, opts ...SQLQueueOption) (*SQLQueue, error) {
	if db == nil {
		return nil, errors.New("database connection is required")
	}

	// Normalize dialect
	normalizedDialect := dialect
	if dialect == "sqlite3" {
		normalizedDialect = "sqlite"
	}

	switch normalizedDialect {
	case "postgres", "mysql", "sqlite", "pgx":
		// Valid dialects
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}

	q := &SQLQueue{
		db:           db,
		dialect:      normalizedDialect,
		retryPolicy:  DefaultRetryPolicy(),
		pollInterval: 100 * time.Millisecond,
		workerID:     uuid.New().String(),
		stopCh:       make(chan struct{}),
	}

	for _, opt := range opts {
		opt(q)
	}

	if err := q.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize queue schema: %w", err)
	}

	return q, nil
}

// SQL statements for schema initialization.
const (
	createQueueTableSQL = `
CREATE TABLE IF NOT EXISTS task_queue (
    id VARCHAR(255) PRIMARY KEY,
    task_id VARCHAR(255) NOT NULL,
    app_name VARCHAR(255) NOT NULL,
    agent_name VARCHAR(255) NOT NULL,
    input TEXT NOT NULL,
    context_id VARCHAR(255),
    user_id VARCHAR(255),
    metadata_json TEXT,
    priority INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    retry_count INT DEFAULT 0,
    status VARCHAR(50) DEFAULT 'pending',
    enqueued_at TIMESTAMP NOT NULL,
    scheduled_for TIMESTAMP,
    deadline_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    last_error TEXT,
    worker_id VARCHAR(255)
)`

	createQueueDequeueIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_task_queue_dequeue 
    ON task_queue(app_name, status, priority DESC, scheduled_for, enqueued_at)`

	createQueueTaskIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_task_queue_task_id 
    ON task_queue(task_id)`

	createQueueStatusIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_task_queue_status 
    ON task_queue(status)`
)

func (q *SQLQueue) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create table
	if _, err := q.db.ExecContext(ctx, createQueueTableSQL); err != nil {
		return fmt.Errorf("failed to create task_queue table: %w", err)
	}

	// Create indexes (separate statements for SQLite compatibility)
	if _, err := q.db.ExecContext(ctx, createQueueDequeueIndexSQL); err != nil {
		return fmt.Errorf("failed to create dequeue index: %w", err)
	}

	if _, err := q.db.ExecContext(ctx, createQueueTaskIndexSQL); err != nil {
		return fmt.Errorf("failed to create task_id index: %w", err)
	}

	if _, err := q.db.ExecContext(ctx, createQueueStatusIndexSQL); err != nil {
		return fmt.Errorf("failed to create status index: %w", err)
	}

	return nil
}

// Enqueue adds a task to the queue.
func (q *SQLQueue) Enqueue(ctx context.Context, item *QueueItem) error {
	if item == nil {
		return errors.New("item is required")
	}

	// Inject trace context from current context into item metadata
	// This links the enqueue operation to the current trace
	item.InjectContext(ctx)

	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	if item.TaskID == "" {
		item.TaskID = uuid.New().String()
	}
	if item.Status == "" {
		item.Status = QueueItemPending
	}
	if item.EnqueuedAt.IsZero() {
		item.EnqueuedAt = time.Now()
	}
	if item.ScheduledFor.IsZero() {
		item.ScheduledFor = item.EnqueuedAt
	}

	metadataJSON, err := item.SerializeMetadata()
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	query := q.enqueueQuery()
	args := []any{
		item.ID,
		item.TaskID,
		item.AppName,
		item.AgentName,
		item.Input,
		item.ContextID,
		item.UserID,
		metadataJSON,
		item.Priority,
		item.MaxRetries,
		item.RetryCount,
		string(item.Status),
		item.EnqueuedAt,
		item.ScheduledFor,
		nullTime(item.DeadlineAt),
	}

	_, err = q.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to enqueue item: %w", err)
	}

	slog.Info("Enqueued task",
		"queue_id", item.ID,
		"task_id", item.TaskID,
		"app", item.AppName,
		"agent", item.AgentName)

	return nil
}

func (q *SQLQueue) enqueueQuery() string {
	if q.dialect == "postgres" || q.dialect == "pgx" {
		return `
INSERT INTO task_queue (
    id, task_id, app_name, agent_name, input, context_id, user_id,
    metadata_json, priority, max_retries, retry_count, status,
    enqueued_at, scheduled_for, deadline_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	}
	return `
INSERT INTO task_queue (
    id, task_id, app_name, agent_name, input, context_id, user_id,
    metadata_json, priority, max_retries, retry_count, status,
    enqueued_at, scheduled_for, deadline_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
}

// Dequeue retrieves the next task for processing.
func (q *SQLQueue) Dequeue(ctx context.Context) (*QueueItem, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-q.stopCh:
			return nil, errors.New("queue stopped")
		default:
		}

		item, err := q.tryDequeue(ctx)
		if err != nil {
			return nil, err
		}
		if item != nil {
			return item, nil
		}

		// No item available, poll after interval
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-q.stopCh:
			return nil, errors.New("queue stopped")
		case <-time.After(q.pollInterval):
		}
	}
}

func (q *SQLQueue) tryDequeue(ctx context.Context) (*QueueItem, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()

	// For PostgreSQL, use SELECT FOR UPDATE SKIP LOCKED
	// For SQLite, use a transaction with immediate locking
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Warn("Failed to rollback transaction", "error", err)
		}
	}()

	// Select the next item
	var item QueueItem
	var metadataJSON sql.NullString
	var scheduledFor, deadlineAt, startedAt, completedAt sql.NullTime
	var contextID, userID, lastError, workerID sql.NullString

	selectQuery := q.dequeueSelectQuery()
	row := tx.QueryRowContext(ctx, selectQuery, string(QueueItemPending), now)

	err = row.Scan(
		&item.ID,
		&item.TaskID,
		&item.AppName,
		&item.AgentName,
		&item.Input,
		&contextID,
		&userID,
		&metadataJSON,
		&item.Priority,
		&item.MaxRetries,
		&item.RetryCount,
		&item.Status,
		&item.EnqueuedAt,
		&scheduledFor,
		&deadlineAt,
		&startedAt,
		&completedAt,
		&lastError,
		&workerID,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No items available
	}
	if err != nil {
		return nil, fmt.Errorf("failed to select item: %w", err)
	}

	// Populate nullable fields
	item.ContextID = contextID.String
	item.UserID = userID.String
	item.LastError = lastError.String
	item.WorkerID = workerID.String
	if scheduledFor.Valid {
		item.ScheduledFor = scheduledFor.Time
	}
	if deadlineAt.Valid {
		item.DeadlineAt = deadlineAt.Time
	}
	if startedAt.Valid {
		item.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	if metadataJSON.Valid {
		if err := item.DeserializeMetadata(metadataJSON.String); err != nil {
			slog.Warn("Failed to deserialize metadata", "id", item.ID, "error", err)
		}
	}

	// Update status to processing
	updateQuery := q.dequeueUpdateQuery()
	startTime := time.Now()
	_, err = tx.ExecContext(ctx, updateQuery, string(QueueItemProcessing), startTime, q.workerID, item.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update item status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	item.Status = QueueItemProcessing
	item.StartedAt = &startTime
	item.WorkerID = q.workerID

	return &item, nil
}

func (q *SQLQueue) dequeueSelectQuery() string {
	if q.dialect == "postgres" || q.dialect == "pgx" {
		return `
SELECT id, task_id, app_name, agent_name, input, context_id, user_id,
       metadata_json, priority, max_retries, retry_count, status,
       enqueued_at, scheduled_for, deadline_at, started_at, completed_at,
       last_error, worker_id
FROM task_queue
WHERE status = $1 AND (scheduled_for IS NULL OR scheduled_for <= $2)
ORDER BY priority DESC, enqueued_at ASC
LIMIT 1
FOR UPDATE SKIP LOCKED`
	}
	// SQLite/MySQL - no SKIP LOCKED, but we're already in a mutex
	return `
SELECT id, task_id, app_name, agent_name, input, context_id, user_id,
       metadata_json, priority, max_retries, retry_count, status,
       enqueued_at, scheduled_for, deadline_at, started_at, completed_at,
       last_error, worker_id
FROM task_queue
WHERE status = ? AND (scheduled_for IS NULL OR scheduled_for <= ?)
ORDER BY priority DESC, enqueued_at ASC
LIMIT 1`
}

func (q *SQLQueue) dequeueUpdateQuery() string {
	if q.dialect == "postgres" || q.dialect == "pgx" {
		return `UPDATE task_queue SET status = $1, started_at = $2, worker_id = $3 WHERE id = $4`
	}
	return `UPDATE task_queue SET status = ?, started_at = ?, worker_id = ? WHERE id = ?`
}

// Ack acknowledges successful task completion.
func (q *SQLQueue) Ack(ctx context.Context, itemID string) error {
	now := time.Now()

	query := q.ackQuery()
	result, err := q.db.ExecContext(ctx, query, string(QueueItemCompleted), now, itemID)
	if err != nil {
		return fmt.Errorf("failed to ack item: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("item not found: %s", itemID)
	}

	return nil
}

func (q *SQLQueue) ackQuery() string {
	if q.dialect == "postgres" || q.dialect == "pgx" {
		return `UPDATE task_queue SET status = $1, completed_at = $2 WHERE id = $3`
	}
	return `UPDATE task_queue SET status = ?, completed_at = ? WHERE id = ?`
}

// Nack negatively acknowledges a task failure.
func (q *SQLQueue) Nack(ctx context.Context, itemID string, errMsg error) error {
	// First, get current retry count
	var retryCount, maxRetries int
	getQuery := q.getRetryQuery()
	err := q.db.QueryRowContext(ctx, getQuery, itemID).Scan(&retryCount, &maxRetries)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}

	newRetryCount := retryCount + 1
	errorStr := ""
	if errMsg != nil {
		errorStr = errMsg.Error()
	}

	if newRetryCount >= maxRetries {
		// Move to dead-letter queue
		query := q.nackDeadQuery()
		_, err = q.db.ExecContext(ctx, query, string(QueueItemDead), newRetryCount, errorStr, time.Now(), itemID)
		if err != nil {
			return fmt.Errorf("failed to move item to DLQ: %w", err)
		}
		slog.Warn("Task moved to DLQ", "id", itemID, "retries", newRetryCount, "error", errorStr)
	} else {
		// Schedule for retry
		nextScheduled := time.Now().Add(q.retryPolicy.NextDelay(newRetryCount))
		query := q.nackRetryQuery()
		_, err = q.db.ExecContext(ctx, query, string(QueueItemPending), newRetryCount, errorStr, nextScheduled, itemID)
		if err != nil {
			return fmt.Errorf("failed to schedule retry: %w", err)
		}
		slog.Info("Task scheduled for retry",
			"id", itemID,
			"retry", newRetryCount,
			"next_at", nextScheduled,
			"error", errorStr)
	}

	return nil
}

func (q *SQLQueue) getRetryQuery() string {
	if q.dialect == "postgres" || q.dialect == "pgx" {
		return `SELECT retry_count, max_retries FROM task_queue WHERE id = $1`
	}
	return `SELECT retry_count, max_retries FROM task_queue WHERE id = ?`
}

func (q *SQLQueue) nackDeadQuery() string {
	if q.dialect == "postgres" || q.dialect == "pgx" {
		return `UPDATE task_queue SET status = $1, retry_count = $2, last_error = $3, completed_at = $4, worker_id = NULL WHERE id = $5`
	}
	return `UPDATE task_queue SET status = ?, retry_count = ?, last_error = ?, completed_at = ?, worker_id = NULL WHERE id = ?`
}

func (q *SQLQueue) nackRetryQuery() string {
	if q.dialect == "postgres" || q.dialect == "pgx" {
		return `UPDATE task_queue SET status = $1, retry_count = $2, last_error = $3, scheduled_for = $4, started_at = NULL, worker_id = NULL WHERE id = $5`
	}
	return `UPDATE task_queue SET status = ?, retry_count = ?, last_error = ?, scheduled_for = ?, started_at = NULL, worker_id = NULL WHERE id = ?`
}

// Requeue moves a dead-letter item back to pending.
func (q *SQLQueue) Requeue(ctx context.Context, itemID string) error {
	now := time.Now()

	query := q.requeueQuery()
	result, err := q.db.ExecContext(ctx, query, string(QueueItemPending), 0, now, itemID, string(QueueItemDead))
	if err != nil {
		return fmt.Errorf("failed to requeue item: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("item not found or not in DLQ: %s", itemID)
	}

	slog.Info("Requeued task from DLQ", "id", itemID)
	return nil
}

func (q *SQLQueue) requeueQuery() string {
	if q.dialect == "postgres" || q.dialect == "pgx" {
		return `UPDATE task_queue SET status = $1, retry_count = $2, scheduled_for = $3, last_error = NULL, started_at = NULL, completed_at = NULL, worker_id = NULL WHERE id = $4 AND status = $5`
	}
	return `UPDATE task_queue SET status = ?, retry_count = ?, scheduled_for = ?, last_error = NULL, started_at = NULL, completed_at = NULL, worker_id = NULL WHERE id = ? AND status = ?`
}

// RecoverStale resets items stuck in processing state.
func (q *SQLQueue) RecoverStale(ctx context.Context, staleThreshold time.Duration) error {
	cutoff := time.Now().Add(-staleThreshold)

	query := q.recoverStaleQuery()
	result, err := q.db.ExecContext(ctx, query, string(QueueItemPending), string(QueueItemProcessing), cutoff)
	if err != nil {
		return fmt.Errorf("failed to recover stale items: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		slog.Info("Recovered stale queue items", "count", rows, "threshold", staleThreshold)
	}

	return nil
}

func (q *SQLQueue) recoverStaleQuery() string {
	if q.dialect == "postgres" || q.dialect == "pgx" {
		return `UPDATE task_queue SET status = $1, started_at = NULL, worker_id = NULL WHERE status = $2 AND started_at < $3`
	}
	return `UPDATE task_queue SET status = ?, started_at = NULL, worker_id = NULL WHERE status = ? AND started_at < ?`
}

// Stats returns queue statistics.
func (q *SQLQueue) Stats(ctx context.Context, appName string) (*QueueStats, error) {
	stats := &QueueStats{}

	query := q.statsQuery()
	rows, err := q.db.QueryContext(ctx, query, appName)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan stats row: %w", err)
		}

		switch QueueItemStatus(status) {
		case QueueItemPending:
			stats.Pending = count
		case QueueItemProcessing:
			stats.Processing = count
		case QueueItemCompleted:
			stats.Completed = count
		case QueueItemFailed:
			stats.Failed = count
		case QueueItemDead:
			stats.DeadLetter = count
		}
	}

	return stats, nil
}

func (q *SQLQueue) statsQuery() string {
	if q.dialect == "postgres" || q.dialect == "pgx" {
		return `SELECT status, COUNT(*) FROM task_queue WHERE app_name = $1 GROUP BY status`
	}
	return `SELECT status, COUNT(*) FROM task_queue WHERE app_name = ? GROUP BY status`
}

// ListDLQ returns items in the dead-letter queue.
func (q *SQLQueue) ListDLQ(ctx context.Context, appName string, limit int) ([]*QueueItem, error) {
	if limit <= 0 {
		limit = 100
	}

	query := q.listDLQQuery()
	rows, err := q.db.QueryContext(ctx, query, appName, string(QueueItemDead), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list DLQ: %w", err)
	}
	defer rows.Close()

	var items []*QueueItem
	for rows.Next() {
		item, err := q.scanQueueItem(rows)
		if err != nil {
			slog.Warn("Failed to scan DLQ item", "error", err)
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

func (q *SQLQueue) listDLQQuery() string {
	if q.dialect == "postgres" || q.dialect == "pgx" {
		return `
SELECT id, task_id, app_name, agent_name, input, context_id, user_id,
       metadata_json, priority, max_retries, retry_count, status,
       enqueued_at, scheduled_for, deadline_at, started_at, completed_at,
       last_error, worker_id
FROM task_queue
WHERE app_name = $1 AND status = $2
ORDER BY enqueued_at DESC
LIMIT $3`
	}
	return `
SELECT id, task_id, app_name, agent_name, input, context_id, user_id,
       metadata_json, priority, max_retries, retry_count, status,
       enqueued_at, scheduled_for, deadline_at, started_at, completed_at,
       last_error, worker_id
FROM task_queue
WHERE app_name = ? AND status = ?
ORDER BY enqueued_at DESC
LIMIT ?`
}

func (q *SQLQueue) scanQueueItem(rows *sql.Rows) (*QueueItem, error) {
	var item QueueItem
	var metadataJSON sql.NullString
	var scheduledFor, deadlineAt, startedAt, completedAt sql.NullTime
	var contextID, userID, lastError, workerID sql.NullString
	var status string

	err := rows.Scan(
		&item.ID,
		&item.TaskID,
		&item.AppName,
		&item.AgentName,
		&item.Input,
		&contextID,
		&userID,
		&metadataJSON,
		&item.Priority,
		&item.MaxRetries,
		&item.RetryCount,
		&status,
		&item.EnqueuedAt,
		&scheduledFor,
		&deadlineAt,
		&startedAt,
		&completedAt,
		&lastError,
		&workerID,
	)
	if err != nil {
		return nil, err
	}

	item.Status = QueueItemStatus(status)
	item.ContextID = contextID.String
	item.UserID = userID.String
	item.LastError = lastError.String
	item.WorkerID = workerID.String
	if scheduledFor.Valid {
		item.ScheduledFor = scheduledFor.Time
	}
	if deadlineAt.Valid {
		item.DeadlineAt = deadlineAt.Time
	}
	if startedAt.Valid {
		item.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	if metadataJSON.Valid {
		if err := item.DeserializeMetadata(metadataJSON.String); err != nil {
			slog.Warn("Failed to deserialize metadata in scan", "id", item.ID, "error", err)
		}
	}

	return &item, nil
}

// Close gracefully shuts down the queue.
func (q *SQLQueue) Close() error {
	if q.stopped.CompareAndSwap(false, true) {
		close(q.stopCh)
	}
	return nil
}

// Helper to convert zero time to NULL
func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// Compile-time interface check
var _ Queue = (*SQLQueue)(nil)
