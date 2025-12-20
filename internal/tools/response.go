package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// Response size limits
const (
	// MaxResultSize is the maximum size of tool results in bytes (100KB for Claude Desktop compatibility)
	// Claude Desktop's context compaction can fail with very large tool results, so we use a conservative limit
	// This is much lower than MCP's 1MB limit to ensure reliable operation
	MaxResultSize = 100 * 1024

	// FinalResponseLimit is the absolute maximum size for the final response text before sending to MCP
	// This ensures we never exceed limits that could cause "compaction failed" errors in Claude Desktop
	FinalResponseLimit = 150 * 1024

	// MaxSSEEvents is the maximum number of SSE events to parse from a query response
	// This prevents memory issues and keeps results manageable for LLM context
	MaxSSEEvents = 50

	// TruncationBufferSize is the buffer size reserved for warning messages when truncating results
	TruncationBufferSize = 500

	// WarningMessageBuffer is the buffer reserved for warning/metadata in truncated results
	WarningMessageBuffer = 1000

	// MinArraySizeForTruncation is the minimum array size before truncation is attempted
	MinArraySizeForTruncation = 10

	// MaxSummaryItems is the maximum number of items to show in result summaries
	MaxSummaryItems = 10

	// MaxTopValues is the maximum number of top values to extract from query results
	MaxTopValues = 5
)

// metadataFieldsToRemove contains metadata keys that add noise without LLM value
var metadataFieldsToRemove = map[string]bool{
	"logid": true, "branchid": true, "templateid": true, "priorityclass": true,
	"processingoutputtimestampnanos": true, "processingoutputtimestampmicros": true,
	"timestampmicros": true, "ingresstimestamp": true,
}

// ResponseMeta contains verification traces for GRPO-style self-correction.
// Modern reasoning models can use these signals to verify execution and adjust behavior.
type ResponseMeta struct {
	ExecutionMs  int64  `json:"execution_ms"`           // Time taken to execute the tool
	ResultHash   string `json:"result_hash"`            // SHA256 hash of result for idempotency verification
	ResultCount  int    `json:"result_count"`           // Number of items in result
	Truncated    bool   `json:"truncated"`              // Whether result was truncated
	ToolName     string `json:"tool_name,omitempty"`    // Tool that generated this response
	RequestHash  string `json:"request_hash,omitempty"` // Hash of request for deduplication
	CacheHit     bool   `json:"cache_hit,omitempty"`    // Whether result came from cache
	ErrorCode    string `json:"error_code,omitempty"`   // Structured error code if failed
	RetryAllowed bool   `json:"retry_allowed"`          // Whether retry is safe for this operation
}

// NewResponseMeta creates a new ResponseMeta with execution timing started
func NewResponseMeta(toolName string) *ResponseMeta {
	return &ResponseMeta{
		ToolName:     toolName,
		RetryAllowed: true, // Default to allowing retries
	}
}

// computeResultHash generates a SHA256 hash of the result for verification
func computeResultHash(result map[string]interface{}) string {
	// Remove volatile fields before hashing
	stableResult := make(map[string]interface{})
	for k, v := range result {
		if k != "_meta" && k != "_rate_limit" && !strings.HasPrefix(k, "_") {
			stableResult[k] = v
		}
	}

	data, err := json.Marshal(stableResult)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8]) // First 8 bytes = 16 hex chars
}

// AddVerificationMeta adds GRPO verification traces to a result map.
// This enables reasoning models to verify execution and detect inconsistencies.
func AddVerificationMeta(result map[string]interface{}, toolName string, startTime time.Time, truncated bool) {
	meta := &ResponseMeta{
		ExecutionMs:  time.Since(startTime).Milliseconds(),
		ResultHash:   computeResultHash(result),
		ResultCount:  countItems(result),
		Truncated:    truncated,
		ToolName:     toolName,
		RetryAllowed: !isDestructiveTool(toolName),
	}

	result["_meta"] = meta
}

// isDestructiveTool returns true if the tool performs destructive operations
func isDestructiveTool(toolName string) bool {
	destructiveTools := map[string]bool{
		"delete_alert":            true,
		"delete_dashboard":        true,
		"delete_policy":           true,
		"delete_e2m":              true,
		"delete_enrichment":       true,
		"delete_view":             true,
		"delete_outgoing_webhook": true,
		"delete_data_access_rule": true,
		"delete_stream":           true,
		"cancel_background_query": true,
	}
	return destructiveTools[toolName]
}

// getMapKeys returns a sorted list of keys from a map for debugging purposes
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// CleanQueryResults removes unnecessary fields from query results to reduce response size
// and improve LLM comprehension. Transforms verbose API format into compact, actionable data.
func CleanQueryResults(result map[string]interface{}) map[string]interface{} {
	events, ok := result["events"].([]interface{})
	if !ok || len(events) == 0 {
		return result
	}

	cleanedEvents := make([]interface{}, 0, len(events))
	for _, event := range events {
		eventMap, ok := event.(map[string]interface{})
		if !ok {
			cleanedEvents = append(cleanedEvents, event)
			continue
		}

		// Skip query_id events (SSE metadata, not actual log data)
		if _, hasQueryID := eventMap["query_id"]; hasQueryID && len(eventMap) == 1 {
			continue
		}

		// Handle the nested result structure from API
		if resultObj, ok := eventMap["result"].(map[string]interface{}); ok {
			if results, ok := resultObj["results"].([]interface{}); ok {
				for _, r := range results {
					if rMap, ok := r.(map[string]interface{}); ok {
						transformed := transformLogEntry(rMap)
						if len(transformed) > 0 {
							cleanedEvents = append(cleanedEvents, transformed)
						}
					}
				}
			}
		} else {
			// Direct event format (including aggregation results)
			transformed := transformLogEntry(eventMap)
			if len(transformed) > 0 {
				cleanedEvents = append(cleanedEvents, transformed)
			}
		}
	}

	// Create compact result
	cleaned := map[string]interface{}{
		"logs": cleanedEvents,
	}

	// Preserve query metadata
	if meta, ok := result["_query_metadata"]; ok {
		cleaned["_query_metadata"] = meta
	}

	return cleaned
}

