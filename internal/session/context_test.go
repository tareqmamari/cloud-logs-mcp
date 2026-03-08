package session

import (
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	ctx := New()

	if ctx == nil {
		t.Fatal("New() returned nil")
	}
	if ctx.LastQuery != nil {
		t.Error("New context should have nil LastQuery")
	}
	if len(ctx.RecentQueries) != 0 {
		t.Errorf("New context should have 0 recent queries, got %d", len(ctx.RecentQueries))
	}
	if len(ctx.RecentErrors) != 0 {
		t.Errorf("New context should have 0 recent errors, got %d", len(ctx.RecentErrors))
	}
	if ctx.LastResources == nil {
		t.Error("LastResources map should be initialized")
	}
	if ctx.ToolCalls != 0 {
		t.Errorf("New context should have 0 tool calls, got %d", ctx.ToolCalls)
	}
	if ctx.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if ctx.maxRecentQueries != 10 {
		t.Errorf("maxRecentQueries should be 10, got %d", ctx.maxRecentQueries)
	}
}

func TestRecordQuery(t *testing.T) {
	ctx := New()

	query := QueryInfo{
		Query:       "source logs | filter $m.severity >= 5",
		Syntax:      "dataprime",
		Tier:        "archive",
		ResultCount: 42,
	}

	ctx.RecordQuery(query)

	if ctx.LastQuery == nil {
		t.Fatal("LastQuery should not be nil after recording")
	}
	if ctx.LastQuery.Query != query.Query {
		t.Errorf("LastQuery.Query = %q, want %q", ctx.LastQuery.Query, query.Query)
	}
	if ctx.ToolCalls != 1 {
		t.Errorf("ToolCalls should be 1, got %d", ctx.ToolCalls)
	}
	if len(ctx.RecentQueries) != 1 {
		t.Errorf("RecentQueries should have 1 entry, got %d", len(ctx.RecentQueries))
	}
	if ctx.LastQuery.Timestamp.IsZero() {
		t.Error("Query timestamp should be set automatically")
	}
}

func TestRecordQuery_MaxBound(t *testing.T) {
	ctx := New()

	// Record more than maxRecentQueries
	for i := 0; i < 15; i++ {
		ctx.RecordQuery(QueryInfo{
			Query: "query_" + string(rune('A'+i)),
		})
	}

	if len(ctx.RecentQueries) != 10 {
		t.Errorf("RecentQueries should be bounded to 10, got %d", len(ctx.RecentQueries))
	}

	// Oldest queries should be evicted (FIFO)
	if ctx.RecentQueries[0].Query != "query_F" {
		t.Errorf("First query should be 'query_F' (6th recorded), got %q", ctx.RecentQueries[0].Query)
	}

	// Last query should be the most recent
	if ctx.LastQuery.Query != "query_O" {
		t.Errorf("LastQuery should be the most recent, got %q", ctx.LastQuery.Query)
	}

	if ctx.ToolCalls != 15 {
		t.Errorf("ToolCalls should be 15, got %d", ctx.ToolCalls)
	}
}

func TestRecordResource(t *testing.T) {
	ctx := New()

	ctx.RecordResource("dashboard", "dash-123", "My Dashboard")

	resource := ctx.GetLastResource("dashboard")
	if resource == nil {
		t.Fatal("GetLastResource returned nil")
	}
	if resource.ID != "dash-123" {
		t.Errorf("Resource ID = %q, want %q", resource.ID, "dash-123")
	}
	if resource.Name != "My Dashboard" {
		t.Errorf("Resource Name = %q, want %q", resource.Name, "My Dashboard")
	}
	if resource.Timestamp.IsZero() {
		t.Error("Resource timestamp should be set")
	}
	if ctx.ToolCalls != 1 {
		t.Errorf("ToolCalls should be 1, got %d", ctx.ToolCalls)
	}
}

func TestRecordResource_OverwritesSameType(t *testing.T) {
	ctx := New()

	ctx.RecordResource("alert", "alert-1", "Alert One")
	ctx.RecordResource("alert", "alert-2", "Alert Two")

	resource := ctx.GetLastResource("alert")
	if resource.ID != "alert-2" {
		t.Errorf("Should overwrite same type, got ID %q", resource.ID)
	}
}

func TestRecordResource_MultipleTypes(t *testing.T) {
	ctx := New()

	ctx.RecordResource("dashboard", "dash-1", "Dashboard")
	ctx.RecordResource("alert", "alert-1", "Alert")

	dash := ctx.GetLastResource("dashboard")
	alert := ctx.GetLastResource("alert")

	if dash == nil || dash.ID != "dash-1" {
		t.Error("Dashboard resource not preserved")
	}
	if alert == nil || alert.ID != "alert-1" {
		t.Error("Alert resource not preserved")
	}
}

