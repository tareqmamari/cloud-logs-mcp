package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestLogger(enabled bool) *Logger {
	return NewLogger(zap.NewNop(), enabled)
}

func TestNewLogger(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		l := newTestLogger(true)
		if l == nil {
			t.Fatal("expected non-nil logger")
		}
		if !l.enabled {
			t.Error("expected logger to be enabled")
		}
		if l.maxEntries != 1000 {
			t.Errorf("expected maxEntries=1000, got %d", l.maxEntries)
		}
	})

	t.Run("disabled", func(t *testing.T) {
		l := newTestLogger(false)
		if l == nil {
			t.Fatal("expected non-nil logger")
		}
		if l.enabled {
			t.Error("expected logger to be disabled")
		}
	})
}

func TestIsEnabled(t *testing.T) {
	if !newTestLogger(true).IsEnabled() {
		t.Error("expected IsEnabled()=true")
	}
	if newTestLogger(false).IsEnabled() {
		t.Error("expected IsEnabled()=false")
	}
}

func TestLog_Disabled(t *testing.T) {
	l := newTestLogger(false)
	l.Log(context.Background(), Entry{Tool: "test", Operation: "read"})

	entries := l.GetRecentEntries(0)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries when disabled, got %d", len(entries))
	}
}

func TestLog_BasicEntry(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	l.Log(ctx, Entry{
		Tool:      "search_logs",
		Operation: "query",
		Success:   true,
		Duration:  50 * time.Millisecond,
	})

	entries := l.GetRecentEntries(0)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Tool != "search_logs" {
		t.Errorf("expected tool=search_logs, got %s", entries[0].Tool)
	}
	if entries[0].Operation != "query" {
		t.Errorf("expected operation=query, got %s", entries[0].Operation)
	}
	if !entries[0].Success {
		t.Error("expected success=true")
	}
}

func TestLog_SetsTimestamp(t *testing.T) {
	l := newTestLogger(true)
	before := time.Now().UTC()

	l.Log(context.Background(), Entry{Tool: "test", Operation: "read"})

	after := time.Now().UTC()
	entries := l.GetRecentEntries(0)
	if len(entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	ts := entries[0].Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v not in expected range [%v, %v]", ts, before, after)
	}
}

func TestLog_TraceEnrichment(t *testing.T) {
	l := newTestLogger(true)
	// background context has no trace info, so fields stay empty
	l.Log(context.Background(), Entry{Tool: "test", Operation: "read"})

	entries := l.GetRecentEntries(0)
	if len(entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	if entries[0].TraceID != "" {
		t.Errorf("expected empty trace_id from background context, got %q", entries[0].TraceID)
	}
	if entries[0].SpanID != "" {
		t.Errorf("expected empty span_id from background context, got %q", entries[0].SpanID)
	}
}

func TestLogToolExecution(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()
	dur := 100 * time.Millisecond

	l.LogToolExecution(ctx, "search_logs", "query", "logs", "log-123", true, dur, nil)

	entries := l.GetRecentEntries(0)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Tool != "search_logs" {
		t.Errorf("tool: got %q, want %q", e.Tool, "search_logs")
	}
	if e.Operation != "query" {
		t.Errorf("operation: got %q, want %q", e.Operation, "query")
	}
	if e.Resource != "logs" {
		t.Errorf("resource: got %q, want %q", e.Resource, "logs")
	}
	if e.ResourceID != "log-123" {
		t.Errorf("resource_id: got %q, want %q", e.ResourceID, "log-123")
	}
	if !e.Success {
		t.Error("expected success=true")
	}
	if e.Duration != dur {
		t.Errorf("duration: got %v, want %v", e.Duration, dur)
	}
	if e.ErrorMsg != "" {
		t.Errorf("expected empty error_msg, got %q", e.ErrorMsg)
	}
}

func TestLogToolExecution_WithError(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()
	testErr := errors.New("connection refused")

	l.LogToolExecution(ctx, "search_logs", "query", "logs", "", false, 10*time.Millisecond, testErr)

	entries := l.GetRecentEntries(0)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ErrorMsg != "connection refused" {
		t.Errorf("error_msg: got %q, want %q", entries[0].ErrorMsg, "connection refused")
	}
	if entries[0].Success {
		t.Error("expected success=false")
	}
}

func TestGetRecentEntries_OrderNewestFirst(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		l.Log(ctx, Entry{
			Tool:      fmt.Sprintf("tool-%d", i),
			Operation: "read",
			Timestamp: time.Date(2025, 1, 1, 0, 0, i, 0, time.UTC),
		})
	}

	entries := l.GetRecentEntries(0)
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(entries))
	}
	// Newest first: tool-4, tool-3, ...
	for i, e := range entries {
		want := fmt.Sprintf("tool-%d", 4-i)
		if e.Tool != want {
			t.Errorf("entry[%d].Tool = %q, want %q", i, e.Tool, want)
		}
	}
}

