package tools

import (
	"fmt"
	"strings"
)

// ProactiveSuggestion represents a contextual next-step suggestion
type ProactiveSuggestion struct {
	Tool        string // Suggested tool to use
	Description string // Why this suggestion is relevant
	Condition   string // When this suggestion applies (for documentation)
}

// SmartSuggestion represents an enhanced suggestion with pre-filled parameters and urgency
type SmartSuggestion struct {
	Tool          string                 `json:"tool"`               // Suggested tool to use
	Description   string                 `json:"description"`        // Why this suggestion is relevant
	Urgency       string                 `json:"urgency"`            // "info", "warning", "critical"
	PrefilledArgs map[string]interface{} `json:"prefilled_args"`     // Pre-populated parameters
	Reason        string                 `json:"reason"`             // Detailed reason for this suggestion
	Confidence    float64                `json:"confidence"`         // 0.0-1.0 confidence score
	Evidence      []string               `json:"evidence,omitempty"` // Supporting evidence for this suggestion
}

// GetSmartSuggestions generates context-aware suggestions with pre-filled arguments
func GetSmartSuggestions(toolName string, result map[string]interface{}, hasError bool) []SmartSuggestion {
	suggestions := []SmartSuggestion{}

	if hasError {
		return getSmartErrorSuggestions(toolName, result)
	}

	switch toolName {
	case "query_logs":
		suggestions = append(suggestions, getQueryResultSuggestions(result)...)
	case "list_alerts":
		suggestions = append(suggestions, getAlertListSuggestions(result)...)
	case "get_alert":
		suggestions = append(suggestions, getAlertDetailSuggestions(result)...)
	case "list_dashboards":
		suggestions = append(suggestions, getDashboardListSuggestions(result)...)
	case "investigate_incident":
		suggestions = append(suggestions, getIncidentSuggestions(result)...)
	case "health_check":
		suggestions = append(suggestions, getHealthCheckSuggestions(result)...)
	}

	return suggestions
}