func TestGetLastResource_NotFound(t *testing.T) {
	ctx := New()

	result := ctx.GetLastResource("nonexistent")
	if result != nil {
		t.Error("Expected nil for non-existent resource type")
	}
}

func TestRecordError(t *testing.T) {
	ctx := New()

	ctx.RecordError("query_logs", "connection timeout", 504)

	errors := ctx.GetRecentErrors()
	if len(errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors))
	}
	if errors[0].Tool != "query_logs" {
		t.Errorf("Error tool = %q, want %q", errors[0].Tool, "query_logs")
	}
	if errors[0].Code != 504 {
		t.Errorf("Error code = %d, want %d", errors[0].Code, 504)
	}
	if errors[0].Timestamp.IsZero() {
		t.Error("Error timestamp should be set")
	}
}

func TestRecordError_MaxBound(t *testing.T) {
	ctx := New()

	for i := 0; i < 15; i++ {
		ctx.RecordError("tool", "error", i)
	}

	errors := ctx.GetRecentErrors()
	if len(errors) != 10 {
		t.Errorf("RecentErrors should be bounded to 10, got %d", len(errors))
	}

	// Oldest errors evicted
	if errors[0].Code != 5 {
		t.Errorf("First error code should be 5, got %d", errors[0].Code)
	}
}

func TestHasRecentErrors(t *testing.T) {
	ctx := New()

	if ctx.HasRecentErrors() {
		t.Error("New context should not have recent errors")
	}

	ctx.RecordError("tool", "error", 500)

	if !ctx.HasRecentErrors() {
		t.Error("Should have recent errors after recording")
	}
}

func TestGetLastQuery_ReturnsThreadSafeCopy(t *testing.T) {
	ctx := New()

	ctx.RecordQuery(QueryInfo{Query: "original"})

	copy1 := ctx.GetLastQuery()
	copy1.Query = "modified"

	copy2 := ctx.GetLastQuery()
	if copy2.Query != "original" {
		t.Error("GetLastQuery should return a copy, modification leaked")
	}
}

func TestGetLastResource_ReturnsThreadSafeCopy(t *testing.T) {
	ctx := New()

	ctx.RecordResource("alert", "alert-1", "Original")

	copy1 := ctx.GetLastResource("alert")
	copy1.Name = "Modified"

	copy2 := ctx.GetLastResource("alert")
	if copy2.Name != "Original" {
		t.Error("GetLastResource should return a copy, modification leaked")
	}
}

func TestGetRecentQueries_ReturnsThreadSafeCopy(t *testing.T) {
	ctx := New()

	ctx.RecordQuery(QueryInfo{Query: "query1"})
	ctx.RecordQuery(QueryInfo{Query: "query2"})

	queries := ctx.GetRecentQueries()
	queries[0].Query = "modified"

	original := ctx.GetRecentQueries()
	if original[0].Query != "query1" {
		t.Error("GetRecentQueries should return a copy, modification leaked")
	}
}

func TestGetStats(t *testing.T) {
	ctx := New()

	ctx.RecordQuery(QueryInfo{Query: "q1"})
	ctx.RecordQuery(QueryInfo{Query: "q2"})
	ctx.RecordResource("alert", "a1", "Alert")
	ctx.RecordError("tool", "err", 500)

	stats := ctx.GetStats()

	if stats["tool_calls"].(int) != 3 { // 2 queries + 1 resource
		t.Errorf("tool_calls = %v, want 3", stats["tool_calls"])
	}
	if stats["queries_count"].(int) != 2 {
		t.Errorf("queries_count = %v, want 2", stats["queries_count"])
	}
	if stats["resources_count"].(int) != 1 {
		t.Errorf("resources_count = %v, want 1", stats["resources_count"])
	}
	if stats["errors_count"].(int) != 1 {
		t.Errorf("errors_count = %v, want 1", stats["errors_count"])
	}
	if stats["age_seconds"].(float64) < 0 {
		t.Error("age_seconds should be non-negative")
	}
}

func TestClear(t *testing.T) {
	ctx := New()

	// Populate state
	ctx.RecordQuery(QueryInfo{Query: "q1"})
	ctx.RecordResource("alert", "a1", "Alert")
	ctx.RecordError("tool", "err", 500)

	ctx.Clear()

	if ctx.LastQuery != nil {
		t.Error("LastQuery should be nil after Clear")
	}
	if len(ctx.RecentQueries) != 0 {
		t.Error("RecentQueries should be empty after Clear")
	}
	if len(ctx.LastResources) != 0 {
		t.Error("LastResources should be empty after Clear")
	}
	if len(ctx.RecentErrors) != 0 {
		t.Error("RecentErrors should be empty after Clear")
	}
	if ctx.ToolCalls != 0 {
		t.Error("ToolCalls should be 0 after Clear")
	}
}

