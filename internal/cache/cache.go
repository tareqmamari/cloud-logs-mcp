// Package cache provides a user-isolated caching layer for MCP tool results.
// Each cache entry is scoped to a specific user and service instance to ensure
// complete isolation between different users and tenants.
package cache

import (
	"sync"
	"time"
)

// Entry represents a cached item with metadata
type Entry struct {
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
	HitCount  int         `json:"hit_count"`
}

// IsExpired checks if the cache entry has expired
func (e *Entry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// UserCache holds cached data for a specific user+instance combination
type UserCache struct {
	mu      sync.RWMutex
	entries map[string]*Entry
	maxSize int
}

// NewUserCache creates a new user-specific cache
func NewUserCache(maxSize int) *UserCache {
	if maxSize <= 0 {
		maxSize = 100 // Default max entries
	}
	return &UserCache{
		entries: make(map[string]*Entry),
		maxSize: maxSize,
	}
}

// Get retrieves a value from the cache
func (c *UserCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if entry.IsExpired() {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, false
	}

	// Update hit count (needs write lock)
	c.mu.Lock()
	entry.HitCount++
	c.mu.Unlock()

	return entry.Value, true
}

// Set stores a value in the cache with TTL
func (c *UserCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict expired entries if at capacity
	if len(c.entries) >= c.maxSize {
		c.evictExpiredLocked()
	}

	// If still at capacity, evict least recently used
	if len(c.entries) >= c.maxSize {
		c.evictLRULocked()
	}

	c.entries[key] = &Entry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
		HitCount:  0,
	}
}

// Delete removes a specific key from the cache
func (c *UserCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// DeleteByPrefix removes all entries with keys starting with prefix
func (c *UserCache) DeleteByPrefix(prefix string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for key := range c.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.entries, key)
			count++
		}
	}
	return count
}

// Clear removes all entries from the cache
func (c *UserCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*Entry)
}

