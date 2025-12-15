package tools

import (
	"fmt"
	"testing"
)

func TestBudgetContext_NewBudgetContext(t *testing.T) {
	budget := NewBudgetContext(100000, 10000)

	if budget.MaxTokens != 100000 {
		t.Errorf("Expected MaxTokens=100000, got %d", budget.MaxTokens)
	}
	if budget.UsedTokens != 0 {
		t.Errorf("Expected UsedTokens=0, got %d", budget.UsedTokens)
	}
	if budget.RemainingTokens != 100000 {
		t.Errorf("Expected RemainingTokens=100000, got %d", budget.RemainingTokens)
	}
	if budget.ResultCompression != BudgetCompressionNone {
		t.Errorf("Expected ResultCompression=none, got %s", budget.ResultCompression)
	}
}

func TestBudgetContext_RecordToolExecution(t *testing.T) {
	budget := NewBudgetContext(10000, 10000)

	budget.RecordToolExecution(100, 200)

	if budget.UsedTokens != 300 {
		t.Errorf("Expected UsedTokens=300, got %d", budget.UsedTokens)
	}
	if budget.RemainingTokens != 9700 {
		t.Errorf("Expected RemainingTokens=9700, got %d", budget.RemainingTokens)
	}
	if budget.ToolCallCount != 1 {
		t.Errorf("Expected ToolCallCount=1, got %d", budget.ToolCallCount)
	}
}

func TestBudgetContext_CompressionLevelAdjustment(t *testing.T) {
	tests := []struct {
		name          string
		maxTokens     int
		usedTokens    int
		expectedLevel BudgetCompressionLevel
	}{
		{"No compression at 10%", 10000, 1000, BudgetCompressionNone},
		{"Light compression at 30%", 10000, 3000, BudgetCompressionLight},
		{"Medium compression at 55%", 10000, 5500, BudgetCompressionMedium},
		{"Heavy compression at 80%", 10000, 8000, BudgetCompressionHeavy},
		{"Minimal compression at 95%", 10000, 9500, BudgetCompressionMinimal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := NewBudgetContext(tt.maxTokens, 10000)
			// Simulate usage by calling RecordToolExecution
			budget.RecordToolExecution(tt.usedTokens, 0)

			if budget.GetCompressionLevel() != tt.expectedLevel {
				t.Errorf("Expected compression=%s, got %s",
					tt.expectedLevel, budget.GetCompressionLevel())
			}
		})
	}
}

func TestBudgetContext_ShouldExecute(t *testing.T) {
	budget := NewBudgetContext(1000, 10000)
	budget.RecordToolExecution(800, 0) // Use 800 tokens

	// Should allow 100 tokens
	canExecute, reason := budget.ShouldExecute(100)
	if !canExecute {
		t.Errorf("Should allow 100 tokens, but got rejection: %s", reason)
	}

	// Should reject 300 tokens
	canExecute, reason = budget.ShouldExecute(300)
	if canExecute {
		t.Error("Should reject 300 tokens (only 200 remaining)")
	}
	if reason == "" {
		t.Error("Expected rejection reason")
	}
}

func TestBudgetContext_GetSummary(t *testing.T) {
	budget := NewBudgetContext(10000, 10000)
	budget.RecordToolExecution(500, 500)

	summary := budget.GetSummary()

	if summary["tokens"] == nil {
		t.Error("Expected tokens in summary")
	}
	if summary["cost"] == nil {
		t.Error("Expected cost in summary")
	}
	if summary["execution"] == nil {
		t.Error("Expected execution in summary")
	}

	tokens := summary["tokens"].(map[string]interface{})
	if tokens["used"].(int) != 1000 {
		t.Errorf("Expected used tokens=1000, got %v", tokens["used"])
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"hello", 2},              // 5 chars ≈ 2 tokens
		{"hello world", 3},        // 11 chars ≈ 3 tokens
		{"a longer text here", 5}, // 18 chars ≈ 5 tokens
	}

	for _, tt := range tests {
		result := EstimateTokens(tt.text)
		// Allow some variance since it's an approximation
		if result < tt.expected-1 || result > tt.expected+1 {
			t.Errorf("EstimateTokens(%q) = %d, expected ~%d", tt.text, result, tt.expected)
		}
	}
}

