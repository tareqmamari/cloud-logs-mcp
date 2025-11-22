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

func NewGetAlertDefinitionTool(client *client.Client, logger *zap.Logger) *GetAlertDefinitionTool {
	return &GetAlertDefinitionTool{BaseTool: NewBaseTool(client, logger)}
}

func (t *GetAlertDefinitionTool) Name() string { return "get_alert_definition" }

func (t *GetAlertDefinitionTool) Description() string {
	return "Retrieve a specific alert definition by its ID"
}

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

func (t *GetAlertDefinitionTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/alert_definitions/" + id})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(result)
}

// ListAlertDefinitionsTool lists all alert definitions
type ListAlertDefinitionsTool struct {
	*BaseTool
}

func NewListAlertDefinitionsTool(client *client.Client, logger *zap.Logger) *ListAlertDefinitionsTool {
	return &ListAlertDefinitionsTool{BaseTool: NewBaseTool(client, logger)}
}

func (t *ListAlertDefinitionsTool) Name() string { return "list_alert_definitions" }

func (t *ListAlertDefinitionsTool) Description() string {
	return "List all alert definitions"
}

func (t *ListAlertDefinitionsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *ListAlertDefinitionsTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "GET", Path: "/v1/alert_definitions"})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(result)
}

// CreateAlertDefinitionTool creates a new alert definition
type CreateAlertDefinitionTool struct {
	*BaseTool
}

func NewCreateAlertDefinitionTool(client *client.Client, logger *zap.Logger) *CreateAlertDefinitionTool {
	return &CreateAlertDefinitionTool{BaseTool: NewBaseTool(client, logger)}
}

func (t *CreateAlertDefinitionTool) Name() string { return "create_alert_definition" }

func (t *CreateAlertDefinitionTool) Description() string {
	return "Create a new alert definition"
}

func (t *CreateAlertDefinitionTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"definition": map[string]interface{}{
				"type": "object", "description": "Alert definition configuration",
			},
		},
		"required": []string{"definition"},
	}
}

func (t *CreateAlertDefinitionTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	def, err := GetObjectParam(arguments, "definition", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "POST", Path: "/v1/alert_definitions", Body: def})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(result)
}

// UpdateAlertDefinitionTool updates an existing alert definition
type UpdateAlertDefinitionTool struct {
	*BaseTool
}

func NewUpdateAlertDefinitionTool(client *client.Client, logger *zap.Logger) *UpdateAlertDefinitionTool {
	return &UpdateAlertDefinitionTool{BaseTool: NewBaseTool(client, logger)}
}

func (t *UpdateAlertDefinitionTool) Name() string { return "update_alert_definition" }

func (t *UpdateAlertDefinitionTool) Description() string {
	return "Update an existing alert definition"
}

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
	return t.FormatResponse(result)
}

// DeleteAlertDefinitionTool deletes an alert definition
type DeleteAlertDefinitionTool struct {
	*BaseTool
}

func NewDeleteAlertDefinitionTool(client *client.Client, logger *zap.Logger) *DeleteAlertDefinitionTool {
	return &DeleteAlertDefinitionTool{BaseTool: NewBaseTool(client, logger)}
}

func (t *DeleteAlertDefinitionTool) Name() string { return "delete_alert_definition" }

func (t *DeleteAlertDefinitionTool) Description() string {
	return "Delete an alert definition"
}

func (t *DeleteAlertDefinitionTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{"type": "string", "description": "Alert definition ID"},
		},
		"required": []string{"id"},
	}
}

func (t *DeleteAlertDefinitionTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	result, err := t.ExecuteRequest(ctx, &client.Request{Method: "DELETE", Path: "/v1/alert_definitions/" + id})
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	return t.FormatResponse(result)
}
