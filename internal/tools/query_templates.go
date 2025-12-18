// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file contains the query templates library for common log analysis patterns.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// QueryTemplate represents a pre-built query pattern for common scenarios
type QueryTemplate struct {
	Name        string              `json:"name"`
	Category    string              `json:"category"`
	Description string              `json:"description"`
	Query       string              `json:"query"`
	Parameters  []TemplateParameter `json:"parameters,omitempty"`
	UseCases    []string            `json:"use_cases"`
	Tips        []string            `json:"tips,omitempty"`
}

// TemplateParameter describes a customizable part of the query
type TemplateParameter struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Placeholder string `json:"placeholder"`
	Example     string `json:"example"`
	Required    bool   `json:"required"`
}

// QueryTemplatesTool provides pre-built query patterns for common log analysis scenarios
type QueryTemplatesTool struct {
	*BaseTool
}

// NewQueryTemplatesTool creates a new QueryTemplatesTool
func NewQueryTemplatesTool(c *client.Client, l *zap.Logger) *QueryTemplatesTool {
	return &QueryTemplatesTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *QueryTemplatesTool) Name() string { return "get_query_templates" }

// Description returns the tool description
func (t *QueryTemplatesTool) Description() string {
	return `Get pre-built DataPrime query templates for common log analysis scenarios.

**Best for:** Learning query patterns, quick-starting analysis, discovering best practices.

**Categories available:**
- discovery: Pattern discovery and root cause analysis
- error: Error investigation and debugging
- performance: Latency and performance analysis
- security: Security auditing and threat detection
- health: Application health monitoring
- usage: Resource utilization tracking
- audit: User activity and compliance

**Usage:**
1. Call without parameters to list all templates
2. Specify category to filter templates
3. Specify name for a specific template with full details

**Related tools:** query_logs, build_query, explain_query`
}

// InputSchema returns the input schema
func (t *QueryTemplatesTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Filter templates by category",
				"enum":        []string{"discovery", "error", "performance", "security", "health", "usage", "audit"},
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Get a specific template by name",
			},
			"application": map[string]interface{}{
				"type":        "string",
				"description": "Application name to substitute in templates (optional)",
			},
			"time_range": map[string]interface{}{
				"type":        "string",
				"description": "Time range to substitute in templates, e.g., '1h', '24h' (optional)",
			},
		},
	}
}

// Execute returns query templates based on filters
func (t *QueryTemplatesTool) Execute(_ context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	category, _ := GetStringParam(args, "category", false)
	name, _ := GetStringParam(args, "name", false)
	application, _ := GetStringParam(args, "application", false)
	timeRange, _ := GetStringParam(args, "time_range", false)

	templates := getQueryTemplates()

	// Filter by name if specified
	if name != "" {
		for _, tmpl := range templates {
			if strings.EqualFold(tmpl.Name, name) {
				// Substitute parameters if provided
				result := substituteTemplateParams(tmpl, application, timeRange)
				return formatTemplateResult(result)
			}
		}
		return NewToolResultError(fmt.Sprintf("Template '%s' not found. Use get_query_templates without name to list all available templates.", name)), nil
	}

	// Filter by category if specified
	if category != "" {
		var filtered []QueryTemplate
		for _, tmpl := range templates {
			if strings.EqualFold(tmpl.Category, category) {
				filtered = append(filtered, tmpl)
			}
		}
		if len(filtered) == 0 {
			return NewToolResultError(fmt.Sprintf("No templates found for category '%s'", category)), nil
		}
		templates = filtered
	}

	// Return template summaries
	return formatTemplateList(templates)
}

// substituteTemplateParams replaces placeholders in a template
func substituteTemplateParams(tmpl QueryTemplate, application, timeRange string) QueryTemplate {
	result := tmpl
	if application != "" {
		result.Query = strings.ReplaceAll(result.Query, "{APPLICATION}", application)
		result.Query = strings.ReplaceAll(result.Query, "{application}", application)
	}
	if timeRange != "" {
		result.Query = strings.ReplaceAll(result.Query, "{TIME_RANGE}", timeRange)
		result.Query = strings.ReplaceAll(result.Query, "{time_range}", timeRange)
	}
	return result
}

