package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// QueryCostEstimateTool estimates the cost and performance impact of a query
type QueryCostEstimateTool struct {
	*BaseTool
}

// NewQueryCostEstimateTool creates a new QueryCostEstimateTool instance
func NewQueryCostEstimateTool(client *client.Client, logger *zap.Logger) *QueryCostEstimateTool {
	return &QueryCostEstimateTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration
func (t *QueryCostEstimateTool) Name() string {
	return "estimate_query_cost"
}

// Annotations returns tool hints for LLMs
func (t *QueryCostEstimateTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("Estimate Query Cost")
}

// Description returns a human-readable description of the tool
func (t *QueryCostEstimateTool) Description() string {
	return `Estimate the cost and performance impact of a log query before running it.

**Use this tool to:**
- Understand how expensive a query will be before execution
- Get recommendations for optimizing queries
- Predict approximate execution time and data scanned
- Identify potential performance issues

**What it analyzes:**
- Query complexity (aggregations, groupings, sorting)
- Time range scope and expected data volume
- Filter efficiency and index usage
- Resource consumption patterns

**Returns:**
- Estimated cost score (low, medium, high, very_high)
- Predicted execution time range
- Estimated data scan size
- Optimization suggestions
- Resource usage breakdown

**When to use:**
- Before running queries with large time ranges
- When queries include complex aggregations
- For queries without specific filters
- To optimize slow-running queries`
}

// Metadata returns semantic metadata for AI-driven discovery
func (t *QueryCostEstimateTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:    []ToolCategory{CategoryQuery, CategoryAIHelper},
		Keywords:      []string{"cost", "estimate", "performance", "optimize", "query", "expensive", "slow", "analyze"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"Estimate query cost", "Optimize queries", "Predict performance", "Analyze query complexity"},
		RelatedTools:  []string{"query_logs", "build_query", "validate_query", "submit_background_query"},
		ChainPosition: ChainStarter,
	}
}

// InputSchema returns the JSON schema for the tool's input parameters
func (t *QueryCostEstimateTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The DataPrime or Lucene query to analyze",
			},
			"time_range": map[string]interface{}{
				"type":        "string",
				"description": "Time range for the query (e.g., '1h', '24h', '7d', '30d')",
				"default":     "1h",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Result limit for the query",
				"default":     100,
			},
		},
		"required": []string{"query"},
	}
}

// CostEstimate represents the estimated cost of a query
type CostEstimate struct {
	CostLevel         string         `json:"cost_level"`          // low, medium, high, very_high
	CostScore         int            `json:"cost_score"`          // 1-100
	EstimatedTime     string         `json:"estimated_time"`      // e.g., "1-5 seconds"
	EstimatedDataScan string         `json:"estimated_data_scan"` // e.g., "~100MB"
	Complexity        string         `json:"complexity"`          // simple, moderate, complex
	Optimizations     []string       `json:"optimizations"`
	Warnings          []string       `json:"warnings"`
	Breakdown         *CostBreakdown `json:"breakdown"`
}

// CostBreakdown provides detailed cost factors
type CostBreakdown struct {
	TimeRangeCost   int    `json:"time_range_cost"`  // 1-25
	FilterCost      int    `json:"filter_cost"`      // 1-25
	AggregationCost int    `json:"aggregation_cost"` // 1-25
	SortingCost     int    `json:"sorting_cost"`     // 1-25
	TimeRangeNote   string `json:"time_range_note"`
	FilterNote      string `json:"filter_note"`
	AggregationNote string `json:"aggregation_note"`
	SortingNote     string `json:"sorting_note"`
}

