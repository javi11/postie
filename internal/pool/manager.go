package pool

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/javi11/nntppool/v4"
	"github.com/javi11/postie/internal/config"
)

// PoolManager defines the interface for connection pool management.
// This interface is implemented by Manager and can be mocked for testing.
type PoolManager interface {
	GetPool() NNTPClient
	GetCheckPool() NNTPClient
}

// Manager manages NNTP connection pools throughout the application lifecycle,
// enabling proper metrics accumulation. Use dependency injection to pass this around.
// Supports separate pools for posting and article verification (check-only providers).
type Manager struct {
	pool      NNTPClient // Main pool for posting
	checkPool NNTPClient // Pool for article verification (check-only servers)
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
func (m *Manager) GetPool() NNTPClient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.pool
}

// GetCheckPool returns the NNTP connection pool for article verification.
// Returns check-only servers if configured, otherwise falls back to posting pool.
func (m *Manager) GetCheckPool() NNTPClient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.checkPool != nil {
		return m.checkPool
	}
	// Fallback to main pool if no dedicated check pool
	return m.pool
}

// providerKey returns a unique key for a server config (used to match providers across config changes).
func providerKey(s config.ServerConfig) string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// UpdateConfig updates the connection pools with a new configuration.
// Uses AddProvider/RemoveProvider for incremental updates instead of destroying pools.
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

	// If pools don't exist yet, create them from scratch
	if m.pool == nil {
		m.mu.Unlock()
		// Temporarily unlock since New doesn't need the lock and we reassign below
		mgr, err := New(newCfg)
		m.mu.Lock()
		if err != nil {
			return fmt.Errorf("failed to create pools: %w", err)
		}
		m.pool = mgr.pool
		m.checkPool = mgr.checkPool
		m.config = newCfg
		return nil
	}

	// Diff posting providers
	oldPostingServers := m.config.GetPostingServers()
	newPostingServers := newCfg.GetPostingServers()
	m.diffProviders(m.pool, oldPostingServers, newPostingServers, "posting")

	// Diff check-only providers
	oldCheckServers := m.config.GetCheckOnlyServers()
	newCheckServers := newCfg.GetCheckOnlyServers()

	// Handle check pool transitions
	hasOldCheckOnly := len(oldCheckServers) > 0
	hasNewCheckOnly := len(newCheckServers) > 0

	switch {
	case hasNewCheckOnly && hasOldCheckOnly && m.checkPool != m.pool:
		// Both old and new have dedicated check pools — diff the check pool
		m.diffProviders(m.checkPool, oldCheckServers, newCheckServers, "check")
	case hasNewCheckOnly && !hasOldCheckOnly:
		// New config introduces check-only servers — create dedicated check pool
		checkPool, err := newCfg.GetCheckPool()
		if err != nil {
			slog.Warn("Failed to create dedicated check pool, will use posting pool", "error", err)
			m.checkPool = m.pool
		} else {
			m.checkPool = checkPool
		}
	case !hasNewCheckOnly && hasOldCheckOnly:
		// Check-only servers removed — close dedicated check pool, fall back to posting pool
		if m.checkPool != nil && m.checkPool != m.pool {
			_ = m.checkPool.Close()
		}
		m.checkPool = m.pool
	default:
		// No check-only servers in either config — checkPool stays as posting pool alias
		m.checkPool = m.pool
	}

	m.config = newCfg

	slog.Info("NNTP connection pools updated successfully",
		"posting_servers", len(newPostingServers),
		"check_only_servers", len(newCheckServers))

	return nil
}

// diffProviders computes the diff between old and new server lists and applies
// AddProvider/RemoveProvider operations on the given pool.
func (m *Manager) diffProviders(pool NNTPClient, oldServers, newServers []config.ServerConfig, poolName string) {
	oldMap := make(map[string]config.ServerConfig, len(oldServers))
	for _, s := range oldServers {
		oldMap[providerKey(s)] = s
	}

	newMap := make(map[string]config.ServerConfig, len(newServers))
	for _, s := range newServers {
		newMap[providerKey(s)] = s
	}

	// Remove providers that are no longer in the new config or have changed
	for key, oldSrv := range oldMap {
		newSrv, exists := newMap[key]
		if !exists || serverConfigChanged(oldSrv, newSrv) {
			if err := pool.RemoveProvider(providerKey(oldSrv)); err != nil {
				slog.Warn("Failed to remove provider from pool",
					"pool", poolName, "provider", key, "error", err)
			} else {
				slog.Info("Removed provider from pool", "pool", poolName, "provider", key)
			}
		}
	}

	// Add providers that are new or have changed
	for key, newSrv := range newMap {
		oldSrv, exists := oldMap[key]
		if !exists || serverConfigChanged(oldSrv, newSrv) {
			provider := config.ServerConfigToProvider(newSrv)
			if err := pool.AddProvider(provider); err != nil {
				slog.Error("Failed to add provider to pool",
					"pool", poolName, "provider", key, "error", err)
			} else {
				slog.Info("Added provider to pool", "pool", poolName, "provider", key)
			}
		}
	}
}

// serverConfigChanged returns true if any relevant fields differ between two server configs.
func serverConfigChanged(a, b config.ServerConfig) bool {
	if a.Host != b.Host || a.Port != b.Port {
		return true
	}
	if a.Username != b.Username || a.Password != b.Password {
		return true
	}
	if a.SSL != b.SSL || a.InsecureSSL != b.InsecureSSL {
		return true
	}
	if a.MaxConnections != b.MaxConnections {
		return true
	}
	if a.Inflight != b.Inflight {
		return true
	}
	if a.MaxConnectionIdleTimeInSeconds != b.MaxConnectionIdleTimeInSeconds {
		return true
	}
	if a.MaxConnectionTTLInSeconds != b.MaxConnectionTTLInSeconds {
		return true
	}
	if a.ProxyURL != b.ProxyURL {
		return true
	}
	aEnabled := a.Enabled == nil || *a.Enabled
	bEnabled := b.Enabled == nil || *b.Enabled
	if aEnabled != bEnabled {
		return true
	}
	aCheckOnly := a.CheckOnly != nil && *a.CheckOnly
	bCheckOnly := b.CheckOnly != nil && *b.CheckOnly
	if aCheckOnly != bCheckOnly {
		return true
	}
	return false
}

// GetMetrics returns the current metrics from the connection pool.
func (m *Manager) GetMetrics() (nntppool.ClientStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nntppool.ClientStats{}, fmt.Errorf("connection pool manager has been closed")
	}

	if m.pool == nil {
		return nntppool.ClientStats{}, fmt.Errorf("connection pool is not available")
	}

	return m.pool.Stats(), nil
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
		_ = m.checkPool.Close()
		m.checkPool = nil
	}

	if m.pool != nil {
		_ = m.pool.Close()
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
