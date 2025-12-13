package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// BuildQueryTool helps users construct log queries without knowing Lucene or DataPrime syntax.
// It takes structured parameters and generates the appropriate query string.
type BuildQueryTool struct {
	*BaseTool
}

// NewBuildQueryTool creates a new BuildQueryTool instance.
func NewBuildQueryTool(client *client.Client, logger *zap.Logger) *BuildQueryTool {
	return &BuildQueryTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *BuildQueryTool) Name() string {
	return "build_query"
}

// Annotations returns tool hints for LLMs
func (t *BuildQueryTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("Build Query")
}

// Description returns a human-readable description of the tool.
func (t *BuildQueryTool) Description() string {
	return `Build a log query from structured parameters without needing to know Lucene or DataPrime syntax.

**Related tools:** query_logs (execute the built query), submit_background_query (for large queries)

This tool helps you construct queries by specifying:
- Text to search for in log messages
- Application and subsystem filters (maps to $l.applicationname, $l.subsystemname in DataPrime)
- Severity levels (debug, info, warning, error, critical)
- Field-value filters for structured log data
- Time-based constraints

**DataPrime Field Prefixes:**
- $l. (labels): applicationname, subsystemname, namespace, pod, container, hostname, etc.
- $m. (metadata): severity, timestamp, priority
- $d. (data): JSON/structured fields from log content (e.g., $d.status_code, $d.user_id)

The tool returns both Lucene and DataPrime versions of the query, which you can use with query_logs.

**When to use this tool:**
- You want to search logs but don't know query syntax
- You need to filter by multiple criteria
- You want to ensure your query is syntactically correct`
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *BuildQueryTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"text_search": map[string]interface{}{
				"type":        "string",
				"description": "Free text to search for in log messages. Searches across all text fields.",
			},
			"applications": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Filter logs by application names (OR logic - matches any)",
			},
			"subsystems": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Filter logs by subsystem/component names (OR logic - matches any)",
			},
			"severities": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string", "enum": []string{"debug", "verbose", "info", "warning", "error", "critical"}},
				"description": "Filter by severity levels (1=debug, 2=verbose, 3=info, 4=warning, 5=error, 6=critical)",
			},
			"min_severity": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"debug", "verbose", "info", "warning", "error", "critical"},
				"description": "Minimum severity level to include (e.g., 'warning' includes warning, error, critical)",
			},
			"fields": map[string]interface{}{
				"type": "array",
				"description": `Field-value filters for structured log data. In DataPrime, fields have prefixes:
- $l. (labels): applicationname, subsystemname, computername, ipaddress, threadid, processid, classname, methodname, category
- $m. (metadata): severity, timestamp, priority
- $d. (data): JSON/structured data fields like json.status_code, json.user_id
Note: 'namespace' is aliased to applicationname, 'component/resource/module' to subsystemname`,
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"field": map[string]interface{}{
							"type":        "string",
							"description": `Field name. Use prefixes for DataPrime: $l.applicationname (label), $m.severity (metadata), $d.status_code (data). Without prefix, common fields auto-map: applicationname->$l, severity->$m, others->$d`,
						},
						"operator": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"equals", "not_equals", "contains", "starts_with", "ends_with", "greater_than", "less_than", "exists", "not_exists"},
							"description": "Comparison operator. contains/starts_with/ends_with use DataPrime string functions (contains(), startsWith(), endsWith())",
						},
						"value": map[string]interface{}{
							"type":        "string",
							"description": "Value to compare against (not needed for exists/not_exists)",
						},
					},
					"required": []string{"field", "operator"},
				},
			},
			"exclude_text": map[string]interface{}{
				"type":        "string",
				"description": "Text to exclude from results (logs containing this text will be filtered out)",
			},
			"output_format": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"lucene", "dataprime", "both"},
				"description": "Which query format to generate (default: both)",
				"default":     "both",
			},
		},
		"examples": []interface{}{
			map[string]interface{}{
				"text_search":  "connection timeout",
				"applications": []string{"api-gateway", "auth-service"},
				"min_severity": "error",
			},
			map[string]interface{}{
				"applications": []string{"payment-service"},
				"fields": []map[string]interface{}{
					{"field": "json.status_code", "operator": "greater_than", "value": "499"},
					{"field": "json.user_id", "operator": "exists"},
				},
				"severities": []string{"error", "critical"},
			},
			map[string]interface{}{
				"text_search":  "failed",
				"exclude_text": "health check",
				"subsystems":   []string{"database", "cache"},
			},
		},
	}
}

