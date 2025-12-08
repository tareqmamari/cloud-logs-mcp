package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// Tool defines the interface that all MCP tools must implement.
// This provides a standard contract for tool registration and execution.
type Tool interface {
	// Name returns the unique identifier for this tool
	Name() string

	// Description returns a human-readable description of what this tool does
	Description() string

	// InputSchema returns the JSON Schema for the tool's input parameters
	InputSchema() interface{}

	// Execute runs the tool with the given arguments and returns the result
	Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error)
}

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
		Category:       "create",
		ResourceType:   "alert",
		Prerequisites:  []string{"list_alert_definitions", "list_outgoing_webhooks"},
		RelatedTools:   []string{"create_alert_def", "create_outgoing_webhook"},
		SupportsDryRun: false, // Could be added in future
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

// BaseTool provides common functionality for all tools
type BaseTool struct {
	client *client.Client
	logger *zap.Logger
}

// NewBaseTool creates a new BaseTool
func NewBaseTool(client *client.Client, logger *zap.Logger) *BaseTool {
	return &BaseTool{
		client: client,
		logger: logger,
	}
}

// APIError represents a structured API error with status code
type APIError struct {
	StatusCode int
	Message    string
	RequestID  string // Request ID for support/debugging
	Details    map[string]interface{}
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("%s (Request-ID: %s)", e.Message, e.RequestID)
	}
	return e.Message
}

// IsNotFound returns true if this is a 404 error
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsTimeout returns true if this is a timeout error
func (e *APIError) IsTimeout() bool {
	return e.StatusCode == 408 || e.StatusCode == 504
}

// ExecuteRequest executes an API request and returns the response
func (t *BaseTool) ExecuteRequest(ctx context.Context, req *client.Request) (map[string]interface{}, error) {
	resp, err := t.client.Do(ctx, req)
	if err != nil {
		// Check for context timeout/cancellation
		if ctx.Err() != nil {
			return nil, &APIError{
				StatusCode: 408,
				Message:    fmt.Sprintf("Request timed out: %v", ctx.Err()),
			}
		}
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	// Extract request ID from response headers (IBM Cloud uses X-Request-ID or X-Correlation-ID)
	requestID := resp.Headers.Get("X-Request-ID")
	if requestID == "" {
		requestID = resp.Headers.Get("X-Correlation-ID")
	}
	if requestID == "" {
		requestID = resp.Headers.Get("X-Global-Transaction-ID")
	}

	// Check for error status codes
	if resp.StatusCode >= 400 {
		var apiError map[string]interface{}
		var errorMessage string
		if err := json.Unmarshal(resp.Body, &apiError); err == nil {
			errorMessage = fmt.Sprintf("API error (HTTP %d): %v", resp.StatusCode, apiError)
		} else {
			errorMessage = fmt.Sprintf("API error (HTTP %d): %s", resp.StatusCode, string(resp.Body))
		}
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    errorMessage,
			RequestID:  requestID,
			Details:    apiError,
		}
	}

	// Parse response - handle both JSON and Server-Sent Events (SSE)
	var result map[string]interface{}
	if len(resp.Body) > 0 {
		// Try parsing as Server-Sent Events first (for query responses)
		if sseResult := parseSSEResponse(resp.Body); sseResult != nil {
			return sseResult, nil
		}

		// Fall back to standard JSON
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return result, nil
}

// parseSSEResponse attempts to parse a response body as Server-Sent Events
// Returns nil if parsing fails or if it doesn't look like SSE
// Limits to MaxSSEEvents to prevent memory issues with large result sets
func parseSSEResponse(body []byte) map[string]interface{} {
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "data:") {
		return nil
	}

	// Simple SSE parser for our use case
	// We expect lines starting with "data: " containing JSON
	// We'll aggregate them into a list of events
	var events []interface{}
	totalCount := 0
	truncated := false
	lines := strings.Split(bodyStr, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			totalCount++
			// Limit the number of events to prevent memory issues
			if len(events) >= MaxSSEEvents {
				truncated = true
				continue // Keep counting but don't add more events
			}
			dataStr := strings.TrimPrefix(line, "data: ")
			var data interface{}
			if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
				events = append(events, data)
			}
		}
	}

	if len(events) > 0 {
		result := map[string]interface{}{
			"events": events,
		}
		if truncated {
			result["_truncated"] = true
			result["_total_events"] = totalCount
			result["_shown_events"] = len(events)
		}
		return result
	}

	return nil
}

