// Package tools provides the MCP tool implementations for IBM Cloud Logs.
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/cache"
	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
	mcperrors "github.com/tareqmamari/cloud-logs-mcp/internal/errors"
	"github.com/tareqmamari/cloud-logs-mcp/internal/tracing"
)

// Tool timeout constants for different operation types.
// These provide sensible defaults based on expected execution times.
const (
	// DefaultToolTimeout is the fallback timeout when no specific timeout is set.
	// Returns 0 to indicate "use client/server default".
	DefaultToolTimeout = 0

	// DefaultListTimeout for list operations (standard API calls)
	DefaultListTimeout = 30 * time.Second

	// DefaultGetTimeout for single resource fetch operations
	DefaultGetTimeout = 15 * time.Second

	// DefaultCreateTimeout for create/update operations
	DefaultCreateTimeout = 30 * time.Second

	// DefaultDeleteTimeout for delete operations (quick)
	DefaultDeleteTimeout = 15 * time.Second

	// DefaultWorkflowTimeout for multi-step workflow operations
	DefaultWorkflowTimeout = 90 * time.Second

	// DefaultHealthCheckTimeout for health check operations
	DefaultHealthCheckTimeout = 45 * time.Second
)

// BaseTool provides common functionality for all tools
type BaseTool struct {
	client client.Doer
	logger *zap.Logger
}

// NewBaseTool creates a new BaseTool
func NewBaseTool(c client.Doer, logger *zap.Logger) *BaseTool {
	return &BaseTool{
		client: c,
		logger: logger,
	}
}

// Annotations returns default annotations for tools.
// Tools should override this method to provide specific annotations.
func (t *BaseTool) Annotations() *mcp.ToolAnnotations {
	// Default: no specific annotations, let MCP use defaults
	return nil
}

// DefaultTimeout returns the default timeout for this tool.
// Returns 0 to use the client/server default timeout.
// Tools should override this method to provide specific timeouts.
func (t *BaseTool) DefaultTimeout() time.Duration {
	return DefaultToolTimeout
}

// GetClient returns the API client, preferring context over stored client.
// This enables per-request client injection for future HTTP transport support
// while maintaining backward compatibility with the current STDIO mode.
func (t *BaseTool) GetClient(ctx context.Context) (client.Doer, error) {
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
		// The client classifies non-2xx statuses as *mcperrors.StructuredError
		// (returned alongside the response). Translate it into *tools.APIError
		// so downstream errors.As matching (HandleGetError/HandleQueryError:
		// IsNotFound/IsTimeout handling, list_* suggestions, request IDs)
		// keeps working exactly as it did when this method built the APIError
		// from the raw status code itself.
		var structuredErr *mcperrors.StructuredError
		if errors.As(err, &structuredErr) && structuredErr.StatusCode >= 400 {
			apiErr := newAPIErrorFromResponse(structuredErr.StatusCode, resp)
			tracing.RecordError(span, apiErr)
			return nil, apiErr
		}
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	// Check for error status codes. The real client returns non-2xx statuses
	// as errors (handled above); this branch remains for Doer implementations
	// that surface the raw response without a classified error.
	if resp.StatusCode >= 400 {
		apiErr := newAPIErrorFromResponse(resp.StatusCode, resp)
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

// newAPIErrorFromResponse builds a *tools.APIError for a non-2xx status,
// pulling the message/details from the response body and the request ID from
// the response headers when a response is available.
func newAPIErrorFromResponse(statusCode int, resp *client.Response) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		Message:    fmt.Sprintf("API error (HTTP %d)", statusCode),
	}
	if resp == nil {
		return apiErr
	}

	// Extract request ID from response headers (IBM Cloud uses X-Request-ID or X-Correlation-ID)
	requestID := resp.Headers.Get("X-Request-ID")
	if requestID == "" {
		requestID = resp.Headers.Get("X-Correlation-ID")
	}
	if requestID == "" {
		requestID = resp.Headers.Get("X-Global-Transaction-ID")
	}
	apiErr.RequestID = requestID

	var apiError map[string]interface{}
	if err := json.Unmarshal(resp.Body, &apiError); err == nil {
		apiErr.Message = fmt.Sprintf("API error (HTTP %d): %v", statusCode, apiError)
		apiErr.Details = apiError
	} else {
		apiErr.Message = fmt.Sprintf("API error (HTTP %d): %s", statusCode, string(resp.Body))
	}
	return apiErr
}

