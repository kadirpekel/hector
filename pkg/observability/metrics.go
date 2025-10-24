package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type MetricsConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

func InitMetrics(ctx context.Context, cfg MetricsConfig) (*PrometheusMetrics, error) {
	if !cfg.Enabled {
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
	), nil
}
