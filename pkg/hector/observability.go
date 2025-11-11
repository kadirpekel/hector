package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/observability"
)

// ObservabilityBuilder provides a fluent API for building observability configurations
type ObservabilityBuilder struct {
	tracingConfig   *TracingBuilder
	metricsEnabled  *bool
}

// NewObservability creates a new observability builder
func NewObservability() *ObservabilityBuilder {
	return &ObservabilityBuilder{}
}

// EnableMetrics enables or disables metrics collection
func (b *ObservabilityBuilder) EnableMetrics(enabled bool) *ObservabilityBuilder {
	b.metricsEnabled = boolPtr(enabled)
	return b
}

// WithTracing sets the tracing configuration using a builder
func (b *ObservabilityBuilder) WithTracing(tracingBuilder *TracingBuilder) *ObservabilityBuilder {
	b.tracingConfig = tracingBuilder
	return b
}

// Build creates the observability configuration
func (b *ObservabilityBuilder) Build() (observability.Config, error) {
	cfg := observability.Config{
		MetricsEnabled: boolValue(b.metricsEnabled, false),
	}

	if b.tracingConfig != nil {
		tracingCfg, err := b.tracingConfig.Build()
		if err != nil {
			return cfg, fmt.Errorf("failed to build tracing config: %w", err)
		}
		cfg.Tracing = tracingCfg
	}

	return cfg, nil
}

// TracingBuilder provides a fluent API for building tracing configurations
type TracingBuilder struct {
	enabled      *bool
	exporterType string
	endpointURL  string
	samplingRate float64
	serviceName  string
}

// NewTracing creates a new tracing builder
func NewTracing() *TracingBuilder {
	return &TracingBuilder{
		samplingRate: 1.0, // Default: 100% sampling
	}
}

// Enable enables or disables tracing
func (b *TracingBuilder) Enable(enabled bool) *TracingBuilder {
	b.enabled = boolPtr(enabled)
	return b
}

// ExporterType sets the exporter type (e.g., "otlp")
func (b *TracingBuilder) ExporterType(exporterType string) *TracingBuilder {
	b.exporterType = exporterType
	return b
}

// EndpointURL sets the tracing endpoint URL
func (b *TracingBuilder) EndpointURL(url string) *TracingBuilder {
	b.endpointURL = url
	return b
}

// SamplingRate sets the sampling rate (0.0 to 1.0)
func (b *TracingBuilder) SamplingRate(rate float64) *TracingBuilder {
	if rate < 0 || rate > 1 {
		panic("sampling rate must be between 0.0 and 1.0")
	}
	b.samplingRate = rate
	return b
}

// ServiceName sets the service name for tracing
func (b *TracingBuilder) ServiceName(name string) *TracingBuilder {
	b.serviceName = name
	return b
}

// Build creates the tracing configuration
func (b *TracingBuilder) Build() (observability.TracerConfig, error) {
	cfg := observability.TracerConfig{
		Enabled:      boolValue(b.enabled, false),
		ExporterType: b.exporterType,
		EndpointURL:  b.endpointURL,
		SamplingRate: b.samplingRate,
		ServiceName:  b.serviceName,
	}

	// Validate if enabled
	if cfg.Enabled {
		if cfg.EndpointURL == "" {
			return cfg, fmt.Errorf("endpoint URL is required when tracing is enabled")
		}
		if cfg.SamplingRate < 0 || cfg.SamplingRate > 1 {
			return cfg, fmt.Errorf("sampling rate must be between 0.0 and 1.0")
		}
	}

	return cfg, nil
}

