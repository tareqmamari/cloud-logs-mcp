// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file contains the Scout tool for pattern discovery and root cause analysis.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// Common noise patterns that typically don't indicate real problems
var defaultNoisePatterns = []string{
	"health", "healthcheck", "health_check", "health-check",
	"readiness", "liveness", "ping", "pong",
	"keepalive", "keep-alive", "heartbeat",
	"metrics", "prometheus", "/metrics",
	"200 OK", "status=200", "HTTP 200",
}

// ScoutLogsTool provides pattern discovery for root cause analysis.
// It builds aggregation queries to identify hotspots without requiring
// specific knowledge of what to look for.
type ScoutLogsTool struct {
	*BaseTool
}

// NewScoutLogsTool creates a new ScoutLogsTool
func NewScoutLogsTool(c *client.Client, l *zap.Logger) *ScoutLogsTool {
	return &ScoutLogsTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ScoutLogsTool) Name() string { return "scout_logs" }

// Annotations returns tool hints for LLMs
func (t *ScoutLogsTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("Scout Logs")
}

// Description returns the tool description
func (t *ScoutLogsTool) Description() string {
	return `Discovery tool for root cause analysis - find patterns without knowing what to look for.

**Best for:** "What's wrong?" investigations, incident triage, pattern discovery.

**Unlike query_logs:** Scout builds aggregation queries to identify hotspots across your entire environment, then helps you drill down.

**Discovery Modes:**
- error_hotspots: Find which services/components have the most errors
- severity_distribution: See error/warning/info breakdown by service
- top_error_messages: Group similar errors to find the most common issues
- anomaly_scan: Find unusual patterns (high error rates, traffic spikes)
- noise_filtered: Exclude health checks and known noise patterns

**Output:** Aggregated summaries, not raw logs. Use query_logs to drill down after finding hotspots.

**Related tools:** query_logs (drill down), investigate_incident (guided investigation)`
}

// InputSchema returns the input schema
func (t *ScoutLogsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"mode": map[string]interface{}{
				"type":        "string",
				"description": "Discovery mode - what pattern to look for",
				"enum":        []string{"error_hotspots", "severity_distribution", "top_error_messages", "anomaly_scan", "traffic_overview", "recent_deployments"},
				"default":     "error_hotspots",
			},
			"time_range": map[string]interface{}{
				"type":        "string",
				"description": "Time range to analyze: 15m, 1h, 6h, 24h (default: 1h)",
				"enum":        []string{"15m", "1h", "6h", "24h"},
				"default":     "1h",
			},
			"exclude_noise": map[string]interface{}{
				"type":        "boolean",
				"description": "Exclude common noise patterns (health checks, metrics endpoints, 200 OK). Default: true",
				"default":     true,
			},
			"custom_exclusions": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Additional patterns to exclude from results (e.g., ['debug', 'test-service'])",
			},
			"application": map[string]interface{}{
				"type":        "string",
				"description": "Optional: Focus on a specific application (leave empty for system-wide scan)",
			},
			"min_count": map[string]interface{}{
				"type":        "integer",
				"description": "Minimum occurrence count to include in results (default: 1, increase to filter noise)",
				"default":     1,
				"minimum":     1,
			},
			"top_n": map[string]interface{}{
				"type":        "integer",
				"description": "Number of top results to return (default: 20)",
				"default":     20,
				"minimum":     5,
				"maximum":     100,
			},
			"tier": map[string]interface{}{
				"type":        "string",
				"description": "Log tier to query. If not specified, uses the default from your TCO policies (fetched at session start). archive (COS/cold storage - logs always land here unless excluded), frequent_search (Priority Insights).",
				"enum":        []string{"archive", "frequent_search"},
			},
		},
		"examples": []interface{}{
			map[string]interface{}{
				"mode":          "error_hotspots",
				"time_range":    "1h",
				"exclude_noise": true,
			},
			map[string]interface{}{
				"mode":              "top_error_messages",
				"time_range":        "6h",
				"custom_exclusions": []string{"expected error", "rate limit"},
				"min_count":         5,
			},
			map[string]interface{}{
				"mode":        "severity_distribution",
				"application": "api-gateway",
				"time_range":  "24h",
			},
		},
	}
}

