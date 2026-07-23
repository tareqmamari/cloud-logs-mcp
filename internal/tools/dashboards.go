package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
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
func NewListDashboardsTool(client client.Doer, logger *zap.Logger) *ListDashboardsTool {
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
	session := GetSessionFromContext(ctx)

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
func NewGetDashboardTool(client client.Doer, logger *zap.Logger) *GetDashboardTool {
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
	session := GetSessionFromContext(ctx)

	dashboardID, ok := arguments["dashboard_id"].(string)
	if !ok || dashboardID == "" {
		session.RecordToolUse(t.Name(), false, arguments)
		return NewToolResultError("dashboard_id is required and must be a string"), nil
	}

	req := &client.Request{
		Method: "GET",
		Path:   apiPath("/v1/dashboards", dashboardID),
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
func NewCreateDashboardTool(client client.Doer, logger *zap.Logger) *CreateDashboardTool {
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
- widgets: Visualizations (line_chart, bar_chart, pie_chart, data_table, gauge, markdown)

**Tier selection:** each widget's data_mode_type (the query tier) is chosen
automatically from the TCO policy that routes the widget's target
application/subsystem — "archive" for logs kept only in the archive tier,
"high_unspecified" (Priority Insights / frequent_search) otherwise — so panels
query the tier where the data actually lives. Set data_mode_type explicitly on
a widget to override. The chosen tiers are reported under _tier_selection.`
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
											"appearance": map[string]interface{}{
												"width": 0,
											},
											"definition": map[string]interface{}{
												"line_chart": map[string]interface{}{
													"legend": map[string]interface{}{
														"is_visible":     true,
														"group_by_query": true,
													},
													"tooltip": map[string]interface{}{
														"show_labels": false,
														"type":        "all",
													},
													"query_definitions": []map[string]interface{}{
														{
															"id":                 "query-uuid-1",
															"is_visible":         true,
															"name":               "Query1",
															"color_scheme":       "classic",
															"data_mode_type":     "high_unspecified",
															"scale_type":         "linear",
															"unit":               "unspecified",
															"series_count_limit": "20",
															"resolution": map[string]interface{}{
																"buckets_presented": 96,
															},
															"query": map[string]interface{}{
																"logs": map[string]interface{}{
																	"lucene_query": map[string]interface{}{
																		"value": "severity:>=5",
																	},
																	"aggregations": []map[string]interface{}{
																		{"count": map[string]interface{}{}},
																	},
																	"filters": []map[string]interface{}{},
																	"group_bys": []map[string]interface{}{
																		{"keypath": []string{"severity"}, "scope": "metadata"},
																	},
																},
															},
														},
													},
													"stacked_line": "unspecified",
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

// ensureRequiredDashboardFields adds missing required fields to dashboard structure
// ensureQueryDefinitionUUID replaces a missing or non-UUID query-definition
// id with a generated UUID. Unlike section/row/widget ids, these are plain
// strings, but the API applies the same UUID requirement.
func ensureQueryDefinitionUUID(qd map[string]interface{}) {
	if val, ok := qd["id"].(string); ok && val != "" {
		if _, err := uuid.Parse(val); err == nil {
			return
		}
	}
	qd["id"] = uuid.NewString()
}

// ensureUUIDID makes sure m carries an API-acceptable id of the shape
// {"id": {"value": "<uuid>"}}. The live API rejects non-UUID id values
// (e.g. "section-1"), so missing or invalid ids are replaced with a
// generated UUID.
func ensureUUIDID(m map[string]interface{}) {
	if idMap, ok := m["id"].(map[string]interface{}); ok {
		if val, ok := idMap["value"].(string); ok {
			if _, err := uuid.Parse(val); err == nil {
				return
			}
		}
	}
	m["id"] = map[string]interface{}{"value": uuid.NewString()}
}

// logsQueryOf returns the "logs" query map nested under a node's "query"
// field (node["query"]["logs"]), or nil when the node has no logs query.
func logsQueryOf(node map[string]interface{}) map[string]interface{} {
	query, ok := node["query"].(map[string]interface{})
	if !ok {
		return nil
	}
	logs, _ := query["logs"].(map[string]interface{})
	return logs
}

func ensureRequiredDashboardFields(layout interface{}, resolver *widgetTierResolver, advisor *vizAdvisor) {
	layoutMap, ok := layout.(map[string]interface{})
	if !ok {
		return
	}

	sections, ok := layoutMap["sections"].([]interface{})
	if !ok {
		return
	}

	for _, section := range sections {
		sectionMap, ok := section.(map[string]interface{})
		if !ok {
			continue
		}
		ensureUUIDID(sectionMap)

		rows, ok := sectionMap["rows"].([]interface{})
		if !ok {
			continue
		}

		for _, row := range rows {
			rowMap, ok := row.(map[string]interface{})
			if !ok {
				continue
			}
			ensureUUIDID(rowMap)

			widgets, ok := rowMap["widgets"].([]interface{})
			if !ok {
				continue
			}

			for _, widget := range widgets {
				widgetMap, ok := widget.(map[string]interface{})
				if !ok {
					continue
				}
				ensureUUIDID(widgetMap)

				// Ensure widget has appearance.width
				if _, hasAppearance := widgetMap["appearance"]; !hasAppearance {
					widgetMap["appearance"] = map[string]interface{}{"width": 0}
				} else if appearance, ok := widgetMap["appearance"].(map[string]interface{}); ok {
					if _, hasWidth := appearance["width"]; !hasWidth {
						appearance["width"] = 0
					}
				}

				definition, ok := widgetMap["definition"].(map[string]interface{})
				if !ok {
					continue
				}

				// Advise/adopt the visualization type from the query shape
				// before the per-type field defaults run. adviseWidgetType is
				// a no-op when advisor is nil or the widget has no logs query.
				adviseWidgetType(definition, advisor)

				// Handle different widget types
				if lineChart, ok := definition["line_chart"].(map[string]interface{}); ok {
					ensureLineChartFields(lineChart, resolver)
				}
				if dataTable, ok := definition["data_table"].(map[string]interface{}); ok {
					ensureDataTableFields(dataTable, resolver)
				}
				if pieChart, ok := definition["pie_chart"].(map[string]interface{}); ok {
					ensureChartFields(pieChart, resolver)
				}
				if barChart, ok := definition["bar_chart"].(map[string]interface{}); ok {
					ensureChartFields(barChart, resolver)
				}
				if gauge, ok := definition["gauge"].(map[string]interface{}); ok {
					ensureGaugeFields(gauge, resolver)
				}
			}
		}
	}
}

// widgetDefinitionKeys are the definition keys that hold a chart widget.
var widgetDefinitionKeys = []string{"line_chart", "data_table", "pie_chart", "bar_chart", "gauge"}

// adviseWidgetType records a visualization recommendation for a widget when
// its chosen chart type clearly mismatches its query shape. It is strictly
// non-destructive: it never rewrites the caller's widget type. A widget with
// no chart-type wrapper is skipped — a widget's logs query always lives inside
// a widget-type definition, so there is no canonical query to classify and no
// type is synthesized.
func adviseWidgetType(definition map[string]interface{}, advisor *vizAdvisor) {
	if advisor == nil {
		return
	}
	current, node := currentWidgetType(definition)
	if current == "" || node == nil {
		return
	}
	logs := logsQueryOf(node)
	if logs == nil {
		return
	}
	advisor.advise(current, logs)
}

// currentWidgetType returns the first present chart widget key and its
// definition node, or ("", nil) if none is set.
func currentWidgetType(definition map[string]interface{}) (string, map[string]interface{}) {
	for _, key := range widgetDefinitionKeys {
		if node, ok := definition[key].(map[string]interface{}); ok {
			return key, node
		}
	}
	return "", nil
}

// aggregationsNeedingField are logs-aggregation types the API requires an
// observation_field for (a value aggregation is meaningless without one).
var aggregationsNeedingField = map[string]bool{
	"average": true, "sum": true, "min": true, "max": true,
	"percentile": true, "count_distinct": true,
}

// validateDashboardStructure checks constraints the live API enforces on
// dashboard layouts that JSON shape alone does not capture: line-chart logs
// queries must carry at least one aggregation, and value aggregations must
// name an observation_field.
func validateDashboardStructure(layout interface{}) []string {
	var errs []string
	layoutMap, ok := layout.(map[string]interface{})
	if !ok {
		return errs
	}
	sections, _ := layoutMap["sections"].([]interface{})
	for si, section := range sections {
		sectionMap, ok := section.(map[string]interface{})
		if !ok {
			continue
		}
		rows, _ := sectionMap["rows"].([]interface{})
		for ri, row := range rows {
			rowMap, ok := row.(map[string]interface{})
			if !ok {
				continue
			}
			widgets, _ := rowMap["widgets"].([]interface{})
			for wi, widget := range widgets {
				widgetMap, ok := widget.(map[string]interface{})
				if !ok {
					continue
				}
				definition, _ := widgetMap["definition"].(map[string]interface{})
				lineChart, ok := definition["line_chart"].(map[string]interface{})
				if !ok {
					continue
				}
				queryDefs, _ := lineChart["query_definitions"].([]interface{})
				for qi, qd := range queryDefs {
					qdMap, ok := qd.(map[string]interface{})
					if !ok {
						continue
					}
					query, _ := qdMap["query"].(map[string]interface{})
					logs, ok := query["logs"].(map[string]interface{})
					if !ok {
						continue
					}
					at := fmt.Sprintf("sections[%d].rows[%d].widgets[%d].query_definitions[%d]", si, ri, wi, qi)
					aggs, _ := logs["aggregations"].([]interface{})
					if len(aggs) == 0 {
						errs = append(errs, at+": line-chart logs query needs at least one aggregation (e.g. {\"count\": {}})")
						continue
					}
					for ai, agg := range aggs {
						aggMap, ok := agg.(map[string]interface{})
						if !ok {
							continue
						}
						for kind, body := range aggMap {
							if !aggregationsNeedingField[kind] {
								continue
							}
							bodyMap, _ := body.(map[string]interface{})
							if _, has := bodyMap["observation_field"]; !has {
								errs = append(errs, fmt.Sprintf("%s.aggregations[%d]: %q aggregation requires an observation_field", at, ai, kind))
							}
						}
					}
				}
			}
		}
	}
	return errs
}

// ensureLineChartFields ensures line chart has all required fields
func ensureLineChartFields(lineChart map[string]interface{}, resolver *widgetTierResolver) {
	// Ensure legend
	if _, hasLegend := lineChart["legend"]; !hasLegend {
		lineChart["legend"] = map[string]interface{}{
			"is_visible":     true,
			"group_by_query": true,
		}
	}

	// Ensure tooltip
	if _, hasTooltip := lineChart["tooltip"]; !hasTooltip {
		lineChart["tooltip"] = map[string]interface{}{
			"show_labels": false,
			"type":        "all",
		}
	}

	// Ensure stacked_line
	if _, hasStackedLine := lineChart["stacked_line"]; !hasStackedLine {
		lineChart["stacked_line"] = "unspecified"
	}

	// Ensure query_definitions have required fields
	if queryDefs, ok := lineChart["query_definitions"].([]interface{}); ok {
		for _, qd := range queryDefs {
			if qdMap, ok := qd.(map[string]interface{}); ok {
				ensureQueryDefinitionFields(qdMap, resolver)
				ensureQueryDefinitionUUID(qdMap)
			}
		}
	}
}

// ensureDataModeType fills in a node's data_mode_type from the TCO-derived
// tier for its logs query, but only when the caller did not set it — an
// explicit value always wins.
func ensureDataModeType(node map[string]interface{}, resolver *widgetTierResolver) {
	if _, hasDataMode := node["data_mode_type"]; hasDataMode {
		return
	}
	node["data_mode_type"] = resolver.dataModeTypeFor(logsQueryOf(node))
}

// ensureDataTableFields ensures data table has required fields
func ensureDataTableFields(dataTable map[string]interface{}, resolver *widgetTierResolver) {
	ensureDataModeType(dataTable, resolver)

	// Ensure query has filters array
	if query, ok := dataTable["query"].(map[string]interface{}); ok {
		ensureQueryFilters(query)
	}
}

// ensureChartFields ensures pie/bar charts have required fields
func ensureChartFields(chart map[string]interface{}, resolver *widgetTierResolver) {
	ensureDataModeType(chart, resolver)

	// Ensure query has filters array
	if query, ok := chart["query"].(map[string]interface{}); ok {
		ensureQueryFilters(query)
	}
}

// ensureGaugeFields ensures gauge has required fields
func ensureGaugeFields(gauge map[string]interface{}, resolver *widgetTierResolver) {
	ensureDataModeType(gauge, resolver)

	// Ensure query has filters array
	if query, ok := gauge["query"].(map[string]interface{}); ok {
		ensureQueryFilters(query)
	}
}

// ensureQueryDefinitionFields ensures query definition has all required fields
func ensureQueryDefinitionFields(queryDef map[string]interface{}, resolver *widgetTierResolver) {
	// Ensure is_visible
	if _, hasVisible := queryDef["is_visible"]; !hasVisible {
		queryDef["is_visible"] = true
	}

	// Ensure data_mode_type (tier) from TCO policy for the query's app/subsystem
	ensureDataModeType(queryDef, resolver)

	// Ensure scale_type
	if _, hasScale := queryDef["scale_type"]; !hasScale {
		queryDef["scale_type"] = "linear"
	}

	// Ensure unit
	if _, hasUnit := queryDef["unit"]; !hasUnit {
		queryDef["unit"] = "unspecified"
	}

	// Ensure resolution
	if _, hasResolution := queryDef["resolution"]; !hasResolution {
		queryDef["resolution"] = map[string]interface{}{
			"buckets_presented": 96,
		}
	} else if resolution, ok := queryDef["resolution"].(map[string]interface{}); ok {
		if _, hasBuckets := resolution["buckets_presented"]; !hasBuckets {
			resolution["buckets_presented"] = 96
		}
	}

	// Ensure query has filters array
	if query, ok := queryDef["query"].(map[string]interface{}); ok {
		ensureQueryFilters(query)
	}
}

// ensureQueryFilters ensures query has filters array
func ensureQueryFilters(query map[string]interface{}) {
	// Check logs query
	if logs, ok := query["logs"].(map[string]interface{}); ok {
		if _, hasFilters := logs["filters"]; !hasFilters {
			logs["filters"] = []interface{}{}
		}
	}

	// Check metrics query
	if metrics, ok := query["metrics"].(map[string]interface{}); ok {
		if _, hasFilters := metrics["filters"]; !hasFilters {
			metrics["filters"] = []interface{}{}
		}
	}

	// Check dataprime query
	if dataprime, ok := query["dataprime"].(map[string]interface{}); ok {
		if _, hasFilters := dataprime["filters"]; !hasFilters {
			dataprime["filters"] = []interface{}{}
		}
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

	// Ensure all required fields are present in the layout. The resolver
	// selects each widget's tier (data_mode_type) from the TCO policy that
	// routes the widget's target application/subsystem, so panels query the
	// tier where those logs actually live.
	resolver := resolverFromContext(ctx, t.client, t.logger)
	advisor := newVizAdvisor()
	ensureRequiredDashboardFields(layout, resolver, advisor)

	// Check structural constraints the API enforces beyond JSON shape
	structureErrors := validateDashboardStructure(layout)
	if len(structureErrors) > 0 && !dryRun {
		return NewToolResultError(fmt.Sprintf("Dashboard layout has %d structural error(s). Please fix them before creating the dashboard:\n- %s",
			len(structureErrors), joinErrors(structureErrors))), nil
	}

	// Extract queries with syntax information for better validation
	queryInfos := extractQueriesWithInfo(layout, "layout")
	invalidQueries := structureErrors
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

	attachTierSelection(result, resolver)
	attachVizRecommendations(result, advisor)
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
func NewUpdateDashboardTool(client client.Doer, logger *zap.Logger) *UpdateDashboardTool {
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

	// Normalize the layout and select each widget's tier (data_mode_type)
	// from the TCO policy for its target application/subsystem, mirroring
	// create so an updated dashboard queries the tier where its logs live.
	resolver := resolverFromContext(ctx, t.client, t.logger)
	advisor := newVizAdvisor()
	ensureRequiredDashboardFields(layout, resolver, advisor)

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
		Path:   apiPath("/v1/dashboards", dashboardID),
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	attachTierSelection(result, resolver)
	attachVizRecommendations(result, advisor)
	return t.FormatResponseWithSuggestions(result, "update_dashboard")
}

// DeleteDashboardTool deletes a dashboard.
type DeleteDashboardTool struct {
	*BaseTool
}

// NewDeleteDashboardTool creates a new DeleteDashboardTool instance.
func NewDeleteDashboardTool(client client.Doer, logger *zap.Logger) *DeleteDashboardTool {
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

	req := &client.Request{
		Method: "DELETE",
		Path:   apiPath("/v1/dashboards", dashboardID),
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponseWithSuggestions(result, "delete_dashboard")
}
