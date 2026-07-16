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
	NamespaceQuery       ToolNamespace = "queries"
	NamespaceAlert       ToolNamespace = "alerts"
	NamespaceDashboard   ToolNamespace = "dashboards"
	NamespacePolicy      ToolNamespace = "policies"
	NamespaceWebhook     ToolNamespace = "webhooks"
	NamespaceE2M         ToolNamespace = "e2m"
	NamespaceStream      ToolNamespace = "streams"
	NamespaceView        ToolNamespace = "views"
	NamespaceRule        ToolNamespace = "rules"
	NamespaceEnrichment  ToolNamespace = "enrichments"
	NamespaceDataAccess  ToolNamespace = "data_access"
	NamespaceWorkflow    ToolNamespace = "workflows"
	NamespaceMeta        ToolNamespace = "meta"
	NamespaceIngestion   ToolNamespace = "ingestion"
	NamespaceDataUsage   ToolNamespace = "data_usage"
	NamespaceEventStream ToolNamespace = "event_streams"
)

// toolNamespaceMapping maps tool names to their namespaces. It is derived once
// from the single descriptor table (see descriptors.go) so it can never drift
// from the set of tools the server actually registers.
var toolNamespaceMapping = buildToolNamespaceMapping()

// buildToolNamespaceMapping constructs the name -> namespace map from the
// canonical descriptor table. Constructors are invoked with nil dependencies
// purely to read each tool's Name(); this is safe (constructors do no I/O).
func buildToolNamespaceMapping() map[string]ToolNamespace {
	descriptors := Descriptors()
	m := make(map[string]ToolNamespace, len(descriptors))
	for _, d := range descriptors {
		m[d.New(nil, nil).Name()] = d.Namespace
	}
	return m
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
	NamespaceQuery:       "Log querying, search, and DataPrime query tools",
	NamespaceAlert:       "Alert creation, management, and AI-powered suggestions",
	NamespaceDashboard:   "Dashboard creation, visualization, and folder management",
	NamespacePolicy:      "TCO policies for log retention and routing",
	NamespaceWebhook:     "Outgoing webhooks for Slack, PagerDuty, and custom integrations",
	NamespaceE2M:         "Events to Metrics - convert logs to aggregated metrics",
	NamespaceStream:      "Log streaming to Kafka, Event Streams, and external systems",
	NamespaceView:        "Saved views and search filters",
	NamespaceRule:        "Rule groups for log parsing and enrichment",
	NamespaceEnrichment:  "Data enrichment configurations",
	NamespaceDataAccess:  "Data access rules and permissions",
	NamespaceWorkflow:    "Automated workflows like incident investigation and health checks",
	NamespaceMeta:        "Tool discovery, session management, and server metadata",
	NamespaceIngestion:   "Direct log ingestion into IBM Cloud Logs",
	NamespaceDataUsage:   "Data usage export and metrics export configuration",
	NamespaceEventStream: "Event stream targets for forwarding logs to external systems",
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
		NamespaceMeta, NamespaceIngestion, NamespaceDataUsage, NamespaceEventStream,
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
