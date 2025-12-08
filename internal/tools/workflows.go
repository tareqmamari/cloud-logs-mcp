// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file contains compound workflow tools that orchestrate multiple operations.
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

// InvestigateIncidentTool provides a guided incident investigation workflow
type InvestigateIncidentTool struct {
	*BaseTool
}

// NewInvestigateIncidentTool creates a new InvestigateIncidentTool
func NewInvestigateIncidentTool(c *client.Client, l *zap.Logger) *InvestigateIncidentTool {
	return &InvestigateIncidentTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *InvestigateIncidentTool) Name() string { return "investigate_incident" }

// Description returns the tool description
func (t *InvestigateIncidentTool) Description() string {
	return `Comprehensive incident investigation workflow that analyzes logs, checks alerts, and provides root cause suggestions.

**Best for:** Incident response, debugging production issues, understanding error patterns.

**What it does:**
1. Queries recent error logs for the specified application/time range
2. Analyzes error patterns and trends
3. Identifies top error sources and frequencies
4. Provides root cause hypotheses
5. Suggests remediation actions

**Input options:**
- application: Focus on specific application (recommended)
- time_range: How far back to look (default: 1h)
- severity: Minimum severity to investigate (default: error)
- keyword: Additional search term to filter results

**Related tools:** query_logs, list_alerts, get_query_templates, create_alert`
}

// InputSchema returns the input schema
func (t *InvestigateIncidentTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"application": map[string]interface{}{
				"type":        "string",
				"description": "Application name to investigate (recommended for focused analysis)",
			},
			"time_range": map[string]interface{}{
				"type":        "string",
				"description": "Time range to investigate: 15m, 1h, 6h, 24h, 7d (default: 1h)",
				"enum":        []string{"15m", "1h", "6h", "24h", "7d"},
				"default":     "1h",
			},
			"severity": map[string]interface{}{
				"type":        "string",
				"description": "Minimum severity level to investigate",
				"enum":        []string{"warning", "error", "critical"},
				"default":     "error",
			},
			"keyword": map[string]interface{}{
				"type":        "string",
				"description": "Additional keyword to search for in logs",
			},
		},
	}
}

// Execute runs the incident investigation workflow
func (t *InvestigateIncidentTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	application, _ := GetStringParam(args, "application", false)
	timeRange, _ := GetStringParam(args, "time_range", false)
	severity, _ := GetStringParam(args, "severity", false)
	keyword, _ := GetStringParam(args, "keyword", false)

	if timeRange == "" {
		timeRange = "1h"
	}
	if severity == "" {
		severity = "error"
	}

	// Build the investigation query
	var queryParts []string
	queryParts = append(queryParts, "source logs")

	// Add severity filter
	severityValue := 5 // error
	if severity == "warning" {
		severityValue = 4
	} else if severity == "critical" {
		severityValue = 6
	}
	queryParts = append(queryParts, fmt.Sprintf("filter $m.severity >= %d", severityValue))

	// Add application filter if specified
	if application != "" {
		queryParts = append(queryParts, fmt.Sprintf("filter $l.applicationname == '%s'", application))
	}

	// Add keyword filter if specified
	if keyword != "" {
		queryParts = append(queryParts, fmt.Sprintf("filter $d.text.contains('%s') || $d.message.contains('%s')", keyword, keyword))
	}

	// Build final query
	query := strings.Join(queryParts, " | ")

	// Calculate time range
	endDate := time.Now().UTC()
	var startDate time.Time
	switch timeRange {
	case "15m":
		startDate = endDate.Add(-15 * time.Minute)
	case "6h":
		startDate = endDate.Add(-6 * time.Hour)
	case "24h":
		startDate = endDate.Add(-24 * time.Hour)
	case "7d":
		startDate = endDate.Add(-7 * 24 * time.Hour)
	default:
		startDate = endDate.Add(-1 * time.Hour)
	}

	// Execute the query
	req := &client.Request{
		Method: "POST",
		Path:   "/v1/query",
		Body: map[string]interface{}{
			"query":      query,
			"tier":       "archive",
			"syntax":     "dataprime",
			"start_date": startDate.Format(time.RFC3339),
			"end_date":   endDate.Format(time.RFC3339),
		},
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return t.formatInvestigationError(err, query, application)
	}

	// Analyze the results
	return t.formatInvestigationResults(result, query, application, timeRange, severity)
}

