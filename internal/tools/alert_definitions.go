// Package tools provides MCP tools for IBM Cloud Logs operations.
package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// GetAlertDefinitionTool retrieves a specific alert definition by ID
type GetAlertDefinitionTool struct {
	*BaseTool
}

// NewGetAlertDefinitionTool creates a new tool instance
func NewGetAlertDefinitionTool(client *client.Client, logger *zap.Logger) *GetAlertDefinitionTool {
	return &GetAlertDefinitionTool{BaseTool: NewBaseTool(client, logger)}
}

// Name returns the tool name
func (t *GetAlertDefinitionTool) Name() string { return "get_alert_definition" }

// Description returns the tool description
func (t *GetAlertDefinitionTool) Description() string {
	return "Retrieve a specific alert definition by its ID"
}

// InputSchema returns the input schema
func (t *GetAlertDefinitionTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type": "string", "description": "Alert definition ID",
			},
		},
		"required": []string{"id"},
	}
}

// Execute executes the tool
func (t *GetAlertDefinitionTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/alert_definitions/" + id})
	if err != nil {
		return HandleGetError(err, "Alert definition", id, "list_alert_definitions"), nil
	}
	return t.FormatResponseWithSuggestions(result, "get_alert_definition")
}

// ListAlertDefinitionsTool lists all alert definitions
type ListAlertDefinitionsTool struct {
	*BaseTool
}

// NewListAlertDefinitionsTool creates a new tool instance
func NewListAlertDefinitionsTool(client *client.Client, logger *zap.Logger) *ListAlertDefinitionsTool {
	return &ListAlertDefinitionsTool{BaseTool: NewBaseTool(client, logger)}
}

// Name returns the tool name
func (t *ListAlertDefinitionsTool) Name() string { return "list_alert_definitions" }

// Description returns the tool description
func (t *ListAlertDefinitionsTool) Description() string {
	return "List all alert definitions"
}

// InputSchema returns the input schema
func (t *ListAlertDefinitionsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// Execute executes the tool
func (t *ListAlertDefinitionsTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/alert_definitions"})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponseWithSuggestions(result, "list_alert_definitions")
}

// CreateAlertDefinitionTool creates a new alert definition
type CreateAlertDefinitionTool struct {
	*BaseTool
}

// NewCreateAlertDefinitionTool creates a new tool instance
func NewCreateAlertDefinitionTool(client *client.Client, logger *zap.Logger) *CreateAlertDefinitionTool {
	return &CreateAlertDefinitionTool{BaseTool: NewBaseTool(client, logger)}
}

// Name returns the tool name
func (t *CreateAlertDefinitionTool) Name() string { return "create_alert_definition" }

// Description returns the tool description
func (t *CreateAlertDefinitionTool) Description() string {
	return `Create a new alert definition to monitor log patterns and trigger notifications.

**Related tools:** list_alert_definitions, get_alert_definition, create_alert, create_outgoing_webhook

**Alert Types:**
- logs_immediate: Triggered immediately when condition matches
- logs_threshold: Triggered when count exceeds threshold over time window
- logs_ratio: Triggered when ratio between two queries exceeds threshold
- logs_anomaly: Triggered on anomaly detection
- logs_new_value: Triggered when a new value appears in logs`
}

// InputSchema returns the input schema
func (t *CreateAlertDefinitionTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"definition": map[string]interface{}{
				"type":        "object",
				"description": "Alert definition configuration",
				"example": map[string]interface{}{
					"name":        "High Error Rate Alert",
					"description": "Triggers when error rate exceeds threshold",
					"enabled":     true,
					"priority":    "P2",
					"type":        "logs_threshold",
					"condition": map[string]interface{}{
						"threshold": map[string]interface{}{
							"condition":            "more_than",
							"threshold":            100,
							"time_window_seconds":  300,
							"group_by_keys":        []string{},
							"condition_match_type": "any",
						},
					},
					"filter": map[string]interface{}{
						"simple_filter": map[string]interface{}{
							"query": "severity:>=5",
						},
					},
					"notification_groups": []map[string]interface{}{
						{
							"notifications": []map[string]interface{}{
								{
									"webhook_id":                     "webhook-uuid-here",
									"notify_on":                      "triggered_only",
									"retriggering_period_seconds":    60,
									"notify_on_resolved":             true,
									"integration_connection_details": map[string]interface{}{},
								},
							},
						},
					},
				},
			},
			"dry_run": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, validates the alert definition without creating it. Use this to preview and check for errors.",
				"default":     false,
			},
		},
		"required": []string{"definition"},
	}
}

