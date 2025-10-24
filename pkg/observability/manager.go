package observability

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	Tracing TracerConfig  `yaml:"tracing"`
	Metrics MetricsConfig `yaml:"metrics"`
}

type Manager struct {
	tracerProvider trace.TracerProvider
	metrics        Metrics
	config         Config
	mu             sync.RWMutex
}

func NewManager(cfg Config) *Manager {
	return &Manager{
		config: cfg,
	}
}

func (m *Manager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tp, err := InitGlobalTracer(ctx, m.config.Tracing)
	if err != nil {
		return err
	}
	m.tracerProvider = tp

	metrics, err := InitMetrics(ctx, m.config.Metrics)
	if err != nil {
		return err
	}
	m.metrics = metrics

	SetGlobalMetrics(m.metrics)

	return nil
}

func (m *Manager) GetTracer(name string) trace.Tracer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tracerProvider.Tracer(name)
}

func (m *Manager) GetMetrics() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics
}

func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if spt, ok := m.tracerProvider.(interface{ Shutdown(context.Context) error }); ok {
		return spt.Shutdown(ctx)
	}
	return nil
}