func TestTokenCounter_Approximate(t *testing.T) {
	counter := &ApproximateTokenCounter{}

	if counter.Name() != "approximate (chars/4)" {
		t.Errorf("Expected name 'approximate (chars/4)', got %s", counter.Name())
	}
	if counter.IsExact() {
		t.Error("ApproximateTokenCounter should not be exact")
	}
	if counter.CountTokens("hello world") != 3 {
		t.Errorf("Expected 3 tokens for 'hello world', got %d", counter.CountTokens("hello world"))
	}
}

func TestTokenCounter_ClientReported(t *testing.T) {
	counter := &ClientReportedTokenCounter{}

	if counter.Name() != "client-reported" {
		t.Errorf("Expected name 'client-reported', got %s", counter.Name())
	}
	if !counter.IsExact() {
		t.Error("ClientReportedTokenCounter should be exact")
	}

	// Record tokens
	counter.RecordClientTokens(100, 200)
	input, output := counter.GetLastTokens()
	if input != 100 || output != 200 {
		t.Errorf("Expected (100, 200), got (%d, %d)", input, output)
	}
}

func TestBudgetContext_ClientReportedTokens(t *testing.T) {
	budget := NewBudgetContext(10000, 10000)

	// Record client-reported tokens
	budget.RecordClientReportedTokens(500, 1000)

	if budget.UsedTokens != 1500 {
		t.Errorf("Expected UsedTokens=1500, got %d", budget.UsedTokens)
	}
	if !budget.IsExactCount {
		t.Error("Expected IsExactCount=true after client-reported tokens")
	}
	if budget.TokenCountingMethod != "client-reported" {
		t.Errorf("Expected TokenCountingMethod='client-reported', got %s", budget.TokenCountingMethod)
	}
}

func TestBudgetContext_SummaryIncludesAccuracy(t *testing.T) {
	budget := NewBudgetContext(10000, 10000)
	summary := budget.GetSummary()

	tokens := summary["tokens"].(map[string]interface{})
	if tokens["accuracy"] != "approximate" {
		t.Errorf("Expected accuracy='approximate', got %v", tokens["accuracy"])
	}
	if tokens["counting_method"] == nil {
		t.Error("Expected counting_method in summary")
	}

	// Now record client-reported tokens
	budget.RecordClientReportedTokens(100, 100)
	summary = budget.GetSummary()
	tokens = summary["tokens"].(map[string]interface{})
	if tokens["accuracy"] != "exact" {
		t.Errorf("Expected accuracy='exact' after client-reported, got %v", tokens["accuracy"])
	}
}

func TestCreateTokenMetrics(t *testing.T) {
	inputArgs := map[string]interface{}{"query": "test"}
	result := map[string]interface{}{"logs": []interface{}{}}

	metrics := CreateTokenMetrics("query_logs", inputArgs, result, false, 0)

	if metrics.ToolName != "query_logs" {
		t.Errorf("Expected ToolName=query_logs, got %s", metrics.ToolName)
	}
	if metrics.TotalTokens <= 0 {
		t.Error("Expected positive TotalTokens")
	}
	if metrics.Compressed {
		t.Error("Expected Compressed=false")
	}
}

func TestCreateTokenMetrics_WithCompression(t *testing.T) {
	inputArgs := map[string]interface{}{"query": "test"}
	result := map[string]interface{}{"summary": "5 logs found"}

	metrics := CreateTokenMetrics("query_logs", inputArgs, result, true, 1000)

	if !metrics.Compressed {
		t.Error("Expected Compressed=true")
	}
	if metrics.CompressionRatio == "" {
		t.Error("Expected CompressionRatio to be set")
	}
}

