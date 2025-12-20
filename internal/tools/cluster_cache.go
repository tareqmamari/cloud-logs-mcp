// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements TTL-based caching for log cluster results to improve
// performance for repeated queries and reduce API load.
package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// ClusterCache provides TTL-based caching for log clustering results.
// This improves performance for repeated queries in multi-agent scenarios
// and reduces unnecessary API calls.
type ClusterCache struct {
	mu      sync.RWMutex
	entries map[string]*ClusterCacheEntry
	maxSize int
	ttl     time.Duration
}

// ClusterCacheEntry represents a cached clustering result
type ClusterCacheEntry struct {
	Clusters  []*LogCluster
	CreatedAt time.Time
	ExpiresAt time.Time
	HitCount  int
}

// DefaultClusterCacheTTL is the default TTL for cluster cache entries
const DefaultClusterCacheTTL = 5 * time.Minute

// DefaultClusterCacheSize is the default maximum number of cached entries
const DefaultClusterCacheSize = 50

// globalClusterCache is the singleton cluster cache instance
var globalClusterCache = NewClusterCache(DefaultClusterCacheSize, DefaultClusterCacheTTL)

// NewClusterCache creates a new cluster cache with the specified size and TTL
func NewClusterCache(maxSize int, ttl time.Duration) *ClusterCache {
	return &ClusterCache{
		entries: make(map[string]*ClusterCacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// generateCacheKey creates a hash key for a set of events
func generateCacheKey(events []interface{}) string {
	// Create a hash based on event content
	// We use a subset of each event to balance uniqueness vs. performance
	h := sha256.New()

	for i, event := range events {
		if i >= 20 { // Sample first 20 events for key generation
			break
		}
		if eventMap, ok := event.(map[string]interface{}); ok {
			// Include message and severity in hash
			if msg, ok := eventMap["message"].(string); ok {
				h.Write([]byte(msg))
			}
			if sev, ok := eventMap["severity"].(string); ok {
				h.Write([]byte(sev))
			}
		}
	}

	// Include event count in hash
	countBytes, _ := json.Marshal(len(events))
	h.Write(countBytes)

	return hex.EncodeToString(h.Sum(nil)[:16])
}

// Get retrieves cached clusters for a set of events
func (c *ClusterCache) Get(events []interface{}) ([]*LogCluster, bool) {
	key := generateCacheKey(events)

	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, false
	}

	// Update hit count
	c.mu.Lock()
	entry.HitCount++
	c.mu.Unlock()

	return entry.Clusters, true
}

// Set stores clusters in the cache
func (c *ClusterCache) Set(events []interface{}, clusters []*LogCluster) {
	key := generateCacheKey(events)
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest entries if at capacity
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = &ClusterCacheEntry{
		Clusters:  clusters,
		CreatedAt: now,
		ExpiresAt: now.Add(c.ttl),
		HitCount:  0,
	}
}

// evictOldest removes the oldest entry from the cache
// Must be called with lock held
func (c *ClusterCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		// Also remove expired entries
		if time.Now().After(entry.ExpiresAt) {
			delete(c.entries, key)
			continue
		}

		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// Clear removes all entries from the cache
func (c *ClusterCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*ClusterCacheEntry)
}

// Stats returns cache statistics
func (c *ClusterCache) Stats() ClusterCacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := ClusterCacheStats{
		Size:    len(c.entries),
		MaxSize: c.maxSize,
		TTL:     c.ttl,
	}

	for _, entry := range c.entries {
		stats.TotalHits += entry.HitCount
		if time.Now().After(entry.ExpiresAt) {
			stats.Expired++
		}
	}

	return stats
}

// ClusterCacheStats contains cache statistics
type ClusterCacheStats struct {
	Size      int           `json:"size"`
	MaxSize   int           `json:"max_size"`
	TTL       time.Duration `json:"ttl"`
	TotalHits int           `json:"total_hits"`
	Expired   int           `json:"expired"`
}

// ClusterLogsWithCache clusters logs using the cache for repeated queries.
// This is a drop-in replacement for ClusterLogs with caching support.
func ClusterLogsWithCache(events []interface{}) []*LogCluster {
	// Skip cache for small event sets
	if len(events) < 10 {
		return ClusterLogs(events)
	}

	// Check cache first
	if clusters, found := globalClusterCache.Get(events); found {
		return clusters
	}

	// Compute clusters
	clusters := ClusterLogs(events)

	// Store in cache
	globalClusterCache.Set(events, clusters)

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
