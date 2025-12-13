package pool

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/javi11/nntppool/v2"
	"github.com/javi11/postie/internal/config"
)

// PoolManager defines the interface for connection pool management.
// This interface is implemented by Manager and can be mocked for testing.
type PoolManager interface {
	GetPool() nntppool.UsenetConnectionPool
	GetCheckPool() nntppool.UsenetConnectionPool
}

// Manager manages NNTP connection pools throughout the application lifecycle,
// enabling proper metrics accumulation. Use dependency injection to pass this around.
// Supports separate pools for posting and article verification (check-only providers).
type Manager struct {
	pool      nntppool.UsenetConnectionPool // Main pool for posting
	checkPool nntppool.UsenetConnectionPool // Pool for article verification (check-only servers)
	config    *config.ConfigData
	mu        sync.RWMutex
	closed    bool
}

// New creates a new connection pool manager with the given configuration.
// Creates separate pools for posting (excluding check-only) and checking (check-only or fallback).
func New(cfg *config.ConfigData) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	slog.Info("Creating NNTP connection pool manager")

	// Create the posting pool (excludes check-only servers)
	postingPool, err := cfg.GetPostingPool()
	if err != nil {
		return nil, fmt.Errorf("failed to create posting pool: %w", err)
	}

	// Create the check pool (check-only servers, or falls back to posting servers)
	checkPool, err := cfg.GetCheckPool()
	if err != nil {
		// If check pool fails, we can still use posting pool for checks
		slog.Warn("Failed to create dedicated check pool, will use posting pool for article verification", "error", err)
		checkPool = postingPool
	}

	// Log pool configuration
	checkOnlyServers := cfg.GetCheckOnlyServers()
	postingServers := cfg.GetPostingServers()
	slog.Info("NNTP connection pools configured",
		"posting_servers", len(postingServers),
		"check_only_servers", len(checkOnlyServers),
		"using_dedicated_check_pool", len(checkOnlyServers) > 0)

	manager := &Manager{
		pool:      postingPool,
		checkPool: checkPool,
		config:    cfg,
		closed:    false,
	}

	slog.Info("NNTP connection pool manager created successfully")
	return manager, nil
}

// GetPool returns the posting NNTP connection pool (excludes check-only servers).
// This is the method that replaces direct calls to cfg.GetNNTPPool().
func (m *Manager) GetPool() nntppool.UsenetConnectionPool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.pool
}

// GetCheckPool returns the NNTP connection pool for article verification.
// Returns check-only servers if configured, otherwise falls back to posting pool.
func (m *Manager) GetCheckPool() nntppool.UsenetConnectionPool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.checkPool != nil {
		return m.checkPool
	}
	// Fallback to main pool if no dedicated check pool
	return m.pool
}

// UpdateConfig updates the connection pools with a new configuration.
// This properly closes the old pools first, then creates new ones to prevent resource leaks.
func (m *Manager) UpdateConfig(newCfg *config.ConfigData) error {
	if newCfg == nil {
		return fmt.Errorf("new configuration cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("connection pool manager has been closed")
	}

	slog.Info("Updating NNTP connection pools with new configuration")

	// Close the old pools first to prevent resource leaks
	if m.pool != nil {
		slog.Info("Closing old posting pool")
		m.pool.Quit()
		m.pool = nil
	}
	if m.checkPool != nil && m.checkPool != m.pool {
		slog.Info("Closing old check pool")
		m.checkPool.Quit()
		m.checkPool = nil
	}

	// Create new posting pool (excludes check-only servers)
	newPostingPool, err := newCfg.GetPostingPool()
	if err != nil {
		slog.Error("Failed to create new posting pool", "error", err)
		return fmt.Errorf("failed to create new posting pool: %w", err)
	}

	// Create new check pool (check-only servers, or falls back to posting servers)
	newCheckPool, err := newCfg.GetCheckPool()
	if err != nil {
		slog.Warn("Failed to create dedicated check pool, will use posting pool", "error", err)
		newCheckPool = newPostingPool
	}

	// Log pool configuration
	checkOnlyServers := newCfg.GetCheckOnlyServers()
	postingServers := newCfg.GetPostingServers()
	slog.Info("NNTP connection pools updated",
		"posting_servers", len(postingServers),
		"check_only_servers", len(checkOnlyServers),
		"using_dedicated_check_pool", len(checkOnlyServers) > 0)

	// Update to new pools and config
	m.pool = newPostingPool
	m.checkPool = newCheckPool
	m.config = newCfg

	slog.Info("NNTP connection pools updated successfully")
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

	// Close check pool first if it's different from the main pool
	if m.checkPool != nil && m.checkPool != m.pool {
		m.checkPool.Quit()
		m.checkPool = nil
	}

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