// getQueryResultSuggestions analyzes query results and provides smart suggestions
func getQueryResultSuggestions(result map[string]interface{}) []SmartSuggestion {
	suggestions := []SmartSuggestion{}

	events, ok := result["events"].([]interface{})
	if !ok {
		return suggestions
	}

	// Analyze severity distribution
	errorCount := 0
	criticalCount := 0
	topApp := ""
	appCounts := make(map[string]int)

	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			// Count severities
			if sev, ok := eventMap["severity"].(float64); ok {
				if int(sev) == 5 {
					errorCount++
				} else if int(sev) == 6 {
					criticalCount++
				}
			}
			// Track applications
			if app, ok := eventMap["applicationname"].(string); ok && app != "" {
				appCounts[app]++
				if topApp == "" || appCounts[app] > appCounts[topApp] {
					topApp = app
				}
			}
		}
	}

	errorRate := float64(errorCount) * 100 / float64(max(len(events), 1))
	criticalRate := float64(criticalCount) * 100 / float64(max(len(events), 1))

	// Critical errors - urgent
	if criticalCount > 0 {
		// Confidence based on critical count and consistency
		confidence := calculateConfidence(criticalCount, len(events), []string{"critical_errors"})
		suggestions = append(suggestions, SmartSuggestion{
			Tool:        "create_alert",
			Description: fmt.Sprintf("Create alert for %d critical errors detected", criticalCount),
			Urgency:     "critical",
			PrefilledArgs: map[string]interface{}{
				"severity_filter": "critical",
				"application":     topApp,
			},
			Reason:     "Critical errors require immediate attention and monitoring",
			Confidence: confidence,
			Evidence: []string{
				fmt.Sprintf("%d critical severity logs found", criticalCount),
				fmt.Sprintf("%.1f%% of results are critical", criticalRate),
				fmt.Sprintf("Top affected application: %s", topApp),
			},
		})
	}

	// High error rate
	if len(events) > 10 && errorCount > len(events)/5 {
		confidence := calculateConfidence(errorCount, len(events), []string{"high_error_rate"})
		suggestions = append(suggestions, SmartSuggestion{
			Tool:        "investigate_incident",
			Description: "High error rate detected - investigate root cause",
			Urgency:     "warning",
			PrefilledArgs: map[string]interface{}{
				"application": topApp,
				"severity":    "error",
			},
			Reason:     fmt.Sprintf("%.0f%% error rate detected", errorRate),
			Confidence: confidence,
			Evidence: []string{
				fmt.Sprintf("%d errors out of %d total events", errorCount, len(events)),
				fmt.Sprintf("Error rate: %.1f%% (threshold: 20%%)", errorRate),
				fmt.Sprintf("Primary application: %s (%d occurrences)", topApp, appCounts[topApp]),
			},
		})
	}

	// Large result set - suggest dashboard
	if len(events) >= MaxSSEEvents {
		confidence := 0.75 // Moderate confidence for dashboards
		suggestions = append(suggestions, SmartSuggestion{
			Tool:        "create_dashboard",
			Description: "Create dashboard to visualize high-volume data",
			Urgency:     "info",
			PrefilledArgs: map[string]interface{}{
				"application_filter": topApp,
			},
			Reason:     "Large result sets are better analyzed through dashboards",
			Confidence: confidence,
			Evidence: []string{
				fmt.Sprintf("Result set hit maximum limit (%d events)", MaxSSEEvents),
				"Visual analysis recommended for patterns",
			},
		})
	}

	// Session-aware suggestions
	session := GetSession()
	if session.GetInvestigation() == nil && (errorCount > 5 || criticalCount > 0) {
		// Suggest starting investigation if not already in one
		suggestions = append(suggestions, SmartSuggestion{
			Tool:        "investigate_incident",
			Description: "Start structured investigation for these errors",
			Urgency:     "info",
			PrefilledArgs: map[string]interface{}{
				"application": topApp,
			},
			Reason:     "Structured investigation helps track findings and root cause",
			Confidence: 0.7,
			Evidence: []string{
				"No active investigation in session",
				"Multiple errors detected suggest systematic issue",
			},
		})
	}

	// Record to session for future context
	session.SetLastQuery(fmt.Sprintf("query returned %d events, %d errors, %d critical", len(events), errorCount, criticalCount))
	if topApp != "" {
		session.SetFilter("last_queried_app", topApp)
	}

	return suggestions
}

// calculateConfidence computes a confidence score based on evidence strength
func calculateConfidence(count, total int, factors []string) float64 {
	// Base confidence from ratio
	ratio := float64(count) / float64(max(total, 1))

	// Higher ratio = higher confidence
	baseConfidence := 0.5 + (ratio * 0.4)

	// Adjust based on sample size
	if total < 10 {
		baseConfidence *= 0.7 // Lower confidence for small samples
	} else if total > 100 {
		baseConfidence *= 1.1 // Higher confidence for large samples
	}

	// Adjust based on factors
	for _, factor := range factors {
		switch factor {
		case "critical_errors":
			baseConfidence += 0.1 // Critical errors are high signal
		case "high_error_rate":
			baseConfidence += 0.05
		}
	}

	// Clamp to [0.0, 1.0]
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}
	if baseConfidence < 0.0 {
		baseConfidence = 0.0
	}

	return baseConfidence
}

// getAlertListSuggestions analyzes alerts and suggests actions
func getAlertListSuggestions(result map[string]interface{}) []SmartSuggestion {
	suggestions := []SmartSuggestion{}

	alerts, ok := result["alerts"].([]interface{})
	if !ok {
		if alertsMap, ok := result["alerts"].(map[string]interface{}); ok {
			// Check if there are triggered alerts
			if triggered, ok := alertsMap["triggered"].([]interface{}); ok && len(triggered) > 0 {
				suggestions = append(suggestions, SmartSuggestion{
					Tool:        "investigate_incident",
					Description: fmt.Sprintf("%d triggered alerts require investigation", len(triggered)),
					Urgency:     "critical",
					Reason:      "Triggered alerts indicate active issues",
				})
			}
		}
		return suggestions
	}

	// No alerts configured
	if len(alerts) == 0 {
		suggestions = append(suggestions, SmartSuggestion{
			Tool:        "suggest_alert",
			Description: "Set up alerting - no alerts configured",
			Urgency:     "warning",
			Reason:      "Proactive monitoring requires alerting configuration",
		})
	}

	return suggestions
}

