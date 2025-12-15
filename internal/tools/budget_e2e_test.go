package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// TestE2E_BudgetWorkflow tests the complete budget tracking workflow
func TestE2E_BudgetWorkflow(t *testing.T) {
	// Reset global state for clean test
	ResetBudgetContext()

	// Create a session context tool (no client needed for budget operations)
	logger, _ := zap.NewDevelopment()
	tool := NewSessionContextTool(nil, logger)

	// Step 1: Check initial budget status
	t.Run("initial_budget_status", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"action": "show_budget",
		})
		if err != nil {
			t.Fatalf("Failed to execute show_budget: %v", err)
		}

		// Parse the result
		var response map[string]interface{}
		if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
			if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}
		} else {
			t.Fatal("Expected TextContent in result")
		}

		// Verify initial state
		budgetStatus := response["budget_status"].(map[string]interface{})
		tokens := budgetStatus["tokens"].(map[string]interface{})

		if tokens["used"].(float64) != 0 {
			t.Errorf("Expected 0 used tokens initially, got %v", tokens["used"])
		}
		if tokens["accuracy"] != "approximate" {
			t.Errorf("Expected 'approximate' accuracy, got %v", tokens["accuracy"])
		}
	})

	// Step 2: Record some tool executions
	t.Run("record_tool_executions", func(t *testing.T) {
		budget := GetBudgetContext()

		// Simulate tool executions with estimated tokens
		budget.RecordToolExecution(1000, 2000) // 3000 tokens
		budget.RecordToolExecution(500, 1500)  // 2000 tokens

		if budget.UsedTokens != 5000 {
			t.Errorf("Expected 5000 used tokens, got %d", budget.UsedTokens)
		}
		if budget.ToolCallCount != 2 {
			t.Errorf("Expected 2 tool calls, got %d", budget.ToolCallCount)
		}
	})

	// Step 3: Record client-reported tokens (exact)
	t.Run("record_client_reported_tokens", func(t *testing.T) {
		budget := GetBudgetContext()

		// Simulate client-reported exact tokens
		budget.RecordClientReportedTokens(2000, 3000)

		if budget.UsedTokens != 10000 {
			t.Errorf("Expected 10000 used tokens, got %d", budget.UsedTokens)
		}
		if !budget.IsExactCount {
			t.Error("Expected IsExactCount=true after client-reported tokens")
		}
		if budget.TokenCountingMethod != "client-reported" {
			t.Errorf("Expected 'client-reported' method, got %s", budget.TokenCountingMethod)
		}
	})

	// Step 4: Verify compression level changes with budget consumption
	t.Run("compression_level_adjustment", func(t *testing.T) {
		budget := GetBudgetContext()

		// We've used 10000 tokens out of 100000 (10%) - should be "none"
		if budget.GetCompressionLevel() != BudgetCompressionNone {
			t.Errorf("Expected 'none' compression at 10%%, got %s", budget.GetCompressionLevel())
		}

		// Use more tokens to trigger light compression (25%)
		budget.RecordToolExecution(15000, 0) // Now at 25000 (25%)
		if budget.GetCompressionLevel() != BudgetCompressionLight {
			t.Errorf("Expected 'light' compression at 25%%, got %s", budget.GetCompressionLevel())
		}

		// Use more tokens to trigger medium compression (50%)
		budget.RecordToolExecution(25000, 0) // Now at 50000 (50%)
		if budget.GetCompressionLevel() != BudgetCompressionMedium {
			t.Errorf("Expected 'medium' compression at 50%%, got %s", budget.GetCompressionLevel())
		}
	})

	// Step 5: Check budget status after usage
	t.Run("final_budget_status", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"action": "show_budget",
		})
		if err != nil {
			t.Fatalf("Failed to execute show_budget: %v", err)
		}

		// Verify we got a result with content
		if len(result.Content) == 0 {
			t.Fatal("Expected content in result")
		}

		// The result should show medium compression and recommendations
		t.Log("Budget status retrieved successfully after workflow")
	})
}

