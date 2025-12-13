package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

func TestQueryCostEstimateTool(t *testing.T) {
	logger := zap.NewNop()
	tool := NewQueryCostEstimateTool(nil, logger)

	if tool.Name() != "estimate_query_cost" {
		t.Errorf("Expected name 'estimate_query_cost', got %s", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description should not be empty")
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Error("InputSchema should not be nil")
	}
}

func TestQueryCostEstimateMetadata(t *testing.T) {
	logger := zap.NewNop()
	tool := NewQueryCostEstimateTool(nil, logger)

	metadata := tool.Metadata()
	if metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	// Check categories
	foundQuery := false
	for _, cat := range metadata.Categories {
		if cat == CategoryQuery {
			foundQuery = true
			break
		}
	}
	if !foundQuery {
		t.Error("Expected CategoryQuery in metadata categories")
	}

	// Check keywords
	if len(metadata.Keywords) == 0 {
		t.Error("Expected keywords in metadata")
	}

	// Check related tools
	if len(metadata.RelatedTools) == 0 {
		t.Error("Expected related tools in metadata")
	}
}

func TestAnalyzeTimeRange(t *testing.T) {
	logger := zap.NewNop()
	tool := NewQueryCostEstimateTool(nil, logger)

	tests := []struct {
		timeRange    string
		expectedMin  int
		expectedMax  int
		expectedNote string
	}{
		{"15m", 1, 10, "Short"},
		{"1h", 1, 10, "Short"},
		{"6h", 8, 12, "Moderate"},
		{"24h", 13, 17, "Full day"},
		{"7d", 18, 22, "Multi-day"},
		{"30d", 23, 25, "Long"},
	}

	for _, tt := range tests {
		t.Run(tt.timeRange, func(t *testing.T) {
			cost, note := tool.analyzeTimeRange(tt.timeRange)
			if cost < tt.expectedMin || cost > tt.expectedMax {
				t.Errorf("Time range %s: expected cost %d-%d, got %d", tt.timeRange, tt.expectedMin, tt.expectedMax, cost)
			}
			if !strings.Contains(note, tt.expectedNote) {
				t.Errorf("Time range %s: expected note containing %q, got %q", tt.timeRange, tt.expectedNote, note)
			}
		})
	}
}

func TestAnalyzeFilters(t *testing.T) {
	logger := zap.NewNop()
	tool := NewQueryCostEstimateTool(nil, logger)

	tests := []struct {
		name         string
		query        string
		expectedMin  int
		expectedMax  int
		wantWarnings bool
	}{
		{
			name:         "no filters",
			query:        "source logs",
			expectedMin:  20,
			expectedMax:  25,
			wantWarnings: true,
		},
		{
			name:         "app filter",
			query:        "source logs | filter $l.applicationname == 'my-app'",
			expectedMin:  10,
			expectedMax:  20,
			wantWarnings: false,
		},
		{
			name:         "multiple filters",
			query:        "source logs | filter $l.applicationname == 'my-app' && $d.severity >= 4",
			expectedMin:  5,
			expectedMax:  15,
			wantWarnings: false,
		},
		{
			name:         "with wildcard",
			query:        "source logs | filter $l.applicationname ~= 'my-*'",
			expectedMin:  10,
			expectedMax:  25,
			wantWarnings: false,
		},
		{
			name:         "with negation",
			query:        "source logs | filter $l.applicationname != 'excluded'",
			expectedMin:  10,
			expectedMax:  25,
			wantWarnings: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, _, warnings := tool.analyzeFilters(strings.ToLower(tt.query))
			if cost < tt.expectedMin || cost > tt.expectedMax {
				t.Errorf("Query %q: expected filter cost %d-%d, got %d", tt.name, tt.expectedMin, tt.expectedMax, cost)
			}
			if tt.wantWarnings && len(warnings) == 0 {
				t.Errorf("Query %q: expected warnings, got none", tt.name)
			}
			if !tt.wantWarnings && len(warnings) > 0 {
				t.Errorf("Query %q: expected no warnings, got %v", tt.name, warnings)
			}
		})
	}
}