// getAlertDetailSuggestions provides suggestions based on alert details
func getAlertDetailSuggestions(result map[string]interface{}) []SmartSuggestion {
	suggestions := []SmartSuggestion{}

	// Check alert status
	if status, ok := result["status"].(string); ok {
		if status == "triggered" || status == "firing" {
			if alertID, ok := result["id"].(string); ok {
				suggestions = append(suggestions, SmartSuggestion{
					Tool:        "query_logs",
					Description: "Query logs for triggered alert context",
					Urgency:     "critical",
					PrefilledArgs: map[string]interface{}{
						"alert_id": alertID,
					},
					Reason: "Alert is currently firing - investigate immediately",
				})
			}
		}
	}

	return suggestions
}

// getDashboardListSuggestions provides suggestions based on dashboard list
func getDashboardListSuggestions(result map[string]interface{}) []SmartSuggestion {
	suggestions := []SmartSuggestion{}

	dashboards, ok := result["dashboards"].([]interface{})
	if !ok || len(dashboards) == 0 {
		suggestions = append(suggestions, SmartSuggestion{
			Tool:        "create_dashboard",
			Description: "Create your first dashboard for log visualization",
			Urgency:     "info",
			Reason:      "Dashboards provide visual insights into log patterns",
		})
	}

	return suggestions
}

// getIncidentSuggestions provides follow-up suggestions after incident investigation
func getIncidentSuggestions(result map[string]interface{}) []SmartSuggestion {
	suggestions := []SmartSuggestion{}

	// Check if root cause was identified
	if hypothesis, ok := result["hypothesis"].([]interface{}); ok && len(hypothesis) > 0 {
		suggestions = append(suggestions, SmartSuggestion{
			Tool:        "create_alert",
			Description: "Create alert for identified error pattern",
			Urgency:     "warning",
			Reason:      "Prevent similar incidents through proactive alerting",
		})
	}

	// Check error patterns
	if patterns, ok := result["error_patterns"].([]interface{}); ok && len(patterns) > 0 {
		suggestions = append(suggestions, SmartSuggestion{
			Tool:        "create_dashboard",
			Description: "Create monitoring dashboard for error patterns",
			Urgency:     "info",
			Reason:      "Track error patterns over time",
		})
	}

	return suggestions
}

// getHealthCheckSuggestions provides suggestions based on health check results
func getHealthCheckSuggestions(result map[string]interface{}) []SmartSuggestion {
	suggestions := []SmartSuggestion{}

	// Check overall health status
	if status, ok := result["status"].(string); ok {
		switch status {
		case "critical", "unhealthy":
			suggestions = append(suggestions, SmartSuggestion{
				Tool:        "investigate_incident",
				Description: "System health is critical - investigate immediately",
				Urgency:     "critical",
				Reason:      "Health check detected critical issues",
			})
		case "warning", "degraded":
			suggestions = append(suggestions, SmartSuggestion{
				Tool:        "query_logs",
				Description: "Query recent errors to understand degraded state",
				Urgency:     "warning",
				PrefilledArgs: map[string]interface{}{
					"severity": "error",
				},
				Reason: "System is showing warning signs",
			})
		}
	}

	return suggestions
}

// getSmartErrorSuggestions provides suggestions for error recovery
func getSmartErrorSuggestions(toolName string, result map[string]interface{}) []SmartSuggestion {
	suggestions := []SmartSuggestion{}

	// Extract error message if available
	errorMsg := ""
	if err, ok := result["error"].(string); ok {
		errorMsg = err
	}

	switch toolName {
	case "query_logs":
		suggestions = append(suggestions, SmartSuggestion{
			Tool:        "validate_query",
			Description: "Validate query syntax before retrying",
			Urgency:     "info",
			Reason:      "Query syntax errors are common - validation helps",
		})
		if strings.Contains(errorMsg, "timeout") {
			suggestions = append(suggestions, SmartSuggestion{
				Tool:        "submit_background_query",
				Description: "Use background query for large/slow queries",
				Urgency:     "info",
				Reason:      "Query timeout suggests a large result set",
			})
		}
	case "create_alert":
		suggestions = append(suggestions, SmartSuggestion{
			Tool:        "list_alert_definitions",
			Description: "Check existing alert definitions for reference",
			Urgency:     "info",
			Reason:      "Understanding existing alerts helps configuration",
		})
	case "create_dashboard":
		suggestions = append(suggestions, SmartSuggestion{
			Tool:        "list_dashboards",
			Description: "Review existing dashboards for examples",
			Urgency:     "info",
			Reason:      "Existing dashboards provide configuration examples",
		})
	}

	return suggestions
}