// MaxResultSize is the maximum size of tool results in bytes (500KB to leave significant headroom under 1MB MCP limit)
// This is reduced to account for JSON formatting, summaries, pagination info, suggestions, and other metadata overhead
const MaxResultSize = 500 * 1024

// FinalResponseLimit is the absolute maximum size for the final response text before sending to MCP
// This ensures we never exceed the 1MB limit even with all metadata added
const FinalResponseLimit = 950 * 1024

// MaxSSEEvents is the maximum number of SSE events to parse from a query response
// This prevents memory issues when queries return very large result sets
const MaxSSEEvents = 200

// FormatResponse formats the response as a text/content for MCP
// If the result exceeds MaxResultSize, it will be truncated with pagination hints
func (t *BaseTool) FormatResponse(result map[string]interface{}) (*mcp.CallToolResult, error) {
	// Handle empty result - len(nil map) is 0, so this covers both nil and empty
	if len(result) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "(no data returned)",
				},
			},
		}, nil
	}

	// Pretty print JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	responseText := string(jsonBytes)

	// Check if response exceeds size limit
	if len(jsonBytes) > MaxResultSize {
		// Try to truncate intelligently by reducing the data
		_, truncatedBytes := truncateResult(result, MaxResultSize)
		if truncatedBytes != nil {
			responseText = string(truncatedBytes)
		} else {
			// Fallback: hard truncate the JSON string
			responseText = string(jsonBytes[:MaxResultSize-500])
		}

		totalItems := countItems(result)
		shownItems := countItemsFromBytes(truncatedBytes)

		// Add pagination guidance with truncation warning
		warningMsg := fmt.Sprintf("\n\n---\n⚠️ RESULT TRUNCATED: Showing %d of %d items (full result was %d bytes, exceeding 1MB limit).\n\n"+
			"**To get ALL results, use pagination by splitting your query:**\n\n"+
			"1. **Time-based pagination** (recommended): Split your time range into smaller chunks:\n"+
			"   - First call: start_date='2024-01-01T00:00:00Z', end_date='2024-01-01T12:00:00Z'\n"+
			"   - Second call: start_date='2024-01-01T12:00:00Z', end_date='2024-01-02T00:00:00Z'\n\n"+
			"2. **Limit-based pagination**: Use smaller limits and note the last timestamp:\n"+
			"   - First call: limit=500\n"+
			"   - Second call: limit=500, start_date=(last timestamp from previous call)\n\n"+
			"3. **Filter more specifically**: Add filters to reduce results:\n"+
			"   - By application: applicationName='your-app' or query='source logs | filter $l.applicationname == \"your-app\"'\n"+
			"   - By subsystem: subsystemName='your-subsystem' or query='source logs | filter $l.subsystemname == \"your-subsystem\"'\n"+
			"   - By severity: query='source logs | filter $m.severity >= 5' (5=error, 6=critical)\n"+
			"   - By keyword: query='source logs | filter $d.text ~~ \"error\"'",
			shownItems, totalItems, len(jsonBytes))
		responseText += warningMsg

		t.logger.Warn("Result truncated due to size limit - pagination recommended",
			zap.Int("original_size", len(jsonBytes)),
			zap.Int("truncated_size", len(responseText)),
			zap.Int("total_items", totalItems),
			zap.Int("shown_items", shownItems),
		)
	}

	// Final safety check
	responseText = ensureResponseLimit(responseText, t.logger)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: responseText,
			},
		},
	}, nil
}