// SSEParseResult holds the classified output from parsing an SSE response.
// It separates log entries from control messages (warnings, errors, query IDs)
// so consumers get clean data without having to re-inspect every event.
type SSEParseResult struct {
	Events    []interface{}          // Flattened log entries from "result" messages
	Warnings  []string               // Warning messages (compile, time range, result limit)
	Errors    []string               // Error messages from the API
	QueryID   string                 // Query ID if provided by the API
	Metadata  map[string]interface{} // Any additional metadata
	Total     int                    // Total SSE data lines seen (before cap)
	Truncated bool                   // Whether events were capped
}

// parseSSEResponse attempts to parse a response body as Server-Sent Events.
// Returns nil if the body doesn't look like SSE.
//
// IBM Cloud Logs /v1/query returns SSE with four message types:
//   - result  → contains results[] array with log entries
//   - warning → compile warnings, time range adjustments, result limit warnings
//   - error   → query execution errors
//   - query_id → internal query identifier
//
// Each log entry in results[] has:
//   - labels[]    → key/value pairs (applicationname, subsystemname, ...)
//   - metadata[]  → key/value pairs (timestamp, severity, ...)
//   - user_data   → JSON string containing the actual log payload
//
// The parser extracts individual log entries, parses user_data JSON strings,
// and flattens labels/metadata into the entry for downstream consumption.
func parseSSEResponse(body []byte) map[string]interface{} {
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "data:") {
		return nil
	}

	parsed := parseSSEMessages(bodyStr, MaxSSEEvents)
	if len(parsed.Events) == 0 && len(parsed.Errors) == 0 && len(parsed.Warnings) == 0 && parsed.QueryID == "" {
		return nil
	}

	result := map[string]interface{}{
		"events": parsed.Events,
	}

	if parsed.Truncated {
		result["_truncated"] = true
		result["_total_events"] = parsed.Total
		result["_shown_events"] = len(parsed.Events)
	}

	if len(parsed.Warnings) > 0 {
		result["_warnings"] = parsed.Warnings
	}

	if len(parsed.Errors) > 0 {
		result["_errors"] = parsed.Errors
	}

	if parsed.QueryID != "" {
		result["_query_id"] = parsed.QueryID
	}

	return result
}

// parseSSEMessages processes raw SSE text and classifies each data line
// by its message type. maxEvents caps how many log entries are kept in memory.
func parseSSEMessages(bodyStr string, maxEvents int) *SSEParseResult {
	parsed := &SSEParseResult{}
	lines := strings.Split(bodyStr, "\n")

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		dataStr := strings.TrimPrefix(line, "data: ")
		if dataStr == "" {
			continue
		}

		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(dataStr), &msg); err != nil {
			continue
		}

		classifySSEMessage(msg, parsed, maxEvents)
	}

	return parsed
}

// classifySSEMessage routes a single parsed SSE JSON object to the correct
// handler based on which top-level key is present.
func classifySSEMessage(msg map[string]interface{}, parsed *SSEParseResult, maxEvents int) {
	switch {
	case msg["result"] != nil:
		handleSSEResult(msg["result"], parsed, maxEvents)
	case msg["error"] != nil:
		handleSSEError(msg["error"], parsed)
	case msg["warning"] != nil:
		handleSSEWarning(msg["warning"], parsed)
	case msg["query_id"] != nil:
		handleSSEQueryID(msg["query_id"], parsed)
	default:
		// Unknown message type — treat as a raw event for backward compatibility
		parsed.Total++
		if len(parsed.Events) < maxEvents {
			parsed.Events = append(parsed.Events, msg)
		} else {
			parsed.Truncated = true
		}
	}
}

// handleSSEResult extracts log entries from a "result" message.
// Each result message contains a results[] array of log entries.
func handleSSEResult(resultVal interface{}, parsed *SSEParseResult, maxEvents int) {
	resultObj, ok := resultVal.(map[string]interface{})
	if !ok {
		return
	}

	results, ok := resultObj["results"].([]interface{})
	if !ok {
		// No nested results — treat the result object itself as an event
		parsed.Total++
		if len(parsed.Events) < maxEvents {
			parsed.Events = append(parsed.Events, resultObj)
		} else {
			parsed.Truncated = true
		}
		return
	}

	for _, r := range results {
		parsed.Total++
		if len(parsed.Events) >= maxEvents {
			parsed.Truncated = true
			continue
		}

		entry, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		parsed.Events = append(parsed.Events, flattenLogEntry(entry))
	}
}