// formatInvestigationError formats an error response with helpful suggestions
func (t *InvestigateIncidentTool) formatInvestigationError(err error, query, application string) (*mcp.CallToolResult, error) {
	var response strings.Builder
	response.WriteString("## ❌ Investigation Query Failed\n\n")
	response.WriteString(fmt.Sprintf("**Error:** %s\n\n", err.Error()))
	response.WriteString("### Troubleshooting Steps\n")
	response.WriteString("1. Verify the application name is correct\n")
	response.WriteString("2. Try expanding the time range\n")
	response.WriteString("3. Check if logs exist for this application with: `query_logs`\n\n")
	response.WriteString(fmt.Sprintf("### Query Used\n```\n%s\n```\n", query))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response.String()},
		},
	}, nil
}

// formatInvestigationResults formats the investigation findings
func (t *InvestigateIncidentTool) formatInvestigationResults(result map[string]interface{}, query, application, timeRange, severity string) (*mcp.CallToolResult, error) {
	var response strings.Builder

	response.WriteString("# 🔍 Incident Investigation Report\n\n")

	// Investigation parameters
	response.WriteString("## Parameters\n")
	if application != "" {
		response.WriteString(fmt.Sprintf("- **Application:** %s\n", application))
	} else {
		response.WriteString("- **Application:** All applications\n")
	}
	response.WriteString(fmt.Sprintf("- **Time Range:** Last %s\n", timeRange))
	response.WriteString(fmt.Sprintf("- **Severity:** %s and above\n\n", severity))

	// Analyze results
	events, ok := result["events"].([]interface{})
	if !ok || len(events) == 0 {
		response.WriteString("## ✅ No Issues Found\n\n")
		response.WriteString("No error logs matching the criteria were found in the specified time range.\n\n")
		response.WriteString("### Possible Interpretations\n")
		response.WriteString("- The system is operating normally\n")
		response.WriteString("- Logs may not be reaching Cloud Logs\n")
		response.WriteString("- The application name or filters may be incorrect\n\n")
		response.WriteString("### Suggested Actions\n")
		response.WriteString("- Expand the time range and try again\n")
		response.WriteString("- Verify logging is configured correctly\n")
		response.WriteString("- Check `list_alerts` for any triggered alerts\n")
	} else {
		// Perform analysis
		analysis := AnalyzeQueryResults(result)

		response.WriteString("## 📊 Findings Summary\n\n")
		response.WriteString(fmt.Sprintf("Found **%d error logs** in the specified time range.\n\n", len(events)))

		// Add analysis
		if analysis != nil {
			if analysis.Statistics != nil && analysis.Statistics.ErrorRate > 0 {
				response.WriteString(fmt.Sprintf("- **Error Rate:** %.1f%%\n", analysis.Statistics.ErrorRate))
			}

			// Top applications
			if analysis.Statistics != nil && len(analysis.Statistics.TopValues["applications"]) > 0 {
				response.WriteString("\n### Top Applications with Errors\n")
				for i, app := range analysis.Statistics.TopValues["applications"] {
					if i >= 5 {
						break
					}
					response.WriteString(fmt.Sprintf("- %s: %d errors\n", app.Value, app.Count))
				}
			}

			// Top subsystems
			if analysis.Statistics != nil && len(analysis.Statistics.TopValues["subsystems"]) > 0 {
				response.WriteString("\n### Top Subsystems with Errors\n")
				for i, sub := range analysis.Statistics.TopValues["subsystems"] {
					if i >= 5 {
						break
					}
					response.WriteString(fmt.Sprintf("- %s: %d errors\n", sub.Value, sub.Count))
				}
			}

			// Anomalies
			if len(analysis.Anomalies) > 0 {
				response.WriteString("\n### ⚠️ Anomalies Detected\n")
				for _, a := range analysis.Anomalies {
					response.WriteString(fmt.Sprintf("- **%s:** %s\n", a.Type, a.Description))
				}
			}

			// Trends
			if len(analysis.Trends) > 0 {
				response.WriteString("\n### 📈 Trends\n")
				for _, trend := range analysis.Trends {
					response.WriteString(fmt.Sprintf("- %s\n", trend.Description))
				}
			}
		}

		// Sample error messages
		response.WriteString("\n### Sample Error Messages\n")
		shown := 0
		for _, event := range events {
			if shown >= 5 {
				break
			}
			if eventMap, ok := event.(map[string]interface{}); ok {
				msg := extractErrorMessage(eventMap)
				if msg != "" {
					response.WriteString(fmt.Sprintf("- `%s`\n", truncateString(msg, 100)))
					shown++
				}
			}
		}

		// Root cause hypotheses
		response.WriteString("\n## 🎯 Root Cause Hypotheses\n")
		hypotheses := generateHypotheses(events)
		for i, h := range hypotheses {
			response.WriteString(fmt.Sprintf("%d. %s\n", i+1, h))
		}

		// Recommended actions
		response.WriteString("\n## 📋 Recommended Actions\n")
		response.WriteString("1. **Drill down:** Use `query_logs` with specific filters to examine individual errors\n")
		response.WriteString("2. **Check alerts:** Run `list_alerts` to see if any alerts have triggered\n")
		response.WriteString("3. **Create monitoring:** Use `suggest_alert` to set up alerting for this pattern\n")
		response.WriteString("4. **Build dashboard:** Use `create_dashboard` to visualize error trends\n")
	}

	// Query used
	response.WriteString(fmt.Sprintf("\n---\n### Query Used\n```dataprime\n%s\n```\n", query))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response.String()},
		},
	}, nil
}

