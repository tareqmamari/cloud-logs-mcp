package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// BaseTool provides common functionality for all tools
type BaseTool struct {
	client *client.Client
	logger *zap.Logger
}

// NewBaseTool creates a new base tool
func NewBaseTool(client *client.Client, logger *zap.Logger) *BaseTool {
	return &BaseTool{
		client: client,
		logger: logger,
	}
}

// ExecuteRequest executes an API request and returns the response
func (t *BaseTool) ExecuteRequest(ctx context.Context, req *client.Request) (map[string]interface{}, error) {
	resp, err := t.client.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode >= 400 {
		var apiError map[string]interface{}
		if err := json.Unmarshal(resp.Body, &apiError); err == nil {
			return nil, fmt.Errorf("API error (HTTP %d): %v", resp.StatusCode, apiError)
		}
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(resp.Body))
	}

	// Parse response - handle both JSON and Server-Sent Events (SSE)
	var result map[string]interface{}
	if len(resp.Body) > 0 {
		// Try parsing as Server-Sent Events first (for query responses)
		if sseResult := parseSSEResponse(resp.Body); sseResult != nil {
			return sseResult, nil
		}

		// Fall back to standard JSON
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return result, nil
}

// parseSSEResponse parses Server-Sent Events format responses
// IBM Cloud Logs query API returns results in SSE format like:
// : success
// data: {"query_id":...}
//
// : success
// data: {"result":{"results":[...]}}
func parseSSEResponse(body []byte) map[string]interface{} {
	bodyStr := string(body)

	// Check if this looks like SSE format
	if !strings.Contains(bodyStr, "data: {") {
		return nil
	}

	result := make(map[string]interface{})
	lines := strings.Split(bodyStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(line, ":") || line == "" {
			continue
		}

		// Parse data lines
		if strings.HasPrefix(line, "data: ") {
			dataJSON := strings.TrimPrefix(line, "data: ")

			var dataObj map[string]interface{}
			if err := json.Unmarshal([]byte(dataJSON), &dataObj); err == nil {
				// Merge all data objects into result
				for k, v := range dataObj {
					result[k] = v
				}
			}
		}
	}

	if len(result) > 0 {
		return result
	}

	return nil
}

// FormatResponse formats the response as a text/content for MCP
func (t *BaseTool) FormatResponse(result map[string]interface{}) (*mcp.CallToolResult, error) {
	// Pretty print JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// GetStringParam safely gets a string parameter from arguments
// It also handles numeric IDs and converts them to strings
func GetStringParam(arguments map[string]interface{}, key string, required bool) (string, error) {
	val, exists := arguments[key]
	if !exists {
		if required {
			return "", fmt.Errorf("missing required parameter: %s", key)
		}
		return "", nil
	}

	// Handle string type
	if str, ok := val.(string); ok {
		return str, nil
	}

	// Handle numeric types (for IDs that might be numbers)
	switch v := val.(type) {
	case float64:
		return fmt.Sprintf("%.0f", v), nil
	case int:
		return fmt.Sprintf("%d", v), nil
	case int64:
		return fmt.Sprintf("%d", v), nil
	default:
		return "", fmt.Errorf("parameter %s must be a string or number", key)
	}
}

// GetObjectParam safely gets an object parameter from arguments
func GetObjectParam(arguments map[string]interface{}, key string, required bool) (map[string]interface{}, error) {
	val, exists := arguments[key]
	if !exists {
		if required {
			return nil, fmt.Errorf("missing required parameter: %s", key)
		}
		return nil, nil
	}

	obj, ok := val.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("parameter %s must be an object", key)
	}

	return obj, nil
}

// GetIntParam safely gets an integer parameter from arguments
func GetIntParam(arguments map[string]interface{}, key string, required bool) (int, error) {
	val, exists := arguments[key]
	if !exists {
		if required {
			return 0, fmt.Errorf("missing required parameter: %s", key)
		}
		return 0, nil
	}

	// Handle both float64 (JSON numbers) and int
	switch v := val.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
		return 0, fmt.Errorf("parameter %s must be a number", key)
	}
}

// GetBoolParam safely gets a boolean parameter from arguments
func GetBoolParam(arguments map[string]interface{}, key string, required bool) (bool, error) {
	val, exists := arguments[key]
	if !exists {
		if required {
			return false, fmt.Errorf("missing required parameter: %s", key)
		}
		return false, nil
	}

	boolVal, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("parameter %s must be a boolean", key)
	}

	return boolVal, nil
}

// PaginationParams holds pagination parameters
type PaginationParams struct {
	Limit  int
	Cursor string
}

// GetPaginationParams extracts pagination parameters from arguments
func GetPaginationParams(arguments map[string]interface{}) (*PaginationParams, error) {
	limit, err := GetIntParam(arguments, "limit", false)
	if err != nil {
		return nil, err
	}

	// Default limit
	if limit == 0 {
		limit = 50
	}

	// Max limit
	if limit > 100 {
		limit = 100
	}

	cursor, err := GetStringParam(arguments, "cursor", false)
	if err != nil {
		return nil, err
	}

	return &PaginationParams{
		Limit:  limit,
		Cursor: cursor,
	}, nil
}

// AddPaginationToQuery adds pagination parameters to request query
func AddPaginationToQuery(query map[string]string, params *PaginationParams) {
	if query == nil {
		return
	}

	if params.Limit > 0 {
		query["limit"] = fmt.Sprintf("%d", params.Limit)
	}

	if params.Cursor != "" {
		query["cursor"] = params.Cursor
	}
}
