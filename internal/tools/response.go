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
	// MaxResultSize is the maximum size of tool results in bytes (500KB to leave significant headroom under 1MB MCP limit)
	// This is reduced to account for JSON formatting, summaries, pagination info, suggestions, and other metadata overhead
	MaxResultSize = 500 * 1024

	// FinalResponseLimit is the absolute maximum size for the final response text before sending to MCP
	// This ensures we never exceed the 1MB limit even with all metadata added
	FinalResponseLimit = 950 * 1024

	// MaxSSEEvents is the maximum number of SSE events to parse from a query response
	// This prevents memory issues when queries return very large result sets
	MaxSSEEvents = 200

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
		return nil, fmt.Errorf("failed to format response: %w", err)
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
		warningMsg := fmt.Sprintf("\n\n---\n⚠️ RESULT TRUNCATED: Showing %d of %d items (full result was %d bytes, exceeding 1MB limit).\n\n"+
			"**To get ALL results, use pagination by splitting your query:**\n\n"+
			"1. **Time-based pagination** (recommended): Split your time range into smaller chunks:\n"+
			"   - First call: start_date='2024-01-01T00:00:00Z', end_date='2024-01-01T12:00:00Z'\n"+
			"   - Second call: start_date='2024-01-01T12:00:00Z', end_date='2024-01-02T00:00:00Z'\n\n"+
			"2. **Limit-based pagination**: Use smaller limits and note the last timestamp:\n"+
			"   - First call: limit=500\n"+
			"   - Second call: limit=500, start_date=(last timestamp from previous call)\n\n"+
			"3. **Filter more specifically**: Add filters to reduce results:\n"+
			"   - By application: applicationName='your-app' or query='source logs | filter $l.applicationname == \"your-app\"'\n"+
			"   - By subsystem: subsystemName='your-subsystem' or query='source logs | filter $l.subsystemname == \"your-subsystem\"'\n"+
			"   - By severity: query='source logs | filter $m.severity >= 5' (5=error, 6=critical)\n"+
			"   - By keyword: query='source logs | filter $d.message.contains(\"error\")'",
			shownItems, totalItems, len(jsonBytes))
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
		return nil, fmt.Errorf("failed to format response: %w", err)
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
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	var responseText string
	if summary != "" {
		responseText = summary + "---\n\n### Raw Data\n\n" + string(jsonBytes)
	} else {
		responseText = string(jsonBytes)
	}

	// Check if response exceeds size limit
	truncatedBySize := false
	if len(responseText) > MaxResultSize {
		truncatedBySize = true
		// Truncate the JSON but keep the summary
		_, truncatedBytes := truncateResult(result, MaxResultSize-len(summary)-TruncationBufferSize)
		if truncatedBytes != nil {
			if summary != "" {
				responseText = summary + "---\n\n### Raw Data (truncated)\n\n" + string(truncatedBytes)
			} else {
				responseText = string(truncatedBytes)
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
