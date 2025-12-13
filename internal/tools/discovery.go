// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements semantic tool discovery for LLMs.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// ToolRegistry maintains metadata for all registered tools
type ToolRegistry struct {
	tools   map[string]*ToolMetadata
	chains  []ToolChain
	intents map[string][]string // intent -> tool names
}

// ToolChain defines a sequence of tools for a workflow
type ToolChain struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Trigger     string   `json:"trigger"`   // Tool that starts chain
	Condition   string   `json:"condition"` // When to suggest chain
	Sequence    []string `json:"sequence"`  // Ordered tool sequence
	UseCases    []string `json:"use_cases"`
}

// ConfidenceLevel represents the confidence in tool matching
type ConfidenceLevel string

const (
	// ConfidenceHigh means we're very confident in the matched tools (0.8+)
	ConfidenceHigh ConfidenceLevel = "high"
	// ConfidenceMedium means moderate confidence, may need clarification (0.6-0.8)
	ConfidenceMedium ConfidenceLevel = "medium"
	// ConfidenceLow means low confidence, suggest alternatives (<0.6)
	ConfidenceLow ConfidenceLevel = "low"
)

// Confidence thresholds
const (
	HighConfidenceThreshold   = 0.8
	MediumConfidenceThreshold = 0.6
	// Tools returned per confidence level
	HighConfidenceMaxTools   = 3
	MediumConfidenceMaxTools = 5
	LowConfidenceMaxTools    = 10
)

// ConfidenceResult provides confidence metadata for discovery results
type ConfidenceResult struct {
	Level          ConfidenceLevel `json:"level"`
	Score          float64         `json:"score"`
	Explanation    string          `json:"explanation"`
	Alternatives   []string        `json:"alternatives,omitempty"`
	Clarifications []string        `json:"clarifications,omitempty"`
}

// DiscoveryResult represents the result of tool discovery
type DiscoveryResult struct {
	Intent          string                 `json:"intent"`
	Confidence      *ConfidenceResult      `json:"confidence"`
	MatchedTools    []ToolMatch            `json:"matched_tools"`
	SuggestedChain  *ToolChain             `json:"suggested_chain,omitempty"`
	AdaptiveChains  []*AdaptiveChain       `json:"adaptive_chains,omitempty"`
	SessionContext  map[string]interface{} `json:"session_context,omitempty"`
	Recommendations []string               `json:"recommendations,omitempty"`
}

// AdaptiveChain represents a tool chain learned from user behavior
type AdaptiveChain struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
	SuccessRate float64  `json:"success_rate"`
	UseCount    int      `json:"use_count"`
	Confidence  string   `json:"confidence"` // "high", "medium", "low"
	Source      string   `json:"source"`     // "learned", "suggested", "static"
}

// ToolMatch represents a tool that matches the discovery query
type ToolMatch struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Relevance   float64  `json:"relevance"` // 0.0-1.0 match score
	Reason      string   `json:"reason"`    // Why this tool was matched
	Categories  []string `json:"categories"`
	Complexity  string   `json:"complexity"`
	UseCases    []string `json:"use_cases,omitempty"`
}

// Global tool registry
var globalRegistry *ToolRegistry

// GetToolRegistry returns the global tool registry
func GetToolRegistry() *ToolRegistry {
	if globalRegistry == nil {
		globalRegistry = NewToolRegistry()
	}
	return globalRegistry
}

// NewToolRegistry creates a new tool registry with predefined metadata
func NewToolRegistry() *ToolRegistry {
	r := &ToolRegistry{
		tools:   make(map[string]*ToolMetadata),
		intents: make(map[string][]string),
	}
	r.registerAllTools()
	r.registerToolChains()
	r.buildIntentIndex()
	return r
}