// transformLogEntry converts verbose log entry to compact format
// Handles multiple response formats from IBM Cloud Logs:
// 1. Array format: labels/metadata as []interface{} with key/value pairs
// 2. Map format: labels/metadata as map[string]interface{} with direct fields
// 3. Flat format: direct fields at the root level (aggregation results)
func transformLogEntry(entry map[string]interface{}) map[string]interface{} {
	compact := make(map[string]interface{})

	// Extract essential labels (app, subsystem)
	// Handle array format: [{"key": "applicationname", "value": "myapp"}, ...]
	if labels, ok := entry["labels"].([]interface{}); ok {
		for _, l := range labels {
			if lm, ok := l.(map[string]interface{}); ok {
				key, _ := lm["key"].(string)
				value, _ := lm["value"].(string)
				if value == "" {
					continue
				}
				switch key {
				case "applicationname":
					compact["app"] = value
				case "subsystemname":
					compact["subsystem"] = value
				}
			}
		}
	} else if labels, ok := entry["labels"].(map[string]interface{}); ok {
		// Handle map format: {"applicationname": "myapp", "subsystemname": "api"}
		if app, ok := labels["applicationname"].(string); ok && app != "" {
			compact["app"] = app
		}
		if sub, ok := labels["subsystemname"].(string); ok && sub != "" {
			compact["subsystem"] = sub
		}
	}

	// Extract essential metadata (timestamp, severity)
	// Handle array format: [{"key": "timestamp", "value": "2024-..."}, ...]
	if metadata, ok := entry["metadata"].([]interface{}); ok {
		for _, m := range metadata {
			if mm, ok := m.(map[string]interface{}); ok {
				key, _ := mm["key"].(string)
				value, _ := mm["value"].(string)
				keyLower := strings.ToLower(key)
				if metadataFieldsToRemove[keyLower] {
					continue
				}
				switch key {
				case "timestamp":
					compact["time"] = value
				case "severity":
					compact["severity"] = value
				}
			}
		}
	} else if metadata, ok := entry["metadata"].(map[string]interface{}); ok {
		// Handle map format: {"timestamp": "2024-...", "severity": "ERROR"}
		if ts, ok := metadata["timestamp"].(string); ok && ts != "" {
			compact["time"] = ts
		}
		if sev, ok := metadata["severity"].(string); ok && sev != "" {
			compact["severity"] = sev
		}
		// Also check for numeric severity
		if sevNum, ok := metadata["severity"].(float64); ok {
			compact["severity"] = severityNumToName(int(sevNum))
		}
	}

	// Parse and extract from user_data
	// Handle string format (JSON that needs parsing)
	if userData, ok := entry["user_data"].(string); ok && userData != "" {
		var ud map[string]interface{}
		if err := json.Unmarshal([]byte(userData), &ud); err == nil {
			extractUserData(ud, compact)
		}
	} else if userData, ok := entry["user_data"].(map[string]interface{}); ok {
		// Handle already-parsed map format
		extractUserData(userData, compact)
	}

	// For flat/aggregation results, try extracting common fields directly from entry
	// This handles DataPrime aggregation results that don't use the nested structure
	// Always try flat extraction if no message was found yet
	if compact["message"] == nil {
		extractFlatFields(entry, compact)
	}

	return compact
}

// severityNumToName converts numeric severity to human-readable name
func severityNumToName(sev int) string {
	severityNames := map[int]string{
		1: "Debug",
		2: "Verbose",
		3: "Info",
		4: "Warning",
		5: "Error",
		6: "Critical",
	}
	if name, ok := severityNames[sev]; ok {
		return name
	}
	return fmt.Sprintf("Level %d", sev)
}

// extractFlatFields extracts fields from flat/aggregation result entries
func extractFlatFields(entry map[string]interface{}, compact map[string]interface{}) {
	// Common timestamp field names (including DataPrime reference formats from groupby)
	for _, tsField := range []string{"timestamp", "@timestamp", "time", "_time", "ts", "$m.timestamp"} {
		if ts, ok := entry[tsField].(string); ok && ts != "" {
			compact["time"] = ts
			break
		}
	}

	// Common message field names
	for _, msgField := range []string{"message", "msg", "text", "_message", "log"} {
		if msg, ok := entry[msgField].(string); ok && msg != "" {
			compact["message"] = msg
			break
		}
	}

	// Common severity field names (including DataPrime reference formats from groupby)
	for _, sevField := range []string{"severity", "level", "log_level", "loglevel", "$m.severity"} {
		if sev, ok := entry[sevField].(string); ok && sev != "" {
			compact["severity"] = sev
			break
		}
		if sevNum, ok := entry[sevField].(float64); ok {
			compact["severity"] = severityNumToName(int(sevNum))
			break
		}
	}

	// Common application field names (including DataPrime reference formats from groupby)
	for _, appField := range []string{"applicationname", "application", "app", "service", "app_name", "$l.applicationname"} {
		if app, ok := entry[appField].(string); ok && app != "" {
			compact["app"] = app
			break
		}
	}

	// Common subsystem field names (including DataPrime reference formats from groupby)
	for _, subField := range []string{"subsystemname", "subsystem", "component", "module", "$l.subsystemname"} {
		if sub, ok := entry[subField].(string); ok && sub != "" {
			compact["subsystem"] = sub
			break
		}
	}

	// If no message was found, include remaining non-standard fields as the message
	// This ensures aggregation results (count, sum, avg, etc.) are visible
	if compact["message"] == nil && len(entry) > 0 {
		// Fields that we've already extracted to standard names
		extractedFields := map[string]bool{
			"timestamp": true, "@timestamp": true, "time": true, "_time": true, "ts": true, "$m.timestamp": true,
			"message": true, "msg": true, "text": true, "_message": true, "log": true,
			"severity": true, "level": true, "log_level": true, "loglevel": true, "$m.severity": true,
			"applicationname": true, "application": true, "app": true, "service": true, "app_name": true, "$l.applicationname": true,
			"subsystemname": true, "subsystem": true, "component": true, "module": true, "$l.subsystemname": true,
			"labels": true, "metadata": true, "user_data": true,
		}

		var parts []string
		for k, v := range entry {
			// Skip internal/metadata fields and already-extracted fields
			if strings.HasPrefix(k, "_") || k == "query_id" || extractedFields[k] {
				continue
			}
			switch val := v.(type) {
			case string:
				if val != "" {
					parts = append(parts, fmt.Sprintf("%s=%s", k, val))
				}
			case float64:
				parts = append(parts, fmt.Sprintf("%s=%.0f", k, val))
			case int:
				parts = append(parts, fmt.Sprintf("%s=%d", k, val))
			case bool:
				parts = append(parts, fmt.Sprintf("%s=%v", k, val))
			}
		}
		if len(parts) > 0 {
			sort.Strings(parts)
			compact["message"] = strings.Join(parts, ", ")
		}
	}
}

// extractUserData extracts essential fields from parsed user_data
func extractUserData(ud map[string]interface{}, compact map[string]interface{}) {
	extractEventFields(ud, compact)
	extractDirectFields(ud, compact)
	extractAggregationFields(ud, compact)
	buildAggregationMessage(ud, compact)
}

// extractEventFields extracts fields from the nested event object
func extractEventFields(ud map[string]interface{}, compact map[string]interface{}) {
	event, ok := ud["event"].(map[string]interface{})
	if !ok {
		return
	}
	if msg, ok := event["_message"].(string); ok {
		compact["message"] = msg
	}
	if sql, ok := event["sql"].(string); ok {
		compact["sql"] = sql
	}
	if execMs, ok := event["execMs"].(float64); ok {
		compact["exec_ms"] = int(execMs)
	}
}

