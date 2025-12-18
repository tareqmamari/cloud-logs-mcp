package tools

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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
		// Handle the nested result structure from API
		if resultObj, ok := eventMap["result"].(map[string]interface{}); ok {
			if results, ok := resultObj["results"].([]interface{}); ok {
				for _, r := range results {
					if rMap, ok := r.(map[string]interface{}); ok {
						cleanedEvents = append(cleanedEvents, transformLogEntry(rMap))
					}
				}
			}
		} else {
			// Direct event format
			cleanedEvents = append(cleanedEvents, transformLogEntry(eventMap))
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
func transformLogEntry(entry map[string]interface{}) map[string]interface{} {
	compact := make(map[string]interface{})

	// Extract essential labels (app, subsystem)
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
	}

	// Extract essential metadata (timestamp, severity)
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
	}

	// Parse and extract from user_data JSON
	if userData, ok := entry["user_data"].(string); ok && userData != "" {
		var ud map[string]interface{}
		if err := json.Unmarshal([]byte(userData), &ud); err == nil {
			extractUserData(ud, compact)
		}
	}

	return compact
}

// extractUserData extracts essential fields from parsed user_data
func extractUserData(ud map[string]interface{}, compact map[string]interface{}) {
	// Extract message
	if event, ok := ud["event"].(map[string]interface{}); ok {
		if msg, ok := event["_message"].(string); ok {
			compact["message"] = msg
		}
		// Extract SQL if present (useful for DB queries)
		if sql, ok := event["sql"].(string); ok {
			compact["sql"] = sql
		}
		// Extract execution time if present
		if execMs, ok := event["execMs"].(float64); ok {
			compact["exec_ms"] = int(execMs)
		}
	}

	// Direct message field
	if msg, ok := ud["message"].(string); ok && compact["message"] == nil {
		compact["message"] = msg
	}

	// Log level
	if level, ok := ud["level"].(string); ok {
		compact["level"] = level
	}

	// Logger name (shortened)
	if logger, ok := ud["logger_name"].(string); ok {
		// Shorten long logger names
		parts := strings.Split(logger, ".")
		if len(parts) > 2 {
			compact["logger"] = parts[len(parts)-2] + "." + parts[len(parts)-1]
		} else {
			compact["logger"] = logger
		}
	}

	// Trace/span IDs (useful for correlation)
	if traceID, ok := ud["trace_id"].(string); ok {
		compact["trace_id"] = traceID
	}
	if spanID, ok := ud["span_id"].(string); ok {
		compact["span_id"] = spanID
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
