package tools

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// BuildQueryTool helps users construct log queries without knowing Lucene or DataPrime syntax.
// It takes structured parameters and generates the appropriate query string.
type BuildQueryTool struct {
	*BaseTool
}

// NewBuildQueryTool creates a new BuildQueryTool instance.
func NewBuildQueryTool(client client.Doer, logger *zap.Logger) *BuildQueryTool {
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
		fmt.Fprintf(&response, "- **Text search:** \"%s\"\n", textSearch)
	}
	if excludeText != "" {
		fmt.Fprintf(&response, "- **Excluding:** \"%s\"\n", excludeText)
	}
	if len(applications) > 0 {
		fmt.Fprintf(&response, "- **Applications:** %s\n", strings.Join(applications, ", "))
	}
	if len(subsystems) > 0 {
		fmt.Fprintf(&response, "- **Subsystems:** %s\n", strings.Join(subsystems, ", "))
	}
	if len(severities) > 0 {
		fmt.Fprintf(&response, "- **Severities:** %s\n", strings.Join(severities, ", "))
	}
	if minSeverity != "" {
		fmt.Fprintf(&response, "- **Minimum severity:** %s\n", minSeverity)
	}
	if len(fields) > 0 {
		fmt.Fprintf(&response, "- **Field filters:** %d\n", len(fields))
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
		fmt.Fprintf(&response, `{"query": "%s", "syntax": "lucene"}`, escapeJSON(luceneQuery))
		response.WriteString("\n```\n\n")
	}

	if outputFormat == "dataprime" || outputFormat == "both" {
		response.WriteString("### DataPrime Query\n\n")
		response.WriteString("```\n")
		response.WriteString(dataprimeQuery)
		response.WriteString("\n```\n\n")
		response.WriteString("**Usage with query_logs:**\n")
		response.WriteString("```json\n")
		fmt.Fprintf(&response, `{"query": "%s", "syntax": "dataprime"}`, escapeJSON(dataprimeQuery))
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
			filters = append(filters, fmt.Sprintf(`$l.applicationname == '%s'`, escapeDataPrimeString(applications[0])))
		} else {
			appParts := make([]string, len(applications))
			for i, app := range applications {
				appParts[i] = fmt.Sprintf(`$l.applicationname == '%s'`, escapeDataPrimeString(app))
			}
			filters = append(filters, fmt.Sprintf("(%s)", strings.Join(appParts, " || ")))
		}
	}

	// Subsystems filter
	if len(subsystems) > 0 {
		if len(subsystems) == 1 {
			filters = append(filters, fmt.Sprintf(`$l.subsystemname == '%s'`, escapeDataPrimeString(subsystems[0])))
		} else {
			subParts := make([]string, len(subsystems))
			for i, sub := range subsystems {
				subParts[i] = fmt.Sprintf(`$l.subsystemname == '%s'`, escapeDataPrimeString(sub))
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
	// f.Field is interpolated unquoted into the Lucene expression in every
	// branch below, so a caller-supplied field name containing Lucene syntax
	// (spaces, ':', '*', boolean operators, ...) would inject into the query.
	// Drop the filter unless the field name is a safe reference.
	if !isSafeDataPrimeFieldRef(f.Field) {
		return ""
	}

	switch f.Operator {
	case "equals":
		return fmt.Sprintf("%s:%s", f.Field, escapeLuceneValue(f.Value))
	case "not_equals":
		return fmt.Sprintf("NOT %s:%s", f.Field, escapeLuceneValue(f.Value))
	case "contains":
		// Wrapping '*' wildcards are added by us and stay live; the user's own
		// value is escaped (including any '*'/'?' it contains).
		return fmt.Sprintf("%s:*%s*", f.Field, escapeLuceneValue(f.Value))
	case "starts_with":
		return fmt.Sprintf("%s:%s*", f.Field, escapeLuceneValue(f.Value))
	case "ends_with":
		return fmt.Sprintf("%s:*%s", f.Field, escapeLuceneValue(f.Value))
	case "greater_than":
		if !isNumericFilterValue(f.Value) {
			return ""
		}
		return fmt.Sprintf("%s:>%s", f.Field, f.Value)
	case "less_than":
		if !isNumericFilterValue(f.Value) {
			return ""
		}
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

	// The resolved field name is interpolated UNQUOTED in every branch below,
	// and toDataPrimeField passes $d./$l./$m.-prefixed inputs through
	// verbatim. Since f.Field is caller-supplied, an unsafe field name would
	// inject raw DataPrime expression syntax through the field position (which
	// value-escaping can't prevent). Drop the filter unless the field name is
	// a safe reference.
	if !isSafeDataPrimeFieldRef(dpField) {
		return ""
	}

	switch f.Operator {
	case "equals":
		return fmt.Sprintf("%s == '%s'", dpField, escapeDataPrimeString(f.Value))
	case "not_equals":
		return fmt.Sprintf("%s != '%s'", dpField, escapeDataPrimeString(f.Value))
	case "contains":
		// Use contains() function - works on all field types
		return fmt.Sprintf("%s.contains('%s')", dpField, escapeDataPrimeString(f.Value))
	case "starts_with":
		// Use startsWith() function - works on all field types
		return fmt.Sprintf("%s.startsWith('%s')", dpField, escapeDataPrimeString(f.Value))
	case "ends_with":
		// Use endsWith() function - works on all field types
		return fmt.Sprintf("%s.endsWith('%s')", dpField, escapeDataPrimeString(f.Value))
	case "greater_than":
		// Numeric comparison operands are interpolated UNQUOTED, so a
		// non-numeric value would inject raw DataPrime expression syntax
		// (escaping only protects inside '...'). Drop the filter unless the
		// value is a valid number.
		if !isNumericFilterValue(f.Value) {
			return ""
		}
		return fmt.Sprintf("%s > %s", dpField, f.Value)
	case "less_than":
		if !isNumericFilterValue(f.Value) {
			return ""
		}
		return fmt.Sprintf("%s < %s", dpField, f.Value)
	case "exists":
		return fmt.Sprintf("%s != null", dpField)
	case "not_exists":
		return fmt.Sprintf("%s == null", dpField)
	default:
		return ""
	}
}

// isNumericFilterValue reports whether s is a valid numeric literal that is
// safe to interpolate unquoted into a numeric comparison. DataPrime/Lucene
// numeric operators (>, <) place the operand into the query without quotes,
// so anything that isn't a plain number (e.g. "0 || $d.secret.contains('x')")
// would be parsed as query syntax rather than data. Requiring a parseable
// float64 neutralizes that injection vector. Values like "0x1f" or "1e999"
// that ParseFloat rejects (or that aren't finite) are treated as unsafe.
func isNumericFilterValue(s string) bool {
	if s == "" {
		return false
	}
	// Reject hex/other prefixes and anything ParseFloat's base-10 grammar
	// wouldn't accept as a bare number.
	if strings.ContainsAny(s, "xXpP") {
		return false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return false
	}
	// ParseFloat happily accepts "Inf"/"NaN" (and their sign/case variants) as
	// valid float64 literals, but they aren't plain numbers and would
	// interpolate as the bare words "Inf"/"NaN" into the query - reject them
	// like any other non-numeric value.
	return !math.IsInf(f, 0) && !math.IsNaN(f)
}

// isSafeDataPrimeFieldName reports whether s consists solely of DataPrime
// field-name characters (ASCII letters, digits, underscore, dot). Field names
// are interpolated unquoted into queries, so any character outside this set
// could inject query syntax. Empty strings are unsafe.
func isSafeDataPrimeFieldName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

// isSafeDataPrimeFieldRef reports whether s is a safe field reference to
// interpolate unquoted into a query. It is like isSafeDataPrimeFieldName but
// additionally permits a single leading DataPrime scope prefix ($d./$l./$m.),
// which toDataPrimeField produces for resolved field references. Everything
// after the optional prefix must still be a bare field-name token, so a
// caller-supplied field like "$d.x=='a'||$d.secret.contains('y'" (which
// toDataPrimeField would pass through verbatim) is rejected instead of
// injecting raw expression syntax through the field position.
func isSafeDataPrimeFieldRef(s string) bool {
	if len(s) >= 3 && s[0] == '$' && (s[1] == 'd' || s[1] == 'l' || s[1] == 'm') && s[2] == '.' {
		s = s[3:]
	}
	return isSafeDataPrimeFieldName(s)
}

// escapeLuceneValue backslash-escapes the characters that are reserved in
// Lucene query syntax so a caller-supplied value is treated as inert data
// rather than query structure. Without this, a value like "a OR bar:*" placed
// after "field:" would split into a new clause (via ':' and the
// whitespace-separated OR operator).
//
// It escapes the standard Apache Lucene reserved set
// (\ + - ! ( ) : ^ [ ] " { } ~ * ? | & /) plus '<'/'>' (ES-dialect range
// operators, so "field:>100" stays a literal instead of drifting to
// greater-than) — escaping '&'/'|' individually
// also covers the two-char '&&'/'||' operators — plus whitespace, so a
// value spanning multiple tokens collapses into a single escaped term and
// boolean keywords (AND/OR/NOT) inside it can no longer act as operators.
//
// Callers that add their own wildcards (contains/starts_with/ends_with wrap
// the value in '*') must call this on the value BEFORE adding the wrapping
// '*', so the user's own '*'/'?' are escaped while the builder's wildcards
// stay live.
func escapeLuceneValue(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 2)
	for _, r := range s {
		switch r {
		case '\\', '+', '-', '!', '(', ')', ':', '^', '[', ']', '"', '{', '}', '~', '*', '?', '|', '&', '/', '<', '>':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			if unicode.IsSpace(r) {
				b.WriteByte('\\')
			}
			b.WriteRune(r)
		}
	}
	return b.String()
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

// escapeDataPrimeString escapes special characters in DataPrime single-quoted
// string literals so caller-supplied values cannot break out of the literal
// or inject additional query syntax.
//
// Backslashes are escaped first (conceptually) by processing the input one
// rune at a time rather than doing sequential strings.ReplaceAll passes: a
// naive "escape ' then escape \" (or vice versa via two ReplaceAll calls)
// either mangles existing backslashes or re-escapes backslashes it just
// inserted. Scanning once and classifying each rune avoids both problems.
//
// A trailing backslash (e.g. `foo\`) is a real-world regression case: with
// only "'" escaped, the lone backslash would sit directly before the closing
// quote and, depending on parser behavior, could be read as escaping (i.e.
// consuming) that quote — leaving the literal unterminated and the rest of
// the crafted input parsed as query syntax.
//
// ASCII control characters (including newlines) are stripped rather than
// escaped: DataPrime has no defined escape for them, and a raw newline could
// otherwise be used to smuggle what looks like a second statement into the
// query.
func escapeDataPrimeString(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\'':
			b.WriteString(`\'`)
		case r < 0x20 || r == 0x7f:
			// Strip ASCII control characters (including \n, \r, \t, NUL).
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeJSON escapes special characters for JSON string
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// DataPrimeReferenceTool provides comprehensive DataPrime query language documentation.
type DataPrimeReferenceTool struct {
	*BaseTool
}

// NewDataPrimeReferenceTool creates a new DataPrimeReferenceTool instance.
func NewDataPrimeReferenceTool(client client.Doer, logger *zap.Logger) *DataPrimeReferenceTool {
	return &DataPrimeReferenceTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *DataPrimeReferenceTool) Name() string {
	return "get_dataprime_reference"
}

// Annotations returns tool hints for LLMs
func (t *DataPrimeReferenceTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("DataPrime Reference")
}

// Description returns a human-readable description of the tool.
func (t *DataPrimeReferenceTool) Description() string {
	return `Get DataPrime query language reference documentation.

Returns syntax, commands, functions, operators, and best practices for writing DataPrime queries.

**Use this when you need:**
- Field prefix reference ($l., $m., $d.)
- Command syntax (filter, groupby, aggregate, etc.)
- Function documentation (string, time, aggregation)
- Operator reference (==, !=, &&, ||)
- Common mistake corrections`
}

// InputSchema returns the input schema for the tool.
func (t *DataPrimeReferenceTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"section": map[string]interface{}{
				"type":        "string",
				"description": "Optional: specific section to retrieve (quick, commands, functions, operators, best_practices). Default: quick reference.",
				"enum":        []string{"quick", "commands", "functions", "operators", "best_practices", "full"},
			},
		},
	}
}

// Execute returns the DataPrime reference documentation.
func (t *DataPrimeReferenceTool) Execute(_ context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	section, _ := GetStringParam(args, "section", false)
	if section == "" {
		section = "quick"
	}

	var content string
	switch section {
	case "quick":
		content = GetDataPrimeQuickReference()
	case "commands":
		content = getDataPrimeCommandsReference()
	case "functions":
		content = getDataPrimeFunctionsReference()
	case "operators":
		content = getDataPrimeOperatorsReference()
	case "best_practices":
		content = getDataPrimeBestPracticesReference()
	case "full":
		content = GetDataPrimeHelp()
	default:
		content = GetDataPrimeQuickReference()
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: content,
			},
		},
	}, nil
}

