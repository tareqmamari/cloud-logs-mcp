// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements dynamic toolset pattern for token-efficient tool discovery.
// Instead of loading all tool schemas upfront, LLMs use search → describe → execute.
package tools

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// ToolBrief provides minimal tool information for search results.
// This drastically reduces token usage compared to full schemas.
type ToolBrief struct {
	Name        string        `json:"name"`
	Namespace   ToolNamespace `json:"namespace"`   // Hierarchical grouping (e.g., "queries", "alerts")
	Description string        `json:"description"` // Max 100 chars
	Category    string        `json:"category"`
	Complexity  string        `json:"complexity"`
}

// ToolSchema provides full tool information when explicitly requested.
type ToolSchema struct {
	Name        string               `json:"name"`
	Namespace   ToolNamespace        `json:"namespace"` // Hierarchical grouping
	Description string               `json:"description"`
	InputSchema interface{}          `json:"input_schema"`
	Annotations *mcp.ToolAnnotations `json:"annotations,omitempty"`
	CostHints   *CostHints           `json:"cost_hints,omitempty"`
	Metadata    *ToolMetadata        `json:"metadata,omitempty"`
}

// toolRegistry holds all registered tools for dynamic lookup
var registeredTools = make(map[string]Tool)

// RegisterToolForDynamic registers a tool in the dynamic registry
func RegisterToolForDynamic(t Tool) {
	registeredTools[t.Name()] = t
}

// GetRegisteredTool returns a tool by name
func GetRegisteredTool(name string) Tool {
	return registeredTools[name]
}

// GetAllToolNames returns all registered tool names
func GetAllToolNames() []string {
	names := make([]string, 0, len(registeredTools))
	for name := range registeredTools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// truncateDescription truncates a description to maxLen characters
func truncateDescription(desc string, maxLen int) string {
	// Remove markdown formatting for brief descriptions
	desc = strings.ReplaceAll(desc, "**", "")
	desc = strings.ReplaceAll(desc, "`", "")

	// Take first line or sentence
	lines := strings.Split(desc, "\n")
	desc = strings.TrimSpace(lines[0])

	if len(desc) <= maxLen {
		return desc
	}

	// Find last space before maxLen
	truncated := desc[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLen/2 {
		return truncated[:lastSpace] + "..."
	}
	return truncated + "..."
}

// SearchToolsTool searches for tools by query, returning minimal info (no schemas).
// This is the first step in the dynamic toolset pattern.
type SearchToolsTool struct {
	*BaseTool
}

// NewSearchToolsTool creates a new SearchToolsTool
func NewSearchToolsTool(c *client.Client, l *zap.Logger) *SearchToolsTool {
	return &SearchToolsTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *SearchToolsTool) Name() string { return "search_tools" }

// Description returns a concise description
func (t *SearchToolsTool) Description() string {
	return "Search available tools by intent or category. Returns tool names and brief descriptions only - use describe_tools for full schemas."
}

// InputSchema returns the input schema
func (t *SearchToolsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Natural language search query (e.g., 'investigate errors', 'create alert')",
			},
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Filter by category",
				"enum":        []string{"query", "alert", "dashboard", "policy", "webhook", "e2m", "stream", "workflow", "meta"},
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Max results to return (default: 10)",
				"default":     10,
			},
		},
	}
}

// Annotations returns tool annotations
func (t *SearchToolsTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("Search Tools")
}

// Execute searches tools and returns brief results
func (t *SearchToolsTool) Execute(_ context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	query, _ := GetStringParam(args, "query", false)
	category, _ := GetStringParam(args, "category", false)
	limit, _ := GetIntParam(args, "limit", false)

	if limit <= 0 {
		limit = 10
	}
	if limit > 25 {
		limit = 25 // Cap to prevent excessive results
	}

	if query == "" && category == "" {
		return NewToolResultError("Provide 'query' (what you want to do) or 'category' to search"), nil
	}

	// Use existing discovery logic but return minimal info
	registry := GetToolRegistry()
	result := registry.DiscoverTools(query, ToolCategory(category), "")

	// Convert to brief format
	briefs := make([]ToolBrief, 0, len(result.MatchedTools))
	for i, match := range result.MatchedTools {
		if i >= limit {
			break
		}

		cat := ""
		if len(match.Categories) > 0 {
			cat = match.Categories[0]
		}

		// Get actual tool description and truncate
		desc := match.Name // fallback
		if tool := GetRegisteredTool(match.Name); tool != nil {
			desc = truncateDescription(tool.Description(), 100)
		}

		briefs = append(briefs, ToolBrief{
			Name:        match.Name,
			Namespace:   GetToolNamespace(match.Name),
			Description: desc,
			Category:    cat,
			Complexity:  match.Complexity,
		})
	}

	response := map[string]interface{}{
		"tools":       briefs,
		"total_found": len(result.MatchedTools),
		"showing":     len(briefs),
		"hint":        "Use describe_tools to get full schemas for tools you want to use",
	}

	// Add confidence info if available
	if result.Confidence != nil {
		response["confidence"] = result.Confidence.Level
		if len(result.Confidence.Clarifications) > 0 {
			response["clarifications"] = result.Confidence.Clarifications
		}
	}

	output, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(output)},
		},
	}, nil
}

