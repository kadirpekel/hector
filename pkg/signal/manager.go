// Package signal provides a clean signal management foundation for the Hector server.
// It supports registering callbacks for reload (SIGHUP) and shutdown (SIGINT/SIGTERM) signals.
package signal

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Manager provides lifecycle management for OS signals.
// It listens for signals and invokes registered callbacks.
type Manager struct {
	ctx           context.Context
	cancel        context.CancelFunc
	reloadFuncs   []func()
	shutdownFuncs []func()
	mu            sync.RWMutex
	started       bool
}

// New creates a new signal Manager.
// The provided parent context is used for lifecycle management.
func New(parent context.Context) *Manager {
	ctx, cancel := context.WithCancel(parent)
	return &Manager{
		ctx:    ctx,
		cancel: cancel,
	}
}

// OnReload registers a callback to be invoked when SIGHUP is received.
// Multiple callbacks can be registered; they are called in registration order.
func (m *Manager) OnReload(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reloadFuncs = append(m.reloadFuncs, fn)
}

// OnShutdown registers a callback to be invoked when SIGINT or SIGTERM is received.
// Multiple callbacks can be registered; they are called in registration order.
func (m *Manager) OnShutdown(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shutdownFuncs = append(m.shutdownFuncs, fn)
}

// Start begins listening for signals in a background goroutine.
// Returns a context that will be cancelled on shutdown signals.
// This method is idempotent; subsequent calls return the same context.
func (m *Manager) Start() context.Context {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return m.ctx
	}
	m.started = true
	m.mu.Unlock()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go m.run(sigCh)

	return m.ctx
}

// run is the main signal handling loop.
func (m *Manager) run(sigCh <-chan os.Signal) {
	for {
		select {
		case <-m.ctx.Done():
			return
		case sig := <-sigCh:
			switch sig {
			case syscall.SIGHUP:
				m.handleReload()
			case syscall.SIGINT, syscall.SIGTERM:
				m.handleShutdown()
				return
			}
		}
	}
}

// handleReload invokes all registered reload callbacks.
func (m *Manager) handleReload() {
	m.mu.RLock()
	funcs := make([]func(), len(m.reloadFuncs))
	copy(funcs, m.reloadFuncs)
	m.mu.RUnlock()

	slog.Info("Received SIGHUP, triggering reload callbacks", "count", len(funcs))

	for i, fn := range funcs {
		func(idx int, callback func()) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Panic in reload callback", "index", idx, "error", r)
				}
			}()
			callback()
		}(i, fn)
	}

	slog.Info("Reload complete")
}

// TriggerReload manually triggers the reload process.
func (m *Manager) TriggerReload() {
	m.handleReload()
}

// handleShutdown invokes all registered shutdown callbacks and cancels the context.
func (m *Manager) handleShutdown() {
	m.mu.RLock()
	funcs := make([]func(), len(m.shutdownFuncs))
	copy(funcs, m.shutdownFuncs)
	m.mu.RUnlock()

	slog.Info("Received shutdown signal, triggering shutdown callbacks", "count", len(funcs))

	for i, fn := range funcs {
		func(idx int, callback func()) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Panic in shutdown callback", "index", idx, "error", r)
				}
			}()
			callback()
		}(i, fn)
	}

	slog.Info("Shutting down...")
	m.cancel()
}

// Stop cancels the manager's context and stops signal handling.
func (m *Manager) Stop() {
	m.cancel()
}
