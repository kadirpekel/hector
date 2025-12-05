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
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

// Manager manages the lifecycle of all observability components.
// It provides a unified interface for initializing, accessing, and shutting
// down tracing and metrics systems.
type Manager struct {
	config  *Config
	tracer  *Tracer
	metrics *Metrics
}

// NewManager creates a new observability Manager from configuration.
func NewManager(ctx context.Context, cfg *Config) (*Manager, error) {
	if cfg == nil {
		return &Manager{}, nil
	}

	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid observability config: %w", err)
	}

	m := &Manager{
		config: cfg,
	}

	// Initialize tracing
	if cfg.Tracing.Enabled {
		var opts []TracerOption

		// Create debug exporter if enabled
		if cfg.Tracing.IsDebugExporterEnabled() {
			debugExporter := NewDebugExporter()
			opts = append(opts, WithDebugExporter(debugExporter))
		}

		// Configure payload capture
		if cfg.Tracing.CapturePayloads {
			opts = append(opts, WithCapturePayloads(true))
		}

		tracer, err := NewTracer(ctx, &cfg.Tracing, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize tracing: %w", err)
		}
		m.tracer = tracer
		slog.Info("observability: tracing initialized",
			"exporter", cfg.Tracing.Exporter,
			"endpoint", cfg.Tracing.Endpoint,
			"sampling_rate", cfg.Tracing.SamplingRate,
		)
	}

	// Initialize metrics
	if cfg.Metrics.Enabled {
		metrics, err := NewMetrics(&cfg.Metrics)
		if err != nil {
			// Shutdown tracer if metrics init fails
			if m.tracer != nil {
				_ = m.tracer.Shutdown(ctx)
			}
			return nil, fmt.Errorf("failed to initialize metrics: %w", err)
		}
		m.metrics = metrics
		slog.Info("observability: metrics initialized",
			"endpoint", cfg.Metrics.Endpoint,
			"namespace", cfg.Metrics.Namespace,
		)
	}

	return m, nil
}

// Tracer returns the tracer instance, or nil if tracing is disabled.
func (m *Manager) Tracer() *Tracer {
	if m == nil {
		return nil
	}
	return m.tracer
}

// Metrics returns the metrics instance, or nil if metrics are disabled.
func (m *Manager) Metrics() *Metrics {
	if m == nil {
		return nil
	}
	return m.metrics
}

// DebugExporter returns the debug span exporter, or nil if not enabled.
func (m *Manager) DebugExporter() *DebugExporter {
	if m == nil || m.tracer == nil {
		return nil
	}
	return m.tracer.DebugExporter()
}

// MetricsHandler returns an HTTP handler for the metrics endpoint.
func (m *Manager) MetricsHandler() http.Handler {
	if m == nil || m.metrics == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("metrics not enabled"))
		})
	}
	return m.metrics.Handler()
}

// MetricsEndpoint returns the configured metrics endpoint path.
func (m *Manager) MetricsEndpoint() string {
	if m == nil || m.config == nil {
		return DefaultMetricsPath
	}
	return m.config.Metrics.Endpoint
}

// TracingEnabled returns whether tracing is enabled.
func (m *Manager) TracingEnabled() bool {
	return m != nil && m.tracer != nil
}

// MetricsEnabled returns whether metrics are enabled.
func (m *Manager) MetricsEnabled() bool {
	return m != nil && m.metrics != nil
}

// Shutdown gracefully shuts down all observability components.
func (m *Manager) Shutdown(ctx context.Context) error {
	if m == nil {
		return nil
	}

	var errs []error

	if m.tracer != nil {
		if err := m.tracer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
		}
		slog.Info("observability: tracing shutdown complete")
	}

	// Metrics don't need explicit shutdown in Prometheus

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}

// =============================================================================
// Factory Functions
// =============================================================================

// NewFromConfig creates a Manager with defaults from a configuration pointer.
// This is useful when the config might be nil.
func NewFromConfig(ctx context.Context, cfg *Config) (*Manager, error) {
	if cfg == nil {
		return &Manager{}, nil
	}
	return NewManager(ctx, cfg)
}

// MustNewManager creates a Manager and panics on error.
// Useful for initialization in main() when errors are fatal.
func MustNewManager(ctx context.Context, cfg *Config) *Manager {
	m, err := NewManager(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create observability manager: %v", err))
	}
	return m
}