// formatTemplateResult formats a single template with full details
func formatTemplateResult(tmpl QueryTemplate) (*mcp.CallToolResult, error) {
	result, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Failed to format template: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(result),
			},
		},
	}, nil
}

// formatTemplateList formats a list of templates with summaries
func formatTemplateList(templates []QueryTemplate) (*mcp.CallToolResult, error) {
	var summaries []map[string]interface{}
	for _, tmpl := range templates {
		summaries = append(summaries, map[string]interface{}{
			"name":        tmpl.Name,
			"category":    tmpl.Category,
			"description": tmpl.Description,
		})
	}

	result, err := json.MarshalIndent(map[string]interface{}{
		"templates": summaries,
		"usage":     "Use get_query_templates with name='<template_name>' to get full template details including the query",
	}, "", "  ")
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Failed to format templates: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(result),
			},
		},
	}, nil
}

// getQueryTemplates returns all available query templates
func getQueryTemplates() []QueryTemplate {
	return []QueryTemplate{
		// Discovery Templates (Root Cause Analysis)
		// These help find patterns when you don't know what to look for
		{
			Name:        "error_hotspots",
			Category:    "discovery",
			Description: "Find which services have the most errors - start investigations here",
			Query:       "source logs | filter $m.severity >= 5 | groupby $l.applicationname, $l.subsystemname | aggregate count() as error_count | sortby -error_count | limit 20",
			UseCases: []string{
				"Start incident investigation without prior knowledge",
				"Identify the source of problems",
				"Prioritize which service to investigate first",
			},
			Tips: []string{
				"This is often the first query in an investigation",
				"High error counts indicate the problem area",
				"Follow up with error_details on top services",
			},
		},
		{
			Name:        "anomaly_detection",
			Category:    "discovery",
			Description: "Find services with unusually high error rates",
			Query:       "source logs | filter $m.severity >= 5 | groupby $l.applicationname | aggregate count() as error_count | filter error_count > 10 | sortby -error_count | limit 20",
			UseCases: []string{
				"Detect services behaving abnormally",
				"Find issues even in low-traffic services",
				"Identify degradation before it becomes critical",
			},
			Tips: []string{
				"Error rate > 1% typically indicates problems",
				"Look at both rate AND absolute count",
				"Compare with historical baseline",
			},
		},
		{
			Name:        "top_error_messages",
			Category:    "discovery",
			Description: "Group similar errors to find the most common problems",
			Query:       "source logs | filter $m.severity >= 5 | groupby $d.message:string | aggregate count() as occurrences, min($m.timestamp) as first_seen, max($m.timestamp) as last_seen | filter occurrences >= 3 | sortby -occurrences | limit 30",
			UseCases: []string{
				"Find the most impactful errors",
				"Identify systemic issues vs one-offs",
				"Prioritize fixes by impact",
			},
			Tips: []string{
				"High occurrence count = high impact",
				"Check first_seen vs last_seen for duration",
				"Similar messages indicate same root cause",
			},
		},
		{
			Name:        "noise_filtered_errors",
			Category:    "discovery",
			Description: "Error analysis excluding health checks and known noise",
			Query:       "source logs | filter $m.severity >= 5 && !$d.message:string.toLowerCase().contains('health') && !$d.message:string.toLowerCase().contains('ping') && !$d.message:string.toLowerCase().contains('heartbeat') && !$d.message:string.toLowerCase().contains('metrics') | groupby $l.applicationname | aggregate count() as errors | sortby -errors | limit 20",
			UseCases: []string{
				"Find real errors without health check noise",
				"Cleaner investigation starting point",
				"Focus on application errors only",
			},
			Tips: []string{
				"Add more exclusions for your environment",
				"Good for production systems with monitoring",
				"Compare with unfiltered to verify exclusions",
			},
		},
		{
			Name:        "traffic_overview",
			Category:    "discovery",
			Description: "See overall traffic patterns by service",
			Query:       "source logs | groupby $l.applicationname | aggregate count() as volume, approx_count_distinct($l.subsystemname) as components | sortby -volume | limit 20",
			UseCases: []string{
				"Understand system topology",
				"Find unexpected traffic patterns",
				"Identify services to investigate",
			},
			Tips: []string{
				"Unexpected high volume may indicate problems",
				"Compare with normal patterns",
				"Use before drilling into specific services",
			},
		},
		{
			Name:        "recent_changes",
			Category:    "discovery",
			Description: "Detect recent deployments or restarts that might explain issues",
			Query:       "source logs | filter $d.message:string.contains('start') || $d.message:string.contains('deploy') || $d.message:string.contains('version') || $d.message:string.contains('initializ') || $d.message:string.contains('shutdown') | groupby $l.applicationname | aggregate count() as events, min($m.timestamp) as earliest, max($m.timestamp) as latest | filter events >= 2 | sortby -latest | limit 20",
			UseCases: []string{
				"Correlate issues with recent changes",
				"Find which services were recently deployed",
				"Identify crash loops or restarts",
			},
			Tips: []string{
				"Issues often follow deployments",
				"Multiple events may indicate crash loops",
				"Compare timing with error onset",
			},
		},

		// Error Investigation Templates
		{
			Name:        "error_spike",
			Category:    "error",
			Description: "Find error spikes in the last hour grouped by application",
			Query:       "source logs | filter $m.severity >= 5 | groupby $l.applicationname | aggregate count() as error_count | sortby -error_count | limit 20",
			UseCases: []string{
				"Identify which applications are producing the most errors",
				"Quick triage during incidents",
				"Prioritize debugging efforts",
			},
			Tips: []string{
				"Severity 5 = Error, 6 = Critical",
				"Expand time range if no results",
				"Follow up with error_details template for specific app",
			},
		},
		{
			Name:        "error_details",
			Category:    "error",
			Description: "Get detailed error logs for a specific application",
			Query:       "source logs | filter $l.applicationname == '{APPLICATION}' && $m.severity >= 5 | select $m.timestamp, $m.severity, $l.subsystemname, $d.message, $d.error, $d.stack_trace | sortby -$m.timestamp | limit 100",
			Parameters: []TemplateParameter{
				{Name: "APPLICATION", Description: "Application name to investigate", Placeholder: "{APPLICATION}", Example: "payment-service", Required: true},
			},
			UseCases: []string{
				"Deep dive into errors for a specific service",
				"Find error messages and stack traces",
				"Identify error patterns",
			},
			Tips: []string{
				"Replace {APPLICATION} with your app name",
				"Add time filters for specific incidents",
				"Look for common patterns in error messages",
			},
		},
		{
			Name:        "error_timeline",
			Category:    "error",
			Description: "Error rate over time for trend analysis",
			Query:       "source logs | filter $m.severity >= 5 | groupby roundTime($m.timestamp, 1m) as time_bucket | aggregate count() as errors | sortby time_bucket",
			UseCases: []string{
				"Visualize error trends over time",
				"Identify when errors started",
				"Correlate with deployments or changes",
			},
			Tips: []string{
				"Adjust roundTime interval for different granularity (1m, 5m, 1h)",
				"Compare with normal periods",
				"Use for dashboard creation",
			},
		},

		// Performance Analysis Templates
		{
			Name:        "slow_requests",
			Category:    "performance",
			Description: "Find slow requests with response time above threshold",
			Query:       "source logs | filter $d.response_time_ms > 1000 | select $m.timestamp, $l.applicationname, $d.endpoint, $d.response_time_ms, $d.status_code | sortby -$d.response_time_ms | limit 50",
			Parameters: []TemplateParameter{
				{Name: "threshold_ms", Description: "Response time threshold in milliseconds", Placeholder: "1000", Example: "500", Required: false},
			},
			UseCases: []string{
				"Identify performance bottlenecks",
				"Find slowest endpoints",
				"SLA compliance checking",
			},
			Tips: []string{
				"Adjust threshold based on your SLOs",
				"Check if slow requests correlate with errors",
				"Group by endpoint to find problematic routes",
			},
		},
		{
			Name:        "latency_percentiles",
			Category:    "performance",
			Description: "Calculate latency percentiles by endpoint",
			Query:       "source logs | filter $d.response_time_ms > 0 | groupby $d.endpoint | aggregate percentile($d.response_time_ms, 50) as p50, percentile($d.response_time_ms, 95) as p95, percentile($d.response_time_ms, 99) as p99, count() as requests | sortby -requests | limit 20",
			UseCases: []string{
				"Understand latency distribution",
				"Set realistic SLO targets",
				"Identify outlier endpoints",
			},
			Tips: []string{
				"P50 = typical user experience",
				"P95/P99 = worst case scenarios",
				"Compare across time periods",
			},
		},
		{
			Name:        "throughput_analysis",
			Category:    "performance",
			Description: "Request throughput over time by application",
			Query:       "source logs | filter $d.request_id != '' | groupby roundTime($m.timestamp, 1m) as time_bucket, $l.applicationname | aggregate count() as requests | sortby time_bucket",
			UseCases: []string{
				"Monitor traffic patterns",
				"Capacity planning",
				"Detect traffic anomalies",
			},
			Tips: []string{
				"Adjust roundTime interval for different granularity (1m, 5m, 1h)",
				"Compare with baseline periods",
				"Useful for dashboard widgets",
			},
		},

		// Security Audit Templates
		{
			Name:        "auth_failures",
			Category:    "security",
			Description: "Find authentication failures grouped by source",
			Query:       "source logs | filter $d.event_type == 'auth_failure' || $d.message.contains('authentication failed') || $d.message.contains('invalid credentials') | groupby $d.source_ip, $d.username | aggregate count() as failures | filter failures > 3 | sortby -failures",
			UseCases: []string{
				"Detect brute force attempts",
				"Identify compromised accounts",
				"Security incident investigation",
			},
			Tips: []string{
				"Threshold of 3 filters noise",
				"Group by IP to detect attacks",
				"Follow up with IP geolocation",
			},
		},
		{
			Name:        "privilege_escalation",
			Category:    "security",
			Description: "Detect privilege escalation attempts",
			Query:       "source logs | filter $d.event_type.contains('privilege') || $d.message.contains('sudo') || $d.message.contains('root access') || $d.message.contains('admin') | select $m.timestamp, $l.applicationname, $d.username, $d.action, $d.message | sortby -$m.timestamp | limit 100",
			UseCases: []string{
				"Detect unauthorized privilege changes",
				"Audit admin actions",
				"Compliance reporting",
			},
			Tips: []string{
				"Customize patterns for your environment",
				"Correlate with known admin activities",
				"Set up alerts for critical matches",
			},
		},
		{
			Name:        "sensitive_data_access",
			Category:    "security",
			Description: "Track access to sensitive resources",
			Query:       "source logs | filter $d.resource.contains('pii') || $d.resource.contains('secrets') || $d.resource.contains('credentials') || $d.endpoint.contains('/admin') | select $m.timestamp, $d.username, $d.resource, $d.action, $d.source_ip | sortby -$m.timestamp | limit 100",
			UseCases: []string{
				"PII access auditing",
				"Secrets management monitoring",
				"Compliance evidence gathering",
			},
			Tips: []string{
				"Adjust resource patterns for your data",
				"Export for compliance reports",
				"Set up alerts for unusual access",
			},
		},

		// Health Monitoring Templates
		{
			Name:        "service_health",
			Category:    "health",
			Description: "Overall health summary by service",
			Query:       "source logs | filter $m.severity >= 5 | groupby $l.applicationname | aggregate count() as error_count | sortby -error_count | limit 20",
			UseCases: []string{
				"Quick health overview",
				"Identify troubled services",
				"SRE dashboards",
			},
			Tips: []string{
				"Error rate > 1% typically needs attention",
				"Compare with historical baselines",
				"Great for morning health checks",
			},
		},
		{
			Name:        "heartbeat_check",
			Category:    "health",
			Description: "Verify services are logging (heartbeat)",
			Query:       "source logs | groupby $l.applicationname | aggregate max($m.timestamp) as last_seen, count() as log_count | sortby last_seen",
			UseCases: []string{
				"Detect silent failures",
				"Verify logging is working",
				"Find stale services",
			},
			Tips: []string{
				"Services missing from results may be down",
				"Check last_seen vs current time",
				"Set up alerts for missing heartbeats",
			},
		},
		{
			Name:        "restart_detection",
			Category:    "health",
			Description: "Detect service restarts and crashes",
			Query:       "source logs | filter $d.message.contains('starting') || $d.message.contains('started') || $d.message.contains('shutdown') || $d.message.contains('terminated') || $d.message.contains('OOMKilled') | select $m.timestamp, $l.applicationname, $l.subsystemname, $d.message | sortby -$m.timestamp | limit 50",
			UseCases: []string{
				"Detect crash loops",
				"Track deployment rollouts",
				"Investigate instability",
			},
			Tips: []string{
				"Multiple restarts indicate problems",
				"OOMKilled suggests memory issues",
				"Correlate with error spikes",
			},
		},

		// Usage Analysis Templates
		{
			Name:        "top_endpoints",
			Category:    "usage",
			Description: "Most frequently called endpoints",
			Query:       "source logs | filter $d.endpoint != '' | groupby $d.endpoint | aggregate count() as calls, avg($d.response_time_ms) as avg_latency | sortby -calls | limit 20",
			UseCases: []string{
				"Identify hot endpoints",
				"Capacity planning",
				"Cost allocation",
			},
			Tips: []string{
				"Focus optimization on top endpoints",
				"High calls + high latency = priority fix",
				"Use for API documentation priorities",
			},
		},
		{
			Name:        "user_activity",
			Category:    "usage",
			Description: "User activity summary",
			Query:       "source logs | filter $d.user_id != '' | groupby $d.user_id | aggregate count() as actions, approx_count_distinct($d.endpoint) as unique_endpoints | sortby -actions | limit 50",
			UseCases: []string{
				"Identify power users",
				"Detect anomalous behavior",
				"User engagement metrics",
			},
			Tips: []string{
				"Unusually high activity may indicate automation or abuse",
				"Low unique_endpoints may indicate bots",
				"Compare with user segments",
			},
		},
		{
			Name:        "data_volume",
			Category:    "usage",
			Description: "Log volume by application over time",
			Query:       "source logs | groupby roundTime($m.timestamp, 1h) as time_bucket, $l.applicationname | aggregate count() as logs | sortby time_bucket",
			UseCases: []string{
				"Cost monitoring",
				"Capacity planning",
				"Detect logging anomalies",
			},
			Tips: []string{
				"Sudden spikes may indicate problems",
				"Use for log retention decisions",
				"Identify chatty applications",
			},
		},

		// Audit and Compliance Templates
		{
			Name:        "config_changes",
			Category:    "audit",
			Description: "Track configuration changes",
			Query:       "source logs | filter $d.event_type.contains('config') || $d.message.contains('configuration changed') || $d.message.contains('settings updated') | select $m.timestamp, $d.username, $d.resource, $d.old_value, $d.new_value, $d.message | sortby -$m.timestamp | limit 100",
			UseCases: []string{
				"Change management audit",
				"Troubleshoot config-related issues",
				"Compliance evidence",
			},
			Tips: []string{
				"Correlate changes with incidents",
				"Export for change management tickets",
				"Set up alerts for critical configs",
			},
		},
		{
			Name:        "data_exports",
			Category:    "audit",
			Description: "Track data export activities",
			Query:       "source logs | filter $d.action.contains('export') || $d.action.contains('download') || $d.message.contains('exported') | select $m.timestamp, $d.username, $d.resource, $d.record_count, $d.destination | sortby -$m.timestamp | limit 100",
			UseCases: []string{
				"Data loss prevention",
				"Compliance auditing",
				"Detect data exfiltration",
			},
			Tips: []string{
				"Large exports need attention",
				"Unusual destinations are suspicious",
				"Compare with business justifications",
			},
		},
		{
			Name:        "api_key_usage",
			Category:    "audit",
			Description: "Track API key usage patterns",
			Query:       "source logs | filter $d.api_key_id != '' || $d.auth_type == 'api_key' | groupby $d.api_key_id, $l.applicationname | aggregate count() as calls, approx_count_distinct($d.source_ip) as unique_ips | sortby -calls | limit 50",
			UseCases: []string{
				"API key auditing",
				"Detect key sharing/abuse",
				"Key rotation planning",
			},
			Tips: []string{
				"Multiple IPs per key may indicate sharing",
				"Inactive keys should be rotated",
				"Monitor for unusual patterns",
			},
		},
	}
}
