// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements hierarchical tool namespacing for cleaner organization.
package tools

import (
	"strings"
)

// ToolNamespace represents a hierarchical grouping of tools
type ToolNamespace string

// Tool namespaces for hierarchical organization
const (
	NamespaceQuery      ToolNamespace = "queries"
	NamespaceAlert      ToolNamespace = "alerts"
	NamespaceDashboard  ToolNamespace = "dashboards"
	NamespacePolicy     ToolNamespace = "policies"
	NamespaceWebhook    ToolNamespace = "webhooks"
	NamespaceE2M        ToolNamespace = "e2m"
	NamespaceStream     ToolNamespace = "streams"
	NamespaceView       ToolNamespace = "views"
	NamespaceRule       ToolNamespace = "rules"
	NamespaceEnrichment ToolNamespace = "enrichments"
	NamespaceDataAccess ToolNamespace = "data_access"
	NamespaceWorkflow   ToolNamespace = "workflows"
	NamespaceMeta       ToolNamespace = "meta"
)

// toolNamespaceMapping maps tool names to their namespaces
var toolNamespaceMapping = map[string]ToolNamespace{
	// Query tools
	"query_logs":                  NamespaceQuery,
	"build_query":                 NamespaceQuery,
	"explain_query":               NamespaceQuery,
	"validate_query":              NamespaceQuery,
	"query_templates":             NamespaceQuery,
	"submit_background_query":     NamespaceQuery,
	"get_background_query_status": NamespaceQuery,
	"get_background_query_data":   NamespaceQuery,

	// Alert tools
	"list_alerts":   NamespaceAlert,
	"get_alert":     NamespaceAlert,
	"create_alert":  NamespaceAlert,
	"update_alert":  NamespaceAlert,
	"delete_alert":  NamespaceAlert,
	"suggest_alert": NamespaceAlert,

	// Dashboard tools
	"list_dashboards":        NamespaceDashboard,
	"get_dashboard":          NamespaceDashboard,
	"create_dashboard":       NamespaceDashboard,
	"update_dashboard":       NamespaceDashboard,
	"delete_dashboard":       NamespaceDashboard,
	"list_dashboard_folders": NamespaceDashboard,

	// Policy tools (TCO)
	"list_policies": NamespacePolicy,
	"get_policy":    NamespacePolicy,
	"create_policy": NamespacePolicy,
	"update_policy": NamespacePolicy,
	"delete_policy": NamespacePolicy,

	// Webhook tools
	"list_outgoing_webhooks":  NamespaceWebhook,
	"get_outgoing_webhook":    NamespaceWebhook,
	"create_outgoing_webhook": NamespaceWebhook,
	"update_outgoing_webhook": NamespaceWebhook,
	"delete_outgoing_webhook": NamespaceWebhook,

	// E2M tools
	"list_e2m":   NamespaceE2M,
	"get_e2m":    NamespaceE2M,
	"create_e2m": NamespaceE2M,
	"update_e2m": NamespaceE2M,
	"delete_e2m": NamespaceE2M,

	// Stream tools
	"list_streams":  NamespaceStream,
	"get_stream":    NamespaceStream,
	"create_stream": NamespaceStream,
	"update_stream": NamespaceStream,
	"delete_stream": NamespaceStream,

	// View tools
	"list_views":        NamespaceView,
	"get_view":          NamespaceView,
	"create_view":       NamespaceView,
	"update_view":       NamespaceView,
	"delete_view":       NamespaceView,
	"list_view_folders": NamespaceView,

	// Rule tools
	"list_rule_groups":  NamespaceRule,
	"get_rule_group":    NamespaceRule,
	"create_rule_group": NamespaceRule,
	"update_rule_group": NamespaceRule,
	"delete_rule_group": NamespaceRule,

	// Enrichment tools
	"list_enrichments":  NamespaceEnrichment,
	"get_enrichment":    NamespaceEnrichment,
	"create_enrichment": NamespaceEnrichment,
	"update_enrichment": NamespaceEnrichment,
	"delete_enrichment": NamespaceEnrichment,

	// Data access tools
	"list_data_access_rules":  NamespaceDataAccess,
	"get_data_access_rule":    NamespaceDataAccess,
	"create_data_access_rule": NamespaceDataAccess,
	"update_data_access_rule": NamespaceDataAccess,
	"delete_data_access_rule": NamespaceDataAccess,

	// Workflow tools
	"investigate_incident": NamespaceWorkflow,
	"health_check":         NamespaceWorkflow,

	// Meta tools
	"discover_tools":       NamespaceMeta,
	"search_tools":         NamespaceMeta,
	"describe_tools":       NamespaceMeta,
	"list_tool_categories": NamespaceMeta,
	"session_context":      NamespaceMeta,
}

