package cache

import (
	"sync"
	"testing"
	"time"
)

func TestUserCache(t *testing.T) {
	cache := NewUserCache(10)

	// Test basic set/get
	cache.Set("key1", "value1", 5*time.Minute)
	val, ok := cache.Get("key1")
	if !ok {
		t.Error("Expected to find key1")
	}
	if val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}

	// Test missing key
	_, ok = cache.Get("nonexistent")
	if ok {
		t.Error("Expected not to find nonexistent key")
	}
}

func TestUserCacheExpiration(t *testing.T) {
	cache := NewUserCache(10)

	// Set with very short TTL
	cache.Set("expiring", "value", 1*time.Millisecond)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should not find expired entry
	_, ok := cache.Get("expiring")
	if ok {
		t.Error("Expected expired entry to be removed")
	}
}

func TestUserCacheDelete(t *testing.T) {
	cache := NewUserCache(10)

	cache.Set("key1", "value1", 5*time.Minute)
	cache.Set("key2", "value2", 5*time.Minute)

	cache.Delete("key1")

	_, ok := cache.Get("key1")
	if ok {
		t.Error("Expected key1 to be deleted")
	}

	_, ok = cache.Get("key2")
	if !ok {
		t.Error("Expected key2 to still exist")
	}
}

func TestUserCacheDeleteByPrefix(t *testing.T) {
	cache := NewUserCache(10)

	cache.Set("list_alerts:all", "alerts", 5*time.Minute)
	cache.Set("list_alerts:filtered", "filtered_alerts", 5*time.Minute)
	cache.Set("list_dashboards:all", "dashboards", 5*time.Minute)

	count := cache.DeleteByPrefix("list_alerts:")
	if count != 2 {
		t.Errorf("Expected 2 deletions, got %d", count)
	}

	_, ok := cache.Get("list_alerts:all")
	if ok {
		t.Error("Expected list_alerts:all to be deleted")
	}

	_, ok = cache.Get("list_dashboards:all")
	if !ok {
		t.Error("Expected list_dashboards:all to still exist")
	}
}

func TestUserCacheClear(t *testing.T) {
	cache := NewUserCache(10)

	cache.Set("key1", "value1", 5*time.Minute)
	cache.Set("key2", "value2", 5*time.Minute)

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected empty cache, got size %d", cache.Size())
	}
}

func TestUserCacheEviction(t *testing.T) {
	cache := NewUserCache(3) // Small cache

	cache.Set("key1", "value1", 5*time.Minute)
	cache.Set("key2", "value2", 5*time.Minute)
	cache.Set("key3", "value3", 5*time.Minute)
	cache.Set("key4", "value4", 5*time.Minute) // Should trigger eviction

	if cache.Size() > 3 {
		t.Errorf("Expected max 3 entries, got %d", cache.Size())
	}
}

func TestUserCacheEvictionIsTrueLRU(t *testing.T) {
	cache := NewUserCache(2) // capacity 2

	cache.Set("A", "valueA", 5*time.Minute)
	time.Sleep(2 * time.Millisecond)
	cache.Set("B", "valueB", 5*time.Minute)
	time.Sleep(2 * time.Millisecond)

	// Touch A so it becomes the most recently used entry.
	if _, ok := cache.Get("A"); !ok {
		t.Fatal("expected to find A")
	}
	time.Sleep(2 * time.Millisecond)

	// Inserting C at capacity should evict B (least recently used), not A.
	cache.Set("C", "valueC", 5*time.Minute)

	if _, ok := cache.Get("A"); !ok {
		t.Error("expected A to survive eviction (it was recently accessed)")
	}
	if _, ok := cache.Get("B"); ok {
		t.Error("expected B to be evicted as the least recently used entry")
	}
	if _, ok := cache.Get("C"); !ok {
		t.Error("expected C to be present")
	}
}

func TestUserCacheStats(t *testing.T) {
	cache := NewUserCache(10)

	cache.Set("key1", "value1", 5*time.Minute)
	cache.Set("key2", "value2", 5*time.Minute)

	// Access key1 twice
	cache.Get("key1")
	cache.Get("key1")

	stats := cache.Stats()
	if stats["size"].(int) != 2 {
		t.Errorf("Expected size 2, got %d", stats["size"])
	}
	if stats["total_hits"].(int) != 2 {
		t.Errorf("Expected 2 hits, got %d", stats["total_hits"])
	}
}

