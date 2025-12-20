// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements elicitation support for interactive user input.
package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RequireConfirmation checks if the confirm parameter is set and returns
// a confirmation prompt if not. This provides a consistent UX for destructive operations.
// Returns (shouldContinue, result) - if shouldContinue is false, return the result immediately.
func RequireConfirmation(arguments map[string]interface{}, resourceType, resourceID string) (bool, *mcp.CallToolResult) {
	confirmed, _ := GetBoolParam(arguments, "confirm", false)
	if confirmed {
		return true, nil
	}

	return false, &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("⚠️ **Confirmation Required**\n\n"+
					"You are about to delete %s `%s`. This action cannot be undone.\n\n"+
					"To proceed, call this tool again with `confirm: true`:\n"+
					"```json\n{\"id\": \"%s\", \"confirm\": true}\n```",
					resourceType, resourceID, resourceID),
			},
		},
	}
}

// ConfirmationInputSchema returns the standard confirm property schema for delete tools
func ConfirmationInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "boolean",
		"description": "Set to true to confirm deletion. Required to prevent accidental deletions.",
		"default":     false,
	}
}

// ElicitationHelper provides utilities for eliciting user input during tool execution.
type ElicitationHelper struct {
	session *mcp.ServerSession
}

// NewElicitationHelper creates a new ElicitationHelper.
// The session is used to send elicitation requests to the client.
func NewElicitationHelper(session *mcp.ServerSession) *ElicitationHelper {
	return &ElicitationHelper{session: session}
}

// ElicitationSchema defines the schema for user input
type ElicitationSchema struct {
	Type       string                         `json:"type"`
	Properties map[string]ElicitationProperty `json:"properties"`
	Required   []string                       `json:"required,omitempty"`
}

// ElicitationProperty defines a single property in the schema
type ElicitationProperty struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// CommonElicitationSchemas provides pre-built schemas for common scenarios
var CommonElicitationSchemas = map[string]*ElicitationSchema{
	// Confirmation for destructive actions
	"confirm_delete": {
		Type: "object",
		Properties: map[string]ElicitationProperty{
			"confirm": {
				Type:        "boolean",
				Description: "Confirm you want to delete this resource",
				Default:     false,
			},
			"reason": {
				Type:        "string",
				Description: "Optional reason for deletion",
			},
		},
		Required: []string{"confirm"},
	},

	// Webhook configuration
	"webhook_config": {
		Type: "object",
		Properties: map[string]ElicitationProperty{
			"webhook_id": {
				Type:        "string",
				Description: "ID of the webhook to use for notifications",
			},
			"notify_on": {
				Type:        "string",
				Description: "When to send notifications",
				Enum:        []string{"triggered_only", "triggered_and_resolved"},
				Default:     "triggered_only",
			},
		},
		Required: []string{"webhook_id"},
	},

	// Time range selection
	"time_range": {
		Type: "object",
		Properties: map[string]ElicitationProperty{
			"preset": {
				Type:        "string",
				Description: "Preset time range",
				Enum:        []string{"15m", "1h", "6h", "24h", "7d", "30d", "custom"},
				Default:     "1h",
			},
			"start_time": {
				Type:        "string",
				Description: "Start time (ISO8601) for custom range",
			},
			"end_time": {
				Type:        "string",
				Description: "End time (ISO8601) for custom range",
			},
		},
		Required: []string{"preset"},
	},

	// Alert threshold configuration
	"alert_threshold": {
		Type: "object",
		Properties: map[string]ElicitationProperty{
			"threshold": {
				Type:        "number",
				Description: "Threshold value to trigger alert",
			},
			"condition": {
				Type:        "string",
				Description: "Condition for threshold comparison",
				Enum:        []string{"more_than", "less_than", "equals"},
				Default:     "more_than",
			},
			"time_window": {
				Type:        "string",
				Description: "Time window for threshold evaluation",
				Enum:        []string{"5m", "15m", "30m", "1h", "6h", "24h"},
				Default:     "5m",
			},
		},
		Required: []string{"threshold"},
	},

	// Severity selection
	"severity": {
		Type: "object",
		Properties: map[string]ElicitationProperty{
			"severity": {
				Type:        "string",
				Description: "Alert severity level",
				Enum:        []string{"info", "warning", "error", "critical"},
				Default:     "warning",
			},
		},
		Required: []string{"severity"},
	},
}

