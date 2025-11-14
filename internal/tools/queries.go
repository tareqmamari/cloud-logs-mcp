package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/observability-c/logs-mcp-server/internal/client"
	"go.uber.org/zap"
)

// QueryTool executes a synchronous query
type QueryTool struct {
	*BaseTool
}

func NewQueryTool(client *client.Client, logger *zap.Logger) *QueryTool {
	return &QueryTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *QueryTool) Name() string {
	return "query_logs"
}

func (t *QueryTool) Description() string {
	return "Execute a synchronous query against IBM Cloud Logs to search and analyze log data"
}

func (t *QueryTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The query string to execute (e.g., Lucene or DataPrime query)",
			},
			"start_time": map[string]interface{}{
				"type":        "string",
				"description": "Start time for the query (ISO 8601 format)",
			},
			"end_time": map[string]interface{}{
				"type":        "string",
				"description": "End time for the query (ISO 8601 format)",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of results to return",
			},
		},
		Required: []string{"query"},
	}
}

func (t *QueryTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	query, err := GetStringParam(arguments, "query", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	body := map[string]interface{}{
		"query": query,
	}

	if startTime, _ := GetStringParam(arguments, "start_time", false); startTime != "" {
		body["start_time"] = startTime
	}
	if endTime, _ := GetStringParam(arguments, "end_time", false); endTime != "" {
		body["end_time"] = endTime
	}
	if limit, _ := GetIntParam(arguments, "limit", false); limit > 0 {
		body["limit"] = limit
	}

	req := &client.Request{
		Method: "POST",
		Path:   "/v1/query",
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// SubmitBackgroundQueryTool submits an asynchronous background query
type SubmitBackgroundQueryTool struct {
	*BaseTool
}

func NewSubmitBackgroundQueryTool(client *client.Client, logger *zap.Logger) *SubmitBackgroundQueryTool {
	return &SubmitBackgroundQueryTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *SubmitBackgroundQueryTool) Name() string {
	return "submit_background_query"
}

func (t *SubmitBackgroundQueryTool) Description() string {
	return "Submit an asynchronous background query for large-scale log analysis"
}

func (t *SubmitBackgroundQueryTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The query string to execute",
			},
			"start_time": map[string]interface{}{
				"type":        "string",
				"description": "Start time for the query (ISO 8601 format)",
			},
			"end_time": map[string]interface{}{
				"type":        "string",
				"description": "End time for the query (ISO 8601 format)",
			},
		},
		Required: []string{"query", "start_time", "end_time"},
	}
}

func (t *SubmitBackgroundQueryTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	query, err := GetStringParam(arguments, "query", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	startTime, err := GetStringParam(arguments, "start_time", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	endTime, err := GetStringParam(arguments, "end_time", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	body := map[string]interface{}{
		"query":      query,
		"start_time": startTime,
		"end_time":   endTime,
	}

	req := &client.Request{
		Method: "POST",
		Path:   "/v1/background_query",
		Body:   body,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// GetBackgroundQueryStatusTool checks the status of a background query
type GetBackgroundQueryStatusTool struct {
	*BaseTool
}

func NewGetBackgroundQueryStatusTool(client *client.Client, logger *zap.Logger) *GetBackgroundQueryStatusTool {
	return &GetBackgroundQueryStatusTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *GetBackgroundQueryStatusTool) Name() string {
	return "get_background_query_status"
}

func (t *GetBackgroundQueryStatusTool) Description() string {
	return "Check the status of a background query"
}

func (t *GetBackgroundQueryStatusTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"query_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the background query",
			},
		},
		Required: []string{"query_id"},
	}
}

func (t *GetBackgroundQueryStatusTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	queryID, err := GetStringParam(arguments, "query_id", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "GET",
		Path:   "/v1/background_query/" + queryID + "/status",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// GetBackgroundQueryDataTool retrieves the results of a background query
type GetBackgroundQueryDataTool struct {
	*BaseTool
}

func NewGetBackgroundQueryDataTool(client *client.Client, logger *zap.Logger) *GetBackgroundQueryDataTool {
	return &GetBackgroundQueryDataTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *GetBackgroundQueryDataTool) Name() string {
	return "get_background_query_data"
}

func (t *GetBackgroundQueryDataTool) Description() string {
	return "Retrieve the results of a completed background query"
}

func (t *GetBackgroundQueryDataTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"query_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the background query",
			},
		},
		Required: []string{"query_id"},
	}
}

func (t *GetBackgroundQueryDataTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	queryID, err := GetStringParam(arguments, "query_id", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "GET",
		Path:   "/v1/background_query/" + queryID + "/data",
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}

// CancelBackgroundQueryTool cancels a running background query
type CancelBackgroundQueryTool struct {
	*BaseTool
}

func NewCancelBackgroundQueryTool(client *client.Client, logger *zap.Logger) *CancelBackgroundQueryTool {
	return &CancelBackgroundQueryTool{
		BaseTool: NewBaseTool(client, logger),
	}
}

func (t *CancelBackgroundQueryTool) Name() string {
	return "cancel_background_query"
}

func (t *CancelBackgroundQueryTool) Description() string {
	return "Cancel a running background query"
}

func (t *CancelBackgroundQueryTool) InputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"query_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the background query to cancel",
			},
		},
		Required: []string{"query_id"},
	}
}

func (t *CancelBackgroundQueryTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	queryID, err := GetStringParam(arguments, "query_id", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	req := &client.Request{
		Method: "DELETE",
		Path:   "/v1/background_query/" + queryID,
	}

	result, err := t.ExecuteRequest(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return t.FormatResponse(result)
}
