package native

import (
	"context"
	"fmt"
	"time"

	"github.com/verikod/hector/pkg/config"
	"github.com/verikod/hector/pkg/execution"
	"github.com/verikod/hector/pkg/observability"
	"github.com/verikod/hector/pkg/task"
)

// NativeProvider implements execution.Provider using a local worker pool and SQL queue.
type NativeProvider struct {
	queue      Queue
	workerPool *WorkerPool
}

// NewProvider creates a new NativeProvider.
func NewProvider(
	pool *config.DBPool,
	dsn string,
	retryPolicy *RetryPolicy,
	executorProvider ExecutorProvider,
	taskService task.Service,
	metrics *observability.Metrics,
	opts WorkerPoolOptions,
) (*NativeProvider, error) {
	// 1. Create Queue
	queue, err := newQueue(pool, dsn, retryPolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to create queue: %w", err)
	}

	// 2. Create WorkerPool
	workerPool := NewWorkerPool(queue, executorProvider, taskService, metrics, opts)

	return &NativeProvider{
		queue:      queue,
		workerPool: workerPool,
	}, nil
}

// Submit enqueues a task for execution.
func (p *NativeProvider) Submit(ctx context.Context, req *execution.Request) error {
	item := &QueueItem{
		// Generate unique queue item ID (distinct from TaskID to allow task resubmission)
		ID:         fmt.Sprintf("native-%s-%d", req.AgentName, time.Now().UnixNano()),
		TaskID:     req.ID,
		AppName:    req.AppID,
		AgentName:  req.AgentName,
		Input:      req.Input,
		ContextID:  req.ContextID,
		UserID:     req.UserID,
		Metadata:   req.Metadata,
		Priority:   req.Priority,
		MaxRetries: 3,
		RetryCount: 0,
		Status:     QueueItemPending,
		EnqueuedAt: time.Now(),
	}

	return p.queue.Enqueue(ctx, item)
}

// Start starts the worker pool.
func (p *NativeProvider) Start(ctx context.Context) error {
	return p.workerPool.Start(ctx)
}

// Stop stops the worker pool.
func (p *NativeProvider) Stop() error {
	err := p.workerPool.Stop()
	p.queue.Close() // Close queue too
	return err
}

// Queue returns the underlying queue (for metrics/admin).
func (p *NativeProvider) Queue() Queue {
	return p.queue
}

// newQueue creates a Queue using the provided database configuration.
func newQueue(pool *config.DBPool, databaseDSN string, retryPolicy *RetryPolicy) (Queue, error) {
	if pool == nil {
		return nil, fmt.Errorf("DBPool is required")
	}

	if databaseDSN == "" {
		return nil, fmt.Errorf("database DSN is required")
	}

	// Parse database config
	dbCfg, err := config.ParseDSN(databaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database DSN: %w", err)
	}

	// Get database connection from pool
	db, err := pool.Get(dbCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	var opts []SQLQueueOption
	if retryPolicy != nil {
		opts = append(opts, WithRetryPolicy(retryPolicy))
	}

	return NewSQLQueue(db, dbCfg.Dialect(), opts...)
}