// Execute executes the tool
func (t *CreateAlertDefinitionTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	def, err := GetObjectParam(arguments, "definition", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Check for dry-run mode
	dryRun, _ := GetBoolParam(arguments, "dry_run", false)
	if dryRun {
		return t.validateAlertDefinition(def)
	}

	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/alert_definitions", Body: def})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponseWithSuggestions(result, "create_alert_definition")
}

// validateAlertDefinition performs dry-run validation
func (t *CreateAlertDefinitionTool) validateAlertDefinition(def map[string]interface{}) (*mcp.CallToolResult, error) {
	result := &ValidationResult{
		Valid:   true,
		Summary: make(map[string]interface{}),
	}

	// Validate required fields
	requiredFields := []string{"name", "type"}
	for _, field := range requiredFields {
		if _, ok := def[field]; !ok {
			result.Errors = append(result.Errors, "Missing required field: "+field)
			result.Valid = false
		}
	}

	// Validate name
	if name, ok := def["name"].(string); ok {
		result.Summary["name"] = name
	}

	// Validate type
	validTypes := map[string]bool{
		"logs_immediate": true,
		"logs_threshold": true,
		"logs_ratio":     true,
		"logs_anomaly":   true,
		"logs_new_value": true,
	}
	if alertType, ok := def["type"].(string); ok {
		if !validTypes[alertType] {
			result.Errors = append(result.Errors, "Invalid alert type: "+alertType+". Valid types: logs_immediate, logs_threshold, logs_ratio, logs_anomaly, logs_new_value")
			result.Valid = false
		}
		result.Summary["type"] = alertType
	}

	// Check for condition (required for most types)
	if _, ok := def["condition"]; !ok {
		result.Warnings = append(result.Warnings, "No condition specified - alert may not trigger as expected")
	}

	// Add suggestions
	if result.Valid {
		result.Suggestions = append(result.Suggestions, "Alert definition configuration is valid")
		result.Suggestions = append(result.Suggestions, "Remove dry_run parameter to create the alert definition")
	} else {
		result.Suggestions = append(result.Suggestions, "Fix the errors above before creating")
	}

	result.EstimatedImpact = &ImpactEstimate{RiskLevel: "low"}
	return FormatDryRunResult(result, "Alert Definition", def), nil
}

// UpdateAlertDefinitionTool updates an existing alert definition
type UpdateAlertDefinitionTool struct {
	*BaseTool
}

// NewUpdateAlertDefinitionTool creates a new tool instance
func NewUpdateAlertDefinitionTool(client *client.Client, logger *zap.Logger) *UpdateAlertDefinitionTool {
	return &UpdateAlertDefinitionTool{BaseTool: NewBaseTool(client, logger)}
}

// Name returns the tool name
func (t *UpdateAlertDefinitionTool) Name() string { return "update_alert_definition" }

// Description returns the tool description
func (t *UpdateAlertDefinitionTool) Description() string {
	return "Update an existing alert definition"
}

// InputSchema returns the input schema
func (t *UpdateAlertDefinitionTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":         map[string]interface{}{"type": "string", "description": "Alert definition ID"},
			"definition": map[string]interface{}{"type": "object", "description": "Updated definition"},
		},
		"required": []string{"id", "definition"},
	}
}

// Execute executes the tool
func (t *UpdateAlertDefinitionTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	def, err := GetObjectParam(arguments, "definition", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/alert_definitions/" + id, Body: def})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponseWithSuggestions(result, "update_alert_definition")
}

// DeleteAlertDefinitionTool deletes an alert definition
type DeleteAlertDefinitionTool struct {
	*BaseTool
}

// NewDeleteAlertDefinitionTool creates a new tool instance
func NewDeleteAlertDefinitionTool(client *client.Client, logger *zap.Logger) *DeleteAlertDefinitionTool {
	return &DeleteAlertDefinitionTool{BaseTool: NewBaseTool(client, logger)}
}

// Name returns the tool name
func (t *DeleteAlertDefinitionTool) Name() string { return "delete_alert_definition" }

// Annotations returns tool hints for LLMs
func (t *DeleteAlertDefinitionTool) Annotations() *mcp.ToolAnnotations {
	return DeleteAnnotations("Delete Alert Definition")
}

// Description returns the tool description
func (t *DeleteAlertDefinitionTool) Description() string {
	return "Delete an alert definition"
}

// InputSchema returns the input schema
func (t *DeleteAlertDefinitionTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{"type": "string", "description": "Alert definition ID"},
		},
		"required": []string{"id"},
	}
}

// Execute executes the tool
func (t *DeleteAlertDefinitionTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/alert_definitions/" + id})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponseWithSuggestions(result, "delete_alert_definition")
}
