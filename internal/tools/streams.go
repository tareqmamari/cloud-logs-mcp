package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// ListStreamsTool lists all streams
type ListStreamsTool struct {
	*BaseTool
}

// NewListStreamsTool creates a new tool instance
func NewListStreamsTool(client *client.Client, logger *zap.Logger) *ListStreamsTool {
	return &ListStreamsTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *ListStreamsTool) Name() string {
	return "list_streams"
}

// Description returns the tool description
func (t *ListStreamsTool) Description() string {
	return "List all streams configured for the IBM Cloud Logs instance"
}

// InputSchema returns the input schema
func (t *ListStreamsTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// Execute executes the tool
func (t *ListStreamsTool) Execute(ctx context.Context, _ map[string]interface{}) (*mcp.CallToolResult, error) {
	session := GetSession()
	cacheHelper := GetCacheHelper()

	// Check cache first
	if cached, ok := cacheHelper.Get(t.Name(), "all"); ok {
		if cachedResult, ok := cached.(map[string]interface{}); ok {
			session.RecordToolUse(t.Name(), true, nil)
			cachedResult["_cached"] = true
			return t.FormatResponseWithSuggestions(cachedResult, "list_streams")
		}
	}

	req := &client.Request{
		Method: "GET",
		Path:   "/v1/streams",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		session.RecordToolUse(t.Name(), false, nil)
		return NewToolResultError(err.Error()), nil
	}

	// Cache the result
	cacheHelper.Set(t.Name(), "all", result)
	session.RecordToolUse(t.Name(), true, nil)

	return t.FormatResponseWithSuggestions(result, "list_streams")
}

// GetStreamTool gets a specific stream by ID
// Note: The API doesn't support GET for individual streams, so we list all and filter
type GetStreamTool struct {
	*BaseTool
}

// NewGetStreamTool creates a new tool instance
func NewGetStreamTool(client *client.Client, logger *zap.Logger) *GetStreamTool {
	return &GetStreamTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *GetStreamTool) Name() string {
	return "get_stream"
}

// Description returns the tool description
func (t *GetStreamTool) Description() string {
	return "Get details of a specific stream by ID"
}

// InputSchema returns the input schema
func (t *GetStreamTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"stream_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the stream",
			},
		},
		"required": []string{"stream_id"},
	}
}

// Execute executes the tool
func (t *GetStreamTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	streamID, err := GetStringParam(arguments, "stream_id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// List all streams and filter for the requested ID
	req := &client.Request{
		Method: "GET",
		Path:   "/v1/streams",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
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
					return t.FormatResponseWithSuggestions(streamMap, "get_stream")
				}
			}
		}
	}

	return NewResourceNotFoundError("Stream", streamID, "list_streams"), nil
}

// CreateStreamTool creates a new stream
type CreateStreamTool struct {
	*BaseTool
}

// NewCreateStreamTool creates a new tool instance
func NewCreateStreamTool(client *client.Client, logger *zap.Logger) *CreateStreamTool {
	return &CreateStreamTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *CreateStreamTool) Name() string {
	return "create_stream"
}

// Description returns the tool description
func (t *CreateStreamTool) Description() string {
	return "Create a new stream for streaming logs to IBM Event Streams (Kafka)"
}

// InputSchema returns the input schema
func (t *CreateStreamTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "The name of the stream (1-4096 characters)",
				"examples":    []string{"production-errors-stream", "frontend-logs-stream"},
			},
			"dpxl_expression": map[string]interface{}{
				"type":        "string",
				"description": "DPXL expression to filter logs (e.g., '<v1>contains(kubernetes.labels.app, \"frontend\")')",
				"examples": []string{
					"<v1>contains(kubernetes.labels.app, \"frontend\")",
					"<v1>severity >= 5",
					"<v1>applicationname == \"api-gateway\"",
				},
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
				"example": map[string]interface{}{
					"brokers": "broker-1.kafka.svc.cluster.local:9092,broker-2.kafka.svc.cluster.local:9092",
					"topic":   "production-logs",
				},
			},
			"dry_run": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, validates the stream configuration without creating it. Use this to preview what will be created and check for errors.",
				"default":     false,
			},
		},
		"required": []string{"name", "dpxl_expression"},
	}
}

