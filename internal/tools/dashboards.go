package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// QueryInfo holds information about an extracted query for validation
type QueryInfo struct {
	Query  string
	Syntax string // "dataprime" or "lucene"
	Source string // description of where the query was found
}

// extractQueriesFromLayout recursively extracts all query strings from a dashboard layout.
// It searches through sections, rows, and widgets to find dataprime or lucene queries.
func extractQueriesFromLayout(layout interface{}) []string {
	queryInfos := extractQueriesWithInfo(layout, "")
	var queries []string
	for _, qi := range queryInfos {
		queries = append(queries, qi.Query)
	}
	return queries
}

// extractQueriesWithInfo extracts queries with metadata about their type and location
func extractQueriesWithInfo(layout interface{}, path string) []QueryInfo {
	var queries []QueryInfo

	switch v := layout.(type) {
	case map[string]interface{}:
		queries = append(queries, extractQueriesFromMap(v, path)...)
	case []interface{}:
		for i, item := range v {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			queries = append(queries, extractQueriesWithInfo(item, itemPath)...)
		}
	}

	return queries
}

// extractQueriesFromMap extracts queries from a map structure
func extractQueriesFromMap(v map[string]interface{}, path string) []QueryInfo {
	var queries []QueryInfo

	// Extract direct queries
	queries = append(queries, extractDirectQueries(v, path)...)

	// Extract from definition (chart widgets)
	queries = append(queries, extractDefinitionQueries(v, path)...)

	// Extract from query_definitions array
	queries = append(queries, extractQueryDefinitions(v, path)...)

	// Recurse into nested structures
	queries = append(queries, extractNestedQueries(v, path)...)

	return queries
}

// extractDirectQueries extracts direct query fields from a map
func extractDirectQueries(v map[string]interface{}, path string) []QueryInfo {
	var queries []QueryInfo

	// Check for direct query field (common in DataPrime queries)
	if query, ok := v["query"].(string); ok && query != "" {
		syntax := "dataprime"
		if _, isLucene := v["lucene_query"]; isLucene {
			syntax = "lucene"
		}
		queries = append(queries, QueryInfo{Query: query, Syntax: syntax, Source: path + ".query"})
	}

	// Check for dataprime_query structure
	if dpQuery, ok := v["dataprime_query"].(map[string]interface{}); ok {
		if text, ok := dpQuery["text"].(string); ok && text != "" {
			queries = append(queries, QueryInfo{Query: text, Syntax: "dataprime", Source: path + ".dataprime_query.text"})
		}
	}

	// Check for lucene_query structure
	if luceneQuery, ok := v["lucene_query"].(map[string]interface{}); ok {
		if value, ok := luceneQuery["value"].(string); ok && value != "" {
			queries = append(queries, QueryInfo{Query: value, Syntax: "lucene", Source: path + ".lucene_query.value"})
		}
	}

	return queries
}

// extractDefinitionQueries extracts queries from widget definition blocks
func extractDefinitionQueries(v map[string]interface{}, path string) []QueryInfo {
	var queries []QueryInfo

	definition, ok := v["definition"].(map[string]interface{})
	if !ok {
		return queries
	}

	if dataprime, ok := definition["dataprime"].(map[string]interface{}); ok {
		if query, ok := dataprime["query"].(string); ok && query != "" {
			queries = append(queries, QueryInfo{Query: query, Syntax: "dataprime", Source: path + ".definition.dataprime.query"})
		}
	}
	if lucene, ok := definition["lucene"].(map[string]interface{}); ok {
		if query, ok := lucene["query"].(string); ok && query != "" {
			queries = append(queries, QueryInfo{Query: query, Syntax: "lucene", Source: path + ".definition.lucene.query"})
		}
	}

	return queries
}

// extractQueryDefinitions extracts queries from query_definitions arrays
func extractQueryDefinitions(v map[string]interface{}, path string) []QueryInfo {
	var queries []QueryInfo

	queryDefs, ok := v["query_definitions"].([]interface{})
	if !ok {
		return queries
	}

	for i, qd := range queryDefs {
		qdPath := fmt.Sprintf("%s.query_definitions[%d]", path, i)
		qdMap, ok := qd.(map[string]interface{})
		if !ok {
			continue
		}
		queryObj, ok := qdMap["query"].(map[string]interface{})
		if !ok {
			continue
		}
		for _, subKey := range []string{"logs", "metrics", "dataprime"} {
			if sub, ok := queryObj[subKey].(map[string]interface{}); ok {
				queries = append(queries, extractQueriesWithInfo(sub, qdPath+".query."+subKey)...)
			}
		}
	}

	return queries
}