// registerAllTools registers metadata for all tools
func (r *ToolRegistry) registerAllTools() {
	// Query tools
	r.tools["query_logs"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryQuery, CategoryObservability},
		Keywords:      []string{"query", "search", "logs", "find", "filter", "investigate", "error", "debug"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"Search logs", "Investigate errors", "Debug issues", "Find patterns"},
		RelatedTools:  []string{"build_query", "explain_query", "submit_background_query"},
		ChainPosition: ChainStarter,
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"events":   map[string]interface{}{"type": "array", "description": "Log entries"},
				"metadata": map[string]interface{}{"type": "object"},
			},
		},
	}

	r.tools["build_query"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryQuery, CategoryAIHelper},
		Keywords:      []string{"build", "construct", "create", "query", "dataprime", "help"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"Build queries without knowing syntax", "Learn DataPrime"},
		RelatedTools:  []string{"query_logs", "explain_query"},
		ChainPosition: ChainStarter,
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query":       map[string]interface{}{"type": "string", "description": "Generated DataPrime query"},
				"explanation": map[string]interface{}{"type": "string", "description": "How the query works"},
				"examples":    map[string]interface{}{"type": "array", "description": "Example variations"},
			},
		},
	}

	r.tools["explain_query"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryQuery, CategoryAIHelper},
		Keywords:      []string{"explain", "understand", "parse", "analyze", "query", "syntax"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"Understand complex queries", "Learn syntax", "Debug queries"},
		RelatedTools:  []string{"query_logs", "build_query"},
		ChainPosition: ChainStarter,
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"explanation":   map[string]interface{}{"type": "string", "description": "Human-readable explanation"},
				"components":    map[string]interface{}{"type": "array", "description": "Query components breakdown"},
				"optimizations": map[string]interface{}{"type": "array", "description": "Suggested optimizations"},
			},
		},
	}

	r.tools["submit_background_query"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryQuery},
		Keywords:      []string{"background", "async", "large", "query", "slow"},
		Complexity:    ComplexityIntermediate,
		UseCases:      []string{"Run large queries", "Async processing", "Avoid timeouts"},
		RelatedTools:  []string{"get_background_query_status", "get_background_query_data"},
		ChainPosition: ChainStarter,
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query_id": map[string]interface{}{"type": "string", "description": "Background query ID for status checks"},
				"status":   map[string]interface{}{"type": "string", "description": "Initial status (queued/running)"},
			},
		},
	}

	// Alert tools
	r.tools["list_alerts"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryAlert, CategoryObservability},
		Keywords:      []string{"list", "alerts", "monitoring", "triggered", "active"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"View all alerts", "Check monitoring status"},
		RelatedTools:  []string{"get_alert", "create_alert"},
		ChainPosition: ChainStarter,
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"alerts": map[string]interface{}{
					"type":        "array",
					"description": "List of alert objects",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id":       map[string]interface{}{"type": "string"},
							"name":     map[string]interface{}{"type": "string"},
							"severity": map[string]interface{}{"type": "string"},
							"state":    map[string]interface{}{"type": "string"},
						},
					},
				},
			},
		},
	}

	r.tools["create_alert"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryAlert, CategoryObservability},
		Keywords:      []string{"create", "alert", "monitoring", "notify", "threshold"},
		Complexity:    ComplexityIntermediate,
		UseCases:      []string{"Set up monitoring", "Create notifications", "Define thresholds"},
		RelatedTools:  []string{"suggest_alert", "list_alerts", "create_outgoing_webhook"},
		ChainPosition: ChainFinisher,
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id":      map[string]interface{}{"type": "string", "description": "Created alert ID"},
				"name":    map[string]interface{}{"type": "string", "description": "Alert name"},
				"message": map[string]interface{}{"type": "string", "description": "Success message"},
			},
		},
	}

	r.tools["suggest_alert"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryAlert, CategoryAIHelper},
		Keywords:      []string{"suggest", "recommend", "alert", "best practice", "template"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"Get alert recommendations", "Best practice alerts"},
		RelatedTools:  []string{"create_alert", "query_logs"},
		ChainPosition: ChainMiddle,
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"suggestions": map[string]interface{}{
					"type":        "array",
					"description": "List of alert suggestions with confidence scores",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name":        map[string]interface{}{"type": "string"},
							"description": map[string]interface{}{"type": "string"},
							"query":       map[string]interface{}{"type": "string"},
							"threshold":   map[string]interface{}{"type": "number"},
							"confidence":  map[string]interface{}{"type": "number"},
							"evidence":    map[string]interface{}{"type": "array"},
						},
					},
				},
			},
		},
	}

	// Dashboard tools
	r.tools["list_dashboards"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryDashboard, CategoryObservability},
		Keywords:      []string{"list", "dashboards", "visualize", "charts"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"View all dashboards", "Find visualizations"},
		RelatedTools:  []string{"get_dashboard", "create_dashboard"},
		ChainPosition: ChainStarter,
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"dashboards": map[string]interface{}{
					"type":        "array",
					"description": "List of dashboard summaries",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id":          map[string]interface{}{"type": "string"},
							"name":        map[string]interface{}{"type": "string"},
							"description": map[string]interface{}{"type": "string"},
						},
					},
				},
			},
		},
	}

	r.tools["create_dashboard"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryDashboard, CategoryObservability},
		Keywords:      []string{"create", "dashboard", "visualize", "chart", "widget"},
		Complexity:    ComplexityAdvanced,
		UseCases:      []string{"Create visualizations", "Build monitoring views"},
		RelatedTools:  []string{"list_dashboards", "query_logs"},
		ChainPosition: ChainFinisher,
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id":      map[string]interface{}{"type": "string", "description": "Created dashboard ID"},
				"name":    map[string]interface{}{"type": "string", "description": "Dashboard name"},
				"message": map[string]interface{}{"type": "string", "description": "Success message"},
			},
		},
	}

	// Workflow tools
	r.tools["investigate_incident"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryWorkflow, CategoryObservability},
		Keywords:      []string{"investigate", "incident", "debug", "troubleshoot", "root cause", "error"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"Investigate incidents", "Find root cause", "Debug errors"},
		RelatedTools:  []string{"query_logs", "list_alerts", "create_alert"},
		ChainPosition: ChainStarter,
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"investigation_id": map[string]interface{}{"type": "string", "description": "Investigation tracking ID"},
				"summary":          map[string]interface{}{"type": "string", "description": "Issue summary"},
				"findings":         map[string]interface{}{"type": "array", "description": "Key findings"},
				"severity":         map[string]interface{}{"type": "string", "description": "Assessed severity"},
				"suggestions":      map[string]interface{}{"type": "array", "description": "Next step suggestions"},
			},
		},
	}

	r.tools["health_check"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryWorkflow, CategoryObservability},
		Keywords:      []string{"health", "status", "check", "overview", "system"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"Check system health", "Get overview", "Quick status"},
		RelatedTools:  []string{"query_logs", "list_alerts"},
		ChainPosition: ChainStarter,
		OutputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"overall_status": map[string]interface{}{"type": "string", "enum": []string{"healthy", "warning", "critical"}},
				"checks": map[string]interface{}{
					"type":        "array",
					"description": "Individual health check results",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"component": map[string]interface{}{"type": "string"},
							"status":    map[string]interface{}{"type": "string"},
							"message":   map[string]interface{}{"type": "string"},
						},
					},
				},
				"recommendations": map[string]interface{}{"type": "array", "description": "Action recommendations"},
			},
		},
	}

	// Ingestion tools
	r.tools["ingest_logs"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryIngestion},
		Keywords:      []string{"ingest", "send", "push", "logs", "write"},
		Complexity:    ComplexityIntermediate,
		UseCases:      []string{"Send logs", "Test ingestion", "Import data"},
		RelatedTools:  []string{"query_logs"},
		ChainPosition: ChainFinisher,
	}

	// Policy tools
	r.tools["list_policies"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryPolicy},
		Keywords:      []string{"list", "policies", "retention", "routing"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"View policies", "Check retention settings"},
		RelatedTools:  []string{"get_policy", "create_policy"},
		ChainPosition: ChainStarter,
	}

	r.tools["create_policy"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryPolicy},
		Keywords:      []string{"create", "policy", "retention", "routing", "archive"},
		Complexity:    ComplexityIntermediate,
		UseCases:      []string{"Set up retention", "Configure routing"},
		RelatedTools:  []string{"list_policies"},
		ChainPosition: ChainFinisher,
	}

	// Stream tools
	r.tools["list_streams"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryStream},
		Keywords:      []string{"list", "streams", "export", "kafka"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"View streams", "Check exports"},
		RelatedTools:  []string{"create_stream"},
		ChainPosition: ChainStarter,
	}

	r.tools["create_stream"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryStream},
		Keywords:      []string{"create", "stream", "export", "kafka", "event streams"},
		Complexity:    ComplexityAdvanced,
		UseCases:      []string{"Export logs", "Set up streaming"},
		RelatedTools:  []string{"list_streams"},
		ChainPosition: ChainFinisher,
	}

	// Webhook tools
	r.tools["list_outgoing_webhooks"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryWebhook},
		Keywords:      []string{"list", "webhooks", "notifications", "slack", "pagerduty"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"View notifications", "Check integrations"},
		RelatedTools:  []string{"create_outgoing_webhook"},
		ChainPosition: ChainStarter,
	}

	r.tools["create_outgoing_webhook"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryWebhook},
		Keywords:      []string{"create", "webhook", "notification", "slack", "pagerduty", "integrate"},
		Complexity:    ComplexityIntermediate,
		UseCases:      []string{"Set up notifications", "Integrate with Slack/PagerDuty"},
		RelatedTools:  []string{"create_alert", "list_outgoing_webhooks"},
		ChainPosition: ChainMiddle,
	}

	// E2M tools
	r.tools["list_e2m"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryE2M, CategoryObservability},
		Keywords:      []string{"list", "e2m", "events", "metrics", "convert"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"View E2M mappings", "Check metrics generation"},
		RelatedTools:  []string{"create_e2m"},
		ChainPosition: ChainStarter,
	}

	r.tools["create_e2m"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryE2M, CategoryObservability},
		Keywords:      []string{"create", "e2m", "events", "metrics", "convert", "aggregate"},
		Complexity:    ComplexityAdvanced,
		UseCases:      []string{"Convert logs to metrics", "Create aggregations"},
		RelatedTools:  []string{"list_e2m", "query_logs"},
		ChainPosition: ChainFinisher,
	}

	// Data access tools
	r.tools["list_data_access_rules"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryDataAccess, CategorySecurity},
		Keywords:      []string{"list", "data access", "rules", "permissions", "security"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"View access rules", "Check permissions"},
		RelatedTools:  []string{"create_data_access_rule"},
		ChainPosition: ChainStarter,
	}

	// View tools
	r.tools["list_views"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryView},
		Keywords:      []string{"list", "views", "saved", "filters"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"View saved views", "Find filters"},
		RelatedTools:  []string{"create_view"},
		ChainPosition: ChainStarter,
	}

	// Query templates
	r.tools["query_templates"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryQuery, CategoryAIHelper},
		Keywords:      []string{"templates", "examples", "patterns", "best practice"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"Get query examples", "Learn patterns"},
		RelatedTools:  []string{"query_logs", "build_query"},
		ChainPosition: ChainStarter,
	}

	// Validation
	r.tools["validate_query"] = &ToolMetadata{
		Categories:    []ToolCategory{CategoryQuery, CategoryAIHelper},
		Keywords:      []string{"validate", "check", "query", "syntax", "verify"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"Check query syntax", "Validate before running"},
		RelatedTools:  []string{"query_logs", "explain_query"},
		ChainPosition: ChainMiddle,
	}
}