func getDataPrimeCommandsReference() string {
	var sb strings.Builder
	sb.WriteString("# DataPrime Commands\n\n")

	sb.WriteString("## Source Commands\n")
	sb.WriteString("- `source logs` - Query log data\n")
	sb.WriteString("- `source spans` - Query trace spans\n")
	sb.WriteString("- `last 1h` - Query last hour\n")
	sb.WriteString("- `between @'2024-01-01' and @'2024-01-02'` - Time range\n\n")

	sb.WriteString("## Filter & Transform\n")
	sb.WriteString("- `filter <condition>` - Filter rows (aliases: f, where)\n")
	sb.WriteString("- `create <field> from <expr>` - Add computed field\n")
	sb.WriteString("- `extract <field> into <target> using <extractor>` - Parse data\n")
	sb.WriteString("- `choose <fields>` - Select fields to include\n")
	sb.WriteString("- `remove <fields>` - Remove fields\n\n")

	sb.WriteString("## Aggregation\n")
	sb.WriteString("- `groupby <field> aggregate <func>` - Group and aggregate\n")
	sb.WriteString("- `aggregate <func>` - Aggregate all rows\n")
	sb.WriteString("- `countby <field>` - Count by field\n")
	sb.WriteString("- `distinct <field>` - Unique values\n\n")

	sb.WriteString("## Ordering & Limits\n")
	sb.WriteString("- `orderby <field> [asc|desc]` - Sort results\n")
	sb.WriteString("- `limit <n>` - Limit results\n")
	sb.WriteString("- `top <n> <field> by <order>` - Top N values\n")

	return sb.String()
}

