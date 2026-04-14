package native

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/google/uuid"

	"github.com/verikod/hector/pkg/app"
	"github.com/verikod/hector/pkg/observability"
	"github.com/verikod/hector/pkg/task"
)

// ExecutorProvider provides executors for app/agent combinations.
// Implemented by server.AppManager.
type ExecutorProvider interface {
	// GetExecutor returns an executor for the given app and agent.
	GetExecutor(ctx context.Context, appID, agentName string) (a2asrv.AgentExecutor, error)
}

// WorkerPoolOptions configures the worker pool.
type WorkerPoolOptions struct {
	// NumWorkers is the number of concurrent workers.
	// Default: 4
	NumWorkers int

	// ShutdownTimeout is how long to wait for in-flight tasks during shutdown.
	// Default: 30 seconds
	ShutdownTimeout time.Duration

	// StaleRecoveryInterval is how often to check for stale "processing" items.
	// Default: 1 minute
	StaleRecoveryInterval time.Duration

	// StaleThreshold is how old a "processing" item must be to be considered stale.
	// Default: 5 minutes
	StaleThreshold time.Duration
}

// SetDefaults applies default values.
func (o *WorkerPoolOptions) SetDefaults() {
	if o.NumWorkers <= 0 {
		o.NumWorkers = 4
	}
	if o.ShutdownTimeout <= 0 {
		o.ShutdownTimeout = 30 * time.Second
	}
	if o.StaleRecoveryInterval <= 0 {
		o.StaleRecoveryInterval = 1 * time.Minute
	}
	if o.StaleThreshold <= 0 {
		o.StaleThreshold = 5 * time.Minute
	}
}

// WorkerPool manages concurrent task processing from the queue.
//
// Workers dequeue items, invoke the appropriate executor, and handle
// success/failure acknowledgements including retry logic.
type WorkerPool struct {
	queue            Queue
	executorProvider ExecutorProvider
	taskService      task.Service
	options          WorkerPoolOptions
	workerID         string
	metrics          *observability.Metrics

	wg       sync.WaitGroup
	stopCh   chan struct{}
	stopped  atomic.Bool
	started  atomic.Bool
	cancelFn context.CancelFunc
}

// NewWorkerPool creates a new worker pool.
func NewWorkerPool(queue Queue, executorProvider ExecutorProvider, taskService task.Service, metrics *observability.Metrics, opts WorkerPoolOptions) *WorkerPool {
	opts.SetDefaults()

	return &WorkerPool{
		queue:            queue,
		executorProvider: executorProvider,
		taskService:      taskService,
		options:          opts,
		workerID:         uuid.New().String(),
		stopCh:           make(chan struct{}),
		metrics:          metrics,
	}
}

// Start begins processing tasks from the queue.
// This method is non-blocking - workers run in goroutines.
func (p *WorkerPool) Start(ctx context.Context) error {
	if !p.started.CompareAndSwap(false, true) {
		return errors.New("worker pool already started")
	}

	// Create cancellable context for workers
	workerCtx, cancel := context.WithCancel(ctx)
	p.cancelFn = cancel

	// Start recovery goroutine
	p.wg.Add(1)
	go p.recoveryLoop(workerCtx)

	// Start worker goroutines
	for i := 0; i < p.options.NumWorkers; i++ {
		p.wg.Add(1)
		go p.worker(workerCtx, i)
	}

	slog.Info("Worker pool started",
		"workers", p.options.NumWorkers,
		"worker_id", p.workerID)

	return nil
}

// Stop gracefully shuts down the worker pool.
// It signals workers to stop and waits for in-flight tasks to complete.
func (p *WorkerPool) Stop() error {
	if !p.stopped.CompareAndSwap(false, true) {
		return errors.New("worker pool already stopped")
	}

	slog.Info("Stopping worker pool", "worker_id", p.workerID)

	// Signal workers to stop
	close(p.stopCh)
	if p.cancelFn != nil {
		p.cancelFn()
	}

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Worker pool stopped gracefully", "worker_id", p.workerID)
		return nil
	case <-time.After(p.options.ShutdownTimeout):
		slog.Warn("Worker pool shutdown timed out",
			"worker_id", p.workerID,
			"timeout", p.options.ShutdownTimeout)
		return errors.New("shutdown timeout: some workers still processing")
	}
}

// worker is the main loop for a worker goroutine.
func (p *WorkerPool) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	slog.Debug("Worker started", "worker_id", p.workerID, "worker_num", id)

	for {
		select {
		case <-ctx.Done():
			slog.Debug("Worker stopped (context cancelled)", "worker_id", p.workerID, "worker_num", id)
			return
		case <-p.stopCh:
			slog.Debug("Worker stopped (shutdown signal)", "worker_id", p.workerID, "worker_num", id)
			return
		default:
		}

		item, err := p.queue.Dequeue(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			slog.Debug("Dequeue error", "worker_id", p.workerID, "worker_num", id, "error", err)
			continue
		}

		if item == nil {
			// Should not happen, but be defensive
			continue
		}

		p.processItem(ctx, item, id)
	}
}