// registerToolChains registers common tool chains
func (r *ToolRegistry) registerToolChains() {
	r.chains = []ToolChain{
		// Investigation workflows
		{
			Name:        "error_investigation",
			Description: "Investigate error spikes and set up alerting",
			Trigger:     "query_logs",
			Condition:   "High error rate detected",
			Sequence:    []string{"query_logs", "investigate_incident", "suggest_alert", "create_alert"},
			UseCases:    []string{"Error investigation", "Incident response"},
		},
		{
			Name:        "latency_investigation",
			Description: "Investigate slow requests and performance issues",
			Trigger:     "query_logs",
			Condition:   "High latency detected",
			Sequence:    []string{"query_logs", "investigate_incident", "list_e2m", "create_dashboard"},
			UseCases:    []string{"Latency investigation", "Performance troubleshooting"},
		},
		{
			Name:        "security_investigation",
			Description: "Investigate security-related incidents and audit access",
			Trigger:     "query_logs",
			Condition:   "Security event detected",
			Sequence:    []string{"query_logs", "list_data_access_rules", "investigate_incident", "create_alert"},
			UseCases:    []string{"Security investigation", "Audit log analysis", "Access review"},
		},

		// Monitoring setup workflows
		{
			Name:        "monitoring_setup",
			Description: "Set up comprehensive monitoring for a service",
			Trigger:     "list_alerts",
			Condition:   "No alerts configured",
			Sequence:    []string{"query_logs", "create_outgoing_webhook", "suggest_alert", "create_alert"},
			UseCases:    []string{"New service monitoring", "Alerting setup"},
		},
		{
			Name:        "complete_observability",
			Description: "Set up full observability stack with alerts, dashboards, and metrics",
			Trigger:     "health_check",
			Condition:   "New service onboarding",
			Sequence:    []string{"health_check", "query_logs", "create_outgoing_webhook", "create_alert", "create_e2m", "create_dashboard"},
			UseCases:    []string{"Service onboarding", "Full observability setup", "Production readiness"},
		},
		{
			Name:        "slack_integration",
			Description: "Set up Slack notifications for alerts",
			Trigger:     "create_outgoing_webhook",
			Condition:   "Slack integration needed",
			Sequence:    []string{"list_outgoing_webhooks", "create_outgoing_webhook", "suggest_alert", "create_alert"},
			UseCases:    []string{"Slack integration", "Team notifications", "ChatOps"},
		},

		// Dashboard workflows
		{
			Name:        "dashboard_creation",
			Description: "Create a dashboard from query results",
			Trigger:     "query_logs",
			Condition:   "Large result set with patterns",
			Sequence:    []string{"query_logs", "build_query", "create_dashboard"},
			UseCases:    []string{"Visualization", "Reporting"},
		},
		{
			Name:        "error_dashboard",
			Description: "Create error monitoring dashboard with alerts",
			Trigger:     "list_dashboards",
			Condition:   "Error monitoring needed",
			Sequence:    []string{"query_logs", "validate_query", "create_dashboard", "create_alert"},
			UseCases:    []string{"Error dashboard", "Error monitoring visualization"},
		},

		// Cost and optimization workflows
		{
			Name:        "cost_optimization",
			Description: "Analyze and optimize log costs",
			Trigger:     "list_policies",
			Condition:   "Cost review needed",
			Sequence:    []string{"list_policies", "list_e2m", "export_data_usage", "create_policy"},
			UseCases:    []string{"Cost optimization", "Retention tuning"},
		},
		{
			Name:        "log_to_metrics",
			Description: "Convert high-volume logs to aggregated metrics",
			Trigger:     "list_e2m",
			Condition:   "High log volume",
			Sequence:    []string{"query_logs", "list_e2m", "create_e2m", "create_policy"},
			UseCases:    []string{"Log reduction", "Metrics generation", "Cost reduction"},
		},
		{
			Name:        "retention_optimization",
			Description: "Optimize retention policies for cost savings",
			Trigger:     "list_policies",
			Condition:   "Retention review needed",
			Sequence:    []string{"list_policies", "export_data_usage", "create_policy"},
			UseCases:    []string{"Retention tuning", "Storage optimization", "Compliance review"},
		},

		// Learning and query workflows
		{
			Name:        "query_learning",
			Description: "Learn and build queries step by step",
			Trigger:     "query_templates",
			Condition:   "New to DataPrime",
			Sequence:    []string{"query_templates", "build_query", "explain_query", "query_logs"},
			UseCases:    []string{"Learning DataPrime", "Query building"},
		},
		{
			Name:        "query_optimization",
			Description: "Analyze and optimize slow queries",
			Trigger:     "explain_query",
			Condition:   "Query performance issue",
			Sequence:    []string{"explain_query", "validate_query", "build_query", "query_logs"},
			UseCases:    []string{"Query optimization", "Performance tuning", "Query debugging"},
		},

		// Export and integration workflows
		{
			Name:        "log_export_setup",
			Description: "Set up log streaming to external systems",
			Trigger:     "list_streams",
			Condition:   "Log export needed",
			Sequence:    []string{"list_streams", "query_logs", "create_stream"},
			UseCases:    []string{"Log export", "SIEM integration", "Data lake export"},
		},
		{
			Name:        "kafka_streaming",
			Description: "Set up Kafka/Event Streams integration",
			Trigger:     "create_stream",
			Condition:   "Kafka integration needed",
			Sequence:    []string{"list_streams", "query_logs", "create_stream"},
			UseCases:    []string{"Kafka streaming", "Event Streams integration", "Real-time export"},
		},

		// Daily operations workflows
		{
			Name:        "morning_check",
			Description: "Daily morning health check and alert review",
			Trigger:     "health_check",
			Condition:   "Daily operations",
			Sequence:    []string{"health_check", "list_alerts", "query_logs", "list_dashboards"},
			UseCases:    []string{"Daily check", "Morning review", "Operations status"},
		},
		{
			Name:        "incident_postmortem",
			Description: "Analyze incident for postmortem documentation",
			Trigger:     "investigate_incident",
			Condition:   "Post-incident analysis",
			Sequence:    []string{"investigate_incident", "query_logs", "list_alerts", "create_view"},
			UseCases:    []string{"Postmortem", "Incident analysis", "RCA documentation"},
		},
	}
}

