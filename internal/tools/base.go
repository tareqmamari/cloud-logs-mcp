package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/observability-c/logs-mcp-server/internal/client"
	"go.uber.org/zap"
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

	// Parse response
	var result map[string]interface{}
	if len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return result, nil
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
func GetStringParam(arguments map[string]interface{}, key string, required bool) (string, error) {
	val, exists := arguments[key]
	if !exists {
		if required {
			return "", fmt.Errorf("missing required parameter: %s", key)
		}
		return "", nil
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s must be a string", key)
	}

	return str, nil
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