func getDataPrimeFunctionsReference() string {
	var sb strings.Builder
	sb.WriteString("# DataPrime Functions\n\n")

	sb.WriteString("## Aggregation\n")
	sb.WriteString("- `count()` - Count rows\n")
	sb.WriteString("- `sum(field)`, `avg(field)`, `min(field)`, `max(field)`\n")
	sb.WriteString("- `distinct_count(field)` - Count unique values\n")
	sb.WriteString("- `percentile(field, p)` - Percentile value\n\n")

	sb.WriteString("## String (use method or function notation)\n")
	sb.WriteString("- `field.contains('text')` or `contains(field, 'text')`\n")
	sb.WriteString("- `field.startsWith('prefix')`, `field.endsWith('suffix')`\n")
	sb.WriteString("- `field.matches(/regex/)` - Regex matching\n")
	sb.WriteString("- `field.toLowerCase()`, `field.toUpperCase()`\n")
	sb.WriteString("- `concat(s1, s2, ...)`, `substr(s, start, len)`\n\n")

	sb.WriteString("## Time\n")
	sb.WriteString("- `now()` - Current timestamp\n")
	sb.WriteString("- `parseTimestamp(str, format)`, `formatTimestamp(ts, format)`\n")
	sb.WriteString("- `diffTime(ts1, ts2)` - Time difference\n\n")

	sb.WriteString("## Conditional\n")
	sb.WriteString("- `if(condition, then, else)`\n")
	sb.WriteString("- `coalesce(val1, val2, ...)` - First non-null\n")
	sb.WriteString("- `case { cond1 -> val1, cond2 -> val2 }`\n")

	return sb.String()
}