func TestProgressiveResult_CreateProgressiveResult(t *testing.T) {
	// Create data with enough events for sampling
	events := make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		events[i] = map[string]interface{}{
			"message":  fmt.Sprintf("test message %d", i),
			"severity": float64(5),
		}
	}
	data := map[string]interface{}{
		"events": events,
	}

	tests := []struct {
		name             string
		compressionLevel BudgetCompressionLevel
		expectedMinLevel int
		hasFullData      bool
	}{
		{"No compression", BudgetCompressionNone, 4, true},
		{"Light compression", BudgetCompressionLight, 3, false},
		{"Medium compression", BudgetCompressionMedium, 2, false},
		{"Heavy compression", BudgetCompressionHeavy, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := NewBudgetContext(100000, 10000)
			budget.ResultCompression = tt.compressionLevel

			result := CreateProgressiveResult(data, budget)

			if result.Level < tt.expectedMinLevel {
				t.Errorf("Expected Level>=%d, got %d", tt.expectedMinLevel, result.Level)
			}
			if (result.FullData != nil) != tt.hasFullData {
				t.Errorf("HasFullData: expected %v, got %v", tt.hasFullData, result.FullData != nil)
			}
		})
	}
}

func TestProgressiveResult_Summary(t *testing.T) {
	data := map[string]interface{}{
		"events": []interface{}{
			map[string]interface{}{"message": "error1"},
			map[string]interface{}{"message": "error2"},
		},
	}

	budget := NewBudgetContext(100000, 10000)
	result := CreateProgressiveResult(data, budget)

	if result.Summary == "" {
		t.Error("Expected non-empty summary")
	}
	if result.TotalCount != 2 {
		t.Errorf("Expected TotalCount=2, got %d", result.TotalCount)
	}
}

func TestBudgetGenerateSummary(t *testing.T) {
	tests := []struct {
		name          string
		data          interface{}
		expectedCount int
	}{
		{
			name: "Events array",
			data: map[string]interface{}{
				"events": []interface{}{1, 2, 3},
			},
			expectedCount: 3,
		},
		{
			name: "Logs array",
			data: map[string]interface{}{
				"logs": []interface{}{1, 2},
			},
			expectedCount: 2,
		},
		{
			name:          "Direct array",
			data:          []interface{}{1, 2, 3, 4},
			expectedCount: 4,
		},
		{
			name:          "Single object",
			data:          map[string]interface{}{"id": "123"},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, count := budgetGenerateSummary(tt.data)
			if count != tt.expectedCount {
				t.Errorf("Expected count=%d, got %d", tt.expectedCount, count)
			}
		})
	}
}

func TestBudgetDetectAnomalies(t *testing.T) {
	// High error rate scenario
	events := make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		severity := float64(3) // INFO
		if i < 6 {
			severity = float64(5) // ERROR
		}
		events[i] = map[string]interface{}{"severity": severity}
	}

	anomalies := budgetDetectAnomalies(events)

	// Should detect high error rate (60% errors)
	hasHighErrorRate := false
	for _, a := range anomalies {
		if budgetTestContains(a, "error rate") {
			hasHighErrorRate = true
			break
		}
	}
	if !hasHighErrorRate {
		t.Error("Expected to detect high error rate anomaly")
	}
}

func TestBudgetDetectPatterns(t *testing.T) {
	// Create events with repeated messages
	events := make([]interface{}, 20)
	for i := 0; i < 20; i++ {
		msg := "Connection timeout to database"
		if i < 5 {
			msg = "Request received"
		}
		events[i] = map[string]interface{}{"message": msg}
	}

	patterns := budgetDetectPatterns(events)

	// Should detect the repeated timeout message
	if len(patterns) == 0 {
		t.Error("Expected to detect repeated message pattern")
	}
}

// Helper function for test
func budgetTestContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && budgetTestContainsSubstr(s, substr))
}

func budgetTestContainsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