// Size returns the number of entries in the cache
func (c *UserCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Stats returns cache statistics
func (c *UserCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalHits := 0
	expiredCount := 0
	now := time.Now()

	for _, entry := range c.entries {
		totalHits += entry.HitCount
		if now.After(entry.ExpiresAt) {
			expiredCount++
		}
	}

	return map[string]interface{}{
		"size":          len(c.entries),
		"max_size":      c.maxSize,
		"total_hits":    totalHits,
		"expired_count": expiredCount,
	}
}

// evictExpiredLocked removes all expired entries (must hold write lock)
func (c *UserCache) evictExpiredLocked() {
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

// evictLRULocked removes the least recently used entry (must hold write lock)
func (c *UserCache) evictLRULocked() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range c.entries {
		if first || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// Manager manages per-user caches with instance isolation
type Manager struct {
	mu     sync.RWMutex
	caches map[string]*UserCache // keyed by userID:instanceID
	config *Config
}

// Config holds cache configuration
type Config struct {
	// MaxEntriesPerUser is the maximum number of cache entries per user
	MaxEntriesPerUser int

	// DefaultTTL is the default time-to-live for cache entries
	DefaultTTL time.Duration

	// TTLByTool allows custom TTLs for specific tools
	TTLByTool map[string]time.Duration

	// Enabled controls whether caching is active
	Enabled bool
}

// DefaultConfig returns the default cache configuration
func DefaultConfig() *Config {
	return &Config{
		MaxEntriesPerUser: 100,
		DefaultTTL:        5 * time.Minute,
		TTLByTool: map[string]time.Duration{
			// Static resources - longer TTL
			"list_alerts":            5 * time.Minute,
			"list_dashboards":        5 * time.Minute,
			"list_policies":          5 * time.Minute,
			"list_outgoing_webhooks": 5 * time.Minute,
			"list_views":             5 * time.Minute,
			"list_e2m":               5 * time.Minute,
			"list_streams":           5 * time.Minute,
			"list_data_access_rules": 5 * time.Minute,

			// Individual resource fetches - medium TTL
			"get_alert":     2 * time.Minute,
			"get_dashboard": 2 * time.Minute,

			// Dynamic data - shorter TTL
			"query_logs":   30 * time.Second,
			"health_check": 1 * time.Minute,

			// AI helpers - can cache suggestions
			"suggest_alert": 3 * time.Minute,
		},
		Enabled: true,
	}
}

// Global cache manager
var (
	globalManager     *Manager
	globalManagerOnce sync.Once
)

// GetManager returns the global cache manager
func GetManager() *Manager {
	globalManagerOnce.Do(func() {
		globalManager = NewManager(DefaultConfig())
	})
	return globalManager
}

// NewManager creates a new cache manager
func NewManager(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}
	return &Manager{
		caches: make(map[string]*UserCache),
		config: config,
	}
}

// cacheKey generates the key for a user+instance combination
func cacheKey(userID, instanceID string) string {
	return userID + ":" + instanceID
}

// GetUserCache returns the cache for a specific user+instance
func (m *Manager) GetUserCache(userID, instanceID string) *UserCache {
	key := cacheKey(userID, instanceID)

	m.mu.RLock()
	cache, exists := m.caches[key]
	m.mu.RUnlock()

	if exists {
		return cache
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if cache, exists := m.caches[key]; exists {
		return cache
	}

	cache = NewUserCache(m.config.MaxEntriesPerUser)
	m.caches[key] = cache
	return cache
}

// Get retrieves a cached value for a user
func (m *Manager) Get(userID, instanceID, toolName string, cacheKey string) (interface{}, bool) {
	if !m.config.Enabled {
		return nil, false
	}

	cache := m.GetUserCache(userID, instanceID)
	fullKey := toolName + ":" + cacheKey
	return cache.Get(fullKey)
}

// Set stores a value in the user's cache
func (m *Manager) Set(userID, instanceID, toolName string, cacheKey string, value interface{}) {
	if !m.config.Enabled {
		return
	}

	cache := m.GetUserCache(userID, instanceID)
	fullKey := toolName + ":" + cacheKey

	// Get TTL for this tool
	ttl := m.config.DefaultTTL
	if toolTTL, ok := m.config.TTLByTool[toolName]; ok {
		ttl = toolTTL
	}

	cache.Set(fullKey, value, ttl)
}

// InvalidateTool removes all cache entries for a specific tool
func (m *Manager) InvalidateTool(userID, instanceID, toolName string) int {
	cache := m.GetUserCache(userID, instanceID)
	return cache.DeleteByPrefix(toolName + ":")
}

// InvalidateRelated invalidates cache for related tools when a mutation occurs
func (m *Manager) InvalidateRelated(userID, instanceID, mutationTool string) {
	// Define what tools should be invalidated for each mutation
	invalidationMap := map[string][]string{
		// Alert mutations invalidate alert-related caches
		"create_alert": {"list_alerts", "get_alert", "suggest_alert"},
		"update_alert": {"list_alerts", "get_alert", "suggest_alert"},
		"delete_alert": {"list_alerts", "get_alert", "suggest_alert"},

		// Dashboard mutations
		"create_dashboard": {"list_dashboards", "get_dashboard"},
		"update_dashboard": {"list_dashboards", "get_dashboard"},
		"delete_dashboard": {"list_dashboards", "get_dashboard"},

		// Policy mutations
		"create_policy": {"list_policies"},
		"update_policy": {"list_policies"},
		"delete_policy": {"list_policies"},

		// Webhook mutations
		"create_outgoing_webhook": {"list_outgoing_webhooks"},
		"update_outgoing_webhook": {"list_outgoing_webhooks"},
		"delete_outgoing_webhook": {"list_outgoing_webhooks"},

		// E2M mutations
		"create_e2m": {"list_e2m"},
		"update_e2m": {"list_e2m"},
		"delete_e2m": {"list_e2m"},

		// Stream mutations
		"create_stream": {"list_streams"},
		"delete_stream": {"list_streams"},

		// View mutations
		"create_view": {"list_views"},
		"update_view": {"list_views"},
		"delete_view": {"list_views"},

		// Data access mutations
		"create_data_access_rule": {"list_data_access_rules"},
		"update_data_access_rule": {"list_data_access_rules"},
		"delete_data_access_rule": {"list_data_access_rules"},

		// Ingestion may affect queries
		"ingest_logs": {"query_logs", "health_check"},
	}

	if toolsToInvalidate, ok := invalidationMap[mutationTool]; ok {
		for _, tool := range toolsToInvalidate {
			m.InvalidateTool(userID, instanceID, tool)
		}
	}
}

// ClearUser removes all cache entries for a user
func (m *Manager) ClearUser(userID, instanceID string) {
	cache := m.GetUserCache(userID, instanceID)
	cache.Clear()
}

// Stats returns cache statistics for a user
func (m *Manager) Stats(userID, instanceID string) map[string]interface{} {
	cache := m.GetUserCache(userID, instanceID)
	stats := cache.Stats()
	stats["user_id"] = userID
	stats["instance_id"] = instanceID
	stats["enabled"] = m.config.Enabled
	return stats
}

// GlobalStats returns statistics across all user caches
func (m *Manager) GlobalStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalSize := 0
	totalHits := 0

	for _, cache := range m.caches {
		stats := cache.Stats()
		totalSize += stats["size"].(int)
		totalHits += stats["total_hits"].(int)
	}

	return map[string]interface{}{
		"user_cache_count": len(m.caches),
		"total_entries":    totalSize,
		"total_hits":       totalHits,
		"enabled":          m.config.Enabled,
	}
}

// SetEnabled enables or disables caching
func (m *Manager) SetEnabled(enabled bool) {
	m.config.Enabled = enabled
}

// IsEnabled returns whether caching is enabled
func (m *Manager) IsEnabled() bool {
	return m.config.Enabled
}
