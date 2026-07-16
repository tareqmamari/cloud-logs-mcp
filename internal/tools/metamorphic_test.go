// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file contains metamorphic and property-based tests for verifying
// invariant properties of the log processing and query systems.
package tools

import (
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ========================================================================
// METAMORPHIC TEST RELATIONS
// ========================================================================
// Metamorphic testing verifies properties that hold across transformations:
// - If Input A produces Output B, then Transform(Input A) should produce
//   a predictable relationship to Output B.
// ========================================================================

// TestMetamorphic_LogClusteringShuffleInvariance verifies that log clustering
// produces the same clusters regardless of input order. This exercises the
// production ClusterLogs (internal/tools/response.go), not a test stub.
// MR1: shuffle(input) → same_clusters(output)
func TestMetamorphic_LogClusteringShuffleInvariance(t *testing.T) {
	// Generate test data
	events := generateMetamorphicTestEvents(100)

	// Get baseline clusters from the production clustering function
	baselineClusters := ClusterLogs(events)
	if len(baselineClusters) == 0 {
		t.Fatal("expected at least one cluster from generated test events")
	}

	// Run multiple shuffle iterations
	for i := 0; i < 10; i++ {
		shuffled := shuffleEventsForTest(events)
		shuffledClusters := ClusterLogs(shuffled)

		// Verify same number of clusters
		if len(baselineClusters) != len(shuffledClusters) {
			t.Errorf("Iteration %d: cluster count mismatch - baseline: %d, shuffled: %d",
				i, len(baselineClusters), len(shuffledClusters))
			continue
		}

		// Verify cluster contents are equivalent (order-independent): the same
		// pattern/count/severity multiset must come out regardless of input order.
		if !clustersEquivalent(baselineClusters, shuffledClusters) {
			t.Errorf("Iteration %d: clusters are not equivalent after shuffle", i)
		}
	}
}

// TestMetamorphic_QueryResultsSubsetInvariance verifies that filtering
// query results produces a strict subset with preserved properties.
// MR2: filter(results, condition) ⊆ results
func TestMetamorphic_QueryResultsSubsetInvariance(t *testing.T) {
	// Create mock query results
	fullResults := createMockQueryResultForTest(100)

	// Clean results to get standardized format
	cleanedFull := CleanQueryResults(fullResults)

	// Apply various filters
	filterTests := []struct {
		name      string
		filterFn  func(map[string]interface{}) map[string]interface{}
		invariant func(full, filtered map[string]interface{}) bool
	}{
		{
			name: "limit_reduction",
			filterFn: func(r map[string]interface{}) map[string]interface{} {
				return limitResults(r, 50)
			},
			invariant: func(full, filtered map[string]interface{}) bool {
				fullEvents := getEvents(full)
				filteredEvents := getEvents(filtered)
				return len(filteredEvents) <= len(fullEvents)
			},
		},
		{
			name: "severity_filter",
			filterFn: func(r map[string]interface{}) map[string]interface{} {
				return filterBySeverity(r, 5) // ERROR and above
			},
			invariant: func(full, filtered map[string]interface{}) bool {
				fullEvents := getEvents(full)
				filteredEvents := getEvents(filtered)
				// Filtered set must be a subset of the full set
				if len(filteredEvents) > len(fullEvents) {
					return false
				}
				for _, e := range filteredEvents {
					if event, ok := e.(map[string]interface{}); ok {
						if sev := getSeverity(event); sev < 5 {
							return false
						}
					}
				}
				return true
			},
		},
	}

	for _, tt := range filterTests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := tt.filterFn(cleanedFull)
			if !tt.invariant(cleanedFull, filtered) {
				t.Errorf("Invariant violated for %s filter", tt.name)
			}
		})
	}
}

