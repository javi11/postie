package pool

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/javi11/nntppool"
	"github.com/javi11/postie/internal/config"
)

// Manager manages a single NNTP connection pool throughout the application lifecycle,
// enabling proper metrics accumulation. Use dependency injection to pass this around.
type Manager struct {
	pool   nntppool.UsenetConnectionPool
	config *config.ConfigData
	mu     sync.RWMutex
	closed bool
}

// New creates a new connection pool manager with the given configuration.
// The pool is always created successfully, even if providers are misconfigured.
func New(cfg *config.ConfigData) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	slog.Info("Creating NNTP connection pool manager")

	// Create the NNTP pool using the configuration
	// With nntppool v1.2.0+, this always succeeds regardless of provider status
	pool, err := cfg.GetNNTPPool()
	if err != nil {
		return nil, fmt.Errorf("failed to create NNTP pool: %w", err)
	}

	manager := &Manager{
		pool:   pool,
		config: cfg,
		closed: false,
	}

	slog.Info("NNTP connection pool manager created successfully")
	return manager, nil
}

// GetPool returns the underlying NNTP connection pool.
// This is the method that replaces direct calls to cfg.GetNNTPPool().
func (m *Manager) GetPool() nntppool.UsenetConnectionPool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.pool
}

// UpdateConfig updates the connection pool with a new configuration.
// This properly closes the old pool first, then creates a new one to prevent resource leaks.
func (m *Manager) UpdateConfig(newCfg *config.ConfigData) error {
	if newCfg == nil {
		return fmt.Errorf("new configuration cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("connection pool manager has been closed")
	}

	slog.Info("Updating NNTP connection pool with new configuration")

	// Close the old pool first to prevent resource leaks
	if m.pool != nil {
		slog.Info("Closing old NNTP connection pool")
		m.pool.Quit()
		m.pool = nil
	}

	// Create new pool with updated configuration
	newPool, err := newCfg.GetNNTPPool()
	if err != nil {
		slog.Error("Failed to create new NNTP pool", "error", err)
		return fmt.Errorf("failed to create new NNTP pool: %w", err)
	}

	// Update to new pool and config
	m.pool = newPool
	m.config = newCfg

	slog.Info("NNTP connection pool updated successfully")
	return nil
}

// GetMetrics returns the current metrics from the connection pool.
func (m *Manager) GetMetrics() (nntppool.PoolMetricsSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nntppool.PoolMetricsSnapshot{}, fmt.Errorf("connection pool manager has been closed")
	}

	if m.pool == nil {
		return nntppool.PoolMetricsSnapshot{}, fmt.Errorf("connection pool is not available")
	}

	return m.pool.GetMetricsSnapshot(), nil
}

// Close gracefully shuts down the connection pool manager.
// This should be called during application shutdown.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil // Already closed
	}

	slog.Info("Closing NNTP connection pool manager")

	if m.pool != nil {
		m.pool.Quit()
		m.pool = nil
	}

	m.closed = true
	slog.Info("NNTP connection pool manager closed successfully")

	return nil
}

// IsClosed returns true if the manager has been closed.
func (m *Manager) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

