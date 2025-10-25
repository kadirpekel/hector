package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TracerConfig holds tracer configuration
type TracerConfig struct {
	Enabled      bool    `yaml:"enabled"`
	ExporterType string  `yaml:"exporter_type"` // jaeger, datadog, honeycomb, otlp
	EndpointURL  string  `yaml:"endpoint_url"`
	SamplingRate float64 `yaml:"sampling_rate"`
	ServiceName  string  `yaml:"service_name"`
}

// InitGlobalTracer initializes and sets the global tracer provider
func InitGlobalTracer(ctx context.Context, cfg TracerConfig) (trace.TracerProvider, error) {
	if !cfg.Enabled {
		return trace.NewNoopTracerProvider(), nil
	}

	var exporter sdktrace.SpanExporter
	var err error

	// All supported exporters use OTLP protocol
	exporter, err = otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.EndpointURL),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create batch span processor
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

// GetTracer returns a named tracer for a package
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