// Execute runs the scout discovery query
func (t *ScoutLogsTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Parse parameters
	mode, _ := GetStringParam(args, "mode", false)
	if mode == "" {
		mode = "error_hotspots"
	}

	timeRange, _ := GetStringParam(args, "time_range", false)
	if timeRange == "" {
		timeRange = "1h"
	}

	excludeNoise := true
	if val, ok := args["exclude_noise"].(bool); ok {
		excludeNoise = val
	}

	var customExclusions []string
	if excl, ok := args["custom_exclusions"].([]interface{}); ok {
		for _, e := range excl {
			if s, ok := e.(string); ok {
				customExclusions = append(customExclusions, s)
			}
		}
	}

	application, _ := GetStringParam(args, "application", false)

	minCount := 1
	if val, ok := args["min_count"].(float64); ok {
		minCount = int(val)
	}

	topN := 20
	if val, ok := args["top_n"].(float64); ok {
		topN = int(val)
	}

	// Tier with default from session TCO config
	// If user doesn't specify a tier, use the session's recommended default
	tier, _ := GetStringParam(args, "tier", false)
	if tier == "" {
		// Get default tier from session TCO config (fetched at session start)
		session := GetSessionFromContext(ctx)
		if session != nil {
			// If application is specified, check for application-specific tier routing
			if application != "" {
				tier = session.GetTierForApplication(application)
			} else {
				tier = session.GetDefaultTier()
			}
		}
		// Final fallback: frequent_search (faster queries when logs go to both tiers)
		if tier == "" {
			tier = "frequent_search"
		}
	} else {
		tier = normalizeTier(tier)
	}

	// Build the discovery query
	query := t.buildDiscoveryQuery(mode, excludeNoise, customExclusions, application, minCount, topN)

	// Calculate time range
	startDate, endDate := calculateTimeRange(timeRange)

	// Execute the query
	apiClient, err := t.GetClient(ctx)
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Failed to get API client: %v", err)), nil
	}

	queryReq := &client.Request{
		Method: "POST",
		Path:   "/v1/query",
		Body: map[string]interface{}{
			"query": map[string]interface{}{
				"logs": map[string]interface{}{
					"dataprime": map[string]interface{}{
						"query": query,
					},
				},
			},
			"metadata": map[string]interface{}{
				"tier":          tier,
				"syntax":        "dataprime",
				"startDate":     startDate,
				"endDate":       endDate,
				"defaultSource": "logs",
			},
		},
		AcceptSSE: true,
		Timeout:   DefaultQueryTimeout,
	}

	resp, err := apiClient.Do(ctx, queryReq)
	if err != nil {
		return NewToolResultErrorWithSuggestion(
			fmt.Sprintf("Scout query failed: %v", err),
			"Try a shorter time range or check if the service is available",
		), nil
	}

	if resp.StatusCode >= 400 {
		return NewToolResultError(fmt.Sprintf("Scout query failed (HTTP %d): %s", resp.StatusCode, string(resp.Body))), nil
	}

	// Parse and format results
	result := t.formatScoutResults(mode, query, resp.Body, timeRange)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil
}

