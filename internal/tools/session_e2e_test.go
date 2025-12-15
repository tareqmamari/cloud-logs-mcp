package tools

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// TestE2E_MultiToolSessionWorkflow tests realistic multi-tool session usage
// This simulates the typical usage pattern that might cause compaction issues
func TestE2E_MultiToolSessionWorkflow(t *testing.T) {
	// Reset global state for clean test
	ResetBudgetContext()
	resetSessionForTest()

	logger, _ := zap.NewDevelopment()
	sessionTool := NewSessionContextTool(nil, logger)

	t.Run("complete_investigation_workflow", func(t *testing.T) {
		ctx := context.Background()
		session := GetSession()

		// Step 1: Set a filter
		result, err := sessionTool.Execute(ctx, map[string]interface{}{
			"action":       "set_filter",
			"filter_key":   "application",
			"filter_value": "api-gateway",
		})
		assertNoError(t, err, "set_filter")
		assertResultSuccess(t, result, "set_filter")

		// Step 2: Start an investigation
		result, err = sessionTool.Execute(ctx, map[string]interface{}{
			"action":      "start_investigation",
			"application": "api-gateway",
			"time_range":  "last 2 hours",
		})
		assertNoError(t, err, "start_investigation")
		assertResultSuccess(t, result, "start_investigation")

		// Verify investigation started
		inv := session.GetInvestigation()
		if inv == nil {
			t.Fatal("Expected investigation to be started")
		}
		if inv.Application != "api-gateway" {
			t.Errorf("Expected application 'api-gateway', got '%s'", inv.Application)
		}

		// Step 3: Record multiple tool uses (simulating query_logs, etc.)
		for i := 0; i < 5; i++ {
			session.RecordToolUse("query_logs", true, map[string]interface{}{
				"application": "api-gateway",
				"time_range":  "1h",
			})
		}

		// Step 4: Add findings
		result, err = sessionTool.Execute(ctx, map[string]interface{}{
			"action":   "add_finding",
			"finding":  "High error rate detected in /api/v1/users endpoint",
			"severity": "critical",
		})
		assertNoError(t, err, "add_finding")
		assertResultSuccess(t, result, "add_finding")

		// Step 5: Set hypothesis
		result, err = sessionTool.Execute(ctx, map[string]interface{}{
			"action":     "set_hypothesis",
			"hypothesis": "Database connection pool exhaustion causing timeouts",
		})
		assertNoError(t, err, "set_hypothesis")
		assertResultSuccess(t, result, "set_hypothesis")

		// Step 6: Check budget status (this is the new feature)
		result, err = sessionTool.Execute(ctx, map[string]interface{}{
			"action": "show_budget",
		})
		assertNoError(t, err, "show_budget")
		assertResultSuccess(t, result, "show_budget")

		// Verify budget response structure
		budgetResponse := parseResultJSON(t, result)
		if budgetResponse["budget_status"] == nil {
			t.Error("Expected budget_status in response")
		}
		if budgetResponse["recommendations"] == nil {
			t.Error("Expected recommendations in response")
		}

		// Step 7: Show full session state
		result, err = sessionTool.Execute(ctx, map[string]interface{}{
			"action": "show",
		})
		assertNoError(t, err, "show")
		assertResultSuccess(t, result, "show")

		sessionResponse := parseResultJSON(t, result)
		// showSession returns the session summary directly with fields like:
		// has_active_filters, active_filters, recent_tools_count, etc.
		if sessionResponse["has_active_filters"] == nil {
			t.Error("Expected has_active_filters in session response")
		}
		if sessionResponse["active_investigation"] == nil {
			t.Error("Expected active_investigation in response (investigation is active)")
		}

		// Step 8: End investigation
		result, err = sessionTool.Execute(ctx, map[string]interface{}{
			"action": "end_investigation",
		})
		assertNoError(t, err, "end_investigation")
		assertResultSuccess(t, result, "end_investigation")

		// Verify investigation ended
		if session.GetInvestigation() != nil {
			t.Error("Expected investigation to be ended")
		}
	})
}

