// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file contains comprehensive benchmarks for measuring performance
// of key operations and validating refactoring improvements.
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// ========================================================================
// PHASE 1: Response Formatting Benchmarks
// ========================================================================

// BenchmarkFormatResponseComprehensive measures response formatting performance
func BenchmarkFormatResponseComprehensive(b *testing.B) {
	// Create a realistic query result
	result := createMockQueryResult(100)
	tool := &BaseTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tool.FormatResponse(result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCleanQueryResults measures query result cleaning performance
func BenchmarkCleanQueryResults(b *testing.B) {
	result := createMockQueryResult(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CleanQueryResults(result)
	}
}

// BenchmarkGenerateSummaryComprehensive measures summary generation performance
func BenchmarkGenerateSummaryComprehensive(b *testing.B) {
	result := createMockQueryResult(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateResultSummary(result, "query results")
	}
}

// BenchmarkTruncateComprehensive measures intelligent truncation performance
func BenchmarkTruncateComprehensive(b *testing.B) {
	result := createMockQueryResult(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = truncateResult(result, MaxResultSize)
	}
}

// ========================================================================
// PHASE 2: Query Processing Benchmarks
// ========================================================================

// BenchmarkPrepareQuery measures query preparation and validation
func BenchmarkPrepareQuery(b *testing.B) {
	testCases := []struct {
		name   string
		query  string
		tier   string
		syntax string
	}{
		{"simple", "source logs | filter $m.severity >= 5 | limit 100", "archive", "dataprime"},
		{"complex", "source logs | filter $l.applicationname == 'myapp' && $m.severity >= 4 | groupby $l.subsystemname aggregate count() as cnt | orderby cnt desc | limit 50", "frequent_search", "dataprime"},
		{"autocorrect", "source logs | filter applicationname == 'test'", "archive", "dataprime"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _, _ = PrepareQuery(tc.query, tc.tier, tc.syntax)
			}
		})
	}
}

// BenchmarkBuildQueryMetadata measures query metadata construction
func BenchmarkBuildQueryMetadata(b *testing.B) {
	args := map[string]interface{}{
		"query":      "source logs | filter $m.severity >= 5",
		"tier":       "archive",
		"syntax":     "dataprime",
		"start_date": "2024-01-01T00:00:00Z",
		"end_date":   "2024-01-02T00:00:00Z",
		"limit":      100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _ = buildQueryMetadata(args)
	}
}

// ========================================================================
// PHASE 3: Parameter Extraction Benchmarks
// ========================================================================

// BenchmarkGetStringParam measures parameter extraction performance
func BenchmarkGetStringParam(b *testing.B) {
	args := map[string]interface{}{
		"id":   "test-id-12345",
		"name": "test-resource",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetStringParam(args, "id", true)
	}
}

// BenchmarkGetPaginationParams measures pagination parameter extraction
func BenchmarkGetPaginationParams(b *testing.B) {
	args := map[string]interface{}{
		"limit":  50,
		"cursor": "abc123",
		"offset": 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetPaginationParams(args)
	}
}

// ========================================================================
// PHASE 4: Memory Allocation Benchmarks
// ========================================================================

// BenchmarkBufferPoolUsage compares pooled vs non-pooled buffer allocation
func BenchmarkBufferPoolUsage(b *testing.B) {
	b.Run("pooled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := compressionBufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			buf.WriteString("test data for compression benchmark")
			compressionBufferPool.Put(buf)
		}
	})

	b.Run("non-pooled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := new(bytes.Buffer)
			buf.WriteString("test data for compression benchmark")
			_ = buf // Prevent optimization
		}
	})
}

// BenchmarkMapAllocation compares map allocation patterns
func BenchmarkMapAllocation(b *testing.B) {
	b.Run("pre-allocated", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := make(map[string]interface{}, 16)
			m["key1"] = "value1"
			m["key2"] = "value2"
			m["key3"] = 123
			_ = m
		}
	})

	b.Run("zero-allocated", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := make(map[string]interface{})
			m["key1"] = "value1"
			m["key2"] = "value2"
			m["key3"] = 123
			_ = m
		}
	})
}

// ========================================================================
// PHASE 5: Concurrency Benchmarks
// ========================================================================

// BenchmarkConcurrentToolExecution measures tool execution under concurrent load
func BenchmarkConcurrentToolExecution(b *testing.B) {
	concurrencyLevels := []int{1, 4, 8, 16, 32}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrent_%d_workers", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				result := createMockQueryResult(50)
				tool := &BaseTool{}

				for pb.Next() {
					_, err := tool.FormatResponse(result)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

// BenchmarkSessionConcurrency measures session access under concurrent load
func BenchmarkSessionConcurrency(b *testing.B) {
	// Initialize session for benchmarking
	SetCurrentUser("benchmark-user", "benchmark-instance")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			session := GetSession()
			session.RecordToolUse("benchmark_tool", true, nil)
			_ = session.GetLastQuery()
		}
	})
}

// ========================================================================
// PHASE 6: JSON Processing Benchmarks
// ========================================================================

// BenchmarkJSONMarshal compares JSON marshaling approaches
func BenchmarkJSONMarshal(b *testing.B) {
	result := createMockQueryResult(100)

	b.Run("standard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(result)
		}
	})

	b.Run("indented", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.MarshalIndent(result, "", "  ")
		}
	})
}

