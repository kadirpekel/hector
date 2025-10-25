package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
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
		otlptracegrpc.WithInsecure(), // Use insecure connection for local Jaeger
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service name
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create batch span processor
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

// GetTracer returns a named tracer for a package
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
