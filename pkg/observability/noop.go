package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// NoopMetrics is a metrics implementation that does nothing
type NoopMetrics struct{}

func (n *NoopMetrics) RecordAgentCall(ctx context.Context, duration time.Duration, tokens int, err error) {
	// No-op
}

func (n *NoopMetrics) RecordToolExecution(ctx context.Context, tool string, duration time.Duration, err error) {
	// No-op
}

func (n *NoopMetrics) RecordLLMCall(ctx context.Context, model string, duration time.Duration, inputTokens, outputTokens int, err error) {
	// No-op
}

// NoopTracer provides a no-op tracer that does nothing
func NoopTracer(name string) trace.Tracer {
	return trace.NewNoopTracerProvider().Tracer(name)
}