// extractErrorMessage extracts the error message from an event
func extractErrorMessage(event map[string]interface{}) string {
	// Try various common field names
	fields := []string{"message", "error", "error_message", "msg", "text"}
	for _, field := range fields {
		if msg, ok := event[field].(string); ok && msg != "" {
			return msg
		}
		// Check in data subfield
		if data, ok := event["data"].(map[string]interface{}); ok {
			if msg, ok := data[field].(string); ok && msg != "" {
				return msg
			}
		}
		// Check in user_data subfield
		if userData, ok := event["user_data"].(map[string]interface{}); ok {
			if msg, ok := userData[field].(string); ok && msg != "" {
				return msg
			}
		}
	}
	return ""
}

// generateHypotheses generates root cause hypotheses based on error patterns
func generateHypotheses(events []interface{}) []string {
	var hypotheses []string

	// Analyze patterns
	errorMessages := make(map[string]int)
	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			msg := extractErrorMessage(eventMap)
			if msg != "" {
				// Normalize message for grouping
				normalized := normalizeErrorMessage(msg)
				errorMessages[normalized]++
			}
		}
	}

	// Generate hypotheses based on common patterns
	for msg, count := range errorMessages {
		if count > len(events)/4 { // More than 25% of errors have this pattern
			if strings.Contains(strings.ToLower(msg), "timeout") {
				hypotheses = append(hypotheses, "Network or service timeout issues - check downstream service health")
			}
			if strings.Contains(strings.ToLower(msg), "connection") {
				hypotheses = append(hypotheses, "Database or service connection issues - verify connection pools and limits")
			}
			if strings.Contains(strings.ToLower(msg), "memory") || strings.Contains(strings.ToLower(msg), "oom") {
				hypotheses = append(hypotheses, "Memory pressure - check container limits and memory leaks")
			}
			if strings.Contains(strings.ToLower(msg), "auth") || strings.Contains(strings.ToLower(msg), "permission") {
				hypotheses = append(hypotheses, "Authentication or authorization failures - verify credentials and permissions")
			}
			if strings.Contains(strings.ToLower(msg), "null") || strings.Contains(strings.ToLower(msg), "undefined") {
				hypotheses = append(hypotheses, "Data integrity issues - check for missing or null values in inputs")
			}
		}
	}

	// Add generic hypotheses if we don't have specific ones
	if len(hypotheses) == 0 {
		hypotheses = append(hypotheses,
			"Recent deployment may have introduced bugs - check recent changes",
			"External dependency failure - verify all downstream services",
			"Infrastructure issue - check CPU, memory, and disk metrics",
		)
	}

	return hypotheses
}

