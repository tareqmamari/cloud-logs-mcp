// Package resources provides MCP resource handlers for the IBM Cloud Logs server.
// Resources expose read-only data to MCP clients for context and status information.
package resources

import (
	"context"
	"encoding/json"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/config"
	"github.com/tareqmamari/logs-mcp-server/internal/metrics"
)

// Registry holds all registered resources and their handlers
type Registry struct {
	config  *config.Config
	metrics *metrics.Metrics
	logger  *zap.Logger
	version string
}

// NewRegistry creates a new resource registry
func NewRegistry(cfg *config.Config, m *metrics.Metrics, logger *zap.Logger, version string) *Registry {
	return &Registry{
		config:  cfg,
		metrics: m,
		logger:  logger,
		version: version,
	}
}

// RegisteredResource represents a resource with its definition and handler
type RegisteredResource struct {
	Resource *mcp.Resource
	Handler  mcp.ResourceHandler
}

// GetResources returns all registered resources with their handlers
func (r *Registry) GetResources() []RegisteredResource {
	return []RegisteredResource{
		r.aboutResource(),
		r.configResource(),
		r.metricsResource(),
		r.healthResource(),
	}
}

// aboutResource returns the about://service resource with service aliases and description
func (r *Registry) aboutResource() RegisteredResource {
	return RegisteredResource{
		Resource: &mcp.Resource{
			URI:         "about://service",
			Name:        "about://service",
			Title:       "About IBM Cloud Logs",
			Description: "Service information, aliases, and capabilities",
			MIMEType:    "application/json",
		},
		Handler: func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			aboutInfo := map[string]interface{}{
				"service": map[string]interface{}{
					"name":        "IBM Cloud Logs",
					"description": "Fully managed cloud-native log management service for IBM Cloud",
					"aliases": []string{
						"IBM Cloud Logs",
						"ICL",
						"Cloud Logs",
						"Logs",
					},
					"powered_by": "Coralogix",
				},
				"query_language": map[string]interface{}{
					"name":    "DataPrime",
					"type":    "Piped syntax query language",
					"example": "source logs | filter $m.severity >= ERROR | limit 100",
				},
				"data_tiers": map[string]interface{}{
					"archive": map[string]string{
						"aliases":     "COS, cold storage, archive tier",
						"description": "Long-term storage with lower cost, slightly higher query latency",
					},
					"frequent_search": map[string]string{
						"aliases":     "Priority Insights, PI, hot tier, real-time",
						"description": "Fast queries for recent data, limited retention",
					},
				},
				"mcp_server": map[string]interface{}{
					"version":      r.version,
					"tool_count":   86,
					"capabilities": []string{"tools", "prompts", "resources"},
				},
			}

			content, err := json.MarshalIndent(aboutInfo, "", "  ")
			if err != nil {
				r.logger.Error("Failed to marshal about info", zap.Error(err))
				return nil, err
			}

			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      "about://service",
						MIMEType: "application/json",
						Text:     string(content),
					},
				},
			}, nil
		},
	}
}

// configResource returns the config://current resource
func (r *Registry) configResource() RegisteredResource {
	return RegisteredResource{
		Resource: &mcp.Resource{
			URI:         "config://current",
			Name:        "config://current",
			Title:       "Server Configuration",
			Description: "Current IBM Cloud Logs MCP server configuration (sensitive values masked)",
			MIMEType:    "application/json",
		},
		Handler: func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			// Create a safe config representation (mask sensitive values)
			safeConfig := map[string]interface{}{
				"service_url":            r.config.ServiceURL,
				"region":                 r.config.Region,
				"instance_id":            r.config.InstanceID,
				"instance_name":          r.config.InstanceName,
				"timeout":                r.config.Timeout.String(),
				"query_timeout":          r.config.QueryTimeout.String(),
				"background_timeout":     r.config.BackgroundPollTimeout.String(),
				"bulk_operation_timeout": r.config.BulkOperationTimeout.String(),
				"max_retries":            r.config.MaxRetries,
				"rate_limit":             r.config.RateLimit,
				"rate_limit_burst":       r.config.RateLimitBurst,
				"rate_limit_enabled":     r.config.EnableRateLimit,
				"tls_verify":             r.config.TLSVerify,
				"tracing_enabled":        r.config.EnableTracing,
				"audit_log_enabled":      r.config.EnableAuditLog,
				"log_level":              r.config.LogLevel,
				"log_format":             r.config.LogFormat,
				"server_version":         r.version,
				"api_key_configured":     r.config.APIKey != "",
			}

			content, err := json.MarshalIndent(safeConfig, "", "  ")
			if err != nil {
				r.logger.Error("Failed to marshal config", zap.Error(err))
				return nil, err
			}

			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      "config://current",
						MIMEType: "application/json",
						Text:     string(content),
					},
				},
			}, nil
		},
	}
}