// Execute executes the tool
func (t *CreateStreamTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	cacheHelper := GetCacheHelper()

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

	compressionType, _ := GetStringParam(arguments, "compression_type", false)
	if compressionType != "" {
		body["compression_type"] = compressionType
	}

	if eventStreams, ok := arguments["ibm_event_streams"].(map[string]interface{}); ok {
		body["ibm_event_streams"] = eventStreams
	}

	// Check for dry-run mode
	dryRun, _ := GetBoolParam(arguments, "dry_run", false)
	if dryRun {
		return t.validateStream(body)
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

	// Invalidate related caches
	cacheHelper.InvalidateRelated(t.Name())

	return t.FormatResponseWithSuggestions(result, "create_stream")
}

// validateStream performs dry-run validation for stream creation
func (t *CreateStreamTool) validateStream(stream map[string]interface{}) (*mcp.CallToolResult, error) {
	result := &ValidationResult{
		Valid:   true,
		Summary: make(map[string]interface{}),
	}

	// Validate required fields
	if name, ok := stream["name"].(string); ok {
		if len(name) < 1 {
			result.Errors = append(result.Errors, "Stream name must not be empty")
			result.Valid = false
		}
		if len(name) > 4096 {
			result.Errors = append(result.Errors, "Stream name must be at most 4096 characters")
			result.Valid = false
		}
		result.Summary["name"] = name
	} else {
		result.Errors = append(result.Errors, "Missing required field: name")
		result.Valid = false
	}

	// Validate DPXL expression
	if dpxl, ok := stream["dpxl_expression"].(string); ok {
		if len(dpxl) < 1 {
			result.Errors = append(result.Errors, "DPXL expression must not be empty")
			result.Valid = false
		}
		// Basic DPXL syntax check
		if !strings.HasPrefix(dpxl, "<v1>") {
			result.Warnings = append(result.Warnings, "DPXL expression should start with '<v1>' version prefix")
		}
		result.Summary["dpxl_expression"] = dpxl
	} else {
		result.Errors = append(result.Errors, "Missing required field: dpxl_expression")
		result.Valid = false
	}

	// Validate compression type
	if compression, ok := stream["compression_type"].(string); ok {
		validCompressions := []string{"gzip", "snappy", "lz4", "zstd", "unspecified"}
		isValid := false
		for _, valid := range validCompressions {
			if compression == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid compression_type: '%s'. Must be one of: %v", compression, validCompressions))
			result.Valid = false
		}
		result.Summary["compression_type"] = compression
	}

	// Validate IBM Event Streams config
	if eventStreams, ok := stream["ibm_event_streams"].(map[string]interface{}); ok {
		if brokers, ok := eventStreams["brokers"].(string); ok && brokers != "" {
			result.Summary["brokers"] = brokers
		} else {
			result.Warnings = append(result.Warnings, "No brokers specified in ibm_event_streams - stream may not function correctly")
		}
		if topic, ok := eventStreams["topic"].(string); ok && topic != "" {
			result.Summary["topic"] = topic
		} else {
			result.Warnings = append(result.Warnings, "No topic specified in ibm_event_streams - stream may not function correctly")
		}
	} else {
		result.Warnings = append(result.Warnings, "No ibm_event_streams configuration provided - stream destination not configured")
	}

	// Validate is_active
	if isActive, ok := stream["is_active"].(bool); ok {
		result.Summary["is_active"] = isActive
	} else {
		result.Summary["is_active"] = true // default
	}

	// Add suggestions
	if result.Valid {
		result.Suggestions = append(result.Suggestions, "Stream configuration is valid")
		result.Suggestions = append(result.Suggestions, "Remove dry_run parameter to create the stream")
	} else {
		result.Suggestions = append(result.Suggestions, "Fix the errors above before creating the stream")
	}

	// Estimate impact
	result.EstimatedImpact = &ImpactEstimate{
		EstimatedCost: "Data egress charges may apply based on stream volume",
		RiskLevel:     "medium", // Streams can have cost implications
	}

	return FormatDryRunResult(result, "Stream", stream), nil
}

// UpdateStreamTool updates an existing stream
// Note: ALL fields are required for update (name, dpxl_expression, compression_type, ibm_event_streams)
type UpdateStreamTool struct {
	*BaseTool
}

// NewUpdateStreamTool creates a new tool instance
func NewUpdateStreamTool(client *client.Client, logger *zap.Logger) *UpdateStreamTool {
	return &UpdateStreamTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *UpdateStreamTool) Name() string {
	return "update_stream"
}

// Description returns the tool description
func (t *UpdateStreamTool) Description() string {
	return "Update an existing stream. All fields must be provided (name, dpxl_expression, compression_type, ibm_event_streams)."
}

// InputSchema returns the input schema
func (t *UpdateStreamTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
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
		"required": []string{"stream_id", "name", "dpxl_expression", "compression_type", "ibm_event_streams"},
	}
}

// Execute executes the tool
func (t *UpdateStreamTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	cacheHelper := GetCacheHelper()

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

	compressionType, err := GetStringParam(arguments, "compression_type", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	eventStreams, err := GetObjectParam(arguments, "ibm_event_streams", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
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
		return NewToolResultError(err.Error()), nil
	}

	// Invalidate related caches
	cacheHelper.InvalidateRelated(t.Name())

	return t.FormatResponseWithSuggestions(result, "update_stream")
}

// DeleteStreamTool deletes a stream
type DeleteStreamTool struct {
	*BaseTool
}

// NewDeleteStreamTool creates a new tool instance
func NewDeleteStreamTool(client *client.Client, logger *zap.Logger) *DeleteStreamTool {
	return &DeleteStreamTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

// Name returns the tool name
func (t *DeleteStreamTool) Name() string {
	return "delete_stream"
}

// Annotations returns tool hints for LLMs
func (t *DeleteStreamTool) Annotations() *mcp.ToolAnnotations {
	return DeleteAnnotations("Delete Stream")
}

// Description returns the tool description
func (t *DeleteStreamTool) Description() string {
	return "Delete a stream"
}

// InputSchema returns the input schema
func (t *DeleteStreamTool) InputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"stream_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the stream to delete",
			},
		},
		"required": []string{"stream_id"},
	}
}

// Execute executes the tool
func (t *DeleteStreamTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	cacheHelper := GetCacheHelper()

	streamID, err := GetStringParam(arguments, "stream_id", true)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "DELETE",
		Path:   "/v1/streams/" + streamID,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return NewToolResultError(err.Error()), nil
	}

	// Invalidate related caches
	cacheHelper.InvalidateRelated(t.Name())

	return t.FormatResponseWithSuggestions(result, "delete_stream")
}