// Execute builds the query from structured parameters.
func (t *BuildQueryTool) Execute(_ context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Extract parameters
	textSearch, _ := GetStringParam(arguments, "text_search", false)
	excludeText, _ := GetStringParam(arguments, "exclude_text", false)
	minSeverity, _ := GetStringParam(arguments, "min_severity", false)
	outputFormat, _ := GetStringParam(arguments, "output_format", false)
	if outputFormat == "" {
		outputFormat = "both"
	}

	applications := getStringArray(arguments, "applications")
	subsystems := getStringArray(arguments, "subsystems")
	severities := getStringArray(arguments, "severities")
	fields := getFieldFilters(arguments)

	// Build Lucene query
	luceneQuery := t.buildLuceneQuery(textSearch, excludeText, applications, subsystems, severities, minSeverity, fields)

	// Build DataPrime query
	dataprimeQuery := t.buildDataPrimeQuery(textSearch, excludeText, applications, subsystems, severities, minSeverity, fields)

	// Format response
	var response strings.Builder
	response.WriteString("## Query Builder Result\n\n")

	if luceneQuery == "" && dataprimeQuery == "" {
		response.WriteString("No filters specified. Please provide at least one search criterion.\n\n")
		response.WriteString("**Available options:**\n")
		response.WriteString("- `text_search`: Free text to search in logs\n")
		response.WriteString("- `applications`: Filter by application names\n")
		response.WriteString("- `subsystems`: Filter by subsystem names\n")
		response.WriteString("- `severities`: Filter by specific severity levels\n")
		response.WriteString("- `min_severity`: Filter by minimum severity\n")
		response.WriteString("- `fields`: Filter by structured field values\n")
		response.WriteString("- `exclude_text`: Exclude logs containing specific text\n")

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: response.String()},
			},
		}, nil
	}

	// Show query summary
	response.WriteString("### Query Summary\n")
	if textSearch != "" {
		response.WriteString(fmt.Sprintf("- **Text search:** \"%s\"\n", textSearch))
	}
	if excludeText != "" {
		response.WriteString(fmt.Sprintf("- **Excluding:** \"%s\"\n", excludeText))
	}
	if len(applications) > 0 {
		response.WriteString(fmt.Sprintf("- **Applications:** %s\n", strings.Join(applications, ", ")))
	}
	if len(subsystems) > 0 {
		response.WriteString(fmt.Sprintf("- **Subsystems:** %s\n", strings.Join(subsystems, ", ")))
	}
	if len(severities) > 0 {
		response.WriteString(fmt.Sprintf("- **Severities:** %s\n", strings.Join(severities, ", ")))
	}
	if minSeverity != "" {
		response.WriteString(fmt.Sprintf("- **Minimum severity:** %s\n", minSeverity))
	}
	if len(fields) > 0 {
		response.WriteString(fmt.Sprintf("- **Field filters:** %d\n", len(fields)))
	}
	response.WriteString("\n")

	// Output queries based on format preference
	if outputFormat == "lucene" || outputFormat == "both" {
		response.WriteString("### Lucene Query\n\n")
		response.WriteString("```\n")
		response.WriteString(luceneQuery)
		response.WriteString("\n```\n\n")
		response.WriteString("**Usage with query_logs:**\n")
		response.WriteString("```json\n")
		response.WriteString(fmt.Sprintf(`{"query": "%s", "syntax": "lucene"}`, escapeJSON(luceneQuery)))
		response.WriteString("\n```\n\n")
	}

	if outputFormat == "dataprime" || outputFormat == "both" {
		response.WriteString("### DataPrime Query\n\n")
		response.WriteString("```\n")
		response.WriteString(dataprimeQuery)
		response.WriteString("\n```\n\n")
		response.WriteString("**Usage with query_logs:**\n")
		response.WriteString("```json\n")
		response.WriteString(fmt.Sprintf(`{"query": "%s", "syntax": "dataprime"}`, escapeJSON(dataprimeQuery)))
		response.WriteString("\n```\n\n")
	}

	response.WriteString("---\n")
	response.WriteString("**Next step:** Use the `query_logs` tool with one of the queries above to search your logs.\n")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response.String()},
		},
	}, nil
}