// ElicitResult represents the result of an elicitation request
type ElicitResult struct {
	Action  string                 // "accept", "decline", or "cancel"
	Content map[string]interface{} // User-provided values (when action is "accept")
}

// Elicit sends an elicitation request to the client and waits for a response.
// Returns the user's response or an error if elicitation is not supported or fails.
func (h *ElicitationHelper) Elicit(ctx context.Context, message string, schema *ElicitationSchema) (*ElicitResult, error) {
	if h.session == nil {
		return nil, fmt.Errorf("no server session available for elicitation")
	}

	params := &mcp.ElicitParams{
		Message:         message,
		RequestedSchema: schema,
	}

	result, err := h.session.Elicit(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("elicitation failed: %w", err)
	}

	return &ElicitResult{
		Action:  result.Action,
		Content: result.Content,
	}, nil
}

// ConfirmDelete asks the user to confirm a delete operation.
// Returns true if the user confirmed, false otherwise.
func (h *ElicitationHelper) ConfirmDelete(ctx context.Context, resourceType, resourceID string) (bool, string, error) {
	message := fmt.Sprintf("Are you sure you want to delete %s '%s'? This action cannot be undone.", resourceType, resourceID)

	result, err := h.Elicit(ctx, message, CommonElicitationSchemas["confirm_delete"])
	if err != nil {
		return false, "", err
	}

	if result.Action != "accept" {
		return false, "", nil
	}

	confirmed, _ := result.Content["confirm"].(bool)
	reason, _ := result.Content["reason"].(string)

	return confirmed, reason, nil
}

// RequestWebhookConfig asks the user to provide webhook configuration.
func (h *ElicitationHelper) RequestWebhookConfig(ctx context.Context, purpose string) (string, string, error) {
	message := fmt.Sprintf("Please provide webhook configuration for %s.", purpose)

	result, err := h.Elicit(ctx, message, CommonElicitationSchemas["webhook_config"])
	if err != nil {
		return "", "", err
	}

	if result.Action != "accept" {
		return "", "", fmt.Errorf("webhook configuration cancelled by user")
	}

	webhookID, _ := result.Content["webhook_id"].(string)
	notifyOn, _ := result.Content["notify_on"].(string)
	if notifyOn == "" {
		notifyOn = "triggered_only"
	}

	return webhookID, notifyOn, nil
}

// RequestTimeRange asks the user to select a time range.
func (h *ElicitationHelper) RequestTimeRange(ctx context.Context) (string, string, string, error) {
	message := "Please select a time range for the query."

	result, err := h.Elicit(ctx, message, CommonElicitationSchemas["time_range"])
	if err != nil {
		return "", "", "", err
	}

	if result.Action != "accept" {
		return "", "", "", fmt.Errorf("time range selection cancelled by user")
	}

	preset, _ := result.Content["preset"].(string)
	startTime, _ := result.Content["start_time"].(string)
	endTime, _ := result.Content["end_time"].(string)

	return preset, startTime, endTime, nil
}

// RequestAlertThreshold asks the user to configure alert threshold.
func (h *ElicitationHelper) RequestAlertThreshold(ctx context.Context, metricName string) (float64, string, string, error) {
	message := fmt.Sprintf("Please configure the alert threshold for %s.", metricName)

	result, err := h.Elicit(ctx, message, CommonElicitationSchemas["alert_threshold"])
	if err != nil {
		return 0, "", "", err
	}

	if result.Action != "accept" {
		return 0, "", "", fmt.Errorf("alert threshold configuration cancelled by user")
	}

	threshold, _ := result.Content["threshold"].(float64)
	condition, _ := result.Content["condition"].(string)
	if condition == "" {
		condition = "more_than"
	}
	timeWindow, _ := result.Content["time_window"].(string)
	if timeWindow == "" {
		timeWindow = "5m"
	}

	return threshold, condition, timeWindow, nil
}

