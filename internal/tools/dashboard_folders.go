package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/observability-c/logs-mcp-server/internal/client"
	"go.uber.org/zap"
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
