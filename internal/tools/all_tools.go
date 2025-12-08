package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// This file contains all remaining tools in a condensed format for brevity

// GetRuleGroupTool retrieves a specific rule group by ID.
type GetRuleGroupTool struct{ *BaseTool }

// NewGetRuleGroupTool creates a new tool instance
func NewGetRuleGroupTool(c *client.Client, l *zap.Logger) *GetRuleGroupTool {
	return &GetRuleGroupTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *GetRuleGroupTool) Name() string { return "get_rule_group" }

// Description returns the tool description
func (t *GetRuleGroupTool) Description() string { return "Get a rule group by ID" }

// InputSchema returns the input schema
func (t *GetRuleGroupTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string", "description": "Rule group ID"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *GetRuleGroupTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/rule_groups/" + id})
	if err != nil {
		return HandleGetError(err, "Rule group", id, "list_rule_groups"), nil
	}
	return t.FormatResponse(res)
}

// ListRuleGroupsTool lists all rule groups.
type ListRuleGroupsTool struct{ *BaseTool }

// NewListRuleGroupsTool creates a new tool instance
func NewListRuleGroupsTool(c *client.Client, l *zap.Logger) *ListRuleGroupsTool {
	return &ListRuleGroupsTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ListRuleGroupsTool) Name() string { return "list_rule_groups" }

// Description returns the tool description
func (t *ListRuleGroupsTool) Description() string {
	return `List all rule groups for parsing and transforming log data.

**Related tools:** get_rule_group, create_rule_group, update_rule_group, delete_rule_group`
}

// InputSchema returns the input schema
func (t *ListRuleGroupsTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}

// Execute executes the tool
func (t *ListRuleGroupsTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/rule_groups"})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// CreateRuleGroupTool creates a new rule group.
type CreateRuleGroupTool struct{ *BaseTool }

// NewCreateRuleGroupTool creates a new tool instance
func NewCreateRuleGroupTool(c *client.Client, l *zap.Logger) *CreateRuleGroupTool {
	return &CreateRuleGroupTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *CreateRuleGroupTool) Name() string { return "create_rule_group" }

// Description returns the tool description
func (t *CreateRuleGroupTool) Description() string { return "Create a new rule group" }

// InputSchema returns the input schema
func (t *CreateRuleGroupTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"rule_group": map[string]interface{}{"type": "object"}}, "required": []string{"rule_group"}}
}

// Execute executes the tool
func (t *CreateRuleGroupTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	rg, _ := GetObjectParam(args, "rule_group", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/rule_groups", Body: rg})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// UpdateRuleGroupTool updates an existing rule group.
type UpdateRuleGroupTool struct{ *BaseTool }

// NewUpdateRuleGroupTool creates a new tool instance
func NewUpdateRuleGroupTool(c *client.Client, l *zap.Logger) *UpdateRuleGroupTool {
	return &UpdateRuleGroupTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *UpdateRuleGroupTool) Name() string { return "update_rule_group" }

// Description returns the tool description
func (t *UpdateRuleGroupTool) Description() string { return "Update a rule group" }

// InputSchema returns the input schema
func (t *UpdateRuleGroupTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "rule_group": map[string]interface{}{"type": "object"}}, "required": []string{"id", "rule_group"}}
}

