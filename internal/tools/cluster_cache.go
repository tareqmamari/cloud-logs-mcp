// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements TTL-based caching for log cluster results to improve
// performance for repeated queries and reduce API load.
//
// SOTA 2025 Optimizations:
// - Sharded cache architecture for reduced lock contention in multi-agent swarms
// - User-scoped namespacing for cache isolation
// - Background cleanup goroutine for expired entries
// - Prometheus-compatible metrics tracking
package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================================
// SHARDED CLUSTER CACHE (SOTA 2025 - Multi-Agent Swarm Optimized)
// ============================================================================

const (
	// DefaultClusterCacheTTL is the default TTL for cluster cache entries
	DefaultClusterCacheTTL = 5 * time.Minute

	// DefaultClusterCacheSize is the default maximum number of cached entries per shard
	DefaultClusterCacheSize = 50

	// DefaultShardCount is the number of cache shards for reduced lock contention
	// Using 16 shards provides good balance between memory overhead and concurrency
	DefaultShardCount = 16

	// CleanupInterval is how often the background cleanup runs
	CleanupInterval = 1 * time.Minute
)

// ClusterCacheEntry represents a cached clustering result
type ClusterCacheEntry struct {
	Clusters  []*LogCluster
	CreatedAt time.Time
	ExpiresAt time.Time
	HitCount  int
	UserID    string // User scope for multi-tenant isolation
	QueryHash string // Original query hash for debugging
}

// ClusterCacheStats contains cache statistics
type ClusterCacheStats struct {
	Size        int           `json:"size"`
	MaxSize     int           `json:"max_size"`
	TTL         time.Duration `json:"ttl"`
	TotalHits   int64         `json:"total_hits"`
	TotalMisses int64         `json:"total_misses"`
	TotalSets   int64         `json:"total_sets"`
	Evictions   int64         `json:"evictions"`
	Expired     int           `json:"expired"`
	ShardCount  int           `json:"shard_count"`
	HitRate     float64       `json:"hit_rate"`
	UserCount   int           `json:"user_count"`    // Unique users with cached data
	AvgEntryAge float64       `json:"avg_entry_age"` // Average entry age in seconds
}

// cacheShard represents a single shard of the cache
type cacheShard struct {
	mu      sync.RWMutex
	entries map[string]*ClusterCacheEntry
	maxSize int
}

// ShardedClusterCache provides a sharded TTL-based cache for log clustering results.
// Sharding reduces lock contention in high-concurrency multi-agent scenarios.
type ShardedClusterCache struct {
	shards      []*cacheShard
	shardCount  int
	ttl         time.Duration
	hits        atomic.Int64
	misses      atomic.Int64
	sets        atomic.Int64
	evictions   atomic.Int64
	stopCleanup chan struct{}
	cleanupDone chan struct{}
}

// globalClusterCache is the singleton sharded cluster cache instance
var globalClusterCache = NewShardedClusterCache(DefaultClusterCacheSize, DefaultClusterCacheTTL, DefaultShardCount)

// NewShardedClusterCache creates a new sharded cluster cache
func NewShardedClusterCache(maxSizePerShard int, ttl time.Duration, shardCount int) *ShardedClusterCache {
	if shardCount <= 0 {
		shardCount = DefaultShardCount
	}

	cache := &ShardedClusterCache{
		shards:      make([]*cacheShard, shardCount),
		shardCount:  shardCount,
		ttl:         ttl,
		stopCleanup: make(chan struct{}),
		cleanupDone: make(chan struct{}),
	}

	for i := 0; i < shardCount; i++ {
		cache.shards[i] = &cacheShard{
			entries: make(map[string]*ClusterCacheEntry),
			maxSize: maxSizePerShard,
		}
	}

	// Start background cleanup
	go cache.cleanupLoop()

	return cache
}

// getShard returns the shard for a given key
func (c *ShardedClusterCache) getShard(key string) *cacheShard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key)) // Error always nil for hash.Hash
	// #nosec G115 - shardCount is always positive and small (16), no overflow risk
	return c.shards[h.Sum32()%uint32(c.shardCount)]
}