// RequestSeverity asks the user to select a severity level.
func (h *ElicitationHelper) RequestSeverity(ctx context.Context, context string) (string, error) {
	message := fmt.Sprintf("Please select the severity level for %s.", context)

	result, err := h.Elicit(ctx, message, CommonElicitationSchemas["severity"])
	if err != nil {
		return "", err
	}

	if result.Action != "accept" {
		return "", fmt.Errorf("severity selection cancelled by user")
	}

	severity, _ := result.Content["severity"].(string)
	if severity == "" {
		severity = "warning"
	}

	return severity, nil
}

// CustomElicit sends a custom elicitation request with a user-defined schema.
func (h *ElicitationHelper) CustomElicit(ctx context.Context, message string, properties map[string]ElicitationProperty, required []string) (*ElicitResult, error) {
	schema := &ElicitationSchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}

	return h.Elicit(ctx, message, schema)
}

// URLElicitationSchema defines the schema for URL mode elicitation (MCP 2025-11-25)
// This allows opening URLs in the client's browser/application
type URLElicitationSchema struct {
	Type       string `json:"type"`
	Format     string `json:"format"` // "uri"
	Properties map[string]ElicitationProperty
}

// OpenURL requests the client to open a URL (MCP 2025-11-25 URL mode elicitation)
// Returns true if the user agreed to open the URL, false otherwise.
// This is useful for directing users to documentation, dashboards, or external resources.
func (h *ElicitationHelper) OpenURL(ctx context.Context, message string, url string) (bool, error) {
	if h.session == nil {
		return false, fmt.Errorf("no server session available for elicitation")
	}

	// URL mode elicitation uses a special schema format
	schema := &ElicitationSchema{
		Type: "object",
		Properties: map[string]ElicitationProperty{
			"url": {
				Type:        "string",
				Description: "URL to open",
				Default:     url,
			},
		},
	}

	// Set meta to indicate URL mode
	params := &mcp.ElicitParams{
		Message:         message,
		RequestedSchema: schema,
	}
	params.Meta = mcp.Meta{"mode": "url", "url": url}

	result, err := h.session.Elicit(ctx, params)
	if err != nil {
		return false, fmt.Errorf("URL elicitation failed: %w", err)
	}

	return result.Action == "accept", nil
}

// OpenDashboard requests the client to open a dashboard URL
func (h *ElicitationHelper) OpenDashboard(ctx context.Context, dashboardID, instanceURL string) (bool, error) {
	dashboardURL := fmt.Sprintf("%s/dashboard/%s", instanceURL, dashboardID)
	message := fmt.Sprintf("Would you like to open the dashboard in IBM Cloud Logs?\n\nDashboard URL: %s", dashboardURL)
	return h.OpenURL(ctx, message, dashboardURL)
}

// OpenAlert requests the client to open an alert URL
func (h *ElicitationHelper) OpenAlert(ctx context.Context, alertID, instanceURL string) (bool, error) {
	alertURL := fmt.Sprintf("%s/alerts/%s", instanceURL, alertID)
	message := fmt.Sprintf("Would you like to open the alert in IBM Cloud Logs?\n\nAlert URL: %s", alertURL)
	return h.OpenURL(ctx, message, alertURL)
}

