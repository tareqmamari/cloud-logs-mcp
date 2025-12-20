// Package tools provides the MCP tool implementations for IBM Cloud Logs.
package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// Annotation helper functions to create common annotation patterns.
// These help ensure consistent annotation across all tools.

// ToolIcon represents an icon for MCP 2025-11-25 tool metadata.
// Icons can be data URIs or HTTPS URLs.
type ToolIcon string

// Tool icons by category (using emoji-style Unicode for simplicity).
// MCP 2025-11-25 supports icon metadata for tools, resources, and prompts.
// These can be replaced with actual SVG data URIs or icon URLs if needed.
const (
	IconQuery       ToolIcon = "🔍"  // Search/query operations
	IconAlert       ToolIcon = "🔔"  // Alert management
	IconDashboard   ToolIcon = "📊"  // Dashboard/visualization
	IconPolicy      ToolIcon = "📋"  // Policy management
	IconWebhook     ToolIcon = "🔗"  // Webhook/integration
	IconE2M         ToolIcon = "📈"  // Events to metrics
	IconEnrichment  ToolIcon = "✨"  // Data enrichment
	IconView        ToolIcon = "👁️" // Views
	IconDataAccess  ToolIcon = "🔐"  // Data access rules
	IconStream      ToolIcon = "🌊"  // Streaming
	IconIngestion   ToolIcon = "📥"  // Log ingestion
	IconWorkflow    ToolIcon = "⚙️" // Workflows
	IconMeta        ToolIcon = "ℹ️" // Meta/discovery
	IconCreate      ToolIcon = "➕"  // Create operations
	IconUpdate      ToolIcon = "✏️" // Update operations
	IconDelete      ToolIcon = "🗑️" // Delete operations
	IconInvestigate ToolIcon = "🔬"  // Investigation
	IconHealth      ToolIcon = "💚"  // Health checks
)

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

// ReadOnlyAnnotations returns annotations for read-only tools (list, get operations).
// These tools don't modify any state and are safe to call repeatedly.
func ReadOnlyAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:          title,
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(false), // IBM Cloud Logs is a bounded system
	}
}

// CreateAnnotations returns annotations for create operations.
// These tools create new resources but don't modify existing ones.
func CreateAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(false), // Create is additive, not destructive
		IdempotentHint:  false,          // Creating twice creates duplicates
		OpenWorldHint:   boolPtr(false),
	}
}

// UpdateAnnotations returns annotations for update operations.
// These tools modify existing resources but don't delete them.
func UpdateAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(false), // Updates preserve resources
		IdempotentHint:  true,           // Same update can be applied multiple times
		OpenWorldHint:   boolPtr(false),
	}
}

// DeleteAnnotations returns annotations for delete operations.
// These tools permanently remove resources and require caution.
func DeleteAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(true), // Delete is destructive
		IdempotentHint:  true,          // Deleting twice is safe (already gone)
		OpenWorldHint:   boolPtr(false),
	}
}

// QueryAnnotations returns annotations for query tools.
// These tools read data but may interact with external data sources.
func QueryAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:          title,
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(false), // Queries a bounded log system
	}
}

// WorkflowAnnotations returns annotations for workflow/composite tools.
// These tools orchestrate multiple operations.
func WorkflowAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:          title,
		ReadOnlyHint:   true, // Workflows analyze but don't modify
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(false),
	}
}

// IngestionAnnotations returns annotations for log ingestion tools.
// These tools add new data but don't modify existing data.
func IngestionAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(false), // Ingestion is additive
		IdempotentHint:  false,          // Same log can be ingested multiple times
		OpenWorldHint:   boolPtr(false),
	}
}

// DefaultAnnotations returns default annotations when no specific hints are needed.
// This provides a consistent baseline for tools.
func DefaultAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:         title,
		OpenWorldHint: boolPtr(false),
	}
}