// metricsResource returns the metrics://server resource
func (r *Registry) metricsResource() RegisteredResource {
	return RegisteredResource{
		Resource: &mcp.Resource{
			URI:         "metrics://server",
			Name:        "metrics://server",
			Title:       "Server Metrics",
			Description: "Operational metrics including request counts, latency, and tool usage statistics",
			MIMEType:    "application/json",
		},
		Handler: func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			stats := r.metrics.GetStats()

			// Convert to JSON-friendly format
			metricsData := map[string]interface{}{
				"requests": map[string]interface{}{
					"total":      stats.TotalRequests,
					"successful": stats.SuccessfulRequests,
					"failed":     stats.FailedRequests,
					"retried":    stats.RetriedRequests,
				},
				"rate_limiting": map[string]interface{}{
					"hits": stats.RateLimitHits,
				},
				"latency": map[string]interface{}{
					"average_ms": stats.AverageLatency.Milliseconds(),
					"max_ms":     stats.MaxLatency.Milliseconds(),
					"min_ms":     stats.MinLatency.Milliseconds(),
				},
				"errors_by_status": stats.ErrorsByStatus,
				"tools": map[string]interface{}{
					"usage":   stats.ToolUsage,
					"errors":  stats.ToolErrors,
					"latency": formatToolLatency(stats.ToolLatency),
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}

			content, err := json.MarshalIndent(metricsData, "", "  ")
			if err != nil {
				r.logger.Error("Failed to marshal metrics", zap.Error(err))
				return nil, err
			}

			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      "metrics://server",
						MIMEType: "application/json",
						Text:     string(content),
					},
				},
			}, nil
		},
	}
}

// healthResource returns the health://status resource
func (r *Registry) healthResource() RegisteredResource {
	return RegisteredResource{
		Resource: &mcp.Resource{
			URI:         "health://status",
			Name:        "health://status",
			Title:       "Health Status",
			Description: "Current health status of the MCP server and IBM Cloud Logs connectivity",
			MIMEType:    "application/json",
		},
		Handler: func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			// Simple health status based on metrics
			stats := r.metrics.GetStats()

			var status string
			var statusMessage string
			errorRate := float64(0)
			if stats.TotalRequests > 0 {
				errorRate = float64(stats.FailedRequests) / float64(stats.TotalRequests) * 100
			}

			if errorRate > 50 {
				status = "unhealthy"
				statusMessage = "High error rate detected"
			} else if errorRate > 10 {
				status = "degraded"
				statusMessage = "Elevated error rate"
			} else {
				status = "healthy"
				statusMessage = "All systems operational"
			}

			healthData := map[string]interface{}{
				"status":  status,
				"message": statusMessage,
				"details": map[string]interface{}{
					"error_rate_percent": errorRate,
					"total_requests":     stats.TotalRequests,
					"failed_requests":    stats.FailedRequests,
					"rate_limit_hits":    stats.RateLimitHits,
				},
				"server": map[string]interface{}{
					"version":  r.version,
					"region":   r.config.Region,
					"instance": r.config.InstanceID,
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}

			content, err := json.MarshalIndent(healthData, "", "  ")
			if err != nil {
				r.logger.Error("Failed to marshal health status", zap.Error(err))
				return nil, err
			}

			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      "health://status",
						MIMEType: "application/json",
						Text:     string(content),
					},
				},
			}, nil
		},
	}
}