// flattenLogEntry converts a raw API log entry into a flat, LLM-friendly map.
//
// Input format (from API):
//
//	{
//	  "labels": [{"key": "applicationname", "value": "myapp"}, ...],
//	  "metadata": [{"key": "timestamp", "value": "2024-..."}, ...],
//	  "user_data": "{\"message\": \"hello\", ...}"
//	}
//
// Output format:
//
//	{
//	  "applicationname": "myapp",
//	  "timestamp": "2024-...",
//	  "severity": "5",
//	  "user_data": {"message": "hello", ...}   // parsed JSON object
//	}
func flattenLogEntry(entry map[string]interface{}) map[string]interface{} {
	flat := make(map[string]interface{})

	// Flatten labels array → top-level keys
	if labels, ok := entry["labels"].([]interface{}); ok {
		for _, l := range labels {
			lm, ok := l.(map[string]interface{})
			if !ok {
				continue
			}
			key, _ := lm["key"].(string)
			value, _ := lm["value"].(string)
			if key != "" && value != "" {
				flat[key] = value
			}
		}
	}

	// Flatten metadata array → top-level keys
	if metadata, ok := entry["metadata"].([]interface{}); ok {
		for _, m := range metadata {
			mm, ok := m.(map[string]interface{})
			if !ok {
				continue
			}
			key, _ := mm["key"].(string)
			value, _ := mm["value"].(string)
			if key != "" {
				flat[key] = value
			}
		}
	}

	// Parse user_data JSON string into a proper object
	if userData, ok := entry["user_data"].(string); ok && userData != "" {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(userData), &parsed); err == nil {
			flat["user_data"] = parsed
		} else {
			// Not valid JSON — keep as string
			flat["user_data"] = userData
		}
	}

	// Preserve any other top-level fields not already handled
	for k, v := range entry {
		if k == "labels" || k == "metadata" || k == "user_data" {
			continue
		}
		flat[k] = v
	}

	return flat
}

// handleSSEError extracts error messages from an "error" SSE message.
//
// API format:
//
//	{"error": {"message": "Failed to run the query ...", "code": {"rate_limit_reached": {}}}}
func handleSSEError(errorVal interface{}, parsed *SSEParseResult) {
	switch e := errorVal.(type) {
	case map[string]interface{}:
		if msg, ok := e["message"].(string); ok {
			parsed.Errors = append(parsed.Errors, msg)
		} else {
			// Serialize the whole error object as fallback
			if b, err := json.Marshal(e); err == nil {
				parsed.Errors = append(parsed.Errors, string(b))
			}
		}
	case string:
		parsed.Errors = append(parsed.Errors, e)
	}
}

// handleSSEWarning extracts warning messages from a "warning" SSE message.
//
// The warning object may contain one of:
//   - compile_warning: {"warning_message": "..."}
//   - time_range_warning: {"warning_message": "...", "start_date": "...", "end_date": "..."}
//   - number_of_results_limit_warning: {"number_of_results_limit": 10000}
func handleSSEWarning(warningVal interface{}, parsed *SSEParseResult) {
	warnObj, ok := warningVal.(map[string]interface{})
	if !ok {
		if s, ok := warningVal.(string); ok {
			parsed.Warnings = append(parsed.Warnings, s)
		}
		return
	}

	// Check each known warning sub-type
	warningTypes := []string{"compile_warning", "time_range_warning", "number_of_results_limit_warning"}
	for _, wt := range warningTypes {
		sub, ok := warnObj[wt].(map[string]interface{})
		if !ok {
			continue
		}
		if msg, ok := sub["warning_message"].(string); ok {
			parsed.Warnings = append(parsed.Warnings, msg)
		} else if limit, ok := sub["number_of_results_limit"].(float64); ok {
			parsed.Warnings = append(parsed.Warnings, fmt.Sprintf("Results capped at %d by the API", int(limit)))
		}
	}

	// If no known sub-type matched, serialize the whole warning
	if len(parsed.Warnings) == 0 {
		if b, err := json.Marshal(warnObj); err == nil {
			parsed.Warnings = append(parsed.Warnings, string(b))
		}
	}
}

// handleSSEQueryID extracts the query identifier from a "query_id" SSE message.
//
// API format: {"query_id": {"query_id": "4rwoNx1XNcc"}}
func handleSSEQueryID(queryIDVal interface{}, parsed *SSEParseResult) {
	switch q := queryIDVal.(type) {
	case map[string]interface{}:
		if id, ok := q["query_id"].(string); ok {
			parsed.QueryID = id
		}
	case string:
		parsed.QueryID = q
	}
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

// GetCacheHelperFromContext returns a cache helper using the session from the given context.
func GetCacheHelperFromContext(ctx context.Context) *CacheHelper {
	session := GetSessionFromContext(ctx)
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