func TestGetRecentEntries_LimitParam(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		l.Log(ctx, Entry{Tool: fmt.Sprintf("tool-%d", i), Operation: "read"})
	}

	entries := l.GetRecentEntries(3)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	// Should be the 3 most recent (newest first)
	if entries[0].Tool != "tool-9" {
		t.Errorf("expected newest entry tool-9, got %s", entries[0].Tool)
	}
}

func TestGetRecentEntries_LimitExceedsEntries(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		l.Log(ctx, Entry{Tool: fmt.Sprintf("tool-%d", i), Operation: "read"})
	}

	entries := l.GetRecentEntries(100)
	if len(entries) != 3 {
		t.Errorf("expected 3 entries (all available), got %d", len(entries))
	}
}

func TestGetRecentEntries_ZeroLimit(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		l.Log(ctx, Entry{Tool: "test", Operation: "read"})
	}

	entries := l.GetRecentEntries(0)
	if len(entries) != 5 {
		t.Errorf("expected 5 entries with zero limit, got %d", len(entries))
	}
}

func TestRingBuffer_EvictsOldest(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	// Fill beyond maxEntries
	for i := 0; i < l.maxEntries+1; i++ {
		l.Log(ctx, Entry{
			Tool:      fmt.Sprintf("tool-%d", i),
			Operation: "read",
		})
	}

	entries := l.GetRecentEntries(0)
	if len(entries) != l.maxEntries {
		t.Fatalf("expected %d entries, got %d", l.maxEntries, len(entries))
	}

	// The oldest entry (tool-0) should have been evicted
	// Newest first, so last entry should be tool-1 (the second oldest surviving)
	last := entries[len(entries)-1]
	if last.Tool != "tool-1" {
		t.Errorf("expected oldest surviving entry to be tool-1, got %s", last.Tool)
	}
	// Newest should be tool-1000
	if entries[0].Tool != fmt.Sprintf("tool-%d", l.maxEntries) {
		t.Errorf("expected newest entry to be tool-%d, got %s", l.maxEntries, entries[0].Tool)
	}
}

func TestGetEntriesByTool(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	l.Log(ctx, Entry{Tool: "search_logs", Operation: "query"})
	l.Log(ctx, Entry{Tool: "get_contexts", Operation: "read"})
	l.Log(ctx, Entry{Tool: "search_logs", Operation: "query"})
	l.Log(ctx, Entry{Tool: "list_sources", Operation: "list"})
	l.Log(ctx, Entry{Tool: "search_logs", Operation: "query"})

	entries := l.GetEntriesByTool("search_logs", 10)
	if len(entries) != 3 {
		t.Errorf("expected 3 entries for search_logs, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Tool != "search_logs" {
			t.Errorf("expected tool=search_logs, got %s", e.Tool)
		}
	}
}

func TestGetEntriesByTool_NoMatch(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	l.Log(ctx, Entry{Tool: "search_logs", Operation: "query"})

	entries := l.GetEntriesByTool("nonexistent", 10)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for unknown tool, got %d", len(entries))
	}
}

func TestGetEntriesByTraceID(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	// Manually set trace IDs since we use background context
	l.Log(ctx, Entry{Tool: "a", Operation: "read", TraceID: "trace-1"})
	l.Log(ctx, Entry{Tool: "b", Operation: "read", TraceID: "trace-2"})
	l.Log(ctx, Entry{Tool: "c", Operation: "read", TraceID: "trace-1"})
	l.Log(ctx, Entry{Tool: "d", Operation: "read", TraceID: "trace-3"})

	entries := l.GetEntriesByTraceID("trace-1")
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries for trace-1, got %d", len(entries))
	}
	for _, e := range entries {
		if e.TraceID != "trace-1" {
			t.Errorf("expected trace_id=trace-1, got %s", e.TraceID)
		}
	}
}

