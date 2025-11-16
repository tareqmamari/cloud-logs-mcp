package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// ListDashboardFoldersTool lists all dashboard folders.
type ListDashboardFoldersTool struct {
	*BaseTool
}

// NewListDashboardFoldersTool creates a new ListDashboardFoldersTool instance.
func NewListDashboardFoldersTool(client *client.Client, logger *zap.Logger) *ListDashboardFoldersTool {
	return &ListDashboardFoldersTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *ListDashboardFoldersTool) Name() string {
	return "list_dashboard_folders"
}

// Description returns a human-readable description of the tool.
func (t *ListDashboardFoldersTool) Description() string {
	return "List all dashboard folders for organizing dashboards in IBM Cloud Logs"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *ListDashboardFoldersTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type:       "object",
		Properties: map[string]interface{}{},
		Required:   []string{},
	}
}

// Execute lists all dashboard folders.
func (t *ListDashboardFoldersTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	req := &client.Request{
		Method: "GET",
		Path:   "/v1/folders",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// GetDashboardFolderTool gets a specific dashboard folder by ID.
type GetDashboardFolderTool struct {
	*BaseTool
}

// NewGetDashboardFolderTool creates a new GetDashboardFolderTool instance.
func NewGetDashboardFolderTool(client *client.Client, logger *zap.Logger) *GetDashboardFolderTool {
	return &GetDashboardFolderTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *GetDashboardFolderTool) Name() string {
	return "get_dashboard_folder"
}

// Description returns a human-readable description of the tool.
func (t *GetDashboardFolderTool) Description() string {
	return "Get details of a specific dashboard folder by ID"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *GetDashboardFolderTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"folder_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the folder",
			},
		},
		Required: []string{"folder_id"},
	}
}

// Execute gets a specific dashboard folder.
func (t *GetDashboardFolderTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	folderID, ok := arguments["folder_id"].(string)
	if !ok || folderID == "" {
		return mcp.NewToolResultError("folder_id is required and must be a string"), nil
	}

	req := &client.Request{
		Method: "GET",
		Path:   "/v1/folders/" + folderID,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// CreateDashboardFolderTool creates a new dashboard folder.
type CreateDashboardFolderTool struct {
	*BaseTool
}

// NewCreateDashboardFolderTool creates a new CreateDashboardFolderTool instance.
func NewCreateDashboardFolderTool(client *client.Client, logger *zap.Logger) *CreateDashboardFolderTool {
	return &CreateDashboardFolderTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *CreateDashboardFolderTool) Name() string {
	return "create_dashboard_folder"
}

// Description returns a human-readable description of the tool.
func (t *CreateDashboardFolderTool) Description() string {
	return "Create a new dashboard folder for organizing dashboards"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *CreateDashboardFolderTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "The name of the folder",
			},
			"parent_id": map[string]interface{}{
				"type":        "string",
				"description": "Optional parent folder ID for nested folders",
			},
		},
		Required: []string{"name"},
	}
}

// Execute creates a new dashboard folder.
func (t *CreateDashboardFolderTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	name, ok := arguments["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name is required and must be a string"), nil
	}

	body := map[string]interface{}{
		"name": name,
	}

	if parentID, ok := arguments["parent_id"].(string); ok && parentID != "" {
		body["parent_id"] = parentID
	}

	req := &client.Request{
		Method: "POST",
		Path:   "/v1/folders",
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// UpdateDashboardFolderTool updates a dashboard folder.
type UpdateDashboardFolderTool struct {
	*BaseTool
}

// NewUpdateDashboardFolderTool creates a new UpdateDashboardFolderTool instance.
func NewUpdateDashboardFolderTool(client *client.Client, logger *zap.Logger) *UpdateDashboardFolderTool {
	return &UpdateDashboardFolderTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *UpdateDashboardFolderTool) Name() string {
	return "update_dashboard_folder"
}

// Description returns a human-readable description of the tool.
func (t *UpdateDashboardFolderTool) Description() string {
	return "Update an existing dashboard folder"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *UpdateDashboardFolderTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"folder_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the folder to update",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "The new name for the folder",
			},
			"parent_id": map[string]interface{}{
				"type":        "string",
				"description": "Optional new parent folder ID",
			},
		},
		Required: []string{"folder_id", "name"},
	}
}

// Execute updates a dashboard folder.
func (t *UpdateDashboardFolderTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	folderID, ok := arguments["folder_id"].(string)
	if !ok || folderID == "" {
		return mcp.NewToolResultError("folder_id is required and must be a string"), nil
	}

	name, ok := arguments["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name is required and must be a string"), nil
	}

	body := map[string]interface{}{
		"name": name,
	}

	if parentID, ok := arguments["parent_id"].(string); ok && parentID != "" {
		body["parent_id"] = parentID
	}

	req := &client.Request{
		Method: "PUT",
		Path:   "/v1/folders/" + folderID,
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// DeleteDashboardFolderTool deletes a dashboard folder.
type DeleteDashboardFolderTool struct {
	*BaseTool
}

// NewDeleteDashboardFolderTool creates a new DeleteDashboardFolderTool instance.
func NewDeleteDashboardFolderTool(client *client.Client, logger *zap.Logger) *DeleteDashboardFolderTool {
	return &DeleteDashboardFolderTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *DeleteDashboardFolderTool) Name() string {
	return "delete_dashboard_folder"
}

// Description returns a human-readable description of the tool.
func (t *DeleteDashboardFolderTool) Description() string {
	return "Delete a dashboard folder"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *DeleteDashboardFolderTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"folder_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the folder to delete",
			},
		},
		Required: []string{"folder_id"},
	}
}