// extractNestedQueries recursively extracts queries from nested map values
func extractNestedQueries(v map[string]interface{}, path string) []QueryInfo {
	var queries []QueryInfo

	skipKeys := map[string]bool{
		"query": true, "dataprime_query": true, "lucene_query": true,
		"definition": true, "query_definitions": true,
	}

	for key, val := range v {
		if skipKeys[key] {
			continue
		}
		newPath := key
		if path != "" {
			newPath = path + "." + key
		}
		queries = append(queries, extractQueriesWithInfo(val, newPath)...)
	}

	return queries
}

// validateQuery tests a query by executing it with a minimal time range.
// Returns an error if the query is invalid.
// It auto-detects whether the query is DataPrime or Lucene syntax.
func (t *BaseTool) validateQuery(ctx context.Context, query string) error {
	return t.validateQueryWithSyntax(ctx, query, "")
}

// validateQueryWithSyntax tests a query with a specified syntax type.
// If syntax is empty, it will auto-detect based on query patterns.
func (t *BaseTool) validateQueryWithSyntax(ctx context.Context, query string, syntax string) error {
	// Auto-detect syntax if not specified
	if syntax == "" {
		syntax = detectQuerySyntax(query)
	}

	// Use a very short time range (1 minute) to minimize data scanned
	now := time.Now().UTC()
	startDate := now.Add(-1 * time.Minute).Format(time.RFC3339)
	endDate := now.Format(time.RFC3339)

	body := map[string]interface{}{
		"query": query,
		"metadata": map[string]interface{}{
			"tier":       "frequent_search",
			"syntax":     syntax,
			"start_date": startDate,
			"end_date":   endDate,
			"limit":      1, // Only need 1 result to validate syntax
		},
	}

	req := &client.Request{
		Method:    "POST",
		Path:      "/v1/query",
		Body:      body,
		AcceptSSE: true,
	}

	_, err := t.ExecuteRequest(ctx, req)
	return err
}

// detectQuerySyntax determines if a query is DataPrime or Lucene based on patterns.
// DataPrime queries typically start with "source" or contain pipe operators "|".
// Lucene queries use field:value syntax without DataPrime operators.
func detectQuerySyntax(query string) string {
	// DataPrime indicators:
	// - Starts with "source" command
	// - Contains pipe operators for chaining
	// - Uses DataPrime functions like "filter", "groupby", "aggregate"
	dataprimeIndicators := []string{
		"source ",
		"source\n",
		"| filter",
		"|filter",
		"| groupby",
		"|groupby",
		"| aggregate",
		"|aggregate",
		"| limit",
		"|limit",
		"| sort",
		"|sort",
		"| top",
		"|top",
		"| count",
		"|count",
		"$d.",        // DataPrime data field reference
		"$l.",        // DataPrime label field reference
		"$m.",        // DataPrime metadata field reference
		".contains(", // DataPrime string contains method
		".matches(",  // DataPrime regex match method
	}

	queryLower := query
	for _, indicator := range dataprimeIndicators {
		if contains(queryLower, indicator) {
			return "dataprime"
		}
	}

	// Default to lucene for simple field:value queries
	return "lucene"
}

// contains checks if a string contains a substring (case-insensitive for some indicators)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// truncateQuery truncates a query string to a maximum length for display
func truncateQuery(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen] + "..."
}

// ListDashboardsTool lists all dashboards in the catalog.
type ListDashboardsTool struct {
	*BaseTool
}

// NewListDashboardsTool creates a new ListDashboardsTool instance.
func NewListDashboardsTool(client *client.Client, logger *zap.Logger) *ListDashboardsTool {
	return &ListDashboardsTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *ListDashboardsTool) Name() string {
	return "list_dashboards"
}