// truncateResult attempts to intelligently truncate the result by reducing array sizes
func truncateResult(result map[string]interface{}, maxSize int) (map[string]interface{}, []byte) {
	// Make a copy to avoid modifying the original
	truncated := make(map[string]interface{})
	for k, v := range result {
		truncated[k] = v
	}

	// Find arrays and truncate them
	for key, val := range truncated {
		if arr, ok := val.([]interface{}); ok && len(arr) > 10 {
			// Binary search for the right size
			low, high := 10, len(arr)
			bestSize := 10

			for low <= high {
				mid := (low + high) / 2
				truncated[key] = arr[:mid]
				testBytes, err := json.MarshalIndent(truncated, "", "  ")
				if err != nil {
					break
				}

				if len(testBytes) <= maxSize-1000 { // Leave room for warning message
					bestSize = mid
					low = mid + 1
				} else {
					high = mid - 1
				}
			}

			truncated[key] = arr[:bestSize]
			truncated["_truncated_info"] = map[string]interface{}{
				"field":          key,
				"original_count": len(arr),
				"shown_count":    bestSize,
			}
		}
	}

	// Also handle nested "events" array (common in query results)
	if events, ok := truncated["events"].([]interface{}); ok && len(events) > 10 {
		low, high := 10, len(events)
		bestSize := 10

		for low <= high {
			mid := (low + high) / 2
			truncated["events"] = events[:mid]
			testBytes, err := json.MarshalIndent(truncated, "", "  ")
			if err != nil {
				break
			}

			if len(testBytes) <= maxSize-1000 {
				bestSize = mid
				low = mid + 1
			} else {
				high = mid - 1
			}
		}

		truncated["events"] = events[:bestSize]
		truncated["_truncated_info"] = map[string]interface{}{
			"field":          "events",
			"original_count": len(events),
			"shown_count":    bestSize,
		}
	}

	// Marshal the truncated result
	truncatedBytes, err := json.MarshalIndent(truncated, "", "  ")
	if err != nil {
		return nil, nil
	}

	return truncated, truncatedBytes
}

// countItems counts the number of items in arrays within the result
func countItems(result map[string]interface{}) int {
	count := 0
	for _, val := range result {
		if arr, ok := val.([]interface{}); ok {
			count += len(arr)
		}
	}
	if count == 0 {
		count = 1 // At least one item (the result itself)
	}
	return count
}

// countItemsFromBytes counts items from JSON bytes (for truncated results)
func countItemsFromBytes(data []byte) int {
	if data == nil {
		return 0
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0
	}
	return countItems(result)
}