// extractDirectFields extracts commonly used direct fields from user_data
func extractDirectFields(ud map[string]interface{}, compact map[string]interface{}) {
	// Direct message field (if not already set from event)
	if compact["message"] == nil {
		if msg, ok := ud["message"].(string); ok {
			compact["message"] = msg
		}
	}

	// Log level
	if level, ok := ud["level"].(string); ok {
		compact["level"] = level
	}

	// Logger name (shortened)
	if logger, ok := ud["logger_name"].(string); ok {
		parts := strings.Split(logger, ".")
		if len(parts) > 2 {
			compact["logger"] = parts[len(parts)-2] + "." + parts[len(parts)-1]
		} else {
			compact["logger"] = logger
		}
	}

	// Trace/span IDs
	if traceID, ok := ud["trace_id"].(string); ok {
		compact["trace_id"] = traceID
	}
	if spanID, ok := ud["span_id"].(string); ok {
		compact["span_id"] = spanID
	}
}

// extractAggregationFields extracts app, subsystem, severity from aggregation results
func extractAggregationFields(ud map[string]interface{}, compact map[string]interface{}) {
	// App name from various field names
	if compact["app"] == nil {
		compact["app"] = findFirstString(ud, "applicationname", "application", "app", "service", "app_name")
	}

	// Subsystem from various field names
	if compact["subsystem"] == nil {
		compact["subsystem"] = findFirstString(ud, "subsystemname", "subsystem", "component", "module")
	}

	// Severity (string or numeric)
	if compact["severity"] == nil {
		for _, field := range []string{"severity", "level", "log_level"} {
			if sev, ok := ud[field].(string); ok && sev != "" {
				compact["severity"] = sev
				return
			}
			if sevNum, ok := ud[field].(float64); ok {
				compact["severity"] = severityNumToName(int(sevNum))
				return
			}
		}
	}
}

// findFirstString returns the first non-empty string value found for any of the given keys
func findFirstString(m map[string]interface{}, keys ...string) interface{} {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return nil
}

// alreadyExtractedFields are fields we've already extracted - skip when building aggregation message
var alreadyExtractedFields = map[string]bool{
	"applicationname": true, "application": true, "app": true, "service": true, "app_name": true,
	"subsystemname": true, "subsystem": true, "component": true, "module": true,
	"severity": true, "level": true, "log_level": true,
	"message": true, "msg": true, "text": true,
	"event": true, "logger_name": true, "trace_id": true, "span_id": true,
}

// buildAggregationMessage creates a message from aggregated values when no message exists
func buildAggregationMessage(ud map[string]interface{}, compact map[string]interface{}) {
	if compact["message"] != nil {
		return
	}

	var parts []string
	for k, v := range ud {
		if alreadyExtractedFields[k] {
			continue
		}
		switch val := v.(type) {
		case float64:
			if val == float64(int(val)) {
				parts = append(parts, fmt.Sprintf("%s=%d", k, int(val)))
			} else {
				parts = append(parts, fmt.Sprintf("%s=%.2f", k, val))
			}
		case string:
			if len(val) <= 100 {
				parts = append(parts, fmt.Sprintf("%s=%s", k, val))
			}
		}
	}

	if len(parts) > 0 {
		sort.Strings(parts)
		compact["message"] = strings.Join(parts, ", ")
	}
}

// FormatResponse formats the response as a text/content for MCP
// If the result exceeds MaxResultSize, it will be truncated with pagination hints
func (t *BaseTool) FormatResponse(result map[string]interface{}) (*mcp.CallToolResult, error) {
	// Handle empty result - len(nil map) is 0, so this covers both nil and empty
	if len(result) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "(no data returned)",
				},
			},
		}, nil
	}

	// Pretty print JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		// Return a valid CallToolResult with error message instead of nil
		// This prevents "compaction failed" errors in Claude Desktop
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Error formatting response: %v\n\nRaw data keys: %v", err, getMapKeys(result)),
				},
			},
			IsError: true,
		}, nil
	}

	responseText := string(jsonBytes)

	// Check if response exceeds size limit
	if len(jsonBytes) > MaxResultSize {
		// Try to truncate intelligently by reducing the data
		_, truncatedBytes := truncateResult(result, MaxResultSize)
		if truncatedBytes != nil {
			responseText = string(truncatedBytes)
		} else {
			// Fallback: hard truncate the JSON string
			responseText = string(jsonBytes[:MaxResultSize-TruncationBufferSize])
		}

		totalItems := countItems(result)
		shownItems := countItemsFromBytes(truncatedBytes)

		// Add pagination guidance with truncation warning
		warningMsg := fmt.Sprintf("\n\n---\n⚠️ RESULT TRUNCATED: Showing %d of %d items (result was %d bytes, limit is %d bytes).\n\n"+
			"**To get complete results:**\n\n"+
			"1. **Use `summary_only: true`** - Get statistical overview without raw events\n"+
			"2. **Add filters** - Narrow by app, severity, or time range:\n"+
			"   - `filter $l.applicationname == 'your-app'`\n"+
			"   - `filter $m.severity >= ERROR`\n"+
			"3. **Use smaller limits** - `limit 20` instead of large numbers\n"+
			"4. **Split time range** - Query smaller time windows",
			shownItems, totalItems, len(jsonBytes), MaxResultSize)
		responseText += warningMsg

		t.logger.Warn("Result truncated due to size limit - pagination recommended",
			zap.Int("original_size", len(jsonBytes)),
			zap.Int("truncated_size", len(responseText)),
			zap.Int("total_items", totalItems),
			zap.Int("shown_items", shownItems),
		)
	}

	// Final safety check
	responseText = ensureResponseLimit(responseText, t.logger)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: responseText,
			},
		},
	}, nil
}