// Description returns a human-readable description of the tool.
func (t *ListDashboardsTool) Description() string {
	return "List all dashboards in the IBM Cloud Logs dashboard catalog"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *ListDashboardsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// Metadata returns semantic metadata for AI-driven discovery
func (t *ListDashboardsTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:   []ToolCategory{CategoryDashboard, CategoryDiscovery, CategoryVisualization},
		Keywords:     []string{"dashboards", "list", "all", "visualizations", "charts", "monitoring"},
		Complexity:   ComplexitySimple,
		UseCases:     []string{"View all dashboards", "Find visualization", "Audit dashboard setup"},
		RelatedTools: []string{"get_dashboard", "create_dashboard", "pin_dashboard"},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"dashboards": map[string]interface{}{
					"type":  "array",
					"items": map[string]string{"type": "object"},
				},
			},
		},
		ChainPosition: ChainStart,
	}
}

// Execute lists all dashboards.
func (t *ListDashboardsTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	session := GetSession()

	req := &client.Request{
		Method: "GET",
		Path:   "/v1/dashboards",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		session.RecordToolUse(t.Name(), false, arguments)
		return NewToolResultError(err.Error()), nil
	}

	// Record successful tool use and cache result
	session.RecordToolUse(t.Name(), true, arguments)
	session.CacheResult(t.Name(), result)

	return t.FormatResponseWithSuggestions(result, "list_dashboards")
}

// GetDashboardTool gets a specific dashboard by ID.
type GetDashboardTool struct {
	*BaseTool
}

// NewGetDashboardTool creates a new GetDashboardTool instance.
func NewGetDashboardTool(client *client.Client, logger *zap.Logger) *GetDashboardTool {
	return &GetDashboardTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *GetDashboardTool) Name() string {
	return "get_dashboard"
}

// Description returns a human-readable description of the tool.
func (t *GetDashboardTool) Description() string {
	return "Get a specific dashboard by ID from IBM Cloud Logs"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *GetDashboardTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"dashboard_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the dashboard",
			},
		},
		"required": []string{"dashboard_id"},
	}
}

// Metadata returns semantic metadata for AI-driven discovery
func (t *GetDashboardTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:   []ToolCategory{CategoryDashboard, CategoryVisualization},
		Keywords:     []string{"dashboard", "get", "retrieve", "fetch", "visualization", "chart"},
		Complexity:   ComplexitySimple,
		UseCases:     []string{"View dashboard details", "Inspect dashboard widgets", "Get dashboard configuration"},
		RelatedTools: []string{"list_dashboards", "update_dashboard", "delete_dashboard"},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id":          map[string]string{"type": "string"},
				"name":        map[string]string{"type": "string"},
				"description": map[string]string{"type": "string"},
				"layout":      map[string]string{"type": "object"},
			},
		},
		ChainPosition: ChainMiddle,
	}
}

// Execute gets a specific dashboard.
func (t *GetDashboardTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	session := GetSession()

	dashboardID, ok := arguments["dashboard_id"].(string)
	if !ok || dashboardID == "" {
		session.RecordToolUse(t.Name(), false, arguments)
		return NewToolResultError("dashboard_id is required and must be a string"), nil
	}

	req := &client.Request{
		Method: "GET",
		Path:   fmt.Sprintf("/v1/dashboards/%s", dashboardID),
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		session.RecordToolUse(t.Name(), false, arguments)
		return HandleGetError(err, "Dashboard", dashboardID, "list_dashboards"), nil
	}

	// Record successful tool use and cache result
	session.RecordToolUse(t.Name(), true, map[string]interface{}{"dashboard_id": dashboardID})
	session.CacheResult(t.Name(), result)

	return t.FormatResponseWithSuggestions(result, "get_dashboard")
}

// CreateDashboardTool creates a new dashboard.
type CreateDashboardTool struct {
	*BaseTool
}

// NewCreateDashboardTool creates a new CreateDashboardTool instance.
func NewCreateDashboardTool(client *client.Client, logger *zap.Logger) *CreateDashboardTool {
	return &CreateDashboardTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *CreateDashboardTool) Name() string {
	return "create_dashboard"
}

