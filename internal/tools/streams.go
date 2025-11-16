package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// ListStreamsTool lists all streams
type ListStreamsTool struct {
	*BaseTool
}

func NewListStreamsTool(client *client.Client, logger *zap.Logger) *ListStreamsTool {
	return &ListStreamsTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *ListStreamsTool) Name() string {
	return "list_streams"
}

func (t *ListStreamsTool) Description() string {
	return "List all streams configured for the IBM Cloud Logs instance"
}

func (t *ListStreamsTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type:       "object",
		Properties: map[string]interface{}{},
		Required:   []string{},
	}
}

func (t *ListStreamsTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	req := &client.Request{
		Method: "GET",
		Path:   "/v1/streams",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// GetStreamTool gets a specific stream by ID
// Note: The API doesn't support GET for individual streams, so we list all and filter
type GetStreamTool struct {
	*BaseTool
}

func NewGetStreamTool(client *client.Client, logger *zap.Logger) *GetStreamTool {
	return &GetStreamTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *GetStreamTool) Name() string {
	return "get_stream"
}

func (t *GetStreamTool) Description() string {
	return "Get details of a specific stream by ID"
}

func (t *GetStreamTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"stream_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the stream",
			},
		},
		Required: []string{"stream_id"},
	}
}

func (t *GetStreamTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	streamID, err := GetStringParam(arguments, "stream_id", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// List all streams and filter for the requested ID
	req := &client.Request{
		Method: "GET",
		Path:   "/v1/streams",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Parse the response to filter by ID
	if streams, ok := result["streams"].([]interface{}); ok {
		for _, stream := range streams {
			if streamMap, ok := stream.(map[string]interface{}); ok {
				// Convert stream ID to string for comparison
				var id string
				switch v := streamMap["id"].(type) {
				case string:
					id = v
				case float64:
					id = fmt.Sprintf("%.0f", v)
				case int:
					id = fmt.Sprintf("%d", v)
				}

				if id == streamID {
					return t.FormatResponse(streamMap)
				}
			}
		}
	}

	return mcp.NewToolResultError("Stream not found with ID: " + streamID), nil
}

// CreateStreamTool creates a new stream
type CreateStreamTool struct {
	*BaseTool
}

func NewCreateStreamTool(client *client.Client, logger *zap.Logger) *CreateStreamTool {
	return &CreateStreamTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *CreateStreamTool) Name() string {
	return "create_stream"
}

func (t *CreateStreamTool) Description() string {
	return "Create a new stream for streaming logs to IBM Event Streams (Kafka)"
}

func (t *CreateStreamTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "The name of the stream (1-4096 characters)",
			},
			"dpxl_expression": map[string]interface{}{
				"type":        "string",
				"description": "DPXL expression to filter logs (e.g., '<v1>contains(kubernetes.labels.app, \"frontend\")')",
			},
			"is_active": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the stream is active (default: true)",
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
		Required: []string{"name", "dpxl_expression"},
	}
}

func (t *CreateStreamTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	name, err := GetStringParam(arguments, "name", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	dpxlExpression, err := GetStringParam(arguments, "dpxl_expression", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
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
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// UpdateStreamTool updates an existing stream
// Note: ALL fields are required for update (name, dpxl_expression, compression_type, ibm_event_streams)
type UpdateStreamTool struct {
	*BaseTool
}

func NewUpdateStreamTool(client *client.Client, logger *zap.Logger) *UpdateStreamTool {
	return &UpdateStreamTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *UpdateStreamTool) Name() string {
	return "update_stream"
}

func (t *UpdateStreamTool) Description() string {
	return "Update an existing stream. All fields must be provided (name, dpxl_expression, compression_type, ibm_event_streams)."
}

func (t *UpdateStreamTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"stream_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the stream to update",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "The name of the stream (1-4096 characters)",
			},
			"dpxl_expression": map[string]interface{}{
				"type":        "string",
				"description": "DPXL expression to filter logs",
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
				"required": []string{"brokers", "topic"},
			},
		},
		Required: []string{"stream_id", "name", "dpxl_expression", "compression_type", "ibm_event_streams"},
	}
}

func (t *UpdateStreamTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	streamID, err := GetStringParam(arguments, "stream_id", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	name, err := GetStringParam(arguments, "name", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	dpxlExpression, err := GetStringParam(arguments, "dpxl_expression", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	compressionType, err := GetStringParam(arguments, "compression_type", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	eventStreams, err := GetObjectParam(arguments, "ibm_event_streams", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	body := map[string]interface{}{
		"name":              name,
		"dpxl_expression":   dpxlExpression,
		"compression_type":  compressionType,
		"ibm_event_streams": eventStreams,
	}

	req := &client.Request{
		Method: "PUT",
		Path:   "/v1/streams/" + streamID,
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// DeleteStreamTool deletes a stream
type DeleteStreamTool struct {
	*BaseTool
}

func NewDeleteStreamTool(client *client.Client, logger *zap.Logger) *DeleteStreamTool {
	return &DeleteStreamTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *DeleteStreamTool) Name() string {
	return "delete_stream"
}

func (t *DeleteStreamTool) Description() string {
	return "Delete a stream"
}

func (t *DeleteStreamTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"stream_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the stream to delete",
			},
		},
		Required: []string{"stream_id"},
	}
}

func (t *DeleteStreamTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	streamID, err := GetStringParam(arguments, "stream_id", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "DELETE",
		Path:   "/v1/streams/" + streamID,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}