// generateCacheKey creates a hash key for a set of events with user scope
func generateCacheKey(events []interface{}, userID string) string {
	h := sha256.New()

	// Include user ID for multi-tenant isolation
	if userID != "" {
		h.Write([]byte("user:"))
		h.Write([]byte(userID))
		h.Write([]byte(":"))
	}

	// Sample first 20 events for key generation
	for i, event := range events {
		if i >= 20 {
			break
		}
		if eventMap, ok := event.(map[string]interface{}); ok {
			if msg, ok := eventMap["message"].(string); ok {
				h.Write([]byte(msg))
			}
			if sev, ok := eventMap["severity"].(string); ok {
				h.Write([]byte(sev))
			}
		}
	}

	// Include event count
	countBytes, _ := json.Marshal(len(events))
	h.Write(countBytes)

	return hex.EncodeToString(h.Sum(nil)[:16])
}

// Get retrieves cached clusters for a set of events
func (c *ShardedClusterCache) Get(events []interface{}) ([]*LogCluster, bool) {
	return c.GetWithUser(events, "")
}

// GetWithUser retrieves cached clusters with user-scoped isolation
func (c *ShardedClusterCache) GetWithUser(events []interface{}, userID string) ([]*LogCluster, bool) {
	key := generateCacheKey(events, userID)
	shard := c.getShard(key)

	shard.mu.RLock()
	entry, exists := shard.entries[key]
	shard.mu.RUnlock()

	if !exists {
		c.misses.Add(1)
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		shard.mu.Lock()
		delete(shard.entries, key)
		shard.mu.Unlock()
		c.misses.Add(1)
		return nil, false
	}

	// Update hit count atomically within lock
	shard.mu.Lock()
	entry.HitCount++
	shard.mu.Unlock()

	c.hits.Add(1)
	return entry.Clusters, true
}

// Set stores clusters in the cache
func (c *ShardedClusterCache) Set(events []interface{}, clusters []*LogCluster) {
	c.SetWithUser(events, clusters, "")
}

// SetWithUser stores clusters with user-scoped isolation
func (c *ShardedClusterCache) SetWithUser(events []interface{}, clusters []*LogCluster, userID string) {
	key := generateCacheKey(events, userID)
	shard := c.getShard(key)
	now := time.Now()

	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Evict oldest entries if at capacity
	if len(shard.entries) >= shard.maxSize {
		c.evictOldestFromShard(shard)
	}

	shard.entries[key] = &ClusterCacheEntry{
		Clusters:  clusters,
		CreatedAt: now,
		ExpiresAt: now.Add(c.ttl),
		HitCount:  0,
		UserID:    userID,
		QueryHash: key,
	}

	c.sets.Add(1)
}

// evictOldestFromShard removes the oldest entry from a shard
// Must be called with shard lock held
func (c *ShardedClusterCache) evictOldestFromShard(shard *cacheShard) {
	var oldestKey string
	var oldestTime time.Time
	now := time.Now()

	for key, entry := range shard.entries {
		// Also remove expired entries
		if now.After(entry.ExpiresAt) {
			delete(shard.entries, key)
			c.evictions.Add(1)
			continue
		}

		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(shard.entries, oldestKey)
		c.evictions.Add(1)
	}
}

// cleanupLoop runs periodic cleanup of expired entries
func (c *ShardedClusterCache) cleanupLoop() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()
	defer close(c.cleanupDone)

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanupExpired removes all expired entries from all shards
func (c *ShardedClusterCache) cleanupExpired() {
	now := time.Now()
	for _, shard := range c.shards {
		shard.mu.Lock()
		for key, entry := range shard.entries {
			if now.After(entry.ExpiresAt) {
				delete(shard.entries, key)
				c.evictions.Add(1)
			}
		}
		shard.mu.Unlock()
	}
}

// Clear removes all entries from the cache
func (c *ShardedClusterCache) Clear() {
	for _, shard := range c.shards {
		shard.mu.Lock()
		shard.entries = make(map[string]*ClusterCacheEntry)
		shard.mu.Unlock()
	}
}

// ClearUser removes all entries for a specific user
func (c *ShardedClusterCache) ClearUser(userID string) {
	for _, shard := range c.shards {
		shard.mu.Lock()
		for key, entry := range shard.entries {
			if entry.UserID == userID {
				delete(shard.entries, key)
			}
		}
		shard.mu.Unlock()
	}
}

