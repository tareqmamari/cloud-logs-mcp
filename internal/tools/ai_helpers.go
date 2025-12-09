// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file contains AI-helper tools that provide advanced analysis and suggestions.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// ExplainQueryTool explains the structure and meaning of a DataPrime or Lucene query
type ExplainQueryTool struct {
	*BaseTool
}

// NewExplainQueryTool creates a new ExplainQueryTool
func NewExplainQueryTool(c *client.Client, l *zap.Logger) *ExplainQueryTool {
	return &ExplainQueryTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ExplainQueryTool) Name() string { return "explain_query" }

// Description returns the tool description
func (t *ExplainQueryTool) Description() string {
	return `Analyze and explain a DataPrime or Lucene query, breaking down its components and what it will match.

**Use Cases:**
- Understand complex queries written by others
- Learn DataPrime or Lucene syntax
- Debug queries that aren't returning expected results
- Verify query logic before execution

**Related tools:** query_logs, build_query, dataprime_reference`
}

// InputSchema returns the input schema
func (t *ExplainQueryTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The query string to explain (DataPrime or Lucene syntax)",
			},
			"syntax": map[string]interface{}{
				"type":        "string",
				"description": "Query syntax type: dataprime (default) or lucene",
				"enum":        []string{"dataprime", "lucene"},
				"default":     "dataprime",
			},
		},
		"required": []string{"query"},
	}
}

// QueryExplanation represents the parsed explanation of a query
type QueryExplanation struct {
	OriginalQuery string           `json:"original_query"`
	Syntax        string           `json:"syntax"`
	Components    []QueryComponent `json:"components"`
	FieldsUsed    []FieldInfo      `json:"fields_used"`
	Summary       string           `json:"summary"`
	Suggestions   []string         `json:"suggestions,omitempty"`
	Warnings      []string         `json:"warnings,omitempty"`
	Examples      []QueryExample   `json:"examples,omitempty"`
}

// QueryComponent represents a parsed component of the query
type QueryComponent struct {
	Type        string `json:"type"`        // source, filter, extract, aggregate, sort, limit, etc.
	Expression  string `json:"expression"`  // The actual expression
	Description string `json:"description"` // Human-readable description
}

// FieldInfo describes a field used in the query
type FieldInfo struct {
	Name        string `json:"name"`
	Prefix      string `json:"prefix,omitempty"`   // $l, $m, $d, etc.
	Category    string `json:"category,omitempty"` // labels, metadata, data
	Description string `json:"description,omitempty"`
}

// QueryExample shows an alternative or improved query
type QueryExample struct {
	Query       string `json:"query"`
	Description string `json:"description"`
}

// Execute executes the tool
func (t *ExplainQueryTool) Execute(_ context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	query, err := GetStringParam(args, "query", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	syntax, _ := GetStringParam(args, "syntax", false)
	if syntax == "" {
		syntax = "dataprime"
	}

	var explanation *QueryExplanation
	if syntax == "dataprime" {
		explanation = explainDataPrimeQuery(query)
	} else {
		explanation = explainLuceneQuery(query)
	}

	result, err := json.MarshalIndent(explanation, "", "  ")
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Failed to format explanation: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(result),
			},
		},
	}, nil
}

// explainDataPrimeQuery parses and explains a DataPrime query
func explainDataPrimeQuery(query string) *QueryExplanation {
	explanation := &QueryExplanation{
		OriginalQuery: query,
		Syntax:        "dataprime",
		Components:    []QueryComponent{},
		FieldsUsed:    []FieldInfo{},
	}

	// Split by pipe to get stages
	stages := splitPipeStages(query)

	for _, stage := range stages {
		stage = strings.TrimSpace(stage)
		if stage == "" {
			continue
		}

		component := parseDataPrimeStage(stage)
		explanation.Components = append(explanation.Components, component)
	}

	// Extract fields used
	explanation.FieldsUsed = extractFieldsFromQuery(query)

	// Build summary
	explanation.Summary = buildQuerySummary(explanation.Components)

	// Add suggestions based on analysis
	explanation.Suggestions = generateQuerySuggestions(query, explanation.Components)

	// Add warnings for potential issues
	explanation.Warnings = detectQueryWarnings(query, explanation.Components)

	// Add example variations
	explanation.Examples = generateQueryExamples(query, explanation.Components)

	return explanation
}

