package tools

// ToolCapability describes what a tool can do for AI planning purposes
type ToolCapability struct {
	// Category groups tools by function (query, create, update, delete, list, etc.)
	Category string `json:"category"`
	// ResourceType indicates what resource the tool operates on
	ResourceType string `json:"resource_type"`
	// IsReadOnly indicates if the tool only reads data (safe to call repeatedly)
	IsReadOnly bool `json:"is_read_only"`
	// RequiresID indicates if the tool needs a resource ID to operate
	RequiresID bool `json:"requires_id,omitempty"`
	// CanPaginate indicates if the tool supports pagination
	CanPaginate bool `json:"can_paginate,omitempty"`
	// SupportsDryRun indicates if the tool supports dry-run validation
	SupportsDryRun bool `json:"supports_dry_run,omitempty"`
	// Prerequisites lists tools that should typically be called before this one
	Prerequisites []string `json:"prerequisites,omitempty"`
	// RelatedTools lists tools that are commonly used together with this one
	RelatedTools []string `json:"related_tools,omitempty"`
}

// ToolCapabilities maps tool names to their capability annotations.
// This helps LLMs understand what each tool does and plan tool usage effectively.
var ToolCapabilities = map[string]ToolCapability{
	// Query tools
	"query_logs": {
		Category:     "query",
		ResourceType: "logs",
		IsReadOnly:   true,
		CanPaginate:  true,
		RelatedTools: []string{"build_query", "create_dashboard", "create_alert"},
	},
	"build_query": {
		Category:     "query",
		ResourceType: "query",
		IsReadOnly:   true,
		RelatedTools: []string{"query_logs"},
	},
	"submit_background_query": {
		Category:     "query",
		ResourceType: "logs",
		IsReadOnly:   true,
		RelatedTools: []string{"get_background_query_status", "get_background_query_data", "cancel_background_query"},
	},
	"get_background_query_status": {
		Category:     "query",
		ResourceType: "background_query",
		IsReadOnly:   true,
		RequiresID:   true,
		RelatedTools: []string{"get_background_query_data"},
	},
	"get_background_query_data": {
		Category:     "query",
		ResourceType: "background_query",
		IsReadOnly:   true,
		RequiresID:   true,
	},
	"cancel_background_query": {
		Category:     "delete",
		ResourceType: "background_query",
		RequiresID:   true,
	},

	// Dashboard tools
	"list_dashboards": {
		Category:     "list",
		ResourceType: "dashboard",
		IsReadOnly:   true,
		RelatedTools: []string{"get_dashboard", "create_dashboard"},
	},
	"get_dashboard": {
		Category:     "read",
		ResourceType: "dashboard",
		IsReadOnly:   true,
		RequiresID:   true,
		RelatedTools: []string{"update_dashboard", "delete_dashboard", "pin_dashboard"},
	},
	"create_dashboard": {
		Category:       "create",
		ResourceType:   "dashboard",
		SupportsDryRun: true,
		RelatedTools:   []string{"list_dashboards", "get_dashboard", "query_logs"},
	},
	"update_dashboard": {
		Category:      "update",
		ResourceType:  "dashboard",
		RequiresID:    true,
		Prerequisites: []string{"get_dashboard"},
	},
	"delete_dashboard": {
		Category:      "delete",
		ResourceType:  "dashboard",
		RequiresID:    true,
		Prerequisites: []string{"get_dashboard"},
	},
	"pin_dashboard": {
		Category:     "update",
		ResourceType: "dashboard",
		RequiresID:   true,
	},
	"unpin_dashboard": {
		Category:     "update",
		ResourceType: "dashboard",
		RequiresID:   true,
	},
	"set_default_dashboard": {
		Category:     "update",
		ResourceType: "dashboard",
		RequiresID:   true,
	},
	"move_dashboard_to_folder": {
		Category:     "update",
		ResourceType: "dashboard",
		RequiresID:   true,
	},

	// Dashboard folder tools
	"list_dashboard_folders": {
		Category:     "list",
		ResourceType: "folder",
		IsReadOnly:   true,
		RelatedTools: []string{"create_dashboard_folder", "move_dashboard_to_folder"},
	},
	"get_dashboard_folder": {
		Category:     "read",
		ResourceType: "folder",
		IsReadOnly:   true,
		RequiresID:   true,
	},
	"create_dashboard_folder": {
		Category:     "create",
		ResourceType: "folder",
		RelatedTools: []string{"move_dashboard_to_folder"},
	},
	"update_dashboard_folder": {
		Category:      "update",
		ResourceType:  "folder",
		RequiresID:    true,
		Prerequisites: []string{"get_dashboard_folder"},
	},
	"delete_dashboard_folder": {
		Category:      "delete",
		ResourceType:  "folder",
		RequiresID:    true,
		Prerequisites: []string{"get_dashboard_folder"},
	},

	// Alert tools
	"list_alerts": {
		Category:     "list",
		ResourceType: "alert",
		IsReadOnly:   true,
		CanPaginate:  true,
		RelatedTools: []string{"get_alert", "create_alert", "list_alert_definitions"},
	},
	"get_alert": {
		Category:     "read",
		ResourceType: "alert",
		IsReadOnly:   true,
		RequiresID:   true,
		RelatedTools: []string{"update_alert", "delete_alert"},
	},
	"create_alert": {
		Category:      "create",
		ResourceType:  "alert",
		Prerequisites: []string{"list_alert_definitions", "list_outgoing_webhooks"},
		RelatedTools:  []string{"create_alert_def", "create_outgoing_webhook"},
	},
	"update_alert": {
		Category:      "update",
		ResourceType:  "alert",
		RequiresID:    true,
		Prerequisites: []string{"get_alert"},
	},
	"delete_alert": {
		Category:      "delete",
		ResourceType:  "alert",
		RequiresID:    true,
		Prerequisites: []string{"get_alert"},
	},

	// Alert definition tools
	"list_alert_definitions": {
		Category:     "list",
		ResourceType: "alert_definition",
		IsReadOnly:   true,
		CanPaginate:  true,
		RelatedTools: []string{"get_alert_definition", "create_alert_definition"},
	},
	"get_alert_definition": {
		Category:     "read",
		ResourceType: "alert_definition",
		IsReadOnly:   true,
		RequiresID:   true,
		RelatedTools: []string{"update_alert_definition", "delete_alert_definition"},
	},
	"create_alert_definition": {
		Category:     "create",
		ResourceType: "alert_definition",
		RelatedTools: []string{"create_alert", "query_logs"},
	},
	"update_alert_definition": {
		Category:      "update",
		ResourceType:  "alert_definition",
		RequiresID:    true,
		Prerequisites: []string{"get_alert_definition"},
	},
	"delete_alert_definition": {
		Category:      "delete",
		ResourceType:  "alert_definition",
		RequiresID:    true,
		Prerequisites: []string{"get_alert_definition"},
	},

	// Ingestion tools
	"ingest_logs": {
		Category:     "create",
		ResourceType: "logs",
		RelatedTools: []string{"query_logs"},
	},

	// Data usage tools
	"export_data_usage": {
		Category:     "read",
		ResourceType: "data_usage",
		IsReadOnly:   true,
		RelatedTools: []string{"update_data_usage_metrics_export_status"},
	},
	"update_data_usage_metrics_export_status": {
		Category:      "update",
		ResourceType:  "data_usage",
		Prerequisites: []string{"export_data_usage"},
	},

	// Stream tools (for streaming logs to Event Streams/Kafka)
	"list_streams": {
		Category:     "list",
		ResourceType: "stream",
		IsReadOnly:   true,
		RelatedTools: []string{"get_stream", "create_stream"},
	},
	"get_stream": {
		Category:     "read",
		ResourceType: "stream",
		IsReadOnly:   true,
		RequiresID:   true,
		RelatedTools: []string{"update_stream", "delete_stream"},
	},
	"create_stream": {
		Category:     "create",
		ResourceType: "stream",
		RelatedTools: []string{"list_streams"},
	},
	"update_stream": {
		Category:      "update",
		ResourceType:  "stream",
		RequiresID:    true,
		Prerequisites: []string{"get_stream"},
	},
	"delete_stream": {
		Category:      "delete",
		ResourceType:  "stream",
		RequiresID:    true,
		Prerequisites: []string{"get_stream"},
	},

	// Event stream target tools (alternative API for streams)
	"get_event_stream_targets": {
		Category:     "list",
		ResourceType: "event_stream",
		IsReadOnly:   true,
		RelatedTools: []string{"create_event_stream_target"},
	},
	"create_event_stream_target": {
		Category:     "create",
		ResourceType: "event_stream",
		RelatedTools: []string{"get_event_stream_targets"},
	},
	"update_event_stream_target": {
		Category:     "update",
		ResourceType: "event_stream",
		RequiresID:   true,
	},
	"delete_event_stream_target": {
		Category:     "delete",
		ResourceType: "event_stream",
		RequiresID:   true,
	},

	// Rule group tools (for parsing and transforming log data)
	"list_rule_groups": {
		Category:     "list",
		ResourceType: "rule_group",
		IsReadOnly:   true,
		RelatedTools: []string{"get_rule_group", "create_rule_group"},
	},
	"get_rule_group": {
		Category:     "read",
		ResourceType: "rule_group",
		IsReadOnly:   true,
		RequiresID:   true,
		RelatedTools: []string{"update_rule_group", "delete_rule_group"},
	},
	"create_rule_group": {
		Category:     "create",
		ResourceType: "rule_group",
		RelatedTools: []string{"list_rule_groups"},
	},
	"update_rule_group": {
		Category:      "update",
		ResourceType:  "rule_group",
		RequiresID:    true,
		Prerequisites: []string{"get_rule_group"},
	},
	"delete_rule_group": {
		Category:      "delete",
		ResourceType:  "rule_group",
		RequiresID:    true,
		Prerequisites: []string{"get_rule_group"},
	},

	// Outgoing webhook tools (for alert notifications)
	"list_outgoing_webhooks": {
		Category:     "list",
		ResourceType: "outgoing_webhook",
		IsReadOnly:   true,
		RelatedTools: []string{"get_outgoing_webhook", "create_outgoing_webhook"},
	},
	"get_outgoing_webhook": {
		Category:     "read",
		ResourceType: "outgoing_webhook",
		IsReadOnly:   true,
		RequiresID:   true,
		RelatedTools: []string{"update_outgoing_webhook", "delete_outgoing_webhook"},
	},
	"create_outgoing_webhook": {
		Category:     "create",
		ResourceType: "outgoing_webhook",
		RelatedTools: []string{"list_outgoing_webhooks", "create_alert"},
	},
	"update_outgoing_webhook": {
		Category:      "update",
		ResourceType:  "outgoing_webhook",
		RequiresID:    true,
		Prerequisites: []string{"get_outgoing_webhook"},
	},
	"delete_outgoing_webhook": {
		Category:      "delete",
		ResourceType:  "outgoing_webhook",
		RequiresID:    true,
		Prerequisites: []string{"get_outgoing_webhook"},
	},

	// Policy tools (TCO policies for log management)
	"list_policies": {
		Category:     "list",
		ResourceType: "policy",
		IsReadOnly:   true,
		RelatedTools: []string{"get_policy", "create_policy"},
	},
	"get_policy": {
		Category:     "read",
		ResourceType: "policy",
		IsReadOnly:   true,
		RequiresID:   true,
		RelatedTools: []string{"update_policy", "delete_policy"},
	},
	"create_policy": {
		Category:     "create",
		ResourceType: "policy",
		RelatedTools: []string{"list_policies"},
	},
	"update_policy": {
		Category:      "update",
		ResourceType:  "policy",
		RequiresID:    true,
		Prerequisites: []string{"get_policy"},
	},
	"delete_policy": {
		Category:      "delete",
		ResourceType:  "policy",
		RequiresID:    true,
		Prerequisites: []string{"get_policy"},
	},

	// E2M tools (Events to Metrics conversion)
	"list_e2m": {
		Category:     "list",
		ResourceType: "e2m",
		IsReadOnly:   true,
		RelatedTools: []string{"get_e2m", "create_e2m"},
	},
	"get_e2m": {
		Category:     "read",
		ResourceType: "e2m",
		IsReadOnly:   true,
		RequiresID:   true,
		RelatedTools: []string{"replace_e2m", "delete_e2m"},
	},
	"create_e2m": {
		Category:     "create",
		ResourceType: "e2m",
		RelatedTools: []string{"list_e2m", "query_logs"},
	},
	"replace_e2m": {
		Category:      "update",
		ResourceType:  "e2m",
		RequiresID:    true,
		Prerequisites: []string{"get_e2m"},
	},
	"delete_e2m": {
		Category:      "delete",
		ResourceType:  "e2m",
		RequiresID:    true,
		Prerequisites: []string{"get_e2m"},
	},

	// Data access rule tools (for controlling data access)
	"list_data_access_rules": {
		Category:     "list",
		ResourceType: "data_access_rule",
		IsReadOnly:   true,
		RelatedTools: []string{"get_data_access_rule", "create_data_access_rule"},
	},
	"get_data_access_rule": {
		Category:     "read",
		ResourceType: "data_access_rule",
		IsReadOnly:   true,
		RequiresID:   true,
		RelatedTools: []string{"update_data_access_rule", "delete_data_access_rule"},
	},
	"create_data_access_rule": {
		Category:     "create",
		ResourceType: "data_access_rule",
		RelatedTools: []string{"list_data_access_rules"},
	},
	"update_data_access_rule": {
		Category:      "update",
		ResourceType:  "data_access_rule",
		RequiresID:    true,
		Prerequisites: []string{"get_data_access_rule"},
	},
	"delete_data_access_rule": {
		Category:      "delete",
		ResourceType:  "data_access_rule",
		RequiresID:    true,
		Prerequisites: []string{"get_data_access_rule"},
	},

	// Enrichment tools (for enriching log data)
	"list_enrichments": {
		Category:     "list",
		ResourceType: "enrichment",
		IsReadOnly:   true,
		RelatedTools: []string{"get_enrichments", "create_enrichment"},
	},
	"get_enrichments": {
		Category:     "read",
		ResourceType: "enrichment",
		IsReadOnly:   true,
		RelatedTools: []string{"update_enrichment", "delete_enrichment"},
	},
	"create_enrichment": {
		Category:     "create",
		ResourceType: "enrichment",
		RelatedTools: []string{"list_enrichments"},
	},
	"update_enrichment": {
		Category:      "update",
		ResourceType:  "enrichment",
		RequiresID:    true,
		Prerequisites: []string{"get_enrichments"},
	},
	"delete_enrichment": {
		Category:      "delete",
		ResourceType:  "enrichment",
		RequiresID:    true,
		Prerequisites: []string{"get_enrichments"},
	},

	// View tools (saved log views)
	"list_views": {
		Category:     "list",
		ResourceType: "view",
		IsReadOnly:   true,
		RelatedTools: []string{"get_view", "create_view"},
	},
	"get_view": {
		Category:     "read",
		ResourceType: "view",
		IsReadOnly:   true,
		RequiresID:   true,
		RelatedTools: []string{"replace_view", "delete_view"},
	},
	"create_view": {
		Category:     "create",
		ResourceType: "view",
		RelatedTools: []string{"list_views", "list_view_folders"},
	},
	"replace_view": {
		Category:      "update",
		ResourceType:  "view",
		RequiresID:    true,
		Prerequisites: []string{"get_view"},
	},
	"delete_view": {
		Category:      "delete",
		ResourceType:  "view",
		RequiresID:    true,
		Prerequisites: []string{"get_view"},
	},

	// View folder tools (for organizing views)
	"list_view_folders": {
		Category:     "list",
		ResourceType: "view_folder",
		IsReadOnly:   true,
		RelatedTools: []string{"get_view_folder", "create_view_folder"},
	},
	"get_view_folder": {
		Category:     "read",
		ResourceType: "view_folder",
		IsReadOnly:   true,
		RequiresID:   true,
		RelatedTools: []string{"replace_view_folder", "delete_view_folder"},
	},
	"create_view_folder": {
		Category:     "create",
		ResourceType: "view_folder",
		RelatedTools: []string{"list_view_folders", "create_view"},
	},
	"replace_view_folder": {
		Category:      "update",
		ResourceType:  "view_folder",
		RequiresID:    true,
		Prerequisites: []string{"get_view_folder"},
	},
	"delete_view_folder": {
		Category:      "delete",
		ResourceType:  "view_folder",
		RequiresID:    true,
		Prerequisites: []string{"get_view_folder"},
	},

	// Data access policy tools
	"list_data_access_policies": {
		Category:     "list",
		ResourceType: "data_access_policy",
		IsReadOnly:   true,
		RelatedTools: []string{"get_data_access_policy", "create_data_access_policy"},
	},
	"get_data_access_policy": {
		Category:     "read",
		ResourceType: "data_access_policy",
		IsReadOnly:   true,
		RequiresID:   true,
		RelatedTools: []string{"update_data_access_policy", "delete_data_access_policy"},
	},
	"create_data_access_policy": {
		Category:     "create",
		ResourceType: "data_access_policy",
		RelatedTools: []string{"list_data_access_policies"},
	},
	"update_data_access_policy": {
		Category:      "update",
		ResourceType:  "data_access_policy",
		RequiresID:    true,
		Prerequisites: []string{"get_data_access_policy"},
	},
	"delete_data_access_policy": {
		Category:      "delete",
		ResourceType:  "data_access_policy",
		RequiresID:    true,
		Prerequisites: []string{"get_data_access_policy"},
	},

	// AI helper tools
	"explain_query": {
		Category:     "query",
		ResourceType: "query",
		IsReadOnly:   true,
		RelatedTools: []string{"query_logs", "build_query"},
	},
	"suggest_alert": {
		Category:     "query",
		ResourceType: "alert",
		IsReadOnly:   true,
		RelatedTools: []string{"create_alert", "query_logs"},
	},
	"get_audit_log": {
		Category:     "read",
		ResourceType: "audit",
		IsReadOnly:   true,
	},
}

// GetToolCapability returns the capability annotation for a tool, or nil if not found
func GetToolCapability(toolName string) *ToolCapability {
	if capability, ok := ToolCapabilities[toolName]; ok {
		return &capability
	}
	return nil
}

// GetToolsByCategory returns all tools in a given category
func GetToolsByCategory(category string) []string {
	var tools []string
	for name, cap := range ToolCapabilities {
		if cap.Category == category {
			tools = append(tools, name)
		}
	}
	return tools
}

// GetToolsByResourceType returns all tools that operate on a given resource type
func GetToolsByResourceType(resourceType string) []string {
	var tools []string
	for name, cap := range ToolCapabilities {
		if cap.ResourceType == resourceType {
			tools = append(tools, name)
		}
	}
	return tools
}

// GetReadOnlyTools returns all tools that are safe to call without side effects
func GetReadOnlyTools() []string {
	var tools []string
	for name, cap := range ToolCapabilities {
		if cap.IsReadOnly {
			tools = append(tools, name)
		}
	}
	return tools
}
