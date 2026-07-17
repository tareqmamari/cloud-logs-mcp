// Package tools provides the MCP tool implementations for IBM Cloud Logs.
package tools

// InputSchemaExamples provides standard examples for common input types.
// Including examples in schemas significantly improves LLM tool usage accuracy.

// QueryExamples provides example queries for the query_logs tool
var QueryExamples = []interface{}{
	"source logs | filter $m.severity >= 4 | limit 100",
	"source logs | filter $l.applicationname == 'api-gateway' | limit 50",
	"source logs | filter $d.message.contains('error') | groupby $l.applicationname | count() as error_count | sort error_count desc",
	"source logs | filter $m.severity == 6 | limit 10",
}

// TimeRangeExamples provides example time ranges
var TimeRangeExamples = []interface{}{
	"1h",
	"24h",
	"7d",
	"30m",
}

// SeverityExamples provides severity level examples with descriptions
var SeverityExamples = []interface{}{
	1, // Debug
	3, // Info
	4, // Warning
	5, // Error
	6, // Critical
}

// ApplicationNameExamples provides example application names
var ApplicationNameExamples = []interface{}{
	"api-gateway",
	"auth-service",
	"payment-processor",
	"user-service",
}

// AlertDefinitionExample provides a complete alert definition example.
// The shape mirrors a payload verified against the live v3 API: the type
// config lives under a key matching the type name, priorities are lowercase
// (p1..p4, p5_or_unspecified), and enums carry the *_or_unspecified variants.
var AlertDefinitionExample = map[string]interface{}{
	"name":        "High Error Rate Alert",
	"description": "Fires when error logs exceed 10% of total logs over 5 minutes",
	"enabled":     true,
	"type":        "logs_ratio_threshold",
	"logs_ratio_threshold": map[string]interface{}{
		"condition_type": "more_than_or_unspecified",
		"group_by_for":   "both_or_unspecified",
		"numerator": map[string]interface{}{
			"simple_filter": map[string]interface{}{
				"lucene_query": "subsystemname:payment-service AND severity:error",
			},
		},
		"numerator_alias": "errors",
		"denominator": map[string]interface{}{
			"simple_filter": map[string]interface{}{
				"lucene_query": "subsystemname:payment-service",
			},
		},
		"denominator_alias": "total",
		"rules": []map[string]interface{}{
			{
				"condition": map[string]interface{}{
					"threshold": 0.1,
					"time_window": map[string]interface{}{
						"logs_ratio_time_window_specific_value": "minutes_5_or_unspecified",
					},
					"condition_type": "more_than_or_unspecified",
				},
				"override": map[string]interface{}{"priority": "p1"},
			},
		},
	},
}

