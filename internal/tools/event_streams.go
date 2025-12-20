package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// GetEventStreamTargetsTool lists all event stream targets
type GetEventStreamTargetsTool struct {
	*BaseTool
}

// NewGetEventStreamTargetsTool creates a new tool instance
func NewGetEventStreamTargetsTool(client *client.Client, logger *zap.Logger) *GetEventStreamTargetsTool {
	return &GetEventStreamTargetsTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *GetEventStreamTargetsTool) Name() string {
	return "get_event_stream_targets"
}

// Description returns the tool description
func (t *GetEventStreamTargetsTool) Description() string {
	return "Get all event stream targets configured for the IBM Cloud Logs instance"
}

// InputSchema returns the input schema
func (t *GetEventStreamTargetsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// Execute executes the tool
func (t *GetEventStreamTargetsTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	req := &client.Request{
		Method: "GET",
		Path:   "/v1/streams",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponseWithSuggestions(result, "get_event_stream_targets")
}

// CreateEventStreamTargetTool creates a new event stream target
type CreateEventStreamTargetTool struct {
	*BaseTool
}

// NewCreateEventStreamTargetTool creates a new tool instance
func NewCreateEventStreamTargetTool(client *client.Client, logger *zap.Logger) *CreateEventStreamTargetTool {
	return &CreateEventStreamTargetTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *CreateEventStreamTargetTool) Name() string {
	return "create_event_stream_target"
}

// Description returns the tool description
func (t *CreateEventStreamTargetTool) Description() string {
	return "Create a new event stream target for streaming logs to IBM Event Streams (Kafka)"
}

// InputSchema returns the input schema
func (t *CreateEventStreamTargetTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "The name of the event stream (1-4096 characters)",
			},
			"dpxl_expression": map[string]interface{}{
				"type":        "string",
				"description": "DPXL expression to filter logs (e.g., '<v1>contains(kubernetes.labels.app, \"frontend\")')",
			},
			"is_active": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the event stream is active (default: true)",
			},
			"compression_type": map[string]interface{}{
				"type":        "string",
				"description": "Compression type: gzip, snappy, lz4, zstd, or unspecified",
				"enum":        []string{"gzip", "snappy", "lz4", "zstd", "unspecified"},
			},
			"ibm_event_streams": map[string]interface{}{
				"type":        "object",
				"description": "IBM Event Streams (Kafka) configuration",
				"properties": map[string]interface{}{
					"brokers": map[string]interface{}{
						"type":        "string",
						"description": "Kafka broker endpoints (comma-separated)",
					},
					"topic": map[string]interface{}{
						"type":        "string",
						"description": "Kafka topic name",
					},
				},
			},
		},
		"required": []string{"name", "dpxl_expression"},
	}
}

// Execute executes the tool
func (t *CreateEventStreamTargetTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	name, err := GetStringParam(arguments, "name", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	dpxlExpression, err := GetStringParam(arguments, "dpxl_expression", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	body := map[string]interface{}{
		"name":            name,
		"dpxl_expression": dpxlExpression,
	}

	// Add optional fields
	if isActive, ok := arguments["is_active"].(bool); ok {
		body["is_active"] = isActive
	}

	if compressionType, _ := GetStringParam(arguments, "compression_type", false); compressionType != "" {
		body["compression_type"] = compressionType
	}

	if eventStreams, ok := arguments["ibm_event_streams"].(map[string]interface{}); ok {
		body["ibm_event_streams"] = eventStreams
	}

	req := &client.Request{
		Method: "POST",
		Path:   "/v1/streams",
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponseWithSuggestions(result, "create_event_stream_target")
}

// UpdateEventStreamTargetTool updates an existing event stream target
type UpdateEventStreamTargetTool struct {
	*BaseTool
}

// NewUpdateEventStreamTargetTool creates a new tool instance
func NewUpdateEventStreamTargetTool(client *client.Client, logger *zap.Logger) *UpdateEventStreamTargetTool {
	return &UpdateEventStreamTargetTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *UpdateEventStreamTargetTool) Name() string {
	return "update_event_stream_target"
}

// Description returns the tool description
func (t *UpdateEventStreamTargetTool) Description() string {
	return "Update an existing event stream target configuration"
}

// InputSchema returns the input schema
func (t *UpdateEventStreamTargetTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"stream_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the event stream to update",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "The name of the event stream (1-4096 characters)",
			},
			"dpxl_expression": map[string]interface{}{
				"type":        "string",
				"description": "DPXL expression to filter logs",
			},
			"is_active": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the event stream is active",
			},
			"compression_type": map[string]interface{}{
				"type":        "string",
				"description": "Compression type: gzip, snappy, lz4, zstd, or unspecified",
				"enum":        []string{"gzip", "snappy", "lz4", "zstd", "unspecified"},
			},
			"ibm_event_streams": map[string]interface{}{
				"type":        "object",
				"description": "IBM Event Streams (Kafka) configuration",
			},
		},
		"required": []string{"stream_id", "name", "dpxl_expression"},
	}
}

// Execute executes the tool
func (t *UpdateEventStreamTargetTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	streamID, err := GetStringParam(arguments, "stream_id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	name, err := GetStringParam(arguments, "name", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	dpxlExpression, err := GetStringParam(arguments, "dpxl_expression", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	body := map[string]interface{}{
		"name":            name,
		"dpxl_expression": dpxlExpression,
	}

	// Add optional fields
	if isActive, ok := arguments["is_active"].(bool); ok {
		body["is_active"] = isActive
	}

	if compressionType, _ := GetStringParam(arguments, "compression_type", false); compressionType != "" {
		body["compression_type"] = compressionType
	}

	if eventStreams, ok := arguments["ibm_event_streams"].(map[string]interface{}); ok {
		body["ibm_event_streams"] = eventStreams
	}

	req := &client.Request{
		Method: "PUT",
		Path:   "/v1/streams/" + streamID,
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponseWithSuggestions(result, "update_event_stream_target")
}

// DeleteEventStreamTargetTool deletes an event stream target
type DeleteEventStreamTargetTool struct {
	*BaseTool
}

// NewDeleteEventStreamTargetTool creates a new tool instance
func NewDeleteEventStreamTargetTool(client *client.Client, logger *zap.Logger) *DeleteEventStreamTargetTool {
	return &DeleteEventStreamTargetTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *DeleteEventStreamTargetTool) Name() string {
	return "delete_event_stream_target"
}

// Annotations returns tool hints for LLMs
func (t *DeleteEventStreamTargetTool) Annotations() *mcp.ToolAnnotations {
	return DeleteAnnotations("Delete Event Stream Target")
}

// Description returns the tool description
func (t *DeleteEventStreamTargetTool) Description() string {
	return "Delete an event stream target"
}

// InputSchema returns the input schema
func (t *DeleteEventStreamTargetTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"stream_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the event stream to delete",
			},
			"confirm": ConfirmationInputSchema(),
		},
		"required": []string{"stream_id"},
	}
}

// Execute executes the tool
func (t *DeleteEventStreamTargetTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	streamID, err := GetStringParam(arguments, "stream_id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}
	if shouldContinue, result := RequireConfirmation(arguments, "event stream target", streamID); !shouldContinue {
		return result, nil
	}

	req := &client.Request{
		Method: "DELETE",
		Path:   "/v1/streams/" + streamID,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	return t.FormatResponseWithSuggestions(result, "delete_event_stream_target")
}