// TestMetamorphic_QueryPreparationIdempotence verifies that preparing
// a query multiple times produces the same result.
// MR3: prepare(prepare(query)) == prepare(query)
func TestMetamorphic_QueryPreparationIdempotence(t *testing.T) {
	queries := []string{
		"source logs | filter $m.severity >= 5 | limit 100",
		"source logs | filter applicationname == 'test'", // Will be auto-corrected
		"source logs | filter $l.applicationname == 'myapp' && $m.severity >= 4",
	}

	for _, query := range queries {
		// First preparation
		prepared1, _, err1 := PrepareQuery(query, "archive", "dataprime")
		if err1 != nil {
			t.Logf("Query preparation failed (expected for some): %v", err1)
			continue
		}

		// Second preparation (should be idempotent)
		prepared2, _, err2 := PrepareQuery(prepared1, "archive", "dataprime")
		if err2 != nil {
			t.Errorf("Second preparation failed: %v", err2)
			continue
		}

		// Verify idempotence
		if prepared1 != prepared2 {
			t.Errorf("Query preparation is not idempotent:\n  First:  %s\n  Second: %s", prepared1, prepared2)
		}
	}
}

// TestMetamorphic_TokenEstimationMonotonicity verifies that token estimation
// is monotonically increasing with input size.
// MR4: len(A) < len(B) → tokens(A) <= tokens(B)
func TestMetamorphic_TokenEstimationMonotonicity(t *testing.T) {
	baseString := "This is a test string for token estimation."

	previousTokens := 0
	for multiplier := 1; multiplier <= 10; multiplier++ {
		input := strings.Repeat(baseString, multiplier)
		tokens := EstimateTokens(input)

		if tokens < previousTokens {
			t.Errorf("Token estimation not monotonic: %d tokens for multiplier %d, but %d tokens for previous",
				tokens, multiplier, previousTokens)
		}
		previousTokens = tokens
	}
}

// TestMetamorphic_ResponseFormattingDeterminism verifies that response
// formatting produces deterministic output for the same input.
// MR5: format(input) == format(input) (always)
func TestMetamorphic_ResponseFormattingDeterminism(t *testing.T) {
	result := createMockQueryResultForTest(50)
	tool := &BaseTool{}

	// Generate multiple formatting attempts
	outputs := make([]string, 5)
	for i := 0; i < 5; i++ {
		formatted, err := tool.FormatResponse(result)
		if err != nil {
			t.Fatalf("Formatting failed: %v", err)
		}
		if len(formatted.Content) > 0 {
			// Extract text content for comparison
			outputs[i] = formatContentToString(t, formatted)
		}
	}

	// Verify all outputs are identical
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("Non-deterministic formatting detected:\n  Attempt 0: %s...\n  Attempt %d: %s...",
				truncateForTest(outputs[0], 100), i, truncateForTest(outputs[i], 100))
		}
	}
}

// TestMetamorphic_LogClusteringOrderDeterminism verifies that ClusterLogs
// returns clusters in a fully deterministic order (not just an equivalent
// set) regardless of input order, including ties in Count. Rewriting
// TestMetamorphic_LogClusteringShuffleInvariance to call the production
// ClusterLogs (instead of the local stub it previously tested) surfaced a
// real bug: sort.Slice is not stable, and clusters with equal counts fell
// back to first-encounter order in the input slice, so shuffling the input
// could reorder tied clusters in the output. ClusterLogs now breaks ties by
// Pattern (see internal/tools/response.go), which this test locks in.
func TestMetamorphic_LogClusteringOrderDeterminism(t *testing.T) {
	events := generateMetamorphicTestEvents(100)
	baseline := ClusterLogs(events)
	if len(baseline) == 0 {
		t.Fatal("expected at least one cluster from generated test events")
	}

	for i := 0; i < 10; i++ {
		shuffled := shuffleEventsForTest(events)
		got := ClusterLogs(shuffled)

		if len(got) != len(baseline) {
			t.Fatalf("iteration %d: cluster count mismatch - baseline: %d, shuffled: %d", i, len(baseline), len(got))
		}
		for idx := range baseline {
			if baseline[idx].Pattern != got[idx].Pattern || baseline[idx].Count != got[idx].Count {
				t.Errorf("iteration %d: cluster order not deterministic at index %d: baseline=%q(%d) got=%q(%d)",
					i, idx, baseline[idx].Pattern, baseline[idx].Count, got[idx].Pattern, got[idx].Count)
			}
		}
	}
}

// ========================================================================
// PROPERTY-BASED TESTS
// ========================================================================

