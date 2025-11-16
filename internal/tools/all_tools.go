package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// This file contains all remaining tools in a condensed format for brevity

// Rule Groups
type GetRuleGroupTool struct{ *BaseTool }

func NewGetRuleGroupTool(c *client.Client, l *zap.Logger) *GetRuleGroupTool {
	return &GetRuleGroupTool{NewBaseTool(c, l)}
}
func (t *GetRuleGroupTool) Name() string        { return "get_rule_group" }
func (t *GetRuleGroupTool) Description() string { return "Get a rule group by ID" }
func (t *GetRuleGroupTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string", "description": "Rule group ID"}}, Required: []string{"id"}}
}
func (t *GetRuleGroupTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/rule_groups/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type ListRuleGroupsTool struct{ *BaseTool }

func NewListRuleGroupsTool(c *client.Client, l *zap.Logger) *ListRuleGroupsTool {
	return &ListRuleGroupsTool{NewBaseTool(c, l)}
}
func (t *ListRuleGroupsTool) Name() string        { return "list_rule_groups" }
func (t *ListRuleGroupsTool) Description() string { return "List all rule groups" }
func (t *ListRuleGroupsTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}}
}
func (t *ListRuleGroupsTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/rule_groups"})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type CreateRuleGroupTool struct{ *BaseTool }

func NewCreateRuleGroupTool(c *client.Client, l *zap.Logger) *CreateRuleGroupTool {
	return &CreateRuleGroupTool{NewBaseTool(c, l)}
}
func (t *CreateRuleGroupTool) Name() string        { return "create_rule_group" }
func (t *CreateRuleGroupTool) Description() string { return "Create a new rule group" }
func (t *CreateRuleGroupTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"rule_group": map[string]interface{}{"type": "object"}}, Required: []string{"rule_group"}}
}
func (t *CreateRuleGroupTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	rg, _ := GetObjectParam(args, "rule_group", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/rule_groups", Body: rg})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type UpdateRuleGroupTool struct{ *BaseTool }

func NewUpdateRuleGroupTool(c *client.Client, l *zap.Logger) *UpdateRuleGroupTool {
	return &UpdateRuleGroupTool{NewBaseTool(c, l)}
}
func (t *UpdateRuleGroupTool) Name() string        { return "update_rule_group" }
func (t *UpdateRuleGroupTool) Description() string { return "Update a rule group" }
func (t *UpdateRuleGroupTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "rule_group": map[string]interface{}{"type": "object"}}, Required: []string{"id", "rule_group"}}
}
func (t *UpdateRuleGroupTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	rg, _ := GetObjectParam(args, "rule_group", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/rule_groups/" + id, Body: rg})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type DeleteRuleGroupTool struct{ *BaseTool }

func NewDeleteRuleGroupTool(c *client.Client, l *zap.Logger) *DeleteRuleGroupTool {
	return &DeleteRuleGroupTool{NewBaseTool(c, l)}
}
func (t *DeleteRuleGroupTool) Name() string        { return "delete_rule_group" }
func (t *DeleteRuleGroupTool) Description() string { return "Delete a rule group" }
func (t *DeleteRuleGroupTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, Required: []string{"id"}}
}
func (t *DeleteRuleGroupTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/rule_groups/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// Outgoing Webhooks
type GetOutgoingWebhookTool struct{ *BaseTool }

func NewGetOutgoingWebhookTool(c *client.Client, l *zap.Logger) *GetOutgoingWebhookTool {
	return &GetOutgoingWebhookTool{NewBaseTool(c, l)}
}
func (t *GetOutgoingWebhookTool) Name() string        { return "get_outgoing_webhook" }
func (t *GetOutgoingWebhookTool) Description() string { return "Get an outgoing webhook by ID" }
func (t *GetOutgoingWebhookTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, Required: []string{"id"}}
}
func (t *GetOutgoingWebhookTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/outgoing_webhooks/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type ListOutgoingWebhooksTool struct{ *BaseTool }

func NewListOutgoingWebhooksTool(c *client.Client, l *zap.Logger) *ListOutgoingWebhooksTool {
	return &ListOutgoingWebhooksTool{NewBaseTool(c, l)}
}
func (t *ListOutgoingWebhooksTool) Name() string        { return "list_outgoing_webhooks" }
func (t *ListOutgoingWebhooksTool) Description() string { return "List all outgoing webhooks" }
func (t *ListOutgoingWebhooksTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}}
}
func (t *ListOutgoingWebhooksTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/outgoing_webhooks"})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type CreateOutgoingWebhookTool struct{ *BaseTool }

func NewCreateOutgoingWebhookTool(c *client.Client, l *zap.Logger) *CreateOutgoingWebhookTool {
	return &CreateOutgoingWebhookTool{NewBaseTool(c, l)}
}
func (t *CreateOutgoingWebhookTool) Name() string        { return "create_outgoing_webhook" }
func (t *CreateOutgoingWebhookTool) Description() string { return "Create a new outgoing webhook" }
func (t *CreateOutgoingWebhookTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"webhook": map[string]interface{}{"type": "object"}}, Required: []string{"webhook"}}
}
func (t *CreateOutgoingWebhookTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	wh, _ := GetObjectParam(args, "webhook", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/outgoing_webhooks", Body: wh})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type UpdateOutgoingWebhookTool struct{ *BaseTool }

func NewUpdateOutgoingWebhookTool(c *client.Client, l *zap.Logger) *UpdateOutgoingWebhookTool {
	return &UpdateOutgoingWebhookTool{NewBaseTool(c, l)}
}
func (t *UpdateOutgoingWebhookTool) Name() string        { return "update_outgoing_webhook" }
func (t *UpdateOutgoingWebhookTool) Description() string { return "Update an outgoing webhook" }
func (t *UpdateOutgoingWebhookTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "webhook": map[string]interface{}{"type": "object"}}, Required: []string{"id", "webhook"}}
}
func (t *UpdateOutgoingWebhookTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	wh, _ := GetObjectParam(args, "webhook", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/outgoing_webhooks/" + id, Body: wh})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type DeleteOutgoingWebhookTool struct{ *BaseTool }

func NewDeleteOutgoingWebhookTool(c *client.Client, l *zap.Logger) *DeleteOutgoingWebhookTool {
	return &DeleteOutgoingWebhookTool{NewBaseTool(c, l)}
}
func (t *DeleteOutgoingWebhookTool) Name() string        { return "delete_outgoing_webhook" }
func (t *DeleteOutgoingWebhookTool) Description() string { return "Delete an outgoing webhook" }
func (t *DeleteOutgoingWebhookTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, Required: []string{"id"}}
}
func (t *DeleteOutgoingWebhookTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/outgoing_webhooks/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// Policies
type GetPolicyTool struct{ *BaseTool }

func NewGetPolicyTool(c *client.Client, l *zap.Logger) *GetPolicyTool {
	return &GetPolicyTool{NewBaseTool(c, l)}
}
func (t *GetPolicyTool) Name() string        { return "get_policy" }
func (t *GetPolicyTool) Description() string { return "Get a policy by ID" }
func (t *GetPolicyTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, Required: []string{"id"}}
}
func (t *GetPolicyTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/policies/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type ListPoliciesTool struct{ *BaseTool }

func NewListPoliciesTool(c *client.Client, l *zap.Logger) *ListPoliciesTool {
	return &ListPoliciesTool{NewBaseTool(c, l)}
}
func (t *ListPoliciesTool) Name() string        { return "list_policies" }
func (t *ListPoliciesTool) Description() string { return "List all policies" }
func (t *ListPoliciesTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}}
}
func (t *ListPoliciesTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/policies"})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type CreatePolicyTool struct{ *BaseTool }

func NewCreatePolicyTool(c *client.Client, l *zap.Logger) *CreatePolicyTool {
	return &CreatePolicyTool{NewBaseTool(c, l)}
}
func (t *CreatePolicyTool) Name() string        { return "create_policy" }
func (t *CreatePolicyTool) Description() string { return "Create a new policy" }
func (t *CreatePolicyTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"policy": map[string]interface{}{"type": "object"}}, Required: []string{"policy"}}
}
func (t *CreatePolicyTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	pol, _ := GetObjectParam(args, "policy", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/policies", Body: pol})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type UpdatePolicyTool struct{ *BaseTool }

func NewUpdatePolicyTool(c *client.Client, l *zap.Logger) *UpdatePolicyTool {
	return &UpdatePolicyTool{NewBaseTool(c, l)}
}
func (t *UpdatePolicyTool) Name() string        { return "update_policy" }
func (t *UpdatePolicyTool) Description() string { return "Update a policy" }
func (t *UpdatePolicyTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "policy": map[string]interface{}{"type": "object"}}, Required: []string{"id", "policy"}}
}
func (t *UpdatePolicyTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	pol, _ := GetObjectParam(args, "policy", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/policies/" + id, Body: pol})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type DeletePolicyTool struct{ *BaseTool }

func NewDeletePolicyTool(c *client.Client, l *zap.Logger) *DeletePolicyTool {
	return &DeletePolicyTool{NewBaseTool(c, l)}
}
func (t *DeletePolicyTool) Name() string        { return "delete_policy" }
func (t *DeletePolicyTool) Description() string { return "Delete a policy" }
func (t *DeletePolicyTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, Required: []string{"id"}}
}
func (t *DeletePolicyTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/policies/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// E2M (Events to Metrics)
type GetE2MTool struct{ *BaseTool }

func NewGetE2MTool(c *client.Client, l *zap.Logger) *GetE2MTool {
	return &GetE2MTool{NewBaseTool(c, l)}
}
func (t *GetE2MTool) Name() string        { return "get_e2m" }
func (t *GetE2MTool) Description() string { return "Get an events-to-metrics configuration by ID" }
func (t *GetE2MTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, Required: []string{"id"}}
}
func (t *GetE2MTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/events2metrics/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type ListE2MTool struct{ *BaseTool }

func NewListE2MTool(c *client.Client, l *zap.Logger) *ListE2MTool {
	return &ListE2MTool{NewBaseTool(c, l)}
}
func (t *ListE2MTool) Name() string        { return "list_e2m" }
func (t *ListE2MTool) Description() string { return "List all events-to-metrics configurations" }
func (t *ListE2MTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}}
}
func (t *ListE2MTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/events2metrics"})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type CreateE2MTool struct{ *BaseTool }

func NewCreateE2MTool(c *client.Client, l *zap.Logger) *CreateE2MTool {
	return &CreateE2MTool{NewBaseTool(c, l)}
}
func (t *CreateE2MTool) Name() string        { return "create_e2m" }
func (t *CreateE2MTool) Description() string { return "Create a new events-to-metrics configuration" }
func (t *CreateE2MTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"e2m": map[string]interface{}{"type": "object"}}, Required: []string{"e2m"}}
}
func (t *CreateE2MTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	e2m, _ := GetObjectParam(args, "e2m", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/events2metrics", Body: e2m})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type ReplaceE2MTool struct{ *BaseTool }

func NewReplaceE2MTool(c *client.Client, l *zap.Logger) *ReplaceE2MTool {
	return &ReplaceE2MTool{NewBaseTool(c, l)}
}
func (t *ReplaceE2MTool) Name() string        { return "replace_e2m" }
func (t *ReplaceE2MTool) Description() string { return "Replace an events-to-metrics configuration" }
func (t *ReplaceE2MTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "e2m": map[string]interface{}{"type": "object"}}, Required: []string{"id", "e2m"}}
}
func (t *ReplaceE2MTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	e2m, _ := GetObjectParam(args, "e2m", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/events2metrics/" + id, Body: e2m})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type DeleteE2MTool struct{ *BaseTool }

func NewDeleteE2MTool(c *client.Client, l *zap.Logger) *DeleteE2MTool {
	return &DeleteE2MTool{NewBaseTool(c, l)}
}
func (t *DeleteE2MTool) Name() string        { return "delete_e2m" }
func (t *DeleteE2MTool) Description() string { return "Delete an events-to-metrics configuration" }
func (t *DeleteE2MTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, Required: []string{"id"}}
}
func (t *DeleteE2MTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/events2metrics/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// Data Access Rules
type ListDataAccessRulesTool struct{ *BaseTool }

func NewListDataAccessRulesTool(c *client.Client, l *zap.Logger) *ListDataAccessRulesTool {
	return &ListDataAccessRulesTool{NewBaseTool(c, l)}
}
func (t *ListDataAccessRulesTool) Name() string        { return "list_data_access_rules" }
func (t *ListDataAccessRulesTool) Description() string { return "List all data access rules" }
func (t *ListDataAccessRulesTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}}
}
func (t *ListDataAccessRulesTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/data_access_rules"})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type GetDataAccessRuleTool struct{ *BaseTool }

func NewGetDataAccessRuleTool(c *client.Client, l *zap.Logger) *GetDataAccessRuleTool {
	return &GetDataAccessRuleTool{NewBaseTool(c, l)}
}
func (t *GetDataAccessRuleTool) Name() string        { return "get_data_access_rule" }
func (t *GetDataAccessRuleTool) Description() string { return "Get a specific data access rule by ID" }
func (t *GetDataAccessRuleTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string", "description": "The unique identifier of the data access rule"}}, Required: []string{"id"}}
}
func (t *GetDataAccessRuleTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/data_access_rules/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type CreateDataAccessRuleTool struct{ *BaseTool }

func NewCreateDataAccessRuleTool(c *client.Client, l *zap.Logger) *CreateDataAccessRuleTool {
	return &CreateDataAccessRuleTool{NewBaseTool(c, l)}
}
func (t *CreateDataAccessRuleTool) Name() string        { return "create_data_access_rule" }
func (t *CreateDataAccessRuleTool) Description() string { return "Create a new data access rule" }
func (t *CreateDataAccessRuleTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"rule": map[string]interface{}{"type": "object"}}, Required: []string{"rule"}}
}
func (t *CreateDataAccessRuleTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	rule, _ := GetObjectParam(args, "rule", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/data_access_rules", Body: rule})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type UpdateDataAccessRuleTool struct{ *BaseTool }

func NewUpdateDataAccessRuleTool(c *client.Client, l *zap.Logger) *UpdateDataAccessRuleTool {
	return &UpdateDataAccessRuleTool{NewBaseTool(c, l)}
}
func (t *UpdateDataAccessRuleTool) Name() string        { return "update_data_access_rule" }
func (t *UpdateDataAccessRuleTool) Description() string { return "Update a data access rule" }
func (t *UpdateDataAccessRuleTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "rule": map[string]interface{}{"type": "object"}}, Required: []string{"id", "rule"}}
}
func (t *UpdateDataAccessRuleTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	rule, _ := GetObjectParam(args, "rule", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/data_access_rules/" + id, Body: rule})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type DeleteDataAccessRuleTool struct{ *BaseTool }

func NewDeleteDataAccessRuleTool(c *client.Client, l *zap.Logger) *DeleteDataAccessRuleTool {
	return &DeleteDataAccessRuleTool{NewBaseTool(c, l)}
}
func (t *DeleteDataAccessRuleTool) Name() string        { return "delete_data_access_rule" }
func (t *DeleteDataAccessRuleTool) Description() string { return "Delete a data access rule" }
func (t *DeleteDataAccessRuleTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, Required: []string{"id"}}
}
func (t *DeleteDataAccessRuleTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/data_access_rules/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// Enrichments
type ListEnrichmentsTool struct{ *BaseTool }

func NewListEnrichmentsTool(c *client.Client, l *zap.Logger) *ListEnrichmentsTool {
	return &ListEnrichmentsTool{NewBaseTool(c, l)}
}
func (t *ListEnrichmentsTool) Name() string        { return "list_enrichments" }
func (t *ListEnrichmentsTool) Description() string { return "List all enrichments" }
func (t *ListEnrichmentsTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}}
}
func (t *ListEnrichmentsTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/enrichments"})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type CreateEnrichmentTool struct{ *BaseTool }

func NewCreateEnrichmentTool(c *client.Client, l *zap.Logger) *CreateEnrichmentTool {
	return &CreateEnrichmentTool{NewBaseTool(c, l)}
}
func (t *CreateEnrichmentTool) Name() string        { return "create_enrichment" }
func (t *CreateEnrichmentTool) Description() string { return "Create a new enrichment" }
func (t *CreateEnrichmentTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"enrichment": map[string]interface{}{"type": "object"}}, Required: []string{"enrichment"}}
}
func (t *CreateEnrichmentTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	enr, _ := GetObjectParam(args, "enrichment", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/enrichments", Body: enr})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type UpdateEnrichmentTool struct{ *BaseTool }

func NewUpdateEnrichmentTool(c *client.Client, l *zap.Logger) *UpdateEnrichmentTool {
	return &UpdateEnrichmentTool{NewBaseTool(c, l)}
}
func (t *UpdateEnrichmentTool) Name() string        { return "update_enrichment" }
func (t *UpdateEnrichmentTool) Description() string { return "Update an existing enrichment" }
func (t *UpdateEnrichmentTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string", "description": "The unique identifier of the enrichment"}, "enrichment": map[string]interface{}{"type": "object"}}, Required: []string{"id", "enrichment"}}
}
func (t *UpdateEnrichmentTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	enr, _ := GetObjectParam(args, "enrichment", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/enrichments/" + id, Body: enr})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type DeleteEnrichmentTool struct{ *BaseTool }

func NewDeleteEnrichmentTool(c *client.Client, l *zap.Logger) *DeleteEnrichmentTool {
	return &DeleteEnrichmentTool{NewBaseTool(c, l)}
}
func (t *DeleteEnrichmentTool) Name() string        { return "delete_enrichment" }
func (t *DeleteEnrichmentTool) Description() string { return "Delete an enrichment" }
func (t *DeleteEnrichmentTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, Required: []string{"id"}}
}
func (t *DeleteEnrichmentTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/enrichments/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type GetEnrichmentsTool struct{ *BaseTool }

func NewGetEnrichmentsTool(c *client.Client, l *zap.Logger) *GetEnrichmentsTool {
	return &GetEnrichmentsTool{NewBaseTool(c, l)}
}
func (t *GetEnrichmentsTool) Name() string { return "get_enrichments" }
func (t *GetEnrichmentsTool) Description() string {
	return "Get all enrichments (alias for list_enrichments)"
}
func (t *GetEnrichmentsTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}}
}
func (t *GetEnrichmentsTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/enrichments"})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// Views
type ListViewsTool struct{ *BaseTool }

func NewListViewsTool(c *client.Client, l *zap.Logger) *ListViewsTool {
	return &ListViewsTool{NewBaseTool(c, l)}
}
func (t *ListViewsTool) Name() string        { return "list_views" }
func (t *ListViewsTool) Description() string { return "List all views" }
func (t *ListViewsTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}}
}
func (t *ListViewsTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/views"})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type CreateViewTool struct{ *BaseTool }

func NewCreateViewTool(c *client.Client, l *zap.Logger) *CreateViewTool {
	return &CreateViewTool{NewBaseTool(c, l)}
}
func (t *CreateViewTool) Name() string        { return "create_view" }
func (t *CreateViewTool) Description() string { return "Create a new view" }
func (t *CreateViewTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"view": map[string]interface{}{"type": "object"}}, Required: []string{"view"}}
}
func (t *CreateViewTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	view, _ := GetObjectParam(args, "view", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/views", Body: view})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type GetViewTool struct{ *BaseTool }

func NewGetViewTool(c *client.Client, l *zap.Logger) *GetViewTool {
	return &GetViewTool{NewBaseTool(c, l)}
}
func (t *GetViewTool) Name() string        { return "get_view" }
func (t *GetViewTool) Description() string { return "Get a view by ID" }
func (t *GetViewTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, Required: []string{"id"}}
}
func (t *GetViewTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/views/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type ReplaceViewTool struct{ *BaseTool }

func NewReplaceViewTool(c *client.Client, l *zap.Logger) *ReplaceViewTool {
	return &ReplaceViewTool{NewBaseTool(c, l)}
}
func (t *ReplaceViewTool) Name() string        { return "replace_view" }
func (t *ReplaceViewTool) Description() string { return "Replace a view" }
func (t *ReplaceViewTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "view": map[string]interface{}{"type": "object"}}, Required: []string{"id", "view"}}
}
func (t *ReplaceViewTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	view, _ := GetObjectParam(args, "view", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/views/" + id, Body: view})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type DeleteViewTool struct{ *BaseTool }

func NewDeleteViewTool(c *client.Client, l *zap.Logger) *DeleteViewTool {
	return &DeleteViewTool{NewBaseTool(c, l)}
}
func (t *DeleteViewTool) Name() string        { return "delete_view" }
func (t *DeleteViewTool) Description() string { return "Delete a view" }
func (t *DeleteViewTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, Required: []string{"id"}}
}
func (t *DeleteViewTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/views/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

// View Folders
type ListViewFoldersTool struct{ *BaseTool }

func NewListViewFoldersTool(c *client.Client, l *zap.Logger) *ListViewFoldersTool {
	return &ListViewFoldersTool{NewBaseTool(c, l)}
}
func (t *ListViewFoldersTool) Name() string        { return "list_view_folders" }
func (t *ListViewFoldersTool) Description() string { return "List all view folders" }
func (t *ListViewFoldersTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}}
}
func (t *ListViewFoldersTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/view_folders"})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type CreateViewFolderTool struct{ *BaseTool }

func NewCreateViewFolderTool(c *client.Client, l *zap.Logger) *CreateViewFolderTool {
	return &CreateViewFolderTool{NewBaseTool(c, l)}
}
func (t *CreateViewFolderTool) Name() string        { return "create_view_folder" }
func (t *CreateViewFolderTool) Description() string { return "Create a new view folder" }
func (t *CreateViewFolderTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"folder": map[string]interface{}{"type": "object"}}, Required: []string{"folder"}}
}
func (t *CreateViewFolderTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	folder, _ := GetObjectParam(args, "folder", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/view_folders", Body: folder})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type GetViewFolderTool struct{ *BaseTool }

func NewGetViewFolderTool(c *client.Client, l *zap.Logger) *GetViewFolderTool {
	return &GetViewFolderTool{NewBaseTool(c, l)}
}
func (t *GetViewFolderTool) Name() string        { return "get_view_folder" }
func (t *GetViewFolderTool) Description() string { return "Get a view folder by ID" }
func (t *GetViewFolderTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, Required: []string{"id"}}
}
func (t *GetViewFolderTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/view_folders/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type ReplaceViewFolderTool struct{ *BaseTool }

func NewReplaceViewFolderTool(c *client.Client, l *zap.Logger) *ReplaceViewFolderTool {
	return &ReplaceViewFolderTool{NewBaseTool(c, l)}
}
func (t *ReplaceViewFolderTool) Name() string        { return "replace_view_folder" }
func (t *ReplaceViewFolderTool) Description() string { return "Replace a view folder" }
func (t *ReplaceViewFolderTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "folder": map[string]interface{}{"type": "object"}}, Required: []string{"id", "folder"}}
}
func (t *ReplaceViewFolderTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	folder, _ := GetObjectParam(args, "folder", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "PUT", Path: "/v1/view_folders/" + id, Body: folder})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}

type DeleteViewFolderTool struct{ *BaseTool }

func NewDeleteViewFolderTool(c *client.Client, l *zap.Logger) *DeleteViewFolderTool {
	return &DeleteViewFolderTool{NewBaseTool(c, l)}
}
func (t *DeleteViewFolderTool) Name() string        { return "delete_view_folder" }
func (t *DeleteViewFolderTool) Description() string { return "Delete a view folder" }
func (t *DeleteViewFolderTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]interface{}{"type": "string"}}, Required: []string{"id"}}
}
func (t *DeleteViewFolderTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	id, _ := GetStringParam(args, "id", true)
	res, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/view_folders/" + id})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(res)
}
