package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func InitMetrics(ctx context.Context, enabled bool) (*PrometheusMetrics, error) {
	if !enabled {
		return &PrometheusMetrics{}, nil
	}

	promExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExporter),
	)

	meter := meterProvider.Meter("hector")

	agentDuration, err := meter.Float64Histogram(
		"hector_agent_call_duration_seconds",
		metric.WithDescription("Agent call duration in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent duration histogram: %w", err)
	}

	agentCalls, err := meter.Int64Counter(
		"hector_agent_calls_total",
		metric.WithDescription("Total agent calls"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent calls counter: %w", err)
	}

	agentErrors, err := meter.Int64Counter(
		"hector_agent_errors_total",
		metric.WithDescription("Total agent errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent errors counter: %w", err)
	}

	agentTokens, err := meter.Int64Counter(
		"hector_agent_tokens_used_total",
		metric.WithDescription("Total tokens used by agents"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent tokens counter: %w", err)
	}

	toolDuration, err := meter.Float64Histogram(
		"hector_tool_execution_duration_seconds",
		metric.WithDescription("Tool execution duration in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool duration histogram: %w", err)
	}

	toolCalls, err := meter.Int64Counter(
		"hector_tool_calls_total",
		metric.WithDescription("Total tool calls"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool calls counter: %w", err)
	}

	toolErrors, err := meter.Int64Counter(
		"hector_tool_errors_total",
		metric.WithDescription("Total tool errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool errors counter: %w", err)
	}

	llmDuration, err := meter.Float64Histogram(
		"hector_llm_request_duration_seconds",
		metric.WithDescription("LLM request duration in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create llm duration histogram: %w", err)
	}

	llmInputTokens, err := meter.Int64Counter(
		"hector_llm_tokens_input_total",
		metric.WithDescription("Total input tokens sent to LLM"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create llm input tokens counter: %w", err)
	}

	llmOutputTokens, err := meter.Int64Counter(
		"hector_llm_tokens_output_total",
		metric.WithDescription("Total output tokens from LLM"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create llm output tokens counter: %w", err)
	}

	llmErrors, err := meter.Int64Counter(
		"hector_llm_errors_total",
		metric.WithDescription("Total LLM errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create llm errors counter: %w", err)
	}

	// HTTP metrics
	httpRequestsTotal, err := meter.Int64Counter(
		"hector_http_requests_total",
		metric.WithDescription("Total HTTP requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http requests counter: %w", err)
	}

	httpDuration, err := meter.Float64Histogram(
		"hector_http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http duration histogram: %w", err)
	}

	httpRequestSize, err := meter.Int64Histogram(
		"hector_http_request_size_bytes",
		metric.WithDescription("HTTP request size in bytes"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request size histogram: %w", err)
	}

	httpResponseSize, err := meter.Int64Histogram(
		"hector_http_response_size_bytes",
		metric.WithDescription("HTTP response size in bytes"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http response size histogram: %w", err)
	}

	// gRPC metrics
	grpcCallsTotal, err := meter.Int64Counter(
		"hector_grpc_calls_total",
		metric.WithDescription("Total gRPC calls"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc calls counter: %w", err)
	}

	grpcDuration, err := meter.Float64Histogram(
		"hector_grpc_call_duration_seconds",
		metric.WithDescription("gRPC call duration in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc duration histogram: %w", err)
	}

	grpcErrors, err := meter.Int64Counter(
		"hector_grpc_errors_total",
		metric.WithDescription("Total gRPC errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc errors counter: %w", err)
	}

	// Business KPI metrics
	sessionDuration, err := meter.Float64Histogram(
		"hector_session_duration_seconds",
		metric.WithDescription("Session duration in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session duration histogram: %w", err)
	}

	sessionTotal, err := meter.Int64Counter(
		"hector_session_total",
		metric.WithDescription("Total sessions"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session counter: %w", err)
	}

	conversationTurns, err := meter.Int64Histogram(
		"hector_conversation_turns",
		metric.WithDescription("Number of conversation turns"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation turns histogram: %w", err)
	}

	return NewPrometheusMetrics(
		agentDuration,
		agentCalls,
		agentErrors,
		agentTokens,
		toolDuration,
		toolCalls,
		toolErrors,
		llmDuration,
		llmInputTokens,
		llmOutputTokens,
		llmErrors,
		httpRequestsTotal,
		httpDuration,
		httpRequestSize,
		httpResponseSize,
		grpcCallsTotal,
		grpcDuration,
		grpcErrors,
		sessionDuration,
		sessionTotal,
		conversationTurns,
	), nil
}
