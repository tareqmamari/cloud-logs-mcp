package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
)

// BaseTool provides common functionality for all tools
type BaseTool struct {
	client *client.Client
	logger *zap.Logger
}

// NewBaseTool creates a new BaseTool
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

// parseSSEResponse attempts to parse a response body as Server-Sent Events
// Returns nil if parsing fails or if it doesn't look like SSE
func parseSSEResponse(body []byte) map[string]interface{} {
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "data:") {
		return nil
	}

	// Simple SSE parser for our use case
	// We expect lines starting with "data: " containing JSON
	// We'll aggregate them into a list of events
	var events []interface{}
	lines := strings.Split(bodyStr, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			dataStr := strings.TrimPrefix(line, "data: ")
			var data interface{}
			if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
				events = append(events, data)
			}
		}
	}

	if len(events) > 0 {
		return map[string]interface{}{
			"events": events,
		}
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

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonBytes),
			},
		},
	}, nil
}

// NewToolResultError creates a new tool result with an error message
func NewToolResultError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: message,
			},
		},
		IsError: true,
	}
}

// GetStringParam safely gets a string parameter from arguments
// It also handles numeric IDs and converts them to strings
func GetStringParam(arguments map[string]interface{}, key string, required bool) (string, error) {
	val, ok := arguments[key]
	if !ok {
		if required {
			return "", fmt.Errorf("missing required argument: %s", key)
		}
		return "", nil
	}

	switch v := val.(type) {
	case string:
		return v, nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	default:
		return "", fmt.Errorf("invalid type for argument %s: expected string or number, got %T", key, val)
	}
}

// GetObjectParam safely gets a map/object parameter from arguments
func GetObjectParam(arguments map[string]interface{}, key string, required bool) (map[string]interface{}, error) {
	val, ok := arguments[key]
	if !ok {
		if required {
			return nil, fmt.Errorf("missing required argument: %s", key)
		}
		return nil, nil
	}

	obj, ok := val.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type for argument %s: expected object", key)
	}

	return obj, nil
}

// GetIntParam safely gets an integer parameter from arguments
func GetIntParam(arguments map[string]interface{}, key string, required bool) (int, error) {
	val, ok := arguments[key]
	if !ok {
		if required {
			return 0, fmt.Errorf("missing required argument: %s", key)
		}
		return 0, nil
	}

	switch v := val.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("invalid type for argument %s: expected number or string, got %T", key, val)
	}
}

// GetBoolParam safely gets a boolean parameter from arguments
func GetBoolParam(arguments map[string]interface{}, key string, required bool) (bool, error) {
	val, ok := arguments[key]
	if !ok {
		if required {
			return false, fmt.Errorf("missing required argument: %s", key)
		}
		return false, nil
	}

	switch v := val.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("invalid type for argument %s: expected boolean or string, got %T", key, val)
	}
}

// GetPaginationParams extracts pagination parameters (limit, cursor)
func GetPaginationParams(arguments map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	if limit, ok := arguments["limit"]; ok {
		params["limit"] = limit
	}

	if cursor, ok := arguments["cursor"]; ok {
		params["cursor"] = cursor
	}

	return params, nil
}

// AddPaginationToQuery adds pagination parameters to query map
func AddPaginationToQuery(query map[string]string, pagination map[string]interface{}) {
	if limit, ok := pagination["limit"]; ok {
		switch v := limit.(type) {
		case float64:
			query["limit"] = strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			query["limit"] = strconv.Itoa(v)
		}
	}

	if cursor, ok := pagination["cursor"]; ok {
		if s, ok := cursor.(string); ok {
			query["cursor"] = s
		}
	}
}