// buildDiscoveryQuery constructs the appropriate aggregation query
func (t *ScoutLogsTool) buildDiscoveryQuery(mode string, excludeNoise bool, customExclusions []string, application string, minCount, topN int) string {
	var filters []string
	var query string

	// Add application filter if specified
	if application != "" {
		filters = append(filters, fmt.Sprintf("$l.applicationname == '%s'", application))
	}

	// Add noise exclusion filters
	if excludeNoise {
		noiseFilters := buildNoiseExclusionFilter(defaultNoisePatterns)
		if noiseFilters != "" {
			filters = append(filters, noiseFilters)
		}
	}

	// Add custom exclusions
	if len(customExclusions) > 0 {
		customFilter := buildNoiseExclusionFilter(customExclusions)
		if customFilter != "" {
			filters = append(filters, customFilter)
		}
	}

	filterClause := ""
	if len(filters) > 0 {
		filterClause = "| filter " + strings.Join(filters, " && ")
	}

	switch mode {
	case "error_hotspots":
		// Find which services have the most errors
		query = fmt.Sprintf(`source logs %s | filter $m.severity >= ERROR | groupby $l.applicationname, $l.subsystemname calculate count() as error_count | filter error_count >= %d | orderby error_count desc | limit %d`,
			filterClause, minCount, topN)

	case "severity_distribution":
		// Severity breakdown by service
		query = fmt.Sprintf(`source logs %s | groupby $l.applicationname, $m.severity calculate count() as log_count | orderby $l.applicationname, -log_count | limit %d`,
			filterClause, topN*5) // More results for distribution

	case "top_error_messages":
		// Group similar error messages
		query = fmt.Sprintf(`source logs %s | filter $m.severity >= ERROR | groupby $d.message:string calculate count() as occurrences, min($m.timestamp) as first_seen, max($m.timestamp) as last_seen | filter occurrences >= %d | orderby occurrences desc | limit %d`,
			filterClause, minCount, topN)

	case "anomaly_scan":
		// Find services with high error rates
		query = fmt.Sprintf(`source logs %s | groupby $l.applicationname calculate count() as total, countif($m.severity >= ERROR) as errors, countif($m.severity == WARNING) as warnings | create error_rate = errors * 100.0 / total | filter error_rate > 1 || errors >= %d | orderby error_rate desc | limit %d`,
			filterClause, minCount, topN)

	case "traffic_overview":
		// Traffic volume by service
		query = fmt.Sprintf(`source logs %s | groupby $l.applicationname calculate count() as log_volume, countdistinct($l.subsystemname) as subsystems | orderby log_volume desc | limit %d`,
			filterClause, topN)

	case "recent_deployments":
		// Detect potential deployments (service restarts, version changes)
		query = fmt.Sprintf(`source logs %s | filter $d.message:string.contains('start') || $d.message:string.contains('deploy') || $d.message:string.contains('version') || $d.message:string.contains('initializ') | groupby $l.applicationname calculate count() as events, min($m.timestamp) as earliest, max($m.timestamp) as latest | filter events >= %d | orderby latest desc | limit %d`,
			filterClause, minCount, topN)

	default:
		// Default to error_hotspots
		query = fmt.Sprintf(`source logs %s | filter $m.severity >= ERROR | groupby $l.applicationname, $l.subsystemname calculate count() as error_count | filter error_count >= %d | orderby error_count desc | limit %d`,
			filterClause, minCount, topN)
	}

	return query
}

// buildNoiseExclusionFilter creates a filter clause to exclude noise patterns
func buildNoiseExclusionFilter(patterns []string) string {
	if len(patterns) == 0 {
		return ""
	}

	var conditions []string
	for _, pattern := range patterns {
		// Use case-insensitive contains for noise filtering
		conditions = append(conditions, fmt.Sprintf("!$d.message:string.toLowerCase().contains('%s')", strings.ToLower(pattern)))
	}

	return "(" + strings.Join(conditions, " && ") + ")"
}

// calculateTimeRange converts a time range string to start/end dates
func calculateTimeRange(timeRange string) (string, string) {
	now := time.Now().UTC()
	var duration time.Duration

	switch timeRange {
	case "15m":
		duration = 15 * time.Minute
	case "1h":
		duration = 1 * time.Hour
	case "6h":
		duration = 6 * time.Hour
	case "24h":
		duration = 24 * time.Hour
	default:
		duration = 1 * time.Hour
	}

	startTime := now.Add(-duration)
	return startTime.Format(time.RFC3339), now.Format(time.RFC3339)
}

// formatScoutResults formats the query results for display
func (t *ScoutLogsTool) formatScoutResults(mode, query string, responseBody []byte, timeRange string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Scout Discovery: %s\n", formatModeName(mode)))
	sb.WriteString(fmt.Sprintf("**Time Range:** Last %s\n\n", timeRange))

	// Parse SSE response
	events := parseSSEResponse(responseBody)
	if events == nil {
		// Try direct JSON parse
		var directResult map[string]interface{}
		if err := json.Unmarshal(responseBody, &directResult); err == nil {
			events = directResult
		}
	}

	if events == nil {
		sb.WriteString("No results found. This could mean:\n")
		sb.WriteString("- No matching logs in the time range\n")
		sb.WriteString("- All logs were filtered by noise exclusion\n")
		sb.WriteString("- Try expanding the time range or disabling noise filtering\n")
		return sb.String()
	}

	// Format the results
	if eventList, ok := events["events"].([]interface{}); ok && len(eventList) > 0 {
		sb.WriteString(fmt.Sprintf("### Found %d pattern(s)\n\n", len(eventList)))

		for i, event := range eventList {
			if i >= 20 { // Safety limit
				sb.WriteString(fmt.Sprintf("\n... and %d more results\n", len(eventList)-20))
				break
			}

			if eventMap, ok := event.(map[string]interface{}); ok {
				sb.WriteString(t.formatEventRow(mode, eventMap, i+1))
			}
		}
	} else {
		sb.WriteString("No significant patterns found in the specified time range.\n")
	}

	// Add drill-down suggestions
	sb.WriteString("\n### Next Steps\n")
	sb.WriteString("Use `query_logs` to drill down into specific findings:\n")
	sb.WriteString("```dataprime\n")
	sb.WriteString("source logs | filter $l.applicationname == '<app_from_above>' && $m.severity >= ERROR | limit 50\n")
	sb.WriteString("```\n")

	// Include the generated query for transparency
	sb.WriteString("\n<details>\n<summary>Generated Scout Query</summary>\n\n")
	sb.WriteString("```dataprime\n")
	sb.WriteString(query)
	sb.WriteString("\n```\n</details>\n")

	return sb.String()
}