// TestE2E_ConcurrentSessionAccess tests concurrent access to session state
// This can reveal race conditions that might cause issues with compaction
func TestE2E_ConcurrentSessionAccess(t *testing.T) {
	ResetBudgetContext()
	resetSessionForTest()

	logger, _ := zap.NewDevelopment()
	sessionTool := NewSessionContextTool(nil, logger)
	ctx := context.Background()

	const numGoroutines = 10
	const numOperations = 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Alternate between different operations
				var err error
				switch j % 4 {
				case 0:
					_, err = sessionTool.Execute(ctx, map[string]interface{}{
						"action": "show",
					})
				case 1:
					_, err = sessionTool.Execute(ctx, map[string]interface{}{
						"action": "show_budget",
					})
				case 2:
					_, err = sessionTool.Execute(ctx, map[string]interface{}{
						"action":       "set_filter",
						"filter_key":   "worker",
						"filter_value": string(rune('0' + workerID)),
					})
				case 3:
					// Record tool use to session
					session := GetSession()
					session.RecordToolUse("test_tool", true, map[string]interface{}{
						"worker": workerID,
						"op":     j,
					})
				}

				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	errCount := 0
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
		errCount++
	}

	if errCount > 0 {
		t.Fatalf("Had %d errors during concurrent access", errCount)
	}

	// Verify session is still in valid state
	session := GetSession()
	summary := session.GetSessionSummary()
	if summary == nil {
		t.Fatal("Session summary should not be nil after concurrent access")
	}

	t.Logf("After concurrent access: %d recent tools recorded", summary["recent_tools_count"])
}

// TestE2E_BudgetExhaustionScenario tests behavior when budget approaches limits
func TestE2E_BudgetExhaustionScenario(t *testing.T) {
	ResetBudgetContext()
	resetSessionForTest()

	logger, _ := zap.NewDevelopment()
	sessionTool := NewSessionContextTool(nil, logger)
	ctx := context.Background()

	budget := GetBudgetContext()

	testCases := []struct {
		name                string
		tokensToUse         int
		expectedCompression BudgetCompressionLevel
	}{
		{"10% usage", 10000, BudgetCompressionNone},
		{"30% usage", 20000, BudgetCompressionLight},   // Total: 30000
		{"55% usage", 25000, BudgetCompressionMedium},  // Total: 55000
		{"80% usage", 25000, BudgetCompressionHeavy},   // Total: 80000
		{"95% usage", 15000, BudgetCompressionMinimal}, // Total: 95000
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate token consumption
			budget.RecordToolExecution(tc.tokensToUse, 0)

			// Check budget status through tool
			result, err := sessionTool.Execute(ctx, map[string]interface{}{
				"action": "show_budget",
			})
			assertNoError(t, err, "show_budget at "+tc.name)
			assertResultSuccess(t, result, "show_budget at "+tc.name)

			// Verify compression level
			currentCompression := budget.GetCompressionLevel()
			if currentCompression != tc.expectedCompression {
				t.Errorf("Expected compression %s, got %s", tc.expectedCompression, currentCompression)
			}

			// Verify recommendations change based on compression level
			response := parseResultJSON(t, result)
			recommendations := response["recommendations"].([]interface{})
			if len(recommendations) == 0 {
				t.Error("Expected at least one recommendation")
			}

			t.Logf("At %s: compression=%s, recommendations=%v",
				tc.name, currentCompression, recommendations)
		})
	}
}

