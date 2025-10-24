package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type NoopMetrics struct{}

func (n *NoopMetrics) RecordAgentCall(ctx context.Context, duration time.Duration, tokens int, err error) {

}

func (n *NoopMetrics) RecordToolExecution(ctx context.Context, tool string, duration time.Duration, err error) {

}

func (n *NoopMetrics) RecordLLMCall(ctx context.Context, model string, duration time.Duration, inputTokens, outputTokens int, err error) {

}

func NoopTracer(name string) trace.Tracer {
	return noop.NewTracerProvider().Tracer(name)
}
