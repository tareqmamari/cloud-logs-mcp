// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements budget-aware execution based on research from:
// - "Budget-Aware Tool-Use Enables Effective Agent Scaling" (arXiv:2511.17006)
// - "ACON: Optimizing Context Compression for Long-horizon LLM Agents" (arXiv:2510.00615)
package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// BudgetContext tracks token/cost budget for a session.
// Based on research showing budget-aware decisions improve agent performance.
type BudgetContext struct {
	// Token tracking
	MaxTokens       int `json:"max_tokens"`
	UsedTokens      int `json:"used_tokens"`
	RemainingTokens int `json:"remaining_tokens"`

	// Cost tracking (in millicents for precision)
	MaxCostMillicents  int `json:"max_cost_millicents"`
	UsedCostMillicents int `json:"used_cost_millicents"`

	// Execution tracking
	ToolCallCount    int       `json:"tool_call_count"`
	SessionStartTime time.Time `json:"session_start_time"`

	// Adaptive behavior
	ResultCompression BudgetCompressionLevel `json:"compression_level"`

	// Token counting method
	TokenCountingMethod string `json:"token_counting_method"` // "exact" or "approximate"
	IsExactCount        bool   `json:"is_exact_count"`

	mu sync.RWMutex
}

// BudgetCompressionLevel determines how aggressively to compress results.
// Based on ACON research showing adaptive compression preserves performance.
type BudgetCompressionLevel string

// BudgetCompressionLevel constants define compression aggressiveness
const (
	BudgetCompressionNone    BudgetCompressionLevel = "none"    // Full results
	BudgetCompressionLight   BudgetCompressionLevel = "light"   // Remove verbose metadata
	BudgetCompressionMedium  BudgetCompressionLevel = "medium"  // Summarize + samples
	BudgetCompressionHeavy   BudgetCompressionLevel = "heavy"   // Stats only
	BudgetCompressionMinimal BudgetCompressionLevel = "minimal" // One-line summary
)

// compressionOrder maps compression levels to numeric values for comparison
var compressionOrder = map[BudgetCompressionLevel]int{
	BudgetCompressionNone:    0,
	BudgetCompressionLight:   1,
	BudgetCompressionMedium:  2,
	BudgetCompressionHeavy:   3,
	BudgetCompressionMinimal: 4,
}

// CompressionLessOrEqual returns true if level a is less aggressive than or equal to level b
func CompressionLessOrEqual(a, b BudgetCompressionLevel) bool {
	return compressionOrder[a] <= compressionOrder[b]
}

// Token cost estimates per 1K tokens (based on Claude pricing)
const (
	InputTokenCostPer1K      = 3  // $0.003 per 1K input tokens (millicents)
	OutputTokenCostPer1K     = 15 // $0.015 per 1K output tokens (millicents)
	DefaultMaxTokens         = 100000
	DefaultMaxCostMillicents = 10000 // $0.10 default budget
)

// Global budget context (per-session)
var (
	globalBudget *BudgetContext
	budgetMu     sync.RWMutex
)

// GetBudgetContext returns the current session's budget context
func GetBudgetContext() *BudgetContext {
	budgetMu.RLock()
	if globalBudget != nil {
		budgetMu.RUnlock()
		return globalBudget
	}
	budgetMu.RUnlock()

	budgetMu.Lock()
	defer budgetMu.Unlock()
	if globalBudget == nil {
		globalBudget = NewBudgetContext(DefaultMaxTokens, DefaultMaxCostMillicents)
	}
	return globalBudget
}

// ResetBudgetContext resets the budget for a new session
func ResetBudgetContext() {
	budgetMu.Lock()
	defer budgetMu.Unlock()
	globalBudget = NewBudgetContext(DefaultMaxTokens, DefaultMaxCostMillicents)
}

// NewBudgetContext creates a new budget context with specified limits
func NewBudgetContext(maxTokens, maxCostMillicents int) *BudgetContext {
	counter := GetTokenCounter()
	return &BudgetContext{
		MaxTokens:           maxTokens,
		UsedTokens:          0,
		RemainingTokens:     maxTokens,
		MaxCostMillicents:   maxCostMillicents,
		UsedCostMillicents:  0,
		ToolCallCount:       0,
		SessionStartTime:    time.Now(),
		ResultCompression:   BudgetCompressionNone,
		TokenCountingMethod: counter.Name(),
		IsExactCount:        counter.IsExact(),
	}
}

