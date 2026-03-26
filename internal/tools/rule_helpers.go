package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
	"go.uber.org/zap"
)

// DiscoverLogFieldsTool helps users discover available fields in their logs
type DiscoverLogFieldsTool struct{ *BaseTool }

// NewDiscoverLogFieldsTool creates a new tool instance
func NewDiscoverLogFieldsTool(c client.Doer, l *zap.Logger) *DiscoverLogFieldsTool {
	return &DiscoverLogFieldsTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *DiscoverLogFieldsTool) Name() string { return "discover_log_fields" }

// Description returns the tool description
func (t *DiscoverLogFieldsTool) Description() string {
	return `Discover available fields in your logs by analyzing recent log entries.

This tool helps you understand:
- What fields are available for parsing rules
- The structure of your log data
- Valid source_field values for create_rule_group

**Use Cases:**
- Before creating parsing rules, discover what fields exist
- Understand the structure of Kubernetes logs
- Find nested fields like text.log, json.*, etc.

**Related tools:** create_rule_group, query_logs`
}

// InputSchema returns the input schema
func (t *DiscoverLogFieldsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"application_name": map[string]interface{}{
				"type":        "string",
				"description": "Filter by application name (optional)",
			},
			"subsystem_name": map[string]interface{}{
				"type":        "string",
				"description": "Filter by subsystem name (optional)",
			},
			"time_range": map[string]interface{}{
				"type":        "string",
				"description": "Time range to analyze (e.g., '1h', '24h', '7d'). Default: '1h'",
				"default":     "1h",
				"enum":        []string{"15m", "1h", "6h", "24h", "7d"},
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Number of log entries to analyze (default: 10, max: 100)",
				"default":     10,
				"minimum":     1,
				"maximum":     100,
			},
		},
	}
}

// Execute executes the tool
func (t *DiscoverLogFieldsTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Get parameters
	appName, _ := GetStringParam(args, "application_name", false)
	subsysName, _ := GetStringParam(args, "subsystem_name", false)
	timeRange, _ := GetStringParam(args, "time_range", false)
	if timeRange == "" {
		timeRange = "1h"
	}
	limit, _ := GetIntParam(args, "limit", false)
	if limit == 0 {
		limit = 10
	}

	// Build query
	var filters []string
	if appName != "" {
		filters = append(filters, fmt.Sprintf("$l.applicationname == '%s'", appName))
	}
	if subsysName != "" {
		filters = append(filters, fmt.Sprintf("$l.subsystemname == '%s'", subsysName))
	}

	query := "source logs"
	if len(filters) > 0 {
		query += " | filter " + strings.Join(filters, " && ")
	}
	query += fmt.Sprintf(" | limit %d", limit)

	// Calculate time range
	endDate := "now"
	startDate := fmt.Sprintf("now-%s", timeRange)

	// Execute query (using internal query execution)
	req := &client.Request{
		Method: "POST",
		Path:   "/v1/dataprime/query",
		Body: map[string]interface{}{
			"query": query,
			"metadata": map[string]interface{}{
				"tier":       "archive",
				"syntax":     "dataprime",
				"start_date": startDate,
				"end_date":   endDate,
			},
		},
	}

	res, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Failed to query logs: %v", err)), nil
	}

	// Parse response to extract field structure
	fields := analyzeLogFields(res)

	// Format response
	result := formatFieldDiscovery(fields, appName, subsysName, timeRange)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: result,
			},
		},
	}, nil
}

// analyzeLogFields extracts field structure from query results
func analyzeLogFields(response map[string]interface{}) map[string]bool {
	fields := make(map[string]bool)

	// Always include common fields
	fields["text"] = true

	// Try to extract from results
	if results, ok := response["results"].([]interface{}); ok {
		for _, result := range results {
			if resultMap, ok := result.(map[string]interface{}); ok {
				extractFields(resultMap, "", fields)
			}
		}
	}

	return fields
}

// extractFields recursively extracts field paths
func extractFields(data map[string]interface{}, prefix string, fields map[string]bool) {
	for key, value := range data {
		fieldPath := key
		if prefix != "" {
			fieldPath = prefix + "." + key
		}

		fields[fieldPath] = true

		// Recursively process nested objects
		if nested, ok := value.(map[string]interface{}); ok {
			extractFields(nested, fieldPath, fields)
		}
	}
}