// buildIntentIndex builds an index from intents to tools
func (r *ToolRegistry) buildIntentIndex() {
	// Common intents mapped to tools - comprehensive natural language mappings
	intentMappings := map[string][]string{
		// ==================== Error Investigation Intents ====================
		"investigate errors": {"investigate_incident", "query_logs", "list_alerts"},
		"find errors":        {"query_logs", "investigate_incident"},
		"debug":              {"query_logs", "investigate_incident", "explain_query"},
		"debug production":   {"investigate_incident", "query_logs", "health_check"},
		"troubleshoot":       {"investigate_incident", "query_logs", "list_alerts"},
		"root cause":         {"investigate_incident", "query_logs"},
		"what went wrong":    {"investigate_incident", "query_logs", "list_alerts"},
		"why is it failing":  {"investigate_incident", "query_logs"},
		"error spike":        {"investigate_incident", "query_logs", "suggest_alert"},
		"error analysis":     {"investigate_incident", "query_logs"},
		"exception":          {"query_logs", "investigate_incident"},
		"stack trace":        {"query_logs", "investigate_incident"},
		"crash":              {"investigate_incident", "query_logs", "list_alerts"},
		"failure":            {"investigate_incident", "query_logs"},
		"broken":             {"investigate_incident", "query_logs"},
		"not working":        {"investigate_incident", "query_logs", "health_check"},
		"500 error":          {"query_logs", "investigate_incident"},
		"http error":         {"query_logs", "investigate_incident"},
		"service down":       {"health_check", "investigate_incident", "query_logs"},
		"outage":             {"investigate_incident", "health_check", "list_alerts"},
		"incident":           {"investigate_incident", "query_logs"},
		"postmortem":         {"investigate_incident", "query_logs", "list_alerts"},
		"rca":                {"investigate_incident", "query_logs"},
		"diagnose":           {"investigate_incident", "query_logs"},
		"analyze error":      {"investigate_incident", "query_logs"},
		"error pattern":      {"query_logs", "investigate_incident", "suggest_alert"},

		// ==================== Search and Query Intents ====================
		"search logs":    {"query_logs", "build_query"},
		"find logs":      {"query_logs", "build_query"},
		"look for":       {"query_logs", "build_query"},
		"show me":        {"query_logs", "list_dashboards"},
		"get logs":       {"query_logs"},
		"query":          {"query_logs", "build_query", "query_templates"},
		"filter logs":    {"query_logs", "build_query"},
		"search for":     {"query_logs", "build_query"},
		"find all":       {"query_logs"},
		"list logs":      {"query_logs"},
		"fetch logs":     {"query_logs"},
		"retrieve logs":  {"query_logs"},
		"grep":           {"query_logs", "build_query"},
		"search pattern": {"query_logs", "build_query"},
		"regex":          {"query_logs", "build_query"},
		"match":          {"query_logs", "build_query"},
		"contains":       {"query_logs", "build_query"},
		"logs from":      {"query_logs"},
		"logs between":   {"query_logs"},
		"last hour":      {"query_logs"},
		"last 24 hours":  {"query_logs"},
		"yesterday":      {"query_logs"},
		"today":          {"query_logs"},
		"recent logs":    {"query_logs"},
		"latest logs":    {"query_logs"},
		"tail logs":      {"query_logs"},
		"live logs":      {"query_logs"},
		"real time":      {"query_logs"},

		// ==================== Alerting Intents ====================
		"set up alerting":        {"suggest_alert", "create_alert", "create_outgoing_webhook"},
		"create alert":           {"create_alert", "suggest_alert"},
		"monitor errors":         {"suggest_alert", "create_alert"},
		"notify me":              {"create_alert", "create_outgoing_webhook"},
		"get notified":           {"create_alert", "create_outgoing_webhook"},
		"alert when":             {"create_alert", "suggest_alert"},
		"alert on":               {"create_alert", "suggest_alert"},
		"threshold alert":        {"create_alert", "suggest_alert"},
		"rate alert":             {"create_alert", "suggest_alert"},
		"volume alert":           {"create_alert", "suggest_alert"},
		"anomaly alert":          {"suggest_alert", "create_alert"},
		"new data alert":         {"create_alert"},
		"unique count alert":     {"create_alert", "suggest_alert"},
		"time relative alert":    {"create_alert", "suggest_alert"},
		"metric alert":           {"create_alert", "suggest_alert"},
		"edit alert":             {"get_alert", "update_alert"},
		"modify alert":           {"get_alert", "update_alert"},
		"update alert":           {"update_alert", "get_alert"},
		"delete alert":           {"delete_alert", "list_alerts"},
		"remove alert":           {"delete_alert", "list_alerts"},
		"disable alert":          {"update_alert", "get_alert"},
		"enable alert":           {"update_alert", "get_alert"},
		"mute alert":             {"update_alert"},
		"silence alert":          {"update_alert"},
		"triggered alerts":       {"list_alerts"},
		"firing alerts":          {"list_alerts"},
		"active alerts":          {"list_alerts"},
		"alert status":           {"list_alerts", "get_alert"},
		"my alerts":              {"list_alerts"},
		"all alerts":             {"list_alerts"},
		"list alerts":            {"list_alerts"},
		"alert recommendations":  {"suggest_alert"},
		"best practice alerts":   {"suggest_alert"},
		"what should i alert on": {"suggest_alert"},

		// ==================== Monitoring and Observability Intents ====================
		"create monitoring":  {"suggest_alert", "create_alert", "create_dashboard"},
		"visualize":          {"create_dashboard", "list_dashboards"},
		"create dashboard":   {"create_dashboard", "list_dashboards"},
		"check health":       {"health_check", "list_alerts"},
		"system status":      {"health_check", "query_logs"},
		"how is system":      {"health_check", "list_alerts"},
		"morning check":      {"health_check", "list_alerts"},
		"overview":           {"health_check", "list_dashboards"},
		"service health":     {"health_check", "query_logs"},
		"application health": {"health_check", "query_logs"},
		"is everything ok":   {"health_check", "list_alerts"},
		"status check":       {"health_check"},
		"system overview":    {"health_check", "list_dashboards"},
		"daily check":        {"health_check", "list_alerts"},
		"ops check":          {"health_check", "list_alerts"},
		"sre check":          {"health_check", "list_alerts", "list_dashboards"},
		"on call check":      {"health_check", "list_alerts"},
		"shift handoff":      {"health_check", "list_alerts", "query_logs"},

		// ==================== Dashboard Intents ====================
		"new dashboard":    {"create_dashboard"},
		"build dashboard":  {"create_dashboard"},
		"dashboard for":    {"create_dashboard", "list_dashboards"},
		"visualize logs":   {"create_dashboard", "query_logs"},
		"chart":            {"create_dashboard", "list_dashboards"},
		"graph":            {"create_dashboard", "list_dashboards"},
		"time series":      {"create_dashboard"},
		"pie chart":        {"create_dashboard"},
		"bar chart":        {"create_dashboard"},
		"line chart":       {"create_dashboard"},
		"table widget":     {"create_dashboard"},
		"gauge":            {"create_dashboard"},
		"heatmap":          {"create_dashboard"},
		"edit dashboard":   {"get_dashboard", "update_dashboard"},
		"modify dashboard": {"get_dashboard", "update_dashboard"},
		"delete dashboard": {"delete_dashboard", "list_dashboards"},
		"my dashboards":    {"list_dashboards"},
		"all dashboards":   {"list_dashboards"},
		"find dashboard":   {"list_dashboards"},
		"open dashboard":   {"get_dashboard", "list_dashboards"},

		// ==================== Ingestion Intents ====================
		"send logs":      {"ingest_logs"},
		"push logs":      {"ingest_logs"},
		"ingest":         {"ingest_logs"},
		"test ingestion": {"ingest_logs", "query_logs"},
		"import logs":    {"ingest_logs"},
		"upload logs":    {"ingest_logs"},
		"forward logs":   {"ingest_logs"},
		"log shipping":   {"ingest_logs"},
		"write logs":     {"ingest_logs"},

		// ==================== Policy and Retention Intents ====================
		"configure retention": {"list_policies", "create_policy"},
		"set retention":       {"create_policy", "list_policies"},
		"cost optimization":   {"list_policies", "list_e2m", "export_data_usage"},
		"reduce costs":        {"list_policies", "list_e2m"},
		"save money":          {"list_policies", "list_e2m", "export_data_usage"},
		"retention policy":    {"list_policies", "create_policy"},
		"data retention":      {"list_policies", "create_policy"},
		"archive policy":      {"list_policies", "create_policy"},
		"tcco":                {"list_policies", "list_e2m"},
		"priority":            {"list_policies", "create_policy"},
		"high priority":       {"create_policy", "list_policies"},
		"medium priority":     {"create_policy", "list_policies"},
		"low priority":        {"create_policy", "list_policies"},
		"block logs":          {"create_policy"},
		"drop logs":           {"create_policy"},
		"filter out":          {"create_policy"},
		"exclude logs":        {"create_policy"},
		"routing":             {"list_policies", "create_policy"},
		"log routing":         {"list_policies", "create_policy"},
		"storage tier":        {"list_policies", "create_policy"},
		"compliance":          {"list_policies", "list_data_access_rules"},

		// ==================== Learning Intents ====================
		"learn dataprime":          {"query_templates", "build_query", "explain_query"},
		"how to query":             {"query_templates", "build_query", "explain_query"},
		"help with query":          {"build_query", "explain_query", "query_templates"},
		"query examples":           {"query_templates"},
		"teach me":                 {"query_templates", "explain_query"},
		"dataprime syntax":         {"query_templates", "explain_query"},
		"dataprime tutorial":       {"query_templates", "explain_query"},
		"query syntax":             {"explain_query", "query_templates"},
		"how does this query work": {"explain_query"},
		"explain this query":       {"explain_query"},
		"what does this query do":  {"explain_query"},
		"query help":               {"build_query", "explain_query"},
		"build a query":            {"build_query"},
		"construct query":          {"build_query"},
		"write query":              {"build_query"},
		"generate query":           {"build_query"},
		"query for":                {"build_query", "query_logs"},
		"sample queries":           {"query_templates"},
		"common queries":           {"query_templates"},
		"useful queries":           {"query_templates"},
		"query best practices":     {"query_templates", "explain_query"},
		"optimize query":           {"explain_query", "validate_query"},
		"fix query":                {"explain_query", "validate_query"},
		"query error":              {"explain_query", "validate_query"},
		"validate":                 {"validate_query"},
		"check syntax":             {"validate_query"},

		// ==================== Integration Intents ====================
		"integrate slack":        {"create_outgoing_webhook"},
		"integrate pagerduty":    {"create_outgoing_webhook"},
		"connect to":             {"create_outgoing_webhook", "create_stream"},
		"webhook":                {"create_outgoing_webhook", "list_outgoing_webhooks"},
		"set up notifications":   {"create_outgoing_webhook", "create_alert"},
		"slack notification":     {"create_outgoing_webhook"},
		"pagerduty notification": {"create_outgoing_webhook"},
		"teams notification":     {"create_outgoing_webhook"},
		"email notification":     {"create_outgoing_webhook"},
		"opsgenie":               {"create_outgoing_webhook"},
		"victorops":              {"create_outgoing_webhook"},
		"custom webhook":         {"create_outgoing_webhook"},
		"generic webhook":        {"create_outgoing_webhook"},
		"notification channel":   {"create_outgoing_webhook", "list_outgoing_webhooks"},
		"alert destination":      {"create_outgoing_webhook", "list_outgoing_webhooks"},
		"list webhooks":          {"list_outgoing_webhooks"},
		"my webhooks":            {"list_outgoing_webhooks"},

		// ==================== Export and Streaming Intents ====================
		"export logs":    {"create_stream", "list_streams"},
		"stream logs":    {"create_stream", "list_streams"},
		"kafka":          {"create_stream", "list_streams"},
		"event streams":  {"create_stream", "list_streams"},
		"siem":           {"create_stream", "list_streams"},
		"splunk":         {"create_stream"},
		"datadog":        {"create_stream"},
		"elastic":        {"create_stream"},
		"s3 export":      {"create_stream"},
		"cos export":     {"create_stream"},
		"object storage": {"create_stream"},
		"data lake":      {"create_stream", "list_streams"},
		"archive logs":   {"create_stream", "list_policies"},
		"backup logs":    {"create_stream"},
		"replicate logs": {"create_stream"},

		// ==================== Performance Intents ====================
		"investigate latency":     {"query_logs", "investigate_incident"},
		"investigate performance": {"query_logs", "investigate_incident"},
		"slow requests":           {"query_logs", "investigate_incident"},
		"performance issues":      {"investigate_incident", "query_logs"},
		"high latency":            {"query_logs", "investigate_incident"},
		"p99":                     {"query_logs", "create_e2m"},
		"p95":                     {"query_logs", "create_e2m"},
		"percentile":              {"query_logs", "create_e2m"},
		"response time":           {"query_logs", "investigate_incident"},
		"timeout":                 {"query_logs", "investigate_incident"},
		"bottleneck":              {"investigate_incident", "query_logs"},
		"slow query":              {"query_logs", "investigate_incident"},
		"database slow":           {"query_logs", "investigate_incident"},
		"api latency":             {"query_logs", "investigate_incident"},
		"endpoint performance":    {"query_logs", "investigate_incident"},

		// ==================== Security Intents ====================
		"security audit":      {"list_data_access_rules", "query_logs"},
		"access control":      {"list_data_access_rules"},
		"permissions":         {"list_data_access_rules"},
		"who can access":      {"list_data_access_rules"},
		"rbac":                {"list_data_access_rules"},
		"data access":         {"list_data_access_rules"},
		"sensitive data":      {"list_data_access_rules", "create_policy"},
		"pii":                 {"list_data_access_rules", "create_policy"},
		"gdpr":                {"list_data_access_rules", "list_policies"},
		"audit logs":          {"query_logs", "list_data_access_rules"},
		"login attempts":      {"query_logs"},
		"authentication":      {"query_logs"},
		"authorization":       {"query_logs", "list_data_access_rules"},
		"suspicious activity": {"query_logs", "investigate_incident"},
		"security incident":   {"investigate_incident", "query_logs"},
		"breach":              {"investigate_incident", "query_logs", "list_data_access_rules"},
		"unauthorized":        {"query_logs", "investigate_incident"},
		"access denied":       {"query_logs"},
		"forbidden":           {"query_logs"},

		// ==================== Session and Discovery Intents ====================
		"what tools":      {"discover_tools"},
		"available tools": {"discover_tools"},
		"help me find":    {"discover_tools"},
		"session":         {"session_context"},
		"current context": {"session_context"},
		"my filters":      {"session_context"},
		"what can i do":   {"discover_tools"},
		"list tools":      {"discover_tools"},
		"tool help":       {"discover_tools"},
		"capabilities":    {"discover_tools"},
		"features":        {"discover_tools"},
		"getting started": {"discover_tools", "query_templates"},
		"help":            {"discover_tools"},
		"how to use":      {"discover_tools", "query_templates"},
		"my session":      {"session_context"},
		"session state":   {"session_context"},
		"recent queries":  {"session_context"},
		"query history":   {"session_context"},

		// ==================== Views Intents ====================
		"saved views":    {"list_views"},
		"my views":       {"list_views"},
		"create view":    {"create_view"},
		"save view":      {"create_view"},
		"bookmark":       {"create_view"},
		"favorite query": {"create_view"},
		"saved search":   {"list_views", "create_view"},
		"list views":     {"list_views"},
		"all views":      {"list_views"},

		// ==================== E2M (Events to Metrics) Intents ====================
		"events to metrics":  {"list_e2m", "create_e2m"},
		"convert to metrics": {"create_e2m"},
		"aggregate logs":     {"create_e2m", "query_logs"},
		"e2m":                {"list_e2m", "create_e2m"},
		"log to metric":      {"create_e2m"},
		"extract metric":     {"create_e2m"},
		"create metric":      {"create_e2m"},
		"custom metric":      {"create_e2m"},
		"count logs":         {"create_e2m", "query_logs"},
		"sum logs":           {"create_e2m", "query_logs"},
		"average":            {"create_e2m", "query_logs"},
		"histogram":          {"create_e2m"},
		"cardinality":        {"create_e2m", "query_logs"},
		"unique values":      {"query_logs", "create_e2m"},
		"list e2m":           {"list_e2m"},
		"my metrics":         {"list_e2m"},

		// ==================== Background Query Intents ====================
		"background query": {"submit_background_query"},
		"async query":      {"submit_background_query"},
		"long query":       {"submit_background_query"},
		"large query":      {"submit_background_query"},
		"query timeout":    {"submit_background_query"},
		"query status":     {"get_background_query_status"},
		"check query":      {"get_background_query_status"},
		"query results":    {"get_background_query_data"},
		"download results": {"get_background_query_data"},

		// ==================== Specific Service/Application Intents ====================
		"kubernetes":     {"query_logs", "build_query"},
		"k8s":            {"query_logs", "build_query"},
		"pod logs":       {"query_logs"},
		"container logs": {"query_logs"},
		"namespace":      {"query_logs", "build_query"},
		"deployment":     {"query_logs"},
		"service mesh":   {"query_logs"},
		"istio":          {"query_logs"},
		"nginx":          {"query_logs", "build_query"},
		"apache":         {"query_logs", "build_query"},
		"java":           {"query_logs", "build_query"},
		"python":         {"query_logs", "build_query"},
		"node":           {"query_logs", "build_query"},
		"golang":         {"query_logs", "build_query"},
		"database logs":  {"query_logs"},
		"mysql":          {"query_logs"},
		"postgres":       {"query_logs"},
		"mongodb":        {"query_logs"},
		"redis":          {"query_logs"},
		"aws":            {"query_logs"},
		"azure":          {"query_logs"},
		"gcp":            {"query_logs"},
		"ibm cloud":      {"query_logs", "health_check"},
		"cloud foundry":  {"query_logs"},
		"openshift":      {"query_logs"},

		// ==================== Team/Collaboration Intents ====================
		"share dashboard": {"get_dashboard", "list_dashboards"},
		"share view":      {"list_views"},
		"team":            {"list_dashboards", "list_views"},
		"collaborate":     {"list_dashboards", "list_views"},
	}

	for intent, tools := range intentMappings {
		r.intents[intent] = tools
	}
}

