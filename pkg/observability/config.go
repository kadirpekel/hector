// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package observability

import (
	"fmt"
	"time"
)

// Config configures the observability system.
type Config struct {
	// Tracing configures OpenTelemetry distributed tracing.
	Tracing TracingConfig `yaml:"tracing,omitempty"`

	// Metrics configures Prometheus metrics collection.
	Metrics MetricsConfig `yaml:"metrics,omitempty"`
}

// TracingConfig configures OpenTelemetry tracing.
type TracingConfig struct {
	// Enabled turns on distributed tracing.
	// Default: false
	Enabled bool `yaml:"enabled,omitempty"`

	// Exporter specifies the trace exporter type.
	// Values: "otlp" (default), "jaeger", "zipkin", "stdout"
	Exporter string `yaml:"exporter,omitempty"`

	// Endpoint is the collector endpoint.
	// For OTLP: "localhost:4317" (gRPC) or "localhost:4318" (HTTP)
	// For Jaeger: "http://localhost:14268/api/traces"
	// For Zipkin: "http://localhost:9411/api/pkg/spans"
	Endpoint string `yaml:"endpoint,omitempty"`

	// SamplingRate controls what fraction of traces are sampled.
	// Range: 0.0 (none) to 1.0 (all)
	// Default: 1.0
	SamplingRate float64 `yaml:"sampling_rate,omitempty"`

	// ServiceName identifies this service in traces.
	// Default: "hector"
	ServiceName string `yaml:"service_name,omitempty"`

	// ServiceVersion is the version of this service.
	ServiceVersion string `yaml:"service_version,omitempty"`

	// Insecure disables TLS for the exporter connection.
	// Default: true (for local development)
	Insecure *bool `yaml:"insecure,omitempty"`

	// Headers are additional headers to send with export requests.
	Headers map[string]string `yaml:"headers,omitempty"`

	// CapturePayloads enables capturing full LLM request/response in spans.
	// Warning: This can produce large spans. Use only for debugging.
	// Default: false
	CapturePayloads bool `yaml:"capture_payloads,omitempty"`

	// DebugExporter enables the in-memory span exporter for web UI.
	// Default: true (when tracing is enabled)
	DebugExporter *bool `yaml:"debug_exporter,omitempty"`

	// Timeout for exporter operations.
	// Default: 10s
	Timeout time.Duration `yaml:"timeout,omitempty"`
}

// MetricsConfig configures Prometheus metrics.
type MetricsConfig struct {
	// Enabled turns on metrics collection.
	// Default: false
	Enabled bool `yaml:"enabled,omitempty"`

	// Endpoint is the path to expose metrics on.
	// Default: "/metrics"
	Endpoint string `yaml:"endpoint,omitempty"`

	// Namespace prefixes all metric names.
	// Default: "hector"
	Namespace string `yaml:"namespace,omitempty"`

	// Subsystem is added between namespace and metric name.
	// Example: With namespace="hector" and subsystem="agent":
	//          metric name becomes "hector_agent_calls_total"
	Subsystem string `yaml:"subsystem,omitempty"`

	// ConstLabels are labels added to all metrics.
	ConstLabels map[string]string `yaml:"const_labels,omitempty"`
}

// SetDefaults applies default values to Config.
func (c *Config) SetDefaults() {
	c.Tracing.SetDefaults()
	c.Metrics.SetDefaults()
}

// Validate checks the Config for errors.
func (c *Config) Validate() error {
	if err := c.Tracing.Validate(); err != nil {
		return fmt.Errorf("tracing: %w", err)
	}
	if err := c.Metrics.Validate(); err != nil {
		return fmt.Errorf("metrics: %w", err)
	}
	return nil
}

// SetDefaults applies default values to TracingConfig.
func (c *TracingConfig) SetDefaults() {
	if c.ServiceName == "" {
		c.ServiceName = DefaultServiceName
	}
	if c.SamplingRate == 0 {
		c.SamplingRate = DefaultSamplingRate
	}
	if c.Exporter == "" {
		c.Exporter = "otlp"
	}
	if c.Endpoint == "" {
		c.Endpoint = DefaultOTLPEndpoint
	}
	if c.Insecure == nil {
		insecure := true
		c.Insecure = &insecure
	}
	if c.DebugExporter == nil && c.Enabled {
		debug := true
		c.DebugExporter = &debug
	}
	if c.Timeout == 0 {
		c.Timeout = 10 * time.Second
	}
}

// Validate checks TracingConfig for errors.
func (c *TracingConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required when tracing is enabled")
	}

	if c.SamplingRate < 0 || c.SamplingRate > 1 {
		return fmt.Errorf("sampling_rate must be between 0 and 1, got %f", c.SamplingRate)
	}

	validExporters := map[string]bool{
		"otlp": true, "jaeger": true, "zipkin": true, "stdout": true,
	}
	if !validExporters[c.Exporter] {
		return fmt.Errorf("invalid exporter %q (valid: otlp, jaeger, zipkin, stdout)", c.Exporter)
	}

	return nil
}

// IsDebugExporterEnabled returns whether the debug exporter should be enabled.
func (c *TracingConfig) IsDebugExporterEnabled() bool {
	if c.DebugExporter == nil {
		return c.Enabled // Default to enabled when tracing is enabled
	}
	return *c.DebugExporter
}

// IsInsecure returns whether to use insecure connection.
func (c *TracingConfig) IsInsecure() bool {
	if c.Insecure == nil {
		return true // Default to insecure for local development
	}
	return *c.Insecure
}

// SetDefaults applies default values to MetricsConfig.
func (c *MetricsConfig) SetDefaults() {
	if c.Endpoint == "" {
		c.Endpoint = DefaultMetricsPath
	}
	if c.Namespace == "" {
		c.Namespace = "hector"
	}
}

// Validate checks MetricsConfig for errors.
func (c *MetricsConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required when metrics are enabled")
	}

	return nil
}