// FormatSmartSuggestions formats smart suggestions as markdown with urgency indicators
func FormatSmartSuggestions(suggestions []SmartSuggestion) string {
	if len(suggestions) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("\n\n---\n💡 **Smart Suggestions:**\n")

	for _, s := range suggestions {
		urgencyIcon := "ℹ️"
		switch s.Urgency {
		case "warning":
			urgencyIcon = "⚠️"
		case "critical":
			urgencyIcon = "🚨"
		}

		// Format confidence as percentage with indicator
		confidenceStr := ""
		if s.Confidence > 0 {
			confidenceIcon := "🔵" // Low
			if s.Confidence >= 0.7 {
				confidenceIcon = "🟢" // High
			} else if s.Confidence >= 0.5 {
				confidenceIcon = "🟡" // Medium
			}
			confidenceStr = fmt.Sprintf(" %s %.0f%% confidence", confidenceIcon, s.Confidence*100)
		}

		builder.WriteString(fmt.Sprintf("\n%s **%s** `%s`%s\n", urgencyIcon, strings.ToUpper(s.Urgency), s.Tool, confidenceStr))
		builder.WriteString(fmt.Sprintf("   %s\n", s.Description))
		if s.Reason != "" {
			builder.WriteString(fmt.Sprintf("   _Reason: %s_\n", s.Reason))
		}

		// Show evidence if available
		if len(s.Evidence) > 0 {
			builder.WriteString("   📋 Evidence:\n")
			for _, e := range s.Evidence {
				builder.WriteString(fmt.Sprintf("      • %s\n", e))
			}
		}

		if len(s.PrefilledArgs) > 0 {
			builder.WriteString("   🔧 Pre-filled args: ")
			args := []string{}
			for k, v := range s.PrefilledArgs {
				args = append(args, fmt.Sprintf("%s=%v", k, v))
			}
			builder.WriteString(strings.Join(args, ", "))
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// GetProactiveSuggestions returns contextual next-step suggestions based on tool results.
// This helps LLMs understand logical next actions after using a tool.
func GetProactiveSuggestions(toolName string, result map[string]interface{}, hasError bool) []ProactiveSuggestion {
	if hasError {
		return getErrorRecoverySuggestions(toolName)
	}
	return getSuccessSuggestions(toolName, result)
}

// getErrorRecoverySuggestions returns suggestions for recovering from errors
func getErrorRecoverySuggestions(toolName string) []ProactiveSuggestion {
	switch toolName {
	case "query_logs":
		return []ProactiveSuggestion{
			{Tool: "build_query", Description: "Use the query builder to construct a valid query from structured parameters"},
			{Tool: "list_dashboards", Description: "Check existing dashboards for working query examples"},
		}
	case "create_dashboard":
		return []ProactiveSuggestion{
			{Tool: "query_logs", Description: "Test queries individually before adding to dashboard"},
			{Tool: "get_dashboard", Description: "Get an existing dashboard to use as a template"},
		}
	case "create_alert":
		return []ProactiveSuggestion{
			{Tool: "list_alerts", Description: "Review existing alerts for configuration examples"},
			{Tool: "query_logs", Description: "Test the alert condition query first"},
		}
	default:
		return nil
	}
}

// staticSuggestions maps tool names to their static suggestions.
// This data-driven approach reduces cyclomatic complexity.
var staticSuggestions = map[string][]ProactiveSuggestion{
	// Query tools
	"submit_background_query": {{Tool: "get_background_query_status", Description: "Check query progress"}},
	"cancel_background_query": {{Tool: "submit_background_query", Description: "Submit a new background query"}},

	// Dashboard tools
	"list_dashboards":       {{Tool: "get_dashboard", Description: "Get details of a specific dashboard"}, {Tool: "create_dashboard", Description: "Create a new dashboard"}},
	"get_dashboard":         {{Tool: "update_dashboard", Description: "Modify this dashboard"}, {Tool: "pin_dashboard", Description: "Pin this dashboard for quick access"}, {Tool: "move_dashboard_to_folder", Description: "Organize into a folder"}},
	"delete_dashboard":      {{Tool: "list_dashboards", Description: "View remaining dashboards"}},
	"pin_dashboard":         {{Tool: "get_dashboard", Description: "View the pinned dashboard"}, {Tool: "list_dashboards", Description: "View all dashboards"}},
	"unpin_dashboard":       {{Tool: "list_dashboards", Description: "View all dashboards"}},
	"set_default_dashboard": {{Tool: "get_dashboard", Description: "View the default dashboard"}, {Tool: "list_dashboards", Description: "View all dashboards"}},

	// Dashboard folder tools
	"list_dashboard_folders":   {{Tool: "create_dashboard_folder", Description: "Create a new folder"}, {Tool: "move_dashboard_to_folder", Description: "Organize dashboards into folders"}},
	"get_dashboard_folder":     {{Tool: "update_dashboard_folder", Description: "Modify this folder"}, {Tool: "move_dashboard_to_folder", Description: "Move dashboards into this folder"}},
	"create_dashboard_folder":  {{Tool: "move_dashboard_to_folder", Description: "Move dashboards into the new folder"}, {Tool: "list_dashboard_folders", Description: "View all folders"}},
	"move_dashboard_to_folder": {{Tool: "get_dashboard", Description: "View the moved dashboard"}, {Tool: "list_dashboard_folders", Description: "View all folders"}},

	// Alert tools
	"list_alerts":  {{Tool: "get_alert", Description: "Get details of a specific alert"}, {Tool: "create_alert", Description: "Create a new alert"}},
	"get_alert":    {{Tool: "update_alert", Description: "Modify this alert configuration"}, {Tool: "activate_alert", Description: "Activate/deactivate this alert"}},
	"create_alert": {{Tool: "list_alerts", Description: "View all alerts including the new one"}, {Tool: "query_logs", Description: "Test the alert condition with a query"}},
	"update_alert": {{Tool: "get_alert", Description: "View the updated alert"}, {Tool: "list_alerts", Description: "View all alerts"}},
	"delete_alert": {{Tool: "list_alerts", Description: "View remaining alerts"}},

	// Alert definition tools
	"list_alert_definitions":  {{Tool: "get_alert_definition", Description: "Get details of a specific alert definition"}, {Tool: "create_alert_definition", Description: "Create a new alert definition"}},
	"get_alert_definition":    {{Tool: "update_alert_definition", Description: "Modify this alert definition"}, {Tool: "create_alert", Description: "Create an alert using this definition"}},
	"create_alert_definition": {{Tool: "create_alert", Description: "Create an alert using this definition"}, {Tool: "list_alert_definitions", Description: "View all alert definitions"}},

	// Data access policy tools
	"list_data_access_policies": {{Tool: "get_data_access_policy", Description: "Get details of a specific policy"}, {Tool: "create_data_access_policy", Description: "Create a new data access policy"}},

	// Ingestion tools
	"ingest_logs": {{Tool: "query_logs", Description: "Query to verify logs were ingested"}},

	// Stream tools
	"list_streams":  {{Tool: "get_stream", Description: "Get details of a specific stream"}, {Tool: "create_stream", Description: "Create a new stream"}},
	"get_stream":    {{Tool: "update_stream", Description: "Modify this stream configuration"}, {Tool: "delete_stream", Description: "Remove this stream"}},
	"create_stream": {{Tool: "list_streams", Description: "View all streams including the new one"}},

	// Event stream target tools
	"get_event_stream_targets":   {{Tool: "create_event_stream_target", Description: "Create a new event stream target"}},
	"create_event_stream_target": {{Tool: "get_event_stream_targets", Description: "View all event stream targets"}},

	// Data usage tools
	"export_data_usage": {{Tool: "update_data_usage_metrics_export_status", Description: "Enable/disable data usage metrics export"}},

	// Rule group tools
	"list_rule_groups":  {{Tool: "get_rule_group", Description: "Get details of a specific rule group"}, {Tool: "create_rule_group", Description: "Create a new rule group"}},
	"get_rule_group":    {{Tool: "update_rule_group", Description: "Modify this rule group"}, {Tool: "delete_rule_group", Description: "Remove this rule group"}},
	"create_rule_group": {{Tool: "list_rule_groups", Description: "View all rule groups including the new one"}},
	"update_rule_group": {{Tool: "get_rule_group", Description: "View the updated rule group"}},
	"delete_rule_group": {{Tool: "list_rule_groups", Description: "View remaining rule groups"}},

	// Outgoing webhook tools
	"list_outgoing_webhooks":  {{Tool: "get_outgoing_webhook", Description: "Get details of a specific webhook"}, {Tool: "create_outgoing_webhook", Description: "Create a new outgoing webhook"}},
	"get_outgoing_webhook":    {{Tool: "update_outgoing_webhook", Description: "Modify this webhook"}, {Tool: "delete_outgoing_webhook", Description: "Remove this webhook"}},
	"create_outgoing_webhook": {{Tool: "list_outgoing_webhooks", Description: "View all webhooks including the new one"}},
	"update_outgoing_webhook": {{Tool: "get_outgoing_webhook", Description: "View the updated webhook"}},
	"delete_outgoing_webhook": {{Tool: "list_outgoing_webhooks", Description: "View remaining webhooks"}},

	// Policy tools
	"list_policies": {{Tool: "get_policy", Description: "Get details of a specific policy"}, {Tool: "create_policy", Description: "Create a new policy"}},
	"get_policy":    {{Tool: "update_policy", Description: "Modify this policy"}, {Tool: "delete_policy", Description: "Remove this policy"}},
	"create_policy": {{Tool: "list_policies", Description: "View all policies including the new one"}},
	"update_policy": {{Tool: "get_policy", Description: "View the updated policy"}},
	"delete_policy": {{Tool: "list_policies", Description: "View remaining policies"}},

	// E2M tools
	"list_e2m":    {{Tool: "get_e2m", Description: "Get details of a specific E2M mapping"}, {Tool: "create_e2m", Description: "Create a new E2M mapping"}},
	"get_e2m":     {{Tool: "replace_e2m", Description: "Replace this E2M mapping"}, {Tool: "delete_e2m", Description: "Remove this E2M mapping"}},
	"create_e2m":  {{Tool: "list_e2m", Description: "View all E2M mappings including the new one"}},
	"replace_e2m": {{Tool: "get_e2m", Description: "View the replaced E2M mapping"}},
	"delete_e2m":  {{Tool: "list_e2m", Description: "View remaining E2M mappings"}},

	// Data access rule tools
	"list_data_access_rules":  {{Tool: "get_data_access_rule", Description: "Get details of a specific data access rule"}, {Tool: "create_data_access_rule", Description: "Create a new data access rule"}},
	"get_data_access_rule":    {{Tool: "update_data_access_rule", Description: "Modify this data access rule"}, {Tool: "delete_data_access_rule", Description: "Remove this data access rule"}},
	"create_data_access_rule": {{Tool: "list_data_access_rules", Description: "View all data access rules including the new one"}},
	"update_data_access_rule": {{Tool: "get_data_access_rule", Description: "View the updated data access rule"}},
	"delete_data_access_rule": {{Tool: "list_data_access_rules", Description: "View remaining data access rules"}},

	// Enrichment tools
	"list_enrichments":  {{Tool: "create_enrichment", Description: "Create a new enrichment"}},
	"get_enrichments":   {{Tool: "create_enrichment", Description: "Create a new enrichment"}},
	"create_enrichment": {{Tool: "list_enrichments", Description: "View all enrichments including the new one"}},
	"update_enrichment": {{Tool: "list_enrichments", Description: "View all enrichments"}},
	"delete_enrichment": {{Tool: "list_enrichments", Description: "View remaining enrichments"}},

	// View tools
	"list_views":   {{Tool: "get_view", Description: "Get details of a specific view"}, {Tool: "create_view", Description: "Create a new view"}},
	"get_view":     {{Tool: "replace_view", Description: "Replace this view"}, {Tool: "delete_view", Description: "Remove this view"}},
	"create_view":  {{Tool: "list_views", Description: "View all views including the new one"}},
	"replace_view": {{Tool: "get_view", Description: "View the replaced view"}},
	"delete_view":  {{Tool: "list_views", Description: "View remaining views"}},

	// View folder tools
	"list_view_folders":   {{Tool: "get_view_folder", Description: "Get details of a specific view folder"}, {Tool: "create_view_folder", Description: "Create a new view folder"}},
	"get_view_folder":     {{Tool: "replace_view_folder", Description: "Replace this view folder"}, {Tool: "delete_view_folder", Description: "Remove this view folder"}},
	"create_view_folder":  {{Tool: "list_view_folders", Description: "View all view folders including the new one"}},
	"replace_view_folder": {{Tool: "get_view_folder", Description: "View the replaced view folder"}},
	"delete_view_folder":  {{Tool: "list_view_folders", Description: "View remaining view folders"}},
}

// getSuccessSuggestions returns suggestions based on successful tool execution.
// Uses a data-driven approach for static mappings and handles dynamic cases separately.
func getSuccessSuggestions(toolName string, result map[string]interface{}) []ProactiveSuggestion {
	// Handle dynamic suggestions that depend on result content
	switch toolName {
	case "query_logs":
		return getQueryLogsSuggestions(result)
	case "get_background_query_status":
		return getBackgroundQueryStatusSuggestions(result)
	case "create_dashboard", "update_dashboard":
		return getDashboardMutationSuggestions(result)
	}

	// Return static suggestions from the lookup table
	if suggestions, ok := staticSuggestions[toolName]; ok {
		return suggestions
	}
	return nil
}

// getQueryLogsSuggestions returns suggestions based on query_logs results
func getQueryLogsSuggestions(result map[string]interface{}) []ProactiveSuggestion {
	events, ok := result["events"].([]interface{})
	if !ok {
		return nil
	}
	if len(events) == 0 {
		return []ProactiveSuggestion{
			{Tool: "query_logs", Description: "Try expanding the time range or relaxing filters - no results found"},
		}
	}
	if len(events) > 100 {
		return []ProactiveSuggestion{
			{Tool: "create_dashboard", Description: "Create a dashboard to visualize these query results"},
			{Tool: "create_alert", Description: "Set up an alert to monitor this condition"},
		}
	}
	return nil
}

// getBackgroundQueryStatusSuggestions returns suggestions based on background query status
func getBackgroundQueryStatusSuggestions(result map[string]interface{}) []ProactiveSuggestion {
	status, ok := result["status"].(string)
	if !ok {
		return nil
	}
	switch status {
	case "completed", "COMPLETED":
		return []ProactiveSuggestion{
			{Tool: "get_background_query_data", Description: "Retrieve the completed query results"},
		}
	case "running", "RUNNING":
		return []ProactiveSuggestion{
			{Tool: "get_background_query_status", Description: "Check again in a few moments"},
			{Tool: "cancel_background_query", Description: "Cancel if no longer needed"},
		}
	}
	return nil
}

// getDashboardMutationSuggestions returns suggestions after dashboard create/update
func getDashboardMutationSuggestions(result map[string]interface{}) []ProactiveSuggestion {
	if id, ok := result["id"].(string); ok && id != "" {
		return []ProactiveSuggestion{
			{Tool: "get_dashboard", Description: "View the created/updated dashboard"},
			{Tool: "pin_dashboard", Description: "Pin for quick access"},
		}
	}
	return nil
}

// FormatProactiveSuggestions formats suggestions as a markdown string
func FormatProactiveSuggestions(suggestions []ProactiveSuggestion) string {
	if len(suggestions) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("\n\n---\n💡 **Next Steps:**\n")
	for _, s := range suggestions {
		builder.WriteString(fmt.Sprintf("- `%s`: %s\n", s.Tool, s.Description))
	}
	return builder.String()
}