// Execute executes the tool
func (t *UpdateRuleGroupTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	rg, _ := GetObjectParam(args, "rule_group", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/rule_groups/" + id, Body: rg})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// DeleteRuleGroupTool deletes a rule group.
type DeleteRuleGroupTool struct{ *BaseTool }

// NewDeleteRuleGroupTool creates a new tool instance
func NewDeleteRuleGroupTool(c *client.Client, l *zap.Logger) *DeleteRuleGroupTool {
	return &DeleteRuleGroupTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *DeleteRuleGroupTool) Name() string { return "delete_rule_group" }

// Description returns the tool description
func (t *DeleteRuleGroupTool) Description() string { return "Delete a rule group" }

// InputSchema returns the input schema
func (t *DeleteRuleGroupTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *DeleteRuleGroupTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/rule_groups/" + id})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// GetOutgoingWebhookTool retrieves a specific outgoing webhook by ID.
type GetOutgoingWebhookTool struct{ *BaseTool }

// NewGetOutgoingWebhookTool creates a new tool instance
func NewGetOutgoingWebhookTool(c *client.Client, l *zap.Logger) *GetOutgoingWebhookTool {
	return &GetOutgoingWebhookTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *GetOutgoingWebhookTool) Name() string { return "get_outgoing_webhook" }

// Description returns the tool description
func (t *GetOutgoingWebhookTool) Description() string { return "Get an outgoing webhook by ID" }

// InputSchema returns the input schema
func (t *GetOutgoingWebhookTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *GetOutgoingWebhookTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/outgoing_webhooks/" + id})
	if err != nil {
		return HandleGetError(err, "Outgoing webhook", id, "list_outgoing_webhooks"), nil
	}
	return t.FormatResponse(res)
}

// ListOutgoingWebhooksTool lists all outgoing webhooks.
type ListOutgoingWebhooksTool struct{ *BaseTool }

// NewListOutgoingWebhooksTool creates a new tool instance
func NewListOutgoingWebhooksTool(c *client.Client, l *zap.Logger) *ListOutgoingWebhooksTool {
	return &ListOutgoingWebhooksTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ListOutgoingWebhooksTool) Name() string { return "list_outgoing_webhooks" }

// Description returns the tool description
func (t *ListOutgoingWebhooksTool) Description() string {
	return `List all outgoing webhooks configured for alert notifications.

**Related tools:** get_outgoing_webhook, create_outgoing_webhook, update_outgoing_webhook, delete_outgoing_webhook, create_alert`
}

// InputSchema returns the input schema
func (t *ListOutgoingWebhooksTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}

// Execute executes the tool
func (t *ListOutgoingWebhooksTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/outgoing_webhooks"})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// CreateOutgoingWebhookTool creates a new outgoing webhook.
type CreateOutgoingWebhookTool struct{ *BaseTool }

// NewCreateOutgoingWebhookTool creates a new tool instance
func NewCreateOutgoingWebhookTool(c *client.Client, l *zap.Logger) *CreateOutgoingWebhookTool {
	return &CreateOutgoingWebhookTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *CreateOutgoingWebhookTool) Name() string { return "create_outgoing_webhook" }

// Description returns the tool description
func (t *CreateOutgoingWebhookTool) Description() string {
	return `Create an outgoing webhook to send notifications from IBM Cloud Logs to external services.

**Related tools:** list_outgoing_webhooks, get_outgoing_webhook, create_alert (connect alerts to webhooks)

**Webhook Types:**
- generic: Custom HTTP webhook to any endpoint
- slack: Slack incoming webhook integration
- pagerduty: PagerDuty integration for incident management
- ibm_event_notifications: IBM Cloud Event Notifications service`
}

// InputSchema returns the input schema
func (t *CreateOutgoingWebhookTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"webhook": map[string]interface{}{
				"type":        "object",
				"description": "Webhook configuration object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Display name for the webhook",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Webhook type: generic, slack, pagerduty, ibm_event_notifications",
						"enum":        []string{"generic", "slack", "pagerduty", "ibm_event_notifications"},
					},
					"url": map[string]interface{}{
						"type":        "string",
						"description": "Target URL for the webhook",
					},
				},
			},
		},
		"required": []string{"webhook"},
		"examples": []interface{}{
			map[string]interface{}{
				"webhook": map[string]interface{}{
					"name": "Slack Alerts",
					"type": "slack",
					"url":  "https://hooks.slack.com/services/XXX/YYY/ZZZ",
				},
			},
			map[string]interface{}{
				"webhook": map[string]interface{}{
					"name": "PagerDuty Critical",
					"type": "pagerduty",
					"url":  "https://events.pagerduty.com/v2/enqueue",
				},
			},
		},
	}
}

// Execute executes the tool
func (t *CreateOutgoingWebhookTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	wh, _ := GetObjectParam(args, "webhook", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/outgoing_webhooks", Body: wh})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// UpdateOutgoingWebhookTool updates an existing outgoing webhook.
type UpdateOutgoingWebhookTool struct{ *BaseTool }

// NewUpdateOutgoingWebhookTool creates a new tool instance
func NewUpdateOutgoingWebhookTool(c *client.Client, l *zap.Logger) *UpdateOutgoingWebhookTool {
	return &UpdateOutgoingWebhookTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *UpdateOutgoingWebhookTool) Name() string { return "update_outgoing_webhook" }

// Description returns the tool description
func (t *UpdateOutgoingWebhookTool) Description() string { return "Update an outgoing webhook" }

// InputSchema returns the input schema
func (t *UpdateOutgoingWebhookTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "webhook": map[string]interface{}{"type": "object"}}, "required": []string{"id", "webhook"}}
}