// formatModeName converts mode to display name
func formatModeName(mode string) string {
	names := map[string]string{
		"error_hotspots":        "Error Hotspots",
		"severity_distribution": "Severity Distribution",
		"top_error_messages":    "Top Error Messages",
		"anomaly_scan":          "Anomaly Scan",
		"traffic_overview":      "Traffic Overview",
		"recent_deployments":    "Recent Deployments",
	}
	if name, ok := names[mode]; ok {
		return name
	}
	return mode
}

// formatEventRow formats a single result row based on mode
func (t *ScoutLogsTool) formatEventRow(mode string, event map[string]interface{}, index int) string {
	var sb strings.Builder

	switch mode {
	case "error_hotspots":
		app := getStringFromMap(event, "$l.applicationname", "unknown")
		subsystem := getStringFromMap(event, "$l.subsystemname", "")
		count := getNumberFromMap(event, "error_count", 0)

		sb.WriteString(fmt.Sprintf("%d. **%s**", index, app))
		if subsystem != "" {
			sb.WriteString(fmt.Sprintf(" / %s", subsystem))
		}
		sb.WriteString(fmt.Sprintf(" - %d errors\n", int(count)))

	case "severity_distribution":
		app := getStringFromMap(event, "$l.applicationname", "unknown")
		severity := getStringFromMap(event, "$m.severity", "")
		count := getNumberFromMap(event, "log_count", 0)
		sb.WriteString(fmt.Sprintf("%d. **%s** [%s]: %d logs\n", index, app, severity, int(count)))

	case "top_error_messages":
		msg := getStringFromMap(event, "$d.message", "")
		if len(msg) > 100 {
			msg = msg[:100] + "..."
		}
		count := getNumberFromMap(event, "occurrences", 0)
		sb.WriteString(fmt.Sprintf("%d. (%d occurrences) `%s`\n", index, int(count), msg))

	case "anomaly_scan":
		app := getStringFromMap(event, "$l.applicationname", "unknown")
		errorRate := getNumberFromMap(event, "error_rate", 0)
		errors := getNumberFromMap(event, "errors", 0)
		total := getNumberFromMap(event, "total", 0)
		sb.WriteString(fmt.Sprintf("%d. **%s** - %.1f%% error rate (%d/%d)\n", index, app, errorRate, int(errors), int(total)))

	case "traffic_overview":
		app := getStringFromMap(event, "$l.applicationname", "unknown")
		volume := getNumberFromMap(event, "log_volume", 0)
		subsystems := getNumberFromMap(event, "subsystems", 0)
		sb.WriteString(fmt.Sprintf("%d. **%s** - %d logs, %d subsystems\n", index, app, int(volume), int(subsystems)))

	case "recent_deployments":
		app := getStringFromMap(event, "$l.applicationname", "unknown")
		events := getNumberFromMap(event, "events", 0)
		sb.WriteString(fmt.Sprintf("%d. **%s** - %d deployment-related events\n", index, app, int(events)))

	default:
		// Generic format
		jsonBytes, _ := json.Marshal(event)
		sb.WriteString(fmt.Sprintf("%d. %s\n", index, string(jsonBytes)))
	}

	return sb.String()
}

// Helper functions
func getStringFromMap(m map[string]interface{}, key, defaultVal string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultVal
}

func getNumberFromMap(m map[string]interface{}, key string, defaultVal float64) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	if val, ok := m[key].(int); ok {
		return float64(val)
	}
	return defaultVal
}