// DiscoverTools finds tools matching the given intent or criteria
func (r *ToolRegistry) DiscoverTools(intent string, category ToolCategory, complexity string) *DiscoveryResult {
	result := &DiscoveryResult{
		Intent:       intent,
		MatchedTools: []ToolMatch{},
	}

	intentLower := strings.ToLower(intent)

	// Check exact intent matches first
	if tools, ok := r.intents[intentLower]; ok {
		for _, toolName := range tools {
			if meta, exists := r.tools[toolName]; exists {
				result.MatchedTools = append(result.MatchedTools, ToolMatch{
					Name:       toolName,
					Relevance:  1.0,
					Reason:     "Direct intent match",
					Categories: categoryStrings(meta.Categories),
					Complexity: meta.Complexity,
					UseCases:   meta.UseCases,
				})
			}
		}
	}

	// Fuzzy intent matching - find partial matches in intent index
	if len(result.MatchedTools) == 0 {
		for registeredIntent, tools := range r.intents {
			similarity := fuzzyMatch(intentLower, registeredIntent)
			if similarity > 0.6 {
				for _, toolName := range tools {
					if meta, exists := r.tools[toolName]; exists {
						// Check if already added
						alreadyAdded := false
						for _, m := range result.MatchedTools {
							if m.Name == toolName {
								alreadyAdded = true
								break
							}
						}
						if !alreadyAdded {
							result.MatchedTools = append(result.MatchedTools, ToolMatch{
								Name:       toolName,
								Relevance:  similarity,
								Reason:     "Fuzzy match: " + registeredIntent,
								Categories: categoryStrings(meta.Categories),
								Complexity: meta.Complexity,
								UseCases:   meta.UseCases,
							})
						}
					}
				}
			}
		}
	}

	// Keyword matching
	intentWords := strings.Fields(intentLower)
	for toolName, meta := range r.tools {
		// Skip if already added from intent match
		alreadyAdded := false
		for _, m := range result.MatchedTools {
			if m.Name == toolName {
				alreadyAdded = true
				break
			}
		}
		if alreadyAdded {
			continue
		}

		// Filter by category if specified
		if category != "" && !containsCategory(meta.Categories, category) {
			continue
		}

		// Filter by complexity if specified
		if complexity != "" && meta.Complexity != complexity {
			continue
		}

		// Calculate relevance from keyword matching
		relevance, reason := calculateRelevance(intentWords, meta)
		if relevance > 0.3 {
			result.MatchedTools = append(result.MatchedTools, ToolMatch{
				Name:       toolName,
				Relevance:  relevance,
				Reason:     reason,
				Categories: categoryStrings(meta.Categories),
				Complexity: meta.Complexity,
				UseCases:   meta.UseCases,
			})
		}
	}

	// Sort by relevance
	sort.Slice(result.MatchedTools, func(i, j int) bool {
		return result.MatchedTools[i].Relevance > result.MatchedTools[j].Relevance
	})

	// Calculate confidence and apply confidence-based filtering
	result.Confidence = r.calculateConfidence(intentLower, result.MatchedTools)
	result.MatchedTools = r.applyConfidenceFiltering(result.MatchedTools, result.Confidence)

	// Find matching tool chain
	result.SuggestedChain = r.findMatchingChain(intentLower)

	// Add session context
	session := GetSession()
	result.SessionContext = session.GetSessionSummary()

	// Generate adaptive chains based on learned patterns
	result.AdaptiveChains = r.generateAdaptiveChains(result.MatchedTools, session)

	// Add recommendations
	result.Recommendations = r.generateRecommendations(result.MatchedTools, session)

	// Add confidence-based recommendations
	result.Recommendations = append(result.Recommendations, r.generateConfidenceRecommendations(result.Confidence)...)

	// Add adaptive chain recommendations
	result.Recommendations = append(result.Recommendations, r.generateAdaptiveChainRecommendations(result.AdaptiveChains)...)

	return result
}

