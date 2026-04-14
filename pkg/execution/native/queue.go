// Package native provides the native execution provider implementation.
//
// The native provider implements execution.Provider using a local worker pool
// and SQL-backed queue for durable task execution. Features include:
//   - Persistent task queue backed by SQL database
//   - Configurable retry policies with exponential backoff
//   - Dead-letter queue for failed tasks
//   - Worker pool for controlled concurrency
package native

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Queue abstracts task queueing for resilience and durability.
//
// The queue provides at-least-once delivery semantics:
//   - Tasks are persisted before acknowledgement
//   - Failed tasks are retried according to RetryPolicy
//   - Tasks exceeding max retries are moved to dead-letter queue
type Queue interface {
	// Enqueue adds a task to the queue for processing.
	// Returns immediately - task will be processed by a worker.
	Enqueue(ctx context.Context, item *QueueItem) error

	// Dequeue retrieves the next task for processing.
	// The item is atomically marked as "processing".
	// Blocks until a task is available or context is cancelled.
	Dequeue(ctx context.Context) (*QueueItem, error)

	// Ack acknowledges successful task completion.
	// The task is marked as completed.
	Ack(ctx context.Context, itemID string) error

	// Nack negatively acknowledges - task failed, may retry.
	// If retry_count >= max_retries, task moves to dead-letter queue.
	Nack(ctx context.Context, itemID string, err error) error

	// Requeue moves a dead-letter item back to pending queue.
	Requeue(ctx context.Context, itemID string) error

	// RecoverStale finds items stuck in "processing" state and resets them.
	// Called on startup to handle items from crashed workers.
	RecoverStale(ctx context.Context, staleThreshold time.Duration) error

	// Stats returns queue statistics for an app.
	Stats(ctx context.Context, appName string) (*QueueStats, error)

	// ListDLQ returns items in dead-letter queue.
	ListDLQ(ctx context.Context, appName string, limit int) ([]*QueueItem, error)

	// Close gracefully shuts down the queue.
	Close() error
}

// QueueItemStatus represents the state of a queue item.
type QueueItemStatus string

const (
	// QueueItemPending - waiting to be processed.
	QueueItemPending QueueItemStatus = "pending"

	// QueueItemProcessing - currently being processed by a worker.
	QueueItemProcessing QueueItemStatus = "processing"

	// QueueItemCompleted - successfully completed.
	QueueItemCompleted QueueItemStatus = "completed"

	// QueueItemFailed - failed but may be retried.
	QueueItemFailed QueueItemStatus = "failed"

	// QueueItemDead - exceeded max retries, in dead-letter queue.
	QueueItemDead QueueItemStatus = "dead"
)

// QueueItem represents a queued task.
type QueueItem struct {
	// ID is the unique identifier for this queue item.
	ID string `json:"id"`

	// TaskID links to the a2a_tasks table for result storage.
	TaskID string `json:"task_id"`

	// AppName identifies the app (multi-tenant isolation).
	AppName string `json:"app_name"`

	// AgentName is the agent to invoke.
	AgentName string `json:"agent_name"`

	// Input is the user input/prompt for the agent.
	Input string `json:"input"`

	// ContextID is the session ID for conversation continuity.
	ContextID string `json:"context_id"`

	// UserID is the user identifier.
	UserID string `json:"user_id"`

	// Metadata contains additional task data.
	Metadata map[string]any `json:"metadata,omitempty"`

	// Priority affects dequeue order (higher = sooner).
	Priority int `json:"priority"`

	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int `json:"max_retries"`

	// RetryCount is the current number of retry attempts.
	RetryCount int `json:"retry_count"`

	// Status is the current state of the queue item.
	Status QueueItemStatus `json:"status"`

	// EnqueuedAt is when the item was added to the queue.
	EnqueuedAt time.Time `json:"enqueued_at"`

	// ScheduledFor is when the item should be processed (for delayed execution).
	ScheduledFor time.Time `json:"scheduled_for"`

	// DeadlineAt is the deadline for task completion.
	DeadlineAt time.Time `json:"deadline_at"`

	// StartedAt is when processing started.
	StartedAt *time.Time `json:"started_at,omitempty"`

	// CompletedAt is when processing completed.
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// LastError is the most recent error message.
	LastError string `json:"last_error,omitempty"`

	// WorkerID is the ID of the worker processing this item.
	WorkerID string `json:"worker_id,omitempty"`
}

// NewQueueItem creates a new queue item with sensible defaults.
func NewQueueItem(appName, agentName, input string) *QueueItem {
	now := time.Now()
	return &QueueItem{
		ID:           uuid.New().String(),
		TaskID:       uuid.New().String(),
		AppName:      appName,
		AgentName:    agentName,
		Input:        input,
		Priority:     0,
		MaxRetries:   3,
		RetryCount:   0,
		Status:       QueueItemPending,
		EnqueuedAt:   now,
		ScheduledFor: now,
		Metadata:     make(map[string]any),
	}
}

// WithContextID sets the session ID for conversation continuity.
func (i *QueueItem) WithContextID(contextID string) *QueueItem {
	i.ContextID = contextID
	return i
}

// WithUserID sets the user identifier.
func (i *QueueItem) WithUserID(userID string) *QueueItem {
	i.UserID = userID
	return i
}

// WithPriority sets the priority (higher = processed sooner).
func (i *QueueItem) WithPriority(priority int) *QueueItem {
	i.Priority = priority
	return i
}

// WithMaxRetries sets the maximum retry attempts.
func (i *QueueItem) WithMaxRetries(maxRetries int) *QueueItem {
	i.MaxRetries = maxRetries
	return i
}