// truncateResult attempts to intelligently truncate the result by reducing array sizes
func truncateResult(result map[string]interface{}, maxSize int) (map[string]interface{}, []byte) {
	// Make a copy to avoid modifying the original
	truncated := make(map[string]interface{})
	for k, v := range result {
		truncated[k] = v
	}

	// Find arrays and truncate them
	for key, val := range truncated {
		if arr, ok := val.([]interface{}); ok && len(arr) > MinArraySizeForTruncation {
			// Binary search for the right size
			low, high := MinArraySizeForTruncation, len(arr)
			bestSize := MinArraySizeForTruncation

			for low <= high {
				mid := (low + high) / 2
				truncated[key] = arr[:mid]
				testBytes, err := json.MarshalIndent(truncated, "", "  ")
				if err != nil {
					break
				}

				if len(testBytes) <= maxSize-WarningMessageBuffer {
					bestSize = mid
					low = mid + 1
				} else {
					high = mid - 1
				}
			}

			truncated[key] = arr[:bestSize]
			truncated["_truncated_info"] = map[string]interface{}{
				"field":          key,
				"original_count": len(arr),
				"shown_count":    bestSize,
			}
		}
	}

	// Also handle nested "events" array (common in query results)
	if events, ok := truncated["events"].([]interface{}); ok && len(events) > MinArraySizeForTruncation {
		low, high := MinArraySizeForTruncation, len(events)
		bestSize := MinArraySizeForTruncation

		for low <= high {
			mid := (low + high) / 2
			truncated["events"] = events[:mid]
			testBytes, err := json.MarshalIndent(truncated, "", "  ")
			if err != nil {
				break
			}

			if len(testBytes) <= maxSize-WarningMessageBuffer {
				bestSize = mid
				low = mid + 1
			} else {
				high = mid - 1
			}
		}

		truncated["events"] = events[:bestSize]
		truncated["_truncated_info"] = map[string]interface{}{
			"field":          "events",
			"original_count": len(events),
			"shown_count":    bestSize,
		}
	}

	// Marshal the truncated result
	truncatedBytes, err := json.MarshalIndent(truncated, "", "  ")
	if err != nil {
		return nil, nil
	}

	return truncated, truncatedBytes
}

// countItems counts the number of items in arrays within the result
func countItems(result map[string]interface{}) int {
	count := 0
	for _, val := range result {
		if arr, ok := val.([]interface{}); ok {
			count += len(arr)
		}
	}
	if count == 0 {
		count = 1 // At least one item (the result itself)
	}
	return count
}

// countItemsFromBytes counts items from JSON bytes (for truncated results)
func countItemsFromBytes(data []byte) int {
	if data == nil {
		return 0
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0
	}
	return countItems(result)
}

// GenerateResultSummary creates an AI-friendly summary of query/list results
// This helps LLMs quickly understand the data without parsing large JSON
func GenerateResultSummary(result map[string]interface{}, resultType string) string {
	var summary strings.Builder

	// Handle query results with events
	if events, ok := result["events"].([]interface{}); ok {
		summary.WriteString("## Query Results Summary\n\n")
		summary.WriteString(fmt.Sprintf("**Total Results:** %d log entries\n\n", len(events)))

		if len(events) > 0 {
			// Analyze severity distribution
			severityDist := analyzeSeverityDistribution(events)
			if len(severityDist) > 0 {
				summary.WriteString("### Severity Distribution\n")
				for sev, count := range severityDist {
					summary.WriteString(fmt.Sprintf("- %s: %d\n", sev, count))
				}
				summary.WriteString("\n")
			}

			// Extract top applications
			topApps := extractTopValues(events, "applicationname", MaxTopValues)
			if len(topApps) > 0 {
				summary.WriteString("### Top Applications\n")
				for _, app := range topApps {
					summary.WriteString(fmt.Sprintf("- %s: %d entries\n", app.Value, app.Count))
				}
				summary.WriteString("\n")
			}

			// Extract top subsystems
			topSubs := extractTopValues(events, "subsystemname", MaxTopValues)
			if len(topSubs) > 0 {
				summary.WriteString("### Top Subsystems\n")
				for _, sub := range topSubs {
					summary.WriteString(fmt.Sprintf("- %s: %d entries\n", sub.Value, sub.Count))
				}
				summary.WriteString("\n")
			}

			// Time range
			timeRange := extractTimeRange(events)
			if timeRange != "" {
				summary.WriteString(fmt.Sprintf("### Time Range\n%s\n\n", timeRange))
			}
		}

		return summary.String()
	}

	// Handle list results (alerts, dashboards, policies, etc.)
	for _, val := range result {
		if arr, ok := val.([]interface{}); ok && len(arr) > 0 {
			summary.WriteString(fmt.Sprintf("## %s Summary\n\n", toTitleCase(resultType)))
			summary.WriteString(fmt.Sprintf("**Total Items:** %d\n\n", len(arr)))

			// Extract names/IDs for quick reference
			names := extractFieldValues(arr, []string{"name", "title", "id"}, MaxSummaryItems)
			if len(names) > 0 {
				summary.WriteString("### Items\n")
				for i, name := range names {
					summary.WriteString(fmt.Sprintf("%d. %s\n", i+1, name))
				}
				if len(arr) > MaxSummaryItems {
					summary.WriteString(fmt.Sprintf("... and %d more\n", len(arr)-MaxSummaryItems))
				}
				summary.WriteString("\n")
			}

			return summary.String()
		}
	}

	// For single item results
	if id, ok := result["id"].(string); ok {
		name, _ := result["name"].(string)
		if name == "" {
			name, _ = result["title"].(string)
		}
		summary.WriteString(fmt.Sprintf("## %s Details\n\n", toTitleCase(resultType)))
		summary.WriteString(fmt.Sprintf("**ID:** %s\n", id))
		if name != "" {
			summary.WriteString(fmt.Sprintf("**Name:** %s\n", name))
		}
		return summary.String()
	}

	return ""
}

// ValueCount represents a value and its occurrence count
type ValueCount struct {
	Value string
	Count int
}

// analyzeSeverityDistribution counts log entries by severity level
func analyzeSeverityDistribution(events []interface{}) map[string]int {
	severityNames := map[int]string{
		1: "Debug",
		2: "Verbose",
		3: "Info",
		4: "Warning",
		5: "Error",
		6: "Critical",
	}

	dist := make(map[string]int)
	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			// Try different severity field locations
			var severity int
			if sev, ok := eventMap["severity"].(float64); ok {
				severity = int(sev)
			} else if labels, ok := eventMap["labels"].(map[string]interface{}); ok {
				if sev, ok := labels["severity"].(float64); ok {
					severity = int(sev)
				}
			} else if metadata, ok := eventMap["metadata"].(map[string]interface{}); ok {
				if sev, ok := metadata["severity"].(float64); ok {
					severity = int(sev)
				}
			}

			if severity > 0 {
				name := severityNames[severity]
				if name == "" {
					name = fmt.Sprintf("Level %d", severity)
				}
				dist[name]++
			}
		}
	}
	return dist
}

// extractTopValues extracts the most common values for a given field
func extractTopValues(events []interface{}, fieldName string, limit int) []ValueCount {
	counts := make(map[string]int)

	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			value := findFieldValue(eventMap, fieldName)
			if value != "" {
				counts[value]++
			}
		}
	}

	// Convert to slice and sort by count
	var result []ValueCount
	for val, count := range counts {
		result = append(result, ValueCount{Value: val, Count: count})
	}

	// Sort by count (descending) - O(n log n)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

