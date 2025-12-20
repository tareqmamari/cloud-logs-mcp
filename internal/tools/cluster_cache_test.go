package tools

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// ============================================================================
// Sharded Cluster Cache Tests - SOTA 2025
// ============================================================================

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
	// Use larger max size to test eviction properly with sharded cache
	// The sharded cache divides size by shard count, so use 32 for 16 shards = 2 per shard
	cache := NewClusterCache(32, 5*time.Minute)

	// Add more entries than max size (50 entries)
	for i := 0; i < 50; i++ {
		events := []interface{}{
			map[string]interface{}{"message": string(rune('a'+i%26)) + fmt.Sprintf("-%d", i)},
		}
		clusters := []*LogCluster{{TemplateID: fmt.Sprintf("t%d", i)}}
		cache.Set(events, clusters)
	}

	stats := cache.Stats()
	// Total max size is 32, but with sharding the effective max is 16 shards * (32/16+1) = 48
	// The cache should limit overall size through eviction
	if stats.Size > stats.MaxSize {
		t.Errorf("Cache size %d should not exceed max size %d", stats.Size, stats.MaxSize)
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

	key1 := generateCacheKey(events1, "")
	key2 := generateCacheKey(events2, "")
	key3 := generateCacheKey(events3, "")

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

// TestGenerateCacheKey_UserScoped tests user-scoped cache key generation
func TestGenerateCacheKey_UserScoped(t *testing.T) {
	events := []interface{}{
		map[string]interface{}{"message": "test", "severity": "INFO"},
	}

	// Same events, different users should have different keys
	key1 := generateCacheKey(events, "user1")
	key2 := generateCacheKey(events, "user2")
	key3 := generateCacheKey(events, "user1")
	keyNoUser := generateCacheKey(events, "")

	if key1 == key2 {
		t.Error("Different users should have different cache keys")
	}
	if key1 != key3 {
		t.Error("Same user should have same cache key")
	}
	if key1 == keyNoUser {
		t.Error("User key should differ from no-user key")
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
		generateCacheKey(events, "benchuser")
	}
}

// ============================================================================
// Sharded Cache Specific Tests (SOTA 2025)
// ============================================================================

func TestShardedClusterCache_UserScoping(t *testing.T) {
	cache := NewShardedClusterCache(10, 5*time.Minute, 4)
	defer cache.Close()

	events := []interface{}{
		map[string]interface{}{"message": "shared message", "severity": "INFO"},
	}

	clustersUser1 := []*LogCluster{{TemplateID: "user1", Count: 1}}
	clustersUser2 := []*LogCluster{{TemplateID: "user2", Count: 2}}

	// Set for different users
	cache.SetWithUser(events, clustersUser1, "user1")
	cache.SetWithUser(events, clustersUser2, "user2")

	// Get for user1
	got1, found1 := cache.GetWithUser(events, "user1")
	if !found1 {
		t.Fatal("Expected to find user1's clusters")
	}
	if got1[0].TemplateID != "user1" {
		t.Errorf("User1 TemplateID = %q, want 'user1'", got1[0].TemplateID)
	}

	// Get for user2
	got2, found2 := cache.GetWithUser(events, "user2")
	if !found2 {
		t.Fatal("Expected to find user2's clusters")
	}
	if got2[0].TemplateID != "user2" {
		t.Errorf("User2 TemplateID = %q, want 'user2'", got2[0].TemplateID)
	}

	// Verify they are isolated
	if got1[0].Count == got2[0].Count {
		t.Error("User caches should be isolated")
	}
}

func TestShardedClusterCache_ClearUser(t *testing.T) {
	cache := NewShardedClusterCache(10, 5*time.Minute, 4)
	defer cache.Close()

	events := []interface{}{
		map[string]interface{}{"message": "test", "severity": "INFO"},
	}

	cache.SetWithUser(events, []*LogCluster{{TemplateID: "t1"}}, "user1")
	cache.SetWithUser(events, []*LogCluster{{TemplateID: "t2"}}, "user2")

	// Clear user1's cache
	cache.ClearUser("user1")

	// user1's cache should be gone
	_, found1 := cache.GetWithUser(events, "user1")
	if found1 {
		t.Error("User1's cache should be cleared")
	}

	// user2's cache should still exist
	_, found2 := cache.GetWithUser(events, "user2")
	if !found2 {
		t.Error("User2's cache should still exist")
	}
}

func TestShardedClusterCache_Concurrent(t *testing.T) {
	cache := NewShardedClusterCache(100, 5*time.Minute, 16)
	defer cache.Close()

	const goroutines = 50
	const operations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < operations; i++ {
				events := []interface{}{
					map[string]interface{}{
						"message":  "concurrent message",
						"severity": "INFO",
						"id":       id*operations + i,
					},
				}
				clusters := []*LogCluster{{TemplateID: "concurrent", Count: id}}

				// Mix of operations
				if i%2 == 0 {
					cache.Set(events, clusters)
				} else {
					cache.Get(events)
				}
			}
		}(g)
	}

	wg.Wait()

	stats := cache.Stats()
	if stats.TotalSets == 0 {
		t.Error("Expected some sets to have occurred")
	}
}