// Execute executes the tool
func (t *UpdateOutgoingWebhookTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	wh, _ := GetObjectParam(args, "webhook", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/outgoing_webhooks/" + id, Body: wh})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// DeleteOutgoingWebhookTool deletes an outgoing webhook.
type DeleteOutgoingWebhookTool struct{ *BaseTool }

// NewDeleteOutgoingWebhookTool creates a new tool instance
func NewDeleteOutgoingWebhookTool(c *client.Client, l *zap.Logger) *DeleteOutgoingWebhookTool {
	return &DeleteOutgoingWebhookTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *DeleteOutgoingWebhookTool) Name() string { return "delete_outgoing_webhook" }

// Description returns the tool description
func (t *DeleteOutgoingWebhookTool) Description() string { return "Delete an outgoing webhook" }

// InputSchema returns the input schema
func (t *DeleteOutgoingWebhookTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *DeleteOutgoingWebhookTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/outgoing_webhooks/" + id})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// GetPolicyTool retrieves a specific policy by ID.
type GetPolicyTool struct{ *BaseTool }

// NewGetPolicyTool creates a new tool instance
func NewGetPolicyTool(c *client.Client, l *zap.Logger) *GetPolicyTool {
	return &GetPolicyTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *GetPolicyTool) Name() string { return "get_policy" }

// Description returns the tool description
func (t *GetPolicyTool) Description() string { return "Get a policy by ID" }

// InputSchema returns the input schema
func (t *GetPolicyTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *GetPolicyTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/policies/" + id})
	if err != nil {
		return HandleGetError(err, "Policy", id, "list_policies"), nil
	}
	return t.FormatResponse(res)
}

// ListPoliciesTool lists all policies.
type ListPoliciesTool struct{ *BaseTool }

// NewListPoliciesTool creates a new tool instance
func NewListPoliciesTool(c *client.Client, l *zap.Logger) *ListPoliciesTool {
	return &ListPoliciesTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ListPoliciesTool) Name() string { return "list_policies" }

// Description returns the tool description
func (t *ListPoliciesTool) Description() string {
	return `List all log routing policies configured in IBM Cloud Logs.

**Related tools:** get_policy, create_policy, update_policy, delete_policy`
}

// InputSchema returns the input schema
func (t *ListPoliciesTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}

// Execute executes the tool
func (t *ListPoliciesTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/policies"})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// CreatePolicyTool creates a new policy.
type CreatePolicyTool struct{ *BaseTool }

// NewCreatePolicyTool creates a new tool instance
func NewCreatePolicyTool(c *client.Client, l *zap.Logger) *CreatePolicyTool {
	return &CreatePolicyTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *CreatePolicyTool) Name() string { return "create_policy" }

// Description returns the tool description
func (t *CreatePolicyTool) Description() string {
	return `Create a log routing policy to control how logs are processed and stored.

**Related tools:** list_policies, get_policy, update_policy, delete_policy

**Policy Types:**
- block: Block logs matching the filter
- send_data: Route logs to specific pipelines or storage
- quota: Apply rate limits to log ingestion

**Priority Levels:**
- type_low: Lowest priority (least important logs)
- type_medium: Standard priority
- type_high: High priority (important logs)
- type_unspecified: Default priority`
}

// InputSchema returns the input schema
func (t *CreatePolicyTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"policy": map[string]interface{}{
				"type":        "object",
				"description": "Policy configuration object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Display name for the policy",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Description of the policy purpose",
					},
					"priority": map[string]interface{}{
						"type":        "string",
						"description": "Priority level for the policy",
						"enum":        []string{"type_low", "type_medium", "type_high", "type_unspecified"},
					},
					"application_rule": map[string]interface{}{
						"type":        "object",
						"description": "Rule to match logs by application name",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type":        "string",
								"description": "Application name to match",
							},
							"rule_type_id": map[string]interface{}{
								"type":        "string",
								"description": "Matching rule type",
								"enum":        []string{"is", "is_not", "includes", "starts_with"},
							},
						},
					},
					"subsystem_rule": map[string]interface{}{
						"type":        "object",
						"description": "Rule to match logs by subsystem name",
					},
					"archive_retention": map[string]interface{}{
						"type":        "object",
						"description": "Archive retention configuration",
					},
					"log_rules": map[string]interface{}{
						"type":        "object",
						"description": "Log filtering rules with severity",
					},
				},
			},
			"dry_run": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, validates the configuration without creating the resource. Use this to preview what will be created.",
				"default":     false,
			},
		},
		"required": []string{"policy"},
		"examples": []interface{}{
			map[string]interface{}{
				"policy": map[string]interface{}{
					"name":        "Production Logs High Priority",
					"description": "Keep production logs with high priority for longer retention",
					"priority":    "type_high",
					"application_rule": map[string]interface{}{
						"name":         "production",
						"rule_type_id": "starts_with",
					},
				},
			},
			map[string]interface{}{
				"policy": map[string]interface{}{
					"name":        "Block Debug Logs",
					"description": "Block debug logs from ingestion to reduce costs",
					"priority":    "type_low",
					"log_rules": map[string]interface{}{
						"severities": []string{"debug", "verbose"},
					},
				},
				"dry_run": true,
			},
		},
	}
}

// Execute executes the tool
func (t *CreatePolicyTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	pol, _ := GetObjectParam(args, "policy", true)

	// Check for dry-run mode
	dryRun, _ := GetBoolParam(args, "dry_run", false)
	if dryRun {
		return t.validatePolicy(pol)
	}

	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/policies", Body: pol})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// validatePolicy performs dry-run validation for policy creation
func (t *CreatePolicyTool) validatePolicy(policy map[string]interface{}) (*mcp.CallToolResult, error) {
	result := &ValidationResult{
		Valid:   true,
		Summary: make(map[string]interface{}),
	}

	// Check required fields
	requiredErrors := ValidateRequiredFields(policy, []string{"name"})
	if len(requiredErrors) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, requiredErrors...)
	}

	// Validate priority field if present
	if errMsg := ValidateEnumField(policy, "priority", []string{"type_low", "type_medium", "type_high", "type_unspecified"}); errMsg != "" {
		result.Valid = false
		result.Errors = append(result.Errors, errMsg)
	}

	// Build summary
	if name, ok := policy["name"].(string); ok {
		result.Summary["name"] = name
	}
	if desc, ok := policy["description"].(string); ok {
		result.Summary["description"] = desc
	}
	if priority, ok := policy["priority"].(string); ok {
		result.Summary["priority"] = priority
	}

	// Add warnings and suggestions
	if _, hasAppRule := policy["application_rule"]; !hasAppRule {
		if _, hasSubRule := policy["subsystem_rule"]; !hasSubRule {
			result.Warnings = append(result.Warnings, "No application_rule or subsystem_rule defined - policy will apply to all logs")
		}
	}

	result.Suggestions = append(result.Suggestions, "Consider setting explicit retention policies to optimize costs")

	return FormatDryRunResult(result, "Policy", policy), nil
}

// UpdatePolicyTool updates an existing policy.
type UpdatePolicyTool struct{ *BaseTool }

// NewUpdatePolicyTool creates a new tool instance
func NewUpdatePolicyTool(c *client.Client, l *zap.Logger) *UpdatePolicyTool {
	return &UpdatePolicyTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *UpdatePolicyTool) Name() string { return "update_policy" }