// Description returns a human-readable description of the tool.
func (t *CreateDashboardTool) Description() string {
	return `Create a new dashboard in IBM Cloud Logs with widgets and layout configuration.

**Related tools:** list_dashboards, get_dashboard, update_dashboard, pin_dashboard, move_dashboard_to_folder

**Dashboard Structure:**
- sections: Array of dashboard sections (logical groupings)
- rows: Horizontal containers within sections (each has height)
- widgets: Visualizations (line_chart, bar_chart, pie_chart, data_table, gauge, markdown)`
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *CreateDashboardTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Display name for the dashboard",
				"examples":    []string{"API Errors Dashboard", "Production Monitoring", "Service Health"},
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Optional description of the dashboard purpose",
				"examples":    []string{"Monitors API error rates and response times", "Overview of production service health metrics"},
			},
			"layout": map[string]interface{}{
				"type":        "object",
				"description": "Dashboard layout configuration with sections and widgets",
				"example": map[string]interface{}{
					"sections": []map[string]interface{}{
						{
							"id": map[string]interface{}{"value": "section-1"},
							"rows": []map[string]interface{}{
								{
									"id":         map[string]interface{}{"value": "row-1"},
									"appearance": map[string]interface{}{"height": 19},
									"widgets": []map[string]interface{}{
										{
											"id":    map[string]interface{}{"value": "widget-1"},
											"title": "Error Count by Severity",
											"definition": map[string]interface{}{
												"line_chart": map[string]interface{}{
													"query_definitions": []map[string]interface{}{
														{
															"query": map[string]interface{}{
																"logs": map[string]interface{}{
																	"lucene_query": map[string]interface{}{
																		"value": "severity:>=5",
																	},
																	"aggregations": []map[string]interface{}{
																		{"count": map[string]interface{}{}},
																	},
																	"group_bys": []map[string]interface{}{
																		{"keypath": []string{"severity"}, "scope": "metadata"},
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
			},
			"dry_run": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, validates the dashboard configuration without creating it. Use this to preview what will be created and check for errors.",
				"default":     false,
			},
		},
		"required": []string{"name", "layout"},
	}
}

// Metadata returns semantic metadata for AI-driven discovery
func (t *CreateDashboardTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:   []ToolCategory{CategoryDashboard, CategoryVisualization, CategoryConfiguration},
		Keywords:     []string{"dashboard", "create", "new", "visualization", "chart", "widget", "build"},
		Complexity:   ComplexityAdvanced,
		UseCases:     []string{"Create monitoring dashboard", "Build visualization", "Set up charts"},
		RelatedTools: []string{"list_dashboards", "get_dashboard", "query_logs", "update_dashboard"},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id":   map[string]string{"type": "string"},
				"name": map[string]string{"type": "string"},
			},
		},
		ChainPosition: ChainEnd,
	}
}

// Execute creates a new dashboard.
// It first validates all queries in the layout to ensure they are syntactically correct.
func (t *CreateDashboardTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	name, ok := arguments["name"].(string)
	if !ok || name == "" {
		return NewToolResultError("name is required and must be a string"), nil
	}

	layout, ok := arguments["layout"]
	if !ok {
		return NewToolResultError("layout is required"), nil
	}

	// Check for dry-run mode
	dryRun, _ := GetBoolParam(arguments, "dry_run", false)

	// Extract queries with syntax information for better validation
	queryInfos := extractQueriesWithInfo(layout, "layout")
	var invalidQueries []string
	var validatedQueries []string

	if len(queryInfos) > 0 {
		t.logger.Info("Validating dashboard queries before creation",
			zap.Int("query_count", len(queryInfos)))

		for _, qi := range queryInfos {
			// Use the detected or specified syntax
			syntax := qi.Syntax
			if syntax == "" {
				syntax = detectQuerySyntax(qi.Query)
			}

			t.logger.Debug("Validating query",
				zap.String("query", qi.Query),
				zap.String("syntax", syntax),
				zap.String("source", qi.Source))

			if err := t.validateQueryWithSyntax(ctx, qi.Query, syntax); err != nil {
				invalidQueries = append(invalidQueries,
					fmt.Sprintf("[%s] Query at %s: '%s' - %s", syntax, qi.Source, truncateQuery(qi.Query, 50), err.Error()))
			} else {
				validatedQueries = append(validatedQueries, qi.Query)
			}
		}

		if len(invalidQueries) > 0 && !dryRun {
			return NewToolResultError(fmt.Sprintf("Dashboard contains %d invalid queries. Please fix them before creating the dashboard:\n- %s",
				len(invalidQueries), joinErrors(invalidQueries))), nil
		}
		if len(invalidQueries) == 0 {
			t.logger.Info("All dashboard queries validated successfully",
				zap.Int("valid_count", len(validatedQueries)))
		}
	}

	// If dry-run mode, return validation result with query details
	if dryRun {
		return t.formatDryRunResult(name, arguments, queryInfos, invalidQueries)
	}

	body := map[string]interface{}{
		"name":   name,
		"layout": layout,
	}

	if description, ok := arguments["description"].(string); ok && description != "" {
		body["description"] = description
	}

	req := &client.Request{
		Method: "POST",
		Path:   "/v1/dashboards",
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponseWithSuggestions(result, "create_dashboard")
}

// formatDryRunResult creates a formatted dry-run validation response for dashboards
func (t *CreateDashboardTool) formatDryRunResult(name string, arguments map[string]interface{}, queries []QueryInfo, invalidQueries []string) (*mcp.CallToolResult, error) {
	result := &ValidationResult{
		Valid:   len(invalidQueries) == 0,
		Summary: make(map[string]interface{}),
	}

	result.Summary["name"] = name
	if desc, ok := arguments["description"].(string); ok && desc != "" {
		result.Summary["description"] = desc
	}
	result.Summary["queries_found"] = len(queries)

	// Add query details with syntax information
	if len(queries) > 0 {
		queryDetails := make([]map[string]string, 0, len(queries))
		for _, qi := range queries {
			queryDetails = append(queryDetails, map[string]string{
				"query":  truncateQuery(qi.Query, 80),
				"syntax": qi.Syntax,
				"source": qi.Source,
			})
		}
		result.Summary["query_details"] = queryDetails
	}

	if len(invalidQueries) > 0 {
		result.Errors = invalidQueries
	}

	// Count widgets if possible
	if layout, ok := arguments["layout"].(map[string]interface{}); ok {
		widgetCount := countWidgets(layout)
		result.Summary["widgets"] = widgetCount
	}

	if result.Valid {
		result.Suggestions = []string{
			"Dashboard configuration is valid",
			"Remove dry_run parameter to create the dashboard",
		}
	} else {
		result.Suggestions = []string{
			"Fix the invalid queries listed above",
			"Use query_logs tool to test query syntax",
		}
	}

	config := map[string]interface{}{
		"name":   name,
		"layout": arguments["layout"],
	}
	if desc, ok := arguments["description"].(string); ok {
		config["description"] = desc
	}

	return FormatDryRunResult(result, "Dashboard", config), nil
}

// countWidgets counts the number of widgets in a layout
func countWidgets(layout map[string]interface{}) int {
	count := 0
	if sections, ok := layout["sections"].([]interface{}); ok {
		for _, section := range sections {
			if sectionMap, ok := section.(map[string]interface{}); ok {
				if rows, ok := sectionMap["rows"].([]interface{}); ok {
					for _, row := range rows {
						if rowMap, ok := row.(map[string]interface{}); ok {
							if widgets, ok := rowMap["widgets"].([]interface{}); ok {
								count += len(widgets)
							}
						}
					}
				}
			}
		}
	}
	return count
}

// joinErrors joins error strings with newlines and bullet points
func joinErrors(errors []string) string {
	result := ""
	for i, err := range errors {
		if i > 0 {
			result += "\n- "
		}
		result += err
	}
	return result
}

// UpdateDashboardTool updates an existing dashboard.
type UpdateDashboardTool struct {
	*BaseTool
}

// NewUpdateDashboardTool creates a new UpdateDashboardTool instance.
func NewUpdateDashboardTool(client *client.Client, logger *zap.Logger) *UpdateDashboardTool {
	return &UpdateDashboardTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *UpdateDashboardTool) Name() string {
	return "update_dashboard"
}

// Description returns a human-readable description of the tool.
func (t *UpdateDashboardTool) Description() string {
	return "Update an existing dashboard in IBM Cloud Logs (replaces the entire dashboard)"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *UpdateDashboardTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"dashboard_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the dashboard to update",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Display name for the dashboard",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Optional description of the dashboard purpose",
			},
			"layout": map[string]interface{}{
				"type":        "object",
				"description": "Dashboard layout configuration with sections and widgets",
			},
		},
		"required": []string{"dashboard_id", "name", "layout"},
	}
}

// Metadata returns semantic metadata for AI-driven discovery
func (t *UpdateDashboardTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:   []ToolCategory{CategoryDashboard, CategoryVisualization, CategoryConfiguration},
		Keywords:     []string{"dashboard", "update", "modify", "edit", "change", "visualization"},
		Complexity:   ComplexityAdvanced,
		UseCases:     []string{"Update dashboard layout", "Modify widgets", "Change dashboard configuration"},
		RelatedTools: []string{"get_dashboard", "list_dashboards", "delete_dashboard"},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id":   map[string]string{"type": "string"},
				"name": map[string]string{"type": "string"},
			},
		},
		ChainPosition: ChainEnd,
	}
}

// Execute updates a dashboard.
// It first validates all queries in the layout to ensure they are syntactically correct.
func (t *UpdateDashboardTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	dashboardID, ok := arguments["dashboard_id"].(string)
	if !ok || dashboardID == "" {
		return NewToolResultError("dashboard_id is required and must be a string"), nil
	}

	name, ok := arguments["name"].(string)
	if !ok || name == "" {
		return NewToolResultError("name is required and must be a string"), nil
	}

	layout, ok := arguments["layout"]
	if !ok {
		return NewToolResultError("layout is required"), nil
	}

	// Extract and validate all queries from the layout before updating the dashboard
	queries := extractQueriesFromLayout(layout)
	if len(queries) > 0 {
		t.logger.Info("Validating dashboard queries before update", zap.Int("query_count", len(queries)))
		var invalidQueries []string
		for _, query := range queries {
			if err := t.validateQuery(ctx, query); err != nil {
				invalidQueries = append(invalidQueries, fmt.Sprintf("Query '%s': %s", query, err.Error()))
			}
		}
		if len(invalidQueries) > 0 {
			return NewToolResultError(fmt.Sprintf("Dashboard contains invalid queries. Please fix them before updating the dashboard:\n%s",
				fmt.Sprintf("- %s", joinErrors(invalidQueries)))), nil
		}
		t.logger.Info("All dashboard queries validated successfully")
	}

	body := map[string]interface{}{
		"name":   name,
		"layout": layout,
	}

	if description, ok := arguments["description"].(string); ok && description != "" {
		body["description"] = description
	}

	req := &client.Request{
		Method: "PUT",
		Path:   fmt.Sprintf("/v1/dashboards/%s", dashboardID),
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponseWithSuggestions(result, "update_dashboard")
}

// DeleteDashboardTool deletes a dashboard.
type DeleteDashboardTool struct {
	*BaseTool
}

// NewDeleteDashboardTool creates a new DeleteDashboardTool instance.
func NewDeleteDashboardTool(client *client.Client, logger *zap.Logger) *DeleteDashboardTool {
	return &DeleteDashboardTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *DeleteDashboardTool) Name() string {
	return "delete_dashboard"
}

// Annotations returns tool hints for LLMs
func (t *DeleteDashboardTool) Annotations() *mcp.ToolAnnotations {
	return DeleteAnnotations("Delete Dashboard")
}

// Description returns a human-readable description of the tool.
func (t *DeleteDashboardTool) Description() string {
	return "Delete a dashboard from IBM Cloud Logs"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *DeleteDashboardTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"dashboard_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the dashboard to delete",
			},
			"confirm": ConfirmationInputSchema(),
		},
		"required": []string{"dashboard_id"},
	}
}

// Metadata returns semantic metadata for AI-driven discovery
func (t *DeleteDashboardTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:   []ToolCategory{CategoryDashboard, CategoryConfiguration},
		Keywords:     []string{"dashboard", "delete", "remove", "visualization"},
		Complexity:   ComplexitySimple,
		UseCases:     []string{"Remove obsolete dashboard", "Clean up dashboards"},
		RelatedTools: []string{"get_dashboard", "list_dashboards"},
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"success": map[string]string{"type": "boolean"},
			},
		},
		ChainPosition: ChainEnd,
	}
}

// Execute deletes a dashboard.
func (t *DeleteDashboardTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	dashboardID, ok := arguments["dashboard_id"].(string)
	if !ok || dashboardID == "" {
		return NewToolResultError("dashboard_id is required and must be a string"), nil
	}
	if shouldContinue, result := RequireConfirmation(arguments, "dashboard", dashboardID); !shouldContinue {
		return result, nil
	}

	req := &client.Request{
		Method: "DELETE",
		Path:   fmt.Sprintf("/v1/dashboards/%s", dashboardID),
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponseWithSuggestions(result, "delete_dashboard")
}