// toTitleCase converts a string to title case (first letter of each word capitalized)
func toTitleCase(s string) string {
	if s == "" {
		return s
	}
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// GenerateResultSummary creates an AI-friendly summary of query/list results
// This helps LLMs quickly understand the data without parsing large JSON
func GenerateResultSummary(result map[string]interface{}, resultType string) string {
	var summary strings.Builder

	// Handle query results with events
	if events, ok := result["events"].([]interface{}); ok {
		summary.WriteString("## Query Results Summary\n\n")
		summary.WriteString(fmt.Sprintf("**Total Results:** %d log entries\n\n", len(events)))

		if len(events) > 0 {
			// Analyze severity distribution
			severityDist := analyzeSeverityDistribution(events)
			if len(severityDist) > 0 {
				summary.WriteString("### Severity Distribution\n")
				for sev, count := range severityDist {
					summary.WriteString(fmt.Sprintf("- %s: %d\n", sev, count))
				}
				summary.WriteString("\n")
			}

			// Extract top applications
			topApps := extractTopValues(events, "applicationname", 5)
			if len(topApps) > 0 {
				summary.WriteString("### Top Applications\n")
				for _, app := range topApps {
					summary.WriteString(fmt.Sprintf("- %s: %d entries\n", app.Value, app.Count))
				}
				summary.WriteString("\n")
			}

			// Extract top subsystems
			topSubs := extractTopValues(events, "subsystemname", 5)
			if len(topSubs) > 0 {
				summary.WriteString("### Top Subsystems\n")
				for _, sub := range topSubs {
					summary.WriteString(fmt.Sprintf("- %s: %d entries\n", sub.Value, sub.Count))
				}
				summary.WriteString("\n")
			}

			// Time range
			timeRange := extractTimeRange(events)
			if timeRange != "" {
				summary.WriteString(fmt.Sprintf("### Time Range\n%s\n\n", timeRange))
			}
		}

		return summary.String()
	}

	// Handle list results (alerts, dashboards, policies, etc.)
	for _, val := range result {
		if arr, ok := val.([]interface{}); ok && len(arr) > 0 {
			summary.WriteString(fmt.Sprintf("## %s Summary\n\n", toTitleCase(resultType)))
			summary.WriteString(fmt.Sprintf("**Total Items:** %d\n\n", len(arr)))

			// Extract names/IDs for quick reference
			names := extractFieldValues(arr, []string{"name", "title", "id"}, 10)
			if len(names) > 0 {
				summary.WriteString("### Items\n")
				for i, name := range names {
					summary.WriteString(fmt.Sprintf("%d. %s\n", i+1, name))
				}
				if len(arr) > 10 {
					summary.WriteString(fmt.Sprintf("... and %d more\n", len(arr)-10))
				}
				summary.WriteString("\n")
			}

			return summary.String()
		}
	}

	// For single item results
	if id, ok := result["id"].(string); ok {
		name, _ := result["name"].(string)
		if name == "" {
			name, _ = result["title"].(string)
		}
		summary.WriteString(fmt.Sprintf("## %s Details\n\n", toTitleCase(resultType)))
		summary.WriteString(fmt.Sprintf("**ID:** %s\n", id))
		if name != "" {
			summary.WriteString(fmt.Sprintf("**Name:** %s\n", name))
		}
		return summary.String()
	}

	return ""
}

// ValueCount represents a value and its occurrence count
type ValueCount struct {
	Value string
	Count int
}

// analyzeSeverityDistribution counts log entries by severity level
func analyzeSeverityDistribution(events []interface{}) map[string]int {
	severityNames := map[int]string{
		1: "Debug",
		2: "Verbose",
		3: "Info",
		4: "Warning",
		5: "Error",
		6: "Critical",
	}

	dist := make(map[string]int)
	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			// Try different severity field locations
			var severity int
			if sev, ok := eventMap["severity"].(float64); ok {
				severity = int(sev)
			} else if labels, ok := eventMap["labels"].(map[string]interface{}); ok {
				if sev, ok := labels["severity"].(float64); ok {
					severity = int(sev)
				}
			} else if metadata, ok := eventMap["metadata"].(map[string]interface{}); ok {
				if sev, ok := metadata["severity"].(float64); ok {
					severity = int(sev)
				}
			}

			if severity > 0 {
				name := severityNames[severity]
				if name == "" {
					name = fmt.Sprintf("Level %d", severity)
				}
				dist[name]++
			}
		}
	}
	return dist
}

// extractTopValues extracts the most common values for a given field
func extractTopValues(events []interface{}, fieldName string, limit int) []ValueCount {
	counts := make(map[string]int)

	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			value := findFieldValue(eventMap, fieldName)
			if value != "" {
				counts[value]++
			}
		}
	}

	// Convert to slice and sort by count
	var result []ValueCount
	for val, count := range counts {
		result = append(result, ValueCount{Value: val, Count: count})
	}

	// Sort by count (descending) - O(n log n)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

// findFieldValue searches for a field value in nested structures
func findFieldValue(data map[string]interface{}, fieldName string) string {
	// Direct lookup
	if val, ok := data[fieldName]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}

	// Check in labels
	if labels, ok := data["labels"].(map[string]interface{}); ok {
		if val, ok := labels[fieldName]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}

	// Check in metadata
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		if val, ok := metadata[fieldName]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}

	// Check in user_data (IBM Cloud Logs specific)
	if userData, ok := data["user_data"].(map[string]interface{}); ok {
		if val, ok := userData[fieldName]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}

	return ""
}

// extractTimeRange extracts the time range from events
func extractTimeRange(events []interface{}) string {
	if len(events) == 0 {
		return ""
	}

	var earliest, latest string

	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			timestamp := ""
			if ts, ok := eventMap["timestamp"].(string); ok {
				timestamp = ts
			} else if ts, ok := eventMap["@timestamp"].(string); ok {
				timestamp = ts
			} else if metadata, ok := eventMap["metadata"].(map[string]interface{}); ok {
				if ts, ok := metadata["timestamp"].(string); ok {
					timestamp = ts
				}
			}

			if timestamp != "" {
				if earliest == "" || timestamp < earliest {
					earliest = timestamp
				}
				if latest == "" || timestamp > latest {
					latest = timestamp
				}
			}
		}
	}

	if earliest != "" && latest != "" {
		return fmt.Sprintf("From: %s\nTo: %s", earliest, latest)
	}
	return ""
}