// DescribeToolsTool returns full schemas for specified tools.
// This is the second step - only called for tools the LLM intends to use.
type DescribeToolsTool struct {
	*BaseTool
}

// NewDescribeToolsTool creates a new DescribeToolsTool
func NewDescribeToolsTool(c *client.Client, l *zap.Logger) *DescribeToolsTool {
	return &DescribeToolsTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *DescribeToolsTool) Name() string { return "describe_tools" }

// Description returns a concise description
func (t *DescribeToolsTool) Description() string {
	return "Get full schemas and documentation for specific tools. Call this before using a tool to understand its parameters."
}

// InputSchema returns the input schema
func (t *DescribeToolsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"names": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Tool names to describe (from search_tools results)",
				"maxItems":    5, // Limit to prevent loading too many schemas
			},
		},
		"required": []string{"names"},
	}
}

// Annotations returns tool annotations
func (t *DescribeToolsTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("Describe Tools")
}

// Execute returns full schemas for requested tools
func (t *DescribeToolsTool) Execute(_ context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	namesRaw, ok := args["names"]
	if !ok {
		return NewToolResultError("'names' parameter is required"), nil
	}

	// Parse names array
	var names []string
	switch v := namesRaw.(type) {
	case []interface{}:
		for _, n := range v {
			if s, ok := n.(string); ok {
				names = append(names, s)
			}
		}
	case []string:
		names = v
	default:
		return NewToolResultError("'names' must be an array of strings"), nil
	}

	if len(names) == 0 {
		return NewToolResultError("Provide at least one tool name"), nil
	}
	if len(names) > 5 {
		names = names[:5] // Limit to 5 tools
	}

	schemas := make(map[string]*ToolSchema)
	notFound := []string{}

	for _, name := range names {
		tool := GetRegisteredTool(name)
		if tool == nil {
			notFound = append(notFound, name)
			continue
		}

		schema := &ToolSchema{
			Name:        tool.Name(),
			Namespace:   GetToolNamespace(name),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
			Annotations: tool.Annotations(),
			CostHints:   GetCostHints(name),
		}

		// Add metadata if available
		if et, ok := tool.(EnhancedTool); ok {
			schema.Metadata = et.Metadata()
		}

		schemas[name] = schema
	}

	response := map[string]interface{}{
		"tools": schemas,
	}
	if len(notFound) > 0 {
		response["not_found"] = notFound
		response["hint"] = "Use search_tools to find valid tool names"
	}

	output, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(output)},
		},
	}, nil
}

// ListToolCategoriesBrief returns a brief overview of tool categories.
// Useful for initial exploration without loading any schemas.
type ListToolCategoriesBrief struct {
	*BaseTool
}

// NewListToolCategoriesBrief creates a new tool
func NewListToolCategoriesBrief(c *client.Client, l *zap.Logger) *ListToolCategoriesBrief {
	return &ListToolCategoriesBrief{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ListToolCategoriesBrief) Name() string { return "list_tool_categories" }

// Description returns a concise description
func (t *ListToolCategoriesBrief) Description() string {
	return "List available tool categories with counts. Use search_tools with a category to explore."
}

// InputSchema returns the input schema
func (t *ListToolCategoriesBrief) InputSchema() interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// Annotations returns tool annotations
func (t *ListToolCategoriesBrief) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("List Categories")
}

// Execute lists tool categories and namespaces
func (t *ListToolCategoriesBrief) Execute(_ context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	// Get dynamic namespace counts
	namespaces := GetAllNamespaces()
	namespaceInfo := make(map[string]interface{})
	for ns, count := range namespaces {
		info := GetNamespaceInfo(ns, false)
		namespaceInfo[string(ns)] = map[string]interface{}{
			"count":       count,
			"description": info.Description,
		}
	}

	// Legacy category mapping for backward compatibility
	categories := map[string]int{
		"query":       6,  // query_logs, build_query, background queries, etc.
		"alert":       10, // alerts and alert definitions
		"dashboard":   14, // dashboards and folders
		"policy":      5,  // TCO policies
		"webhook":     5,  // outgoing webhooks
		"e2m":         5,  // events to metrics
		"stream":      5,  // log streaming
		"workflow":    2,  // investigate_incident, health_check
		"meta":        5,  // search_tools, describe_tools, discover_tools, etc.
		"view":        10, // views and folders
		"rule":        5,  // rule groups
		"enrichment":  5,  // enrichments
		"data_access": 5,  // data access rules
	}

	response := map[string]interface{}{
		"namespaces":  namespaceInfo,
		"categories":  categories, // Kept for backward compatibility
		"total_tools": len(registeredTools),
		"usage": map[string]string{
			"by_namespace": "search_tools(category='queries') - browse by namespace",
			"by_intent":    "search_tools(query='investigate errors') - search by intent",
			"get_details":  "describe_tools(names=['query_logs']) - get full schema",
		},
	}

	output, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(output)},
		},
	}, nil
}

// GetToolSummary returns a one-line summary for response formatting
func GetToolSummary(toolName string) string {
	tool := GetRegisteredTool(toolName)
	if tool == nil {
		return toolName
	}
	return truncateDescription(tool.Description(), 80)
}