// buildLuceneQuery constructs a Lucene query from the parameters
func (t *BuildQueryTool) buildLuceneQuery(textSearch, excludeText string, applications, subsystems, severities []string, minSeverity string, fields []fieldFilter) string {
	var parts []string

	// Text search
	if textSearch != "" {
		// Quote if contains spaces
		if strings.Contains(textSearch, " ") {
			parts = append(parts, fmt.Sprintf(`"%s"`, textSearch))
		} else {
			parts = append(parts, textSearch)
		}
	}

	// Exclude text
	if excludeText != "" {
		if strings.Contains(excludeText, " ") {
			parts = append(parts, fmt.Sprintf(`NOT "%s"`, excludeText))
		} else {
			parts = append(parts, fmt.Sprintf("NOT %s", excludeText))
		}
	}

	// Applications filter
	if len(applications) > 0 {
		if len(applications) == 1 {
			parts = append(parts, fmt.Sprintf("applicationname:%s", applications[0]))
		} else {
			appParts := make([]string, len(applications))
			for i, app := range applications {
				appParts[i] = fmt.Sprintf("applicationname:%s", app)
			}
			parts = append(parts, fmt.Sprintf("(%s)", strings.Join(appParts, " OR ")))
		}
	}

	// Subsystems filter
	if len(subsystems) > 0 {
		if len(subsystems) == 1 {
			parts = append(parts, fmt.Sprintf("subsystemname:%s", subsystems[0]))
		} else {
			subParts := make([]string, len(subsystems))
			for i, sub := range subsystems {
				subParts[i] = fmt.Sprintf("subsystemname:%s", sub)
			}
			parts = append(parts, fmt.Sprintf("(%s)", strings.Join(subParts, " OR ")))
		}
	}

	// Severity filter
	if minSeverity != "" {
		minLevel := severityToInt(minSeverity)
		if minLevel > 0 {
			parts = append(parts, fmt.Sprintf("severity:>=%d", minLevel))
		}
	} else if len(severities) > 0 {
		if len(severities) == 1 {
			parts = append(parts, fmt.Sprintf("severity:%d", severityToInt(severities[0])))
		} else {
			sevParts := make([]string, 0, len(severities))
			for _, sev := range severities {
				level := severityToInt(sev)
				if level > 0 {
					sevParts = append(sevParts, fmt.Sprintf("severity:%d", level))
				}
			}
			if len(sevParts) > 0 {
				parts = append(parts, fmt.Sprintf("(%s)", strings.Join(sevParts, " OR ")))
			}
		}
	}

	// Field filters
	for _, f := range fields {
		lucenePart := fieldToLucene(f)
		if lucenePart != "" {
			parts = append(parts, lucenePart)
		}
	}

	return strings.Join(parts, " AND ")
}