// normalizeErrorMessage normalizes error messages for grouping
func normalizeErrorMessage(msg string) string {
	// Remove numbers, timestamps, IDs for grouping
	// Keep first 50 chars for pattern matching
	msg = strings.ToLower(msg)
	if len(msg) > 50 {
		msg = msg[:50]
	}
	return msg
}

// truncateString truncates a string to max length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// HealthCheckTool provides a quick system health overview
type HealthCheckTool struct {
	*BaseTool
}

// NewHealthCheckTool creates a new HealthCheckTool
func NewHealthCheckTool(c *client.Client, l *zap.Logger) *HealthCheckTool {
	return &HealthCheckTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *HealthCheckTool) Name() string { return "health_check" }

// Description returns the tool description
func (t *HealthCheckTool) Description() string {
	return `Quick system health check that summarizes recent activity, error rates, and potential issues.

**Best for:** Morning health checks, shift handoffs, quick status overview.

**What it does:**
1. Checks recent error rates across all applications
2. Identifies top error sources
3. Verifies log ingestion is working
4. Provides overall health assessment

**Related tools:** investigate_incident, query_logs, list_alerts`
}

// InputSchema returns the input schema
func (t *HealthCheckTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"time_range": map[string]interface{}{
				"type":        "string",
				"description": "Time range to check: 15m, 1h, 6h (default: 1h)",
				"enum":        []string{"15m", "1h", "6h"},
				"default":     "1h",
			},
		},
	}
}

