package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// GetAlertTool retrieves a specific alert by ID
type GetAlertTool struct {
	*BaseTool
}

// NewGetAlertTool creates a new tool instance
func NewGetAlertTool(client *client.Client, logger *zap.Logger) *GetAlertTool {
	return &GetAlertTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *GetAlertTool) Name() string {
	return "get_alert"
}

// Description returns the tool description
func (t *GetAlertTool) Description() string {
	return "Retrieve a specific alert by its ID from IBM Cloud Logs"
}

// InputSchema returns the input schema
func (t *GetAlertTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the alert",
			},
		},
		"required": []string{"id"},
	}
}

// Execute executes the tool
func (t *GetAlertTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "GET",
		Path:   "/v1/alerts/" + id,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// ListAlertsTool lists all alerts
type ListAlertsTool struct {
	*BaseTool
}

// NewListAlertsTool creates a new tool instance
func NewListAlertsTool(client *client.Client, logger *zap.Logger) *ListAlertsTool {
	return &ListAlertsTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *ListAlertsTool) Name() string {
	return "list_alerts"
}

// Description returns the tool description
func (t *ListAlertsTool) Description() string {
	return "List all alerts in IBM Cloud Logs"
}

// InputSchema returns the input schema
func (t *ListAlertsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results (default: 50, max: 100)",
			},
			"cursor": map[string]interface{}{
				"type":        "string",
				"description": "Pagination cursor from previous response",
			},
		},
	}
}

// Execute executes the tool
func (t *ListAlertsTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Get pagination parameters
	pagination, err := GetPaginationParams(arguments)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Build query with pagination
	query := make(map[string]string)
	AddPaginationToQuery(query, pagination)

	req := &client.Request{
		Method: "GET",
		Path:   "/v1/alerts",
		Query:  query,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// CreateAlertTool creates a new alert
type CreateAlertTool struct {
	*BaseTool
}

// NewCreateAlertTool creates a new tool instance
func NewCreateAlertTool(client *client.Client, logger *zap.Logger) *CreateAlertTool {
	return &CreateAlertTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *CreateAlertTool) Name() string {
	return "create_alert"
}

// Description returns the tool description
func (t *CreateAlertTool) Description() string {
	return "Create a new alert in IBM Cloud Logs"
}

// InputSchema returns the input schema
func (t *CreateAlertTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"alert": map[string]interface{}{
				"type":        "object",
				"description": "The alert configuration object",
			},
		},
		"required": []string{"alert"},
	}
}

// Execute executes the tool
func (t *CreateAlertTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	alert, err := GetObjectParam(arguments, "alert", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "POST",
		Path:   "/v1/alerts",
		Body:   alert,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// UpdateAlertTool updates an existing alert
type UpdateAlertTool struct {
	*BaseTool
}

// NewUpdateAlertTool creates a new tool instance
func NewUpdateAlertTool(client *client.Client, logger *zap.Logger) *UpdateAlertTool {
	return &UpdateAlertTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *UpdateAlertTool) Name() string {
	return "update_alert"
}

// Description returns the tool description
func (t *UpdateAlertTool) Description() string {
	return "Update an existing alert in IBM Cloud Logs"
}

// InputSchema returns the input schema
func (t *UpdateAlertTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the alert to update",
			},
			"alert": map[string]interface{}{
				"type":        "object",
				"description": "The updated alert configuration",
			},
		},
		"required": []string{"id", "alert"},
	}
}

// Execute executes the tool
func (t *UpdateAlertTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	alert, err := GetObjectParam(arguments, "alert", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "PUT",
		Path:   "/v1/alerts/" + id,
		Body:   alert,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// DeleteAlertTool deletes an alert
type DeleteAlertTool struct {
	*BaseTool
}

// NewDeleteAlertTool creates a new tool instance
func NewDeleteAlertTool(client *client.Client, logger *zap.Logger) *DeleteAlertTool {
	return &DeleteAlertTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *DeleteAlertTool) Name() string {
	return "delete_alert"
}

// Description returns the tool description
func (t *DeleteAlertTool) Description() string {
	return "Delete an alert from IBM Cloud Logs"
}

// InputSchema returns the input schema
func (t *DeleteAlertTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the alert to delete",
			},
		},
		"required": []string{"id"},
	}
}

// Execute executes the tool
func (t *DeleteAlertTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	id, err := GetStringParam(arguments, "id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "DELETE",
		Path:   "/v1/alerts/" + id,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}
