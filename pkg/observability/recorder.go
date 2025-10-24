package observability

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	globalMetrics Metrics
	metricsMu     sync.RWMutex
)

type Metrics interface {
	RecordAgentCall(ctx context.Context, duration time.Duration, tokens int, err error)
	RecordToolExecution(ctx context.Context, tool string, duration time.Duration, err error)
	RecordLLMCall(ctx context.Context, model string, duration time.Duration, inputTokens, outputTokens int, err error)
}

type PrometheusMetrics struct {
	agentDuration    metric.Float64Histogram
	agentCallsTotal  metric.Int64Counter
	agentErrorsTotal metric.Int64Counter
	agentTokensTotal metric.Int64Counter

	toolDuration    metric.Float64Histogram
	toolCallsTotal  metric.Int64Counter
	toolErrorsTotal metric.Int64Counter

	llmDuration     metric.Float64Histogram
	llmInputTokens  metric.Int64Counter
	llmOutputTokens metric.Int64Counter
	llmErrorsTotal  metric.Int64Counter
}

func NewPrometheusMetrics(
	agentDuration metric.Float64Histogram,
	agentCallsTotal metric.Int64Counter,
	agentErrorsTotal metric.Int64Counter,
	agentTokensTotal metric.Int64Counter,
	toolDuration metric.Float64Histogram,
	toolCallsTotal metric.Int64Counter,
	toolErrorsTotal metric.Int64Counter,
	llmDuration metric.Float64Histogram,
	llmInputTokens metric.Int64Counter,
	llmOutputTokens metric.Int64Counter,
	llmErrorsTotal metric.Int64Counter,
) *PrometheusMetrics {
	return &PrometheusMetrics{
		agentDuration:    agentDuration,
		agentCallsTotal:  agentCallsTotal,
		agentErrorsTotal: agentErrorsTotal,
		agentTokensTotal: agentTokensTotal,
		toolDuration:     toolDuration,
		toolCallsTotal:   toolCallsTotal,
		toolErrorsTotal:  toolErrorsTotal,
		llmDuration:      llmDuration,
		llmInputTokens:   llmInputTokens,
		llmOutputTokens:  llmOutputTokens,
		llmErrorsTotal:   llmErrorsTotal,
	}
}

func (m *PrometheusMetrics) RecordAgentCall(ctx context.Context, duration time.Duration, tokens int, err error) {
	if m == nil || m.agentDuration == nil || m.agentCallsTotal == nil {
		return
	}

	m.agentDuration.Record(ctx, duration.Seconds())
	m.agentCallsTotal.Add(ctx, 1)

	if tokens > 0 && m.agentTokensTotal != nil {
		m.agentTokensTotal.Add(ctx, int64(tokens))
	}

	if err != nil && m.agentErrorsTotal != nil {
		m.agentErrorsTotal.Add(ctx, 1)
	}
}

func (m *PrometheusMetrics) RecordToolExecution(ctx context.Context, tool string, duration time.Duration, err error) {
	if m == nil || m.toolDuration == nil || m.toolCallsTotal == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("tool", tool),
	}

	m.toolDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	m.toolCallsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))

	if err != nil && m.toolErrorsTotal != nil {
		m.toolErrorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

func (m *PrometheusMetrics) RecordLLMCall(ctx context.Context, model string, duration time.Duration, inputTokens, outputTokens int, err error) {
	if m == nil || m.llmDuration == nil || m.llmInputTokens == nil || m.llmOutputTokens == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("model", model),
	}

	m.llmDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	m.llmInputTokens.Add(ctx, int64(inputTokens), metric.WithAttributes(attrs...))
	m.llmOutputTokens.Add(ctx, int64(outputTokens), metric.WithAttributes(attrs...))

	if err != nil && m.llmErrorsTotal != nil {
		m.llmErrorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

func SetGlobalMetrics(m Metrics) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	globalMetrics = m
}

func GetGlobalMetrics() Metrics {
	metricsMu.RLock()
	defer metricsMu.RUnlock()
	return globalMetrics
}