// BenchmarkSSEParsing measures SSE response parsing
func BenchmarkSSEParsing(b *testing.B) {
	// Create mock SSE response
	var sseData strings.Builder
	for i := 0; i < 100; i++ {
		sseData.WriteString(`data: {"result":{"results":[{"labels":{"applicationname":"app1"},"metadata":{"severity":"ERROR","timestamp":"2024-01-01T12:00:00Z"},"user_data":"{\"message\":\"Test error message\"}"}]}}`)
		sseData.WriteString("\n\n")
	}
	sseBytes := []byte(sseData.String())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseSSEResponse(sseBytes)
	}
}

// ========================================================================
// PHASE 7: Log Clustering Benchmarks
// ========================================================================

// BenchmarkLogClustering measures log clustering performance
func BenchmarkLogClustering(b *testing.B) {
	events := createMockEvents(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ClusterLogs(events)
	}
}

// BenchmarkClusteredSummary measures clustered summary formatting
func BenchmarkClusteredSummary(b *testing.B) {
	events := createMockEvents(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FormatClusteredSummary(events, 10)
	}
}

// ========================================================================
// PHASE 8: Tool Discovery Benchmarks
// ========================================================================

// BenchmarkToolDiscovery measures tool discovery performance
func BenchmarkToolDiscovery(b *testing.B) {
	// Initialize registry with mock tools
	initMockRegistry()

	b.Run("by_category", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = GetToolsByCategory(string(CategoryQuery))
		}
	})

	b.Run("get_all_names", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = GetAllToolNames()
		}
	})
}

// ========================================================================
// PHASE 9: Compression Benchmarks
// ========================================================================

// BenchmarkCompression measures compression performance
func BenchmarkCompression(b *testing.B) {
	result := createMockQueryResult(500)

	b.Run("gzip", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, _ = CompressJSON(result)
		}
	})
}

// ========================================================================
// PHASE 10: Token Estimation Benchmarks
// ========================================================================

// BenchmarkTokenEstimation measures token estimation performance
func BenchmarkTokenEstimation(b *testing.B) {
	testStrings := []string{
		"Short string",
		strings.Repeat("Medium length string with some content. ", 10),
		strings.Repeat("Very long string with lots of content that needs to be estimated for tokens. This includes technical terms like Kubernetes, observability, and log analysis. ", 100),
	}

	for i, s := range testStrings {
		b.Run([]string{"short", "medium", "long"}[i], func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				_ = EstimateTokens(s)
			}
		})
	}
}

// ========================================================================
// Helper Functions
// ========================================================================

// createMockQueryResult creates a realistic query result for benchmarking
func createMockQueryResult(eventCount int) map[string]interface{} {
	events := make([]interface{}, eventCount)
	for i := 0; i < eventCount; i++ {
		events[i] = map[string]interface{}{
			"labels": map[string]interface{}{
				"applicationname": "benchmark-app",
				"subsystemname":   "api-gateway",
			},
			"metadata": map[string]interface{}{
				"severity":  float64(5),
				"timestamp": time.Now().Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
			},
			"user_data": `{"message": "Error processing request", "trace_id": "abc123", "span_id": "def456"}`,
		}
	}

	return map[string]interface{}{
		"events": events,
	}
}

// createMockEvents creates mock events for clustering benchmarks
func createMockEvents(count int) []interface{} {
	events := make([]interface{}, count)
	messages := []string{
		"Connection refused to database",
		"Request timeout after 30s",
		"Authentication failed for user",
		"Memory limit exceeded",
		"Disk space low warning",
	}

	for i := 0; i < count; i++ {
		events[i] = map[string]interface{}{
			"message":  messages[i%len(messages)],
			"severity": float64((i % 3) + 4), // 4-6 (Warning to Critical)
			"app":      "test-app",
		}
	}

	return events
}

// initMockRegistry initializes the tool registry with mock data
func initMockRegistry() {
	// This would initialize the global registry for benchmarking
	// In practice, the registry is already initialized by GetAllTools
	once.Do(func() {
		// Registry initialization happens automatically
	})
}

var once sync.Once

// ========================================================================
// Memory Profiling Benchmarks
// ========================================================================

// BenchmarkMemoryAllocation profiles memory usage patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("small_result", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := createMockQueryResult(10)
			_ = CleanQueryResults(result)
		}
	})

	b.Run("medium_result", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := createMockQueryResult(100)
			_ = CleanQueryResults(result)
		}
	})

	b.Run("large_result", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := createMockQueryResult(1000)
			_ = CleanQueryResults(result)
		}
	})
}

// ========================================================================
// Latency Distribution Benchmarks
// ========================================================================

// BenchmarkResponseLatency measures end-to-end response formatting latency
func BenchmarkResponseLatency(b *testing.B) {
	tool := &BaseTool{}
	result := createMockQueryResult(50)

	var totalDuration time.Duration
	var count int

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, _ = tool.FormatResponseWithSummary(result, "query results")
		totalDuration += time.Since(start)
		count++
	}

	if count > 0 {
		b.ReportMetric(float64(totalDuration.Nanoseconds()/int64(count)), "ns/op-avg")
	}
}

// ========================================================================
// Context Usage Benchmarks
// ========================================================================

// BenchmarkContextOperations measures context creation and extraction overhead
func BenchmarkContextOperations(b *testing.B) {
	ctx := context.Background()
	session := GetSession()

	b.Run("with_session", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx2 := WithSession(ctx, session)
			_ = GetSessionFromContext(ctx2)
		}
	})
}