// TestE2E_SessionStateGrowth tests that session state doesn't grow unbounded
func TestE2E_SessionStateGrowth(t *testing.T) {
	ResetBudgetContext()
	resetSessionForTest()

	session := GetSession()

	// Record many tool uses (more than the limit of 20)
	for i := 0; i < 50; i++ {
		session.RecordToolUse("tool_"+string(rune('A'+i%26)), true, map[string]interface{}{
			"iteration": i,
		})
	}

	// Verify tool history is bounded
	recentTools := session.GetRecentTools(100) // Request more than limit
	if len(recentTools) > 20 {
		t.Errorf("Expected max 20 recent tools, got %d", len(recentTools))
	}

	// Cache many results (more than limit of 5)
	for i := 0; i < 10; i++ {
		session.CacheResult("tool_"+string(rune('A'+i)), map[string]interface{}{
			"data": "result_" + string(rune('0'+i)),
		})
	}

	// Count cached results through session summary
	summary := session.GetSessionSummary()
	t.Logf("Session state size check - recent_tools: %d", summary["recent_tools_count"])

	// Verify learned patterns don't grow unbounded
	if session.LearnedPatterns != nil {
		if len(session.LearnedPatterns.FrequentSequences) > 20 {
			t.Errorf("Expected max 20 frequent sequences, got %d",
				len(session.LearnedPatterns.FrequentSequences))
		}
	}
}

// TestE2E_SessionPersistenceRoundtrip tests session save/load
func TestE2E_SessionPersistenceRoundtrip(t *testing.T) {
	// Create a session manager with a temp directory
	tempDir := t.TempDir()
	manager := NewSessionManager(tempDir)

	// Create a session with state
	session := manager.GetOrCreateSession("test-api-key", "test-instance")
	session.SetFilter("application", "test-app")
	session.SetLastQuery("source logs | filter severity >= 4")
	session.RecordToolUse("query_logs", true, map[string]interface{}{
		"time_range": "1h",
	})
	session.StartInvestigation("test-app", "1h")
	session.AddFinding("query_logs", "Found errors", "warning", "sample evidence")

	// Save session
	userID := GenerateUserID("test-api-key", "test-instance")
	err := manager.SaveSession(userID)
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Create new manager and load
	manager2 := NewSessionManager(tempDir)
	loadedSession := manager2.GetOrCreateSession("test-api-key", "test-instance")

	// Verify state was preserved
	if loadedSession.GetFilter("application") != "test-app" {
		t.Error("Filter not preserved after load")
	}
	if loadedSession.GetLastQuery() != "source logs | filter severity >= 4" {
		t.Error("Last query not preserved after load")
	}
	if loadedSession.GetInvestigation() == nil {
		t.Error("Investigation not preserved after load")
	}
	if len(loadedSession.GetRecentTools(10)) != 1 {
		t.Error("Recent tools not preserved after load")
	}
}

// TestE2E_IntentAndBudgetIntegration tests intent verification with budget-aware responses
func TestE2E_IntentAndBudgetIntegration(t *testing.T) {
	ResetBudgetContext()

	testCases := []struct {
		name              string
		budgetUsedPercent int
		intent            string
		expectQuestion    bool
	}{
		{
			name:              "fresh_budget_clear_intent",
			budgetUsedPercent: 10,
			intent:            "search for errors in api-gateway last hour",
			expectQuestion:    false, // Clear intent, no need to ask
		},
		{
			name:              "low_budget_vague_intent",
			budgetUsedPercent: 80,
			intent:            "look at the logs",
			expectQuestion:    true, // Vague intent, should ask to conserve budget
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ResetBudgetContext()
			budget := GetBudgetContext()

			// Set budget to desired level
			tokensToUse := (tc.budgetUsedPercent * budget.MaxTokens) / 100
			budget.RecordToolExecution(tokensToUse, 0)

			// Verify intent
			verification := VerifyIntent(tc.intent)

			hasQuestions := len(verification.ClarifyingQuestions) > 0
			if hasQuestions != tc.expectQuestion {
				t.Errorf("Expected questions=%v, got=%v (questions: %v)",
					tc.expectQuestion, hasQuestions, verification.ClarifyingQuestions)
			}

			// When budget is low, progressive disclosure should kick in
			if tc.budgetUsedPercent >= 75 {
				compression := budget.GetCompressionLevel()
				if compression == BudgetCompressionNone {
					t.Error("Expected compression to be active at 75%+ budget usage")
				}
			}
		})
	}
}