// Description returns the tool description
func (t *UpdatePolicyTool) Description() string { return "Update a policy" }

// InputSchema returns the input schema
func (t *UpdatePolicyTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "policy": map[string]interface{}{"type": "object"}}, "required": []string{"id", "policy"}}
}

// Execute executes the tool
func (t *UpdatePolicyTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	pol, _ := GetObjectParam(args, "policy", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/policies/" + id, Body: pol})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// DeletePolicyTool deletes a policy.
type DeletePolicyTool struct{ *BaseTool }

// NewDeletePolicyTool creates a new tool instance
func NewDeletePolicyTool(c *client.Client, l *zap.Logger) *DeletePolicyTool {
	return &DeletePolicyTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *DeletePolicyTool) Name() string { return "delete_policy" }

// Description returns the tool description
func (t *DeletePolicyTool) Description() string { return "Delete a policy" }

// InputSchema returns the input schema
func (t *DeletePolicyTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *DeletePolicyTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/policies/" + id})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// GetE2MTool retrieves a specific events-to-metrics configuration by ID.
type GetE2MTool struct{ *BaseTool }

// NewGetE2MTool creates a new tool instance
func NewGetE2MTool(c *client.Client, l *zap.Logger) *GetE2MTool {
	return &GetE2MTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *GetE2MTool) Name() string { return "get_e2m" }

// Description returns the tool description
func (t *GetE2MTool) Description() string { return "Get an events-to-metrics configuration by ID" }

// InputSchema returns the input schema
func (t *GetE2MTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *GetE2MTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/events2metrics/" + id})
	if err != nil {
		return HandleGetError(err, "Events-to-metrics configuration", id, "list_e2m"), nil
	}
	return t.FormatResponse(res)
}

// ListE2MTool lists all events-to-metrics configurations.
type ListE2MTool struct{ *BaseTool }

// NewListE2MTool creates a new tool instance
func NewListE2MTool(c *client.Client, l *zap.Logger) *ListE2MTool {
	return &ListE2MTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ListE2MTool) Name() string { return "list_e2m" }

// Description returns the tool description
func (t *ListE2MTool) Description() string {
	return `List all Events-to-Metrics (E2M) configurations for converting logs to metrics.

**Related tools:** get_e2m, create_e2m, replace_e2m, delete_e2m`
}

// InputSchema returns the input schema
func (t *ListE2MTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}

// Execute executes the tool
func (t *ListE2MTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/events2metrics"})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// CreateE2MTool creates a new events-to-metrics configuration.
type CreateE2MTool struct{ *BaseTool }

// NewCreateE2MTool creates a new tool instance
func NewCreateE2MTool(c *client.Client, l *zap.Logger) *CreateE2MTool {
	return &CreateE2MTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *CreateE2MTool) Name() string { return "create_e2m" }

// Description returns the tool description
func (t *CreateE2MTool) Description() string {
	return `Create an Events-to-Metrics (E2M) configuration to convert log data into metrics.

**Related tools:** list_e2m, get_e2m, replace_e2m, delete_e2m

**Use Cases:**
- Convert error counts into metrics for dashboards
- Create SLI/SLO metrics from log data
- Reduce storage costs by summarizing logs as metrics
- Build custom metrics from structured log fields

**Metric Types:**
- counter: Counts occurrences of matching events
- gauge: Samples values from log fields
- histogram: Creates distribution of values`
}

// InputSchema returns the input schema
func (t *CreateE2MTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"e2m": map[string]interface{}{
				"type":        "object",
				"description": "Events-to-Metrics configuration object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Display name for the E2M configuration",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Description of what this metric measures",
					},
					"permutations_limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of unique label combinations (default: 30000)",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Source data type: logs2metrics or spans2metrics",
						"enum":        []string{"logs2metrics", "spans2metrics"},
					},
					"logs_query": map[string]interface{}{
						"type":        "object",
						"description": "Query to filter which logs to convert to metrics",
						"properties": map[string]interface{}{
							"lucene": map[string]interface{}{
								"type":        "string",
								"description": "Lucene query to filter logs",
							},
							"applicationname_filters": map[string]interface{}{
								"type":        "array",
								"description": "Application names to include",
							},
							"subsystemname_filters": map[string]interface{}{
								"type":        "array",
								"description": "Subsystem names to include",
							},
							"severity_filters": map[string]interface{}{
								"type":        "array",
								"description": "Severity levels to include",
							},
						},
					},
					"metric_fields": map[string]interface{}{
						"type":        "array",
						"description": "Fields to extract as metric values",
					},
					"metric_labels": map[string]interface{}{
						"type":        "array",
						"description": "Fields to use as metric labels",
					},
				},
			},
		},
		"required": []string{"e2m"},
		"examples": []interface{}{
			map[string]interface{}{
				"e2m": map[string]interface{}{
					"name":        "error_count_by_service",
					"description": "Count errors per service for SLO tracking",
					"type":        "logs2metrics",
					"logs_query": map[string]interface{}{
						"lucene":           "level:error",
						"severity_filters": []string{"error", "critical"},
					},
					"metric_labels": []map[string]interface{}{
						{"target_label": "service", "source_field": "applicationName"},
						{"target_label": "component", "source_field": "subsystemName"},
					},
				},
			},
			map[string]interface{}{
				"e2m": map[string]interface{}{
					"name":        "response_time_histogram",
					"description": "Response time distribution from API logs",
					"type":        "logs2metrics",
					"logs_query": map[string]interface{}{
						"lucene": "json.endpoint:* AND json.response_time:*",
					},
					"metric_fields": []map[string]interface{}{
						{"target_base_metric_name": "response_time_ms", "source_field": "json.response_time"},
					},
				},
			},
		},
	}
}