// formatToolLatency converts time.Duration map to milliseconds for JSON
func formatToolLatency(latency map[string]time.Duration) map[string]int64 {
	result := make(map[string]int64, len(latency))
	for tool, duration := range latency {
		result[tool] = duration.Milliseconds()
	}
	return result
}

// GetResourceTemplates returns resource templates for common configurations
// These templates help LLMs understand the structure of resources they can create
func (r *Registry) GetResourceTemplates() []mcp.ResourceTemplate {
	return []mcp.ResourceTemplate{
		{
			URITemplate: "template://alert/{name}",
			Name:        "Alert Configuration Template",
			Description: "Template for creating alert configurations. Use this to understand the structure of alerts before creating them with create_alert tool.",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "template://dashboard/{name}",
			Name:        "Dashboard Configuration Template",
			Description: "Template for creating dashboard configurations. Use this to understand widget layouts and query structures before creating dashboards.",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "template://query/{syntax}",
			Name:        "Query Syntax Template",
			Description: "Template showing query syntax examples. Supports 'dataprime' and 'lucene' syntaxes. Use this to understand query structure before running query_logs.",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "template://webhook/{type}",
			Name:        "Webhook Configuration Template",
			Description: "Template for creating outgoing webhook configurations. Supports types: 'slack', 'pagerduty', 'generic'.",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "template://policy/{type}",
			Name:        "TCO Policy Template",
			Description: "Template for creating TCO (Total Cost of Ownership) policy configurations. Supports types: 'logs' and 'spans'.",
			MIMEType:    "application/json",
		},
	}
}

// GetTemplateHandler returns a handler for resource templates
func (r *Registry) GetTemplateHandler() mcp.ResourceHandler {
	return func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri := req.Params.URI

		var content map[string]interface{}
		var templateName string

		switch {
		case matchTemplate(uri, "template://alert/"):
			templateName = extractTemplateName(uri, "template://alert/")
			content = getAlertTemplate(templateName)
		case matchTemplate(uri, "template://dashboard/"):
			templateName = extractTemplateName(uri, "template://dashboard/")
			content = getDashboardTemplate(templateName)
		case matchTemplate(uri, "template://query/"):
			syntax := extractTemplateName(uri, "template://query/")
			content = getQueryTemplate(syntax)
		case matchTemplate(uri, "template://webhook/"):
			webhookType := extractTemplateName(uri, "template://webhook/")
			content = getWebhookTemplate(webhookType)
		case matchTemplate(uri, "template://policy/"):
			policyType := extractTemplateName(uri, "template://policy/")
			content = getPolicyTemplate(policyType)
		default:
			content = map[string]interface{}{
				"error": "Unknown template type",
				"available_templates": []string{
					"template://alert/{name}",
					"template://dashboard/{name}",
					"template://query/{syntax}",
					"template://webhook/{type}",
					"template://policy/{type}",
				},
			}
		}

		jsonContent, err := json.MarshalIndent(content, "", "  ")
		if err != nil {
			r.logger.Error("Failed to marshal template", zap.Error(err))
			return nil, err
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "application/json",
					Text:     string(jsonContent),
				},
			},
		}, nil
	}
}

func matchTemplate(uri, prefix string) bool {
	return len(uri) > len(prefix) && uri[:len(prefix)] == prefix
}

func extractTemplateName(uri, prefix string) string {
	return uri[len(prefix):]
}

// getAlertTemplate returns an alert configuration template
func getAlertTemplate(name string) map[string]interface{} {
	return map[string]interface{}{
		"_template_info": map[string]interface{}{
			"description": "Alert configuration template for IBM Cloud Logs",
			"usage":       "Modify this template and use create_alert tool",
			"name":        name,
		},
		"alert": map[string]interface{}{
			"name":        name,
			"description": "Description of the alert purpose",
			"is_active":   true,
			"severity":    "error",
			"condition": map[string]interface{}{
				"type":                "logs_immediate",
				"threshold":           10,
				"time_window_minutes": 5,
			},
			"notification_groups": []map[string]interface{}{
				{
					"group_by_fields": []string{"applicationName", "severity"},
					"notifications": []map[string]interface{}{
						{
							"integration_id":           "<webhook-id>",
							"retriggering_period_mins": 60,
						},
					},
				},
			},
			"filters": map[string]interface{}{
				"text":         "severity:error",
				"filter_type":  "text",
				"applications": []string{"my-app"},
				"severities":   []string{"error", "critical"},
				"subsystems":   []string{},
			},
			"meta_labels": []map[string]string{
				{"key": "team", "value": "platform"},
				{"key": "environment", "value": "production"},
			},
		},
		"_related_tools": []string{
			"create_alert",
			"list_alerts",
			"list_outgoing_webhooks",
			"create_alert_def",
		},
	}
}

