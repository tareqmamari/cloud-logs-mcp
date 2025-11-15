package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/observability-c/logs-mcp-server/internal/client"
	"go.uber.org/zap"
)

// ListDashboardsTool lists all dashboards in the catalog.
type ListDashboardsTool struct {
	*BaseTool
}

// NewListDashboardsTool creates a new ListDashboardsTool instance.
func NewListDashboardsTool(client *client.Client, logger *zap.Logger) *ListDashboardsTool {
	return &ListDashboardsTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *ListDashboardsTool) Name() string {
	return "list_dashboards"
}

// Description returns a human-readable description of the tool.
func (t *ListDashboardsTool) Description() string {
	return "List all dashboards in the IBM Cloud Logs dashboard catalog"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *ListDashboardsTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type:       "object",
		Properties: map[string]interface{}{},
		Required:   []string{},
	}
}

// Execute lists all dashboards.
func (t *ListDashboardsTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	req := &client.Request{
		Method: "GET",
		Path:   "/v1/dashboards",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// GetDashboardTool gets a specific dashboard by ID.
type GetDashboardTool struct {
	*BaseTool
}

// NewGetDashboardTool creates a new GetDashboardTool instance.
func NewGetDashboardTool(client *client.Client, logger *zap.Logger) *GetDashboardTool {
	return &GetDashboardTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *GetDashboardTool) Name() string {
	return "get_dashboard"
}

// Description returns a human-readable description of the tool.
func (t *GetDashboardTool) Description() string {
	return "Get a specific dashboard by ID from IBM Cloud Logs"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *GetDashboardTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"dashboard_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the dashboard",
			},
		},
		Required: []string{"dashboard_id"},
	}
}

// Execute gets a specific dashboard.
func (t *GetDashboardTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	dashboardID, ok := arguments["dashboard_id"].(string)
	if !ok || dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required and must be a string"), nil
	}

	req := &client.Request{
		Method: "GET",
		Path:   fmt.Sprintf("/v1/dashboards/%s", dashboardID),
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// CreateDashboardTool creates a new dashboard.
type CreateDashboardTool struct {
	*BaseTool
}

// NewCreateDashboardTool creates a new CreateDashboardTool instance.
func NewCreateDashboardTool(client *client.Client, logger *zap.Logger) *CreateDashboardTool {
	return &CreateDashboardTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *CreateDashboardTool) Name() string {
	return "create_dashboard"
}

// Description returns a human-readable description of the tool.
func (t *CreateDashboardTool) Description() string {
	return "Create a new dashboard in IBM Cloud Logs with widgets and layout configuration"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *CreateDashboardTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Display name for the dashboard",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Optional description of the dashboard purpose",
			},
			"layout": map[string]interface{}{
				"type":        "object",
				"description": "Dashboard layout configuration with sections and widgets",
			},
		},
		Required: []string{"name", "layout"},
	}
}

// Execute creates a new dashboard.
func (t *CreateDashboardTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	name, ok := arguments["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name is required and must be a string"), nil
	}

	layout, ok := arguments["layout"]
	if !ok {
		return mcp.NewToolResultError("layout is required"), nil
	}

	body := map[string]interface{}{
		"name":   name,
		"layout": layout,
	}

	if description, ok := arguments["description"].(string); ok && description != "" {
		body["description"] = description
	}

	req := &client.Request{
		Method: "POST",
		Path:   "/v1/dashboards",
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// UpdateDashboardTool updates an existing dashboard.
type UpdateDashboardTool struct{
	*BaseTool
}

// NewUpdateDashboardTool creates a new UpdateDashboardTool instance.
func NewUpdateDashboardTool(client *client.Client, logger *zap.Logger) *UpdateDashboardTool {
	return &UpdateDashboardTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *UpdateDashboardTool) Name() string {
	return "update_dashboard"
}

// Description returns a human-readable description of the tool.
func (t *UpdateDashboardTool) Description() string {
	return "Update an existing dashboard in IBM Cloud Logs (replaces the entire dashboard)"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *UpdateDashboardTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"dashboard_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the dashboard to update",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Display name for the dashboard",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Optional description of the dashboard purpose",
			},
			"layout": map[string]interface{}{
				"type":        "object",
				"description": "Dashboard layout configuration with sections and widgets",
			},
		},
		Required: []string{"dashboard_id", "name", "layout"},
	}
}

// Execute updates a dashboard.
func (t *UpdateDashboardTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	dashboardID, ok := arguments["dashboard_id"].(string)
	if !ok || dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required and must be a string"), nil
	}

	name, ok := arguments["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name is required and must be a string"), nil
	}

	layout, ok := arguments["layout"]
	if !ok {
		return mcp.NewToolResultError("layout is required"), nil
	}

	body := map[string]interface{}{
		"name":   name,
		"layout": layout,
	}

	if description, ok := arguments["description"].(string); ok && description != "" {
		body["description"] = description
	}

	req := &client.Request{
		Method: "PUT",
		Path:   fmt.Sprintf("/v1/dashboards/%s", dashboardID),
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// DeleteDashboardTool deletes a dashboard.
type DeleteDashboardTool struct {
	*BaseTool
}

// NewDeleteDashboardTool creates a new DeleteDashboardTool instance.
func NewDeleteDashboardTool(client *client.Client, logger *zap.Logger) *DeleteDashboardTool {
	return &DeleteDashboardTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *DeleteDashboardTool) Name() string {
	return "delete_dashboard"
}

// Description returns a human-readable description of the tool.
func (t *DeleteDashboardTool) Description() string {
	return "Delete a dashboard from IBM Cloud Logs"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *DeleteDashboardTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"dashboard_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the dashboard to delete",
			},
		},
		Required: []string{"dashboard_id"},
	}
}

// Execute deletes a dashboard.
func (t *DeleteDashboardTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	dashboardID, ok := arguments["dashboard_id"].(string)
	if !ok || dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required and must be a string"), nil
	}

	req := &client.Request{
		Method: "DELETE",
		Path:   fmt.Sprintf("/v1/dashboards/%s", dashboardID),
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}