// OpenDocumentation requests the client to open documentation
func (h *ElicitationHelper) OpenDocumentation(ctx context.Context, topic string) (bool, error) {
	// Map topics to IBM Cloud Logs documentation URLs
	docURLs := map[string]string{
		"dataprime":   "https://cloud.ibm.com/docs/cloud-logs?topic=cloud-logs-dataprime-reference",
		"alerts":      "https://cloud.ibm.com/docs/cloud-logs?topic=cloud-logs-alerts",
		"dashboards":  "https://cloud.ibm.com/docs/cloud-logs?topic=cloud-logs-custom-dashboards",
		"policies":    "https://cloud.ibm.com/docs/cloud-logs?topic=cloud-logs-tco-optimizer",
		"enrichments": "https://cloud.ibm.com/docs/cloud-logs?topic=cloud-logs-enriching-data",
		"iam":         "https://cloud.ibm.com/docs/cloud-logs?topic=cloud-logs-iam-actions",
		"general":     "https://cloud.ibm.com/docs/cloud-logs",
	}

	url, ok := docURLs[topic]
	if !ok {
		url = docURLs["general"]
	}

	message := fmt.Sprintf("Would you like to open the IBM Cloud Logs documentation for %s?\n\nDocumentation URL: %s", topic, url)
	return h.OpenURL(ctx, message, url)
}

// ============================================================================
// RCA (Root Cause Analysis) Elicitation Support - SOTA 2025
// ============================================================================

// RCA-specific elicitation schemas for interactive root cause analysis
func init() {
	// Incident context clarification
	CommonElicitationSchemas["rca_incident_context"] = &ElicitationSchema{
		Type: "object",
		Properties: map[string]ElicitationProperty{
			"incident_time": {
				Type:        "string",
				Description: "When did the incident start? (ISO8601 format or relative like '30 minutes ago')",
			},
			"affected_services": {
				Type:        "string",
				Description: "Which services are affected? (comma-separated)",
			},
			"symptoms": {
				Type:        "string",
				Description: "What symptoms are you observing? (e.g., high latency, errors, timeouts)",
			},
			"recent_changes": {
				Type:        "string",
				Description: "Any recent deployments or configuration changes?",
			},
		},
		Required: []string{"symptoms"},
	}

	// Root cause hypothesis confirmation
	CommonElicitationSchemas["rca_hypothesis_confirm"] = &ElicitationSchema{
		Type: "object",
		Properties: map[string]ElicitationProperty{
			"hypothesis_accepted": {
				Type:        "boolean",
				Description: "Do you agree with this root cause hypothesis?",
				Default:     false,
			},
			"additional_context": {
				Type:        "string",
				Description: "Any additional context or corrections?",
			},
			"next_action": {
				Type:        "string",
				Description: "What would you like to do next?",
				Enum:        []string{"investigate_further", "create_alert", "view_traces", "done"},
				Default:     "investigate_further",
			},
		},
		Required: []string{"hypothesis_accepted"},
	}

	// Trace ID selection for distributed tracing
	CommonElicitationSchemas["rca_trace_selection"] = &ElicitationSchema{
		Type: "object",
		Properties: map[string]ElicitationProperty{
			"trace_id": {
				Type:        "string",
				Description: "Enter a trace ID to investigate the full request flow",
			},
			"time_window": {
				Type:        "string",
				Description: "Time window to search for the trace",
				Enum:        []string{"15m", "1h", "6h", "24h"},
				Default:     "1h",
			},
		},
		Required: []string{"trace_id"},
	}

	// Delta analysis parameters
	CommonElicitationSchemas["rca_delta_params"] = &ElicitationSchema{
		Type: "object",
		Properties: map[string]ElicitationProperty{
			"incident_time": {
				Type:        "string",
				Description: "When did the incident start? (ISO8601 format)",
			},
			"comparison_window": {
				Type:        "string",
				Description: "How far before/after to compare?",
				Enum:        []string{"5m", "15m", "30m", "1h"},
				Default:     "15m",
			},
			"focus_service": {
				Type:        "string",
				Description: "Focus on a specific service? (optional)",
			},
		},
		Required: []string{"incident_time"},
	}
}

// RequestIncidentContext asks the user for incident context to improve RCA
func (h *ElicitationHelper) RequestIncidentContext(ctx context.Context) (*RCAIncidentContext, error) {
	message := `To help investigate this incident, please provide some context:

- When did the problem start?
- Which services are affected?
- What symptoms are you seeing?
- Any recent changes?`

	result, err := h.Elicit(ctx, message, CommonElicitationSchemas["rca_incident_context"])
	if err != nil {
		return nil, err
	}

	if result.Action != "accept" {
		return nil, fmt.Errorf("incident context request cancelled")
	}

	return &RCAIncidentContext{
		IncidentTime:     result.Content["incident_time"].(string),
		AffectedServices: result.Content["affected_services"].(string),
		Symptoms:         result.Content["symptoms"].(string),
		RecentChanges:    result.Content["recent_changes"].(string),
	}, nil
}