// findFieldValue searches for a field value in nested structures
func findFieldValue(data map[string]interface{}, fieldName string) string {
	// Direct lookup
	if val, ok := data[fieldName]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}

	// Check in labels
	if labels, ok := data["labels"].(map[string]interface{}); ok {
		if val, ok := labels[fieldName]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}

	// Check in metadata
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		if val, ok := metadata[fieldName]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}

	// Check in user_data (IBM Cloud Logs specific)
	if userData, ok := data["user_data"].(map[string]interface{}); ok {
		if val, ok := userData[fieldName]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}

	return ""
}

// extractTimeRange extracts the time range from events
func extractTimeRange(events []interface{}) string {
	if len(events) == 0 {
		return ""
	}

	var earliest, latest string

	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			timestamp := ""
			if ts, ok := eventMap["timestamp"].(string); ok {
				timestamp = ts
			} else if ts, ok := eventMap["@timestamp"].(string); ok {
				timestamp = ts
			} else if metadata, ok := eventMap["metadata"].(map[string]interface{}); ok {
				if ts, ok := metadata["timestamp"].(string); ok {
					timestamp = ts
				}
			}

			if timestamp != "" {
				if earliest == "" || timestamp < earliest {
					earliest = timestamp
				}
				if latest == "" || timestamp > latest {
					latest = timestamp
				}
			}
		}
	}

	if earliest != "" && latest != "" {
		return fmt.Sprintf("From: %s\nTo: %s", earliest, latest)
	}
	return ""
}

// extractLastTimestamp extracts the latest timestamp from events for pagination
// Returns the timestamp that can be used as start_date for the next page
func extractLastTimestamp(events []interface{}) string {
	if len(events) == 0 {
		return ""
	}

	var latest string

	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			timestamp := ""
			if ts, ok := eventMap["timestamp"].(string); ok {
				timestamp = ts
			} else if ts, ok := eventMap["@timestamp"].(string); ok {
				timestamp = ts
			} else if metadata, ok := eventMap["metadata"].(map[string]interface{}); ok {
				if ts, ok := metadata["timestamp"].(string); ok {
					timestamp = ts
				}
			}

			if timestamp != "" && (latest == "" || timestamp > latest) {
				latest = timestamp
			}
		}
	}

	return latest
}

// PaginationInfo contains metadata for paginating through results
type PaginationInfo struct {
	HasMore       bool   `json:"has_more"`
	TotalReturned int    `json:"total_returned"`
	LastTimestamp string `json:"last_timestamp,omitempty"`
	NextStartDate string `json:"next_start_date,omitempty"`
}

// extractPaginationInfo extracts pagination metadata from query results
func extractPaginationInfo(result map[string]interface{}, limit int, wasTruncated bool) *PaginationInfo {
	events, ok := result["events"].([]interface{})
	if !ok {
		return nil
	}

	info := &PaginationInfo{
		TotalReturned: len(events),
		HasMore:       wasTruncated || len(events) >= limit,
	}

	if lastTs := extractLastTimestamp(events); lastTs != "" {
		info.LastTimestamp = lastTs
		info.NextStartDate = lastTs
	}

	return info
}

// extractFieldValues extracts values from a list of items for given field names
func extractFieldValues(items []interface{}, fieldNames []string, limit int) []string {
	var values []string

	for _, item := range items {
		if len(values) >= limit {
			break
		}
		if itemMap, ok := item.(map[string]interface{}); ok {
			for _, fieldName := range fieldNames {
				if val, ok := itemMap[fieldName]; ok {
					if str, ok := val.(string); ok && str != "" {
						// If we have both name and ID, combine them
						id, hasID := itemMap["id"].(string)
						if fieldName == "name" && hasID {
							values = append(values, fmt.Sprintf("%s (ID: %s)", str, id))
						} else {
							values = append(values, str)
						}
						break
					}
				}
			}
		}
	}

	return values
}

// FormatResponseWithSuggestions formats the response with proactive suggestions
func (t *BaseTool) FormatResponseWithSuggestions(result map[string]interface{}, toolName string) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	// Handle empty result - len(nil map) is 0, so this covers both nil and empty
	if len(result) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "(no data returned)",
				},
			},
		}, nil
	}

	truncated := false

	// Pretty print JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		// Return a valid CallToolResult with error message instead of nil
		// This prevents "compaction failed" errors in Claude Desktop
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Error formatting response: %v\n\nRaw data keys: %v", err, getMapKeys(result)),
				},
			},
			IsError: true,
		}, nil
	}

	responseText := string(jsonBytes)

	// Check if response exceeds size limit
	if len(jsonBytes) > MaxResultSize {
		truncated = true
		_, truncatedBytes := truncateResult(result, MaxResultSize)
		if truncatedBytes != nil {
			responseText = string(truncatedBytes)
		} else {
			responseText = string(jsonBytes[:MaxResultSize-TruncationBufferSize])
		}

		totalItems := countItems(result)
		shownItems := countItemsFromBytes(truncatedBytes)

		warningMsg := fmt.Sprintf("\n\n---\n⚠️ RESULT TRUNCATED: Showing %d of %d items (full result was %d bytes, exceeding 1MB limit).\n\n"+
			"💡 **To get ALL results:** Use time-based pagination or add filters to reduce results.",
			shownItems, totalItems, len(jsonBytes))
		responseText += warningMsg

		t.logger.Warn("Result truncated due to size limit",
			zap.Int("original_size", len(jsonBytes)),
			zap.Int("truncated_size", len(responseText)),
		)
	}

	// Add verification metadata for GRPO-style self-correction
	AddVerificationMeta(result, toolName, startTime, truncated)

	// Add proactive suggestions based on tool and result
	suggestions := GetProactiveSuggestions(toolName, result, false)
	if len(suggestions) > 0 {
		responseText += FormatProactiveSuggestions(suggestions)
	}

	// Add cost hints for follow-up tools
	if toolName != "" {
		hints := GetCostHints(toolName)
		if len(hints.SuggestedFollowup) > 0 {
			responseText += "\n\n---\n**Suggested next tools:**"
			for _, followup := range hints.SuggestedFollowup {
				followupHints := GetCostHints(followup)
				responseText += fmt.Sprintf("\n- `%s` (%s, %s)", followup, followupHints.ExecutionSpeed, followupHints.APICost)
			}
		}
	}

	// Final safety check
	responseText = ensureResponseLimit(responseText, t.logger)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: responseText,
			},
		},
	}, nil
}

// FormatResponseWithSummary formats the response with an AI-friendly summary header
func (t *BaseTool) FormatResponseWithSummary(result map[string]interface{}, resultType string) (*mcp.CallToolResult, error) {
	return t.FormatResponseWithSummaryAndSuggestions(result, resultType, "")
}