// TestE2E_IntentVerificationWorkflow tests the complete intent verification workflow
func TestE2E_IntentVerificationWorkflow(t *testing.T) {
	testCases := []struct {
		name            string
		intent          string
		expectedType    IntentType
		alternativeType IntentType // Some intents can match multiple types
		expectEntities  bool
		expectQuestions bool
	}{
		{
			name:            "clear_investigate_intent",
			intent:          "investigate the error spike in api-gateway service last hour",
			expectedType:    IntentInvestigate,
			alternativeType: IntentInvestigate,
			expectEntities:  true,
			expectQuestions: false, // Clear intent, high confidence
		},
		{
			name:            "ambiguous_intent",
			intent:          "search logs and create alert",
			expectedType:    IntentQuery,   // Primary intent could be query
			alternativeType: IntentMonitor, // Or monitor due to "alert"
			expectEntities:  false,
			expectQuestions: true, // Ambiguous, should ask questions
		},
		{
			name:            "vague_intent",
			intent:          "do something",
			expectedType:    IntentUnknown,
			alternativeType: IntentUnknown,
			expectEntities:  false,
			expectQuestions: true, // Vague, should ask questions
		},
		{
			name:            "dashboard_visualization",
			intent:          "visualize dashboard with charts",
			expectedType:    IntentVisualize,
			alternativeType: IntentVisualize,
			expectEntities:  false,
			expectQuestions: true, // Missing time range
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := VerifyIntent(tc.intent)

			// Verify intent type (allow alternative type for ambiguous cases)
			if result.IntentType != tc.expectedType && result.IntentType != tc.alternativeType {
				t.Errorf("Expected intent type %s or %s, got %s", tc.expectedType, tc.alternativeType, result.IntentType)
			}

			// Verify entities extraction
			if tc.expectEntities && result.ExtractedEntities == nil {
				t.Error("Expected entities to be extracted")
			}

			// Verify clarifying questions
			hasQuestions := len(result.ClarifyingQuestions) > 0
			if tc.expectQuestions && !hasQuestions {
				t.Error("Expected clarifying questions")
			}

			// Log the verification result for debugging
			t.Logf("Intent: %q", tc.intent)
			t.Logf("  Type: %s, Confidence: %.2f", result.IntentType, result.Confidence)
			if result.ExtractedEntities != nil {
				t.Logf("  Services: %v, TimeRange: %s, Severity: %s",
					result.ExtractedEntities.Services,
					result.ExtractedEntities.TimeRange,
					result.ExtractedEntities.Severity)
			}
			if len(result.ClarifyingQuestions) > 0 {
				t.Logf("  Questions: %v", result.ClarifyingQuestions)
			}
		})
	}
}

// TestE2E_ProgressiveDisclosure tests the progressive result disclosure
func TestE2E_ProgressiveDisclosure(t *testing.T) {
	// Create sample log data
	events := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		severity := float64(3) // INFO
		if i%10 == 0 {
			severity = float64(5) // ERROR every 10th entry
		}
		events[i] = map[string]interface{}{
			"message":         "Test log message " + string(rune('0'+i%10)),
			"severity":        severity,
			"applicationname": "test-service",
			"timestamp":       "2024-01-15T10:00:00Z",
		}
	}
	data := map[string]interface{}{
		"events": events,
	}

	testCases := []struct {
		name             string
		compressionLevel BudgetCompressionLevel
		expectFullData   bool
		expectSamples    bool
		expectInsights   bool
		minLevel         int
	}{
		{
			name:             "no_compression",
			compressionLevel: BudgetCompressionNone,
			expectFullData:   true,
			expectSamples:    true,
			expectInsights:   true,
			minLevel:         4,
		},
		{
			name:             "light_compression",
			compressionLevel: BudgetCompressionLight,
			expectFullData:   false,
			expectSamples:    true,
			expectInsights:   true,
			minLevel:         3,
		},
		{
			name:             "medium_compression",
			compressionLevel: BudgetCompressionMedium,
			expectFullData:   false,
			expectSamples:    false,
			expectInsights:   true,
			minLevel:         2,
		},
		{
			name:             "heavy_compression",
			compressionLevel: BudgetCompressionHeavy,
			expectFullData:   false,
			expectSamples:    false,
			expectInsights:   false,
			minLevel:         1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			budget := NewBudgetContext(100000, 10000)
			budget.ResultCompression = tc.compressionLevel

			result := CreateProgressiveResult(data, budget)

			// Verify disclosure level
			if result.Level < tc.minLevel {
				t.Errorf("Expected level >= %d, got %d", tc.minLevel, result.Level)
			}

			// Verify full data presence
			hasFullData := result.FullData != nil
			if hasFullData != tc.expectFullData {
				t.Errorf("Expected fullData=%v, got %v", tc.expectFullData, hasFullData)
			}

			// Verify samples presence
			hasSamples := len(result.Samples) > 0
			if hasSamples != tc.expectSamples {
				t.Errorf("Expected samples=%v, got %v", tc.expectSamples, hasSamples)
			}

			// Verify insights presence
			hasInsights := result.Insights != nil
			if hasInsights != tc.expectInsights {
				t.Errorf("Expected insights=%v, got %v", tc.expectInsights, hasInsights)
			}

			// Verify summary is always present
			if result.Summary == "" {
				t.Error("Expected summary to always be present")
			}

			// Log result for debugging
			t.Logf("Compression: %s, Level: %d, Summary: %s",
				tc.compressionLevel, result.Level, result.Summary)
			if result.NextLevelHint != "" {
				t.Logf("  Next level hint: %s", result.NextLevelHint)
			}
		})
	}
}