func TestUserCacheSweepsExpiredEntriesOnSetWithoutReads(t *testing.T) {
	cache := NewUserCache(10) // capacity well above the entries we insert
	// Sweeping is throttled to at most once per sweepInterval (see
	// TestUserCacheThrottlesExpiredSweepOnSet); a 0 interval preserves this
	// test's original intent of observing a sweep on every Set.
	cache.sweepInterval = 0

	cache.Set("expiring1", "v1", 1*time.Millisecond)
	cache.Set("expiring2", "v2", 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	// Trigger a sweep purely by writing a new, unrelated key. No Get call
	// is made on the expired keys, so lazy-on-read eviction can't be what
	// removes them.
	cache.Set("fresh", "v3", 5*time.Minute)

	if size := cache.Size(); size != 1 {
		t.Errorf("expected expired entries to be swept opportunistically on Set, got size %d", size)
	}
}

// TestUserCacheThrottlesExpiredSweepOnSet guards against Task 2's fix (a full
// expired-entry sweep on every Set) regressing back to O(n) per Set. The
// first Set after construction always sweeps (no prior sweep to throttle
// against), but a second Set within the throttle interval must NOT trigger
// another full scan.
func TestUserCacheThrottlesExpiredSweepOnSet(t *testing.T) {
	cache := NewUserCache(10)
	cache.sweepInterval = 1 * time.Hour // effectively "never sweep again" within this test

	// The very first Set on a virgin cache always sweeps (lastSweep is the
	// zero value), trivially - there's nothing to remove yet. This
	// establishes the throttle baseline for everything that follows.
	cache.Set("seed", "v0", 5*time.Minute)
	if got := cache.sweepRuns; got != 1 {
		t.Fatalf("expected the first-ever Set to run the baseline sweep, sweepRuns = %d, want 1", got)
	}

	cache.Set("expiring1", "v1", 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	// This Set falls well within the 1h throttle window, so it must NOT
	// trigger another full scan: sweepRuns stays at 1, and the now-expired
	// entry is still sitting in the map, unswept.
	cache.Set("fresh1", "v2", 5*time.Minute)

	if got := cache.sweepRuns; got != 1 {
		t.Errorf("expected the sweep to be throttled (still 1 total run), got %d", got)
	}
	cache.mu.RLock()
	_, stillPresent := cache.entries["expiring1"]
	cache.mu.RUnlock()
	if !stillPresent {
		t.Error("expected the expired-but-unswept entry to still be present within the throttle window")
	}
}

// TestUserCacheEventuallySweepsAfterThrottleInterval verifies Task 2's core
// guarantee - expired entries are removed on Set without any Get calls -
// still holds once the (now throttled) sweep interval has elapsed.
func TestUserCacheEventuallySweepsAfterThrottleInterval(t *testing.T) {
	cache := NewUserCache(10)
	cache.sweepInterval = 5 * time.Millisecond

	// Baseline sweep (trivial - nothing to remove yet), establishes lastSweep.
	cache.Set("seed", "v0", 5*time.Minute)

	cache.Set("expiring1", "v1", 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond) // both the TTL and the throttle interval elapse

	cache.Set("fresh1", "v2", 5*time.Minute)
	if got := cache.Size(); got != 2 {
		t.Errorf("expected the sweep (after the throttle interval elapsed) to remove the expired entry without any Get call, size = %d, want 2 (seed, fresh1)", got)
	}
	if _, exists := cache.entries["expiring1"]; exists {
		t.Error("expected expiring1 to have been swept once the throttle interval elapsed")
	}
}

func TestCacheManagerUserIsolation(t *testing.T) {
	config := &Config{
		MaxEntriesPerUser: 100,
		DefaultTTL:        5 * time.Minute,
		TTLByTool:         make(map[string]time.Duration),
		Enabled:           true,
	}
	manager := NewManager(config)

	// User1 sets a value
	manager.Set("user1", "instance1", "list_alerts", "all", "user1_alerts")

	// User2 sets a different value for the same key
	manager.Set("user2", "instance1", "list_alerts", "all", "user2_alerts")

	// Verify user1 sees their own value
	val, ok := manager.Get("user1", "instance1", "list_alerts", "all")
	if !ok {
		t.Error("Expected user1 to have cached value")
	}
	if val != "user1_alerts" {
		t.Errorf("Expected user1_alerts, got %v", val)
	}

	// Verify user2 sees their own value
	val, ok = manager.Get("user2", "instance1", "list_alerts", "all")
	if !ok {
		t.Error("Expected user2 to have cached value")
	}
	if val != "user2_alerts" {
		t.Errorf("Expected user2_alerts, got %v", val)
	}
}

func TestCacheManagerInstanceIsolation(t *testing.T) {
	config := &Config{
		MaxEntriesPerUser: 100,
		DefaultTTL:        5 * time.Minute,
		TTLByTool:         make(map[string]time.Duration),
		Enabled:           true,
	}
	manager := NewManager(config)

	// Same user, different instances
	manager.Set("user1", "instance1", "list_alerts", "all", "instance1_alerts")
	manager.Set("user1", "instance2", "list_alerts", "all", "instance2_alerts")

	// Verify instance1 has its own value
	val, ok := manager.Get("user1", "instance1", "list_alerts", "all")
	if !ok {
		t.Error("Expected instance1 to have cached value")
	}
	if val != "instance1_alerts" {
		t.Errorf("Expected instance1_alerts, got %v", val)
	}

	// Verify instance2 has its own value
	val, ok = manager.Get("user1", "instance2", "list_alerts", "all")
	if !ok {
		t.Error("Expected instance2 to have cached value")
	}
	if val != "instance2_alerts" {
		t.Errorf("Expected instance2_alerts, got %v", val)
	}
}

func TestCacheManagerInvalidateTool(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	manager.Set("user1", "instance1", "list_alerts", "all", "alerts")
	manager.Set("user1", "instance1", "list_alerts", "filtered", "filtered_alerts")
	manager.Set("user1", "instance1", "list_dashboards", "all", "dashboards")

	count := manager.InvalidateTool("user1", "instance1", "list_alerts")
	if count != 2 {
		t.Errorf("Expected 2 invalidations, got %d", count)
	}

	_, ok := manager.Get("user1", "instance1", "list_alerts", "all")
	if ok {
		t.Error("Expected list_alerts:all to be invalidated")
	}

	_, ok = manager.Get("user1", "instance1", "list_dashboards", "all")
	if !ok {
		t.Error("Expected list_dashboards:all to still be cached")
	}
}

func TestCacheManagerInvalidateRelated(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	manager.Set("user1", "instance1", "list_alerts", "all", "alerts")
	manager.Set("user1", "instance1", "get_alert", "alert123", "alert_detail")
	manager.Set("user1", "instance1", "suggest_alert", "errors", "suggestions")
	manager.Set("user1", "instance1", "list_dashboards", "all", "dashboards")

	// Creating an alert should invalidate alert-related caches
	manager.InvalidateRelated("user1", "instance1", "create_alert")

	_, ok := manager.Get("user1", "instance1", "list_alerts", "all")
	if ok {
		t.Error("Expected list_alerts to be invalidated")
	}

	_, ok = manager.Get("user1", "instance1", "get_alert", "alert123")
	if ok {
		t.Error("Expected get_alert to be invalidated")
	}

	_, ok = manager.Get("user1", "instance1", "suggest_alert", "errors")
	if ok {
		t.Error("Expected suggest_alert to be invalidated")
	}

	// Dashboards should not be affected
	_, ok = manager.Get("user1", "instance1", "list_dashboards", "all")
	if !ok {
		t.Error("Expected list_dashboards to still be cached")
	}
}

func TestCacheManagerDisabled(t *testing.T) {
	config := &Config{
		MaxEntriesPerUser: 100,
		DefaultTTL:        5 * time.Minute,
		TTLByTool:         make(map[string]time.Duration),
		Enabled:           false, // Disabled
	}
	manager := NewManager(config)

	manager.Set("user1", "instance1", "list_alerts", "all", "alerts")

	_, ok := manager.Get("user1", "instance1", "list_alerts", "all")
	if ok {
		t.Error("Expected cache to be disabled")
	}
}

func TestCacheManagerClearUser(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	manager.Set("user1", "instance1", "list_alerts", "all", "alerts")
	manager.Set("user1", "instance1", "list_dashboards", "all", "dashboards")
	manager.Set("user2", "instance1", "list_alerts", "all", "user2_alerts")

	manager.ClearUser("user1", "instance1")

	_, ok := manager.Get("user1", "instance1", "list_alerts", "all")
	if ok {
		t.Error("Expected user1 cache to be cleared")
	}

	// User2 should be unaffected
	_, ok = manager.Get("user2", "instance1", "list_alerts", "all")
	if !ok {
		t.Error("Expected user2 cache to be intact")
	}
}

func TestCacheManagerToolTTL(t *testing.T) {
	config := &Config{
		MaxEntriesPerUser: 100,
		DefaultTTL:        5 * time.Minute,
		TTLByTool: map[string]time.Duration{
			"fast_expiring": 1 * time.Millisecond,
			"slow_expiring": 1 * time.Hour,
		},
		Enabled: true,
	}
	manager := NewManager(config)

	manager.Set("user1", "instance1", "fast_expiring", "key", "fast_value")
	manager.Set("user1", "instance1", "slow_expiring", "key", "slow_value")

	// Wait for fast_expiring to expire
	time.Sleep(10 * time.Millisecond)

	_, ok := manager.Get("user1", "instance1", "fast_expiring", "key")
	if ok {
		t.Error("Expected fast_expiring entry to be expired")
	}

	_, ok = manager.Get("user1", "instance1", "slow_expiring", "key")
	if !ok {
		t.Error("Expected slow_expiring entry to still be valid")
	}
}

func TestCacheManagerGlobalStats(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	manager.Set("user1", "instance1", "list_alerts", "all", "alerts")
	manager.Set("user2", "instance1", "list_alerts", "all", "alerts")
	manager.Set("user1", "instance2", "list_dashboards", "all", "dashboards")

	stats := manager.GlobalStats()
	if stats["user_cache_count"].(int) != 3 {
		t.Errorf("Expected 3 user caches, got %d", stats["user_cache_count"])
	}
	if stats["total_entries"].(int) != 3 {
		t.Errorf("Expected 3 total entries, got %d", stats["total_entries"])
	}
}

func TestCacheManagerConcurrency(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	// Run concurrent operations
	done := make(chan bool, 100)

	for i := 0; i < 50; i++ {
		go func(id int) {
			userID := "user1"
			if id%2 == 0 {
				userID = "user2"
			}
			manager.Set(userID, "instance1", "list_alerts", "all", "alerts")
			manager.Get(userID, "instance1", "list_alerts", "all")
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	// Verify no panics and caches are consistent
	stats := manager.GlobalStats()
	if stats["user_cache_count"].(int) != 2 {
		t.Errorf("Expected 2 user caches, got %d", stats["user_cache_count"])
	}
}

func TestManagerSetEnabledConcurrentAccess(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Writer: continuously flips Enabled while readers read it via Get/Set/IsEnabled.
	wg.Add(1)
	go func() {
		defer wg.Done()
		enabled := true
		for {
			select {
			case <-stop:
				return
			default:
				manager.SetEnabled(enabled)
				enabled = !enabled
			}
		}
	}()

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				manager.Set("user1", "instance1", "list_alerts", "all", "alerts")
				manager.Get("user1", "instance1", "list_alerts", "all")
				manager.IsEnabled()
			}
		}()
	}

	time.Sleep(50 * time.Millisecond)
	close(stop)
	wg.Wait()

	// -race is the primary guard against data races here, but it wouldn't
	// catch a lost/corrupted final write from a correctly-locked-but-buggy
	// SetEnabled/Get/Set. Once every goroutine has stopped, force a
	// deterministic final state and verify it's observed exactly and the
	// cache is left in a consistent, readable condition.
	manager.SetEnabled(true)
	if !manager.IsEnabled() {
		t.Error("expected IsEnabled() to reflect the deterministic SetEnabled(true) call made after the concurrent storm stopped")
	}
	manager.Set("user1", "instance1", "list_alerts", "all", "final-value")
	if val, ok := manager.Get("user1", "instance1", "list_alerts", "all"); !ok || val != "final-value" {
		t.Errorf("expected the cache to be in a consistent, readable state after the concurrent storm; Get returned (%v, %v), want (\"final-value\", true)", val, ok)
	}
}

func TestGetManager(t *testing.T) {
	// Reset global for testing
	globalManager = nil
	globalManagerOnce = sync.Once{}

	manager1 := GetManager()
	if manager1 == nil {
		t.Error("Expected non-nil cache manager")
	}

	manager2 := GetManager()
	if manager1 != manager2 {
		t.Error("Expected singleton instance")
	}
}

func TestEntry(t *testing.T) {
	// Test non-expired entry
	entry := &Entry{
		Value:     "test",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
	if entry.IsExpired() {
		t.Error("Entry should not be expired")
	}

	// Test expired entry
	expiredEntry := &Entry{
		Value:     "test",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	if !expiredEntry.IsExpired() {
		t.Error("Entry should be expired")
	}
}
