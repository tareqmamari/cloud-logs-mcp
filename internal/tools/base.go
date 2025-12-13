// Package tools provides the MCP tool implementations for IBM Cloud Logs.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/cache"
	"github.com/tareqmamari/logs-mcp-server/internal/client"
	"github.com/tareqmamari/logs-mcp-server/internal/tracing"
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

// Annotations returns default annotations for tools.
// Tools should override this method to provide specific annotations.
func (t *BaseTool) Annotations() *mcp.ToolAnnotations {
	// Default: no specific annotations, let MCP use defaults
	return nil
}

// GetClient returns the API client, preferring context over stored client.
// This enables per-request client injection for future HTTP transport support
// while maintaining backward compatibility with the current STDIO mode.
func (t *BaseTool) GetClient(ctx context.Context) (*client.Client, error) {
	// First try context (future HTTP mode, testing)
	if c, err := GetClientFromContext(ctx); err == nil {
		return c, nil
	}

	// Fall back to stored client (current STDIO mode)
	if t.client != nil {
		return t.client, nil
	}

	return nil, ErrNoClientInContext
}

// ExecuteRequest executes an API request and returns the response
func (t *BaseTool) ExecuteRequest(ctx context.Context, req *client.Request) (map[string]interface{}, error) {
	// Start OpenTelemetry span for API call
	ctx, span := tracing.APISpan(ctx, req.Method, req.Path)
	defer span.End()

	apiClient, err := t.GetClient(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, fmt.Errorf("failed to get API client: %w", err)
	}

	resp, err := apiClient.Do(ctx, req)
	if err != nil {
		tracing.RecordError(span, err)
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
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Message:    errorMessage,
			RequestID:  requestID,
			Details:    apiError,
		}
		tracing.RecordError(span, apiErr)
		return nil, apiErr
	}

	// Mark span as successful
	tracing.SetSuccess(span)

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

// CacheHelper provides cache operations scoped to the current user/instance
type CacheHelper struct {
	userID     string
	instanceID string
	manager    *cache.Manager
}

// GetCacheHelper returns a cache helper for the current session
func GetCacheHelper() *CacheHelper {
	session := GetSession()
	return &CacheHelper{
		userID:     session.UserID,
		instanceID: session.InstanceID,
		manager:    cache.GetManager(),
	}
}

// Get retrieves a cached value for a tool
func (h *CacheHelper) Get(toolName, cacheKey string) (interface{}, bool) {
	return h.manager.Get(h.userID, h.instanceID, toolName, cacheKey)
}

// Set stores a value in the cache for a tool
func (h *CacheHelper) Set(toolName, cacheKey string, value interface{}) {
	h.manager.Set(h.userID, h.instanceID, toolName, cacheKey, value)
}

// InvalidateTool removes all cache entries for a specific tool
func (h *CacheHelper) InvalidateTool(toolName string) {
	h.manager.InvalidateTool(h.userID, h.instanceID, toolName)
}

// InvalidateRelated invalidates cache for related tools after a mutation
func (h *CacheHelper) InvalidateRelated(mutationTool string) {
	h.manager.InvalidateRelated(h.userID, h.instanceID, mutationTool)
}

// Clear removes all cache entries for the current user
func (h *CacheHelper) Clear() {
	h.manager.ClearUser(h.userID, h.instanceID)
}

// Stats returns cache statistics for the current user
func (h *CacheHelper) Stats() map[string]interface{} {
	return h.manager.Stats(h.userID, h.instanceID)
}

// IsEnabled returns whether caching is enabled
func (h *CacheHelper) IsEnabled() bool {
	return h.manager.IsEnabled()
}