// extractLastTimestamp extracts the latest timestamp from events for pagination
// Returns the timestamp that can be used as start_date for the next page
func extractLastTimestamp(events []interface{}) string {
	if len(events) == 0 {
		return ""
	}

	var latest string

	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			timestamp := ""
			if ts, ok := eventMap["timestamp"].(string); ok {
				timestamp = ts
			} else if ts, ok := eventMap["@timestamp"].(string); ok {
				timestamp = ts
			} else if metadata, ok := eventMap["metadata"].(map[string]interface{}); ok {
				if ts, ok := metadata["timestamp"].(string); ok {
					timestamp = ts
				}
			}

			if timestamp != "" && (latest == "" || timestamp > latest) {
				latest = timestamp
			}
		}
	}

	return latest
}

// PaginationInfo contains metadata for paginating through results
type PaginationInfo struct {
	HasMore       bool   `json:"has_more"`
	TotalReturned int    `json:"total_returned"`
	LastTimestamp string `json:"last_timestamp,omitempty"`
	NextStartDate string `json:"next_start_date,omitempty"`
}

// extractPaginationInfo extracts pagination metadata from query results
func extractPaginationInfo(result map[string]interface{}, limit int, wasTruncated bool) *PaginationInfo {
	events, ok := result["events"].([]interface{})
	if !ok {
		return nil
	}

	info := &PaginationInfo{
		TotalReturned: len(events),
		HasMore:       wasTruncated || len(events) >= limit,
	}

	if lastTs := extractLastTimestamp(events); lastTs != "" {
		info.LastTimestamp = lastTs
		info.NextStartDate = lastTs
	}

	return info
}

// extractFieldValues extracts values from a list of items for given field names
func extractFieldValues(items []interface{}, fieldNames []string, limit int) []string {
	var values []string

	for _, item := range items {
		if len(values) >= limit {
			break
		}
		if itemMap, ok := item.(map[string]interface{}); ok {
			for _, fieldName := range fieldNames {
				if val, ok := itemMap[fieldName]; ok {
					if str, ok := val.(string); ok && str != "" {
						// If we have both name and ID, combine them
						id, hasID := itemMap["id"].(string)
						if fieldName == "name" && hasID {
							values = append(values, fmt.Sprintf("%s (ID: %s)", str, id))
						} else {
							values = append(values, str)
						}
						break
					}
				}
			}
		}
	}

	return values
}

// FormatResponseWithSuggestions formats the response with proactive suggestions
func (t *BaseTool) FormatResponseWithSuggestions(result map[string]interface{}, toolName string) (*mcp.CallToolResult, error) {
	// Handle empty result - len(nil map) is 0, so this covers both nil and empty
	if len(result) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "(no data returned)",
				},
			},
		}, nil
	}

	// Pretty print JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	responseText := string(jsonBytes)

	// Check if response exceeds size limit
	if len(jsonBytes) > MaxResultSize {
		_, truncatedBytes := truncateResult(result, MaxResultSize)
		if truncatedBytes != nil {
			responseText = string(truncatedBytes)
		} else {
			responseText = string(jsonBytes[:MaxResultSize-500])
		}

		totalItems := countItems(result)
		shownItems := countItemsFromBytes(truncatedBytes)

		warningMsg := fmt.Sprintf("\n\n---\n⚠️ RESULT TRUNCATED: Showing %d of %d items (full result was %d bytes, exceeding 1MB limit).\n\n"+
			"💡 **To get ALL results:** Use time-based pagination or add filters to reduce results.",
			shownItems, totalItems, len(jsonBytes))
		responseText += warningMsg

		t.logger.Warn("Result truncated due to size limit",
			zap.Int("original_size", len(jsonBytes)),
			zap.Int("truncated_size", len(responseText)),
		)
	}

	// Add proactive suggestions based on tool and result
	suggestions := GetProactiveSuggestions(toolName, result, false)
	if len(suggestions) > 0 {
		responseText += FormatProactiveSuggestions(suggestions)
	}

	// Final safety check
	responseText = ensureResponseLimit(responseText, t.logger)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: responseText,
			},
		},
	}, nil
}