// WithDeadline sets when the task must complete by.
func (i *QueueItem) WithDeadline(deadline time.Time) *QueueItem {
	i.DeadlineAt = deadline
	return i
}

// WithScheduledFor sets when the task should be processed.
func (i *QueueItem) WithScheduledFor(scheduledFor time.Time) *QueueItem {
	i.ScheduledFor = scheduledFor
	return i
}

// WithMetadata sets additional metadata.
func (i *QueueItem) WithMetadata(key string, value any) *QueueItem {
	if i.Metadata == nil {
		i.Metadata = make(map[string]any)
	}
	i.Metadata[key] = value
	return i
}

// SerializeMetadata converts metadata to JSON.
func (i *QueueItem) SerializeMetadata() (string, error) {
	if len(i.Metadata) == 0 {
		return "{}", nil
	}
	data, err := json.Marshal(i.Metadata)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DeserializeMetadata parses metadata from JSON.
func (i *QueueItem) DeserializeMetadata(data string) error {
	if data == "" || data == "{}" {
		i.Metadata = make(map[string]any)
		return nil
	}
	return json.Unmarshal([]byte(data), &i.Metadata)
}

// InjectContext injects the current context's trace information into the item's metadata.
// This allows the trace to be propagated across the queue boundary.
func (i *QueueItem) InjectContext(ctx context.Context) {
	if i.Metadata == nil {
		i.Metadata = make(map[string]any)
	}

	// Create a map carrier to hold the propagation headers
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	// Store carrier in metadata under a reserved key
	i.Metadata["_trace_context"] = carrier
}

// ExtractContext extracts trace information from the item's metadata and returns a new context.
// This restores the trace context from the queue item.
func (i *QueueItem) ExtractContext(ctx context.Context) context.Context {
	if i.Metadata == nil {
		return ctx
	}

	// Retrieve the carrier from metadata
	val, ok := i.Metadata["_trace_context"]
	if !ok {
		return ctx
	}

	// Helper to safely convert map[string]any to map[string]string for MapCarrier
	// Since JSON unmarshalling might give us map[string]any
	rawMap, ok := val.(map[string]any)
	if !ok {
		// Handle the case where it's already a MapCarrier (e.g. in-memory or test)
		if carrier, ok := val.(propagation.MapCarrier); ok {
			return otel.GetTextMapPropagator().Extract(ctx, carrier)
		}
		// Handle plain map[string]string
		if strMap, ok := val.(map[string]string); ok {
			return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(strMap))
		}
		// If it's a proxy from JSON (e.g. from DB), it might be map[string]interface{}
		return ctx
	}

	// Convert map[string]any to map[string]string
	carrier := make(propagation.MapCarrier)
	for k, v := range rawMap {
		if strVal, ok := v.(string); ok {
			carrier[k] = strVal
		}
	}

	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// IsExpired returns true if the item has passed its deadline.
func (i *QueueItem) IsExpired() bool {
	if i.DeadlineAt.IsZero() {
		return false
	}
	return time.Now().After(i.DeadlineAt)
}

// IsScheduled returns true if the item is scheduled for future processing.
func (i *QueueItem) IsScheduled() bool {
	if i.ScheduledFor.IsZero() {
		return false
	}
	return time.Now().Before(i.ScheduledFor)
}

// QueueStats contains queue statistics.
type QueueStats struct {
	// Pending is the count of items waiting to be processed.
	Pending int64 `json:"pending"`

	// Processing is the count of items currently being processed.
	Processing int64 `json:"processing"`

	// Completed is the count of successfully completed items.
	Completed int64 `json:"completed"`

	// Failed is the count of failed items (may be retried).
	Failed int64 `json:"failed"`

	// DeadLetter is the count of items in dead-letter queue.
	DeadLetter int64 `json:"dead_letter"`

	// OldestPending is the age of the oldest pending item.
	OldestPending time.Duration `json:"oldest_pending"`

	// AvgProcessingTime is the average processing time for completed items.
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
}

// RetryPolicy configures retry behavior.
type RetryPolicy struct {
	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int `yaml:"max_retries" json:"max_retries"`

	// InitialDelay is the delay before the first retry.
	InitialDelay time.Duration `yaml:"initial_delay" json:"initial_delay"`

	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration `yaml:"max_delay" json:"max_delay"`

	// BackoffFactor is the multiplier for exponential backoff.
	BackoffFactor float64 `yaml:"backoff_factor" json:"backoff_factor"`
}

// SetDefaults applies default values to the retry policy.
func (p *RetryPolicy) SetDefaults() {
	if p.MaxRetries == 0 {
		p.MaxRetries = 3
	}
	if p.InitialDelay == 0 {
		p.InitialDelay = 1 * time.Second
	}
	if p.MaxDelay == 0 {
		p.MaxDelay = 5 * time.Minute
	}
	if p.BackoffFactor == 0 {
		p.BackoffFactor = 2.0
	}
}

// NextDelay calculates the delay before the next retry attempt.
func (p *RetryPolicy) NextDelay(attempt int) time.Duration {
	delay := float64(p.InitialDelay) * math.Pow(p.BackoffFactor, float64(attempt))
	if time.Duration(delay) > p.MaxDelay {
		return p.MaxDelay
	}
	return time.Duration(delay)
}

// ShouldRetry returns true if another retry attempt should be made.
func (p *RetryPolicy) ShouldRetry(retryCount int) bool {
	return retryCount < p.MaxRetries
}

// DefaultRetryPolicy returns a retry policy with sensible defaults.
func DefaultRetryPolicy() *RetryPolicy {
	p := &RetryPolicy{}
	p.SetDefaults()
	return p
}