// calculateConfidence calculates the confidence level for the discovery results
func (r *ToolRegistry) calculateConfidence(intent string, matches []ToolMatch) *ConfidenceResult {
	confidence := &ConfidenceResult{
		Level:       ConfidenceLow,
		Score:       0.0,
		Explanation: "No matching tools found",
	}

	if len(matches) == 0 {
		confidence.Alternatives = r.findAlternativeIntents(intent)
		confidence.Clarifications = []string{
			"Try using more specific keywords",
			"Describe what you want to accomplish",
			"Use category filter (e.g., 'alert', 'query', 'dashboard')",
		}
		return confidence
	}

	// Calculate aggregate confidence score based on top matches
	topScore := matches[0].Relevance
	avgScore := 0.0
	scoreCount := min(3, len(matches))
	for i := 0; i < scoreCount; i++ {
		avgScore += matches[i].Relevance
	}
	avgScore /= float64(scoreCount)

	// Weight: 60% top score, 40% average of top 3
	confidence.Score = topScore*0.6 + avgScore*0.4

	// Determine confidence level
	switch {
	case confidence.Score >= HighConfidenceThreshold:
		confidence.Level = ConfidenceHigh
		confidence.Explanation = "Strong match found for your intent"
		if topScore == 1.0 {
			confidence.Explanation = "Exact intent match found"
		}
	case confidence.Score >= MediumConfidenceThreshold:
		confidence.Level = ConfidenceMedium
		confidence.Explanation = "Moderate match - results may need refinement"
		confidence.Clarifications = r.generateClarifications(intent, matches)
	default:
		confidence.Level = ConfidenceLow
		confidence.Explanation = "Weak match - consider rephrasing your intent"
		confidence.Alternatives = r.findAlternativeIntents(intent)
		confidence.Clarifications = r.generateClarifications(intent, matches)
	}

	return confidence
}

// applyConfidenceFiltering limits results based on confidence level
func (r *ToolRegistry) applyConfidenceFiltering(matches []ToolMatch, confidence *ConfidenceResult) []ToolMatch {
	if len(matches) == 0 {
		return matches
	}

	var maxTools int
	switch confidence.Level {
	case ConfidenceHigh:
		maxTools = HighConfidenceMaxTools
	case ConfidenceMedium:
		maxTools = MediumConfidenceMaxTools
	default:
		maxTools = LowConfidenceMaxTools
	}

	if len(matches) > maxTools {
		return matches[:maxTools]
	}
	return matches
}

// findAlternativeIntents suggests alternative intents the user might have meant
func (r *ToolRegistry) findAlternativeIntents(intent string) []string {
	alternatives := []string{}
	intentLower := strings.ToLower(intent)
	scores := make(map[string]float64)

	// Find partial matches in registered intents
	for registeredIntent := range r.intents {
		similarity := fuzzyMatch(intentLower, registeredIntent)
		if similarity > 0.3 && similarity < 0.6 {
			scores[registeredIntent] = similarity
		}
	}

	// Sort by score and take top 5
	type scoredIntent struct {
		intent string
		score  float64
	}
	var scored []scoredIntent
	for intent, score := range scores {
		scored = append(scored, scoredIntent{intent, score})
	}
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	for i := 0; i < len(scored) && i < 5; i++ {
		alternatives = append(alternatives, scored[i].intent)
	}

	// If no partial matches, suggest common intents based on keywords
	if len(alternatives) == 0 {
		words := strings.Fields(intentLower)
		for _, word := range words {
			for registeredIntent := range r.intents {
				if strings.Contains(registeredIntent, word) && len(alternatives) < 5 {
					// Avoid duplicates
					found := false
					for _, alt := range alternatives {
						if alt == registeredIntent {
							found = true
							break
						}
					}
					if !found {
						alternatives = append(alternatives, registeredIntent)
					}
				}
			}
		}
	}

	return alternatives
}

// generateClarifications generates clarifying questions for medium/low confidence
func (r *ToolRegistry) generateClarifications(intent string, matches []ToolMatch) []string {
	clarifications := []string{}
	intentLower := strings.ToLower(intent)

	// Analyze the types of matches to suggest clarifications
	hasQueryTools := false
	hasAlertTools := false
	hasDashboardTools := false
	hasWorkflowTools := false

	for _, match := range matches {
		for _, cat := range match.Categories {
			switch cat {
			case string(CategoryQuery):
				hasQueryTools = true
			case string(CategoryAlert):
				hasAlertTools = true
			case string(CategoryDashboard):
				hasDashboardTools = true
			case string(CategoryWorkflow):
				hasWorkflowTools = true
			}
		}
	}

	// Generate category-based clarifications
	if hasQueryTools && hasAlertTools {
		clarifications = append(clarifications, "Do you want to search logs or set up alerting?")
	}
	if hasQueryTools && hasDashboardTools {
		clarifications = append(clarifications, "Do you want to query data or visualize it?")
	}
	if hasWorkflowTools {
		clarifications = append(clarifications, "Do you need a guided workflow for this task?")
	}

	// Intent-specific clarifications
	if strings.Contains(intentLower, "error") || strings.Contains(intentLower, "issue") {
		clarifications = append(clarifications, "Are you investigating an active incident or setting up monitoring?")
	}
	if strings.Contains(intentLower, "log") {
		clarifications = append(clarifications, "Do you want to search existing logs or send new logs?")
	}
	if strings.Contains(intentLower, "alert") {
		clarifications = append(clarifications, "Do you want to create a new alert or view existing alerts?")
	}

	// Limit clarifications
	if len(clarifications) > 3 {
		clarifications = clarifications[:3]
	}

	return clarifications
}

