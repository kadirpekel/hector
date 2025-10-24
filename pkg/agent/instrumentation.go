package agent

import (
	"context"
	"time"

	"github.com/kadirpekel/hector/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func startAgentSpan(ctx context.Context, agentName, llmModel, input string) (context.Context, trace.Span) {
	tracer := observability.GetTracer("hector.agent")

	newCtx, span := tracer.Start(ctx, observability.SpanAgentCall,
		trace.WithAttributes(
			attribute.String(observability.AttrAgentName, agentName),
			attribute.String(observability.AttrAgentLLM, llmModel),
			attribute.String("input_preview", truncateString(input, 100)),
		),
	)

	return newCtx, span
}

func recordAgentMetrics(ctx context.Context, duration time.Duration, tokens int, err error) {
	metrics := observability.GetGlobalMetrics()
	if metrics == nil {
		return
	}

	metrics.RecordAgentCall(ctx, duration, tokens, err)
}