// Execute executes the tool
func (t *CreateE2MTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	e2m, _ := GetObjectParam(args, "e2m", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/events2metrics", Body: e2m})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// ReplaceE2MTool replaces an events-to-metrics configuration.
type ReplaceE2MTool struct{ *BaseTool }

// NewReplaceE2MTool creates a new tool instance
func NewReplaceE2MTool(c *client.Client, l *zap.Logger) *ReplaceE2MTool {
	return &ReplaceE2MTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ReplaceE2MTool) Name() string { return "replace_e2m" }

// Description returns the tool description
func (t *ReplaceE2MTool) Description() string { return "Replace an events-to-metrics configuration" }

// InputSchema returns the input schema
func (t *ReplaceE2MTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "e2m": map[string]interface{}{"type": "object"}}, "required": []string{"id", "e2m"}}
}

// Execute executes the tool
func (t *ReplaceE2MTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	e2m, _ := GetObjectParam(args, "e2m", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/events2metrics/" + id, Body: e2m})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// DeleteE2MTool deletes an events-to-metrics configuration.
type DeleteE2MTool struct{ *BaseTool }

// NewDeleteE2MTool creates a new tool instance
func NewDeleteE2MTool(c *client.Client, l *zap.Logger) *DeleteE2MTool {
	return &DeleteE2MTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *DeleteE2MTool) Name() string { return "delete_e2m" }

// Description returns the tool description
func (t *DeleteE2MTool) Description() string { return "Delete an events-to-metrics configuration" }

// InputSchema returns the input schema
func (t *DeleteE2MTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *DeleteE2MTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/events2metrics/" + id})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// ListDataAccessRulesTool lists all data access rules.
type ListDataAccessRulesTool struct{ *BaseTool }

// NewListDataAccessRulesTool creates a new tool instance
func NewListDataAccessRulesTool(c *client.Client, l *zap.Logger) *ListDataAccessRulesTool {
	return &ListDataAccessRulesTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ListDataAccessRulesTool) Name() string { return "list_data_access_rules" }

// Description returns the tool description
func (t *ListDataAccessRulesTool) Description() string {
	return `List all data access rules controlling log visibility.

**Related tools:** get_data_access_rule, create_data_access_rule, update_data_access_rule, delete_data_access_rule`
}

// InputSchema returns the input schema
func (t *ListDataAccessRulesTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}

// Execute executes the tool
func (t *ListDataAccessRulesTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/data_access_rules"})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// GetDataAccessRuleTool retrieves a specific data access rule by ID.
type GetDataAccessRuleTool struct{ *BaseTool }

// NewGetDataAccessRuleTool creates a new tool instance
func NewGetDataAccessRuleTool(c *client.Client, l *zap.Logger) *GetDataAccessRuleTool {
	return &GetDataAccessRuleTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *GetDataAccessRuleTool) Name() string { return "get_data_access_rule" }

// Description returns the tool description
func (t *GetDataAccessRuleTool) Description() string { return "Get a specific data access rule by ID" }

// InputSchema returns the input schema
func (t *GetDataAccessRuleTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string", "description": "The unique identifier of the data access rule"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *GetDataAccessRuleTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/data_access_rules/" + id})
	if err != nil {
		return HandleGetError(err, "Data access rule", id, "list_data_access_rules"), nil
	}
	return t.FormatResponse(res)
}

// CreateDataAccessRuleTool creates a new data access rule.
type CreateDataAccessRuleTool struct{ *BaseTool }

// NewCreateDataAccessRuleTool creates a new tool instance
func NewCreateDataAccessRuleTool(c *client.Client, l *zap.Logger) *CreateDataAccessRuleTool {
	return &CreateDataAccessRuleTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *CreateDataAccessRuleTool) Name() string { return "create_data_access_rule" }

// Description returns the tool description
func (t *CreateDataAccessRuleTool) Description() string {
	return `Create a data access rule to control which logs users can view based on filters.

**Related tools:** list_data_access_rules, get_data_access_rule, update_data_access_rule, delete_data_access_rule

**Use Cases:**
- Restrict access to sensitive logs by team
- Implement data isolation for multi-tenant environments
- Control access based on application or subsystem
- Enforce compliance with data visibility requirements`
}

// InputSchema returns the input schema
func (t *CreateDataAccessRuleTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"rule": map[string]interface{}{
				"type":        "object",
				"description": "Data access rule configuration object",
				"properties": map[string]interface{}{
					"display_name": map[string]interface{}{
						"type":        "string",
						"description": "Display name for the access rule",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Description of the access rule purpose",
					},
					"default_expression": map[string]interface{}{
						"type":        "string",
						"description": "Default filter expression applied to all users",
					},
					"filters": map[string]interface{}{
						"type":        "array",
						"description": "Array of filter configurations",
					},
				},
			},
		},
		"required": []string{"rule"},
		"examples": []interface{}{
			map[string]interface{}{
				"rule": map[string]interface{}{
					"display_name": "Production Team Access",
					"description":  "Restrict access to production logs only",
					"filters": []map[string]interface{}{
						{
							"entity_type": "logs",
							"expression":  "applicationName.startsWith('production')",
						},
					},
				},
			},
			map[string]interface{}{
				"rule": map[string]interface{}{
					"display_name":       "Non-PII Access",
					"description":        "Filter out logs containing PII data",
					"default_expression": "NOT subsystemName:'pii-service'",
				},
			},
		},
	}
}

