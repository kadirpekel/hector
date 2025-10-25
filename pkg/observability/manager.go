package observability

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

// Config holds all observability configuration
type Config struct {
	Tracing TracerConfig  `yaml:"tracing"`
	Metrics MetricsConfig `yaml:"metrics"`
}

// Manager orchestrates all observability components
type Manager struct {
	tracerProvider trace.TracerProvider
	metrics        Metrics
	config         Config
	mu             sync.RWMutex
}

// NewManager creates a new observability manager
func NewManager(cfg Config) *Manager {
	return &Manager{
		config: cfg,
	}
}

// Initialize sets up all observability infrastructure
func (m *Manager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize tracing
	tp, err := InitGlobalTracer(ctx, m.config.Tracing)
	if err != nil {
		return err
	}
	m.tracerProvider = tp

	// Initialize metrics
	metrics, err := InitMetrics(ctx, m.config.Metrics)
	if err != nil {
		return err
	}
	m.metrics = metrics

	// Set global metrics
	SetGlobalMetrics(m.metrics)

	return nil
}

// GetTracer returns a tracer for the given package
func (m *Manager) GetTracer(name string) trace.Tracer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tracerProvider.Tracer(name)
}

// GetMetrics returns the metrics instance
func (m *Manager) GetMetrics() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics
}

// Shutdown gracefully shuts down observability infrastructure
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if spt, ok := m.tracerProvider.(interface{ Shutdown(context.Context) error }); ok {
		return spt.Shutdown(ctx)
	}
	return nil
}