// FormatResponseWithSummary formats the response with an AI-friendly summary header
func (t *BaseTool) FormatResponseWithSummary(result map[string]interface{}, resultType string) (*mcp.CallToolResult, error) {
	return t.FormatResponseWithSummaryAndSuggestions(result, resultType, "")
}

// FormatResponseWithSummaryAndSuggestions formats the response with summary and proactive suggestions
func (t *BaseTool) FormatResponseWithSummaryAndSuggestions(result map[string]interface{}, resultType string, toolName string) (*mcp.CallToolResult, error) {
	// Handle empty result - len(nil map) is 0, so this covers both nil and empty
	if len(result) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("(no %s returned)", resultType),
				},
			},
		}, nil
	}

	// Generate summary
	summary := GenerateResultSummary(result, resultType)

	// Check for truncation from SSE parsing
	wasTruncated := false
	if truncated, ok := result["_truncated"].(bool); ok && truncated {
		wasTruncated = true
	}

	// Pretty print JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	var responseText string
	if summary != "" {
		responseText = summary + "---\n\n### Raw Data\n\n" + string(jsonBytes)
	} else {
		responseText = string(jsonBytes)
	}

	// Check if response exceeds size limit
	truncatedBySize := false
	if len(responseText) > MaxResultSize {
		truncatedBySize = true
		// Truncate the JSON but keep the summary
		_, truncatedBytes := truncateResult(result, MaxResultSize-len(summary)-500)
		if truncatedBytes != nil {
			if summary != "" {
				responseText = summary + "---\n\n### Raw Data (truncated)\n\n" + string(truncatedBytes)
			} else {
				responseText = string(truncatedBytes)
			}
		}
	}

	// Add pagination info for query results
	if resultType == "query results" {
		paginationInfo := extractPaginationInfo(result, MaxSSEEvents, wasTruncated || truncatedBySize)
		if paginationInfo != nil && paginationInfo.HasMore {
			paginationMsg := fmt.Sprintf("\n\n---\n📄 **PAGINATION INFO:**\n"+
				"- Results returned: %d\n"+
				"- More results available: Yes\n",
				paginationInfo.TotalReturned)

			if paginationInfo.NextStartDate != "" {
				paginationMsg += fmt.Sprintf("- Last timestamp: `%s`\n\n"+
					"**To fetch next page**, use the same query with:\n"+
					"```json\n{\"start_date\": \"%s\"}\n```\n",
					paginationInfo.LastTimestamp,
					paginationInfo.NextStartDate)
			}
			responseText += paginationMsg
		}
	} else if truncatedBySize {
		totalItems := countItems(result)
		shownItems := countItemsFromBytes(nil) // Will return 0 if nil

		warningMsg := fmt.Sprintf("\n\n---\n⚠️ **RESULT TRUNCATED:** Showing %d of %d items.\n\n"+
			"💡 **To get all results:** Use time-based pagination or add filters to reduce results.",
			shownItems, totalItems)
		responseText += warningMsg
	}

	// Add proactive suggestions if tool name provided
	if toolName != "" {
		suggestions := GetProactiveSuggestions(toolName, result, false)
		if len(suggestions) > 0 {
			responseText += FormatProactiveSuggestions(suggestions)
		}
	}

	// Final safety check: ensure response doesn't exceed absolute limit
	responseText = ensureResponseLimit(responseText, t.logger)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: responseText,
			},
		},
	}, nil
}

// ensureResponseLimit ensures the response text doesn't exceed FinalResponseLimit
// This is a safety net to prevent MCP 1MB limit errors
func ensureResponseLimit(text string, logger *zap.Logger) string {
	if len(text) <= FinalResponseLimit {
		return text
	}

	if logger != nil {
		logger.Warn("Response exceeded final limit, truncating",
			zap.Int("original_size", len(text)),
			zap.Int("limit", FinalResponseLimit),
		)
	}

	// Hard truncate and add warning
	truncated := text[:FinalResponseLimit-200]
	truncated += "\n\n---\n⚠️ **Response truncated** due to size limits. Use filters or pagination to get complete results."
	return truncated
}

// NewToolResultError creates a new tool result with an error message
func NewToolResultError(message string) *mcp.CallToolResult {
	// Ensure message is never empty
	if message == "" {
		message = "An unknown error occurred"
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: message,
			},
		},
		IsError: true,
	}
}

