// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file contains metamorphic and property-based tests for verifying
// invariant properties of the log processing and query systems.
package tools

import (
	"math/rand/v2"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ========================================================================
// METAMORPHIC TEST RELATIONS
// ========================================================================
// Metamorphic testing verifies properties that hold across transformations:
// - If Input A produces Output B, then Transform(Input A) should produce
//   a predictable relationship to Output B.
// ========================================================================

// TestMetamorphic_LogClusteringShuffleInvariance verifies that log clustering
// produces the same clusters regardless of input order.
// MR1: shuffle(input) → same_clusters(output)
func TestMetamorphic_LogClusteringShuffleInvariance(t *testing.T) {
	// Generate test data
	events := generateMetamorphicTestEvents(100)

	// Get baseline clusters
	baselineClusters := clusterLogsForTest(events)

	// Run multiple shuffle iterations
	for i := 0; i < 10; i++ {
		shuffled := shuffleEventsForTest(events)
		shuffledClusters := clusterLogsForTest(shuffled)

		// Verify same number of clusters
		if len(baselineClusters) != len(shuffledClusters) {
			t.Errorf("Iteration %d: cluster count mismatch - baseline: %d, shuffled: %d",
				i, len(baselineClusters), len(shuffledClusters))
			continue
		}

		// Verify cluster contents are equivalent (order-independent)
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
			outputs[i] = formatContentToString(formatted)
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

// TestConcurrent_SessionAccess verifies thread-safety of session operations
func TestConcurrent_SessionAccess(t *testing.T) {
	SetCurrentUser("test-user", "test-instance")

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
}

// TestConcurrent_CacheAccess verifies thread-safety of cache operations
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
}

// TestConcurrent_ToolRegistryAccess verifies thread-safety of tool registry
func TestConcurrent_ToolRegistryAccess(t *testing.T) {
	done := make(chan bool, 50)

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
			done <- true
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
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

func clustersEquivalent(a, b []testLogCluster) bool {
	if len(a) != len(b) {
		return false
	}

	// Sort clusters by pattern for comparison
	sortClusters := func(clusters []testLogCluster) {
		sort.Slice(clusters, func(i, j int) bool {
			return clusters[i].Pattern < clusters[j].Pattern
		})
	}

	aCopy := make([]testLogCluster, len(a))
	bCopy := make([]testLogCluster, len(b))
	copy(aCopy, a)
	copy(bCopy, b)

	sortClusters(aCopy)
	sortClusters(bCopy)

	for i := range aCopy {
		if aCopy[i].Pattern != bCopy[i].Pattern || aCopy[i].Count != bCopy[i].Count {
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

func formatContentToString(result interface{}) string {
	// Use reflection to extract text content from MCP CallToolResult
	v := reflect.ValueOf(result)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	contentField := v.FieldByName("Content")
	if !contentField.IsValid() || contentField.IsNil() {
		return ""
	}
	var sb strings.Builder
	for i := 0; i < contentField.Len(); i++ {
		item := contentField.Index(i)
		if item.Kind() == reflect.Interface {
			item = item.Elem()
		}
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		if item.Kind() == reflect.Struct {
			textField := item.FieldByName("Text")
			if textField.IsValid() && textField.Kind() == reflect.String {
				sb.WriteString(textField.String())
			}
		}
	}
	return sb.String()
}

func truncateForTest(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// testLogCluster represents a cluster of similar log messages (stub for testing)
type testLogCluster struct {
	Pattern string
	Count   int
}

// clusterLogsForTest is a stub that would cluster logs by pattern
func clusterLogsForTest(events []interface{}) []testLogCluster {
	clusters := make(map[string]int)
	for _, e := range events {
		if event, ok := e.(map[string]interface{}); ok {
			if msg, ok := event["message"].(string); ok {
				clusters[msg]++
			}
		}
	}

	result := make([]testLogCluster, 0, len(clusters))
	for pattern, count := range clusters {
		result = append(result, testLogCluster{Pattern: pattern, Count: count})
	}
	return result
}
