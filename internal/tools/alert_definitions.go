// Package tools provides MCP tools for IBM Cloud Logs operations.
package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// GetAlertDefinitionTool retrieves a specific alert definition by ID
type GetAlertDefinitionTool struct {
	*BaseTool
}

// NewGetAlertDefinitionTool creates a new tool instance
func NewGetAlertDefinitionTool(client client.Doer, logger *zap.Logger) *GetAlertDefinitionTool {
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
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: apiPath("/v1/alert_definitions", id)})
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
func NewListAlertDefinitionsTool(client client.Doer, logger *zap.Logger) *ListAlertDefinitionsTool {
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
func NewCreateAlertDefinitionTool(client client.Doer, logger *zap.Logger) *CreateAlertDefinitionTool {
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
					"description": "Fires when error logs exceed 10% of total logs over 5 minutes",
					"enabled":     true,
					"type":        "logs_ratio_threshold",
					"logs_ratio_threshold": map[string]interface{}{
						"condition_type": "more_than_or_unspecified",
						"group_by_for":   "both_or_unspecified",
						"numerator": map[string]interface{}{
							"simple_filter": map[string]interface{}{
								"lucene_query": "subsystemname:payment-service AND severity:error",
							},
						},
						"numerator_alias": "errors",
						"denominator": map[string]interface{}{
							"simple_filter": map[string]interface{}{
								"lucene_query": "subsystemname:payment-service",
							},
						},
						"denominator_alias": "total",
						"rules": []map[string]interface{}{
							{
								"condition": map[string]interface{}{
									"threshold": 0.1,
									"time_window": map[string]interface{}{
										"logs_ratio_time_window_specific_value": "minutes_5_or_unspecified",
									},
									"condition_type": "more_than_or_unspecified",
								},
								"override": map[string]interface{}{"priority": "p1"},
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

// v3 alert definition enums, as enforced by the live API's deserializer.
var validAlertDefTypes = []string{
	"logs_immediate_or_unspecified",
	"logs_threshold",
	"logs_anomaly",
	"logs_ratio_threshold",
	"logs_new_value",
	"logs_unique_count",
	"logs_time_relative_threshold",
	"metric_threshold",
	"metric_anomaly",
	"flow",
}

var validAlertDefPriorities = []string{"p5_or_unspecified", "p4", "p3", "p2", "p1"}

// alertTypeConfigKey returns the definition key that must hold the config
// object for a given alert type. For every type except the immediate default,
// the key matches the type name (e.g. type "logs_ratio_threshold" requires a
// "logs_ratio_threshold" object).
func alertTypeConfigKey(alertType string) string {
	if alertType == "logs_immediate_or_unspecified" {
		return ""
	}
	return alertType
}

// rulesHaveOverride reports whether any rule inside the type config object
// carries an override. The live API rejects definitions that set a top-level
// priority alongside rule overrides.
func rulesHaveOverride(typeConfig map[string]interface{}) bool {
	rules, ok := typeConfig["rules"].([]interface{})
	if !ok {
		return false
	}
	for _, r := range rules {
		if rule, ok := r.(map[string]interface{}); ok {
			if _, has := rule["override"]; has {
				return true
			}
		}
	}
	return false
}

// validateAlertDefinitionConfig checks an alert definition against the live
// v3 API contract. It intentionally covers the failure modes the API is known
// to reject (type/priority enums, missing type config, priority-vs-override
// conflict) rather than replicating the full server-side schema.
func validateAlertDefinitionConfig(def map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		Valid:   true,
		Summary: make(map[string]interface{}),
	}

	for _, field := range []string{"name", "type"} {
		if _, ok := def[field]; !ok {
			result.Errors = append(result.Errors, "Missing required field: "+field)
			result.Valid = false
		}
	}

	if name, ok := def["name"].(string); ok {
		result.Summary["name"] = name
	}

	alertType, _ := def["type"].(string)
	if alertType != "" {
		result.Summary["type"] = alertType
		valid := false
		for _, t := range validAlertDefTypes {
			if alertType == t {
				valid = true
				break
			}
		}
		if !valid {
			result.Errors = append(result.Errors,
				"Invalid alert type: "+alertType+". Valid types: "+joinStrings(validAlertDefTypes, ", "))
			result.Valid = false
		} else if key := alertTypeConfigKey(alertType); key != "" {
			if _, ok := def[key].(map[string]interface{}); !ok {
				result.Errors = append(result.Errors,
					"Alert type "+alertType+" requires a matching config object under the key \""+key+"\"")
				result.Valid = false
			}
		}
	}

	if priority, ok := def["priority"].(string); ok {
		valid := false
		for _, p := range validAlertDefPriorities {
			if priority == p {
				valid = true
				break
			}
		}
		if !valid {
			result.Errors = append(result.Errors,
				"Invalid priority: "+priority+". Valid values (lowercase): "+joinStrings(validAlertDefPriorities, ", "))
			result.Valid = false
		}
		if key := alertTypeConfigKey(alertType); key != "" {
			if typeConfig, ok := def[key].(map[string]interface{}); ok && rulesHaveOverride(typeConfig) {
				result.Errors = append(result.Errors,
					"Cannot set a top-level priority when rules define an override — remove one of them")
				result.Valid = false
			}
		}
	}

	if result.Valid {
		result.Suggestions = append(result.Suggestions, "Alert definition configuration is valid")
		result.Suggestions = append(result.Suggestions, "Remove dry_run parameter to create the alert definition")
		result.Warnings = append(result.Warnings,
			"Dry-run checks known API constraints but is not a full schema validation — the API may still reject unknown fields or enum values")
	} else {
		result.Suggestions = append(result.Suggestions, "Fix the errors above before creating")
	}

	result.EstimatedImpact = &ImpactEstimate{RiskLevel: "low"}
	return result
}

// validateAlertDefinition performs dry-run validation
func (t *CreateAlertDefinitionTool) validateAlertDefinition(def map[string]interface{}) (*mcp.CallToolResult, error) {
	return FormatDryRunResult(validateAlertDefinitionConfig(def), "Alert Definition", def), nil
}

// UpdateAlertDefinitionTool updates an existing alert definition
type UpdateAlertDefinitionTool struct {
	*BaseTool
}

// NewUpdateAlertDefinitionTool creates a new tool instance
func NewUpdateAlertDefinitionTool(client client.Doer, logger *zap.Logger) *UpdateAlertDefinitionTool {
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
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: apiPath("/v1/alert_definitions", id), Body: def})
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
func NewDeleteAlertDefinitionTool(client client.Doer, logger *zap.Logger) *DeleteAlertDefinitionTool {
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
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: apiPath("/v1/alert_definitions", id)})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponseWithSuggestions(result, "delete_alert_definition")
}