// TestE2E_RapidToolExecution tests rapid successive tool executions
// This simulates patterns that might trigger compaction issues
func TestE2E_RapidToolExecution(t *testing.T) {
	ResetBudgetContext()
	resetSessionForTest()

	logger, _ := zap.NewDevelopment()
	sessionTool := NewSessionContextTool(nil, logger)
	ctx := context.Background()

	// Execute many rapid operations
	const iterations = 100
	start := time.Now()

	for i := 0; i < iterations; i++ {
		var err error
		switch i % 3 {
		case 0:
			_, err = sessionTool.Execute(ctx, map[string]interface{}{
				"action": "show",
			})
		case 1:
			_, err = sessionTool.Execute(ctx, map[string]interface{}{
				"action": "show_budget",
			})
		case 2:
			_, err = sessionTool.Execute(ctx, map[string]interface{}{
				"action":       "set_filter",
				"filter_key":   "iteration",
				"filter_value": string(rune('0' + i%10)),
			})
		}

		if err != nil {
			t.Fatalf("Error at iteration %d: %v", i, err)
		}
	}

	elapsed := time.Since(start)
	t.Logf("Executed %d operations in %v (%.2f ops/sec)",
		iterations, elapsed, float64(iterations)/elapsed.Seconds())

	// Verify final state is consistent
	result, err := sessionTool.Execute(ctx, map[string]interface{}{
		"action": "show",
	})
	assertNoError(t, err, "final show")
	assertResultSuccess(t, result, "final show")
}

// TestE2E_LargeResultHandling tests handling of large results with progressive disclosure
func TestE2E_LargeResultHandling(t *testing.T) {
	ResetBudgetContext()

	// Create large dataset
	events := make([]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		severity := float64(3) // INFO
		if i%10 == 0 {
			severity = float64(5) // ERROR every 10th
		}
		events[i] = map[string]interface{}{
			"message":         "Log message " + string(rune('0'+i%10)) + " with detailed content",
			"severity":        severity,
			"applicationname": "test-service",
			"timestamp":       "2024-01-15T10:00:00Z",
			"trace_id":        "abc123def456",
		}
	}
	data := map[string]interface{}{
		"events": events,
	}

	compressionLevels := []struct {
		level        BudgetCompressionLevel
		expectFull   bool
		expectSample bool
	}{
		{BudgetCompressionNone, true, true},
		{BudgetCompressionLight, false, true},
		{BudgetCompressionMedium, false, false},
		{BudgetCompressionHeavy, false, false},
	}

	for _, tc := range compressionLevels {
		t.Run(string(tc.level), func(t *testing.T) {
			budget := NewBudgetContext(100000, 10000)
			budget.ResultCompression = tc.level

			result := CreateProgressiveResult(data, budget)

			hasFullData := result.FullData != nil
			hasSamples := len(result.Samples) > 0

			if hasFullData != tc.expectFull {
				t.Errorf("FullData: expected %v, got %v", tc.expectFull, hasFullData)
			}
			if hasSamples != tc.expectSample {
				t.Errorf("Samples: expected %v, got %v", tc.expectSample, hasSamples)
			}

			// Summary should always be present
			if result.Summary == "" {
				t.Error("Summary should always be present")
			}

			t.Logf("Level %s: level=%d, summary=%s, samples=%d, hasFullData=%v",
				tc.level, result.Level, result.Summary, result.SampleCount, hasFullData)
		})
	}
}

// Helper functions

func resetSessionForTest() {
	// Reset to a fresh session state
	session := GetSession()
	session.ClearSession()
}

func assertNoError(t *testing.T, err error, context string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", context, err)
	}
}

func assertResultSuccess(t *testing.T, result *mcp.CallToolResult, context string) {
	t.Helper()
	if result == nil {
		t.Fatalf("%s: result is nil", context)
	}
	if result.IsError {
		t.Fatalf("%s: result is error", context)
	}
	if len(result.Content) == 0 {
		t.Fatalf("%s: result has no content", context)
	}
}

func parseResultJSON(t *testing.T, result *mcp.CallToolResult) map[string]interface{} {
	t.Helper()
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent in result")
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	return response
}