// generateConfidenceRecommendations adds recommendations based on confidence level
func (r *ToolRegistry) generateConfidenceRecommendations(confidence *ConfidenceResult) []string {
	recs := []string{}

	switch confidence.Level {
	case ConfidenceHigh:
		recs = append(recs, "High confidence match - proceed with the top tool")
	case ConfidenceMedium:
		recs = append(recs, "Review the matched tools to find the best fit")
		if len(confidence.Clarifications) > 0 {
			recs = append(recs, "Consider the clarifying questions to refine your search")
		}
	case ConfidenceLow:
		recs = append(recs, "Consider rephrasing your intent for better matches")
		if len(confidence.Alternatives) > 0 {
			recs = append(recs, "Did you mean one of the suggested alternatives?")
		}
	}

	return recs
}

// generateAdaptiveChains creates adaptive tool chain suggestions based on learned patterns
func (r *ToolRegistry) generateAdaptiveChains(matches []ToolMatch, session *SessionContext) []*AdaptiveChain {
	chains := []*AdaptiveChain{}

	if session == nil {
		return chains
	}

	// Get learned patterns from session
	patterns := session.LearnedPatterns
	if patterns == nil || len(patterns.FrequentSequences) == 0 {
		// No learned patterns yet - suggest chains based on matched tools
		return r.suggestChainsFromMatches(matches)
	}

	// Build adaptive chains from learned sequences
	matchedToolNames := make(map[string]bool)
	for _, m := range matches {
		matchedToolNames[m.Name] = true
	}

	for _, seq := range patterns.FrequentSequences {
		// Only include sequences that start with a matched tool or are highly successful
		includeChain := false
		if len(seq.Tools) > 0 && matchedToolNames[seq.Tools[0]] {
			includeChain = true
		}
		if seq.SuccessRate >= 80.0 && seq.Count >= 3 {
			includeChain = true
		}

		if !includeChain {
			continue
		}

		// Determine confidence based on usage and success rate
		confidence := "low"
		if seq.Count >= 5 && seq.SuccessRate >= 80.0 {
			confidence = "high"
		} else if seq.Count >= 3 && seq.SuccessRate >= 60.0 {
			confidence = "medium"
		}

		chain := &AdaptiveChain{
			Name:        generateChainName(seq.Tools),
			Description: generateChainDescription(seq.Tools),
			Tools:       seq.Tools,
			SuccessRate: seq.SuccessRate,
			UseCount:    seq.Count,
			Confidence:  confidence,
			Source:      "learned",
		}
		chains = append(chains, chain)
	}

	// Sort by confidence and success rate
	sortAdaptiveChains(chains)

	// Limit to top 5 chains
	if len(chains) > 5 {
		chains = chains[:5]
	}

	// If no learned chains, suggest based on matches
	if len(chains) == 0 {
		chains = r.suggestChainsFromMatches(matches)
	}

	return chains
}

// suggestChainsFromMatches creates suggested chains based on matched tools
func (r *ToolRegistry) suggestChainsFromMatches(matches []ToolMatch) []*AdaptiveChain {
	chains := []*AdaptiveChain{}

	if len(matches) < 2 {
		return chains
	}

	// Common workflow patterns based on tool categories
	workflowPatterns := map[string][]string{
		"investigation": {"query_logs", "investigate_incident", "suggest_alert"},
		"monitoring":    {"list_alerts", "create_alert", "create_outgoing_webhook"},
		"visualization": {"query_logs", "create_dashboard"},
		"optimization":  {"list_policies", "list_e2m", "export_data_usage"},
	}

	// Check which patterns match our results
	matchedTools := make(map[string]bool)
	for _, m := range matches {
		matchedTools[m.Name] = true
	}

	for patternName, tools := range workflowPatterns {
		matchCount := 0
		for _, tool := range tools {
			if matchedTools[tool] {
				matchCount++
			}
		}

		// If at least 2 tools from the pattern match, suggest the chain
		if matchCount >= 2 {
			chains = append(chains, &AdaptiveChain{
				Name:        patternName + " workflow",
				Description: "Suggested workflow: " + strings.Join(tools, " → "),
				Tools:       tools,
				SuccessRate: 0, // No history yet
				UseCount:    0,
				Confidence:  "suggested",
				Source:      "suggested",
			})
		}
	}

	return chains
}

// generateChainName creates a human-readable name for a tool chain
func generateChainName(tools []string) string {
	if len(tools) == 0 {
		return "Empty chain"
	}
	if len(tools) == 1 {
		return tools[0]
	}

	// Extract the action from tool names (remove common prefixes)
	actions := []string{}
	for _, tool := range tools {
		action := tool
		for _, prefix := range []string{"list_", "get_", "create_", "update_", "delete_"} {
			if strings.HasPrefix(tool, prefix) {
				action = strings.TrimPrefix(tool, prefix)
				break
			}
		}
		actions = append(actions, action)
	}

	return strings.Join(actions[:min(3, len(actions))], " → ")
}

// generateChainDescription creates a description for a tool chain
func generateChainDescription(tools []string) string {
	if len(tools) == 0 {
		return ""
	}

	// Generate a natural language description
	switch {
	case containsAll(tools, "query_logs", "investigate_incident"):
		return "Search logs and investigate issues"
	case containsAll(tools, "list_alerts", "create_alert"):
		return "Review and create alerts"
	case containsAll(tools, "query_logs", "create_dashboard"):
		return "Query data and visualize results"
	case containsAll(tools, "suggest_alert", "create_alert"):
		return "Get alert suggestions and create them"
	default:
		return "Sequence: " + strings.Join(tools, " → ")
	}
}

// containsAll checks if tools slice contains all specified items
func containsAll(tools []string, items ...string) bool {
	toolSet := make(map[string]bool)
	for _, t := range tools {
		toolSet[t] = true
	}
	for _, item := range items {
		if !toolSet[item] {
			return false
		}
	}
	return true
}

// sortAdaptiveChains sorts chains by confidence and success rate
func sortAdaptiveChains(chains []*AdaptiveChain) {
	// Simple bubble sort for small lists
	for i := 0; i < len(chains); i++ {
		for j := i + 1; j < len(chains); j++ {
			// Compare by confidence first, then success rate
			iScore := chainScore(chains[i])
			jScore := chainScore(chains[j])
			if jScore > iScore {
				chains[i], chains[j] = chains[j], chains[i]
			}
		}
	}
}

// chainScore calculates a score for sorting chains
func chainScore(chain *AdaptiveChain) float64 {
	confidenceScore := 0.0
	switch chain.Confidence {
	case "high":
		confidenceScore = 3.0
	case "medium":
		confidenceScore = 2.0
	case "low", "suggested":
		confidenceScore = 1.0
	}

	// Combine confidence with success rate and usage
	return confidenceScore*100 + chain.SuccessRate + float64(chain.UseCount)*0.1
}