// FormatResponseWithSummaryAndSuggestions formats the response with summary and proactive suggestions
func (t *BaseTool) FormatResponseWithSummaryAndSuggestions(result map[string]interface{}, resultType string, toolName string) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	// Handle empty result - len(nil map) is 0, so this covers both nil and empty
	if len(result) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("(no %s returned)", resultType),
				},
			},
		}, nil
	}

	// Clean query results to remove unnecessary fields (reduces response size ~30-50%)
	if resultType == "query results" {
		result = CleanQueryResults(result)
	}

	// Generate summary
	summary := GenerateResultSummary(result, resultType)

	// Check for truncation from SSE parsing
	wasTruncated := false
	if truncated, ok := result["_truncated"].(bool); ok && truncated {
		wasTruncated = true
	}

	// Pretty print JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		// Return a valid CallToolResult with error message instead of nil
		// This prevents "compaction failed" errors in Claude Desktop
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Error formatting response: %v\n\nRaw data keys: %v", err, getMapKeys(result)),
				},
			},
			IsError: true,
		}, nil
	}

	// For query results, use markdown format instead of raw JSON
	// This prevents the AI from misinterpreting log message content as JSON structure
	var responseText string
	if resultType == "query results" {
		responseText = formatLogsAsMarkdown(result, summary)
	} else {
		if summary != "" {
			responseText = summary + "---\n\n### Raw Data\n\n" + string(jsonBytes)
		} else {
			responseText = string(jsonBytes)
		}
	}

	// Check if response exceeds size limit
	truncatedBySize := false
	if len(responseText) > MaxResultSize {
		truncatedBySize = true
		// Regenerate with fewer entries
		if resultType == "query results" {
			responseText = formatLogsAsMarkdownTruncated(result, summary, MaxResultSize-TruncationBufferSize)
		} else {
			_, truncatedBytes := truncateResult(result, MaxResultSize-len(summary)-TruncationBufferSize)
			if truncatedBytes != nil {
				if summary != "" {
					responseText = summary + "---\n\n### Raw Data (truncated)\n\n" + string(truncatedBytes)
				} else {
					responseText = string(truncatedBytes)
				}
			}
		}
	}

	// Add pagination info for query results
	if resultType == "query results" {
		paginationInfo := extractPaginationInfo(result, MaxSSEEvents, wasTruncated || truncatedBySize)
		if paginationInfo != nil && paginationInfo.HasMore {
			paginationMsg := fmt.Sprintf("\n\n---\n📄 **PAGINATION INFO:**\n"+
				"- Results returned: %d\n"+
				"- More results available: Yes\n",
				paginationInfo.TotalReturned)

			if paginationInfo.NextStartDate != "" {
				paginationMsg += fmt.Sprintf("- Last timestamp: `%s`\n\n"+
					"**To fetch next page**, use the same query with:\n"+
					"```json\n{\"start_date\": \"%s\"}\n```\n",
					paginationInfo.LastTimestamp,
					paginationInfo.NextStartDate)
			}
			responseText += paginationMsg
		}
	} else if truncatedBySize {
		totalItems := countItems(result)
		shownItems := countItemsFromBytes(nil) // Will return 0 if nil

		warningMsg := fmt.Sprintf("\n\n---\n⚠️ **RESULT TRUNCATED:** Showing %d of %d items.\n\n"+
			"💡 **To get all results:** Use time-based pagination or add filters to reduce results.",
			shownItems, totalItems)
		responseText += warningMsg
	}

	// Add proactive suggestions if tool name provided
	if toolName != "" {
		suggestions := GetProactiveSuggestions(toolName, result, false)
		if len(suggestions) > 0 {
			responseText += FormatProactiveSuggestions(suggestions)
		}
	}

	// Add verification metadata for GRPO-style self-correction
	AddVerificationMeta(result, toolName, startTime, wasTruncated || truncatedBySize)

	// Final safety check: ensure response doesn't exceed absolute limit
	responseText = ensureResponseLimit(responseText, t.logger)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: responseText,
			},
		},
	}, nil
}

// AddRateLimitMetadata adds rate limit information to a result map.
// This helps LLMs understand API limits and pace requests appropriately.
func AddRateLimitMetadata(result map[string]interface{}, available float64, limit int, enabled bool) {
	if !enabled {
		return
	}

	result["_rate_limit"] = map[string]interface{}{
		"available":        int(available),
		"limit_per_second": limit,
		"status":           getRateLimitStatus(available, limit),
	}
}

// getRateLimitStatus returns a human-readable rate limit status
func getRateLimitStatus(available float64, limit int) string {
	ratio := available / float64(limit)
	switch {
	case ratio < 0.1:
		return "critical" // Less than 10% remaining
	case ratio < 0.25:
		return "low" // Less than 25% remaining
	case ratio < 0.5:
		return "moderate" // Less than 50% remaining
	default:
		return "healthy" // More than 50% remaining
	}
}

// FormatCompactSummary returns only statistical summary without raw events.
// This dramatically reduces token usage (~90% reduction) while preserving key insights.
// Use this for initial exploration before drilling down with full results.
func (t *BaseTool) FormatCompactSummary(result map[string]interface{}, _ string) (*mcp.CallToolResult, error) {
	if len(result) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "(no data returned)"},
			},
		}, nil
	}

	var summary strings.Builder
	summary.WriteString("## Query Results Summary (compact mode)\n\n")

	// Extract events for analysis
	events, hasEvents := result["events"].([]interface{})
	if hasEvents && len(events) > 0 {
		summary.WriteString(fmt.Sprintf("**Total Results:** %d log entries\n\n", len(events)))

		// Severity distribution
		severityDist := analyzeSeverityDistribution(events)
		if len(severityDist) > 0 {
			summary.WriteString("### Severity Distribution\n")
			// Sort by severity level for consistent output
			severityOrder := []string{"Critical", "Error", "Warning", "Info", "Verbose", "Debug"}
			for _, sev := range severityOrder {
				if count, ok := severityDist[sev]; ok {
					summary.WriteString(fmt.Sprintf("- **%s**: %d\n", sev, count))
				}
			}
			summary.WriteString("\n")
		}

		// Top applications
		topApps := extractTopValues(events, "applicationname", 5)
		if len(topApps) > 0 {
			summary.WriteString("### Top Applications\n")
			for _, app := range topApps {
				summary.WriteString(fmt.Sprintf("- %s: %d entries\n", app.Value, app.Count))
			}
			summary.WriteString("\n")
		}

		// Top subsystems
		topSubs := extractTopValues(events, "subsystemname", 5)
		if len(topSubs) > 0 {
			summary.WriteString("### Top Subsystems\n")
			for _, sub := range topSubs {
				summary.WriteString(fmt.Sprintf("- %s: %d entries\n", sub.Value, sub.Count))
			}
			summary.WriteString("\n")
		}

		// Time range
		timeRange := extractTimeRange(events)
		if timeRange != "" {
			summary.WriteString(fmt.Sprintf("### Time Range\n%s\n\n", timeRange))
		}

		// Sample messages (first 3 unique error/warning messages)
		sampleMessages := extractSampleMessages(events, 3)
		if len(sampleMessages) > 0 {
			summary.WriteString("### Sample Messages\n")
			for i, msg := range sampleMessages {
				// Truncate long messages
				if len(msg) > 150 {
					msg = msg[:147] + "..."
				}
				summary.WriteString(fmt.Sprintf("%d. `%s`\n", i+1, msg))
			}
			summary.WriteString("\n")
		}
	} else {
		summary.WriteString("**No events found** matching the query criteria.\n\n")
	}

	// Add query metadata if present
	if meta, ok := result["_query_metadata"].(map[string]interface{}); ok {
		summary.WriteString("### Query Info\n")
		if tier, ok := meta["tier"].(string); ok {
			summary.WriteString(fmt.Sprintf("- Tier: %s\n", tier))
		}
		if start, ok := meta["start_date"].(string); ok {
			summary.WriteString(fmt.Sprintf("- Start: %s\n", start))
		}
		if end, ok := meta["end_date"].(string); ok {
			summary.WriteString(fmt.Sprintf("- End: %s\n", end))
		}
		summary.WriteString("\n")
	}

	// Add guidance for getting full results
	summary.WriteString("---\n")
	summary.WriteString("💡 **To see full log entries**, run the same query without `summary_only: true`\n")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: summary.String()},
		},
	}, nil
}