func TestGetStats(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	l.Log(ctx, Entry{Tool: "search_logs", Operation: "query", Success: true, Duration: 100 * time.Millisecond})
	l.Log(ctx, Entry{Tool: "search_logs", Operation: "query", Success: true, Duration: 200 * time.Millisecond})
	l.Log(ctx, Entry{Tool: "get_contexts", Operation: "read", Success: false, Duration: 50 * time.Millisecond, ErrorCode: "NOT_FOUND"})
	l.Log(ctx, Entry{Tool: "search_logs", Operation: "query", Success: false, Duration: 150 * time.Millisecond, ErrorCode: "TIMEOUT"})

	stats := l.GetStats()

	if stats.TotalEntries != 4 {
		t.Errorf("TotalEntries: got %d, want 4", stats.TotalEntries)
	}

	// 2 out of 4 succeeded = 50%
	if stats.SuccessRate != 50.0 {
		t.Errorf("SuccessRate: got %.2f, want 50.00", stats.SuccessRate)
	}

	// Average: (100+200+50+150)/4 = 125ms
	wantAvg := 125 * time.Millisecond
	if stats.AverageDuration != wantAvg {
		t.Errorf("AverageDuration: got %v, want %v", stats.AverageDuration, wantAvg)
	}

	if stats.ToolUsage["search_logs"] != 3 {
		t.Errorf("ToolUsage[search_logs]: got %d, want 3", stats.ToolUsage["search_logs"])
	}
	if stats.ToolUsage["get_contexts"] != 1 {
		t.Errorf("ToolUsage[get_contexts]: got %d, want 1", stats.ToolUsage["get_contexts"])
	}

	if stats.OperationCounts["query"] != 3 {
		t.Errorf("OperationCounts[query]: got %d, want 3", stats.OperationCounts["query"])
	}
}

func TestGetStats_Empty(t *testing.T) {
	l := newTestLogger(true)
	stats := l.GetStats()

	if stats.TotalEntries != 0 {
		t.Errorf("TotalEntries: got %d, want 0", stats.TotalEntries)
	}
	if stats.SuccessRate != 0 {
		t.Errorf("SuccessRate: got %.2f, want 0", stats.SuccessRate)
	}
	if stats.AverageDuration != 0 {
		t.Errorf("AverageDuration: got %v, want 0", stats.AverageDuration)
	}
}

func TestGetStats_ErrorCounts(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	l.Log(ctx, Entry{Tool: "t", Operation: "r", Success: false, ErrorCode: "TIMEOUT"})
	l.Log(ctx, Entry{Tool: "t", Operation: "r", Success: false, ErrorCode: "TIMEOUT"})
	l.Log(ctx, Entry{Tool: "t", Operation: "r", Success: false, ErrorCode: "NOT_FOUND"})
	l.Log(ctx, Entry{Tool: "t", Operation: "r", Success: true})

	stats := l.GetStats()

	if stats.ErrorCounts["TIMEOUT"] != 2 {
		t.Errorf("ErrorCounts[TIMEOUT]: got %d, want 2", stats.ErrorCounts["TIMEOUT"])
	}
	if stats.ErrorCounts["NOT_FOUND"] != 1 {
		t.Errorf("ErrorCounts[NOT_FOUND]: got %d, want 1", stats.ErrorCounts["NOT_FOUND"])
	}
	// Successful entries should not appear in error counts
	if len(stats.ErrorCounts) != 2 {
		t.Errorf("expected 2 error code types, got %d", len(stats.ErrorCounts))
	}
}

func TestToJSON(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	l.Log(ctx, Entry{Tool: "search_logs", Operation: "query", Success: true, Duration: 100 * time.Millisecond})
	l.Log(ctx, Entry{Tool: "get_contexts", Operation: "read", Success: false, ErrorCode: "ERR"})

	stats := l.GetStats()
	jsonStr := stats.ToJSON()

	if jsonStr == "" {
		t.Fatal("expected non-empty JSON output")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("ToJSON produced invalid JSON: %v", err)
	}

	if _, ok := parsed["total_entries"]; !ok {
		t.Error("expected total_entries field in JSON")
	}
	if _, ok := parsed["success_rate_pct"]; !ok {
		t.Error("expected success_rate_pct field in JSON")
	}
	if _, ok := parsed["tool_usage"]; !ok {
		t.Error("expected tool_usage field in JSON")
	}
}

func TestClear(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		l.Log(ctx, Entry{Tool: "test", Operation: "read"})
	}

	if len(l.GetRecentEntries(0)) != 10 {
		t.Fatal("expected 10 entries before clear")
	}

	l.Clear()

	entries := l.GetRecentEntries(0)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(entries))
	}
}

func TestConcurrentLogAndRead(t *testing.T) {
	l := newTestLogger(true)
	ctx := context.Background()

	var wg sync.WaitGroup
	const writers = 10
	const readers = 5
	const entriesPerWriter = 100

	// Writers
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < entriesPerWriter; i++ {
				l.Log(ctx, Entry{
					Tool:      fmt.Sprintf("tool-%d", id),
					Operation: "write",
					Success:   true,
					Duration:  time.Duration(i) * time.Millisecond,
				})
			}
		}(w)
	}

	// Readers
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < entriesPerWriter; i++ {
				_ = l.GetRecentEntries(10)
				_ = l.GetEntriesByTool("tool-0", 5)
				_ = l.GetStats()
			}
		}()
	}

	wg.Wait()

	total := l.GetRecentEntries(0)
	if len(total) != writers*entriesPerWriter {
		t.Errorf("expected %d entries, got %d", writers*entriesPerWriter, len(total))
	}
}