// buildDataPrimeQuery constructs a DataPrime query from the parameters
func (t *BuildQueryTool) buildDataPrimeQuery(textSearch, excludeText string, applications, subsystems, severities []string, minSeverity string, fields []fieldFilter) string {
	var filters []string

	// Text search - DataPrime uses contains() for substring matching
	if textSearch != "" {
		filters = append(filters, fmt.Sprintf(`$d.message.contains('%s')`, escapeDataPrimeString(textSearch)))
	}

	// Exclude text - use NOT contains()
	if excludeText != "" {
		filters = append(filters, fmt.Sprintf(`NOT $d.message.contains('%s')`, escapeDataPrimeString(excludeText)))
	}

	// Applications filter
	if len(applications) > 0 {
		if len(applications) == 1 {
			filters = append(filters, fmt.Sprintf(`$l.applicationname == '%s'`, applications[0]))
		} else {
			appParts := make([]string, len(applications))
			for i, app := range applications {
				appParts[i] = fmt.Sprintf(`$l.applicationname == '%s'`, app)
			}
			filters = append(filters, fmt.Sprintf("(%s)", strings.Join(appParts, " || ")))
		}
	}

	// Subsystems filter
	if len(subsystems) > 0 {
		if len(subsystems) == 1 {
			filters = append(filters, fmt.Sprintf(`$l.subsystemname == '%s'`, subsystems[0]))
		} else {
			subParts := make([]string, len(subsystems))
			for i, sub := range subsystems {
				subParts[i] = fmt.Sprintf(`$l.subsystemname == '%s'`, sub)
			}
			filters = append(filters, fmt.Sprintf("(%s)", strings.Join(subParts, " || ")))
		}
	}

	// Severity filter
	if minSeverity != "" {
		minLevel := severityToInt(minSeverity)
		if minLevel > 0 {
			filters = append(filters, fmt.Sprintf("$m.severity >= %d", minLevel))
		}
	} else if len(severities) > 0 {
		if len(severities) == 1 {
			filters = append(filters, fmt.Sprintf("$m.severity == %d", severityToInt(severities[0])))
		} else {
			sevParts := make([]string, 0, len(severities))
			for _, sev := range severities {
				level := severityToInt(sev)
				if level > 0 {
					sevParts = append(sevParts, fmt.Sprintf("$m.severity == %d", level))
				}
			}
			if len(sevParts) > 0 {
				filters = append(filters, fmt.Sprintf("(%s)", strings.Join(sevParts, " || ")))
			}
		}
	}

	// Field filters
	for _, f := range fields {
		dpPart := fieldToDataPrime(f)
		if dpPart != "" {
			filters = append(filters, dpPart)
		}
	}

	if len(filters) == 0 {
		return ""
	}

	return fmt.Sprintf("source logs | filter %s", strings.Join(filters, " && "))
}

// fieldFilter represents a structured field filter
type fieldFilter struct {
	Field    string
	Operator string
	Value    string
}