// getDashboardTemplate returns a dashboard configuration template
func getDashboardTemplate(name string) map[string]interface{} {
	return map[string]interface{}{
		"_template_info": map[string]interface{}{
			"description": "Dashboard configuration template for IBM Cloud Logs",
			"usage":       "Modify this template and use create_dashboard tool",
			"name":        name,
		},
		"name":        name,
		"description": "Dashboard description",
		"layout": map[string]interface{}{
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
									"title": "Error Count Over Time",
									"definition": map[string]interface{}{
										"line_chart": map[string]interface{}{
											"query_definitions": []map[string]interface{}{
												{
													"id": "query-1",
													"query": map[string]interface{}{
														"logs": map[string]interface{}{
															"lucene_query": map[string]interface{}{
																"value": "severity:>=5",
															},
															"aggregations": []map[string]interface{}{
																{"count": map[string]interface{}{}},
															},
															"group_bys": []map[string]interface{}{},
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
		"_widget_types": []string{
			"line_chart",
			"bar_chart",
			"pie_chart",
			"data_table",
			"gauge",
			"markdown",
		},
		"_related_tools": []string{
			"create_dashboard",
			"list_dashboards",
			"query_logs",
			"validate_query",
		},
	}
}

// getQueryTemplate returns a query syntax template
func getQueryTemplate(syntax string) map[string]interface{} {
	if syntax == "dataprime" {
		return map[string]interface{}{
			"_template_info": map[string]interface{}{
				"description": "DataPrime query syntax template for IBM Cloud Logs",
				"syntax":      "dataprime",
				"usage":       "Use these examples with query_logs tool",
			},
			"examples": []map[string]interface{}{
				{
					"name":        "Basic filter",
					"query":       "source logs | filter $d.severity == 'error'",
					"description": "Filter logs by severity field",
				},
				{
					"name":        "Count by application",
					"query":       "source logs | groupby $l.applicationName | count",
					"description": "Group logs by application and count",
				},
				{
					"name":        "Top errors",
					"query":       "source logs | filter $d.severity == 'error' | groupby $d.message | count | sort -count | limit 10",
					"description": "Find top 10 error messages",
				},
				{
					"name":        "Time-based aggregation",
					"query":       "source logs | filter $d.severity >= 4 | groupby $m.timestamp.bucket(15m) | count",
					"description": "Count high-severity logs per 15-minute bucket",
				},
				{
					"name":        "String contains",
					"query":       "source logs | filter $d.message.contains('timeout')",
					"description": "Find logs containing 'timeout'",
				},
			},
			"field_references": map[string]string{
				"$d.field": "Data field (from log payload)",
				"$l.field": "Label field (applicationName, subsystemName, etc.)",
				"$m.field": "Metadata field (timestamp, severity, etc.)",
			},
			"_related_tools": []string{
				"query_logs",
				"validate_query",
			},
		}
	}

	// Default to Lucene
	return map[string]interface{}{
		"_template_info": map[string]interface{}{
			"description": "Lucene query syntax template for IBM Cloud Logs",
			"syntax":      "lucene",
			"usage":       "Use these examples with query_logs tool",
		},
		"examples": []map[string]interface{}{
			{
				"name":        "Field exact match",
				"query":       "severity:error",
				"description": "Match exact value in field",
			},
			{
				"name":        "Range query",
				"query":       "severity:>=5",
				"description": "Match severity 5 or higher",
			},
			{
				"name":        "Wildcard",
				"query":       "message:*timeout*",
				"description": "Match messages containing 'timeout'",
			},
			{
				"name":        "Boolean AND",
				"query":       "severity:error AND applicationName:api-gateway",
				"description": "Combine conditions with AND",
			},
			{
				"name":        "Boolean OR",
				"query":       "severity:error OR severity:critical",
				"description": "Match either condition",
			},
			{
				"name":        "Phrase search",
				"query":       "message:\"connection refused\"",
				"description": "Match exact phrase",
			},
		},
		"operators": []string{
			"AND", "OR", "NOT", ":", ">=", "<=", ">", "<", "*", "?", "~",
		},
		"_related_tools": []string{
			"query_logs",
			"validate_query",
		},
	}
}

// getWebhookTemplate returns a webhook configuration template
func getWebhookTemplate(webhookType string) map[string]interface{} {
	base := map[string]interface{}{
		"_template_info": map[string]interface{}{
			"description": "Outgoing webhook configuration template",
			"type":        webhookType,
			"usage":       "Modify this template and use create_outgoing_webhook tool",
		},
	}

	switch webhookType {
	case "slack":
		base["webhook"] = map[string]interface{}{
			"type": "slack",
			"name": "Slack Alert Webhook",
			"url":  "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
			"slack_config": map[string]interface{}{
				"channel":    "#alerts",
				"username":   "IBM Cloud Logs",
				"icon_emoji": ":warning:",
			},
		}
	case "pagerduty":
		base["webhook"] = map[string]interface{}{
			"type": "pagerduty",
			"name": "PagerDuty Integration",
			"pagerduty_config": map[string]interface{}{
				"routing_key": "YOUR_PAGERDUTY_ROUTING_KEY",
				"severity":    "error",
			},
		}
	default:
		base["webhook"] = map[string]interface{}{
			"type": "generic",
			"name": "Generic Webhook",
			"url":  "https://your-endpoint.example.com/webhook",
			"generic_config": map[string]interface{}{
				"method": "POST",
				"headers": map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer YOUR_TOKEN",
				},
			},
		}
	}

	base["_related_tools"] = []string{
		"create_outgoing_webhook",
		"list_outgoing_webhooks",
		"create_alert",
	}

	return base
}

// getPolicyTemplate returns a TCO policy configuration template
func getPolicyTemplate(policyType string) map[string]interface{} {
	if policyType == "spans" {
		return map[string]interface{}{
			"_template_info": map[string]interface{}{
				"description": "TCO policy template for spans",
				"type":        "spans",
				"usage":       "Modify this template and use create_policy tool",
			},
			"policy": map[string]interface{}{
				"name":        "Spans TCO Policy",
				"description": "TCO policy for optimizing span storage",
				"priority":    "medium",
				"archive_retention": map[string]interface{}{
					"enabled": true,
					"days":    90,
				},
				"spans_filter": map[string]interface{}{
					"service_name":   []string{"my-service"},
					"operation_name": []string{},
					"min_duration":   "1ms",
				},
			},
			"_related_tools": []string{
				"create_policy",
				"list_policies",
			},
		}
	}

	// Default to logs
	return map[string]interface{}{
		"_template_info": map[string]interface{}{
			"description": "TCO policy template for logs",
			"type":        "logs",
			"usage":       "Modify this template and use create_policy tool",
		},
		"policy": map[string]interface{}{
			"name":        "Logs TCO Policy",
			"description": "TCO policy for optimizing log storage and costs",
			"priority":    "high",
			"archive_retention": map[string]interface{}{
				"enabled": true,
				"days":    30,
			},
			"logs_filter": map[string]interface{}{
				"applications": []string{"my-app"},
				"subsystems":   []string{},
				"severities":   []string{"debug", "verbose"},
			},
		},
		"_tco_tiers": map[string]string{
			"frequent_search": "Hot storage - fast queries, higher cost",
			"monitoring":      "Warm storage - balanced cost/performance",
			"compliance":      "Cold storage - archival, lowest cost",
		},
		"_related_tools": []string{
			"create_policy",
			"list_policies",
		},
	}
}