// Execute executes the tool
func (t *CreateDataAccessRuleTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	rule, _ := GetObjectParam(args, "rule", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/data_access_rules", Body: rule})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// UpdateDataAccessRuleTool updates an existing data access rule.
type UpdateDataAccessRuleTool struct{ *BaseTool }

// NewUpdateDataAccessRuleTool creates a new tool instance
func NewUpdateDataAccessRuleTool(c *client.Client, l *zap.Logger) *UpdateDataAccessRuleTool {
	return &UpdateDataAccessRuleTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *UpdateDataAccessRuleTool) Name() string { return "update_data_access_rule" }

// Description returns the tool description
func (t *UpdateDataAccessRuleTool) Description() string { return "Update a data access rule" }

// InputSchema returns the input schema
func (t *UpdateDataAccessRuleTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "rule": map[string]interface{}{"type": "object"}}, "required": []string{"id", "rule"}}
}

// Execute executes the tool
func (t *UpdateDataAccessRuleTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	rule, _ := GetObjectParam(args, "rule", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/data_access_rules/" + id, Body: rule})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// DeleteDataAccessRuleTool deletes a data access rule.
type DeleteDataAccessRuleTool struct{ *BaseTool }

// NewDeleteDataAccessRuleTool creates a new tool instance
func NewDeleteDataAccessRuleTool(c *client.Client, l *zap.Logger) *DeleteDataAccessRuleTool {
	return &DeleteDataAccessRuleTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *DeleteDataAccessRuleTool) Name() string { return "delete_data_access_rule" }

// Description returns the tool description
func (t *DeleteDataAccessRuleTool) Description() string { return "Delete a data access rule" }

// InputSchema returns the input schema
func (t *DeleteDataAccessRuleTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *DeleteDataAccessRuleTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/data_access_rules/" + id})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// ListEnrichmentsTool lists all enrichments.
type ListEnrichmentsTool struct{ *BaseTool }

// NewListEnrichmentsTool creates a new tool instance
func NewListEnrichmentsTool(c *client.Client, l *zap.Logger) *ListEnrichmentsTool {
	return &ListEnrichmentsTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ListEnrichmentsTool) Name() string { return "list_enrichments" }

// Description returns the tool description
func (t *ListEnrichmentsTool) Description() string {
	return `List all data enrichments that add context to incoming logs.

**Related tools:** get_enrichments, create_enrichment, update_enrichment, delete_enrichment`
}

// InputSchema returns the input schema
func (t *ListEnrichmentsTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}

// Execute executes the tool
func (t *ListEnrichmentsTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/enrichments"})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// CreateEnrichmentTool creates a new enrichment.
type CreateEnrichmentTool struct{ *BaseTool }

// NewCreateEnrichmentTool creates a new tool instance
func NewCreateEnrichmentTool(c *client.Client, l *zap.Logger) *CreateEnrichmentTool {
	return &CreateEnrichmentTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *CreateEnrichmentTool) Name() string { return "create_enrichment" }

// Description returns the tool description
func (t *CreateEnrichmentTool) Description() string {
	return `Create a data enrichment to add context to incoming logs.

**Related tools:** list_enrichments, get_enrichments, update_enrichment, delete_enrichment

**Enrichment Types:**
- geo_ip: Add geographic information based on IP addresses
- custom_enrichment: Add custom fields from lookup tables

**Use Cases:**
- Add geographic location from IP addresses
- Enrich logs with user/customer metadata
- Add environment or deployment context
- Map error codes to descriptions`
}

// InputSchema returns the input schema
func (t *CreateEnrichmentTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"enrichment": map[string]interface{}{
				"type":        "object",
				"description": "Enrichment configuration object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Display name for the enrichment",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Description of what this enrichment adds",
					},
					"field_name": map[string]interface{}{
						"type":        "string",
						"description": "Source field to use for enrichment lookup",
					},
					"enrichment_type": map[string]interface{}{
						"type":        "string",
						"description": "Type of enrichment: geo_ip or custom_enrichment",
						"enum":        []string{"geo_ip", "custom_enrichment"},
					},
					"geo_ip_config": map[string]interface{}{
						"type":        "object",
						"description": "Configuration for geo_ip enrichment type",
					},
					"custom_enrichment_config": map[string]interface{}{
						"type":        "object",
						"description": "Configuration for custom_enrichment type",
					},
				},
			},
		},
		"required": []string{"enrichment"},
		"examples": []interface{}{
			map[string]interface{}{
				"enrichment": map[string]interface{}{
					"name":            "Client IP Geolocation",
					"description":     "Add geographic data from client IP addresses",
					"field_name":      "json.client_ip",
					"enrichment_type": "geo_ip",
				},
			},
			map[string]interface{}{
				"enrichment": map[string]interface{}{
					"name":            "Customer Tier Lookup",
					"description":     "Enrich logs with customer tier information",
					"field_name":      "json.customer_id",
					"enrichment_type": "custom_enrichment",
					"custom_enrichment_config": map[string]interface{}{
						"lookup_table_id": "customer-tiers-table",
					},
				},
			},
		},
	}
}

