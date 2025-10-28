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

	// HTTP metrics
	RecordHTTPRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration, responseSize int)

	// gRPC metrics
	RecordGRPCCall(ctx context.Context, service, method, statusCode string, duration time.Duration, err error)

	// Business KPI metrics
	RecordSession(ctx context.Context, agentName string, duration time.Duration, successful bool)
	RecordConversationTurn(ctx context.Context, agentName string, turnCount int)
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

	// HTTP metrics
	httpRequestsTotal metric.Int64Counter
	httpDuration      metric.Float64Histogram
	httpRequestSize   metric.Int64Histogram
	httpResponseSize  metric.Int64Histogram

	// gRPC metrics
	grpcCallsTotal  metric.Int64Counter
	grpcDuration    metric.Float64Histogram
	grpcErrorsTotal metric.Int64Counter

	// Business KPI metrics
	sessionDuration   metric.Float64Histogram
	sessionTotal      metric.Int64Counter
	conversationTurns metric.Int64Histogram
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
	httpRequestsTotal metric.Int64Counter,
	httpDuration metric.Float64Histogram,
	httpRequestSize metric.Int64Histogram,
	httpResponseSize metric.Int64Histogram,
	grpcCallsTotal metric.Int64Counter,
	grpcDuration metric.Float64Histogram,
	grpcErrorsTotal metric.Int64Counter,
	sessionDuration metric.Float64Histogram,
	sessionTotal metric.Int64Counter,
	conversationTurns metric.Int64Histogram,
) *PrometheusMetrics {
	return &PrometheusMetrics{
		agentDuration:     agentDuration,
		agentCallsTotal:   agentCallsTotal,
		agentErrorsTotal:  agentErrorsTotal,
		agentTokensTotal:  agentTokensTotal,
		toolDuration:      toolDuration,
		toolCallsTotal:    toolCallsTotal,
		toolErrorsTotal:   toolErrorsTotal,
		llmDuration:       llmDuration,
		llmInputTokens:    llmInputTokens,
		llmOutputTokens:   llmOutputTokens,
		llmErrorsTotal:    llmErrorsTotal,
		httpRequestsTotal: httpRequestsTotal,
		httpDuration:      httpDuration,
		httpRequestSize:   httpRequestSize,
		httpResponseSize:  httpResponseSize,
		grpcCallsTotal:    grpcCallsTotal,
		grpcDuration:      grpcDuration,
		grpcErrorsTotal:   grpcErrorsTotal,
		sessionDuration:   sessionDuration,
		sessionTotal:      sessionTotal,
		conversationTurns: conversationTurns,
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
	if globalMetrics == nil {
		return &NoopMetrics{}
	}
	return globalMetrics
}

// RecordHTTPRequest records HTTP request metrics
func (m *PrometheusMetrics) RecordHTTPRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration, responseSize int) {
	if m == nil || m.httpRequestsTotal == nil || m.httpDuration == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("method", method),
		attribute.String("path", path),
		attribute.Int("status_code", statusCode),
	}

	m.httpRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.httpDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if m.httpResponseSize != nil && responseSize > 0 {
		m.httpResponseSize.Record(ctx, int64(responseSize), metric.WithAttributes(attrs...))
	}
}

// RecordGRPCCall records gRPC call metrics
func (m *PrometheusMetrics) RecordGRPCCall(ctx context.Context, service, method, statusCode string, duration time.Duration, err error) {
	if m == nil || m.grpcCallsTotal == nil || m.grpcDuration == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("service", service),
		attribute.String("method", method),
		attribute.String("status_code", statusCode),
	}

	m.grpcCallsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.grpcDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if err != nil && m.grpcErrorsTotal != nil {
		m.grpcErrorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordSession records session-level metrics for business KPIs
func (m *PrometheusMetrics) RecordSession(ctx context.Context, agentName string, duration time.Duration, successful bool) {
	if m == nil || m.sessionTotal == nil || m.sessionDuration == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("agent", agentName),
		attribute.Bool("successful", successful),
	}

	m.sessionTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.sessionDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordConversationTurn records conversation turn count for business insights
func (m *PrometheusMetrics) RecordConversationTurn(ctx context.Context, agentName string, turnCount int) {
	if m == nil || m.conversationTurns == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("agent", agentName),
	}

	m.conversationTurns.Record(ctx, int64(turnCount), metric.WithAttributes(attrs...))
}
