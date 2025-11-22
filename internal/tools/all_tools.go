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
		return NewToolResultError(err.Error()), nil
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
func (t *ListRuleGroupsTool) Description() string { return "List all rule groups" }

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
		return NewToolResultError(err.Error()), nil
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
func (t *ListOutgoingWebhooksTool) Description() string { return "List all outgoing webhooks" }

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
func (t *CreateOutgoingWebhookTool) Description() string { return "Create a new outgoing webhook" }

// InputSchema returns the input schema
func (t *CreateOutgoingWebhookTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"webhook": map[string]interface{}{"type": "object"}}, "required": []string{"webhook"}}
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
		return NewToolResultError(err.Error()), nil
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
func (t *ListPoliciesTool) Description() string { return "List all policies" }

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
func (t *CreatePolicyTool) Description() string { return "Create a new policy" }

// InputSchema returns the input schema
func (t *CreatePolicyTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"policy": map[string]interface{}{"type": "object"}}, "required": []string{"policy"}}
}

// Execute executes the tool
func (t *CreatePolicyTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	pol, _ := GetObjectParam(args, "policy", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/policies", Body: pol})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
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
		return NewToolResultError(err.Error()), nil
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
func (t *ListE2MTool) Description() string { return "List all events-to-metrics configurations" }

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
func (t *CreateE2MTool) Description() string { return "Create a new events-to-metrics configuration" }

// InputSchema returns the input schema
func (t *CreateE2MTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"e2m": map[string]interface{}{"type": "object"}}, "required": []string{"e2m"}}
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
func (t *ListDataAccessRulesTool) Description() string { return "List all data access rules" }

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
		return NewToolResultError(err.Error()), nil
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
func (t *CreateDataAccessRuleTool) Description() string { return "Create a new data access rule" }

// InputSchema returns the input schema
func (t *CreateDataAccessRuleTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"rule": map[string]interface{}{"type": "object"}}, "required": []string{"rule"}}
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
func (t *ListEnrichmentsTool) Description() string { return "List all enrichments" }

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
func (t *CreateEnrichmentTool) Description() string { return "Create a new enrichment" }

// InputSchema returns the input schema
func (t *CreateEnrichmentTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"enrichment": map[string]interface{}{"type": "object"}}, "required": []string{"enrichment"}}
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
func (t *ListViewsTool) Description() string { return "List all views" }

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
func (t *CreateViewTool) Description() string { return "Create a new view" }

// InputSchema returns the input schema
func (t *CreateViewTool) InputSchema() interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"view": map[string]interface{}{"type": "object"}}, "required": []string{"view"}}
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
		return NewToolResultError(err.Error()), nil
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
		return NewToolResultError(err.Error()), nil
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