// getStringArray extracts a string array from arguments
func getStringArray(arguments map[string]interface{}, key string) []string {
	arr, ok := arguments[key].([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// getFieldFilters extracts field filters from arguments
func getFieldFilters(arguments map[string]interface{}) []fieldFilter {
	arr, ok := arguments["fields"].([]interface{})
	if !ok {
		return nil
	}
	var filters []fieldFilter
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			f := fieldFilter{}
			if field, ok := m["field"].(string); ok {
				f.Field = field
			}
			if op, ok := m["operator"].(string); ok {
				f.Operator = op
			}
			if val, ok := m["value"].(string); ok {
				f.Value = val
			}
			if f.Field != "" && f.Operator != "" {
				filters = append(filters, f)
			}
		}
	}
	return filters
}

// severityToInt converts severity name to integer level
func severityToInt(severity string) int {
	switch strings.ToLower(severity) {
	case "debug":
		return 1
	case "verbose":
		return 2
	case "info":
		return 3
	case "warning", "warn":
		return 4
	case "error":
		return 5
	case "critical", "fatal":
		return 6
	default:
		return 0
	}
}

// fieldToLucene converts a field filter to Lucene syntax
func fieldToLucene(f fieldFilter) string {
	switch f.Operator {
	case "equals":
		return fmt.Sprintf("%s:%s", f.Field, f.Value)
	case "not_equals":
		return fmt.Sprintf("NOT %s:%s", f.Field, f.Value)
	case "contains":
		return fmt.Sprintf("%s:*%s*", f.Field, f.Value)
	case "starts_with":
		return fmt.Sprintf("%s:%s*", f.Field, f.Value)
	case "ends_with":
		return fmt.Sprintf("%s:*%s", f.Field, f.Value)
	case "greater_than":
		return fmt.Sprintf("%s:>%s", f.Field, f.Value)
	case "less_than":
		return fmt.Sprintf("%s:<%s", f.Field, f.Value)
	case "exists":
		return fmt.Sprintf("%s:*", f.Field)
	case "not_exists":
		return fmt.Sprintf("NOT %s:*", f.Field)
	default:
		return ""
	}
}

// fieldToDataPrime converts a field filter to DataPrime syntax
// Uses DataPrime string functions (contains, startsWith, endsWith, matches) instead of ~~ operator
func fieldToDataPrime(f fieldFilter) string {
	// Convert field name to DataPrime reference
	dpField := toDataPrimeField(f.Field)

	switch f.Operator {
	case "equals":
		return fmt.Sprintf("%s == '%s'", dpField, f.Value)
	case "not_equals":
		return fmt.Sprintf("%s != '%s'", dpField, f.Value)
	case "contains":
		// Use contains() function - works on all field types
		return fmt.Sprintf("%s.contains('%s')", dpField, f.Value)
	case "starts_with":
		// Use startsWith() function - works on all field types
		return fmt.Sprintf("%s.startsWith('%s')", dpField, f.Value)
	case "ends_with":
		// Use endsWith() function - works on all field types
		return fmt.Sprintf("%s.endsWith('%s')", dpField, f.Value)
	case "greater_than":
		return fmt.Sprintf("%s > %s", dpField, f.Value)
	case "less_than":
		return fmt.Sprintf("%s < %s", dpField, f.Value)
	case "exists":
		return fmt.Sprintf("%s != null", dpField)
	case "not_exists":
		return fmt.Sprintf("%s == null", dpField)
	default:
		return ""
	}
}

// toDataPrimeField converts a field name to DataPrime reference format
func toDataPrimeField(field string) string {
	// Already has a prefix
	if strings.HasPrefix(field, "$d.") || strings.HasPrefix(field, "$l.") || strings.HasPrefix(field, "$m.") {
		return field
	}

	// JSON fields go to $d (data)
	if strings.HasPrefix(field, "json.") {
		return "$d." + strings.TrimPrefix(field, "json.")
	}

	// Field aliases - these map user-friendly names to actual IBM Cloud Logs field names
	// In IBM Cloud Logs, Kubernetes namespace is typically stored in applicationname
	fieldAliases := map[string]string{
		"namespace": "applicationname", // K8s namespace -> applicationname
		"app":       "applicationname",
		"component": "subsystemname",
		"resource":  "subsystemname",
		"module":    "subsystemname",
	}
	lowerField := strings.ToLower(field)
	if aliasTarget, ok := fieldAliases[lowerField]; ok {
		return "$l." + aliasTarget
	}

	// Known label fields ($l prefix in DataPrime)
	// These are standard IBM Cloud Logs label fields that identify log sources
	labelFields := map[string]bool{
		// Core identification labels (these exist in IBM Cloud Logs)
		"applicationname": true,
		"subsystemname":   true,
		"computername":    true,
		"ipaddress":       true,
		// Thread and process context
		"threadid":  true,
		"processid": true,
		// Code location labels
		"classname":  true,
		"methodname": true,
		"category":   true,
	}
	if labelFields[lowerField] {
		return "$l." + field
	}

	// Known metadata fields
	metadataFields := map[string]bool{
		"severity":  true,
		"timestamp": true,
		"priority":  true,
	}
	if metadataFields[strings.ToLower(field)] {
		return "$m." + field
	}

	// Default to data field
	return "$d." + field
}

// escapeDataPrimeString escapes special characters in DataPrime strings
func escapeDataPrimeString(s string) string {
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

// escapeJSON escapes special characters for JSON string
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