// Execute deletes a dashboard folder.
func (t *DeleteDashboardFolderTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	folderID, ok := arguments["folder_id"].(string)
	if !ok || folderID == "" {
		return mcp.NewToolResultError("folder_id is required and must be a string"), nil
	}

	req := &client.Request{
		Method: "DELETE",
		Path:   "/v1/folders/" + folderID,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// MoveDashboardToFolderTool moves a dashboard to a specific folder.
type MoveDashboardToFolderTool struct {
	*BaseTool
}

// NewMoveDashboardToFolderTool creates a new MoveDashboardToFolderTool instance.
func NewMoveDashboardToFolderTool(client *client.Client, logger *zap.Logger) *MoveDashboardToFolderTool {
	return &MoveDashboardToFolderTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *MoveDashboardToFolderTool) Name() string {
	return "move_dashboard_to_folder"
}

// Description returns a human-readable description of the tool.
func (t *MoveDashboardToFolderTool) Description() string {
	return "Move a dashboard to a specific folder for better organization"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *MoveDashboardToFolderTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"dashboard_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the dashboard to move",
			},
			"folder_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the target folder",
			},
		},
		Required: []string{"dashboard_id", "folder_id"},
	}
}

// Execute moves a dashboard to a folder.
func (t *MoveDashboardToFolderTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	dashboardID, ok := arguments["dashboard_id"].(string)
	if !ok || dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required and must be a string"), nil
	}

	folderID, ok := arguments["folder_id"].(string)
	if !ok || folderID == "" {
		return mcp.NewToolResultError("folder_id is required and must be a string"), nil
	}

	req := &client.Request{
		Method: "PUT",
		Path:   fmt.Sprintf("/v1/dashboards/%s/folder/%s", dashboardID, folderID),
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// PinDashboardTool pins a dashboard for quick access.
type PinDashboardTool struct {
	*BaseTool
}

// NewPinDashboardTool creates a new PinDashboardTool instance.
func NewPinDashboardTool(client *client.Client, logger *zap.Logger) *PinDashboardTool {
	return &PinDashboardTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *PinDashboardTool) Name() string {
	return "pin_dashboard"
}

// Description returns a human-readable description of the tool.
func (t *PinDashboardTool) Description() string {
	return "Pin a dashboard for quick access in IBM Cloud Logs"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *PinDashboardTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"dashboard_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the dashboard to pin",
			},
		},
		Required: []string{"dashboard_id"},
	}
}

// Execute pins a dashboard.
func (t *PinDashboardTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	dashboardID, ok := arguments["dashboard_id"].(string)
	if !ok || dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required and must be a string"), nil
	}

	req := &client.Request{
		Method: "PUT",
		Path:   fmt.Sprintf("/v1/dashboards/%s/pinned", dashboardID),
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// UnpinDashboardTool unpins a dashboard.
type UnpinDashboardTool struct {
	*BaseTool
}

// NewUnpinDashboardTool creates a new UnpinDashboardTool instance.
func NewUnpinDashboardTool(client *client.Client, logger *zap.Logger) *UnpinDashboardTool {
	return &UnpinDashboardTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *UnpinDashboardTool) Name() string {
	return "unpin_dashboard"
}

// Description returns a human-readable description of the tool.
func (t *UnpinDashboardTool) Description() string {
	return "Unpin a dashboard in IBM Cloud Logs"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *UnpinDashboardTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"dashboard_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the dashboard to unpin",
			},
		},
		Required: []string{"dashboard_id"},
	}
}

// Execute unpins a dashboard.
func (t *UnpinDashboardTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	dashboardID, ok := arguments["dashboard_id"].(string)
	if !ok || dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required and must be a string"), nil
	}

	req := &client.Request{
		Method: "DELETE",
		Path:   fmt.Sprintf("/v1/dashboards/%s/pinned", dashboardID),
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// SetDefaultDashboardTool sets a dashboard as the default.
type SetDefaultDashboardTool struct {
	*BaseTool
}

// NewSetDefaultDashboardTool creates a new SetDefaultDashboardTool instance.
func NewSetDefaultDashboardTool(client *client.Client, logger *zap.Logger) *SetDefaultDashboardTool {
	return &SetDefaultDashboardTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name for MCP registration.
func (t *SetDefaultDashboardTool) Name() string {
	return "set_default_dashboard"
}

// Description returns a human-readable description of the tool.
func (t *SetDefaultDashboardTool) Description() string {
	return "Set a dashboard as the default dashboard in IBM Cloud Logs"
}

// InputSchema returns the JSON schema for the tool's input parameters.
func (t *SetDefaultDashboardTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"dashboard_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the dashboard to set as default",
			},
		},
		Required: []string{"dashboard_id"},
	}
}

// Execute sets a dashboard as default.
func (t *SetDefaultDashboardTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	dashboardID, ok := arguments["dashboard_id"].(string)
	if !ok || dashboardID == "" {
		return mcp.NewToolResultError("dashboard_id is required and must be a string"), nil
	}

	req := &client.Request{
		Method: "PUT",
		Path:   fmt.Sprintf("/v1/dashboards/%s/default", dashboardID),
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}