// RecordToolExecution records token usage from a tool execution (estimated)
func (b *BudgetContext) RecordToolExecution(inputTokens, outputTokens int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	totalTokens := inputTokens + outputTokens
	b.UsedTokens += totalTokens
	b.RemainingTokens = b.MaxTokens - b.UsedTokens
	b.ToolCallCount++

	// Calculate cost
	inputCost := (inputTokens * InputTokenCostPer1K) / 1000
	outputCost := (outputTokens * OutputTokenCostPer1K) / 1000
	b.UsedCostMillicents += inputCost + outputCost

	// Adjust compression level based on remaining budget
	b.updateCompressionLevel()
}

// RecordClientReportedTokens records exact token counts from the MCP client
// This is the preferred method when the client provides actual token usage
func (b *BudgetContext) RecordClientReportedTokens(inputTokens, outputTokens int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	totalTokens := inputTokens + outputTokens
	b.UsedTokens += totalTokens
	b.RemainingTokens = b.MaxTokens - b.UsedTokens
	b.ToolCallCount++

	// Mark as exact count
	b.IsExactCount = true
	b.TokenCountingMethod = "client-reported"

	// Calculate cost
	inputCost := (inputTokens * InputTokenCostPer1K) / 1000
	outputCost := (outputTokens * OutputTokenCostPer1K) / 1000
	b.UsedCostMillicents += inputCost + outputCost

	// Adjust compression level based on remaining budget
	b.updateCompressionLevel()
}

// updateCompressionLevel adjusts compression based on budget consumption
func (b *BudgetContext) updateCompressionLevel() {
	usageRatio := float64(b.UsedTokens) / float64(b.MaxTokens)

	switch {
	case usageRatio >= 0.9:
		b.ResultCompression = BudgetCompressionMinimal
	case usageRatio >= 0.75:
		b.ResultCompression = BudgetCompressionHeavy
	case usageRatio >= 0.5:
		b.ResultCompression = BudgetCompressionMedium
	case usageRatio >= 0.25:
		b.ResultCompression = BudgetCompressionLight
	default:
		b.ResultCompression = BudgetCompressionNone
	}
}

// ShouldExecute checks if a tool should execute given current budget
func (b *BudgetContext) ShouldExecute(estimatedTokens int) (bool, string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.RemainingTokens < estimatedTokens {
		return false, fmt.Sprintf("Insufficient token budget: need ~%d, have %d remaining",
			estimatedTokens, b.RemainingTokens)
	}

	estimatedCost := (estimatedTokens * OutputTokenCostPer1K) / 1000
	if b.UsedCostMillicents+estimatedCost > b.MaxCostMillicents {
		return false, fmt.Sprintf("Would exceed cost budget: estimated $%.4f, budget remaining $%.4f",
			float64(estimatedCost)/100, float64(b.MaxCostMillicents-b.UsedCostMillicents)/100)
	}

	return true, ""
}

// GetCompressionLevel returns the current recommended compression level
func (b *BudgetContext) GetCompressionLevel() BudgetCompressionLevel {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.ResultCompression
}

// GetSummary returns a summary of budget usage
func (b *BudgetContext) GetSummary() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	usagePct := float64(0)
	if b.MaxTokens > 0 {
		usagePct = float64(b.UsedTokens) / float64(b.MaxTokens) * 100
	}

	tokenAccuracy := "approximate"
	if b.IsExactCount {
		tokenAccuracy = "exact"
	}

	return map[string]interface{}{
		"tokens": map[string]interface{}{
			"used":            b.UsedTokens,
			"remaining":       b.RemainingTokens,
			"max":             b.MaxTokens,
			"usage_pct":       usagePct,
			"counting_method": b.TokenCountingMethod,
			"accuracy":        tokenAccuracy,
		},
		"cost": map[string]interface{}{
			"used_millicents":      b.UsedCostMillicents,
			"remaining_millicents": b.MaxCostMillicents - b.UsedCostMillicents,
			"max_millicents":       b.MaxCostMillicents,
			"remaining_pct":        float64(b.MaxCostMillicents-b.UsedCostMillicents) / float64(b.MaxCostMillicents) * 100,
		},
		"execution": map[string]interface{}{
			"tool_calls":        b.ToolCallCount,
			"session_duration":  time.Since(b.SessionStartTime).String(),
			"compression_level": string(b.ResultCompression),
		},
	}
}