// Execute runs the health check
func (t *HealthCheckTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	timeRange, _ := GetStringParam(args, "time_range", false)
	if timeRange == "" {
		timeRange = "1h"
	}

	// Calculate time range
	endDate := time.Now().UTC()
	var startDate time.Time
	switch timeRange {
	case "15m":
		startDate = endDate.Add(-15 * time.Minute)
	case "6h":
		startDate = endDate.Add(-6 * time.Hour)
	default:
		startDate = endDate.Add(-1 * time.Hour)
	}

	var response strings.Builder
	response.WriteString("# 🏥 System Health Check\n\n")
	response.WriteString(fmt.Sprintf("**Time Range:** Last %s (ending %s UTC)\n\n", timeRange, endDate.Format("15:04")))

	// Query for health summary
	healthQuery := "source logs | groupby $l.applicationname calculate count() as total, countif($m.severity >= 5) as errors, countif($m.severity >= 4) as warnings | create error_rate = errors * 100.0 / total | sortby -error_rate | limit 15"

	req := &client.Request{
		Method: "POST",
		Path:   "/v1/query",
		Body: map[string]interface{}{
			"query":      healthQuery,
			"tier":       "archive",
			"syntax":     "dataprime",
			"start_date": startDate.Format(time.RFC3339),
			"end_date":   endDate.Format(time.RFC3339),
		},
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		response.WriteString("## ⚠️ Health Check Failed\n\n")
		response.WriteString(fmt.Sprintf("Unable to query logs: %s\n\n", err.Error()))
		response.WriteString("This may indicate connectivity issues or misconfiguration.\n")

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: response.String()},
			},
		}, nil
	}

	// Analyze results
	events, ok := result["events"].([]interface{})
	if !ok || len(events) == 0 {
		response.WriteString("## ⚠️ No Data Available\n\n")
		response.WriteString("No logs found in the specified time range.\n\n")
		response.WriteString("### Possible Causes\n")
		response.WriteString("- Log ingestion may not be configured\n")
		response.WriteString("- Time range may be too narrow\n")
		response.WriteString("- No services are currently logging\n")

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: response.String()},
			},
		}, nil
	}

	// Calculate overall health
	totalLogs := 0
	totalErrors := 0
	unhealthyApps := []string{}

	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			total, _ := eventMap["total"].(float64)
			errors, _ := eventMap["errors"].(float64)
			errorRate, _ := eventMap["error_rate"].(float64)
			appName, _ := eventMap["$l.applicationname"].(string)
			if appName == "" {
				appName, _ = eventMap["applicationname"].(string)
			}

			totalLogs += int(total)
			totalErrors += int(errors)

			if errorRate > 5 && appName != "" {
				unhealthyApps = append(unhealthyApps, fmt.Sprintf("%s (%.1f%% errors)", appName, errorRate))
			}
		}
	}

	overallErrorRate := float64(0)
	if totalLogs > 0 {
		overallErrorRate = float64(totalErrors) * 100.0 / float64(totalLogs)
	}

	// Overall status
	status := "✅ Healthy"
	statusDesc := "All systems operating normally"
	if overallErrorRate > 10 {
		status = "🚨 Critical"
		statusDesc = "High error rates detected - immediate attention required"
	} else if overallErrorRate > 5 {
		status = "⚠️ Warning"
		statusDesc = "Elevated error rates - investigation recommended"
	} else if overallErrorRate > 1 {
		status = "ℹ️ Monitor"
		statusDesc = "Some errors present - keep monitoring"
	}

	response.WriteString(fmt.Sprintf("## %s\n\n", status))
	response.WriteString(fmt.Sprintf("%s\n\n", statusDesc))

	// Summary statistics
	response.WriteString("### Summary\n")
	response.WriteString(fmt.Sprintf("- **Total Logs:** %d\n", totalLogs))
	response.WriteString(fmt.Sprintf("- **Total Errors:** %d\n", totalErrors))
	response.WriteString(fmt.Sprintf("- **Overall Error Rate:** %.2f%%\n", overallErrorRate))
	response.WriteString(fmt.Sprintf("- **Applications Monitored:** %d\n\n", len(events)))

	// Unhealthy applications
	if len(unhealthyApps) > 0 {
		response.WriteString("### ⚠️ Applications Needing Attention\n")
		for _, app := range unhealthyApps {
			response.WriteString(fmt.Sprintf("- %s\n", app))
		}
		response.WriteString("\n")
	}

	// Recommendations
	response.WriteString("### Recommended Actions\n")
	if len(unhealthyApps) > 0 {
		response.WriteString("1. Run `investigate_incident` for applications with high error rates\n")
	}
	if overallErrorRate > 1 {
		response.WriteString("2. Check `list_alerts` to see if any alerts have triggered\n")
	}
	response.WriteString("3. Use `query_logs` for detailed investigation of specific issues\n")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response.String()},
		},
	}, nil
}

// ValidateQueryTool validates a query without executing it
type ValidateQueryTool struct {
	*BaseTool
}

// NewValidateQueryTool creates a new ValidateQueryTool
func NewValidateQueryTool(c *client.Client, l *zap.Logger) *ValidateQueryTool {
	return &ValidateQueryTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ValidateQueryTool) Name() string { return "validate_query" }

// Description returns the tool description
func (t *ValidateQueryTool) Description() string {
	return `Validate a DataPrime query for syntax errors without executing it.

**Best for:** Checking queries before use, debugging query syntax, learning DataPrime.

**What it does:**
1. Parses the query for syntax errors
2. Identifies invalid field references
3. Suggests corrections for common mistakes
4. Provides query structure analysis

**Related tools:** query_logs, build_query, explain_query, get_query_templates`
}

// InputSchema returns the input schema
func (t *ValidateQueryTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The DataPrime query to validate",
			},
		},
		"required": []string{"query"},
	}
}