// DashboardExample provides a complete dashboard example.
// The shape mirrors a payload verified against the live API: every id is a
// {"value": "<uuid>"} object (the API rejects non-UUID ids), widgets define a
// typed chart under definition, and logs queries carry aggregations
// (value aggregations like average also need an observation_field).
var DashboardExample = map[string]interface{}{
	"name":        "Application Overview",
	"description": "Monitor application health and performance",
	"layout": map[string]interface{}{
		"sections": []map[string]interface{}{
			{
				"id": map[string]interface{}{"value": "aaaaaaaa-0000-4000-8000-000000000001"},
				"rows": []map[string]interface{}{
					{
						"id":         map[string]interface{}{"value": "aaaaaaaa-0000-4000-8000-000000000002"},
						"appearance": map[string]interface{}{"height": 19},
						"widgets": []map[string]interface{}{
							{
								"id":         map[string]interface{}{"value": "aaaaaaaa-0000-4000-8000-000000000003"},
								"title":      "Errors by subsystem",
								"appearance": map[string]interface{}{"width": 0},
								"definition": map[string]interface{}{
									"line_chart": map[string]interface{}{
										"legend":  map[string]interface{}{"is_visible": true, "group_by_query": true},
										"tooltip": map[string]interface{}{"show_labels": false, "type": "all"},
										"query_definitions": []map[string]interface{}{
											{
												"id":                 "aaaaaaaa-0000-4000-8000-000000000004",
												"name":               "errors",
												"is_visible":         true,
												"color_scheme":       "classic",
												"scale_type":         "linear",
												"series_count_limit": "20",
												"unit":               "unspecified",
												"resolution":         map[string]interface{}{"buckets_presented": 96},
												"data_mode_type":     "high_unspecified",
												"query": map[string]interface{}{
													"logs": map[string]interface{}{
														"lucene_query": map[string]interface{}{"value": "severity:error"},
														"filters":      []interface{}{},
														"aggregations": []map[string]interface{}{
															{"count": map[string]interface{}{}},
														},
														"group_bys": []map[string]interface{}{
															{"keypath": []string{"subsystemname"}, "scope": "label"},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

// PolicyExample provides a complete policy example
var PolicyExample = map[string]interface{}{
	"name":     "High Priority Logs",
	"priority": "high",
	"application_rule": map[string]interface{}{
		"rule_type": "starts_with",
		"name":      "prod-",
	},
	"subsystem_rule": map[string]interface{}{
		"rule_type": "is",
		"name":      "api",
	},
}

// WebhookExample provides a complete webhook example
var WebhookExample = map[string]interface{}{
	"name": "Slack Notifications",
	"type": "slack",
	"config": map[string]interface{}{
		"url": "https://hooks.slack.com/services/XXX/YYY/ZZZ",
	},
}

// LogEntryExample provides a complete log entry example for ingestion
var LogEntryExample = map[string]interface{}{
	"applicationName": "api-gateway",
	"subsystemName":   "auth",
	"severity":        5,
	"text":            "Authentication failed for user john@example.com",
	"json": map[string]interface{}{
		"user_id":    "12345",
		"ip_address": "192.168.1.100",
		"error_code": "AUTH_FAILED",
	},
}

// DescriptionTemplates provides templates for enhanced tool descriptions.
// Use these to ensure consistent, helpful descriptions across all tools.

// DescriptionTemplate generates a standardized tool description
type DescriptionTemplate struct {
	Summary    string   // One-line summary of what the tool does
	WhenToUse  []string // Scenarios when this tool should be used
	Related    []string // Related tool names
	Notes      []string // Important notes or caveats
	Parameters []string // Key parameter descriptions
}

// Format returns a formatted description string
func (d *DescriptionTemplate) Format() string {
	result := d.Summary

	if len(d.WhenToUse) > 0 {
		result += "\n\n**When to use:**"
		for _, use := range d.WhenToUse {
			result += "\n- " + use
		}
	}

	if len(d.Notes) > 0 {
		result += "\n\n**Notes:**"
		for _, note := range d.Notes {
			result += "\n- " + note
		}
	}

	if len(d.Related) > 0 {
		result += "\n\n**Related tools:** " + joinStrings(d.Related, ", ")
	}

	return result
}

// joinStrings joins strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// StandardDescriptions provides pre-built descriptions for common tool patterns
var StandardDescriptions = struct {
	ListAlerts          string
	GetAlert            string
	CreateAlert         string
	QueryLogs           string
	ListDashboards      string
	CreateDashboard     string
	IngestLogs          string
	InvestigateIncident string
	DiscoverTools       string
}{
	ListAlerts: (&DescriptionTemplate{
		Summary: "List all alerts in IBM Cloud Logs.",
		WhenToUse: []string{
			"Before creating a new alert (to check for duplicates)",
			"To audit current alerting configuration",
			"To find a specific alert's ID for updates or deletion",
			"After creating an alert (to verify it was created)",
		},
		Related: []string{"get_alert", "create_alert", "list_alert_definitions", "list_outgoing_webhooks"},
	}).Format(),

	GetAlert: (&DescriptionTemplate{
		Summary: "Retrieve details of a specific alert by ID.",
		WhenToUse: []string{
			"To view an alert's full configuration",
			"Before updating an alert (to see current state)",
			"When investigating why an alert fired",
			"To check if an alert is active or disabled",
		},
		Related: []string{"list_alerts", "update_alert", "get_alert_definition"},
	}).Format(),

	CreateAlert: (&DescriptionTemplate{
		Summary: "Create a new alert linking an alert definition to notification webhooks.",
		WhenToUse: []string{
			"After creating an alert definition and webhook",
			"To set up monitoring for a new service",
			"To add notifications for specific log patterns",
		},
		Notes: []string{
			"Requires an alert_definition_id (use create_alert_definition first)",
			"Requires a notification_group_id for notifications to work",
			"Use dry_run=true to validate before creating",
		},
		Related: []string{"list_alerts", "create_alert_definition", "create_outgoing_webhook"},
	}).Format(),

	QueryLogs: (&DescriptionTemplate{
		Summary: "Execute a DataPrime or Lucene query to search logs.",
		WhenToUse: []string{
			"To search for specific log patterns or errors",
			"To investigate incidents or anomalies",
			"To analyze log trends over time",
			"Before creating alerts (to validate query returns expected results)",
		},
		Notes: []string{
			"Default limit is 100 results. Use limit parameter for more.",
			"For large result sets, use submit_background_query instead.",
			"Use applicationName and subsystemName filters to narrow results.",
		},
		Related: []string{"submit_background_query", "build_query", "explain_query", "get_query_templates"},
	}).Format(),

	ListDashboards: (&DescriptionTemplate{
		Summary: "List all dashboards in IBM Cloud Logs.",
		WhenToUse: []string{
			"To see available dashboards",
			"Before creating a new dashboard (to check for duplicates)",
			"To find a dashboard ID for viewing or editing",
		},
		Related: []string{"get_dashboard", "create_dashboard", "list_dashboard_folders"},
	}).Format(),

	CreateDashboard: (&DescriptionTemplate{
		Summary: "Create a new dashboard for log visualization.",
		WhenToUse: []string{
			"To build visual monitoring for a service",
			"After defining useful queries you want to track",
			"To create operational dashboards for teams",
		},
		Notes: []string{
			"Use dry_run=true to validate configuration first",
			"Dashboards can be organized into folders",
		},
		Related: []string{"list_dashboards", "get_dashboard", "create_dashboard_folder"},
	}).Format(),

	IngestLogs: (&DescriptionTemplate{
		Summary: "Send log entries to IBM Cloud Logs for ingestion.",
		WhenToUse: []string{
			"To test log ingestion configuration",
			"To send custom log entries from scripts or tools",
			"To backfill historical log data",
		},
		Notes: []string{
			"Maximum batch size is 1000 entries per request",
			"Timestamp is auto-generated if not provided",
			"applicationName and subsystemName are required",
		},
		Related: []string{"query_logs", "list_policies"},
	}).Format(),

	InvestigateIncident: (&DescriptionTemplate{
		Summary: "Start a structured incident investigation workflow.",
		WhenToUse: []string{
			"When errors or anomalies are detected",
			"To systematically analyze an issue",
			"When an alert fires and needs investigation",
			"To generate a root cause analysis",
		},
		Related: []string{"query_logs", "list_alerts", "create_alert"},
	}).Format(),

	DiscoverTools: (&DescriptionTemplate{
		Summary: "Find relevant tools based on natural language intent.",
		WhenToUse: []string{
			"When unsure which tool to use for a task",
			"To explore available capabilities",
			"To get suggestions for multi-step workflows",
		},
		Notes: []string{
			"Returns tools ranked by relevance with confidence scores",
			"Includes suggested tool chains for complex tasks",
		},
		Related: []string{"session_context", "health_check"},
	}).Format(),
}
