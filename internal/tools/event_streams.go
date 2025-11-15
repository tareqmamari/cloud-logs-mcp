package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/observability-c/logs-mcp-server/internal/client"
	"go.uber.org/zap"
)

// GetEventStreamTargetsTool lists all event stream targets
type GetEventStreamTargetsTool struct {
	*BaseTool
}

func NewGetEventStreamTargetsTool(client *client.Client, logger *zap.Logger) *GetEventStreamTargetsTool {
	return &GetEventStreamTargetsTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *GetEventStreamTargetsTool) Name() string {
	return "get_event_stream_targets"
}

func (t *GetEventStreamTargetsTool) Description() string {
	return "Get all event stream targets configured for the IBM Cloud Logs instance"
}

func (t *GetEventStreamTargetsTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type:       "object",
		Properties: map[string]interface{}{},
		Required:   []string{},
	}
}

func (t *GetEventStreamTargetsTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	req := &client.Request{
		Method: "GET",
		Path:   "/v1/config/event_stream_targets",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// CreateEventStreamTargetTool creates a new event stream target
type CreateEventStreamTargetTool struct {
	*BaseTool
}

func NewCreateEventStreamTargetTool(client *client.Client, logger *zap.Logger) *CreateEventStreamTargetTool {
	return &CreateEventStreamTargetTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *CreateEventStreamTargetTool) Name() string {
	return "create_event_stream_target"
}

func (t *CreateEventStreamTargetTool) Description() string {
	return "Create a new event stream target for streaming logs to external systems"
}

func (t *CreateEventStreamTargetTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"target_type": map[string]interface{}{
				"type":        "string",
				"description": "Type of event stream target (e.g., event_streams, event_notifications)",
			},
			"config": map[string]interface{}{
				"type":        "object",
				"description": "Configuration for the event stream target (specific to target_type)",
			},
		},
		Required: []string{"target_type", "config"},
	}
}

func (t *CreateEventStreamTargetTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	targetType, err := GetStringParam(arguments, "target_type", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	config, ok := arguments["config"].(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("config must be an object"), nil
	}

	body := map[string]interface{}{
		"target_type": targetType,
		"config":      config,
	}

	req := &client.Request{
		Method: "POST",
		Path:   "/v1/config/event_stream_targets",
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// UpdateEventStreamTargetTool updates an existing event stream target
type UpdateEventStreamTargetTool struct {
	*BaseTool
}

func NewUpdateEventStreamTargetTool(client *client.Client, logger *zap.Logger) *UpdateEventStreamTargetTool {
	return &UpdateEventStreamTargetTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *UpdateEventStreamTargetTool) Name() string {
	return "update_event_stream_target"
}

func (t *UpdateEventStreamTargetTool) Description() string {
	return "Update an existing event stream target configuration"
}

func (t *UpdateEventStreamTargetTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"target_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the event stream target to update",
			},
			"target_type": map[string]interface{}{
				"type":        "string",
				"description": "Type of event stream target (e.g., event_streams, event_notifications)",
			},
			"config": map[string]interface{}{
				"type":        "object",
				"description": "Updated configuration for the event stream target",
			},
		},
		Required: []string{"target_id", "target_type", "config"},
	}
}

func (t *UpdateEventStreamTargetTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	targetID, err := GetStringParam(arguments, "target_id", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	targetType, err := GetStringParam(arguments, "target_type", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	config, ok := arguments["config"].(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("config must be an object"), nil
	}

	body := map[string]interface{}{
		"target_type": targetType,
		"config":      config,
	}

	req := &client.Request{
		Method: "PUT",
		Path:   "/v1/config/event_stream_targets/" + targetID,
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// DeleteEventStreamTargetTool deletes an event stream target
type DeleteEventStreamTargetTool struct {
	*BaseTool
}

func NewDeleteEventStreamTargetTool(client *client.Client, logger *zap.Logger) *DeleteEventStreamTargetTool {
	return &DeleteEventStreamTargetTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *DeleteEventStreamTargetTool) Name() string {
	return "delete_event_stream_target"
}

func (t *DeleteEventStreamTargetTool) Description() string {
	return "Delete an event stream target"
}

func (t *DeleteEventStreamTargetTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"target_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the event stream target to delete",
			},
		},
		Required: []string{"target_id"},
	}
}

func (t *DeleteEventStreamTargetTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	targetID, err := GetStringParam(arguments, "target_id", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "DELETE",
		Path:   "/v1/config/event_stream_targets/" + targetID,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}