func TestShardedClusterCache_Stats(t *testing.T) {
	cache := NewShardedClusterCache(10, 5*time.Minute, 4)
	defer cache.Close()

	// Initial stats
	stats := cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Initial size = %d, want 0", stats.Size)
	}
	if stats.ShardCount != 4 {
		t.Errorf("ShardCount = %d, want 4", stats.ShardCount)
	}

	// Add some entries
	for i := 0; i < 5; i++ {
		events := []interface{}{
			map[string]interface{}{
				"message":  "msg " + string(rune('a'+i)),
				"severity": "INFO",
			},
		}
		cache.SetWithUser(events, []*LogCluster{{TemplateID: "t"}}, "user"+string(rune('a'+i)))
	}

	stats = cache.Stats()
	if stats.Size != 5 {
		t.Errorf("Size after adds = %d, want 5", stats.Size)
	}
	if stats.TotalSets != 5 {
		t.Errorf("TotalSets = %d, want 5", stats.TotalSets)
	}
	if stats.UserCount != 5 {
		t.Errorf("UserCount = %d, want 5", stats.UserCount)
	}
}

func TestClusterLogsWithCacheAndUser(t *testing.T) {
	ClearClusterCache()

	events := make([]interface{}, 20) // Above cache threshold
	for i := 0; i < 20; i++ {
		events[i] = map[string]interface{}{
			"message":  "user scoped cache test",
			"severity": "ERROR",
		}
	}

	// First call - cache miss
	clusters1 := ClusterLogsWithCacheAndUser(events, "testuser")
	if len(clusters1) == 0 {
		t.Fatal("Expected non-empty clusters")
	}

	// Second call - should be cache hit
	stats1 := GetClusterCacheStats()
	clusters2 := ClusterLogsWithCacheAndUser(events, "testuser")
	stats2 := GetClusterCacheStats()

	if stats2.TotalHits <= stats1.TotalHits {
		t.Error("Expected cache hit on second call")
	}

	if len(clusters1) != len(clusters2) {
		t.Error("Cached results should match original")
	}
}

// ============================================================================
// Parallel Benchmarks (SOTA 2025)
// ============================================================================

func BenchmarkShardedClusterCache_Concurrent_Set(b *testing.B) {
	cache := NewShardedClusterCache(10000, 5*time.Minute, 16)
	defer cache.Close()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			events := []interface{}{
				map[string]interface{}{
					"message":  "parallel bench",
					"severity": "ERROR",
					"id":       i,
				},
			}
			clusters := []*LogCluster{{TemplateID: "bench"}}
			cache.Set(events, clusters)
			i++
		}
	})
}

func BenchmarkShardedClusterCache_Concurrent_GetSet(b *testing.B) {
	cache := NewShardedClusterCache(10000, 5*time.Minute, 16)
	defer cache.Close()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		events := []interface{}{
			map[string]interface{}{
				"message":  "pre-populate",
				"severity": "INFO",
				"id":       i % 100,
			},
		}
		cache.Set(events, []*LogCluster{{TemplateID: "prepop"}})
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			events := []interface{}{
				map[string]interface{}{
					"message":  "mixed bench",
					"severity": "INFO",
					"id":       i % 100,
				},
			}
			if i%10 == 0 {
				cache.Set(events, []*LogCluster{{TemplateID: "mixed"}})
			} else {
				cache.Get(events)
			}
			i++
		}
	})
}