func TestSuggestNextTools(t *testing.T) {
	t.Run("suggests alerts after query with errors", func(t *testing.T) {
		ctx := New()
		ctx.RecordQuery(QueryInfo{Query: "q1", HasErrors: true})

		suggestions := ctx.SuggestNextTools()
		found := map[string]bool{}
		for _, s := range suggestions {
			found[s] = true
		}
		if !found["create_alert"] {
			t.Error("Expected create_alert suggestion after error query")
		}
		if !found["create_dashboard"] {
			t.Error("Expected create_dashboard suggestion after error query")
		}
	})

	t.Run("suggests dashboard tools after dashboard access", func(t *testing.T) {
		ctx := New()
		ctx.RecordResource("dashboard", "d1", "Dash")

		suggestions := ctx.SuggestNextTools()
		found := map[string]bool{}
		for _, s := range suggestions {
			found[s] = true
		}
		if !found["pin_dashboard"] {
			t.Error("Expected pin_dashboard suggestion")
		}
	})

	t.Run("suggests webhook tools after alert access", func(t *testing.T) {
		ctx := New()
		ctx.RecordResource("alert", "a1", "Alert")

		suggestions := ctx.SuggestNextTools()
		found := map[string]bool{}
		for _, s := range suggestions {
			found[s] = true
		}
		if !found["list_outgoing_webhooks"] {
			t.Error("Expected list_outgoing_webhooks suggestion")
		}
	})

	t.Run("suggests debugging after errors", func(t *testing.T) {
		ctx := New()
		ctx.RecordError("query_logs", "timeout", 504)

		suggestions := ctx.SuggestNextTools()
		found := map[string]bool{}
		for _, s := range suggestions {
			found[s] = true
		}
		if !found["get_query_templates"] {
			t.Error("Expected get_query_templates suggestion after errors")
		}
	})

	t.Run("empty suggestions for fresh context", func(t *testing.T) {
		ctx := New()
		suggestions := ctx.SuggestNextTools()
		if len(suggestions) != 0 {
			t.Errorf("Expected no suggestions for fresh context, got %v", suggestions)
		}
	})
}

// Test multi-tenant session isolation
func TestConcurrentSessionIsolation(t *testing.T) {
	// Simulate two users with separate contexts
	user1Ctx := New()
	user2Ctx := New()

	var wg sync.WaitGroup

	// User 1 writes
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			user1Ctx.RecordQuery(QueryInfo{
				Query: "user1_query",
				Tier:  "archive",
			})
			user1Ctx.RecordResource("alert", "user1-alert", "User1 Alert")
		}
	}()

	// User 2 writes
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			user2Ctx.RecordQuery(QueryInfo{
				Query: "user2_query",
				Tier:  "frequent_search",
			})
			user2Ctx.RecordResource("dashboard", "user2-dash", "User2 Dashboard")
		}
	}()

	wg.Wait()

	// Verify isolation: user1's data never appears in user2's context
	u1Query := user1Ctx.GetLastQuery()
	u2Query := user2Ctx.GetLastQuery()

	if u1Query.Query != "user1_query" {
		t.Errorf("User1 context contaminated: %q", u1Query.Query)
	}
	if u2Query.Query != "user2_query" {
		t.Errorf("User2 context contaminated: %q", u2Query.Query)
	}

	// User1 should not have dashboard resource
	if user1Ctx.GetLastResource("dashboard") != nil {
		t.Error("User1 context has dashboard resource from user2")
	}
	// User2 should not have alert resource
	if user2Ctx.GetLastResource("alert") != nil {
		t.Error("User2 context has alert resource from user1")
	}
}

// Test concurrent reads and writes on the same context (race detection)
func TestConcurrentReadWrite(_ *testing.T) {
	ctx := New()

	var wg sync.WaitGroup
	const goroutines = 20

	// Concurrent writers
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			ctx.RecordQuery(QueryInfo{Query: "query"})
			ctx.RecordResource("alert", "a1", "Alert")
			ctx.RecordError("tool", "error", 500)
		}(i)
	}

	// Concurrent readers
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ctx.GetLastQuery()
			_ = ctx.GetLastResource("alert")
			_ = ctx.GetRecentQueries()
			_ = ctx.GetRecentErrors()
			_ = ctx.HasRecentErrors()
			_ = ctx.GetStats()
			_ = ctx.SuggestNextTools()
		}()
	}

	wg.Wait()
	// Test passes if no race detected (run with -race flag)
}

// Test Clear during concurrent access
func TestConcurrentClear(_ *testing.T) {
	ctx := New()

	var wg sync.WaitGroup

	// Writer goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				ctx.RecordQuery(QueryInfo{Query: "q"})
			}
		}()
	}

	// Clear goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				ctx.Clear()
			}
		}()
	}

	wg.Wait()
	// No panic or race = success
}