// extractSampleMessages extracts unique log messages from events
func extractSampleMessages(events []interface{}, limit int) []string {
	seen := make(map[string]bool)
	var messages []string

	for _, event := range events {
		if len(messages) >= limit {
			break
		}
		if eventMap, ok := event.(map[string]interface{}); ok {
			msg := ""
			// Try different message field locations
			if m, ok := eventMap["message"].(string); ok {
				msg = m
			} else if userData, ok := eventMap["user_data"].(map[string]interface{}); ok {
				if m, ok := userData["message"].(string); ok {
					msg = m
				}
			} else if text, ok := eventMap["text"].(string); ok {
				msg = text
			}

			if msg != "" && !seen[msg] {
				seen[msg] = true
				messages = append(messages, msg)
			}
		}
	}
	return messages
}

// ensureResponseLimit ensures the response text doesn't exceed FinalResponseLimit
// This is a safety net to prevent MCP 1MB limit errors
func ensureResponseLimit(text string, logger *zap.Logger) string {
	if len(text) <= FinalResponseLimit {
		return text
	}

	if logger != nil {
		logger.Warn("Response exceeded final limit, truncating",
			zap.Int("original_size", len(text)),
			zap.Int("limit", FinalResponseLimit),
		)
	}

	// Hard truncate and add warning
	truncated := text[:FinalResponseLimit-TruncationBufferSize]
	truncated += "\n\n---\n⚠️ **Response truncated** due to size limits. Use filters or pagination to get complete results."
	return truncated
}

// formatLogsAsMarkdown formats log results as readable markdown instead of raw JSON
// This prevents the AI from misinterpreting log message content as JSON/code structure
func formatLogsAsMarkdown(result map[string]interface{}, summary string) string {
	var sb strings.Builder

	if summary != "" {
		sb.WriteString(summary)
		sb.WriteString("---\n\n")
	}

	sb.WriteString("### Log Entries\n\n")

	// Get logs from cleaned result
	logs, ok := result["logs"].([]interface{})
	if !ok {
		// Try events for uncleaned results
		logs, ok = result["events"].([]interface{})
	}

	if !ok || len(logs) == 0 {
		sb.WriteString("No log entries found.\n")
		return sb.String()
	}

	for i, log := range logs {
		formatSingleLogEntry(&sb, log, i+1)
	}

	// Add query metadata if present
	if meta, ok := result["_query_metadata"].(map[string]interface{}); ok {
		sb.WriteString("\n---\n### Query Metadata\n")
		if tier, ok := meta["tier"].(string); ok {
			sb.WriteString(fmt.Sprintf("- **Tier:** %s\n", tier))
		}
		if inst, ok := meta["instance"].(map[string]interface{}); ok {
			if name, ok := inst["instance_name"].(string); ok && name != "" {
				sb.WriteString(fmt.Sprintf("- **Instance:** %s\n", name))
			}
			if region, ok := inst["region"].(string); ok {
				sb.WriteString(fmt.Sprintf("- **Region:** %s\n", region))
			}
		}

		// Show auto-corrections that were applied - this helps the AI learn correct DataPrime syntax
		if corrections, ok := meta["auto_corrections"].([]string); ok && len(corrections) > 0 {
			sb.WriteString("\n⚠️ **Query Auto-Corrections Applied:**\n")
			sb.WriteString("*The following corrections were automatically applied. Please use the correct syntax in future queries:*\n")
			for _, correction := range corrections {
				sb.WriteString(fmt.Sprintf("  - `%s`\n", correction))
			}
			if correctedQuery, ok := meta["corrected_query"].(string); ok {
				sb.WriteString(fmt.Sprintf("\n**Corrected Query:** `%s`\n", correctedQuery))
			}
		} else if corrections, ok := meta["auto_corrections"].([]interface{}); ok && len(corrections) > 0 {
			// Handle case where corrections are stored as []interface{}
			sb.WriteString("\n⚠️ **Query Auto-Corrections Applied:**\n")
			sb.WriteString("*The following corrections were automatically applied. Please use the correct syntax in future queries:*\n")
			for _, correction := range corrections {
				if corr, ok := correction.(string); ok {
					sb.WriteString(fmt.Sprintf("  - `%s`\n", corr))
				}
			}
			if correctedQuery, ok := meta["corrected_query"].(string); ok {
				sb.WriteString(fmt.Sprintf("\n**Corrected Query:** `%s`\n", correctedQuery))
			}
		}
	}

	return sb.String()
}

// formatLogsAsMarkdownTruncated formats logs with a size limit
func formatLogsAsMarkdownTruncated(result map[string]interface{}, summary string, maxSize int) string {
	var sb strings.Builder

	if summary != "" {
		sb.WriteString(summary)
		sb.WriteString("---\n\n")
	}

	sb.WriteString("### Log Entries (truncated)\n\n")

	logs, ok := result["logs"].([]interface{})
	if !ok {
		logs, ok = result["events"].([]interface{})
	}

	if !ok || len(logs) == 0 {
		sb.WriteString("No log entries found.\n")
		return sb.String()
	}

	totalLogs := len(logs)
	shownLogs := 0

	for i, log := range logs {
		// Check if we're approaching the limit
		if sb.Len() > maxSize-1000 {
			break
		}
		formatSingleLogEntry(&sb, log, i+1)
		shownLogs++
	}

	if shownLogs < totalLogs {
		sb.WriteString(fmt.Sprintf("\n---\n⚠️ **Showing %d of %d log entries.** Use `summary_only: true` or add filters to reduce results.\n", shownLogs, totalLogs))
	}

	// Also show auto-corrections in truncated output - important for AI learning
	if meta, ok := result["_query_metadata"].(map[string]interface{}); ok {
		if corrections, ok := meta["auto_corrections"].([]string); ok && len(corrections) > 0 {
			sb.WriteString("\n⚠️ **Query Auto-Corrections Applied:**\n")
			for _, correction := range corrections {
				sb.WriteString(fmt.Sprintf("  - `%s`\n", correction))
			}
		} else if corrections, ok := meta["auto_corrections"].([]interface{}); ok && len(corrections) > 0 {
			sb.WriteString("\n⚠️ **Query Auto-Corrections Applied:**\n")
			for _, correction := range corrections {
				if corr, ok := correction.(string); ok {
					sb.WriteString(fmt.Sprintf("  - `%s`\n", corr))
				}
			}
		}
	}

	return sb.String()
}