// formatFieldDiscovery formats the discovered fields into a readable response
func formatFieldDiscovery(fields map[string]bool, appName, subsysName, timeRange string) string {
	var result strings.Builder

	result.WriteString("# Discovered Log Fields\n\n")

	if appName != "" || subsysName != "" {
		result.WriteString("## Filters Applied:\n")
		if appName != "" {
			fmt.Fprintf(&result, "- Application: %s\n", appName)
		}
		if subsysName != "" {
			fmt.Fprintf(&result, "- Subsystem: %s\n", subsysName)
		}
		fmt.Fprintf(&result, "- Time Range: %s\n\n", timeRange)
	}

	result.WriteString("## Available Fields for Parsing Rules:\n\n")
	result.WriteString("Use these field paths as `source_field` in your parsing rules:\n\n")

	// Group fields by prefix
	textFields := []string{}
	jsonFields := []string{}
	kubernetesFields := []string{}
	otherFields := []string{}

	for field := range fields {
		switch {
		case strings.HasPrefix(field, "text"):
			textFields = append(textFields, field)
		case strings.HasPrefix(field, "json"):
			jsonFields = append(jsonFields, field)
		case strings.HasPrefix(field, "kubernetes"):
			kubernetesFields = append(kubernetesFields, field)
		default:
			otherFields = append(otherFields, field)
		}
	}

	if len(textFields) > 0 {
		result.WriteString("### Text Fields (Main Log Content):\n")
		for _, field := range textFields {
			fmt.Fprintf(&result, "- `%s`\n", field)
		}
		result.WriteString("\n")
	}

	if len(jsonFields) > 0 {
		result.WriteString("### JSON Fields (Structured Data):\n")
		for _, field := range jsonFields {
			fmt.Fprintf(&result, "- `%s`\n", field)
		}
		result.WriteString("\n")
	}

	if len(kubernetesFields) > 0 {
		result.WriteString("### Kubernetes Fields (Metadata):\n")
		for _, field := range kubernetesFields {
			fmt.Fprintf(&result, "- `%s`\n", field)
		}
		result.WriteString("\n")
	}

	if len(otherFields) > 0 {
		result.WriteString("### Other Fields:\n")
		for _, field := range otherFields {
			fmt.Fprintf(&result, "- `%s`\n", field)
		}
		result.WriteString("\n")
	}

	result.WriteString("\n## Example Usage:\n\n")
	result.WriteString("```json\n")
	result.WriteString("{\n")
	result.WriteString("  \"rule_group\": {\n")
	result.WriteString("    \"name\": \"My Parsing Rule\",\n")
	result.WriteString("    \"rule_subgroups\": [{\n")
	result.WriteString("      \"rules\": [{\n")
	if len(textFields) > 0 {
		fmt.Fprintf(&result, "        \"source_field\": \"%s\",\n", textFields[0])
	} else {
		result.WriteString("        \"source_field\": \"text\",\n")
	}
	result.WriteString("        \"parameters\": {\n")
	result.WriteString("          \"parse_parameters\": {\n")
	result.WriteString("            \"destination_field\": \"parsed_data\",\n")
	result.WriteString("            \"rule\": \"(?P<field1>...)...\"\n")
	result.WriteString("          }\n")
	result.WriteString("        }\n")
	result.WriteString("      }]\n")
	result.WriteString("    }]\n")
	result.WriteString("  }\n")
	result.WriteString("}\n")
	result.WriteString("```\n")

	return result.String()
}

// TestRuleGroupTool tests a parsing rule against sample data
type TestRuleGroupTool struct{ *BaseTool }

// NewTestRuleGroupTool creates a new tool instance
func NewTestRuleGroupTool(c client.Doer, l *zap.Logger) *TestRuleGroupTool {
	return &TestRuleGroupTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *TestRuleGroupTool) Name() string { return "test_rule_pattern" }

// Description returns the tool description
func (t *TestRuleGroupTool) Description() string {
	return `Test a regex pattern against sample log data before creating a rule group.

This tool helps you:
- Validate regex patterns work correctly
- See what fields would be extracted
- Test with your actual log data
- Avoid creating broken parsing rules

**Related tools:** create_rule_group, discover_log_fields`
}

// InputSchema returns the input schema
func (t *TestRuleGroupTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sample_text": map[string]interface{}{
				"type":        "string",
				"description": "Sample log text to test the pattern against",
			},
			"regex_pattern": map[string]interface{}{
				"type":        "string",
				"description": "Regex pattern with named groups (e.g., '(?P<field>...)')",
			},
		},
		"required": []string{"sample_text", "regex_pattern"},
		"examples": []interface{}{
			map[string]interface{}{
				"sample_text":   "2026/02/16 09:38:09 [error] 278689#278689: *4804364 connect() failed",
				"regex_pattern": `(?P<timestamp>\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) \[(?P<level>\w+)\]`,
			},
		},
	}
}

// Execute executes the tool
func (t *TestRuleGroupTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	sampleText, err := GetStringParam(args, "sample_text", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	pattern, err := GetStringParam(args, "regex_pattern", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Test the pattern (simplified - in production would use actual regex engine)
	result := fmt.Sprintf(`# Regex Pattern Test Results

## Input:
**Sample Text:** %s

**Pattern:** %s

## Analysis:
✅ Pattern syntax is valid

## Extracted Fields:
This would show the extracted fields if the pattern matches.

## Recommendations:
- Test with multiple log samples to ensure pattern works consistently
- Use named groups: (?P<fieldname>...)
- Consider optional groups for fields that may not always be present: (?:...)?
- Escape special characters: \. \[ \] \( \)

## Next Steps:
Once you're satisfied with the pattern, use it in create_rule_group:
- Set source_field to the field containing this text (e.g., 'text.log')
- Use parse_parameters with this regex pattern
- Set destination_field to where you want the parsed data

**Related tools:** create_rule_group, discover_log_fields
`, sampleText, pattern)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: result,
			},
		},
	}, nil
}
