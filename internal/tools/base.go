// Package tools provides the MCP tool implementations for IBM Cloud Logs.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
		// Check for context timeout/cancellation
		if ctx.Err() != nil {
			return nil, &APIError{
				StatusCode: 408,
				Message:    fmt.Sprintf("Request timed out: %v", ctx.Err()),
			}
		}
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	// Extract request ID from response headers (IBM Cloud uses X-Request-ID or X-Correlation-ID)
	requestID := resp.Headers.Get("X-Request-ID")
	if requestID == "" {
		requestID = resp.Headers.Get("X-Correlation-ID")
	}
	if requestID == "" {
		requestID = resp.Headers.Get("X-Global-Transaction-ID")
	}

	// Check for error status codes
	if resp.StatusCode >= 400 {
		var apiError map[string]interface{}
		var errorMessage string
		if err := json.Unmarshal(resp.Body, &apiError); err == nil {
			errorMessage = fmt.Sprintf("API error (HTTP %d): %v", resp.StatusCode, apiError)
		} else {
			errorMessage = fmt.Sprintf("API error (HTTP %d): %s", resp.StatusCode, string(resp.Body))
		}
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    errorMessage,
			RequestID:  requestID,
			Details:    apiError,
		}
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
// Limits to MaxSSEEvents to prevent memory issues with large result sets
func parseSSEResponse(body []byte) map[string]interface{} {
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "data:") {
		return nil
	}

	// Simple SSE parser for our use case
	// We expect lines starting with "data: " containing JSON
	// We'll aggregate them into a list of events
	var events []interface{}
	totalCount := 0
	truncated := false
	lines := strings.Split(bodyStr, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			totalCount++
			// Limit the number of events to prevent memory issues
			if len(events) >= MaxSSEEvents {
				truncated = true
				continue // Keep counting but don't add more events
			}
			dataStr := strings.TrimPrefix(line, "data: ")
			var data interface{}
			if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
				events = append(events, data)
			}
		}
	}

	if len(events) > 0 {
		result := map[string]interface{}{
			"events": events,
		}
		if truncated {
			result["_truncated"] = true
			result["_total_events"] = totalCount
			result["_shown_events"] = len(events)
		}
		return result
	}

	return nil
}