// splitPipeStages splits a query by pipe operator, respecting quoted strings
func splitPipeStages(query string) []string {
	var stages []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, ch := range query {
		if (ch == '\'' || ch == '"') && !inQuote {
			inQuote = true
			quoteChar = ch
			current.WriteRune(ch)
		} else if ch == quoteChar && inQuote {
			inQuote = false
			current.WriteRune(ch)
		} else if ch == '|' && !inQuote {
			stages = append(stages, current.String())
			current.Reset()
		} else {
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		stages = append(stages, current.String())
	}

	return stages
}

// parseDataPrimeStage parses a single DataPrime stage
func parseDataPrimeStage(stage string) QueryComponent {
	stage = strings.TrimSpace(stage)
	stageLower := strings.ToLower(stage)

	component := QueryComponent{
		Expression: stage,
	}

	switch {
	case strings.HasPrefix(stageLower, "source"):
		component.Type = "source"
		source := strings.TrimSpace(strings.TrimPrefix(stageLower, "source"))
		component.Description = fmt.Sprintf("Reads data from the '%s' data source", source)

	case strings.HasPrefix(stageLower, "filter"):
		component.Type = "filter"
		condition := strings.TrimSpace(strings.TrimPrefix(stage, "filter"))
		condition = strings.TrimSpace(strings.TrimPrefix(condition, "Filter"))
		component.Description = fmt.Sprintf("Filters records where: %s", describeCondition(condition))

	case strings.HasPrefix(stageLower, "extract"):
		component.Type = "extract"
		component.Description = "Extracts structured data from fields using patterns"

	case strings.HasPrefix(stageLower, "groupby") || strings.HasPrefix(stageLower, "group by"):
		component.Type = "aggregate"
		component.Description = "Groups records and computes aggregations"

	case strings.HasPrefix(stageLower, "orderby") || strings.HasPrefix(stageLower, "order by") || strings.HasPrefix(stageLower, "sortby") || strings.HasPrefix(stageLower, "sort by"):
		component.Type = "sort"
		component.Description = "Sorts records by specified fields"

	case strings.HasPrefix(stageLower, "limit"):
		component.Type = "limit"
		component.Description = "Limits the number of returned records"

	case strings.HasPrefix(stageLower, "select"):
		component.Type = "select"
		component.Description = "Selects specific fields to include in output"

	case strings.HasPrefix(stageLower, "create"):
		component.Type = "create"
		component.Description = "Creates new computed fields"

	case strings.HasPrefix(stageLower, "distinct"):
		component.Type = "distinct"
		component.Description = "Returns only unique values"

	case strings.HasPrefix(stageLower, "count"):
		component.Type = "aggregate"
		component.Description = "Counts matching records"

	default:
		component.Type = "unknown"
		component.Description = "Custom or unrecognized stage"
	}

	return component
}

// describeCondition provides a human-readable description of a filter condition
func describeCondition(condition string) string {
	// Replace common operators with human-readable versions
	condition = strings.ReplaceAll(condition, "==", " equals ")
	condition = strings.ReplaceAll(condition, "!=", " does not equal ")
	condition = strings.ReplaceAll(condition, ">=", " is greater than or equal to ")
	condition = strings.ReplaceAll(condition, "<=", " is less than or equal to ")
	condition = strings.ReplaceAll(condition, ">", " is greater than ")
	condition = strings.ReplaceAll(condition, "<", " is less than ")
	condition = strings.ReplaceAll(condition, "&&", " AND ")
	condition = strings.ReplaceAll(condition, "||", " OR ")
	condition = strings.ReplaceAll(condition, ".contains(", " contains ")

	return condition
}

// extractFieldsFromQuery extracts all field references from a query
func extractFieldsFromQuery(query string) []FieldInfo {
	var fields []FieldInfo
	seen := make(map[string]bool)

	// Pattern for DataPrime field references: $l.xxx, $m.xxx, $d.xxx
	fieldPattern := regexp.MustCompile(`\$([lmd])\.([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := fieldPattern.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			fullName := "$" + match[1] + "." + match[2]
			if seen[fullName] {
				continue
			}
			seen[fullName] = true

			field := FieldInfo{
				Name:   match[2],
				Prefix: "$" + match[1],
			}

			switch match[1] {
			case "l":
				field.Category = "labels"
				field.Description = getKnownLabelDescription(match[2])
			case "m":
				field.Category = "metadata"
				field.Description = getKnownMetadataDescription(match[2])
			case "d":
				field.Category = "data"
				field.Description = "User data field from log content"
			}

			fields = append(fields, field)
		}
	}

	return fields
}

// getKnownLabelDescription returns description for known label fields
func getKnownLabelDescription(name string) string {
	descriptions := map[string]string{
		"applicationname": "Application that generated the log",
		"subsystemname":   "Subsystem/component within the application",
		"namespace":       "Kubernetes namespace or logical grouping",
		"pod":             "Kubernetes pod name",
		"container":       "Container name",
		"hostname":        "Host where the log originated",
		"environment":     "Deployment environment (prod, staging, dev)",
		"region":          "Geographic region",
		"cluster":         "Cluster identifier",
	}
	if desc, ok := descriptions[strings.ToLower(name)]; ok {
		return desc
	}
	return "Label field"
}

// getKnownMetadataDescription returns description for known metadata fields
func getKnownMetadataDescription(name string) string {
	descriptions := map[string]string{
		"severity":  "Log severity level (1-6: debug, verbose, info, warning, error, critical)",
		"timestamp": "Time when the log was generated",
		"priority":  "Log priority level",
		"logid":     "Unique identifier for the log entry",
	}
	if desc, ok := descriptions[strings.ToLower(name)]; ok {
		return desc
	}
	return "Metadata field"
}

// buildQuerySummary creates a human-readable summary of the query
func buildQuerySummary(components []QueryComponent) string {
	if len(components) == 0 {
		return "Empty or unparseable query"
	}

	var parts []string
	for _, comp := range components {
		switch comp.Type {
		case "source":
			parts = append(parts, "reads from data source")
		case "filter":
			parts = append(parts, "applies filter conditions")
		case "aggregate":
			parts = append(parts, "aggregates data")
		case "sort":
			parts = append(parts, "sorts results")
		case "limit":
			parts = append(parts, "limits output")
		case "select":
			parts = append(parts, "selects specific fields")
		}
	}

	if len(parts) == 0 {
		return "Query with custom stages"
	}

	return "This query " + strings.Join(parts, ", then ")
}

// generateQuerySuggestions provides suggestions for improving the query
func generateQuerySuggestions(query string, components []QueryComponent) []string {
	var suggestions []string

	// Check if there's a limit
	hasLimit := false
	for _, comp := range components {
		if comp.Type == "limit" {
			hasLimit = true
			break
		}
	}
	if !hasLimit {
		suggestions = append(suggestions, "Consider adding '| limit N' to restrict results and improve performance")
	}

	// Check for broad queries
	filterCount := 0
	for _, comp := range components {
		if comp.Type == "filter" {
			filterCount++
		}
	}
	if filterCount == 0 {
		suggestions = append(suggestions, "Query has no filters - consider adding filters to narrow results")
	}

	// Check for ~~ operator (invalid DataPrime syntax)
	if strings.Contains(query, "~~") {
		suggestions = append(suggestions, "The ~~ operator is not valid DataPrime - use .contains() for substring matching or .matches(/regex/) for patterns")
	}

	return suggestions
}

// detectQueryWarnings detects potential issues in the query
func detectQueryWarnings(query string, components []QueryComponent) []string {
	var warnings []string

	// Check for common typos
	if strings.Contains(strings.ToLower(query), "applicationame") {
		warnings = append(warnings, "Possible typo: 'applicationame' should be 'applicationname'")
	}

	// Check for missing source
	hasSource := false
	for _, comp := range components {
		if comp.Type == "source" {
			hasSource = true
			break
		}
	}
	if !hasSource && !strings.Contains(strings.ToLower(query), "source") {
		warnings = append(warnings, "Query may be missing 'source logs' - ensure default_source is set or add source explicitly")
	}

	return warnings
}

// generateQueryExamples provides example variations of the query
func generateQueryExamples(query string, components []QueryComponent) []QueryExample {
	var examples []QueryExample

	// If query has filters, suggest adding aggregation
	for _, comp := range components {
		if comp.Type == "filter" {
			examples = append(examples, QueryExample{
				Query:       query + " | groupby $l.applicationname calculate count() as error_count",
				Description: "Add aggregation to count matches by application",
			})
			break
		}
	}

	return examples
}

// explainLuceneQuery parses and explains a Lucene query
func explainLuceneQuery(query string) *QueryExplanation {
	explanation := &QueryExplanation{
		OriginalQuery: query,
		Syntax:        "lucene",
		Components:    []QueryComponent{},
		FieldsUsed:    []FieldInfo{},
	}

	// Parse Lucene terms
	terms := strings.Fields(query)
	for _, term := range terms {
		if strings.Contains(term, ":") {
			parts := strings.SplitN(term, ":", 2)
			explanation.Components = append(explanation.Components, QueryComponent{
				Type:        "field_query",
				Expression:  term,
				Description: fmt.Sprintf("Matches documents where '%s' contains '%s'", parts[0], parts[1]),
			})
			explanation.FieldsUsed = append(explanation.FieldsUsed, FieldInfo{
				Name:        parts[0],
				Description: "Lucene field",
			})
		} else if term == "AND" || term == "OR" || term == "NOT" {
			explanation.Components = append(explanation.Components, QueryComponent{
				Type:        "operator",
				Expression:  term,
				Description: fmt.Sprintf("Boolean %s operator", term),
			})
		} else if !strings.HasPrefix(term, "(") && !strings.HasSuffix(term, ")") {
			explanation.Components = append(explanation.Components, QueryComponent{
				Type:        "text_search",
				Expression:  term,
				Description: fmt.Sprintf("Full-text search for '%s'", term),
			})
		}
	}

	explanation.Summary = "Lucene query that searches across specified fields"

	return explanation
}

// SuggestAlertTool suggests alert configurations based on query patterns
type SuggestAlertTool struct {
	*BaseTool
}

// NewSuggestAlertTool creates a new SuggestAlertTool
func NewSuggestAlertTool(c *client.Client, l *zap.Logger) *SuggestAlertTool {
	return &SuggestAlertTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *SuggestAlertTool) Name() string { return "suggest_alert" }

// Description returns the tool description
func (t *SuggestAlertTool) Description() string {
	return `Generate alert configuration suggestions based on a query or use case description.

**Use Cases:**
- Quickly create alerts from existing queries
- Get best practice alert configurations
- Learn alert configuration patterns
- Set up monitoring for common scenarios

**Related tools:** create_alert, list_alerts, query_logs, list_outgoing_webhooks`
}

// InputSchema returns the input schema
func (t *SuggestAlertTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "A query to base the alert on (optional if use_case is provided)",
			},
			"use_case": map[string]interface{}{
				"type":        "string",
				"description": "Description of what you want to alert on (e.g., 'high error rate', 'slow response times', 'disk space low')",
			},
			"severity": map[string]interface{}{
				"type":        "string",
				"description": "Desired alert severity",
				"enum":        []string{"info", "warning", "error", "critical"},
				"default":     "warning",
			},
		},
		"required": []string{},
	}
}

// AlertSuggestion represents a suggested alert configuration
type AlertSuggestion struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Query         string                 `json:"query"`
	Condition     AlertCondition         `json:"condition"`
	Schedule      AlertSchedule          `json:"schedule"`
	Notifications []string               `json:"notifications,omitempty"`
	Configuration map[string]interface{} `json:"configuration"`
	Explanation   string                 `json:"explanation"`
	BestPractices []string               `json:"best_practices"`
}

// AlertCondition represents the alert trigger condition
type AlertCondition struct {
	Type       string `json:"type"` // threshold, ratio, new_value, etc.
	Threshold  int    `json:"threshold,omitempty"`
	Operator   string `json:"operator,omitempty"` // more_than, less_than, etc.
	TimeWindow string `json:"time_window,omitempty"`
}

// AlertSchedule represents when to evaluate the alert
type AlertSchedule struct {
	Frequency     string `json:"frequency"`      // 1m, 5m, 15m, 1h, etc.
	ActiveWindows string `json:"active_windows"` // always, business_hours, etc.
}

// Execute executes the tool
func (t *SuggestAlertTool) Execute(_ context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	query, _ := GetStringParam(args, "query", false)
	useCase, _ := GetStringParam(args, "use_case", false)
	severity, _ := GetStringParam(args, "severity", false)

	if query == "" && useCase == "" {
		return NewToolResultError("Either 'query' or 'use_case' must be provided"), nil
	}

	if severity == "" {
		severity = "warning"
	}

	var suggestions []AlertSuggestion

	if useCase != "" {
		suggestions = suggestAlertFromUseCase(useCase, severity)
	}

	if query != "" {
		suggestions = append(suggestions, suggestAlertFromQuery(query, severity))
	}

	result, err := json.MarshalIndent(map[string]interface{}{
		"suggestions": suggestions,
		"next_steps": []string{
			"Review and customize the suggested configuration",
			"Use list_outgoing_webhooks to find webhook IDs for notifications",
			"Use create_alert to create the alert with the configuration",
			"Test the alert with a query_logs to verify it matches expected logs",
		},
	}, "", "  ")
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Failed to format suggestions: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(result),
			},
		},
	}, nil
}

// suggestAlertFromUseCase generates alert suggestions based on common use cases
func suggestAlertFromUseCase(useCase string, _ string) []AlertSuggestion {
	useCaseLower := strings.ToLower(useCase)
	var suggestions []AlertSuggestion

	if strings.Contains(useCaseLower, "error") {
		suggestions = append(suggestions, AlertSuggestion{
			Name:        "Error Rate Alert",
			Description: "Alert when error rate exceeds threshold",
			Query:       "source logs | filter $m.severity >= 5",
			Condition: AlertCondition{
				Type:       "threshold",
				Threshold:  10,
				Operator:   "more_than",
				TimeWindow: "5m",
			},
			Schedule: AlertSchedule{
				Frequency:     "1m",
				ActiveWindows: "always",
			},
			Configuration: map[string]interface{}{
				"alert_type": "logs_threshold",
				"condition": map[string]interface{}{
					"type":        "more_than",
					"threshold":   10,
					"time_window": "5m",
				},
				"notification_group": map[string]interface{}{
					"group_by_keys": []string{"applicationname", "subsystemname"},
				},
			},
			Explanation: "This alert triggers when more than 10 error or critical logs occur within a 5-minute window",
			BestPractices: []string{
				"Group alerts by application to reduce noise",
				"Set appropriate thresholds based on normal error rates",
				"Include relevant context in notifications",
				"Consider different thresholds for different environments",
			},
		})
	}

	if strings.Contains(useCaseLower, "response") || strings.Contains(useCaseLower, "latency") || strings.Contains(useCaseLower, "slow") {
		suggestions = append(suggestions, AlertSuggestion{
			Name:        "High Latency Alert",
			Description: "Alert when response times exceed acceptable limits",
			Query:       "source logs | filter $d.response_time_ms > 1000",
			Condition: AlertCondition{
				Type:       "threshold",
				Threshold:  5,
				Operator:   "more_than",
				TimeWindow: "5m",
			},
			Schedule: AlertSchedule{
				Frequency:     "1m",
				ActiveWindows: "always",
			},
			Configuration: map[string]interface{}{
				"alert_type": "logs_threshold",
				"condition": map[string]interface{}{
					"type":        "more_than",
					"threshold":   5,
					"time_window": "5m",
				},
			},
			Explanation: "Triggers when more than 5 requests exceed 1000ms response time in 5 minutes",
			BestPractices: []string{
				"Adjust response_time_ms threshold based on your SLOs",
				"Consider percentile-based alerting for more accuracy",
				"Include endpoint information in the query",
			},
		})
	}

	if strings.Contains(useCaseLower, "security") || strings.Contains(useCaseLower, "auth") || strings.Contains(useCaseLower, "login") {
		suggestions = append(suggestions, AlertSuggestion{
			Name:        "Security Alert - Failed Authentication",
			Description: "Alert on suspicious authentication failures",
			Query:       "source logs | filter $d.event_type == 'auth_failure'",
			Condition: AlertCondition{
				Type:       "threshold",
				Threshold:  5,
				Operator:   "more_than",
				TimeWindow: "10m",
			},
			Schedule: AlertSchedule{
				Frequency:     "1m",
				ActiveWindows: "always",
			},
			Configuration: map[string]interface{}{
				"alert_type": "logs_threshold",
				"condition": map[string]interface{}{
					"type":        "more_than",
					"threshold":   5,
					"time_window": "10m",
				},
				"notification_group": map[string]interface{}{
					"group_by_keys": []string{"source_ip", "username"},
				},
			},
			Explanation: "Triggers when more than 5 auth failures from the same source within 10 minutes",
			BestPractices: []string{
				"Group by source IP to detect brute force attempts",
				"Set up immediate notifications for security alerts",
				"Consider integrating with SIEM systems",
			},
		})
	}

	if strings.Contains(useCaseLower, "disk") || strings.Contains(useCaseLower, "space") || strings.Contains(useCaseLower, "storage") {
		suggestions = append(suggestions, AlertSuggestion{
			Name:        "Low Disk Space Alert",
			Description: "Alert when disk usage is critically high",
			Query:       "source logs | filter $d.disk_usage_percent > 85",
			Condition: AlertCondition{
				Type:       "threshold",
				Threshold:  1,
				Operator:   "more_than",
				TimeWindow: "15m",
			},
			Schedule: AlertSchedule{
				Frequency:     "5m",
				ActiveWindows: "always",
			},
			Configuration: map[string]interface{}{
				"alert_type": "logs_threshold",
				"condition": map[string]interface{}{
					"type":        "more_than",
					"threshold":   1,
					"time_window": "15m",
				},
			},
			Explanation: "Triggers when any disk reports usage above 85%",
			BestPractices: []string{
				"Set up multiple thresholds (warning at 80%, critical at 90%)",
				"Include hostname in alert grouping",
				"Automate disk cleanup or expansion if possible",
			},
		})
	}

	// Default suggestion if no specific match
	if len(suggestions) == 0 {
		suggestions = append(suggestions, AlertSuggestion{
			Name:        "Custom Log Alert",
			Description: fmt.Sprintf("Alert for: %s", useCase),
			Query:       "source logs | filter /* add your conditions here */",
			Condition: AlertCondition{
				Type:       "threshold",
				Threshold:  1,
				Operator:   "more_than",
				TimeWindow: "5m",
			},
			Schedule: AlertSchedule{
				Frequency:     "1m",
				ActiveWindows: "always",
			},
			Configuration: map[string]interface{}{
				"alert_type": "logs_threshold",
			},
			Explanation: "Customize this template based on your specific requirements",
			BestPractices: []string{
				"Start with a broad query and refine based on results",
				"Test your query with query_logs before creating the alert",
				"Set appropriate thresholds based on historical data",
			},
		})
	}

	return suggestions
}

// suggestAlertFromQuery generates an alert suggestion from an existing query
func suggestAlertFromQuery(query string, _ string) AlertSuggestion {
	// Analyze the query to suggest appropriate thresholds
	threshold := 10
	timeWindow := "5m"

	if strings.Contains(strings.ToLower(query), "error") || strings.Contains(strings.ToLower(query), "severity >= 5") {
		threshold = 5
		timeWindow = "5m"
	} else if strings.Contains(strings.ToLower(query), "critical") || strings.Contains(strings.ToLower(query), "severity >= 6") {
		threshold = 1
		timeWindow = "1m"
	}

	return AlertSuggestion{
		Name:        "Alert from Query",
		Description: "Alert based on provided query",
		Query:       query,
		Condition: AlertCondition{
			Type:       "threshold",
			Threshold:  threshold,
			Operator:   "more_than",
			TimeWindow: timeWindow,
		},
		Schedule: AlertSchedule{
			Frequency:     "1m",
			ActiveWindows: "always",
		},
		Configuration: map[string]interface{}{
			"alert_type": "logs_threshold",
			"condition": map[string]interface{}{
				"type":        "more_than",
				"threshold":   threshold,
				"time_window": timeWindow,
			},
		},
		Explanation: fmt.Sprintf("Triggers when more than %d logs matching the query occur within %s", threshold, timeWindow),
		BestPractices: []string{
			"Verify the query returns expected results before creating the alert",
			"Adjust threshold based on normal log volume",
			"Consider adding grouping to reduce alert noise",
		},
	}
}

// GetAuditLogTool retrieves recent audit entries for tool executions
type GetAuditLogTool struct {
	*BaseTool
}

// NewGetAuditLogTool creates a new GetAuditLogTool
func NewGetAuditLogTool(c *client.Client, l *zap.Logger) *GetAuditLogTool {
	return &GetAuditLogTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *GetAuditLogTool) Name() string { return "get_audit_log" }

// Description returns the tool description
func (t *GetAuditLogTool) Description() string {
	return `Retrieve audit log entries showing recent tool executions and operations.

**Use Cases:**
- Review what actions have been performed
- Debug issues by tracing operations
- Understand usage patterns
- Verify operations completed successfully

Note: Audit logs are stored in memory and are cleared on server restart.`
}

// InputSchema returns the input schema
func (t *GetAuditLogTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of entries to return (default: 50, max: 1000)",
				"default":     50,
				"minimum":     1,
				"maximum":     1000,
			},
			"tool": map[string]interface{}{
				"type":        "string",
				"description": "Filter by specific tool name (optional)",
			},
			"trace_id": map[string]interface{}{
				"type":        "string",
				"description": "Filter by trace ID to see all operations in a trace (optional)",
			},
		},
	}
}

// Execute executes the tool - note: this requires integration with the audit logger
func (t *GetAuditLogTool) Execute(_ context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	// This tool requires access to the audit logger which is initialized at server level
	// For now, return a placeholder message explaining how to access audit logs
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: `Audit logging is enabled. To view audit logs:

1. Check the server logs with log level 'debug' for detailed audit entries
2. Look for log entries with logger name "audit"
3. Each entry includes: timestamp, trace_id, tool name, operation, success status, duration

Example audit log entry:
{
  "level": "info",
  "logger": "audit",
  "timestamp": "2024-01-15T10:30:00Z",
  "trace_id": "abc123...",
  "tool": "query_logs",
  "operation": "query",
  "success": true,
  "duration": "1.234s"
}

To enable verbose audit logging, set LOG_LEVEL=debug in your environment.`,
			},
		},
	}, nil
}