// GetToolNamespace returns the namespace for a tool
func GetToolNamespace(toolName string) ToolNamespace {
	if ns, ok := toolNamespaceMapping[toolName]; ok {
		return ns
	}
	return NamespaceMeta // Default to meta for unknown tools
}

// GetToolsByNamespace returns all tools in a namespace
func GetToolsByNamespace(namespace ToolNamespace) []string {
	tools := []string{}
	for name, ns := range toolNamespaceMapping {
		if ns == namespace {
			tools = append(tools, name)
		}
	}
	return tools
}

// GetAllNamespaces returns all available namespaces with their tool counts
func GetAllNamespaces() map[ToolNamespace]int {
	counts := make(map[ToolNamespace]int)
	for _, ns := range toolNamespaceMapping {
		counts[ns]++
	}
	return counts
}

// NamespaceInfo provides information about a namespace
type NamespaceInfo struct {
	Name        ToolNamespace `json:"name"`
	Description string        `json:"description"`
	ToolCount   int           `json:"tool_count"`
	Tools       []string      `json:"tools,omitempty"`
}

// namespaceDescriptions provides human-readable descriptions for namespaces
var namespaceDescriptions = map[ToolNamespace]string{
	NamespaceQuery:      "Log querying, search, and DataPrime query tools",
	NamespaceAlert:      "Alert creation, management, and AI-powered suggestions",
	NamespaceDashboard:  "Dashboard creation, visualization, and folder management",
	NamespacePolicy:     "TCO policies for log retention and routing",
	NamespaceWebhook:    "Outgoing webhooks for Slack, PagerDuty, and custom integrations",
	NamespaceE2M:        "Events to Metrics - convert logs to aggregated metrics",
	NamespaceStream:     "Log streaming to Kafka, Event Streams, and external systems",
	NamespaceView:       "Saved views and search filters",
	NamespaceRule:       "Rule groups for log parsing and enrichment",
	NamespaceEnrichment: "Data enrichment configurations",
	NamespaceDataAccess: "Data access rules and permissions",
	NamespaceWorkflow:   "Automated workflows like incident investigation and health checks",
	NamespaceMeta:       "Tool discovery, session management, and server metadata",
}

// GetNamespaceInfo returns detailed information about a namespace
func GetNamespaceInfo(namespace ToolNamespace, includeTools bool) *NamespaceInfo {
	tools := GetToolsByNamespace(namespace)
	info := &NamespaceInfo{
		Name:        namespace,
		Description: namespaceDescriptions[namespace],
		ToolCount:   len(tools),
	}
	if includeTools {
		info.Tools = tools
	}
	return info
}

// GetAllNamespaceInfo returns info for all namespaces
func GetAllNamespaceInfo(includeTools bool) []*NamespaceInfo {
	namespaces := []ToolNamespace{
		NamespaceQuery, NamespaceAlert, NamespaceDashboard, NamespacePolicy,
		NamespaceWebhook, NamespaceE2M, NamespaceStream, NamespaceView,
		NamespaceRule, NamespaceEnrichment, NamespaceDataAccess, NamespaceWorkflow,
		NamespaceMeta,
	}

	infos := make([]*NamespaceInfo, 0, len(namespaces))
	for _, ns := range namespaces {
		infos = append(infos, GetNamespaceInfo(ns, includeTools))
	}
	return infos
}

// ParseNamespacedTool parses a namespaced tool reference (e.g., "queries/query_logs")
// Returns the namespace and tool name, or empty strings if not namespaced
func ParseNamespacedTool(ref string) (ToolNamespace, string) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return "", ref // Not namespaced, return original
	}
	return ToolNamespace(parts[0]), parts[1]
}

// FormatNamespacedTool returns the fully qualified tool name
func FormatNamespacedTool(toolName string) string {
	ns := GetToolNamespace(toolName)
	return string(ns) + "/" + toolName
}