func getDataPrimeOperatorsReference() string {
	var sb strings.Builder
	sb.WriteString("# DataPrime Operators\n\n")

	sb.WriteString("## Comparison\n")
	sb.WriteString("- `==` Equal, `!=` Not equal\n")
	sb.WriteString("- `>`, `<`, `>=`, `<=` Numeric comparison\n")
	sb.WriteString("- `~` Contains (string), `!~` Does not contain\n\n")

	sb.WriteString("## Logical (NOT 'AND'/'OR' keywords!)\n")
	sb.WriteString("- `&&` Logical AND\n")
	sb.WriteString("- `||` Logical OR\n")
	sb.WriteString("- `!` Logical NOT\n\n")

	sb.WriteString("## Important Notes\n")
	sb.WriteString("- Use `==` not `=` for equality\n")
	sb.WriteString("- Use `&&` not `AND`\n")
	sb.WriteString("- Use `||` not `OR`\n")
	sb.WriteString("- Use single quotes: `'value'` not `\"value\"`\n")
	sb.WriteString("- `~~` is NOT supported - use `matches(/regex/)`\n")

	return sb.String()
}

func getDataPrimeBestPracticesReference() string {
	var sb strings.Builder
	sb.WriteString("# DataPrime Best Practices\n\n")

	sb.WriteString("## Performance\n")
	sb.WriteString("- Always specify time range to limit data scanned\n")
	sb.WriteString("- Use exact matches (==) over contains() when possible\n")
	sb.WriteString("- Filter early in query to reduce data before expensive ops\n")
	sb.WriteString("- Avoid groupby on high-cardinality fields without limits\n\n")

	sb.WriteString("## Common Patterns\n")
	sb.WriteString("- Total count: `groupby true aggregate count()`\n")
	sb.WriteString("- Case-insensitive: `field.toLowerCase().contains('text')`\n")
	sb.WriteString("- Free-text search: `lucene 'error AND timeout'`\n\n")

	sb.WriteString("## Field Access\n")
	sb.WriteString("- `$d.` prefix optional for data fields\n")
	sb.WriteString("- Labels: `$l.applicationname`, `$l.subsystemname`\n")
	sb.WriteString("- Metadata: `$m.severity`, `$m.timestamp`\n\n")

	sb.WriteString("## Common Mistakes\n")
	sb.WriteString("- Using AND/OR instead of &&/||\n")
	sb.WriteString("- Using = instead of ==\n")
	sb.WriteString("- Using double quotes instead of single\n")
	sb.WriteString("- Using ~~ (not supported) instead of matches()\n")

	return sb.String()
}
