// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements elicitation support for interactive user input.
package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
