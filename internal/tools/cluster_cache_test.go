package tools

import (
	"testing"
	"time"
)

func TestClusterCache_SetAndGet(t *testing.T) {
	cache := NewClusterCache(10, 5*time.Minute)

	events := []interface{}{
		map[string]interface{}{"message": "error 1", "severity": "ERROR"},
		map[string]interface{}{"message": "error 2", "severity": "ERROR"},
	}

	clusters := []*LogCluster{
		{TemplateID: "t1", Template: "error <NUM>", Count: 2},
	}

	// Set and get
	cache.Set(events, clusters)
	result, found := cache.Get(events)

	if !found {
		t.Fatal("Expected to find cached entry")
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(result))
	}

	if result[0].TemplateID != "t1" {
		t.Errorf("Expected template ID 't1', got %q", result[0].TemplateID)
	}
}

func TestClusterCache_TTLExpiry(t *testing.T) {
	cache := NewClusterCache(10, 50*time.Millisecond)

	events := []interface{}{
		map[string]interface{}{"message": "test"},
	}

	clusters := []*LogCluster{{TemplateID: "t1"}}

	cache.Set(events, clusters)

	// Should find immediately
	_, found := cache.Get(events)
	if !found {
		t.Fatal("Expected to find entry before expiry")
	}

	// Wait for expiry
	time.Sleep(60 * time.Millisecond)

	// Should not find after expiry
	_, found = cache.Get(events)
	if found {
		t.Error("Expected entry to be expired")
	}
}

func TestClusterCache_Eviction(t *testing.T) {
	cache := NewClusterCache(3, 5*time.Minute)

	// Add 4 entries (exceeds max of 3)
	for i := 0; i < 4; i++ {
		events := []interface{}{
			map[string]interface{}{"message": string(rune('a' + i))},
		}
		clusters := []*LogCluster{{TemplateID: string(rune('a' + i))}}
		cache.Set(events, clusters)
	}

	stats := cache.Stats()
	if stats.Size > 3 {
		t.Errorf("Cache size should not exceed max size, got %d", stats.Size)
	}
}

func TestClusterCache_HitCount(t *testing.T) {
	cache := NewClusterCache(10, 5*time.Minute)

	events := []interface{}{
		map[string]interface{}{"message": "test"},
	}

	clusters := []*LogCluster{{TemplateID: "t1"}}
	cache.Set(events, clusters)

	// Access multiple times
	for i := 0; i < 5; i++ {
		cache.Get(events)
	}

	stats := cache.Stats()
	if stats.TotalHits != 5 {
		t.Errorf("Expected 5 hits, got %d", stats.TotalHits)
	}
}

func TestClusterCache_Clear(t *testing.T) {
	cache := NewClusterCache(10, 5*time.Minute)

	events := []interface{}{
		map[string]interface{}{"message": "test"},
	}

	clusters := []*LogCluster{{TemplateID: "t1"}}
	cache.Set(events, clusters)

	cache.Clear()

	_, found := cache.Get(events)
	if found {
		t.Error("Expected cache to be empty after clear")
	}

	stats := cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Expected size 0 after clear, got %d", stats.Size)
	}
}

func TestGenerateCacheKey(t *testing.T) {
	events1 := []interface{}{
		map[string]interface{}{"message": "error A", "severity": "ERROR"},
	}
	events2 := []interface{}{
		map[string]interface{}{"message": "error B", "severity": "ERROR"},
	}
	events3 := []interface{}{
		map[string]interface{}{"message": "error A", "severity": "ERROR"},
	}

	key1 := generateCacheKey(events1)
	key2 := generateCacheKey(events2)
	key3 := generateCacheKey(events3)

	// Different events should produce different keys
	if key1 == key2 {
		t.Error("Different events should produce different keys")
	}

	// Same events should produce same keys
	if key1 != key3 {
		t.Error("Same events should produce same keys")
	}

	// Keys should not be empty
	if key1 == "" || key2 == "" {
		t.Error("Keys should not be empty")
	}
}

func TestClusterLogsWithCache(t *testing.T) {
	// Clear global cache first
	ClearClusterCache()

	events := make([]interface{}, 20)
	for i := 0; i < 20; i++ {
		events[i] = map[string]interface{}{
			"message":  "Connection timeout to server",
			"severity": "ERROR",
		}
	}

	// First call - should compute
	result1 := ClusterLogsWithCache(events)

	// Second call - should use cache
	result2 := ClusterLogsWithCache(events)

	if len(result1) != len(result2) {
		t.Errorf("Cached result should match: %d vs %d", len(result1), len(result2))
	}

	stats := GetClusterCacheStats()
	if stats.TotalHits < 1 {
		t.Error("Expected at least 1 cache hit")
	}
}

func TestClusterLogsWithCache_SmallEvents(t *testing.T) {
	ClearClusterCache()

	// Small event sets should bypass cache
	events := []interface{}{
		map[string]interface{}{"message": "small set"},
	}

	ClusterLogsWithCache(events)
	ClusterLogsWithCache(events)

	stats := GetClusterCacheStats()
	// Small sets don't use cache, so no entries
	if stats.Size > 0 {
		t.Errorf("Small event sets should not be cached, but size is %d", stats.Size)
	}
}

func BenchmarkClusterCache_Get(b *testing.B) {
	cache := NewClusterCache(100, 5*time.Minute)

	events := make([]interface{}, 50)
	for i := 0; i < 50; i++ {
		events[i] = map[string]interface{}{
			"message":  "benchmark message",
			"severity": "ERROR",
		}
	}

	clusters := []*LogCluster{{TemplateID: "bench"}}
	cache.Set(events, clusters)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(events)
	}
}

func BenchmarkClusterCache_KeyGeneration(b *testing.B) {
	events := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		events[i] = map[string]interface{}{
			"message":  "benchmark message for key generation",
			"severity": "ERROR",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateCacheKey(events)
	}
}