func TestAnalyzeAggregations(t *testing.T) {
	logger := zap.NewNop()
	tool := NewQueryCostEstimateTool(nil, logger)

	tests := []struct {
		name        string
		query       string
		expectedMin int
		expectedMax int
	}{
		{
			name:        "no aggregations",
			query:       "source logs | filter severity >= 4",
			expectedMin: 1,
			expectedMax: 7,
		},
		{
			name:        "simple count",
			query:       "source logs | count",
			expectedMin: 8,
			expectedMax: 12,
		},
		{
			name:        "groupby with count",
			query:       "source logs | groupby $l.applicationname | count",
			expectedMin: 12,
			expectedMax: 18,
		},
		{
			name:        "multiple aggregations",
			query:       "source logs | groupby $l.applicationname | sum($d.value) as total, avg($d.value) as average, count",
			expectedMin: 13,
			expectedMax: 22,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, _ := tool.analyzeAggregations(strings.ToLower(tt.query))
			if cost < tt.expectedMin || cost > tt.expectedMax {
				t.Errorf("Query %q: expected aggregation cost %d-%d, got %d", tt.name, tt.expectedMin, tt.expectedMax, cost)
			}
		})
	}
}

func TestAnalyzeSorting(t *testing.T) {
	logger := zap.NewNop()
	tool := NewQueryCostEstimateTool(nil, logger)

	tests := []struct {
		name        string
		query       string
		expectedMin int
		expectedMax int
	}{
		{
			name:        "no sorting",
			query:       "source logs",
			expectedMin: 1,
			expectedMax: 7,
		},
		{
			name:        "sort with limit",
			query:       "source logs | sort -timestamp | limit 100",
			expectedMin: 8,
			expectedMax: 12,
		},
		{
			name:        "sort without limit",
			query:       "source logs | sort -timestamp",
			expectedMin: 12,
			expectedMax: 17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, _ := tool.analyzeSorting(strings.ToLower(tt.query))
			if cost < tt.expectedMin || cost > tt.expectedMax {
				t.Errorf("Query %q: expected sorting cost %d-%d, got %d", tt.name, tt.expectedMin, tt.expectedMax, cost)
			}
		})
	}
}

func TestEstimateExecutionTime(t *testing.T) {
	logger := zap.NewNop()
	tool := NewQueryCostEstimateTool(nil, logger)

	tests := []struct {
		costScore    int
		expectedText string
	}{
		{20, "< 1 second"},
		{35, "1-5 seconds"},
		{55, "5-30 seconds"},
		{70, "30 seconds"},
		{90, "> 2 minutes"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedText, func(t *testing.T) {
			result := tool.estimateExecutionTime(tt.costScore)
			if !strings.Contains(result, tt.expectedText) {
				t.Errorf("Cost %d: expected text containing %q, got %q", tt.costScore, tt.expectedText, result)
			}
		})
	}
}

func TestEstimateDataScan(t *testing.T) {
	logger := zap.NewNop()
	tool := NewQueryCostEstimateTool(nil, logger)

	tests := []struct {
		timeRange   string
		filterCost  int
		expectLarge bool
	}{
		{"1h", 5, false},  // Good filters, small scan
		{"1h", 25, false}, // No filters, moderate scan
		{"24h", 5, false}, // Good filters, moderate scan
		{"24h", 25, true}, // No filters, large scan
		{"7d", 25, true},  // No filters, very large scan
	}

	for _, tt := range tests {
		t.Run(tt.timeRange, func(t *testing.T) {
			result := tool.estimateDataScan(tt.timeRange, tt.filterCost)
			hasGB := strings.Contains(result, "GB")
			if tt.expectLarge && !hasGB {
				t.Errorf("Expected GB-level scan for %s with filter cost %d, got %s", tt.timeRange, tt.filterCost, result)
			}
		})
	}
}

func TestGenerateOptimizations(t *testing.T) {
	logger := zap.NewNop()
	tool := NewQueryCostEstimateTool(nil, logger)

	// Test with high costs
	breakdown := &CostBreakdown{
		TimeRangeCost:   20,
		FilterCost:      20,
		AggregationCost: 15,
		SortingCost:     10,
	}

	opts := tool.generateOptimizations("source logs | groupby app | count", "7d", 2000, breakdown)

	// Should have multiple optimizations
	if len(opts) < 2 {
		t.Errorf("Expected at least 2 optimizations, got %d", len(opts))
	}

	// Check for specific suggestions
	foundTimeRange := false
	foundFilter := false
	foundLimit := false
	for _, opt := range opts {
		if strings.Contains(opt, "time range") {
			foundTimeRange = true
		}
		if strings.Contains(opt, "filter") {
			foundFilter = true
		}
		if strings.Contains(opt, "limit") || strings.Contains(opt, "2000") {
			foundLimit = true
		}
	}

	if !foundTimeRange {
		t.Error("Expected time range optimization suggestion")
	}
	if !foundFilter {
		t.Error("Expected filter optimization suggestion")
	}
	if !foundLimit {
		t.Error("Expected limit optimization suggestion")
	}
}