// formatSingleLogEntry formats a single log entry as markdown
func formatSingleLogEntry(sb *strings.Builder, log interface{}, index int) {
	logMap, ok := log.(map[string]interface{})
	if !ok {
		return
	}

	fmt.Fprintf(sb, "**[%d]** ", index)

	// Time
	if t, ok := logMap["time"].(string); ok {
		fmt.Fprintf(sb, "`%s` ", t)
	}

	// Severity
	if sev, ok := logMap["severity"].(string); ok {
		fmt.Fprintf(sb, "[%s] ", sev)
	}

	// App and subsystem
	if app, ok := logMap["app"].(string); ok {
		fmt.Fprintf(sb, "**%s**", app)
		if sub, ok := logMap["subsystem"].(string); ok {
			fmt.Fprintf(sb, "/%s", sub)
		}
		sb.WriteString(" ")
	}

	sb.WriteString("\n")

	// Message - escape it to prevent markdown/code interpretation
	if msg, ok := logMap["message"].(string); ok && msg != "" {
		// Truncate very long messages
		if len(msg) > 500 {
			msg = msg[:497] + "..."
		}
		// Use blockquote to safely display the message without interpretation
		sb.WriteString("> ")
		// Replace newlines in message to keep blockquote format
		msg = strings.ReplaceAll(msg, "\n", "\n> ")
		sb.WriteString(msg)
		sb.WriteString("\n")
	}

	// Additional fields (exec_ms, sql, etc.)
	if execMs, ok := logMap["exec_ms"].(int); ok {
		fmt.Fprintf(sb, "  - Execution time: %dms\n", execMs)
	}
	if execMs, ok := logMap["exec_ms"].(float64); ok {
		fmt.Fprintf(sb, "  - Execution time: %.0fms\n", execMs)
	}

	sb.WriteString("\n")
}

// ============================================================================
// HIGH-CARDINALITY FILTERING (Server-side Token Drowning Prevention)
// ============================================================================

// FormatClusteredSummary formats logs as semantic clusters instead of raw dumps.
// This dramatically reduces token usage for large result sets while preserving insights.
// Implements the LogAssist/LogBatcher 2025 pattern for high-cardinality filtering.
func FormatClusteredSummary(events []interface{}, maxClusters int) string {
	if len(events) == 0 {
		return "No log events to cluster."
	}

	// Cluster logs by template
	clusters := ClusterLogs(events)

	var sb strings.Builder
	sb.WriteString("## 🔬 Clustered Log Analysis\n\n")
	sb.WriteString(fmt.Sprintf("**Total Events:** %d\n", len(events)))
	sb.WriteString(fmt.Sprintf("**Unique Patterns:** %d\n\n", len(clusters)))

	if maxClusters <= 0 {
		maxClusters = 10
	}

	// Show top clusters
	sb.WriteString("### Top Error Patterns\n\n")
	for i, cluster := range clusters {
		if i >= maxClusters {
			sb.WriteString(fmt.Sprintf("... and %d more patterns\n", len(clusters)-maxClusters))
			break
		}

		// Format cluster
		sb.WriteString(fmt.Sprintf("#### Pattern %d: `%s`\n", i+1, cluster.TemplateID))
		sb.WriteString(fmt.Sprintf("- **Count:** %d occurrences\n", cluster.Count))
		sb.WriteString(fmt.Sprintf("- **Severity:** %s\n", cluster.Severity))
		sb.WriteString(fmt.Sprintf("- **Root Cause:** %s\n", cluster.RootCause))
		if len(cluster.Apps) > 0 {
			sb.WriteString(fmt.Sprintf("- **Apps:** %s\n", strings.Join(cluster.Apps, ", ")))
		}
		sb.WriteString(fmt.Sprintf("- **Template:** `%s`\n", truncateString(cluster.Template, 80)))
		if len(cluster.Samples) > 0 {
			sb.WriteString(fmt.Sprintf("- **Sample:** `%s`\n", truncateString(cluster.Samples[0], 100)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ClusteredAnalysis extends the base analysis with semantic log clustering.
// This provides SOTA 2025 pattern detection (LogAssist/LogBatcher paradigm).
type ClusteredAnalysis struct {
	TotalEvents int              `json:"total_events"`
	TimeRange   *ClusterTimeInfo `json:"time_range,omitempty"`
	Clusters    []*LogCluster    `json:"clusters,omitempty"`
	RootCauses  []string         `json:"root_causes,omitempty"`
}

// ClusterTimeInfo contains time range metadata for clustering
type ClusterTimeInfo struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Duration string `json:"duration"`
}

// AnalyzeQueryResultsWithClustering performs analysis with semantic log clustering.
// This extends the base AnalyzeQueryResults with SOTA 2025 pattern recognition.
func AnalyzeQueryResultsWithClustering(result map[string]interface{}) *ClusteredAnalysis {
	events, ok := result["events"].([]interface{})
	if !ok {
		return nil
	}

	analysis := &ClusteredAnalysis{
		TotalEvents: len(events),
	}

	if len(events) == 0 {
		return analysis
	}

	// Time range extraction
	timeRangeStr := extractTimeRange(events)
	if timeRangeStr != "" {
		parts := strings.Split(timeRangeStr, "\n")
		if len(parts) >= 2 {
			analysis.TimeRange = &ClusterTimeInfo{
				Start: strings.TrimPrefix(parts[0], "From: "),
				End:   strings.TrimPrefix(parts[1], "To: "),
			}
		}
	}

	// Cluster analysis for pattern detection
	clusters := ClusterLogs(events)
	if len(clusters) > 0 {
		analysis.Clusters = clusters

		// Extract unique root causes
		seenCauses := make(map[string]bool)
		for _, cluster := range clusters {
			if cluster.RootCause != "UNKNOWN" && !seenCauses[cluster.RootCause] {
				seenCauses[cluster.RootCause] = true
				analysis.RootCauses = append(analysis.RootCauses, cluster.RootCause)
			}
		}
	}

	return analysis
}