// Execute runs the query cost estimation
func (t *QueryCostEstimateTool) Execute(_ context.Context, params map[string]interface{}) (*mcp.CallToolResult, error) {
	query, _ := params["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	timeRange := "1h"
	if tr, ok := params["time_range"].(string); ok && tr != "" {
		timeRange = tr
	}

	limit := 100
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}

	// Analyze the query
	estimate := t.analyzeQuery(query, timeRange, limit)

	// Format the response
	var builder strings.Builder
	builder.WriteString("## Query Cost Estimation\n\n")

	// Cost level with emoji
	costEmoji := "🟢"
	switch estimate.CostLevel {
	case "medium":
		costEmoji = "🟡"
	case "high":
		costEmoji = "🟠"
	case "very_high":
		costEmoji = "🔴"
	}

	builder.WriteString(fmt.Sprintf("**Cost Level:** %s %s (Score: %d/100)\n\n", costEmoji, estimate.CostLevel, estimate.CostScore))
	builder.WriteString(fmt.Sprintf("**Complexity:** %s\n", estimate.Complexity))
	builder.WriteString(fmt.Sprintf("**Estimated Execution Time:** %s\n", estimate.EstimatedTime))
	builder.WriteString(fmt.Sprintf("**Estimated Data Scan:** %s\n\n", estimate.EstimatedDataScan))

	// Breakdown
	builder.WriteString("### Cost Breakdown\n\n")
	builder.WriteString("| Factor | Score | Notes |\n")
	builder.WriteString("|--------|-------|-------|\n")
	builder.WriteString(fmt.Sprintf("| Time Range | %d/25 | %s |\n", estimate.Breakdown.TimeRangeCost, estimate.Breakdown.TimeRangeNote))
	builder.WriteString(fmt.Sprintf("| Filter Efficiency | %d/25 | %s |\n", estimate.Breakdown.FilterCost, estimate.Breakdown.FilterNote))
	builder.WriteString(fmt.Sprintf("| Aggregations | %d/25 | %s |\n", estimate.Breakdown.AggregationCost, estimate.Breakdown.AggregationNote))
	builder.WriteString(fmt.Sprintf("| Sorting | %d/25 | %s |\n\n", estimate.Breakdown.SortingCost, estimate.Breakdown.SortingNote))

	// Warnings
	if len(estimate.Warnings) > 0 {
		builder.WriteString("### ⚠️ Warnings\n\n")
		for _, warning := range estimate.Warnings {
			builder.WriteString(fmt.Sprintf("- %s\n", warning))
		}
		builder.WriteString("\n")
	}

	// Optimizations
	if len(estimate.Optimizations) > 0 {
		builder.WriteString("### 💡 Optimization Suggestions\n\n")
		for _, opt := range estimate.Optimizations {
			builder.WriteString(fmt.Sprintf("- %s\n", opt))
		}
		builder.WriteString("\n")
	}

	// Add query context
	builder.WriteString("### Query Details\n\n")
	builder.WriteString("```\n")
	builder.WriteString(query)
	builder.WriteString("\n```\n")
	builder.WriteString(fmt.Sprintf("Time Range: %s | Limit: %d\n", timeRange, limit))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: builder.String(),
			},
		},
	}, nil
}

// analyzeQuery performs the cost analysis
func (t *QueryCostEstimateTool) analyzeQuery(query, timeRange string, limit int) *CostEstimate {
	estimate := &CostEstimate{
		Breakdown: &CostBreakdown{},
	}

	queryLower := strings.ToLower(query)

	// Analyze time range cost (1-25)
	timeRangeCost, timeNote := t.analyzeTimeRange(timeRange)
	estimate.Breakdown.TimeRangeCost = timeRangeCost
	estimate.Breakdown.TimeRangeNote = timeNote

	// Analyze filter cost (1-25)
	filterCost, filterNote, filterWarnings := t.analyzeFilters(queryLower)
	estimate.Breakdown.FilterCost = filterCost
	estimate.Breakdown.FilterNote = filterNote
	estimate.Warnings = append(estimate.Warnings, filterWarnings...)

	// Analyze aggregation cost (1-25)
	aggCost, aggNote := t.analyzeAggregations(queryLower)
	estimate.Breakdown.AggregationCost = aggCost
	estimate.Breakdown.AggregationNote = aggNote

	// Analyze sorting cost (1-25)
	sortCost, sortNote := t.analyzeSorting(queryLower)
	estimate.Breakdown.SortingCost = sortCost
	estimate.Breakdown.SortingNote = sortNote

	// Calculate total cost score
	totalCost := timeRangeCost + filterCost + aggCost + sortCost
	estimate.CostScore = totalCost

	// Determine cost level
	switch {
	case totalCost <= 30:
		estimate.CostLevel = "low"
	case totalCost <= 50:
		estimate.CostLevel = "medium"
	case totalCost <= 75:
		estimate.CostLevel = "high"
	default:
		estimate.CostLevel = "very_high"
	}

	// Determine complexity
	switch {
	case aggCost <= 5 && sortCost <= 5:
		estimate.Complexity = "simple"
	case aggCost <= 15 && sortCost <= 15:
		estimate.Complexity = "moderate"
	default:
		estimate.Complexity = "complex"
	}

	// Estimate execution time
	estimate.EstimatedTime = t.estimateExecutionTime(totalCost)

	// Estimate data scan
	estimate.EstimatedDataScan = t.estimateDataScan(timeRange, filterCost)

	// Generate optimizations
	estimate.Optimizations = t.generateOptimizations(query, timeRange, limit, estimate.Breakdown)

	// Add limit warning if needed
	if limit > 1000 {
		estimate.Warnings = append(estimate.Warnings, fmt.Sprintf("High result limit (%d) may slow down response", limit))
	}

	return estimate
}