// NewToolResultErrorWithSuggestion creates a tool result with an error and recovery guidance
func NewToolResultErrorWithSuggestion(message, suggestion string) *mcp.CallToolResult {
	fullMessage := fmt.Sprintf("%s\n\n💡 **Suggestion:** %s", message, suggestion)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fullMessage,
			},
		},
		IsError: true,
	}
}

// NewResourceNotFoundError creates an error for missing resources with list suggestion
func NewResourceNotFoundError(resourceType, id, listToolName string) *mcp.CallToolResult {
	message := fmt.Sprintf("%s not found with ID: %s", resourceType, id)
	suggestion := fmt.Sprintf("Use '%s' to see available %ss and their IDs.", listToolName, strings.ToLower(resourceType))
	return NewToolResultErrorWithSuggestion(message, suggestion)
}

// NewTimeoutErrorWithFallback creates a timeout error with alternative tool suggestion
func NewTimeoutErrorWithFallback(operation, fallbackTool, fallbackReason string) *mcp.CallToolResult {
	message := fmt.Sprintf("Operation '%s' timed out", operation)
	suggestion := fmt.Sprintf("For large operations, use '%s' which %s.", fallbackTool, fallbackReason)
	return NewToolResultErrorWithSuggestion(message, suggestion)
}

// HandleGetError handles errors from get_* tools with appropriate suggestions
func HandleGetError(err error, resourceType, resourceID, listToolName string) *mcp.CallToolResult {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.IsNotFound() {
			return NewResourceNotFoundError(resourceType, resourceID, listToolName)
		}
		if apiErr.IsTimeout() {
			return NewToolResultErrorWithSuggestion(
				apiErr.Message,
				"Try again in a few moments, or check your network connection.",
			)
		}
	}
	return NewToolResultError(err.Error())
}

// HandleQueryError handles errors from query tools with appropriate suggestions
func HandleQueryError(err error, queryType string) *mcp.CallToolResult {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.IsTimeout() {
			return NewTimeoutErrorWithFallback(
				queryType,
				"submit_background_query",
				"processes queries asynchronously and can handle larger time ranges",
			)
		}
	}
	// Check for context timeout
	if strings.Contains(err.Error(), "context deadline exceeded") ||
		strings.Contains(err.Error(), "context canceled") {
		return NewTimeoutErrorWithFallback(
			queryType,
			"submit_background_query",
			"processes queries asynchronously and can handle larger time ranges",
		)
	}
	return NewToolResultError(err.Error())
}

// GetStringParam safely gets a string parameter from arguments
// It also handles numeric IDs and converts them to strings
func GetStringParam(arguments map[string]interface{}, key string, required bool) (string, error) {
	val, ok := arguments[key]
	if !ok {
		if required {
			return "", fmt.Errorf("missing required argument: %s", key)
		}
		return "", nil
	}

	switch v := val.(type) {
	case string:
		return v, nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	default:
		return "", fmt.Errorf("invalid type for argument %s: expected string or number, got %T", key, val)
	}
}

// GetObjectParam safely gets a map/object parameter from arguments
func GetObjectParam(arguments map[string]interface{}, key string, required bool) (map[string]interface{}, error) {
	val, ok := arguments[key]
	if !ok {
		if required {
			return nil, fmt.Errorf("missing required argument: %s", key)
		}
		return nil, nil
	}

	obj, ok := val.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type for argument %s: expected object", key)
	}

	return obj, nil
}

// GetIntParam safely gets an integer parameter from arguments
func GetIntParam(arguments map[string]interface{}, key string, required bool) (int, error) {
	val, ok := arguments[key]
	if !ok {
		if required {
			return 0, fmt.Errorf("missing required argument: %s", key)
		}
		return 0, nil
	}

	switch v := val.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("invalid type for argument %s: expected number or string, got %T", key, val)
	}
}

// GetBoolParam safely gets a boolean parameter from arguments
func GetBoolParam(arguments map[string]interface{}, key string, required bool) (bool, error) {
	val, ok := arguments[key]
	if !ok {
		if required {
			return false, fmt.Errorf("missing required argument: %s", key)
		}
		return false, nil
	}

	switch v := val.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("invalid type for argument %s: expected boolean or string, got %T", key, val)
	}
}