// Execute executes the tool
func (t *CreateEnrichmentTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	enr, _ := GetObjectParam(args, "enrichment", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/enrichments", Body: enr})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// UpdateEnrichmentTool updates an existing enrichment.
type UpdateEnrichmentTool struct{ *BaseTool }

// NewUpdateEnrichmentTool creates a new tool instance
func NewUpdateEnrichmentTool(c *client.Client, l *zap.Logger) *UpdateEnrichmentTool {
	return &UpdateEnrichmentTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *UpdateEnrichmentTool) Name() string { return "update_enrichment" }

// Description returns the tool description
func (t *UpdateEnrichmentTool) Description() string { return "Update an existing enrichment" }

// InputSchema returns the input schema
func (t *UpdateEnrichmentTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string", "description": "The unique identifier of the enrichment"}, "enrichment": map[string]interface{}{"type": "object"}}, "required": []string{"id", "enrichment"}}
}

// Execute executes the tool
func (t *UpdateEnrichmentTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	enr, _ := GetObjectParam(args, "enrichment", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/enrichments/" + id, Body: enr})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// DeleteEnrichmentTool deletes an enrichment.
type DeleteEnrichmentTool struct{ *BaseTool }

// NewDeleteEnrichmentTool creates a new tool instance
func NewDeleteEnrichmentTool(c *client.Client, l *zap.Logger) *DeleteEnrichmentTool {
	return &DeleteEnrichmentTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *DeleteEnrichmentTool) Name() string { return "delete_enrichment" }

// Description returns the tool description
func (t *DeleteEnrichmentTool) Description() string { return "Delete an enrichment" }

// InputSchema returns the input schema
func (t *DeleteEnrichmentTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *DeleteEnrichmentTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/enrichments/" + id})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// GetEnrichmentsTool retrieves all enrichments (alias for list_enrichments).
type GetEnrichmentsTool struct{ *BaseTool }

// NewGetEnrichmentsTool creates a new tool instance
func NewGetEnrichmentsTool(c *client.Client, l *zap.Logger) *GetEnrichmentsTool {
	return &GetEnrichmentsTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *GetEnrichmentsTool) Name() string { return "get_enrichments" }

// Description returns the tool description
func (t *GetEnrichmentsTool) Description() string {
	return "Get all enrichments (alias for list_enrichments)"
}

// InputSchema returns the input schema
func (t *GetEnrichmentsTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}

// Execute executes the tool
func (t *GetEnrichmentsTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/enrichments"})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// ListViewsTool lists all views.
type ListViewsTool struct{ *BaseTool }

// NewListViewsTool creates a new tool instance
func NewListViewsTool(c *client.Client, l *zap.Logger) *ListViewsTool {
	return &ListViewsTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ListViewsTool) Name() string { return "list_views" }

// Description returns the tool description
func (t *ListViewsTool) Description() string {
	return `List all saved views with their filter and query configurations.

**Related tools:** get_view, create_view, replace_view, delete_view, list_view_folders`
}

// InputSchema returns the input schema
func (t *ListViewsTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}

// Execute executes the tool
func (t *ListViewsTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/views"})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// CreateViewTool creates a new view.
type CreateViewTool struct{ *BaseTool }

// NewCreateViewTool creates a new tool instance
func NewCreateViewTool(c *client.Client, l *zap.Logger) *CreateViewTool {
	return &CreateViewTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *CreateViewTool) Name() string { return "create_view" }

// Description returns the tool description
func (t *CreateViewTool) Description() string {
	return `Create a saved view with predefined filters and query settings.

**Related tools:** list_views, get_view, replace_view, delete_view, list_view_folders, create_view_folder

**Use Cases:**
- Save commonly used log queries for quick access
- Create team-specific views for different services
- Set up debugging views with specific filters
- Share standardized views across the organization`
}

// InputSchema returns the input schema
func (t *CreateViewTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"view": map[string]interface{}{
				"type":        "object",
				"description": "View configuration object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Display name for the view",
					},
					"search_query": map[string]interface{}{
						"type":        "object",
						"description": "Query configuration for the view",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "string",
								"description": "Lucene or DataPrime query string",
							},
						},
					},
					"time_selection": map[string]interface{}{
						"type":        "object",
						"description": "Time range selection for the view",
					},
					"filters": map[string]interface{}{
						"type":        "object",
						"description": "Filter configuration",
					},
					"folder_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of folder to place the view in (optional)",
					},
				},
			},
		},
		"required": []string{"view"},
		"examples": []interface{}{
			map[string]interface{}{
				"view": map[string]interface{}{
					"name": "Production Errors",
					"search_query": map[string]interface{}{
						"query": "application:production AND level:error",
					},
					"time_selection": map[string]interface{}{
						"quick_selection": map[string]interface{}{
							"seconds": 3600,
						},
					},
				},
			},
			map[string]interface{}{
				"view": map[string]interface{}{
					"name": "API Gateway Logs",
					"search_query": map[string]interface{}{
						"query": "subsystem:api-gateway",
					},
					"filters": map[string]interface{}{
						"severity_filters": []string{"warning", "error", "critical"},
					},
				},
			},
		},
	}
}