// analyzeTimeRange scores the time range cost
func (t *QueryCostEstimateTool) analyzeTimeRange(timeRange string) (int, string) {
	timeRange = strings.ToLower(timeRange)

	// Parse duration
	hours := 1.0
	if strings.HasSuffix(timeRange, "m") {
		hours = 0.016 // ~1 minute
	} else if strings.HasSuffix(timeRange, "h") {
		_, _ = fmt.Sscanf(timeRange, "%fh", &hours)
	} else if strings.HasSuffix(timeRange, "d") {
		var days float64
		_, _ = fmt.Sscanf(timeRange, "%fd", &days)
		hours = days * 24
	}

	switch {
	case hours <= 1:
		return 5, "Short time range (≤1h)"
	case hours <= 6:
		return 10, "Moderate time range (1-6h)"
	case hours <= 24:
		return 15, "Full day query"
	case hours <= 168: // 7 days
		return 20, "Multi-day query (1-7 days)"
	default:
		return 25, "Long time range (>7 days)"
	}
}

// analyzeFilters scores the filter efficiency
func (t *QueryCostEstimateTool) analyzeFilters(query string) (int, string, []string) {
	warnings := []string{}

	// Check for specific filters
	hasAppFilter := strings.Contains(query, "applicationname") || strings.Contains(query, "$l.application")
	hasSubFilter := strings.Contains(query, "subsystemname") || strings.Contains(query, "$l.subsystem")
	hasSeverityFilter := strings.Contains(query, "severity") || strings.Contains(query, "$m.severity") || strings.Contains(query, "$d.severity")
	hasFieldFilter := strings.Contains(query, "$d.") || strings.Contains(query, "json.")

	filterCount := 0
	if hasAppFilter {
		filterCount++
	}
	if hasSubFilter {
		filterCount++
	}
	if hasSeverityFilter {
		filterCount++
	}
	if hasFieldFilter {
		filterCount++
	}

	// Check for wildcard queries
	hasWildcard := strings.Contains(query, "*") && !strings.Contains(query, "\"*\"")

	// Check for negation (expensive)
	hasNegation := strings.Contains(query, "NOT ") || strings.Contains(query, "!=") || strings.Contains(query, "!~")

	// Score based on filter presence
	var cost int
	var note string

	switch filterCount {
	case 0:
		cost = 25
		note = "No specific filters - full scan"
		warnings = append(warnings, "Query has no specific filters - will scan all data")
	case 1:
		cost = 15
		note = "Single filter"
	case 2:
		cost = 10
		note = "Well-filtered query"
	default:
		cost = 5
		note = "Highly specific query"
	}

	// Add cost for wildcards
	if hasWildcard {
		cost = min(cost+5, 25)
		note += " (with wildcards)"
	}

	// Add cost for negation
	if hasNegation {
		cost = min(cost+3, 25)
		note += " (with negation)"
	}

	return cost, note, warnings
}

// analyzeAggregations scores aggregation complexity
func (t *QueryCostEstimateTool) analyzeAggregations(query string) (int, string) {
	// Count aggregation operations
	aggOps := []string{"groupby", "count", "sum", "avg", "min", "max", "percentile", "distinct", "top", "bottom"}

	aggCount := 0
	for _, op := range aggOps {
		if strings.Contains(query, op) {
			aggCount++
		}
	}

	// Check for window functions
	hasWindow := strings.Contains(query, "window") || strings.Contains(query, "rolling")

	// Check for subqueries
	hasSubquery := strings.Count(query, "source") > 1

	switch {
	case aggCount == 0:
		return 5, "No aggregations"
	case aggCount == 1 && !hasWindow && !hasSubquery:
		return 10, "Simple aggregation"
	case aggCount <= 3 && !hasWindow && !hasSubquery:
		return 15, "Multiple aggregations"
	case hasWindow || hasSubquery:
		return 22, "Complex (window/subquery)"
	default:
		return 20, "Heavy aggregation"
	}
}