// GetPaginationParams extracts pagination parameters (limit, cursor)
func GetPaginationParams(arguments map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	if limit, ok := arguments["limit"]; ok {
		params["limit"] = limit
	}

	if cursor, ok := arguments["cursor"]; ok {
		params["cursor"] = cursor
	}

	return params, nil
}

// AddPaginationToQuery adds pagination parameters to query map
func AddPaginationToQuery(query map[string]string, pagination map[string]interface{}) {
	if limit, ok := pagination["limit"]; ok {
		switch v := limit.(type) {
		case float64:
			query["limit"] = strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			query["limit"] = strconv.Itoa(v)
		}
	}

	if cursor, ok := pagination["cursor"]; ok {
		if s, ok := cursor.(string); ok {
			query["cursor"] = s
		}
	}
}

// ValidationResult represents the result of a dry-run validation
type ValidationResult struct {
	Valid       bool                   `json:"valid"`
	Errors      []string               `json:"errors,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
	Summary     map[string]interface{} `json:"summary,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// ValidateRequiredFields checks if all required fields are present in the configuration
func ValidateRequiredFields(config map[string]interface{}, requiredFields []string) []string {
	var errors []string
	for _, field := range requiredFields {
		if _, ok := config[field]; !ok {
			errors = append(errors, fmt.Sprintf("Missing required field: %s", field))
		}
	}
	return errors
}

// ValidateEnumField validates that a field value is one of the allowed values
func ValidateEnumField(config map[string]interface{}, field string, allowedValues []string) string {
	val, ok := config[field]
	if !ok {
		return ""
	}
	strVal, ok := val.(string)
	if !ok {
		return fmt.Sprintf("Field '%s' must be a string", field)
	}
	for _, allowed := range allowedValues {
		if strVal == allowed {
			return ""
		}
	}
	return fmt.Sprintf("Invalid value for '%s': got '%s', must be one of: %v", field, strVal, allowedValues)
}

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

// FormatDryRunResult creates a formatted response for dry-run validation
func FormatDryRunResult(result *ValidationResult, resourceType string, config map[string]interface{}) *mcp.CallToolResult {
	var builder strings.Builder

	builder.WriteString("## Dry-Run Validation Result\n\n")
	builder.WriteString(fmt.Sprintf("**Resource Type:** %s\n\n", resourceType))

	if result.Valid {
		builder.WriteString("✅ **Status:** Valid - configuration is ready for creation\n\n")
	} else {
		builder.WriteString("❌ **Status:** Invalid - please fix errors before creating\n\n")
	}

	if len(result.Errors) > 0 {
		builder.WriteString("### Errors\n\n")
		for _, err := range result.Errors {
			builder.WriteString(fmt.Sprintf("- ❌ %s\n", err))
		}
		builder.WriteString("\n")
	}

	if len(result.Warnings) > 0 {
		builder.WriteString("### Warnings\n\n")
		for _, warn := range result.Warnings {
			builder.WriteString(fmt.Sprintf("- ⚠️ %s\n", warn))
		}
		builder.WriteString("\n")
	}

	if len(result.Summary) > 0 {
		builder.WriteString("### Configuration Summary\n\n")
		for key, val := range result.Summary {
			builder.WriteString(fmt.Sprintf("- **%s:** %v\n", toTitleCase(key), val))
		}
		builder.WriteString("\n")
	}

	if len(result.Suggestions) > 0 {
		builder.WriteString("### Suggestions\n\n")
		for _, sug := range result.Suggestions {
			builder.WriteString(fmt.Sprintf("- 💡 %s\n", sug))
		}
		builder.WriteString("\n")
	}

	// Add the raw config for reference
	builder.WriteString("### Submitted Configuration\n\n```json\n")
	configBytes, _ := json.MarshalIndent(config, "", "  ")
	builder.WriteString(string(configBytes))
	builder.WriteString("\n```\n")

	if result.Valid {
		builder.WriteString("\n---\n**Next step:** Remove the `dry_run: true` parameter to actually create this resource.\n")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: builder.String(),
			},
		},
	}
}