// Execute validates the query
func (t *ValidateQueryTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	query, err := GetStringParam(args, "query", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	validation := validateDataPrimeQuery(query)

	result, err := json.MarshalIndent(validation, "", "  ")
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Failed to format validation: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(result)},
		},
	}, nil
}

// QueryValidation represents the result of query validation
type QueryValidation struct {
	Valid       bool         `json:"valid"`
	Query       string       `json:"query"`
	Errors      []QueryError `json:"errors,omitempty"`
	Warnings    []string     `json:"warnings,omitempty"`
	Structure   []string     `json:"structure"`
	Suggestions []string     `json:"suggestions,omitempty"`
}

// QueryError represents a specific error in the query
type QueryError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Fix     string `json:"fix,omitempty"`
}

// validateDataPrimeQuery validates a DataPrime query
func validateDataPrimeQuery(query string) *QueryValidation {
	validation := &QueryValidation{
		Valid:     true,
		Query:     query,
		Errors:    []QueryError{},
		Warnings:  []string{},
		Structure: []string{},
	}

	// Check for empty query
	if strings.TrimSpace(query) == "" {
		validation.Valid = false
		validation.Errors = append(validation.Errors, QueryError{
			Type:    "empty_query",
			Message: "Query is empty",
			Fix:     "Provide a DataPrime query starting with 'source logs'",
		})
		return validation
	}

	// Parse stages
	stages := splitPipeStages(query)
	for _, stage := range stages {
		stage = strings.TrimSpace(stage)
		if stage != "" {
			component := parseDataPrimeStage(stage)
			validation.Structure = append(validation.Structure, fmt.Sprintf("%s: %s", component.Type, stage))
		}
	}

	// Check for source
	hasSource := false
	for _, stage := range stages {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(stage)), "source") {
			hasSource = true
			break
		}
	}
	if !hasSource {
		validation.Warnings = append(validation.Warnings, "Query does not include 'source' - it will use the default source if set")
	}

	// Check for common syntax errors
	// Double quotes check
	if strings.Contains(query, "== \"") {
		validation.Valid = false
		validation.Errors = append(validation.Errors, QueryError{
			Type:    "syntax_error",
			Message: "Using double quotes instead of single quotes",
			Fix:     "Use single quotes for string values: $l.applicationname == 'myapp'",
		})
	}

	// Single = check - but avoid matching == by checking for = followed by space and quote but not preceded by =
	// Use a simple regex or iterate to check properly
	singleEqualCheck := false
	for i := 0; i < len(query)-2; i++ {
		if query[i] == '=' && query[i+1] == ' ' && query[i+2] == '\'' {
			// Check this is not part of ==
			if i == 0 || query[i-1] != '=' {
				singleEqualCheck = true
				break
			}
		}
	}
	if singleEqualCheck {
		validation.Valid = false
		validation.Errors = append(validation.Errors, QueryError{
			Type:    "syntax_error",
			Message: "Using single = instead of == for comparison",
			Fix:     "Use == for equality comparison: $l.applicationname == 'myapp'",
		})
	}

	// Typo checks
	typoErrors := []struct {
		pattern string
		message string
		fix     string
	}{
		{"applicationame", "Typo in 'applicationname'", "Correct spelling: applicationname (not applicationame)"},
		{"subsytemname", "Typo in 'subsystemname'", "Correct spelling: subsystemname (not subsytemname)"},
		{"serverity", "Typo in 'severity'", "Correct spelling: severity (not serverity)"},
	}

	for _, check := range typoErrors {
		if strings.Contains(strings.ToLower(query), check.pattern) {
			validation.Valid = false
			validation.Errors = append(validation.Errors, QueryError{
				Type:    "syntax_error",
				Message: check.message,
				Fix:     check.fix,
			})
		}
	}

	// Add suggestions
	if !strings.Contains(query, "limit") {
		validation.Suggestions = append(validation.Suggestions, "Consider adding '| limit N' to restrict result size")
	}
	if len(stages) == 1 {
		validation.Suggestions = append(validation.Suggestions, "Consider adding filters to narrow results")
	}

	return validation
}