// Execute executes the tool
func (t *CreateViewTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	view, _ := GetObjectParam(args, "view", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/views", Body: view})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// GetViewTool retrieves a specific view by ID.
type GetViewTool struct{ *BaseTool }

// NewGetViewTool creates a new tool instance
func NewGetViewTool(c *client.Client, l *zap.Logger) *GetViewTool {
	return &GetViewTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *GetViewTool) Name() string { return "get_view" }

// Description returns the tool description
func (t *GetViewTool) Description() string { return "Get a view by ID" }

// InputSchema returns the input schema
func (t *GetViewTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *GetViewTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/views/" + id})
	if err != nil {
		return HandleGetError(err, "View", id, "list_views"), nil
	}
	return t.FormatResponse(res)
}

// ReplaceViewTool replaces a view.
type ReplaceViewTool struct{ *BaseTool }

// NewReplaceViewTool creates a new tool instance
func NewReplaceViewTool(c *client.Client, l *zap.Logger) *ReplaceViewTool {
	return &ReplaceViewTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ReplaceViewTool) Name() string { return "replace_view" }

// Description returns the tool description
func (t *ReplaceViewTool) Description() string { return "Replace a view" }

// InputSchema returns the input schema
func (t *ReplaceViewTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "view": map[string]interface{}{"type": "object"}}, "required": []string{"id", "view"}}
}

// Execute executes the tool
func (t *ReplaceViewTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	view, _ := GetObjectParam(args, "view", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/views/" + id, Body: view})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// DeleteViewTool deletes a view.
type DeleteViewTool struct{ *BaseTool }

// NewDeleteViewTool creates a new tool instance
func NewDeleteViewTool(c *client.Client, l *zap.Logger) *DeleteViewTool {
	return &DeleteViewTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *DeleteViewTool) Name() string { return "delete_view" }

// Description returns the tool description
func (t *DeleteViewTool) Description() string { return "Delete a view" }

// InputSchema returns the input schema
func (t *DeleteViewTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *DeleteViewTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/views/" + id})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// ListViewFoldersTool lists all view folders.
type ListViewFoldersTool struct{ *BaseTool }

// NewListViewFoldersTool creates a new tool instance
func NewListViewFoldersTool(c *client.Client, l *zap.Logger) *ListViewFoldersTool {
	return &ListViewFoldersTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ListViewFoldersTool) Name() string { return "list_view_folders" }

// Description returns the tool description
func (t *ListViewFoldersTool) Description() string { return "List all view folders" }

// InputSchema returns the input schema
func (t *ListViewFoldersTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}

// Execute executes the tool
func (t *ListViewFoldersTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/view_folders"})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// CreateViewFolderTool creates a new view folder.
type CreateViewFolderTool struct{ *BaseTool }

// NewCreateViewFolderTool creates a new tool instance
func NewCreateViewFolderTool(c *client.Client, l *zap.Logger) *CreateViewFolderTool {
	return &CreateViewFolderTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *CreateViewFolderTool) Name() string { return "create_view_folder" }

// Description returns the tool description
func (t *CreateViewFolderTool) Description() string { return "Create a new view folder" }

// InputSchema returns the input schema
func (t *CreateViewFolderTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"folder": map[string]interface{}{"type": "object"}}, "required": []string{"folder"}}
}

// Execute executes the tool
func (t *CreateViewFolderTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	folder, _ := GetObjectParam(args, "folder", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/view_folders", Body: folder})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// GetViewFolderTool retrieves a specific view folder by ID.
type GetViewFolderTool struct{ *BaseTool }

// NewGetViewFolderTool creates a new tool instance
func NewGetViewFolderTool(c *client.Client, l *zap.Logger) *GetViewFolderTool {
	return &GetViewFolderTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *GetViewFolderTool) Name() string { return "get_view_folder" }

// Description returns the tool description
func (t *GetViewFolderTool) Description() string { return "Get a view folder by ID" }

// InputSchema returns the input schema
func (t *GetViewFolderTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *GetViewFolderTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/view_folders/" + id})
	if err != nil {
		return HandleGetError(err, "View folder", id, "list_view_folders"), nil
	}
	return t.FormatResponse(res)
}

// ReplaceViewFolderTool replaces a view folder.
type ReplaceViewFolderTool struct{ *BaseTool }

// NewReplaceViewFolderTool creates a new tool instance
func NewReplaceViewFolderTool(c *client.Client, l *zap.Logger) *ReplaceViewFolderTool {
	return &ReplaceViewFolderTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *ReplaceViewFolderTool) Name() string { return "replace_view_folder" }

// Description returns the tool description
func (t *ReplaceViewFolderTool) Description() string { return "Replace a view folder" }

// InputSchema returns the input schema
func (t *ReplaceViewFolderTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "folder": map[string]interface{}{"type": "object"}}, "required": []string{"id", "folder"}}
}

// Execute executes the tool
func (t *ReplaceViewFolderTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	folder, _ := GetObjectParam(args, "folder", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/view_folders/" + id, Body: folder})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// DeleteViewFolderTool deletes a view folder.
type DeleteViewFolderTool struct{ *BaseTool }

// NewDeleteViewFolderTool creates a new tool instance
func NewDeleteViewFolderTool(c *client.Client, l *zap.Logger) *DeleteViewFolderTool {
	return &DeleteViewFolderTool{NewBaseTool(c, l)}
}

// Name returns the tool name
func (t *DeleteViewFolderTool) Name() string { return "delete_view_folder" }

// Description returns the tool description
func (t *DeleteViewFolderTool) Description() string { return "Delete a view folder" }

// InputSchema returns the input schema
func (t *DeleteViewFolderTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, "required": []string{"id"}}
}

// Execute executes the tool
func (t *DeleteViewFolderTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/view_folders/" + id})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}
