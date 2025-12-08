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

// getSuccessSuggestions returns suggestions based on successful tool execution
func getSuccessSuggestions(toolName string, result map[string]interface{}) []ProactiveSuggestion {
	switch toolName {
	// Query tools
	case "query_logs":
		suggestions := []ProactiveSuggestion{}
		if events, ok := result["events"].([]interface{}); ok {
			if len(events) == 0 {
				suggestions = append(suggestions,
					ProactiveSuggestion{Tool: "query_logs", Description: "Try expanding the time range or relaxing filters - no results found"},
				)
			} else if len(events) > 100 {
				suggestions = append(suggestions,
					ProactiveSuggestion{Tool: "create_dashboard", Description: "Create a dashboard to visualize these query results"},
					ProactiveSuggestion{Tool: "create_alert", Description: "Set up an alert to monitor this condition"},
				)
			}
		}
		return suggestions

	case "submit_background_query":
		return []ProactiveSuggestion{
			{Tool: "get_background_query_status", Description: "Check query progress"},
		}

	case "get_background_query_status":
		if status, ok := result["status"].(string); ok {
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
		}
		return nil

	// Dashboard tools
	case "list_dashboards":
		return []ProactiveSuggestion{
			{Tool: "get_dashboard", Description: "Get details of a specific dashboard"},
			{Tool: "create_dashboard", Description: "Create a new dashboard"},
		}

	case "get_dashboard":
		return []ProactiveSuggestion{
			{Tool: "update_dashboard", Description: "Modify this dashboard"},
			{Tool: "pin_dashboard", Description: "Pin this dashboard for quick access"},
			{Tool: "move_dashboard_to_folder", Description: "Organize into a folder"},
		}

	case "create_dashboard", "update_dashboard":
		if id, ok := result["id"].(string); ok && id != "" {
			return []ProactiveSuggestion{
				{Tool: "get_dashboard", Description: "View the created/updated dashboard"},
				{Tool: "pin_dashboard", Description: "Pin for quick access"},
			}
		}
		return nil

	// Alert tools
	case "list_alerts":
		return []ProactiveSuggestion{
			{Tool: "get_alert", Description: "Get details of a specific alert"},
			{Tool: "create_alert", Description: "Create a new alert"},
		}

	case "get_alert":
		return []ProactiveSuggestion{
			{Tool: "update_alert", Description: "Modify this alert configuration"},
			{Tool: "activate_alert", Description: "Activate/deactivate this alert"},
		}

	case "create_alert":
		return []ProactiveSuggestion{
			{Tool: "list_alerts", Description: "View all alerts including the new one"},
			{Tool: "query_logs", Description: "Test the alert condition with a query"},
		}

	// Data access policy tools
	case "list_data_access_policies":
		return []ProactiveSuggestion{
			{Tool: "get_data_access_policy", Description: "Get details of a specific policy"},
			{Tool: "create_data_access_policy", Description: "Create a new data access policy"},
		}

	// Ingestion tools
	case "ingest_logs":
		return []ProactiveSuggestion{
			{Tool: "query_logs", Description: "Query to verify logs were ingested"},
		}

	// Folder tools
	case "list_dashboard_folders":
		return []ProactiveSuggestion{
			{Tool: "create_dashboard_folder", Description: "Create a new folder"},
			{Tool: "move_dashboard_to_folder", Description: "Organize dashboards into folders"},
		}

	case "get_dashboard_folder":
		return []ProactiveSuggestion{
			{Tool: "update_dashboard_folder", Description: "Modify this folder"},
			{Tool: "move_dashboard_to_folder", Description: "Move dashboards into this folder"},
		}

	case "create_dashboard_folder":
		return []ProactiveSuggestion{
			{Tool: "move_dashboard_to_folder", Description: "Move dashboards into the new folder"},
			{Tool: "list_dashboard_folders", Description: "View all folders"},
		}

	case "move_dashboard_to_folder":
		return []ProactiveSuggestion{
			{Tool: "get_dashboard", Description: "View the moved dashboard"},
			{Tool: "list_dashboard_folders", Description: "View all folders"},
		}

	case "pin_dashboard":
		return []ProactiveSuggestion{
			{Tool: "get_dashboard", Description: "View the pinned dashboard"},
			{Tool: "list_dashboards", Description: "View all dashboards"},
		}

	case "unpin_dashboard":
		return []ProactiveSuggestion{
			{Tool: "list_dashboards", Description: "View all dashboards"},
		}

	case "set_default_dashboard":
		return []ProactiveSuggestion{
			{Tool: "get_dashboard", Description: "View the default dashboard"},
			{Tool: "list_dashboards", Description: "View all dashboards"},
		}

	case "delete_dashboard":
		return []ProactiveSuggestion{
			{Tool: "list_dashboards", Description: "View remaining dashboards"},
		}

	case "update_alert":
		return []ProactiveSuggestion{
			{Tool: "get_alert", Description: "View the updated alert"},
			{Tool: "list_alerts", Description: "View all alerts"},
		}

	case "delete_alert":
		return []ProactiveSuggestion{
			{Tool: "list_alerts", Description: "View remaining alerts"},
		}

	case "cancel_background_query":
		return []ProactiveSuggestion{
			{Tool: "submit_background_query", Description: "Submit a new background query"},
		}

	// Alert definition tools
	case "list_alert_definitions":
		return []ProactiveSuggestion{
			{Tool: "get_alert_definition", Description: "Get details of a specific alert definition"},
			{Tool: "create_alert_definition", Description: "Create a new alert definition"},
		}

	case "get_alert_definition":
		return []ProactiveSuggestion{
			{Tool: "update_alert_definition", Description: "Modify this alert definition"},
			{Tool: "create_alert", Description: "Create an alert using this definition"},
		}

	case "create_alert_definition":
		return []ProactiveSuggestion{
			{Tool: "create_alert", Description: "Create an alert using this definition"},
			{Tool: "list_alert_definitions", Description: "View all alert definitions"},
		}

	// Stream tools
	case "list_streams":
		return []ProactiveSuggestion{
			{Tool: "get_stream", Description: "Get details of a specific stream"},
			{Tool: "create_stream", Description: "Create a new stream"},
		}

	case "get_stream":
		return []ProactiveSuggestion{
			{Tool: "update_stream", Description: "Modify this stream configuration"},
			{Tool: "delete_stream", Description: "Remove this stream"},
		}

	case "create_stream":
		return []ProactiveSuggestion{
			{Tool: "list_streams", Description: "View all streams including the new one"},
		}

	// Event stream target tools
	case "get_event_stream_targets":
		return []ProactiveSuggestion{
			{Tool: "create_event_stream_target", Description: "Create a new event stream target"},
		}

	case "create_event_stream_target":
		return []ProactiveSuggestion{
			{Tool: "get_event_stream_targets", Description: "View all event stream targets"},
		}

	// Data usage tools
	case "export_data_usage":
		return []ProactiveSuggestion{
			{Tool: "update_data_usage_metrics_export_status", Description: "Enable/disable data usage metrics export"},
		}

	// Rule group tools
	case "list_rule_groups":
		return []ProactiveSuggestion{
			{Tool: "get_rule_group", Description: "Get details of a specific rule group"},
			{Tool: "create_rule_group", Description: "Create a new rule group"},
		}

	case "get_rule_group":
		return []ProactiveSuggestion{
			{Tool: "update_rule_group", Description: "Modify this rule group"},
			{Tool: "delete_rule_group", Description: "Remove this rule group"},
		}

	case "create_rule_group":
		return []ProactiveSuggestion{
			{Tool: "list_rule_groups", Description: "View all rule groups including the new one"},
		}

	case "update_rule_group":
		return []ProactiveSuggestion{
			{Tool: "get_rule_group", Description: "View the updated rule group"},
		}

	case "delete_rule_group":
		return []ProactiveSuggestion{
			{Tool: "list_rule_groups", Description: "View remaining rule groups"},
		}

	// Outgoing webhook tools
	case "list_outgoing_webhooks":
		return []ProactiveSuggestion{
			{Tool: "get_outgoing_webhook", Description: "Get details of a specific webhook"},
			{Tool: "create_outgoing_webhook", Description: "Create a new outgoing webhook"},
		}

	case "get_outgoing_webhook":
		return []ProactiveSuggestion{
			{Tool: "update_outgoing_webhook", Description: "Modify this webhook"},
			{Tool: "delete_outgoing_webhook", Description: "Remove this webhook"},
		}

	case "create_outgoing_webhook":
		return []ProactiveSuggestion{
			{Tool: "list_outgoing_webhooks", Description: "View all webhooks including the new one"},
		}

	case "update_outgoing_webhook":
		return []ProactiveSuggestion{
			{Tool: "get_outgoing_webhook", Description: "View the updated webhook"},
		}

	case "delete_outgoing_webhook":
		return []ProactiveSuggestion{
			{Tool: "list_outgoing_webhooks", Description: "View remaining webhooks"},
		}

	// Policy tools
	case "list_policies":
		return []ProactiveSuggestion{
			{Tool: "get_policy", Description: "Get details of a specific policy"},
			{Tool: "create_policy", Description: "Create a new policy"},
		}

	case "get_policy":
		return []ProactiveSuggestion{
			{Tool: "update_policy", Description: "Modify this policy"},
			{Tool: "delete_policy", Description: "Remove this policy"},
		}

	case "create_policy":
		return []ProactiveSuggestion{
			{Tool: "list_policies", Description: "View all policies including the new one"},
		}

	case "update_policy":
		return []ProactiveSuggestion{
			{Tool: "get_policy", Description: "View the updated policy"},
		}

	case "delete_policy":
		return []ProactiveSuggestion{
			{Tool: "list_policies", Description: "View remaining policies"},
		}

	// E2M (Events to Metrics) tools
	case "list_e2m":
		return []ProactiveSuggestion{
			{Tool: "get_e2m", Description: "Get details of a specific E2M mapping"},
			{Tool: "create_e2m", Description: "Create a new E2M mapping"},
		}

	case "get_e2m":
		return []ProactiveSuggestion{
			{Tool: "replace_e2m", Description: "Replace this E2M mapping"},
			{Tool: "delete_e2m", Description: "Remove this E2M mapping"},
		}

	case "create_e2m":
		return []ProactiveSuggestion{
			{Tool: "list_e2m", Description: "View all E2M mappings including the new one"},
		}

	case "replace_e2m":
		return []ProactiveSuggestion{
			{Tool: "get_e2m", Description: "View the replaced E2M mapping"},
		}

	case "delete_e2m":
		return []ProactiveSuggestion{
			{Tool: "list_e2m", Description: "View remaining E2M mappings"},
		}

	// Data access rule tools
	case "list_data_access_rules":
		return []ProactiveSuggestion{
			{Tool: "get_data_access_rule", Description: "Get details of a specific data access rule"},
			{Tool: "create_data_access_rule", Description: "Create a new data access rule"},
		}

	case "get_data_access_rule":
		return []ProactiveSuggestion{
			{Tool: "update_data_access_rule", Description: "Modify this data access rule"},
			{Tool: "delete_data_access_rule", Description: "Remove this data access rule"},
		}

	case "create_data_access_rule":
		return []ProactiveSuggestion{
			{Tool: "list_data_access_rules", Description: "View all data access rules including the new one"},
		}

	case "update_data_access_rule":
		return []ProactiveSuggestion{
			{Tool: "get_data_access_rule", Description: "View the updated data access rule"},
		}

	case "delete_data_access_rule":
		return []ProactiveSuggestion{
			{Tool: "list_data_access_rules", Description: "View remaining data access rules"},
		}

	// Enrichment tools
	case "list_enrichments", "get_enrichments":
		return []ProactiveSuggestion{
			{Tool: "create_enrichment", Description: "Create a new enrichment"},
		}

	case "create_enrichment":
		return []ProactiveSuggestion{
			{Tool: "list_enrichments", Description: "View all enrichments including the new one"},
		}

	case "update_enrichment":
		return []ProactiveSuggestion{
			{Tool: "list_enrichments", Description: "View all enrichments"},
		}

	case "delete_enrichment":
		return []ProactiveSuggestion{
			{Tool: "list_enrichments", Description: "View remaining enrichments"},
		}

	// View tools
	case "list_views":
		return []ProactiveSuggestion{
			{Tool: "get_view", Description: "Get details of a specific view"},
			{Tool: "create_view", Description: "Create a new view"},
		}

	case "get_view":
		return []ProactiveSuggestion{
			{Tool: "replace_view", Description: "Replace this view"},
			{Tool: "delete_view", Description: "Remove this view"},
		}

	case "create_view":
		return []ProactiveSuggestion{
			{Tool: "list_views", Description: "View all views including the new one"},
		}

	case "replace_view":
		return []ProactiveSuggestion{
			{Tool: "get_view", Description: "View the replaced view"},
		}

	case "delete_view":
		return []ProactiveSuggestion{
			{Tool: "list_views", Description: "View remaining views"},
		}

	// View folder tools
	case "list_view_folders":
		return []ProactiveSuggestion{
			{Tool: "get_view_folder", Description: "Get details of a specific view folder"},
			{Tool: "create_view_folder", Description: "Create a new view folder"},
		}

	case "get_view_folder":
		return []ProactiveSuggestion{
			{Tool: "replace_view_folder", Description: "Replace this view folder"},
			{Tool: "delete_view_folder", Description: "Remove this view folder"},
		}

	case "create_view_folder":
		return []ProactiveSuggestion{
			{Tool: "list_view_folders", Description: "View all view folders including the new one"},
		}

	case "replace_view_folder":
		return []ProactiveSuggestion{
			{Tool: "get_view_folder", Description: "View the replaced view folder"},
		}

	case "delete_view_folder":
		return []ProactiveSuggestion{
			{Tool: "list_view_folders", Description: "View remaining view folders"},
		}

	default:
		return nil
	}
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