// RCAIncidentContext contains user-provided incident context
type RCAIncidentContext struct {
	IncidentTime     string
	AffectedServices string
	Symptoms         string
	RecentChanges    string
}

// ConfirmRCAHypothesis asks the user to confirm or reject a root cause hypothesis
func (h *ElicitationHelper) ConfirmRCAHypothesis(ctx context.Context, hypothesis string, confidence string) (*RCAHypothesisResponse, error) {
	message := fmt.Sprintf(`## Root Cause Hypothesis

**Hypothesis:** %s

**Confidence:** %s

Do you agree with this assessment? Would you like to investigate further?`, hypothesis, confidence)

	result, err := h.Elicit(ctx, message, CommonElicitationSchemas["rca_hypothesis_confirm"])
	if err != nil {
		return nil, err
	}

	if result.Action != "accept" {
		return nil, fmt.Errorf("hypothesis confirmation cancelled")
	}

	accepted, _ := result.Content["hypothesis_accepted"].(bool)
	additionalContext, _ := result.Content["additional_context"].(string)
	nextAction, _ := result.Content["next_action"].(string)
	if nextAction == "" {
		nextAction = "investigate_further"
	}

	return &RCAHypothesisResponse{
		Accepted:          accepted,
		AdditionalContext: additionalContext,
		NextAction:        nextAction,
	}, nil
}

// RCAHypothesisResponse contains the user's response to a hypothesis
type RCAHypothesisResponse struct {
	Accepted          bool
	AdditionalContext string
	NextAction        string // investigate_further, create_alert, view_traces, done
}

// RequestTraceID asks the user for a trace ID to investigate
func (h *ElicitationHelper) RequestTraceID(ctx context.Context) (string, string, error) {
	message := `To trace the full request flow across services, please provide a trace ID.

You can find trace IDs in:
- Log entries (look for trace_id, traceId, or X-Trace-ID fields)
- APM tools (Instana, Datadog, etc.)
- Request headers from affected requests`

	result, err := h.Elicit(ctx, message, CommonElicitationSchemas["rca_trace_selection"])
	if err != nil {
		return "", "", err
	}

	if result.Action != "accept" {
		return "", "", fmt.Errorf("trace ID request cancelled")
	}

	traceID, _ := result.Content["trace_id"].(string)
	timeWindow, _ := result.Content["time_window"].(string)
	if timeWindow == "" {
		timeWindow = "1h"
	}

	return traceID, timeWindow, nil
}

// RequestDeltaAnalysisParams asks the user for delta analysis parameters
func (h *ElicitationHelper) RequestDeltaAnalysisParams(ctx context.Context) (*RCADeltaParams, error) {
	message := `To compare log patterns before and after the incident, please provide:

- When the incident started (ISO8601 format)
- How far to look before/after that time
- Optionally, a specific service to focus on`

	result, err := h.Elicit(ctx, message, CommonElicitationSchemas["rca_delta_params"])
	if err != nil {
		return nil, err
	}

	if result.Action != "accept" {
		return nil, fmt.Errorf("delta analysis configuration cancelled")
	}

	incidentTime, _ := result.Content["incident_time"].(string)
	window, _ := result.Content["comparison_window"].(string)
	if window == "" {
		window = "15m"
	}
	focusService, _ := result.Content["focus_service"].(string)

	return &RCADeltaParams{
		IncidentTime: incidentTime,
		Window:       window,
		FocusService: focusService,
	}, nil
}

// RCADeltaParams contains parameters for delta analysis
type RCADeltaParams struct {
	IncidentTime string
	Window       string
	FocusService string
}