// processItem handles a single queue item.
func (p *WorkerPool) processItem(ctx context.Context, item *QueueItem, workerNum int) {
	// Restore trace context from item metadata
	// This links the async execution to the original request trace
	ctx = item.ExtractContext(ctx)

	startTime := time.Now()

	slog.Info("Processing task",
		"queue_id", item.ID,
		"task_id", item.TaskID,
		"app", item.AppName,
		"agent", item.AgentName,
		"retry", item.RetryCount,
		"worker_num", workerNum)

	// Check if expired
	if item.IsExpired() {
		slog.Warn("Task expired, moving to DLQ",
			"queue_id", item.ID,
			"task_id", item.TaskID,
			"deadline", item.DeadlineAt)
		if err := p.queue.Nack(ctx, item.ID, errors.New("task deadline exceeded")); err != nil {
			slog.Warn("Failed to Nack expired task", "queue_id", item.ID, "error", err)
		}
		return
	}

	// Apply deadline if set
	taskCtx := ctx
	if !item.DeadlineAt.IsZero() {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithDeadline(ctx, item.DeadlineAt)
		defer cancel()
	}

	// Enrich context with app ID
	taskCtx = app.WithAppID(taskCtx, item.AppName)

	// Execute the task
	err := p.executeTask(taskCtx, item)

	// Calculate queue latency: Time from enqueue to processing start
	if !item.EnqueuedAt.IsZero() {
		queueLatency := startTime.Sub(item.EnqueuedAt)
		p.metrics.RecordQueueLatency(item.AppName, item.AgentName, queueLatency)
	}

	// Calculate processing duration
	duration := time.Since(startTime)
	p.metrics.RecordQueueProcessingDuration(item.AppName, item.AgentName, duration)

	if err != nil {
		slog.Warn("Task failed",
			"queue_id", item.ID,
			"task_id", item.TaskID,
			"app", item.AppName,
			"agent", item.AgentName,
			"retry", item.RetryCount,
			"duration", duration,
			"error", err)
		if err := p.queue.Nack(ctx, item.ID, err); err != nil {
			slog.Warn("Failed to Nack failed task", "queue_id", item.ID, "error", err)
		}
	} else {
		slog.Info("Task completed",
			"queue_id", item.ID,
			"task_id", item.TaskID,
			"app", item.AppName,
			"agent", item.AgentName,
			"duration", duration)
		if err := p.queue.Ack(ctx, item.ID); err != nil {
			slog.Warn("Failed to Ack completed task", "queue_id", item.ID, "error", err)
		}
	}
}

// executeTask invokes the executor for a queue item.
func (p *WorkerPool) executeTask(ctx context.Context, item *QueueItem) error {
	// Get executor for this app/agent
	executor, err := p.executorProvider.GetExecutor(ctx, item.AppName, item.AgentName)
	if err != nil {
		return fmt.Errorf("failed to get executor: %w", err)
	}

	// Build A2A request context
	reqCtx := &a2asrv.RequestContext{
		TaskID:    a2a.TaskID(item.TaskID),
		ContextID: item.ContextID,
		Message:   p.buildMessage(item),
		Metadata: map[string]any{
			"hector:source":   "queue",
			"hector:queue_id": item.ID,
			"hector:user_id":  item.UserID,
		},
	}

	// Use a collecting event handler (we don't stream to client in worker mode)
	eventCollector := NewCollectingEventQueue()

	// Execute
	if err := executor.Execute(ctx, reqCtx, eventCollector); err != nil {
		return fmt.Errorf("executor failed: %w", err)
	}

	// Check collected events for failures
	// In worker mode, we don't stream to a client,
	// but we need to check if the task failed
	for _, event := range eventCollector.Events() {

		// Check for failure events
		if statusEvent, ok := event.(*a2a.TaskStatusUpdateEvent); ok {
			if statusEvent.Status.State == a2a.TaskStateFailed {
				errMsg := "task failed"
				if statusEvent.Status.Message != nil {
					for _, part := range statusEvent.Status.Message.Parts {
						if textPart, ok := part.(a2a.TextPart); ok {
							errMsg = textPart.Text
							break
						}
					}
				}
				return errors.New(errMsg)
			}
		}
	}

	return nil
}

// buildMessage creates an A2A message from a queue item.
func (p *WorkerPool) buildMessage(item *QueueItem) *a2a.Message {
	return &a2a.Message{
		Role: a2a.MessageRoleUser,
		Parts: []a2a.Part{
			a2a.TextPart{Text: item.Input},
		},
		Metadata: item.Metadata,
	}
}

// recoveryLoop periodically checks for stale items.
func (p *WorkerPool) recoveryLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.options.StaleRecoveryInterval)
	defer ticker.Stop()

	// Run immediately on startup
	if err := p.queue.RecoverStale(ctx, p.options.StaleThreshold); err != nil {
		slog.Warn("Failed to recover stale items on startup", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			if err := p.queue.RecoverStale(ctx, p.options.StaleThreshold); err != nil {
				slog.Warn("Failed to recover stale items", "error", err)
			}
		}
	}
}

// Stats returns current worker pool statistics.
func (p *WorkerPool) Stats() WorkerPoolStats {
	return WorkerPoolStats{
		WorkerID:   p.workerID,
		NumWorkers: p.options.NumWorkers,
		Started:    p.started.Load(),
		Stopped:    p.stopped.Load(),
	}
}

// WorkerPoolStats contains worker pool statistics.
type WorkerPoolStats struct {
	WorkerID   string `json:"worker_id"`
	NumWorkers int    `json:"num_workers"`
	Started    bool   `json:"started"`
	Stopped    bool   `json:"stopped"`
}