// toolIconMap maps exact tool names to their icons
var toolIconMap = map[string]ToolIcon{
	// CRUD operations
	"create_alert": IconCreate, "update_alert": IconUpdate, "delete_alert": IconDelete,
	"create_dashboard": IconCreate, "update_dashboard": IconUpdate, "delete_dashboard": IconDelete,
	"create_dashboard_folder": IconCreate, "update_dashboard_folder": IconUpdate, "delete_dashboard_folder": IconDelete,
	"create_policy": IconCreate, "update_policy": IconUpdate, "delete_policy": IconDelete,
	"create_outgoing_webhook": IconCreate, "update_outgoing_webhook": IconUpdate, "delete_outgoing_webhook": IconDelete,
	"create_e2m": IconCreate, "replace_e2m": IconUpdate, "delete_e2m": IconDelete,
	"create_enrichment": IconCreate, "update_enrichment": IconUpdate, "delete_enrichment": IconDelete,
	"create_view": IconCreate, "replace_view": IconUpdate, "delete_view": IconDelete,
	"create_view_folder": IconCreate, "replace_view_folder": IconUpdate, "delete_view_folder": IconDelete,
	"create_data_access_rule": IconCreate, "update_data_access_rule": IconUpdate, "delete_data_access_rule": IconDelete,
	"create_stream": IconCreate, "update_stream": IconUpdate, "delete_stream": IconDelete,
	"create_event_stream_target": IconCreate, "update_event_stream_target": IconUpdate, "delete_event_stream_target": IconDelete,
	"create_rule_group": IconCreate, "update_rule_group": IconUpdate, "delete_rule_group": IconDelete,
	"create_alert_definition": IconCreate, "update_alert_definition": IconUpdate, "delete_alert_definition": IconDelete,
	// Special tools
	"ingest_logs":          IconIngestion,
	"investigate_incident": IconInvestigate,
	"smart_investigate":    IconInvestigate,
	"health_check":         IconHealth,
}

// toolIconPrefixes maps tool name prefixes to their icons (checked in order)
var toolIconPrefixes = []struct {
	prefix string
	icon   ToolIcon
}{
	// Query tools
	{"query_", IconQuery}, {"build_query", IconQuery}, {"get_dataprime", IconQuery},
	{"explain_query", IconQuery}, {"validate_query", IconQuery},
	{"submit_background", IconQuery}, {"get_background", IconQuery}, {"cancel_background", IconQuery},
	// List/Get operations by category
	{"list_alert", IconAlert}, {"get_alert", IconAlert}, {"suggest_alert", IconAlert},
	{"list_dashboard", IconDashboard}, {"get_dashboard", IconDashboard},
	{"list_polic", IconPolicy}, {"get_policy", IconPolicy},
	{"list_outgoing", IconWebhook}, {"get_outgoing", IconWebhook},
	{"list_e2m", IconE2M}, {"get_e2m", IconE2M},
	{"list_enrichment", IconEnrichment}, {"get_enrichment", IconEnrichment},
	{"list_view", IconView}, {"get_view", IconView},
	{"list_data_access", IconDataAccess}, {"get_data_access", IconDataAccess},
	{"list_stream", IconStream}, {"get_stream", IconStream}, {"get_event_stream", IconStream},
	{"list_rule", IconPolicy}, {"get_rule", IconPolicy},
	// Meta tools
	{"search_tool", IconMeta}, {"describe_tool", IconMeta}, {"list_tool", IconMeta}, {"discover_tool", IconMeta},
}

// GetToolIcon returns the appropriate icon for a tool based on its name.
// MCP 2025-11-25 supports icon metadata for tools.
func GetToolIcon(toolName string) ToolIcon {
	// Check exact match first
	if icon, ok := toolIconMap[toolName]; ok {
		return icon
	}

	// Check prefix matches
	for _, p := range toolIconPrefixes {
		if hasPrefix(toolName, p.prefix) {
			return p.icon
		}
	}

	return IconWorkflow
}

// hasPrefix checks if name starts with the given prefix
func hasPrefix(name, prefix string) bool {
	return len(name) >= len(prefix) && name[:len(prefix)] == prefix
}