// analyzeSorting scores sorting complexity
func (t *QueryCostEstimateTool) analyzeSorting(query string) (int, string) {
	hasSort := strings.Contains(query, "sort") || strings.Contains(query, "order")
	hasLimit := strings.Contains(query, "limit")

	// Check for multiple sort keys (expensive)
	sortRegex := regexp.MustCompile(`sort\s+[\-\+]?\w+(\s*,\s*[\-\+]?\w+)+`)
	hasMultiSort := sortRegex.MatchString(query)

	switch {
	case !hasSort:
		return 5, "No sorting"
	case hasSort && hasLimit && !hasMultiSort:
		return 10, "Sort with limit"
	case hasSort && hasMultiSort:
		return 20, "Multi-key sorting"
	default:
		return 15, "Sorting applied"
	}
}

// estimateExecutionTime estimates query execution time
func (t *QueryCostEstimateTool) estimateExecutionTime(costScore int) string {
	switch {
	case costScore <= 25:
		return "< 1 second"
	case costScore <= 40:
		return "1-5 seconds"
	case costScore <= 60:
		return "5-30 seconds"
	case costScore <= 80:
		return "30 seconds - 2 minutes"
	default:
		return "> 2 minutes"
	}
}

// estimateDataScan estimates data volume scanned
func (t *QueryCostEstimateTool) estimateDataScan(timeRange string, filterCost int) string {
	// Parse time range to hours
	hours := 1.0
	timeRange = strings.ToLower(timeRange)
	if strings.HasSuffix(timeRange, "m") {
		hours = 0.016
	} else if strings.HasSuffix(timeRange, "h") {
		_, _ = fmt.Sscanf(timeRange, "%fh", &hours)
	} else if strings.HasSuffix(timeRange, "d") {
		var days float64
		_, _ = fmt.Sscanf(timeRange, "%fd", &days)
		hours = days * 24
	}

	// Estimate based on typical log volumes
	// Assume ~100MB/hour of logs for a medium-sized deployment
	baseMB := hours * 100

	// Adjust for filter efficiency
	var filterMultiplier float64
	switch {
	case filterCost <= 10:
		filterMultiplier = 0.1 // Good filters = 10% scan
	case filterCost <= 15:
		filterMultiplier = 0.3 // Moderate filters = 30% scan
	case filterCost <= 20:
		filterMultiplier = 0.6 // Weak filters = 60% scan
	default:
		filterMultiplier = 1.0 // No filters = full scan
	}

	estimatedMB := baseMB * filterMultiplier

	switch {
	case estimatedMB < 100:
		return fmt.Sprintf("~%.0f MB", estimatedMB)
	case estimatedMB < 1000:
		return fmt.Sprintf("~%.1f GB", estimatedMB/1000)
	default:
		return fmt.Sprintf("~%.1f GB", estimatedMB/1000)
	}
}

// generateOptimizations creates optimization suggestions
func (t *QueryCostEstimateTool) generateOptimizations(query, _ string, limit int, breakdown *CostBreakdown) []string {
	optimizations := []string{}

	// Time range optimizations
	if breakdown.TimeRangeCost >= 20 {
		optimizations = append(optimizations, "Consider reducing time range if possible - start with a smaller window and expand if needed")
	}

	// Filter optimizations
	if breakdown.FilterCost >= 20 {
		optimizations = append(optimizations, "Add specific filters (applicationname, subsystemname, severity) to reduce data scan")
	}
	if strings.Contains(strings.ToLower(query), "*") {
		optimizations = append(optimizations, "Replace wildcards with specific values when possible")
	}

	// Aggregation optimizations
	if breakdown.AggregationCost >= 15 {
		optimizations = append(optimizations, "Consider adding a LIMIT to aggregation results")
	}
	if strings.Contains(strings.ToLower(query), "distinct") {
		optimizations = append(optimizations, "DISTINCT operations are expensive - ensure you have good filters first")
	}

	// Limit optimizations
	if limit > 500 {
		optimizations = append(optimizations, fmt.Sprintf("Consider reducing result limit from %d - fetch more if needed", limit))
	}

	// General optimizations
	if breakdown.TimeRangeCost >= 15 && breakdown.FilterCost >= 15 {
		optimizations = append(optimizations, "For large scans, consider using submit_background_query instead of query_logs")
	}

	return optimizations
}
