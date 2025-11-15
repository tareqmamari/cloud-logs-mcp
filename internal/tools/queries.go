package tools

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// normalizeTier maps user-friendly tier names to API values
func normalizeTier(tier string) string {
	// Convert to lowercase for case-insensitive matching
	tier = strings.ToLower(strings.TrimSpace(tier))

	// Map Priority Insights / frequent search aliases
	frequentSearchAliases := []string{
		"pi", "priority", "insights", "priority insights",
		"frequent", "quick", "fast", "hot", "realtime", "real-time",
	}
	for _, alias := range frequentSearchAliases {
		if strings.Contains(tier, alias) {
			return "frequent_search"
		}
	}

	// Map archive / cold storage aliases
	archiveAliases := []string{
		"archive", "storage", "cos", "cold", "s3", "object storage",
		"long term", "long-term", "historical",
	}
	for _, alias := range archiveAliases {
		if strings.Contains(tier, alias) {
			return "archive"
		}
	}

	// If it's already a valid value, return it
	validTiers := map[string]bool{
		"unspecified":     true,
		"archive":         true,
		"frequent_search": true,
	}
	if validTiers[tier] {
		return tier
	}

	// Default to frequent_search if unrecognized
	return "frequent_search"
}

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
				"description": "The query string to execute (DataPrime or Lucene syntax)",
			},
			"tier": map[string]interface{}{
				"type":        "string",
				"description": "Log tier to query. frequent_search (default, aliases: PI, priority, insights, quick), archive (aliases: COS, storage, cold), or unspecified",
				"enum":        []string{"unspecified", "archive", "frequent_search"},
				"default":     "frequent_search",
			},
			"syntax": map[string]interface{}{
				"type":        "string",
				"description": "Query syntax: dataprime (default), lucene, or unspecified",
				"enum":        []string{"unspecified", "lucene", "dataprime"},
				"default":     "dataprime",
			},
			"start_date": map[string]interface{}{
				"type":        "string",
				"description": "Start date for the query (ISO 8601 format, e.g., 2024-05-01T20:47:12.940Z)",
			},
			"end_date": map[string]interface{}{
				"type":        "string",
				"description": "End date for the query (ISO 8601 format, e.g., 2024-05-01T20:47:12.940Z)",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of results to return (default: 2000, max: 50000)",
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

	// Build metadata object with all optional parameters
	metadata := make(map[string]interface{})

	// Apply defaults for tier and syntax with intelligent alias mapping
	tier, _ := GetStringParam(arguments, "tier", false)
	if tier == "" {
		tier = "frequent_search"
	} else {
		// Normalize user-friendly aliases (PI, archive, COS, etc.)
		tier = normalizeTier(tier)
	}
	metadata["tier"] = tier

	syntax, _ := GetStringParam(arguments, "syntax", false)
	if syntax == "" {
		syntax = "dataprime"
	}
	metadata["syntax"] = syntax

	// Add date range if provided
	if startDate, _ := GetStringParam(arguments, "start_date", false); startDate != "" {
		metadata["start_date"] = startDate
	}
	if endDate, _ := GetStringParam(arguments, "end_date", false); endDate != "" {
		metadata["end_date"] = endDate
	}

	// Add limit with proper default
	limit, _ := GetIntParam(arguments, "limit", false)
	if limit > 0 {
		metadata["limit"] = limit
	} else {
		metadata["limit"] = 2000 // Default limit per API spec
	}

	// Build request body with query at top level and metadata nested
	body := map[string]interface{}{
		"query":    query,
		"metadata": metadata,
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
				"description": "The query string to execute (1-4096 characters)",
			},
			"syntax": map[string]interface{}{
				"type":        "string",
				"description": "Query syntax: dataprime (default), lucene, or unspecified",
				"enum":        []string{"unspecified", "lucene", "dataprime"},
				"default":     "dataprime",
			},
			"start_date": map[string]interface{}{
				"type":        "string",
				"description": "Start date for the query (ISO 8601 format, e.g., 2024-05-01T20:47:12.940Z). Optional, defaults to end - 15 minutes",
			},
			"end_date": map[string]interface{}{
				"type":        "string",
				"description": "End date for the query (ISO 8601 format, e.g., 2024-05-01T20:47:12.940Z). Optional, defaults to now",
			},
		},
		Required: []string{"query", "syntax"},
	}
}

func (t *SubmitBackgroundQueryTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	query, err := GetStringParam(arguments, "query", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	syntax, err := GetStringParam(arguments, "syntax", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Build request body with required fields
	body := map[string]interface{}{
		"query":  query,
		"syntax": syntax,
	}

	// Add optional date fields if provided
	if startDate, _ := GetStringParam(arguments, "start_date", false); startDate != "" {
		body["start_date"] = startDate
	}
	if endDate, _ := GetStringParam(arguments, "end_date", false); endDate != "" {
		body["end_date"] = endDate
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