// TestProperty_PaginationBounds verifies pagination parameter bounds
func TestProperty_PaginationBounds(t *testing.T) {
	testCases := []struct {
		input    map[string]interface{}
		minLimit int
		maxLimit int
	}{
		{map[string]interface{}{"limit": 0}, 1, 100},
		{map[string]interface{}{"limit": -1}, 1, 100},
		{map[string]interface{}{"limit": 200}, 1, 100},
		{map[string]interface{}{"limit": 50}, 1, 100},
		{map[string]interface{}{}, 1, 100}, // Default case
	}

	for _, tc := range testCases {
		params, err := GetPaginationParams(tc.input)
		if err != nil {
			t.Errorf("GetPaginationParams failed: %v", err)
			continue
		}

		limit := params["limit"].(int)
		if limit < tc.minLimit || limit > tc.maxLimit {
			t.Errorf("Limit %d out of bounds [%d, %d] for input %v",
				limit, tc.minLimit, tc.maxLimit, tc.input)
		}
	}
}

// TestProperty_StringParamTypeCoercion verifies type coercion properties
func TestProperty_StringParamTypeCoercion(t *testing.T) {
	testCases := []struct {
		input    interface{}
		expected string
		valid    bool
	}{
		{"hello", "hello", true},
		{123, "123", true},
		{123.456, "123.456", true},
		{int64(999), "999", true},
		{true, "", false},
		{nil, "", false},
	}

	for _, tc := range testCases {
		args := map[string]interface{}{"key": tc.input}
		result, err := GetStringParam(args, "key", false)

		if tc.valid {
			if err != nil {
				t.Errorf("Expected valid conversion for %v, got error: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("Expected %s for input %v, got %s", tc.expected, tc.input, result)
			}
		} else {
			if err == nil && tc.input != nil {
				t.Errorf("Expected error for input %v, got success", tc.input)
			}
		}
	}
}

// TestProperty_ValidationResultConsistency verifies validation result properties
func TestProperty_ValidationResultConsistency(t *testing.T) {
	validator := NewResourceValidator("test", []string{"name"})
	validator.AddFieldValidator("name", FieldValidator{Type: "string", MinLength: 1, MaxLength: 100})

	testCases := []struct {
		config      map[string]interface{}
		expectValid bool
	}{
		{map[string]interface{}{"name": "valid"}, true},
		{map[string]interface{}{"name": ""}, false},
		{map[string]interface{}{}, false},
		{map[string]interface{}{"name": strings.Repeat("a", 101)}, false},
	}

	for _, tc := range testCases {
		result := validator.Validate(tc.config)

		// Property: Valid == true implies no errors
		if result.Valid && len(result.Errors) > 0 {
			t.Errorf("Inconsistent validation: Valid=true but has errors: %v", result.Errors)
		}

		// Property: Valid == false implies at least one error
		if !result.Valid && len(result.Errors) == 0 {
			t.Errorf("Inconsistent validation: Valid=false but no errors")
		}

		// Check expected validity
		if result.Valid != tc.expectValid {
			t.Errorf("Expected Valid=%v for config %v, got Valid=%v",
				tc.expectValid, tc.config, result.Valid)
		}
	}
}

// ========================================================================
// RACE DETECTION TESTS
// ========================================================================

// TestConcurrent_SessionAccess verifies thread-safety of session operations.
// -race is the primary guard against data races here; the post-conditions
// below additionally verify no writes were silently lost under concurrency
// (which -race alone would not catch, since a lost update from a correctly
// locked-but-buggy method isn't a race).
func TestConcurrent_SessionAccess(t *testing.T) {
	SetCurrentUser("test-user", "test-instance")
	session := GetSession()

	// TotalToolCalls is incremented once per RecordToolUse call (unlike
	// RecentTools, which is capped at 20 entries), so it's a reliable counter
	// for "did every concurrent write actually land".
	before := session.LearnedPatterns.TotalToolCalls

	// Run concurrent operations
	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func(id int) {
			session := GetSession()
			session.RecordToolUse("test_tool", true, nil)
			lastQuery := session.GetLastQuery()
			session.SetLastQuery("test query " + strconv.Itoa(id))
			// Verify session operations don't panic under concurrency
			if session == nil {
				t.Error("session should not be nil")
			}
			// Use lastQuery to verify reads work concurrently
			if lastQuery == "IMPOSSIBLE_SENTINEL" {
				t.Error("unexpected sentinel value")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Post-condition: all 100 RecordToolUse calls must be reflected exactly
	// once each - a lost update here would mean the mutex isn't actually
	// protecting TotalToolCalls end-to-end.
	if got, want := session.LearnedPatterns.TotalToolCalls-before, 100; got != want {
		t.Errorf("expected TotalToolCalls to increase by %d after 100 concurrent RecordToolUse calls, got %d", want, got)
	}

	// Post-condition: the last SetLastQuery call (whichever goroutine won the
	// race) must have actually landed - not been dropped or corrupted.
	lastQuery := session.GetLastQuery()
	if !strings.HasPrefix(lastQuery, "test query ") {
		t.Errorf("expected LastQuery to be one of the concurrently-written \"test query N\" values, got %q", lastQuery)
	}
}

// TestConcurrent_CacheAccess verifies thread-safety of cache operations.
// -race is the primary guard; the post-condition below additionally checks
// that every key written by the 100 concurrent goroutines is still present
// and well-formed afterward (a lost or corrupted entry wouldn't necessarily
// race but would still be a real bug).
func TestConcurrent_CacheAccess(t *testing.T) {
	cacheHelper := GetCacheHelper()

	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func(id int) {
			key := "key_" + string(rune('0'+id%10))
			cacheHelper.Set("test_tool", key, map[string]interface{}{"id": id})
			val, ok := cacheHelper.Get("test_tool", key)
			// Verify cache operations work concurrently without panics
			if ok && val == nil {
				t.Error("cached value should not be nil when found")
			}
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	// Post-condition: all 10 distinct keys (id%10) written across the 100
	// goroutines must still be retrievable and hold a well-formed value.
	for i := 0; i < 10; i++ {
		key := "key_" + string(rune('0'+i))
		val, ok := cacheHelper.Get("test_tool", key)
		if !ok {
			t.Errorf("expected key %q to be present after concurrent writes, but it was missing", key)
			continue
		}
		m, ok := val.(map[string]interface{})
		if !ok {
			t.Errorf("expected value for key %q to be a map[string]interface{}, got %T", key, val)
			continue
		}
		if _, hasID := m["id"]; !hasID {
			t.Errorf("expected value for key %q to contain an \"id\" field, got %v", key, m)
		}
	}
}

// TestConcurrent_ToolRegistryAccess verifies thread-safety of tool registry
// reads. The registry is populated once at startup and treated as read-only
// thereafter, so every concurrent caller must observe identical results;
// -race guards against a hidden write, and the post-condition below checks
// that concurrent reads didn't observe a torn/partial view of the maps.
func TestConcurrent_ToolRegistryAccess(t *testing.T) {
	done := make(chan bool, 50)
	nameCounts := make(chan int, 50)
	toolCounts := make(chan int, 50)

	for i := 0; i < 50; i++ {
		go func() {
			names := GetAllToolNames()
			tools := GetToolsByCategory(string(CategoryQuery))
			// Verify registry returns consistent data under concurrency
			if len(names) == 0 {
				t.Error("expected at least one tool name")
			}
			if len(tools) == 0 {
				t.Error("expected at least one query tool")
			}
			nameCounts <- len(names)
			toolCounts <- len(tools)
			done <- true
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
	}
	close(nameCounts)
	close(toolCounts)

	// Post-condition: every concurrent caller must see the same counts. The
	// registry is never mutated after init, so any divergence here would mean
	// a caller observed a torn/partial read of the underlying maps/slices.
	first := -1
	for n := range nameCounts {
		if first == -1 {
			first = n
			continue
		}
		if n != first {
			t.Errorf("GetAllToolNames returned inconsistent counts under concurrency: %d vs %d", first, n)
		}
	}
	firstTools := -1
	for n := range toolCounts {
		if firstTools == -1 {
			firstTools = n
			continue
		}
		if n != firstTools {
			t.Errorf("GetToolsByCategory returned inconsistent counts under concurrency: %d vs %d", firstTools, n)
		}
	}
}

// ========================================================================
// HELPER FUNCTIONS
// ========================================================================

func generateMetamorphicTestEvents(count int) []interface{} {
	events := make([]interface{}, count)
	messages := []string{
		"Connection timeout to database",
		"Authentication failed for user",
		"Rate limit exceeded",
		"Memory allocation failed",
		"Disk space warning",
	}

	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)) //nolint:gosec // test-only randomness
	for i := 0; i < count; i++ {
		events[i] = map[string]interface{}{
			"message":  messages[rng.IntN(len(messages))],
			"severity": float64(rng.IntN(6) + 1),
			"timestamp": time.Now().Add(-time.Duration(rng.IntN(3600)) * time.Second).
				Format(time.RFC3339),
		}
	}
	return events
}

func shuffleEventsForTest(events []interface{}) []interface{} {
	shuffled := make([]interface{}, len(events))
	copy(shuffled, events)

	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)) //nolint:gosec // test-only randomness
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled
}

// clustersEquivalent compares two ClusterLogs results as sets, ignoring
// slice order: the same Pattern/Count/Severity multiset must be present in
// both, regardless of what order ClusterLogs happened to return them in.
func clustersEquivalent(a, b []LogCluster) bool {
	if len(a) != len(b) {
		return false
	}

	// Sort clusters by pattern for comparison
	sortClusters := func(clusters []LogCluster) {
		sort.Slice(clusters, func(i, j int) bool {
			return clusters[i].Pattern < clusters[j].Pattern
		})
	}

	aCopy := make([]LogCluster, len(a))
	bCopy := make([]LogCluster, len(b))
	copy(aCopy, a)
	copy(bCopy, b)

	sortClusters(aCopy)
	sortClusters(bCopy)

	for i := range aCopy {
		if aCopy[i].Pattern != bCopy[i].Pattern || aCopy[i].Count != bCopy[i].Count || aCopy[i].Severity != bCopy[i].Severity {
			return false
		}
	}

	return true
}

func createMockQueryResultForTest(count int) map[string]interface{} {
	events := make([]interface{}, count)
	for i := 0; i < count; i++ {
		events[i] = map[string]interface{}{
			"labels": map[string]interface{}{
				"applicationname": "test-app",
			},
			"metadata": map[string]interface{}{
				"severity":  float64((i % 6) + 1),
				"timestamp": time.Now().Format(time.RFC3339),
			},
			"user_data": `{"message": "Test message"}`,
		}
	}
	return map[string]interface{}{"events": events}
}

func getEvents(result map[string]interface{}) []interface{} {
	if events, ok := result["events"].([]interface{}); ok {
		return events
	}
	return nil
}

func getSeverity(event map[string]interface{}) float64 {
	if meta, ok := event["metadata"].(map[string]interface{}); ok {
		if sev, ok := meta["severity"].(float64); ok {
			return sev
		}
	}
	return 0
}

func limitResults(result map[string]interface{}, limit int) map[string]interface{} {
	events := getEvents(result)
	if len(events) > limit {
		events = events[:limit]
	}
	return map[string]interface{}{"events": events}
}

func filterBySeverity(result map[string]interface{}, minSeverity float64) map[string]interface{} {
	events := getEvents(result)
	filtered := make([]interface{}, 0)
	for _, e := range events {
		if event, ok := e.(map[string]interface{}); ok {
			if getSeverity(event) >= minSeverity {
				filtered = append(filtered, e)
			}
		}
	}
	return map[string]interface{}{"events": filtered}
}

// formatContentToString extracts the text of an MCP CallToolResult's content
// items, failing the test outright if a content item isn't the
// *mcp.TextContent shape every FormatResponse* helper in this package
// produces. The previous implementation used reflection and silently
// returned "" on any shape mismatch, which meant a caller could compare two
// empty strings and see a spurious pass instead of a failure.
func formatContentToString(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	if result == nil {
		t.Fatal("formatContentToString: result is nil")
	}

	var sb strings.Builder
	for i, item := range result.Content {
		textContent, ok := item.(*mcp.TextContent)
		if !ok {
			t.Fatalf("formatContentToString: content[%d] has unexpected type %T; want *mcp.TextContent", i, item)
		}
		sb.WriteString(textContent.Text)
	}
	return sb.String()
}

func truncateForTest(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
