package native

import (
	"context"
	"log/slog"
	"time"

	"github.com/verikod/hector/pkg/observability"
)

// MetricsExporter periodicall collects queue stats and exports them to metrics.
type MetricsExporter struct {
	queue   Queue
	metrics *observability.Metrics
	appName string
}

// NewMetricsExporter creates a new metrics exporter.
func NewMetricsExporter(queue Queue, metrics *observability.Metrics, appName string) *MetricsExporter {
	return &MetricsExporter{
		queue:   queue,
		metrics: metrics,
		appName: appName,
	}
}

// Start begins periodic metrics collection. blocking.
func (e *MetricsExporter) Start(ctx context.Context, interval time.Duration) {
	if e.metrics == nil {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	e.collect(ctx) // Collect immediately

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.collect(ctx)
		}
	}
}

func (e *MetricsExporter) collect(ctx context.Context) {
	stats, err := e.queue.Stats(ctx, e.appName)
	if err != nil {
		slog.Warn("Failed to collect queue stats for metrics", "app", e.appName, "error", err)
		return
	}
	if stats == nil {
		return
	}

	e.metrics.SetQueueDepth(e.appName, "pending", float64(stats.Pending))
	e.metrics.SetQueueDepth(e.appName, "processing", float64(stats.Processing))
	e.metrics.SetQueueDepth(e.appName, "completed", float64(stats.Completed))
	e.metrics.SetQueueDepth(e.appName, "failed", float64(stats.Failed))
	e.metrics.SetQueueDepth(e.appName, "dead_letter", float64(stats.DeadLetter))
}