// Stats returns comprehensive cache statistics
func (c *ShardedClusterCache) Stats() ClusterCacheStats {
	var totalSize int
	var expired int
	var totalHitCount int
	var totalAge float64
	var entryCount int
	users := make(map[string]bool)
	now := time.Now()

	for _, shard := range c.shards {
		shard.mu.RLock()
		totalSize += len(shard.entries)
		for _, entry := range shard.entries {
			totalHitCount += entry.HitCount
			if entry.UserID != "" {
				users[entry.UserID] = true
			}
			if now.After(entry.ExpiresAt) {
				expired++
			}
			totalAge += now.Sub(entry.CreatedAt).Seconds()
			entryCount++
		}
		shard.mu.RUnlock()
	}

	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	avgAge := float64(0)
	if entryCount > 0 {
		avgAge = totalAge / float64(entryCount)
	}

	return ClusterCacheStats{
		Size:        totalSize,
		MaxSize:     c.shards[0].maxSize * c.shardCount,
		TTL:         c.ttl,
		TotalHits:   hits,
		TotalMisses: misses,
		TotalSets:   c.sets.Load(),
		Evictions:   c.evictions.Load(),
		Expired:     expired,
		ShardCount:  c.shardCount,
		HitRate:     hitRate,
		UserCount:   len(users),
		AvgEntryAge: avgAge,
	}
}

// Close stops the background cleanup goroutine
func (c *ShardedClusterCache) Close() {
	close(c.stopCleanup)
	<-c.cleanupDone
}

// ============================================================================
// BACKWARD-COMPATIBLE WRAPPER (ClusterCache interface)
// ============================================================================

// ClusterCache provides a backward-compatible interface for the sharded cache
type ClusterCache struct {
	sharded *ShardedClusterCache
}

// NewClusterCache creates a new cluster cache (backward compatible)
func NewClusterCache(maxSize int, ttl time.Duration) *ClusterCache {
	return &ClusterCache{
		sharded: NewShardedClusterCache(maxSize/DefaultShardCount+1, ttl, DefaultShardCount),
	}
}

// Get retrieves cached clusters (backward compatible)
func (c *ClusterCache) Get(events []interface{}) ([]*LogCluster, bool) {
	return c.sharded.Get(events)
}

// Set stores clusters in the cache (backward compatible)
func (c *ClusterCache) Set(events []interface{}, clusters []*LogCluster) {
	c.sharded.Set(events, clusters)
}

// Clear removes all entries (backward compatible)
func (c *ClusterCache) Clear() {
	c.sharded.Clear()
}

// Stats returns cache statistics (backward compatible)
func (c *ClusterCache) Stats() ClusterCacheStats {
	return c.sharded.Stats()
}

// ============================================================================
// GLOBAL CACHE FUNCTIONS
// ============================================================================

// ClusterLogsWithCache clusters logs using the cache for repeated queries.
// This is a drop-in replacement for ClusterLogs with caching support.
func ClusterLogsWithCache(events []interface{}) []*LogCluster {
	return ClusterLogsWithCacheAndUser(events, "")
}

// ClusterLogsWithCacheAndUser clusters logs with user-scoped caching
func ClusterLogsWithCacheAndUser(events []interface{}, userID string) []*LogCluster {
	// Skip cache for small event sets
	if len(events) < 10 {
		return ClusterLogs(events)
	}

	// Check cache first
	if clusters, found := globalClusterCache.GetWithUser(events, userID); found {
		return clusters
	}

	// Compute clusters
	clusters := ClusterLogs(events)

	// Store in cache
	globalClusterCache.SetWithUser(events, clusters, userID)

	return clusters
}

// GetClusterCacheStats returns the global cluster cache statistics
func GetClusterCacheStats() ClusterCacheStats {
	return globalClusterCache.Stats()
}

// ClearClusterCache clears the global cluster cache
func ClearClusterCache() {
	globalClusterCache.Clear()
}

// ClearUserClusterCache clears cache entries for a specific user
func ClearUserClusterCache(userID string) {
	globalClusterCache.ClearUser(userID)
}

// CloseClusterCache stops background cleanup (call during shutdown)
func CloseClusterCache() {
	globalClusterCache.Close()
}
