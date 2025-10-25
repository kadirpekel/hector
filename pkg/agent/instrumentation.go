package agent

import (
	"context"
	"time"

	"github.com/kadirpekel/hector/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// truncateString returns a truncated version of a string for safe logging
func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// startAgentSpan creates and starts a span for agent execution
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

// recordAgentMetrics records metrics for agent execution
func recordAgentMetrics(ctx context.Context, duration time.Duration, tokens int, err error) {
	metrics := observability.GetGlobalMetrics()
	if metrics == nil {
		return
	}

	metrics.RecordAgentCall(ctx, duration, tokens, err)
}

// recordToolMetrics records metrics for tool execution
func recordToolMetrics(ctx context.Context, toolName string, duration time.Duration, err error) {
	metrics := observability.GetGlobalMetrics()
	if metrics == nil {
		return
	}

	metrics.RecordToolExecution(ctx, toolName, duration, err)
}

// recordLLMMetrics records metrics for LLM calls
func recordLLMMetrics(ctx context.Context, model string, duration time.Duration, inputTokens, outputTokens int, err error) {
	metrics := observability.GetGlobalMetrics()
	if metrics == nil {
		return
	}

	metrics.RecordLLMCall(ctx, model, duration, inputTokens, outputTokens, err)
}