// TokenMetrics tracks token usage for a single tool execution
type TokenMetrics struct {
	ToolName         string `json:"tool_name"`
	InputTokens      int    `json:"input_tokens"`
	OutputTokens     int    `json:"output_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	EstimatedCostUSD string `json:"estimated_cost_usd"`
	Compressed       bool   `json:"compressed"`
	CompressionRatio string `json:"compression_ratio,omitempty"`
}

// TokenCounter defines the interface for token counting strategies
type TokenCounter interface {
	CountTokens(text string) int
	Name() string
	IsExact() bool
}

// ApproximateTokenCounter uses character-based estimation (~4 chars/token)
// This is a rough approximation and should only be used when exact counts are unavailable
type ApproximateTokenCounter struct{}

// CountTokens implements TokenCounter.
func (c *ApproximateTokenCounter) CountTokens(text string) int {
	return (len(text) + 3) / 4
}

// Name implements TokenCounter.
func (c *ApproximateTokenCounter) Name() string {
	return "approximate (chars/4)"
}

// IsExact implements TokenCounter.
func (c *ApproximateTokenCounter) IsExact() bool {
	return false
}

// ClientReportedTokenCounter uses token counts reported by the MCP client
type ClientReportedTokenCounter struct {
	lastInputTokens  int
	lastOutputTokens int
}

// CountTokens implements TokenCounter.
func (c *ClientReportedTokenCounter) CountTokens(_ string) int {
	// Not used directly - tokens are reported via RecordClientTokens
	return 0
}

// Name implements TokenCounter.
func (c *ClientReportedTokenCounter) Name() string {
	return "client-reported"
}

// IsExact implements TokenCounter.
func (c *ClientReportedTokenCounter) IsExact() bool {
	return true
}

// RecordClientTokens records token counts reported by the MCP client
func (c *ClientReportedTokenCounter) RecordClientTokens(input, output int) {
	c.lastInputTokens = input
	c.lastOutputTokens = output
}

// GetLastTokens returns the last reported token counts
func (c *ClientReportedTokenCounter) GetLastTokens() (input, output int) {
	return c.lastInputTokens, c.lastOutputTokens
}

// Default token counter (approximate)
var defaultTokenCounter TokenCounter = &ApproximateTokenCounter{}

// SetTokenCounter sets the token counting strategy
func SetTokenCounter(counter TokenCounter) {
	defaultTokenCounter = counter
}

// GetTokenCounter returns the current token counter
func GetTokenCounter() TokenCounter {
	return defaultTokenCounter
}

// EstimateTokens estimates token count for a string
// Note: Uses approximate counting (~4 chars/token) unless a client-reported counter is set
func EstimateTokens(text string) int {
	return defaultTokenCounter.CountTokens(text)
}

// EstimateJSONTokens estimates tokens for a JSON structure
func EstimateJSONTokens(data interface{}) int {
	bytes, err := json.Marshal(data)
	if err != nil {
		return 100 // Default estimate
	}
	return EstimateTokens(string(bytes))
}

// CreateTokenMetrics creates token metrics for a tool result
func CreateTokenMetrics(toolName string, inputArgs interface{}, result interface{}, compressed bool, originalSize int) *TokenMetrics {
	inputTokens := EstimateJSONTokens(inputArgs)
	outputTokens := EstimateJSONTokens(result)

	costMillicents := (inputTokens*InputTokenCostPer1K + outputTokens*OutputTokenCostPer1K) / 1000

	metrics := &TokenMetrics{
		ToolName:         toolName,
		InputTokens:      inputTokens,
		OutputTokens:     outputTokens,
		TotalTokens:      inputTokens + outputTokens,
		EstimatedCostUSD: fmt.Sprintf("$%.4f", float64(costMillicents)/100),
		Compressed:       compressed,
	}

	if compressed && originalSize > 0 {
		currentSize := outputTokens
		ratio := float64(currentSize) / float64(originalSize)
		metrics.CompressionRatio = fmt.Sprintf("%.1f%% of original", ratio*100)
	}

	return metrics
}

// ProgressiveResult implements progressive disclosure pattern.
// Based on research showing summary-first reduces token usage by 40-60%.
type ProgressiveResult struct {
	// Level 1: Always included - minimal summary
	Summary    string `json:"summary"`
	TotalCount int    `json:"total_count"`
	HasMore    bool   `json:"has_more"`

	// Level 2: Key insights (included if budget allows)
	Insights *BudgetResultInsights `json:"insights,omitempty"`

	// Level 3: Sample data (included if budget allows)
	Samples     []interface{} `json:"samples,omitempty"`
	SampleCount int           `json:"sample_count,omitempty"`

	// Level 4: Full data (only on explicit request)
	FullData interface{} `json:"full_data,omitempty"`

	// Metadata
	Level          int           `json:"disclosure_level"`
	TokenMetrics   *TokenMetrics `json:"_token_metrics,omitempty"`
	NextLevelHint  string        `json:"next_level_hint,omitempty"`
	DrillDownQuery string        `json:"drill_down_query,omitempty"`
}

// BudgetResultInsights contains aggregated insights from results
type BudgetResultInsights struct {
	TopValues    map[string][]ValueCount `json:"top_values,omitempty"`
	Distribution map[string]int          `json:"distribution,omitempty"`
	TimeRange    *BudgetTimeRange        `json:"time_range,omitempty"`
	Anomalies    []string                `json:"anomalies,omitempty"`
	Patterns     []string                `json:"patterns,omitempty"`
}

// BudgetTimeRange represents a time range
type BudgetTimeRange struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Duration string `json:"duration"`
}

// CreateProgressiveResult creates a progressive result based on budget
func CreateProgressiveResult(data interface{}, budget *BudgetContext) *ProgressiveResult {
	result := &ProgressiveResult{
		Level: 1,
	}

	// Always generate Level 1: Summary
	result.Summary, result.TotalCount = budgetGenerateSummary(data)
	result.HasMore = result.TotalCount > 0

	compression := budget.GetCompressionLevel()

	// Level 2: Add insights if budget allows (none, light, or medium compression)
	if CompressionLessOrEqual(compression, BudgetCompressionMedium) {
		result.Insights = budgetGenerateInsights(data)
		result.Level = 2
	}

	// Level 3: Add samples if budget allows (none or light compression)
	if CompressionLessOrEqual(compression, BudgetCompressionLight) {
		result.Samples, result.SampleCount = budgetExtractSamples(data, 5)
		result.Level = 3
	}

	// Level 4: Full data only if no compression
	if compression == BudgetCompressionNone {
		result.FullData = data
		result.Level = 4
	}

	// Add hints for drilling down
	if result.Level < 4 {
		result.NextLevelHint = fmt.Sprintf("To see full data (%d items), use summary_only=false", result.TotalCount)
	}

	return result
}

// budgetGenerateSummary creates a one-line summary of data
func budgetGenerateSummary(data interface{}) (string, int) {
	switch v := data.(type) {
	case map[string]interface{}:
		// Check for events/logs array
		if events, ok := v["events"].([]interface{}); ok {
			return fmt.Sprintf("Found %d log entries", len(events)), len(events)
		}
		if logs, ok := v["logs"].([]interface{}); ok {
			return fmt.Sprintf("Found %d log entries", len(logs)), len(logs)
		}
		// Check for list results
		for key, val := range v {
			if arr, ok := val.([]interface{}); ok {
				return fmt.Sprintf("Found %d %s", len(arr), key), len(arr)
			}
		}
		return "Single result returned", 1

	case []interface{}:
		return fmt.Sprintf("Found %d items", len(v)), len(v)

	default:
		return "Result returned", 1
	}
}

// budgetGenerateInsights extracts key insights from data
func budgetGenerateInsights(data interface{}) *BudgetResultInsights {
	insights := &BudgetResultInsights{
		TopValues:    make(map[string][]ValueCount),
		Distribution: make(map[string]int),
	}

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return insights
	}

	// Find events array
	var events []interface{}
	if e, ok := dataMap["events"].([]interface{}); ok {
		events = e
	} else if l, ok := dataMap["logs"].([]interface{}); ok {
		events = l
	}

	if len(events) == 0 {
		return insights
	}

	// Extract severity distribution
	severityDist := analyzeSeverityDistribution(events)
	for sev, count := range severityDist {
		insights.Distribution[sev] = count
	}

	// Extract top applications
	topApps := extractTopValues(events, "applicationname", 3)
	if len(topApps) > 0 {
		insights.TopValues["applications"] = topApps
	}

	// Extract time range
	timeRange := extractTimeRange(events)
	if timeRange != "" {
		parts := strings.Split(timeRange, "\n")
		if len(parts) >= 2 {
			insights.TimeRange = &BudgetTimeRange{
				Start: strings.TrimPrefix(parts[0], "From: "),
				End:   strings.TrimPrefix(parts[1], "To: "),
			}
		}
	}

	// Detect anomalies (simple pattern detection)
	insights.Anomalies = budgetDetectAnomalies(events)
	insights.Patterns = budgetDetectPatterns(events)

	return insights
}

// budgetExtractSamples extracts representative samples from data
func budgetExtractSamples(data interface{}, count int) ([]interface{}, int) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, 0
	}

	// Find events array
	var events []interface{}
	if e, ok := dataMap["events"].([]interface{}); ok {
		events = e
	} else if l, ok := dataMap["logs"].([]interface{}); ok {
		events = l
	}

	if len(events) == 0 {
		return nil, 0
	}

	// Take first N and last N for representative samples
	samples := make([]interface{}, 0, count)
	halfCount := count / 2

	// First half from beginning
	for i := 0; i < halfCount && i < len(events); i++ {
		samples = append(samples, events[i])
	}

	// Second half from end (if different from beginning)
	startIdx := len(events) - (count - halfCount)
	if startIdx < halfCount {
		startIdx = halfCount
	}
	for i := startIdx; i < len(events); i++ {
		samples = append(samples, events[i])
	}

	return samples, len(samples)
}

// budgetDetectAnomalies detects potential anomalies in events
func budgetDetectAnomalies(events []interface{}) []string {
	var anomalies []string

	// Count errors
	errorCount := 0
	criticalCount := 0
	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			if sev, ok := eventMap["severity"].(float64); ok {
				if sev >= 5 {
					errorCount++
				}
				if sev >= 6 {
					criticalCount++
				}
			}
		}
	}

	// Check for high error ratio
	if len(events) > 0 {
		errorRatio := float64(errorCount) / float64(len(events))
		if errorRatio > 0.5 {
			anomalies = append(anomalies, fmt.Sprintf("High error rate: %.0f%% of events are errors", errorRatio*100))
		}
		if criticalCount > 0 {
			anomalies = append(anomalies, fmt.Sprintf("%d critical events detected", criticalCount))
		}
	}

	return anomalies
}

// budgetDetectPatterns detects common patterns in events
func budgetDetectPatterns(events []interface{}) []string {
	var patterns []string

	// Count messages to find repetitive patterns
	messageCounts := make(map[string]int)
	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			if msg, ok := eventMap["message"].(string); ok {
				// Truncate to find pattern
				if len(msg) > 50 {
					msg = msg[:50]
				}
				messageCounts[msg]++
			}
		}
	}

	// Find repetitive messages
	for msg, count := range messageCounts {
		if count > 5 && float64(count)/float64(len(events)) > 0.1 {
			patterns = append(patterns, fmt.Sprintf("Repeated %dx: \"%s...\"", count, msg))
		}
	}

	if len(patterns) > 3 {
		patterns = patterns[:3]
	}

	return patterns
}