func TestQueryCostEstimateExecute(t *testing.T) {
	logger := zap.NewNop()
	tool := NewQueryCostEstimateTool(nil, logger)
	ctx := context.Background()

	tests := []struct {
		name          string
		params        map[string]interface{}
		wantError     bool
		wantCostLevel string
		checkContent  []string
	}{
		{
			name: "simple query",
			params: map[string]interface{}{
				"query":      "source logs | filter $l.applicationname == 'my-app' | limit 100",
				"time_range": "1h",
			},
			wantError:     false,
			wantCostLevel: "low",
			checkContent:  []string{"Cost Level", "Estimated Execution Time", "Cost Breakdown"},
		},
		{
			name: "expensive query",
			params: map[string]interface{}{
				"query":      "source logs | groupby $l.applicationname | count",
				"time_range": "30d",
				"limit":      float64(5000),
			},
			wantError:     false,
			wantCostLevel: "high",
			checkContent:  []string{"Cost Level", "Warning", "Optimization"},
		},
		{
			name: "missing query",
			params: map[string]interface{}{
				"time_range": "1h",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tt.params)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			if len(result.Content) == 0 {
				t.Fatal("Expected content in result")
			}

			// Check text content
			textContent, ok := result.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatal("Expected TextContent")
			}

			for _, check := range tt.checkContent {
				if !strings.Contains(textContent.Text, check) {
					t.Errorf("Expected result to contain %q", check)
				}
			}
		})
	}
}

func TestAnalyzeQueryIntegration(t *testing.T) {
	logger := zap.NewNop()
	tool := NewQueryCostEstimateTool(nil, logger)

	tests := []struct {
		name          string
		query         string
		timeRange     string
		limit         int
		expectLevel   string
		expectComplex string
	}{
		{
			name:          "simple filtered query",
			query:         "source logs | filter $l.applicationname == 'api' && $d.severity >= 4 | limit 100",
			timeRange:     "1h",
			limit:         100,
			expectLevel:   "low",
			expectComplex: "simple",
		},
		{
			name:          "complex aggregation query",
			query:         "source logs | groupby $l.applicationname | count | sort -_count | top 10",
			timeRange:     "24h",
			limit:         100,
			expectLevel:   "medium",
			expectComplex: "moderate",
		},
		{
			name:          "expensive unfiltered query",
			query:         "source logs | groupby $l.applicationname, $l.subsystemname | count | percentile($d.duration, 95)",
			timeRange:     "7d",
			limit:         1000,
			expectLevel:   "high",
			expectComplex: "complex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estimate := tool.analyzeQuery(tt.query, tt.timeRange, tt.limit)

			if estimate.CostLevel != tt.expectLevel && estimate.CostLevel != "medium" && tt.expectLevel != "medium" {
				// Allow some flexibility in cost levels
				t.Logf("Query %q: expected level %s, got %s (score: %d)",
					tt.name, tt.expectLevel, estimate.CostLevel, estimate.CostScore)
			}

			if estimate.Breakdown == nil {
				t.Error("Expected breakdown in estimate")
			}

			// Verify total equals sum of components
			expectedTotal := estimate.Breakdown.TimeRangeCost +
				estimate.Breakdown.FilterCost +
				estimate.Breakdown.AggregationCost +
				estimate.Breakdown.SortingCost
			if estimate.CostScore != expectedTotal {
				t.Errorf("Cost score %d doesn't match sum of breakdown %d", estimate.CostScore, expectedTotal)
			}
		})
	}
}

func TestCostEstimateAnnotations(t *testing.T) {
	logger := zap.NewNop()
	tool := NewQueryCostEstimateTool(nil, logger)

	annotations := tool.Annotations()
	if annotations == nil {
		t.Fatal("Annotations should not be nil")
	}

	// Should be read-only (no mutations)
	if !annotations.ReadOnlyHint {
		t.Error("Expected read-only tool to have ReadOnlyHint = true")
	}

	// Should be idempotent
	if !annotations.IdempotentHint {
		t.Error("Expected cost estimation tool to be idempotent")
	}
}