// generateAdaptiveChainRecommendations creates recommendations based on adaptive chains
func (r *ToolRegistry) generateAdaptiveChainRecommendations(chains []*AdaptiveChain) []string {
	recs := []string{}

	if len(chains) == 0 {
		return recs
	}

	// Find the best learned chain
	var bestLearned *AdaptiveChain
	for _, chain := range chains {
		if chain.Source == "learned" && chain.Confidence == "high" {
			bestLearned = chain
			break
		}
	}

	if bestLearned != nil {
		recs = append(recs, fmt.Sprintf(
			"Based on your history, try: %s (%.0f%% success rate, used %d times)",
			strings.Join(bestLearned.Tools, " → "),
			bestLearned.SuccessRate,
			bestLearned.UseCount,
		))
	}

	// Count learned vs suggested
	learnedCount := 0
	for _, chain := range chains {
		if chain.Source == "learned" {
			learnedCount++
		}
	}

	if learnedCount >= 3 {
		recs = append(recs, "Multiple learned patterns available - check adaptive_chains for personalized workflows")
	}

	return recs
}

// fuzzyMatch calculates similarity between two strings using word overlap and substring matching
func fuzzyMatch(query, target string) float64 {
	if query == target {
		return 1.0
	}

	// Check if query is contained in target or vice versa
	if strings.Contains(target, query) {
		return 0.9
	}
	if strings.Contains(query, target) {
		return 0.85
	}

	// Word-based matching
	queryWords := strings.Fields(query)
	targetWords := strings.Fields(target)

	if len(queryWords) == 0 || len(targetWords) == 0 {
		return 0
	}

	matchCount := 0
	for _, qw := range queryWords {
		for _, tw := range targetWords {
			// Exact word match
			if qw == tw {
				matchCount += 2
				continue
			}
			// Partial word match (prefix/suffix)
			if len(qw) >= 3 && len(tw) >= 3 {
				if strings.HasPrefix(qw, tw[:3]) || strings.HasPrefix(tw, qw[:3]) {
					matchCount++
				}
			}
		}
	}

	// Calculate similarity based on word matches
	maxWords := len(queryWords)
	if len(targetWords) > maxWords {
		maxWords = len(targetWords)
	}

	similarity := float64(matchCount) / float64(maxWords*2)
	if similarity > 1.0 {
		similarity = 1.0
	}

	return similarity
}

// calculateRelevance calculates how well a tool matches the intent
func calculateRelevance(intentWords []string, meta *ToolMetadata) (float64, string) {
	matchedKeywords := []string{}
	matchedUseCases := []string{}

	for _, word := range intentWords {
		for _, keyword := range meta.Keywords {
			if strings.Contains(keyword, word) || strings.Contains(word, keyword) {
				matchedKeywords = append(matchedKeywords, keyword)
			}
		}
		for _, useCase := range meta.UseCases {
			if strings.Contains(strings.ToLower(useCase), word) {
				matchedUseCases = append(matchedUseCases, useCase)
			}
		}
	}

	if len(matchedKeywords) == 0 && len(matchedUseCases) == 0 {
		return 0, ""
	}

	relevance := float64(len(matchedKeywords))*0.3 + float64(len(matchedUseCases))*0.2
	if relevance > 1.0 {
		relevance = 1.0
	}

	reason := ""
	if len(matchedKeywords) > 0 {
		reason = "Matched keywords: " + strings.Join(unique(matchedKeywords), ", ")
	}
	if len(matchedUseCases) > 0 {
		if reason != "" {
			reason += "; "
		}
		reason += "Relevant for: " + strings.Join(unique(matchedUseCases), ", ")
	}

	return relevance, reason
}

// findMatchingChain finds a tool chain that matches the intent
func (r *ToolRegistry) findMatchingChain(intent string) *ToolChain {
	for _, chain := range r.chains {
		for _, useCase := range chain.UseCases {
			if strings.Contains(strings.ToLower(useCase), intent) ||
				strings.Contains(intent, strings.ToLower(useCase)) {
				return &chain
			}
		}
		if strings.Contains(strings.ToLower(chain.Description), intent) {
			return &chain
		}
	}
	return nil
}

// generateRecommendations creates recommendations based on matches and session
func (r *ToolRegistry) generateRecommendations(matches []ToolMatch, session *SessionContext) []string {
	recs := []string{}

	if len(matches) == 0 {
		recs = append(recs, "Try broader search terms or specify a category")
		return recs
	}

	// Recommend based on complexity
	hasSimple := false
	hasAdvanced := false
	for _, m := range matches {
		if m.Complexity == ComplexitySimple {
			hasSimple = true
		}
		if m.Complexity == ComplexityAdvanced {
			hasAdvanced = true
		}
	}

	if hasAdvanced && !hasSimple {
		recs = append(recs, "These are advanced tools - consider starting with list_* tools first")
	}

	// Recommend based on session
	if session.GetLastQuery() != "" {
		recs = append(recs, "Previous query context available - tools can reference it")
	}

	if inv := session.GetInvestigation(); inv != nil {
		recs = append(recs, "Active investigation in progress - findings will be tracked")
	}

	return recs
}

// Helper functions
func containsCategory(cats []ToolCategory, cat ToolCategory) bool {
	for _, c := range cats {
		if c == cat {
			return true
		}
	}
	return false
}

func categoryStrings(cats []ToolCategory) []string {
	result := make([]string, len(cats))
	for i, c := range cats {
		result[i] = string(c)
	}
	return result
}

func unique(items []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// DiscoverToolsTool implements the discover_tools meta-tool
type DiscoverToolsTool struct {
	*BaseTool
}

// NewDiscoverToolsTool creates a new DiscoverToolsTool
func NewDiscoverToolsTool(c *client.Client, l *zap.Logger) *DiscoverToolsTool {
	return &DiscoverToolsTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *DiscoverToolsTool) Name() string { return "discover_tools" }

// Description returns the tool description
func (t *DiscoverToolsTool) Description() string {
	return `Find the most relevant tools for your task using semantic search.

**Use this tool when you're unsure which tool to use, or want to explore available capabilities.**

**Examples:**
- discover_tools(intent: "investigate high error rates")
- discover_tools(intent: "set up monitoring for my service")
- discover_tools(category: "alert")
- discover_tools(intent: "learn dataprime query syntax")

**Returns:**
- Ranked list of matching tools with relevance scores
- Suggested tool chain for multi-step workflows
- Current session context (active filters, recent tools)
- Recommendations based on your usage patterns`
}

// InputSchema returns the input schema
func (t *DiscoverToolsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"intent": map[string]interface{}{
				"type":        "string",
				"description": "Natural language description of what you want to do",
				"examples": []string{
					"investigate errors in production",
					"set up alerting for my service",
					"create a dashboard for monitoring",
					"learn how to write queries",
				},
			},
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Filter by tool category",
				"enum": []string{
					"query", "alert", "dashboard", "policy", "webhook",
					"e2m", "stream", "workflow", "ai_helper", "observability",
				},
			},
			"complexity": map[string]interface{}{
				"type":        "string",
				"description": "Filter by complexity level",
				"enum":        []string{"simple", "intermediate", "advanced"},
			},
		},
	}
}

// Annotations returns tool annotations
func (t *DiscoverToolsTool) Annotations() *mcp.ToolAnnotations {
	return ReadOnlyAnnotations("Discover Tools")
}

// Execute executes the tool
func (t *DiscoverToolsTool) Execute(_ context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	intent, _ := GetStringParam(args, "intent", false)
	categoryStr, _ := GetStringParam(args, "category", false)
	complexity, _ := GetStringParam(args, "complexity", false)

	if intent == "" && categoryStr == "" {
		return NewToolResultError("Please provide either 'intent' (what you want to do) or 'category' (tool category to browse)"), nil
	}

	var category ToolCategory
	if categoryStr != "" {
		category = ToolCategory(categoryStr)
	}

	registry := GetToolRegistry()
	result := registry.DiscoverTools(intent, category, complexity)

	// Format output
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return NewToolResultError("Failed to format results: " + err.Error()), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(output),
			},
		},
	}, nil
}

// Metadata returns tool metadata for discovery
func (t *DiscoverToolsTool) Metadata() *ToolMetadata {
	return &ToolMetadata{
		Categories:    []ToolCategory{CategoryMeta, CategoryAIHelper},
		Keywords:      []string{"discover", "find", "search", "help", "tools", "capabilities"},
		Complexity:    ComplexitySimple,
		UseCases:      []string{"Find relevant tools", "Explore capabilities", "Get help"},
		RelatedTools:  []string{},
		ChainPosition: ChainStarter,
	}
}
