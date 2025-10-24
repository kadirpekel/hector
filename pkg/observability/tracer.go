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
	"go.opentelemetry.io/otel/trace/noop"
)

type TracerConfig struct {
	Enabled      bool    `yaml:"enabled"`
	ExporterType string  `yaml:"exporter_type"`
	EndpointURL  string  `yaml:"endpoint_url"`
	SamplingRate float64 `yaml:"sampling_rate"`
	ServiceName  string  `yaml:"service_name"`
}

func InitGlobalTracer(ctx context.Context, cfg TracerConfig) (trace.TracerProvider, error) {
	if !cfg.Enabled {
		return noop.NewTracerProvider(), nil
	}

	var exporter sdktrace.SpanExporter
	var err error

	exporter, err = otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.EndpointURL),
		otlptracegrpc.WithInsecure(),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
